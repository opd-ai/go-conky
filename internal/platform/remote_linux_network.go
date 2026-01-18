package platform

import (
	"fmt"
	"strings"
)

// remoteLinuxNetworkProvider collects network metrics from remote Linux systems via SSH.
type remoteLinuxNetworkProvider struct {
	runner commandRunner
}

func newRemoteLinuxNetworkProvider(p *sshPlatform) *remoteLinuxNetworkProvider {
	return &remoteLinuxNetworkProvider{
		runner: p,
	}
}

// newTestableRemoteLinuxNetworkProviderWithRunner creates a provider with an injectable runner for testing.
func newTestableRemoteLinuxNetworkProviderWithRunner(runner commandRunner) *remoteLinuxNetworkProvider {
	return &remoteLinuxNetworkProvider{
		runner: runner,
	}
}

func (n *remoteLinuxNetworkProvider) Interfaces() ([]string, error) {
	output, err := n.runner.runCommand("cat /proc/net/dev | tail -n +3")
	if err != nil {
		return nil, fmt.Errorf("failed to read /proc/net/dev: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	interfaces := make([]string, 0, len(lines))

	for _, line := range lines {
		parts := strings.Split(line, ":")
		if len(parts) < 2 {
			continue
		}
		iface := strings.TrimSpace(parts[0])
		interfaces = append(interfaces, iface)
	}

	return interfaces, nil
}

func (n *remoteLinuxNetworkProvider) Stats(interfaceName string) (*NetworkStats, error) {
	// Read all interfaces and search for the requested one to avoid command injection
	output, err := n.runner.runCommand("cat /proc/net/dev | tail -n +3")
	if err != nil {
		return nil, fmt.Errorf("failed to read network stats for %s: %w", interfaceName, err)
	}

	var line string
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, l := range lines {
		parts := strings.Split(l, ":")
		if len(parts) < 2 {
			continue
		}
		iface := strings.TrimSpace(parts[0])
		if iface == interfaceName {
			line = l
			break
		}
	}

	line = strings.TrimSpace(line)
	if line == "" {
		return nil, fmt.Errorf("interface %s not found", interfaceName)
	}

	// Parse /proc/net/dev format:
	// interface: bytes packets errs drop fifo frame compressed multicast
	parts := strings.Split(line, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("unexpected /proc/net/dev format: %s", line)
	}

	fields := strings.Fields(parts[1])
	if len(fields) < 16 {
		return nil, fmt.Errorf("unexpected /proc/net/dev field count: %d", len(fields))
	}

	stats := &NetworkStats{
		BytesRecv:   parseUint64(fields[0]),
		PacketsRecv: parseUint64(fields[1]),
		ErrorsIn:    parseUint64(fields[2]),
		DropIn:      parseUint64(fields[3]),
		BytesSent:   parseUint64(fields[8]),
		PacketsSent: parseUint64(fields[9]),
		ErrorsOut:   parseUint64(fields[10]),
		DropOut:     parseUint64(fields[11]),
	}

	return stats, nil
}

func (n *remoteLinuxNetworkProvider) AllStats() (map[string]*NetworkStats, error) {
	interfaces, err := n.Interfaces()
	if err != nil {
		return nil, err
	}

	stats := make(map[string]*NetworkStats)
	for _, iface := range interfaces {
		ifaceStats, err := n.Stats(iface)
		if err != nil {
			continue // Skip interfaces we can't read
		}
		stats[iface] = ifaceStats
	}

	return stats, nil
}
