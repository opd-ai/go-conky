// Package monitor provides IMAP and POP3 mail monitoring.
package monitor

import (
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// MailStats contains mail account statistics.
type MailStats struct {
	// Accounts maps account name to account statistics.
	Accounts map[string]MailAccountStats
}

// MailAccountStats contains statistics for a single mail account.
type MailAccountStats struct {
	// Name is the account identifier.
	Name string
	// Type is the protocol type ("imap" or "pop3").
	Type string
	// Unseen is the count of unread messages.
	Unseen int
	// Total is the total message count.
	Total int
	// LastCheck is when the account was last checked.
	LastCheck time.Time
	// Error contains the last error message, if any.
	Error string
}

// MailConfig contains mail account configuration.
type MailConfig struct {
	// Name is a unique identifier for this account.
	Name string
	// Type is "imap" or "pop3".
	Type string
	// Host is the mail server hostname.
	Host string
	// Port is the mail server port.
	Port int
	// Username for authentication.
	Username string
	// Password for authentication.
	Password string
	// UseTLS enables TLS/SSL connection.
	UseTLS bool
	// Folder is the mailbox folder (IMAP only, default "INBOX").
	Folder string
	// Interval is how often to check (minimum 60 seconds).
	Interval time.Duration
}

// mailReader reads mail statistics via IMAP/POP3.
type mailReader struct {
	mu       sync.RWMutex
	accounts map[string]*mailAccountReader
}

// mailAccountReader handles reading from a single mail account.
type mailAccountReader struct {
	config      MailConfig
	cache       MailAccountStats
	lastCheck   time.Time
	mu          sync.Mutex
	dialTimeout time.Duration
	readTimeout time.Duration
}

// newMailReader creates a new mailReader.
func newMailReader() *mailReader {
	return &mailReader{
		accounts: make(map[string]*mailAccountReader),
	}
}

// AddAccount adds a mail account configuration.
func (r *mailReader) AddAccount(config MailConfig) error {
	if config.Name == "" {
		return fmt.Errorf("mail account name is required")
	}
	if config.Type != "imap" && config.Type != "pop3" {
		return fmt.Errorf("mail type must be 'imap' or 'pop3'")
	}
	if config.Host == "" {
		return fmt.Errorf("mail host is required")
	}
	if config.Interval < 60*time.Second {
		config.Interval = 60 * time.Second
	}
	if config.Folder == "" {
		config.Folder = "INBOX"
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.accounts[config.Name] = &mailAccountReader{
		config:      config,
		dialTimeout: 10 * time.Second,
		readTimeout: 30 * time.Second,
	}

	return nil
}

// RemoveAccount removes a mail account.
func (r *mailReader) RemoveAccount(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.accounts, name)
}

// ReadStats returns current mail statistics for all accounts.
func (r *mailReader) ReadStats() (MailStats, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := MailStats{
		Accounts: make(map[string]MailAccountStats, len(r.accounts)),
	}

	for name, account := range r.accounts {
		stats.Accounts[name] = account.getStats()
	}

	return stats, nil
}

// GetAccountStats returns stats for a specific account.
func (r *mailReader) GetAccountStats(name string) (MailAccountStats, bool) {
	r.mu.RLock()
	account, ok := r.accounts[name]
	r.mu.RUnlock()

	if !ok {
		return MailAccountStats{}, false
	}

	return account.getStats(), true
}

// getStats returns cached stats, refreshing if interval has passed.
func (ar *mailAccountReader) getStats() MailAccountStats {
	ar.mu.Lock()
	defer ar.mu.Unlock()

	// Check if cache is still valid
	if time.Since(ar.lastCheck) < ar.config.Interval {
		return ar.cache
	}

	// Refresh stats
	ar.refreshStatsLocked()
	return ar.cache
}

// refreshStatsLocked fetches new stats from the mail server.
// Caller must hold ar.mu.
func (ar *mailAccountReader) refreshStatsLocked() {
	ar.cache.Name = ar.config.Name
	ar.cache.Type = ar.config.Type
	ar.cache.LastCheck = time.Now()
	ar.lastCheck = time.Now()

	var unseen, total int
	var err error

	switch ar.config.Type {
	case "imap":
		unseen, total, err = ar.checkIMAP()
	case "pop3":
		unseen, total, err = ar.checkPOP3()
	default:
		err = fmt.Errorf("unknown mail type: %s", ar.config.Type)
	}

	if err != nil {
		ar.cache.Error = err.Error()
	} else {
		ar.cache.Error = ""
		ar.cache.Unseen = unseen
		ar.cache.Total = total
	}
}

// checkIMAP connects to IMAP server and gets message counts.
func (ar *mailAccountReader) checkIMAP() (unseen, total int, err error) {
	addr := net.JoinHostPort(ar.config.Host, strconv.Itoa(ar.config.Port))

	var conn net.Conn
	if ar.config.UseTLS {
		tlsConfig := &tls.Config{
			ServerName: ar.config.Host,
			MinVersion: tls.VersionTLS12,
		}
		conn, err = tls.DialWithDialer(&net.Dialer{Timeout: ar.dialTimeout}, "tcp", addr, tlsConfig)
	} else {
		conn, err = net.DialTimeout("tcp", addr, ar.dialTimeout)
	}
	if err != nil {
		return 0, 0, fmt.Errorf("connection failed: %w", err)
	}
	defer conn.Close()

	// Set read timeout for entire session
	if err := conn.SetDeadline(time.Now().Add(ar.readTimeout)); err != nil {
		return 0, 0, fmt.Errorf("set deadline: %w", err)
	}

	// Read greeting
	if _, err := ar.readIMAPResponse(conn); err != nil {
		return 0, 0, fmt.Errorf("read greeting: %w", err)
	}

	// Login
	if err := ar.sendIMAPCommand(conn, "a001", fmt.Sprintf("LOGIN %s %s", ar.config.Username, ar.config.Password)); err != nil {
		return 0, 0, fmt.Errorf("login failed: %w", err)
	}

	// Select folder
	resp, err := ar.sendIMAPCommandGetResponse(conn, "a002", fmt.Sprintf("SELECT %s", ar.config.Folder))
	if err != nil {
		return 0, 0, fmt.Errorf("select folder failed: %w", err)
	}

	// Parse EXISTS from SELECT response
	total = ar.parseIMAPExists(resp)

	// Search for unseen messages
	resp, err = ar.sendIMAPCommandGetResponse(conn, "a003", "SEARCH UNSEEN")
	if err != nil {
		return 0, total, fmt.Errorf("search failed: %w", err)
	}
	unseen = ar.parseIMAPSearchCount(resp)

	// Logout
	_ = ar.sendIMAPCommand(conn, "a004", "LOGOUT")

	return unseen, total, nil
}

// sendIMAPCommand sends a command and reads the response.
func (ar *mailAccountReader) sendIMAPCommand(conn net.Conn, tag, cmd string) error {
	_, err := fmt.Fprintf(conn, "%s %s\r\n", tag, cmd)
	if err != nil {
		return err
	}

	resp, err := ar.readIMAPResponseUntilTag(conn, tag)
	if err != nil {
		return err
	}

	if !strings.Contains(resp, tag+" OK") {
		return fmt.Errorf("command failed: %s", resp)
	}

	return nil
}

// sendIMAPCommandGetResponse sends a command and returns the full response.
func (ar *mailAccountReader) sendIMAPCommandGetResponse(conn net.Conn, tag, cmd string) (string, error) {
	_, err := fmt.Fprintf(conn, "%s %s\r\n", tag, cmd)
	if err != nil {
		return "", err
	}

	return ar.readIMAPResponseUntilTag(conn, tag)
}

// readIMAPResponse reads a single line response.
func (ar *mailAccountReader) readIMAPResponse(conn net.Conn) (string, error) {
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return "", err
	}
	return string(buf[:n]), nil
}

