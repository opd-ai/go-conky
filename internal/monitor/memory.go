package monitor

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// memoryReader reads memory statistics from /proc filesystem.
type memoryReader struct {
	procMemInfoPath string
}

// newMemoryReader creates a new memoryReader with default paths.
func newMemoryReader() *memoryReader {
	return &memoryReader{
		procMemInfoPath: "/proc/meminfo",
	}
}

// ReadStats reads current memory statistics from /proc/meminfo.
func (r *memoryReader) ReadStats() (MemoryStats, error) {
	file, err := os.Open(r.procMemInfoPath)
	if err != nil {
		return MemoryStats{}, fmt.Errorf("opening %s: %w", r.procMemInfoPath, err)
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
		values[key] = value * 1024
	}

	if err := scanner.Err(); err != nil {
		return MemoryStats{}, fmt.Errorf("scanning %s: %w", r.procMemInfoPath, err)
	}

	stats := MemoryStats{
		Total:     values["MemTotal"],
		Free:      values["MemFree"],
		Available: values["MemAvailable"],
		Buffers:   values["Buffers"],
		Cached:    values["Cached"],
		SwapTotal: values["SwapTotal"],
		SwapFree:  values["SwapFree"],
	}

	// Calculate used memory (total - free - buffers - cached)
	// Use safe subtraction to prevent underflow with unsigned integers
	if stats.Total >= stats.Free+stats.Buffers+stats.Cached {
		stats.Used = stats.Total - stats.Free - stats.Buffers - stats.Cached
	} else if stats.Total >= stats.Free {
		stats.Used = stats.Total - stats.Free
	} else {
		stats.Used = 0
	}

	// Calculate swap used
	stats.SwapUsed = stats.SwapTotal - stats.SwapFree

	// Calculate percentages
	if stats.Total > 0 {
		stats.UsagePercent = float64(stats.Used) / float64(stats.Total) * 100.0
	}
	if stats.SwapTotal > 0 {
		stats.SwapPercent = float64(stats.SwapUsed) / float64(stats.SwapTotal) * 100.0
	}

	return stats, nil
}
