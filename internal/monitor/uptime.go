package monitor

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// uptimeReader reads uptime statistics from /proc filesystem.
type uptimeReader struct {
	procUptimePath string
}

// newUptimeReader creates a new uptimeReader with default paths.
func newUptimeReader() *uptimeReader {
	return &uptimeReader{
		procUptimePath: "/proc/uptime",
	}
}

// ReadStats reads current uptime statistics from /proc/uptime.
func (r *uptimeReader) ReadStats() (UptimeStats, error) {
	data, err := os.ReadFile(r.procUptimePath)
	if err != nil {
		return UptimeStats{}, fmt.Errorf("reading %s: %w", r.procUptimePath, err)
	}

	fields := strings.Fields(string(data))
	if len(fields) < 2 {
		return UptimeStats{}, fmt.Errorf("invalid format in %s: expected 2 fields, got %d", r.procUptimePath, len(fields))
	}

	uptime, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return UptimeStats{}, fmt.Errorf("parsing uptime value: %w", err)
	}

	idle, err := strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return UptimeStats{}, fmt.Errorf("parsing idle value: %w", err)
	}

	return UptimeStats{
		Seconds:     uptime,
		IdleSeconds: idle,
		Duration:    time.Duration(uptime * float64(time.Second)),
	}, nil
}