// readIMAPResponseUntilTag reads responses until the tagged response is found.
func (ar *mailAccountReader) readIMAPResponseUntilTag(conn net.Conn, tag string) (string, error) {
	var result strings.Builder
	buf := make([]byte, 4096)

	for {
		n, err := conn.Read(buf)
		if err != nil {
			return result.String(), err
		}

		chunk := string(buf[:n])
		result.WriteString(chunk)

		// Check if we have the tagged response
		if strings.Contains(chunk, tag+" OK") || strings.Contains(chunk, tag+" NO") || strings.Contains(chunk, tag+" BAD") {
			break
		}
	}

	return result.String(), nil
}

// parseIMAPExists extracts the EXISTS count from SELECT response.
func (ar *mailAccountReader) parseIMAPExists(resp string) int {
	lines := strings.Split(resp, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "* ") && strings.Contains(line, " EXISTS") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				count, _ := strconv.Atoi(parts[1])
				return count
			}
		}
	}
	return 0
}

// parseIMAPSearchCount counts message IDs in SEARCH response.
func (ar *mailAccountReader) parseIMAPSearchCount(resp string) int {
	lines := strings.Split(resp, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "* SEARCH") {
			// Count the message numbers after "* SEARCH"
			parts := strings.Fields(line)
			if len(parts) > 2 {
				return len(parts) - 2 // Subtract "* SEARCH"
			}
			return 0
		}
	}
	return 0
}

