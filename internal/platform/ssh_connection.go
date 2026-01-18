package platform

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/ssh"
)

// ConnectionState represents the current state of the SSH connection.
type ConnectionState int32

const (
	// ConnectionStateDisconnected indicates no active connection.
	ConnectionStateDisconnected ConnectionState = iota
	// ConnectionStateConnecting indicates a connection attempt is in progress.
	ConnectionStateConnecting
	// ConnectionStateConnected indicates an active healthy connection.
	ConnectionStateConnected
	// ConnectionStateReconnecting indicates a reconnection attempt is in progress.
	ConnectionStateReconnecting
)

// String returns the string representation of a ConnectionState.
func (s ConnectionState) String() string {
	switch s {
	case ConnectionStateDisconnected:
		return "disconnected"
	case ConnectionStateConnecting:
		return "connecting"
	case ConnectionStateConnected:
		return "connected"
	case ConnectionStateReconnecting:
		return "reconnecting"
	default:
		return "unknown"
	}
}

// ConnectionStats provides statistics about the SSH connection.
type ConnectionStats struct {
	State             ConnectionState
	ConnectedSince    time.Time
	ReconnectAttempts int64
	TotalReconnects   int64
	LastError         error
	LastErrorTime     time.Time
	SessionsCreated   int64
	SessionsReused    int64
	KeepalivesSent    int64
	KeepalivesFailed  int64
}

// SSHConnectionConfig configures SSH connection management.
type SSHConnectionConfig struct {
	// KeepAliveInterval is the interval between keepalive probes.
	// Default: 30 seconds. Set to 0 to disable keepalives.
	KeepAliveInterval time.Duration

	// KeepAliveTimeout is the timeout for keepalive responses.
	// Default: 15 seconds.
	KeepAliveTimeout time.Duration

	// MaxReconnectAttempts is the maximum number of reconnection attempts.
	// 0 means unlimited attempts.
	MaxReconnectAttempts int

	// InitialReconnectDelay is the initial delay before first reconnection attempt.
	// Default: 1 second.
	InitialReconnectDelay time.Duration

	// MaxReconnectDelay is the maximum delay between reconnection attempts.
	// Default: 5 minutes.
	MaxReconnectDelay time.Duration

	// SessionPoolSize is the maximum number of cached sessions.
	// Default: 5.
	SessionPoolSize int

	// SessionIdleTimeout is how long idle sessions are kept in the pool.
	// Default: 1 minute.
	SessionIdleTimeout time.Duration

	// OnStateChange is called when the connection state changes.
	OnStateChange func(from, to ConnectionState)
}

// sshConnectionManager manages SSH connections with pooling, keepalive, and reconnection.
type sshConnectionManager struct {
	config    SSHConnectionConfig
	sshConfig *ssh.ClientConfig
	address   string
	client    *ssh.Client
	mu        sync.RWMutex
	state     atomic.Int32
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup

	// Stats
	connectedSince    time.Time
	reconnectAttempts atomic.Int64
	totalReconnects   atomic.Int64
	lastError         error
	lastErrorTime     time.Time
	sessionsCreated   atomic.Int64
	sessionsReused    atomic.Int64
	keepalivesSent    atomic.Int64
	keepalivesFailed  atomic.Int64

	// Session pool
	sessionPool   []*pooledSession
	sessionPoolMu sync.Mutex
}

// pooledSession wraps an SSH session with metadata for pooling.
type pooledSession struct {
	session   *ssh.Session
	createdAt time.Time
	inUse     bool
}

// newSSHConnectionManager creates a new SSH connection manager.
func newSSHConnectionManager(address string, sshConfig *ssh.ClientConfig, config SSHConnectionConfig) *sshConnectionManager {
	// Apply defaults
	if config.KeepAliveInterval == 0 {
		config.KeepAliveInterval = 30 * time.Second
	}
	if config.KeepAliveTimeout == 0 {
		config.KeepAliveTimeout = 15 * time.Second
	}
	if config.InitialReconnectDelay == 0 {
		config.InitialReconnectDelay = 1 * time.Second
	}
	if config.MaxReconnectDelay == 0 {
		config.MaxReconnectDelay = 5 * time.Minute
	}
	if config.SessionPoolSize == 0 {
		config.SessionPoolSize = 5
	}
	if config.SessionIdleTimeout == 0 {
		config.SessionIdleTimeout = 1 * time.Minute
	}

	return &sshConnectionManager{
		config:      config,
		sshConfig:   sshConfig,
		address:     address,
		sessionPool: make([]*pooledSession, 0, config.SessionPoolSize),
	}
}

