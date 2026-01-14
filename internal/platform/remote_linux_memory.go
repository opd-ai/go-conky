package platform

import (
	"fmt"
	"strconv"
	"strings"
)

// remoteLinuxMemoryProvider collects memory metrics from remote Linux systems via SSH.
type remoteLinuxMemoryProvider struct {
	platform *sshPlatform
}

func newRemoteLinuxMemoryProvider(p *sshPlatform) *remoteLinuxMemoryProvider {
	return &remoteLinuxMemoryProvider{
		platform: p,
	}
}

func (m *remoteLinuxMemoryProvider) Stats() (*MemoryStats, error) {
	output, err := m.platform.runCommand("cat /proc/meminfo")
	if err != nil {
		return nil, fmt.Errorf("failed to read /proc/meminfo: %w", err)
	}

	stats := &MemoryStats{}
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		key := strings.TrimSuffix(fields[0], ":")
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}

		// Values in /proc/meminfo are in KB, convert to bytes
		value *= 1024

		switch key {
		case "MemTotal":
			stats.Total = value
		case "MemFree":
			stats.Free = value
		case "MemAvailable":
			stats.Available = value
		case "Buffers":
			stats.Buffers = value
		case "Cached":
			stats.Cached = value
		}
	}

	// Calculate used memory
	stats.Used = stats.Total - stats.Free - stats.Buffers - stats.Cached

	// Calculate percentage
	if stats.Total > 0 {
		stats.UsedPercent = float64(stats.Used) / float64(stats.Total) * 100
	}

	return stats, nil
}

func (m *remoteLinuxMemoryProvider) SwapStats() (*SwapStats, error) {
	output, err := m.platform.runCommand("cat /proc/meminfo | grep '^Swap'")
	if err != nil {
		return nil, fmt.Errorf("failed to read swap stats: %w", err)
	}

	stats := &SwapStats{}
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		key := strings.TrimSuffix(fields[0], ":")
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}

		// Values in /proc/meminfo are in KB, convert to bytes
		value *= 1024

		switch key {
		case "SwapTotal":
			stats.Total = value
		case "SwapFree":
			stats.Free = value
		}
	}

	// Calculate used swap
	if stats.Total > stats.Free {
		stats.Used = stats.Total - stats.Free
	}

	// Calculate percentage
	if stats.Total > 0 {
		stats.UsedPercent = float64(stats.Used) / float64(stats.Total) * 100
	}

	return stats, nil
}