// checkPOP3 connects to POP3 server and gets message count.
func (ar *mailAccountReader) checkPOP3() (unseen, total int, err error) {
	addr := net.JoinHostPort(ar.config.Host, strconv.Itoa(ar.config.Port))

	var conn net.Conn
	if ar.config.UseTLS {
		tlsConfig := &tls.Config{
			ServerName: ar.config.Host,
			MinVersion: tls.VersionTLS12,
		}
		conn, err = tls.DialWithDialer(&net.Dialer{Timeout: ar.dialTimeout}, "tcp", addr, tlsConfig)
	} else {
		conn, err = net.DialTimeout("tcp", addr, ar.dialTimeout)
	}
	if err != nil {
		return 0, 0, fmt.Errorf("connection failed: %w", err)
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(ar.readTimeout)); err != nil {
		return 0, 0, fmt.Errorf("set deadline: %w", err)
	}

	// Read greeting
	if _, err := ar.readPOP3Response(conn); err != nil {
		return 0, 0, fmt.Errorf("read greeting: %w", err)
	}

	// USER
	if err := ar.sendPOP3Command(conn, fmt.Sprintf("USER %s", ar.config.Username)); err != nil {
		return 0, 0, fmt.Errorf("user command failed: %w", err)
	}

	// PASS
	if err := ar.sendPOP3Command(conn, fmt.Sprintf("PASS %s", ar.config.Password)); err != nil {
		return 0, 0, fmt.Errorf("pass command failed: %w", err)
	}

	// STAT - returns total message count
	resp, err := ar.sendPOP3CommandGetResponse(conn, "STAT")
	if err != nil {
		return 0, 0, fmt.Errorf("stat command failed: %w", err)
	}

	// Parse: +OK nn mm (nn = message count, mm = mailbox size)
	parts := strings.Fields(resp)
	if len(parts) >= 2 {
		total, _ = strconv.Atoi(parts[1])
	}

	// POP3 doesn't have an unseen concept, so we report all as unseen
	unseen = total

	// QUIT
	_ = ar.sendPOP3Command(conn, "QUIT")

	return unseen, total, nil
}

// sendPOP3Command sends a POP3 command and checks for +OK response.
func (ar *mailAccountReader) sendPOP3Command(conn net.Conn, cmd string) error {
	_, err := fmt.Fprintf(conn, "%s\r\n", cmd)
	if err != nil {
		return err
	}

	resp, err := ar.readPOP3Response(conn)
	if err != nil {
		return err
	}

	if !strings.HasPrefix(resp, "+OK") {
		return fmt.Errorf("command failed: %s", resp)
	}

	return nil
}

// sendPOP3CommandGetResponse sends a POP3 command and returns the response.
func (ar *mailAccountReader) sendPOP3CommandGetResponse(conn net.Conn, cmd string) (string, error) {
	_, err := fmt.Fprintf(conn, "%s\r\n", cmd)
	if err != nil {
		return "", err
	}

	return ar.readPOP3Response(conn)
}

// readPOP3Response reads a single line response.
func (ar *mailAccountReader) readPOP3Response(conn net.Conn) (string, error) {
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(buf[:n])), nil
}

// GetUnseenCount returns the unseen message count for a named account.
func (r *mailReader) GetUnseenCount(name string) int {
	stats, ok := r.GetAccountStats(name)
	if !ok {
		return 0
	}
	return stats.Unseen
}

// GetTotalCount returns the total message count for a named account.
func (r *mailReader) GetTotalCount(name string) int {
	stats, ok := r.GetAccountStats(name)
	if !ok {
		return 0
	}
	return stats.Total
}

// GetTotalUnseen returns the sum of unseen messages across all accounts.
func (r *mailReader) GetTotalUnseen() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var total int
	for _, account := range r.accounts {
		stats := account.getStats()
		total += stats.Unseen
	}
	return total
}

// GetTotalMessages returns the sum of all messages across all accounts.
func (r *mailReader) GetTotalMessages() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var total int
	for _, account := range r.accounts {
		stats := account.getStats()
		total += stats.Total
	}
	return total
}
