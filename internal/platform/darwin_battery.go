//go:build darwin
// +build darwin

package platform

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// darwinBatteryProvider implements BatteryProvider for macOS/Darwin systems using pmset.
type darwinBatteryProvider struct{}

func newDarwinBatteryProvider() *darwinBatteryProvider {
	return &darwinBatteryProvider{}
}

// Count returns the number of batteries in the system.
func (b *darwinBatteryProvider) Count() int {
	// macOS typically has 0 or 1 battery (laptops vs desktops)
	// Use pmset to check if battery is present
	cmd := exec.Command("pmset", "-g", "batt")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	// Check if output contains battery information
	if strings.Contains(string(output), "InternalBattery") || strings.Contains(string(output), "Battery") {
		return 1
	}

	return 0
}

// Stats returns battery statistics for a specific battery index.
func (b *darwinBatteryProvider) Stats(index int) (*BatteryStats, error) {
	if index != 0 {
		return nil, fmt.Errorf("invalid battery index %d (macOS supports only index 0)", index)
	}

	// Use pmset -g batt to get battery information
	cmd := exec.Command("pmset", "-g", "batt")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("running pmset: %w", err)
	}

	return b.parsePmsetOutput(output)
}

// parsePmsetOutput parses the output of "pmset -g batt" command.
// Example output:
// Now drawing from 'Battery Power'
// -InternalBattery-0 (id=12345678)	95%; discharging; 5:23 remaining present: true
func (b *darwinBatteryProvider) parsePmsetOutput(output []byte) (*BatteryStats, error) {
	scanner := bufio.NewScanner(bytes.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()

		// Look for battery line (starts with -InternalBattery or contains Battery)
		if !strings.Contains(line, "InternalBattery") && !strings.Contains(line, "Battery") {
			continue
		}

		// Skip the "Now drawing from" line
		if strings.HasPrefix(line, "Now drawing from") {
			continue
		}

		// Parse the battery status line
		// Format: -InternalBattery-0 (id=12345678)	95%; discharging; 5:23 remaining present: true

		// Split by tab
		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			continue
		}

		// Parse the status part
		statusPart := parts[1]
		fields := strings.Split(statusPart, ";")

		stats := &BatteryStats{
			Charging: false,
		}

		// Parse percentage
		if len(fields) > 0 {
			percentStr := strings.TrimSpace(fields[0])
			percentStr = strings.TrimSuffix(percentStr, "%")
			if percent, err := strconv.ParseFloat(percentStr, 64); err == nil {
				stats.Percent = percent
			}
		}

		// Parse charging status
		if len(fields) > 1 {
			status := strings.TrimSpace(fields[1])
			stats.Charging = strings.Contains(strings.ToLower(status), "charging") &&
				!strings.Contains(strings.ToLower(status), "discharging")
		}

		// Parse time remaining
		if len(fields) > 2 {
			timeStr := strings.TrimSpace(fields[2])
			if strings.Contains(timeStr, "remaining") {
				// Extract time (format: "5:23 remaining")
				timeParts := strings.Fields(timeStr)
				if len(timeParts) > 0 {
					duration, err := parseTimeRemaining(timeParts[0])
					if err == nil {
						stats.TimeRemaining = duration
					}
				}
			}
		}

		return stats, nil
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("parsing pmset output: %w", err)
	}

	return nil, fmt.Errorf("no battery information found in pmset output")
}

// parseTimeRemaining converts a time string like "5:23" to time.Duration.
func parseTimeRemaining(timeStr string) (time.Duration, error) {
	parts := strings.Split(timeStr, ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid time format: %s", timeStr)
	}

	hours, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("parsing hours: %w", err)
	}

	minutes, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, fmt.Errorf("parsing minutes: %w", err)
	}

	return time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute, nil
}