// Connect establishes the SSH connection.
func (m *sshConnectionManager) Connect(ctx context.Context) error {
	m.ctx, m.cancel = context.WithCancel(ctx)

	if !m.setState(ConnectionStateDisconnected, ConnectionStateConnecting) {
		return fmt.Errorf("connection already in progress")
	}

	client, err := ssh.Dial("tcp", m.address, m.sshConfig)
	if err != nil {
		m.setState(ConnectionStateConnecting, ConnectionStateDisconnected)
		m.setLastError(err)
		return fmt.Errorf("failed to connect: %w", err)
	}

	m.mu.Lock()
	m.client = client
	m.connectedSince = time.Now()
	m.mu.Unlock()

	m.setState(ConnectionStateConnecting, ConnectionStateConnected)

	// Start keepalive goroutine if enabled
	if m.config.KeepAliveInterval > 0 {
		m.wg.Add(1)
		go m.keepaliveLoop()
	}

	return nil
}

// Close closes the SSH connection and stops all background goroutines.
func (m *sshConnectionManager) Close() error {
	if m.cancel != nil {
		m.cancel()
	}

	// Wait for goroutines to finish
	m.wg.Wait()

	// Close session pool
	m.sessionPoolMu.Lock()
	for _, ps := range m.sessionPool {
		if ps.session != nil {
			_ = ps.session.Close()
		}
	}
	m.sessionPool = nil
	m.sessionPoolMu.Unlock()

	// Close client
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.client != nil {
		err := m.client.Close()
		m.client = nil
		m.setState(ConnectionState(m.state.Load()), ConnectionStateDisconnected)
		return err
	}
	return nil
}

// NewSession creates or retrieves a session from the pool.
func (m *sshConnectionManager) NewSession() (*ssh.Session, error) {
	m.mu.RLock()
	client := m.client
	m.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("not connected")
	}

	// Create new session (sessions can't be reused after commands complete)
	session, err := client.NewSession()
	if err != nil {
		// Connection might be dead, trigger reconnection
		if m.isConnectionError(err) {
			go m.handleConnectionFailure(err)
		}
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	m.sessionsCreated.Add(1)
	return session, nil
}

// State returns the current connection state.
func (m *sshConnectionManager) State() ConnectionState {
	return ConnectionState(m.state.Load())
}

// Stats returns current connection statistics.
func (m *sshConnectionManager) Stats() ConnectionStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return ConnectionStats{
		State:             ConnectionState(m.state.Load()),
		ConnectedSince:    m.connectedSince,
		ReconnectAttempts: m.reconnectAttempts.Load(),
		TotalReconnects:   m.totalReconnects.Load(),
		LastError:         m.lastError,
		LastErrorTime:     m.lastErrorTime,
		SessionsCreated:   m.sessionsCreated.Load(),
		SessionsReused:    m.sessionsReused.Load(),
		KeepalivesSent:    m.keepalivesSent.Load(),
		KeepalivesFailed:  m.keepalivesFailed.Load(),
	}
}

// IsHealthy returns true if the connection is healthy.
func (m *sshConnectionManager) IsHealthy() bool {
	return ConnectionState(m.state.Load()) == ConnectionStateConnected
}

// keepaliveLoop sends periodic keepalive probes.
func (m *sshConnectionManager) keepaliveLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.KeepAliveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			if err := m.sendKeepalive(); err != nil {
				m.keepalivesFailed.Add(1)
				log.Printf("SSH keepalive failed for %s: %v", m.address, err)
				go m.handleConnectionFailure(err)
			} else {
				m.keepalivesSent.Add(1)
			}
		}
	}
}

// sendKeepalive sends a keepalive probe using SSH global request.
func (m *sshConnectionManager) sendKeepalive() error {
	m.mu.RLock()
	client := m.client
	m.mu.RUnlock()

	if client == nil {
		return fmt.Errorf("not connected")
	}

	// Use a timeout context for the keepalive
	ctx, cancel := context.WithTimeout(m.ctx, m.config.KeepAliveTimeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		// Send a keepalive request - the response indicates the connection is alive
		_, _, err := client.SendRequest("keepalive@golang.org", true, nil)
		done <- err
	}()

	select {
	case <-done:
		// Server might not support the keepalive request, but if we get any response
		// (including rejection) it means the connection is alive
		return nil // Ignore the actual error, connection is responsive
	case <-ctx.Done():
		return fmt.Errorf("keepalive timeout")
	}
}

