package platform

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// linuxMemoryProvider implements MemoryProvider for Linux systems by reading /proc/meminfo.
type linuxMemoryProvider struct {
	procMemInfoPath string
}

func newLinuxMemoryProvider() *linuxMemoryProvider {
	return &linuxMemoryProvider{
		procMemInfoPath: "/proc/meminfo",
	}
}

func (m *linuxMemoryProvider) Stats() (*MemoryStats, error) {
	file, err := os.Open(m.procMemInfoPath)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", m.procMemInfoPath, err)
	}
	defer file.Close()

	values := make(map[string]uint64)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		valueStr := strings.TrimSpace(parts[1])

		// Remove "kB" suffix and parse
		valueStr = strings.TrimSuffix(valueStr, " kB")
		value, err := strconv.ParseUint(valueStr, 10, 64)
		if err != nil {
			continue
		}

		// Convert kB to bytes
		const bytesPerKB = 1024
		maxBeforeMultiply := ^uint64(0) / bytesPerKB
		if value > maxBeforeMultiply {
			// Skip values that would overflow
			continue
		}
		values[key] = value * bytesPerKB
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning %s: %w", m.procMemInfoPath, err)
	}

	total := values["MemTotal"]
	free := values["MemFree"]
	available := values["MemAvailable"]
	buffers := values["Buffers"]
	cached := values["Cached"]

	// Calculate used memory
	var used uint64
	if total >= free {
		used = total - free
		if used >= buffers {
			used -= buffers
		}
		if used >= cached {
			used -= cached
		}
	}

	// Calculate usage percentage
	var usedPercent float64
	if total > 0 {
		usedPercent = float64(used) / float64(total) * 100.0
	}

	return &MemoryStats{
		Total:       total,
		Used:        used,
		Free:        free,
		Available:   available,
		Cached:      cached,
		Buffers:     buffers,
		UsedPercent: usedPercent,
	}, nil
}

func (m *linuxMemoryProvider) SwapStats() (*SwapStats, error) {
	file, err := os.Open(m.procMemInfoPath)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", m.procMemInfoPath, err)
	}
	defer file.Close()

	values := make(map[string]uint64)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		valueStr := strings.TrimSpace(parts[1])

		// Remove "kB" suffix and parse
		valueStr = strings.TrimSuffix(valueStr, " kB")
		value, err := strconv.ParseUint(valueStr, 10, 64)
		if err != nil {
			continue
		}

		// Convert kB to bytes
		const bytesPerKB = 1024
		maxBeforeMultiply := ^uint64(0) / bytesPerKB
		if value > maxBeforeMultiply {
			// Skip values that would overflow
			continue
		}
		values[key] = value * bytesPerKB
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning %s: %w", m.procMemInfoPath, err)
	}

	total := values["SwapTotal"]
	free := values["SwapFree"]

	// Calculate used swap
	var used uint64
	if total >= free {
		used = total - free
	}

	// Calculate usage percentage
	var usedPercent float64
	if total > 0 {
		usedPercent = float64(used) / float64(total) * 100.0
	}

	return &SwapStats{
		Total:       total,
		Used:        used,
		Free:        free,
		UsedPercent: usedPercent,
	}, nil
}
