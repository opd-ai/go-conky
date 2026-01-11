package monitor

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// bytesPerKB is the number of bytes in a kilobyte.
const bytesPerKB = 1024

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

		// Convert kB to bytes with overflow check
		maxBeforeMultiply := ^uint64(0) / bytesPerKB
		if value > maxBeforeMultiply {
			// Skip values that would overflow when converted to bytes
			continue
		}
		values[key] = value * bytesPerKB
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

	stats.Used = calculateUsedMemory(stats.Total, stats.Free, stats.Buffers, stats.Cached)
	stats.SwapUsed = safeSubtract(stats.SwapTotal, stats.SwapFree)

	// Calculate percentages
	if stats.Total > 0 {
		stats.UsagePercent = float64(stats.Used) / float64(stats.Total) * 100.0
	}
	if stats.SwapTotal > 0 {
		stats.SwapPercent = float64(stats.SwapUsed) / float64(stats.SwapTotal) * 100.0
	}

	return stats, nil
}

// calculateUsedMemory calculates used memory using safe stepwise subtraction
// to prevent underflow and avoid overflow in additions.
func calculateUsedMemory(total, free, buffers, cached uint64) uint64 {
	if total < free {
		return 0
	}
	remainingAfterFree := total - free

	if remainingAfterFree < buffers {
		return total - free
	}
	remainingAfterBuffers := remainingAfterFree - buffers

	if remainingAfterBuffers < cached {
		return total - free
	}
	return remainingAfterBuffers - cached
}

// safeSubtract performs subtraction with underflow protection.
func safeSubtract(a, b uint64) uint64 {
	if a >= b {
		return a - b
	}
	return 0
}