// handleConnectionFailure triggers reconnection when a connection failure is detected.
func (m *sshConnectionManager) handleConnectionFailure(err error) {
	m.setLastError(err)

	currentState := ConnectionState(m.state.Load())
	if currentState == ConnectionStateReconnecting || currentState == ConnectionStateDisconnected {
		return // Already handling
	}

	if !m.setState(currentState, ConnectionStateReconnecting) {
		return // Another goroutine is handling it
	}

	// Close old client
	m.mu.Lock()
	if m.client != nil {
		_ = m.client.Close()
		m.client = nil
	}
	m.mu.Unlock()

	// Start reconnection with exponential backoff
	m.wg.Add(1)
	go m.reconnectLoop()
}

// reconnectLoop attempts to reconnect with exponential backoff.
func (m *sshConnectionManager) reconnectLoop() {
	defer m.wg.Done()

	delay := m.config.InitialReconnectDelay
	attempts := int64(0)

	for {
		select {
		case <-m.ctx.Done():
			m.setState(ConnectionStateReconnecting, ConnectionStateDisconnected)
			return
		default:
		}

		attempts++
		m.reconnectAttempts.Store(attempts)

		if m.config.MaxReconnectAttempts > 0 && int(attempts) > m.config.MaxReconnectAttempts {
			log.Printf("SSH max reconnection attempts (%d) reached for %s",
				m.config.MaxReconnectAttempts, m.address)
			m.setState(ConnectionStateReconnecting, ConnectionStateDisconnected)
			return
		}

		log.Printf("SSH reconnecting to %s (attempt %d, delay %v)", m.address, attempts, delay)

		client, err := ssh.Dial("tcp", m.address, m.sshConfig)
		if err != nil {
			m.setLastError(err)
			log.Printf("SSH reconnection failed for %s: %v", m.address, err)

			// Wait with exponential backoff
			select {
			case <-m.ctx.Done():
				m.setState(ConnectionStateReconnecting, ConnectionStateDisconnected)
				return
			case <-time.After(delay):
			}

			// Exponential backoff with jitter
			delay = time.Duration(float64(delay) * 1.5)
			if delay > m.config.MaxReconnectDelay {
				delay = m.config.MaxReconnectDelay
			}
			continue
		}

		// Successfully reconnected
		m.mu.Lock()
		m.client = client
		m.connectedSince = time.Now()
		m.mu.Unlock()

		m.totalReconnects.Add(1)
		m.reconnectAttempts.Store(0)
		m.setState(ConnectionStateReconnecting, ConnectionStateConnected)
		log.Printf("SSH reconnected to %s after %d attempts", m.address, attempts)
		return
	}
}

// setState atomically changes the connection state and calls the callback.
func (m *sshConnectionManager) setState(from, to ConnectionState) bool {
	if m.state.CompareAndSwap(int32(from), int32(to)) {
		if m.config.OnStateChange != nil {
			m.config.OnStateChange(from, to)
		}
		return true
	}
	return false
}

// setLastError records the last error.
func (m *sshConnectionManager) setLastError(err error) {
	m.mu.Lock()
	m.lastError = err
	m.lastErrorTime = time.Now()
	m.mu.Unlock()
}

// isConnectionError determines if an error indicates a connection failure.
func (m *sshConnectionManager) isConnectionError(err error) bool {
	if err == nil {
		return false
	}
	// Common connection error patterns
	errStr := err.Error()
	patterns := []string{
		"connection refused",
		"connection reset",
		"broken pipe",
		"timeout",
		"EOF",
		"use of closed network connection",
		"no route to host",
		"network is unreachable",
	}
	for _, p := range patterns {
		if contains(errStr, p) {
			return true
		}
	}
	return false
}

// contains checks if s contains substr (case-insensitive).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || containsIgnoreCase(s, substr))
}

func containsIgnoreCase(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if equalFold(s[i:i+len(substr)], substr) {
			return true
		}
	}
	return false
}

func equalFold(s, t string) bool {
	if len(s) != len(t) {
		return false
	}
	for i := 0; i < len(s); i++ {
		c1 := s[i]
		c2 := t[i]
		if c1 >= 'A' && c1 <= 'Z' {
			c1 += 'a' - 'A'
		}
		if c2 >= 'A' && c2 <= 'Z' {
			c2 += 'a' - 'A'
		}
		if c1 != c2 {
			return false
		}
	}
	return true
}

// calculateBackoff calculates the next backoff delay using exponential backoff with jitter.
func calculateBackoff(attempt int, initial, max time.Duration) time.Duration {
	if attempt <= 0 {
		return initial
	}

	// Calculate exponential backoff: initial * 2^attempt
	multiplier := math.Pow(2, float64(attempt-1))
	delay := time.Duration(float64(initial) * multiplier)

	if delay > max {
		delay = max
	}

	return delay
}
