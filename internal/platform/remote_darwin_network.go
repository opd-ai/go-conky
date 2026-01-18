package platform

import (
	"fmt"
	"strings"
)

// remoteDarwinNetworkProvider collects network metrics from remote macOS systems via SSH.
type remoteDarwinNetworkProvider struct {
	runner commandRunner
}

func newRemoteDarwinNetworkProvider(p *sshPlatform) *remoteDarwinNetworkProvider {
	return &remoteDarwinNetworkProvider{
		runner: p,
	}
}

// newTestableRemoteDarwinNetworkProviderWithRunner creates a provider with an injectable runner for testing.
func newTestableRemoteDarwinNetworkProviderWithRunner(runner commandRunner) *remoteDarwinNetworkProvider {
	return &remoteDarwinNetworkProvider{
		runner: runner,
	}
}

func (n *remoteDarwinNetworkProvider) Interfaces() ([]string, error) {
	output, err := n.runner.runCommand("netstat -i | tail -n +2 | awk '{print $1}' | sort -u")
	if err != nil {
		return nil, fmt.Errorf("failed to list network interfaces: %w", err)
	}

	var interfaces []string
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		iface := strings.TrimSpace(line)
		if iface != "" && iface != "Name" {
			interfaces = append(interfaces, iface)
		}
	}

	return interfaces, nil
}

func (n *remoteDarwinNetworkProvider) Stats(interfaceName string) (*NetworkStats, error) {
	// Use shell-escaped interface name
	cmd := fmt.Sprintf("netstat -ib -I %s | tail -n 1", shellEscape(interfaceName))
	output, err := n.runner.runCommand(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to read network stats for %s: %w", interfaceName, err)
	}

	fields := strings.Fields(output)
	if len(fields) < 10 {
		return nil, fmt.Errorf("unexpected netstat output format: %s", output)
	}

	// netstat -ib output format:
	// Name Mtu Network Address Ipkts Ierrs Ibytes Opkts Oerrs Obytes Coll
	stats := &NetworkStats{
		PacketsRecv: parseUint64(fields[4]),
		ErrorsIn:    parseUint64(fields[5]),
		BytesRecv:   parseUint64(fields[6]),
		PacketsSent: parseUint64(fields[7]),
		ErrorsOut:   parseUint64(fields[8]),
		BytesSent:   parseUint64(fields[9]),
	}

	return stats, nil
}

func (n *remoteDarwinNetworkProvider) AllStats() (map[string]*NetworkStats, error) {
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
