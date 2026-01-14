package platform

import (
	"fmt"
	"strconv"
	"strings"
)

// remoteLinuxSensorProvider collects sensor metrics from remote Linux systems via SSH.
type remoteLinuxSensorProvider struct {
	platform *sshPlatform
}

func newRemoteLinuxSensorProvider(p *sshPlatform) *remoteLinuxSensorProvider {
	return &remoteLinuxSensorProvider{
		platform: p,
	}
}

func (s *remoteLinuxSensorProvider) Temperatures() ([]SensorReading, error) {
	// Try to read from /sys/class/hwmon
	output, err := s.platform.runCommand("find /sys/class/hwmon -name 'temp*_input' 2>/dev/null")
	if err != nil {
		return nil, fmt.Errorf("failed to find temperature sensors: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return []SensorReading{}, nil
	}

	var readings []SensorReading

	for _, path := range lines {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}

		// Validate the path to prevent command injection
		if !validatePath(path) {
			continue
		}

		// Read temperature value using shell-escaped path
		tempOutput, err := s.platform.runCommand(fmt.Sprintf("cat %s", shellEscape(path)))
		if err != nil {
			continue
		}

		tempStr := strings.TrimSpace(tempOutput)
		temp, err := strconv.ParseFloat(tempStr, 64)
		if err != nil {
			continue
		}

		// Temperature is in millidegrees Celsius, convert to degrees
		temp /= 1000

		// Try to get label
		labelPath := strings.Replace(path, "_input", "_label", 1)
		labelOutput, err := s.platform.runCommand(fmt.Sprintf("cat %s 2>/dev/null || echo ''", shellEscape(labelPath)))
		label := extractSensorName(path)
		if err == nil && strings.TrimSpace(labelOutput) != "" {
			label = strings.TrimSpace(labelOutput)
		}

		reading := SensorReading{
			Name:  extractSensorName(path),
			Label: label,
			Value: temp,
			Unit:  "°C",
		}

		// Try to get critical threshold
		critPath := strings.Replace(path, "_input", "_crit", 1)
		critOutput, err := s.platform.runCommand(fmt.Sprintf("cat %s 2>/dev/null || echo ''", shellEscape(critPath)))
		if err == nil && strings.TrimSpace(critOutput) != "" {
			if crit, err := strconv.ParseFloat(strings.TrimSpace(critOutput), 64); err == nil {
				reading.Critical = crit / 1000
			}
		}

		readings = append(readings, reading)
	}

	return readings, nil
}

func (s *remoteLinuxSensorProvider) Fans() ([]SensorReading, error) {
	// Try to read from /sys/class/hwmon
	output, err := s.platform.runCommand("find /sys/class/hwmon -name 'fan*_input' 2>/dev/null")
	if err != nil {
		return nil, fmt.Errorf("failed to find fan sensors: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return []SensorReading{}, nil
	}

	var readings []SensorReading

	for _, path := range lines {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}

		// Validate the path to prevent command injection
		if !validatePath(path) {
			continue
		}

		// Read fan speed value using shell-escaped path
		speedOutput, err := s.platform.runCommand(fmt.Sprintf("cat %s", shellEscape(path)))
		if err != nil {
			continue
		}

		speedStr := strings.TrimSpace(speedOutput)
		speed, err := strconv.ParseFloat(speedStr, 64)
		if err != nil {
			continue
		}

		// Try to get label
		labelPath := strings.Replace(path, "_input", "_label", 1)
		labelOutput, err := s.platform.runCommand(fmt.Sprintf("cat %s 2>/dev/null || echo ''", shellEscape(labelPath)))
		label := extractSensorName(path)
		if err == nil && strings.TrimSpace(labelOutput) != "" {
			label = strings.TrimSpace(labelOutput)
		}

		reading := SensorReading{
			Name:  extractSensorName(path),
			Label: label,
			Value: speed,
			Unit:  "RPM",
		}

		readings = append(readings, reading)
	}

	return readings, nil
}

// extractSensorName extracts a meaningful sensor name from a path like
// /sys/class/hwmon/hwmon0/temp1_input -> hwmon0_temp1
func extractSensorName(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) >= 2 {
		hwmon := parts[len(parts)-2]
		sensor := parts[len(parts)-1]
		sensor = strings.TrimSuffix(sensor, "_input")
		return hwmon + "_" + sensor
	}
	return path
}
