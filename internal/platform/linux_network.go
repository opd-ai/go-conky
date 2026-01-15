//go:build linux && !android
// +build linux,!android

package platform

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
)

// linuxNetworkProvider implements NetworkProvider for Linux systems by reading /proc/net/dev.
type linuxNetworkProvider struct {
	mu             sync.Mutex
	procNetDevPath string
}

// rawNetStats stores raw network counters for rate calculation.
type rawNetStats struct {
	bytesRecv   uint64
	packetsRecv uint64
	errorsIn    uint64
	dropIn      uint64
	bytesSent   uint64
	packetsSent uint64
	errorsOut   uint64
	dropOut     uint64
}

func newLinuxNetworkProvider() *linuxNetworkProvider {
	return &linuxNetworkProvider{
		procNetDevPath: "/proc/net/dev",
	}
}

func (n *linuxNetworkProvider) Interfaces() ([]string, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	stats, err := n.readProcNetDev()
	if err != nil {
		return nil, err
	}

	interfaces := make([]string, 0, len(stats))
	for name := range stats {
		interfaces = append(interfaces, name)
	}

	return interfaces, nil
}

func (n *linuxNetworkProvider) Stats(interfaceName string) (*NetworkStats, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	currentStats, err := n.readProcNetDev()
	if err != nil {
		return nil, err
	}

	curr, found := currentStats[interfaceName]
	if !found {
		return nil, fmt.Errorf("interface %s not found", interfaceName)
	}

	return &NetworkStats{
		BytesRecv:   curr.bytesRecv,
		BytesSent:   curr.bytesSent,
		PacketsRecv: curr.packetsRecv,
		PacketsSent: curr.packetsSent,
		ErrorsIn:    curr.errorsIn,
		ErrorsOut:   curr.errorsOut,
		DropIn:      curr.dropIn,
		DropOut:     curr.dropOut,
	}, nil
}

func (n *linuxNetworkProvider) AllStats() (map[string]*NetworkStats, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	currentStats, err := n.readProcNetDev()
	if err != nil {
		return nil, err
	}

	result := make(map[string]*NetworkStats, len(currentStats))
	for name, curr := range currentStats {
		result[name] = &NetworkStats{
			BytesRecv:   curr.bytesRecv,
			BytesSent:   curr.bytesSent,
			PacketsRecv: curr.packetsRecv,
			PacketsSent: curr.packetsSent,
			ErrorsIn:    curr.errorsIn,
			ErrorsOut:   curr.errorsOut,
			DropIn:      curr.dropIn,
			DropOut:     curr.dropOut,
		}
	}

	return result, nil
}

// readProcNetDev parses /proc/net/dev and returns raw network statistics.
func (n *linuxNetworkProvider) readProcNetDev() (map[string]rawNetStats, error) {
	file, err := os.Open(n.procNetDevPath)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", n.procNetDevPath, err)
	}
	defer file.Close()

	stats := make(map[string]rawNetStats)
	scanner := bufio.NewScanner(file)

	// Skip header lines
	for i := 0; i < 2 && scanner.Scan(); i++ {
		// Skip the two header lines
	}

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		interfaceName := strings.TrimSpace(parts[0])
		fields := strings.Fields(parts[1])
		if len(fields) < 16 {
			continue
		}

		stats[interfaceName] = rawNetStats{
			bytesRecv:   parseUint64(fields[0]),
			packetsRecv: parseUint64(fields[1]),
			errorsIn:    parseUint64(fields[2]),
			dropIn:      parseUint64(fields[3]),
			bytesSent:   parseUint64(fields[8]),
			packetsSent: parseUint64(fields[9]),
			errorsOut:   parseUint64(fields[10]),
			dropOut:     parseUint64(fields[11]),
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning %s: %w", n.procNetDevPath, err)
	}

	return stats, nil
}
