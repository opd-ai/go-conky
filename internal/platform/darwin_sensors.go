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
)

// darwinSensorProvider implements SensorProvider for macOS/Darwin systems.
// Note: macOS doesn't provide easy access to hardware sensors without SMC (System Management Controller).
// This implementation uses powermetrics for available temperature data.
type darwinSensorProvider struct{}

func newDarwinSensorProvider() *darwinSensorProvider {
	return &darwinSensorProvider{}
}

// Temperatures returns all temperature sensor readings.
// On macOS, we try to use powermetrics which requires root privileges.
func (s *darwinSensorProvider) Temperatures() ([]SensorReading, error) {
	// Try using powermetrics to get CPU temperature
	// Note: This requires root/sudo privileges
	cmd := exec.Command("powermetrics", "--samplers", "smc", "-i", "1", "-n", "1")
	output, err := cmd.Output()
	if err != nil {
		// powermetrics failed (likely no permissions or not available)
		// Return empty list rather than error
		return []SensorReading{}, nil
	}

	return s.parsePowermetricsOutput(output)
}

// Fans returns all fan speed sensor readings.
func (s *darwinSensorProvider) Fans() ([]SensorReading, error) {
	// Fan speed information is also available through powermetrics
	cmd := exec.Command("powermetrics", "--samplers", "smc", "-i", "1", "-n", "1")
	output, err := cmd.Output()
	if err != nil {
		// powermetrics failed (likely no permissions or not available)
		// Return empty list rather than error
		return []SensorReading{}, nil
	}

	return s.parseFanOutput(output)
}

// parsePowermetricsOutput extracts temperature readings from powermetrics output.
// Example output includes lines like:
// CPU die temperature: 45.00 C
func (s *darwinSensorProvider) parsePowermetricsOutput(output []byte) ([]SensorReading, error) {
	var readings []SensorReading

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		// Look for temperature lines
		if strings.Contains(strings.ToLower(line), "temperature") && strings.Contains(line, "C") {
			reading := s.parseTemperatureLine(line)
			if reading != nil {
				readings = append(readings, *reading)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("parsing powermetrics output: %w", err)
	}

	return readings, nil
}

// parseTemperatureLine parses a temperature line from powermetrics.
// Example: "CPU die temperature: 45.00 C"
func (s *darwinSensorProvider) parseTemperatureLine(line string) *SensorReading {
	// Split by colon
	parts := strings.Split(line, ":")
	if len(parts) != 2 {
		return nil
	}

	name := strings.TrimSpace(parts[0])
	valuePart := strings.TrimSpace(parts[1])

	// Extract temperature value
	// Format: "45.00 C" or "45 C"
	fields := strings.Fields(valuePart)
	if len(fields) < 2 {
		return nil
	}

	tempStr := fields[0]
	temp, err := strconv.ParseFloat(tempStr, 64)
	if err != nil {
		return nil
	}

	return &SensorReading{
		Name:  name,
		Label: name,
		Value: temp,
		Unit:  "°C",
	}
}

// parseFanOutput extracts fan speed readings from powermetrics output.
// Example output includes lines like:
// Fan: 2000 rpm
func (s *darwinSensorProvider) parseFanOutput(output []byte) ([]SensorReading, error) {
	var readings []SensorReading

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		// Look for fan lines
		if strings.Contains(strings.ToLower(line), "fan") && strings.Contains(strings.ToLower(line), "rpm") {
			reading := s.parseFanLine(line)
			if reading != nil {
				readings = append(readings, *reading)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("parsing powermetrics fan output: %w", err)
	}

	return readings, nil
}

// parseFanLine parses a fan speed line from powermetrics.
// Example: "Fan: 2000 rpm"
func (s *darwinSensorProvider) parseFanLine(line string) *SensorReading {
	// Split by colon
	parts := strings.Split(line, ":")
	if len(parts) != 2 {
		return nil
	}

	name := strings.TrimSpace(parts[0])
	valuePart := strings.TrimSpace(parts[1])

	// Extract fan speed value
	// Format: "2000 rpm"
	fields := strings.Fields(valuePart)
	if len(fields) < 2 {
		return nil
	}

	speedStr := fields[0]
	speed, err := strconv.ParseFloat(speedStr, 64)
	if err != nil {
		return nil
	}

	return &SensorReading{
		Name:  name,
		Label: name,
		Value: speed,
		Unit:  "RPM",
	}
}
