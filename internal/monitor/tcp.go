// Package monitor provides TCP connection monitoring for Linux systems.
// It reads TCP connection info from /proc/net/tcp and /proc/net/tcp6.
package monitor

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
)

// TCPConnection represents a single TCP connection.
type TCPConnection struct {
	LocalIP    string
	LocalPort  int
	RemoteIP   string
	RemotePort int
	State      string
	UID        int
}

// TCPState constants matching /proc/net/tcp state values.
const (
	TCPEstablished = "01"
	TCPSynSent     = "02"
	TCPSynRecv     = "03"
	TCPFinWait1    = "04"
	TCPFinWait2    = "05"
	TCPTimeWait    = "06"
	TCPClose       = "07"
	TCPCloseWait   = "08"
	TCPLastAck     = "09"
	TCPListen      = "0A"
	TCPClosing     = "0B"
)

// TCPStats contains TCP connection statistics.
type TCPStats struct {
	Connections []TCPConnection
	TotalCount  int
	ListenCount int
}

// tcpReader reads TCP connection info from /proc filesystem.
type tcpReader struct {
	mu           sync.RWMutex
	procTCPPath  string
	procTCP6Path string
}

// newTCPReader creates a new tcpReader with default paths.
func newTCPReader() *tcpReader {
	return &tcpReader{
		procTCPPath:  "/proc/net/tcp",
		procTCP6Path: "/proc/net/tcp6",
	}
}

// ReadStats reads all TCP connections.
func (r *tcpReader) ReadStats() (TCPStats, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	stats := TCPStats{}

	// Read IPv4 connections
	conns4, err := r.readProcTCP(r.procTCPPath, false)
	if err == nil {
		stats.Connections = append(stats.Connections, conns4...)
	}

	// Read IPv6 connections
	conns6, err := r.readProcTCP(r.procTCP6Path, true)
	if err == nil {
		stats.Connections = append(stats.Connections, conns6...)
	}

	stats.TotalCount = len(stats.Connections)
	for _, conn := range stats.Connections {
		if conn.State == "LISTEN" {
			stats.ListenCount++
		}
	}

	return stats, nil
}

// CountInRange counts connections in the given port range.
func (r *tcpReader) CountInRange(minPort, maxPort int) int {
	stats, err := r.ReadStats()
	if err != nil {
		return 0
	}

	count := 0
	for _, conn := range stats.Connections {
		if conn.LocalPort >= minPort && conn.LocalPort <= maxPort {
			count++
		}
	}
	return count
}

// GetConnectionByIndex returns connection info at given index for port range.
func (r *tcpReader) GetConnectionByIndex(minPort, maxPort, index int) *TCPConnection {
	stats, err := r.ReadStats()
	if err != nil {
		return nil
	}

	var matching []TCPConnection
	for _, conn := range stats.Connections {
		if conn.LocalPort >= minPort && conn.LocalPort <= maxPort {
			matching = append(matching, conn)
		}
	}

	if index < 0 || index >= len(matching) {
		return nil
	}
	return &matching[index]
}

// readProcTCP parses /proc/net/tcp or /proc/net/tcp6.
func (r *tcpReader) readProcTCP(path string, isIPv6 bool) ([]TCPConnection, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var connections []TCPConnection
	scanner := bufio.NewScanner(file)

	// Skip header line
	if !scanner.Scan() {
		return connections, nil
	}

	for scanner.Scan() {
		conn, err := r.parseTCPLine(scanner.Text(), isIPv6)
		if err != nil {
			continue
		}
		connections = append(connections, conn)
	}

	return connections, scanner.Err()
}

// parseTCPLine parses a single line from /proc/net/tcp.
// Format: sl local_address rem_address st tx_queue rx_queue tr tm->when retrnsmt uid timeout inode
func (r *tcpReader) parseTCPLine(line string, isIPv6 bool) (TCPConnection, error) {
	fields := strings.Fields(line)
	if len(fields) < 10 {
		return TCPConnection{}, fmt.Errorf("insufficient fields")
	}

	localIP, localPort, err := r.parseAddress(fields[1], isIPv6)
	if err != nil {
		return TCPConnection{}, err
	}

	remoteIP, remotePort, err := r.parseAddress(fields[2], isIPv6)
	if err != nil {
		return TCPConnection{}, err
	}

	uid, _ := strconv.Atoi(fields[7])

	return TCPConnection{
		LocalIP:    localIP,
		LocalPort:  localPort,
		RemoteIP:   remoteIP,
		RemotePort: remotePort,
		State:      r.stateToString(fields[3]),
		UID:        uid,
	}, nil
}

// parseAddress parses hex address:port format.
func (r *tcpReader) parseAddress(addr string, isIPv6 bool) (string, int, error) {
	parts := strings.Split(addr, ":")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid address format")
	}

	port, err := strconv.ParseInt(parts[1], 16, 32)
	if err != nil {
		return "", 0, err
	}

	ip, err := r.hexToIP(parts[0], isIPv6)
	if err != nil {
		return "", 0, err
	}

	return ip, int(port), nil
}

// hexToIP converts hex IP address to string.
func (r *tcpReader) hexToIP(hexIP string, isIPv6 bool) (string, error) {
	bytes, err := hex.DecodeString(hexIP)
	if err != nil {
		return "", err
	}

	if isIPv6 {
		if len(bytes) != 16 {
			return "", fmt.Errorf("invalid IPv6 length")
		}
		// IPv6 is stored in 4-byte groups, little-endian within groups
		for i := 0; i < 16; i += 4 {
			bytes[i], bytes[i+3] = bytes[i+3], bytes[i]
			bytes[i+1], bytes[i+2] = bytes[i+2], bytes[i+1]
		}
		return net.IP(bytes).String(), nil
	}

	// IPv4: stored in little-endian
	if len(bytes) != 4 {
		return "", fmt.Errorf("invalid IPv4 length")
	}
	// Reverse byte order
	return fmt.Sprintf("%d.%d.%d.%d", bytes[3], bytes[2], bytes[1], bytes[0]), nil
}

// stateToString converts state hex to readable string.
func (r *tcpReader) stateToString(state string) string {
	switch state {
	case TCPEstablished:
		return "ESTABLISHED"
	case TCPSynSent:
		return "SYN_SENT"
	case TCPSynRecv:
		return "SYN_RECV"
	case TCPFinWait1:
		return "FIN_WAIT1"
	case TCPFinWait2:
		return "FIN_WAIT2"
	case TCPTimeWait:
		return "TIME_WAIT"
	case TCPClose:
		return "CLOSE"
	case TCPCloseWait:
		return "CLOSE_WAIT"
	case TCPLastAck:
		return "LAST_ACK"
	case TCPListen:
		return "LISTEN"
	case TCPClosing:
		return "CLOSING"
	default:
		return "UNKNOWN"
	}
}
