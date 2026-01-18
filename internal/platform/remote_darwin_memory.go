package platform

import (
	"fmt"
	"strconv"
	"strings"
)

// remoteDarwinMemoryProvider collects memory metrics from remote macOS systems via SSH.
type remoteDarwinMemoryProvider struct {
	runner commandRunner
}

func newRemoteDarwinMemoryProvider(p *sshPlatform) *remoteDarwinMemoryProvider {
	return &remoteDarwinMemoryProvider{
		runner: p,
	}
}

// newTestableRemoteDarwinMemoryProviderWithRunner creates a provider with an injectable runner for testing.
func newTestableRemoteDarwinMemoryProviderWithRunner(runner commandRunner) *remoteDarwinMemoryProvider {
	return &remoteDarwinMemoryProvider{
		runner: runner,
	}
}

func (m *remoteDarwinMemoryProvider) Stats() (*MemoryStats, error) {
	// Get total memory
	totalOutput, err := m.runner.runCommand("sysctl -n hw.memsize")
	if err != nil {
		return nil, fmt.Errorf("failed to read total memory: %w", err)
	}

	total, err := strconv.ParseUint(strings.TrimSpace(totalOutput), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse total memory: %w", err)
	}

	// Get memory statistics from vm_stat
	output, err := m.runner.runCommand("vm_stat")
	if err != nil {
		return nil, fmt.Errorf("failed to read vm_stat: %w", err)
	}

	stats := &MemoryStats{
		Total: total,
	}

	// Parse vm_stat output
	lines := strings.Split(output, "\n")
	pageSize := uint64(4096) // Default page size

	for _, line := range lines {
		if strings.Contains(line, "page size of") {
			// Extract page size from first line
			parts := strings.Fields(line)
			if len(parts) >= 8 {
				if ps, err := strconv.ParseUint(parts[7], 10, 64); err == nil {
					pageSize = ps
				}
			}
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		valueStr := strings.TrimSpace(strings.TrimSuffix(parts[1], "."))
		value, err := strconv.ParseUint(valueStr, 10, 64)
		if err != nil {
			continue
		}

		// Convert pages to bytes
		bytes := value * pageSize

		switch key {
		case "Pages free":
			stats.Free += bytes
		case "Pages inactive":
			stats.Free += bytes
		case "Pages active":
			stats.Used += bytes
		case "Pages wired down":
			stats.Used += bytes
		case "File-backed pages":
			stats.Cached += bytes
		}
	}

	// Calculate available memory
	stats.Available = stats.Free + stats.Cached

	// Calculate percentage
	if stats.Total > 0 {
		stats.UsedPercent = float64(stats.Used) / float64(stats.Total) * 100
	}

	return stats, nil
}

func (m *remoteDarwinMemoryProvider) SwapStats() (*SwapStats, error) {
	output, err := m.runner.runCommand("sysctl -n vm.swapusage")
	if err != nil {
		return nil, fmt.Errorf("failed to read swap usage: %w", err)
	}

	// Output format: total = 2048.00M  used = 512.00M  free = 1536.00M
	stats := &SwapStats{}
	parts := strings.Fields(output)

	for i, part := range parts {
		switch part {
		case "total":
			if i+2 < len(parts) {
				if total, err := parseMemorySize(parts[i+2]); err == nil {
					stats.Total = total
				}
			}
		case "used":
			if i+2 < len(parts) {
				if used, err := parseMemorySize(parts[i+2]); err == nil {
					stats.Used = used
				}
			}
		case "free":
			if i+2 < len(parts) {
				if free, err := parseMemorySize(parts[i+2]); err == nil {
					stats.Free = free
				}
			}
		}
	}

	// Calculate percentage
	if stats.Total > 0 {
		stats.UsedPercent = float64(stats.Used) / float64(stats.Total) * 100
	}

	return stats, nil
}

// parseMemorySize parses memory sizes like "2048.00M" or "1.5G"
func parseMemorySize(s string) (uint64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty size string")
	}

	// Extract unit (last character)
	unit := s[len(s)-1]
	valueStr := s[:len(s)-1]

	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return 0, err
	}

	switch unit {
	case 'K', 'k':
		return uint64(value * 1024), nil
	case 'M', 'm':
		return uint64(value * 1024 * 1024), nil
	case 'G', 'g':
		return uint64(value * 1024 * 1024 * 1024), nil
	default:
		// Try parsing as plain number (bytes)
		return strconv.ParseUint(s, 10, 64)
	}
}
