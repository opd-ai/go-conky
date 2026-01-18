package platform

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// parseMemInfoOutput parses the output of /proc/meminfo and returns MemoryStats.
// This function is used by both local and remote Linux providers.
func parseMemInfoOutput(output string) (*MemoryStats, error) {
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

// parseSwapOutput parses the swap-related lines from /proc/meminfo and returns SwapStats.
func parseSwapOutput(output string) (*SwapStats, error) {
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

// parseLoadAverage parses the output of /proc/loadavg and returns load averages.
func parseLoadAverage(output string) (float64, float64, float64, error) {
	fields := strings.Fields(output)
	if len(fields) < 3 {
		return 0, 0, 0, fmt.Errorf("unexpected /proc/loadavg format: %s", output)
	}

	load1, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to parse 1min load: %w", err)
	}

	load5, err := strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to parse 5min load: %w", err)
	}

	load15, err := strconv.ParseFloat(fields[2], 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to parse 15min load: %w", err)
	}

	return load1, load5, load15, nil
}

// parseTotalCPUUsage parses the first line of /proc/stat and calculates CPU usage.
// It uses previous stats stored in the map (key -1 for aggregate CPU).
func parseTotalCPUUsage(output string, prevStats map[int]cpuTimes, mu *sync.Mutex) (float64, error) {
	fields := strings.Fields(output)
	if len(fields) < 5 {
		return 0, fmt.Errorf("unexpected /proc/stat format: %s", output)
	}

	mu.Lock()
	defer mu.Unlock()

	current := cpuTimes{
		user:   parseUint64(fields[1]),
		nice:   parseUint64(fields[2]),
		system: parseUint64(fields[3]),
		idle:   parseUint64(fields[4]),
	}
	if len(fields) > 5 {
		current.iowait = parseUint64(fields[5])
	}
	if len(fields) > 6 {
		current.irq = parseUint64(fields[6])
	}
	if len(fields) > 7 {
		current.softirq = parseUint64(fields[7])
	}
	if len(fields) > 8 {
		current.steal = parseUint64(fields[8])
	}

	prev, exists := prevStats[-1] // -1 for aggregate CPU
	prevStats[-1] = current

	if !exists {
		return 0, nil
	}

	// Calculate usage percentage
	totalDelta := float64(
		(current.user - prev.user) +
			(current.nice - prev.nice) +
			(current.system - prev.system) +
			(current.idle - prev.idle) +
			(current.iowait - prev.iowait) +
			(current.irq - prev.irq) +
			(current.softirq - prev.softirq) +
			(current.steal - prev.steal))

	idleDelta := float64(current.idle - prev.idle + current.iowait - prev.iowait)

	if totalDelta > 0 {
		return 100 * (1 - idleDelta/totalDelta), nil
	}
	return 0, nil
}
