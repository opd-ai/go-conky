//go:build linux && !android
// +build linux,!android

package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// linuxSensorProvider implements SensorProvider for Linux systems.
type linuxSensorProvider struct {
	hwmonPath string
}

func newLinuxSensorProvider() *linuxSensorProvider {
	return &linuxSensorProvider{
		hwmonPath: "/sys/class/hwmon",
	}
}

func (s *linuxSensorProvider) Temperatures() ([]SensorReading, error) {
	var readings []SensorReading

	// Check if hwmon directory exists
	if _, err := os.Stat(s.hwmonPath); os.IsNotExist(err) {
		return readings, nil // No hwmon support
	}

	entries, err := os.ReadDir(s.hwmonPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", s.hwmonPath, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "hwmon") {
			continue
		}

		devicePath := filepath.Join(s.hwmonPath, entry.Name())
		deviceReadings, err := s.readTemperatureSensors(devicePath)
		if err != nil {
			// Skip devices that fail to read
			continue
		}

		readings = append(readings, deviceReadings...)
	}

	return readings, nil
}

func (s *linuxSensorProvider) Fans() ([]SensorReading, error) {
	var readings []SensorReading

	// Check if hwmon directory exists
	if _, err := os.Stat(s.hwmonPath); os.IsNotExist(err) {
		return readings, nil // No hwmon support
	}

	entries, err := os.ReadDir(s.hwmonPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", s.hwmonPath, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "hwmon") {
			continue
		}

		devicePath := filepath.Join(s.hwmonPath, entry.Name())
		deviceReadings, err := s.readFanSensors(devicePath)
		if err != nil {
			// Skip devices that fail to read
			continue
		}

		readings = append(readings, deviceReadings...)
	}

	return readings, nil
}

// readTemperatureSensors reads all temperature sensors from a hwmon device.
func (s *linuxSensorProvider) readTemperatureSensors(devicePath string) ([]SensorReading, error) {
	// Read device name
	deviceName := s.readDeviceName(devicePath)

	// Find all temperature sensors (temp1_input, temp2_input, etc.)
	entries, err := os.ReadDir(devicePath)
	if err != nil {
		return nil, fmt.Errorf("reading device directory: %w", err)
	}

	readings := make([]SensorReading, 0, len(entries)/4)

	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, "temp") || !strings.HasSuffix(name, "_input") {
			continue
		}

		// Extract sensor type (e.g., "temp1" from "temp1_input")
		sensorType := strings.TrimSuffix(name, "_input")

		// Read temperature value (in millidegrees Celsius)
		inputPath := filepath.Join(devicePath, name)
		tempMilliC, ok := readInt64File(inputPath)
		if !ok {
			continue
		}

		// Read label
		label := s.readSensorLabel(devicePath, sensorType)
		if label == "" {
			label = sensorType
		}

		// Read critical threshold
		critPath := filepath.Join(devicePath, sensorType+"_crit")
		critMilliC, hasCrit := readInt64File(critPath)

		reading := SensorReading{
			Name:  deviceName,
			Label: label,
			Value: float64(tempMilliC) / 1000.0, // Convert to degrees Celsius
			Unit:  "°C",
		}

		if hasCrit {
			reading.Critical = float64(critMilliC) / 1000.0
		}

		readings = append(readings, reading)
	}

	return readings, nil
}

// readFanSensors reads all fan sensors from a hwmon device.
func (s *linuxSensorProvider) readFanSensors(devicePath string) ([]SensorReading, error) {
	// Read device name
	deviceName := s.readDeviceName(devicePath)

	// Find all fan sensors (fan1_input, fan2_input, etc.)
	entries, err := os.ReadDir(devicePath)
	if err != nil {
		return nil, fmt.Errorf("reading device directory: %w", err)
	}

	readings := make([]SensorReading, 0, len(entries)/4)

	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, "fan") || !strings.HasSuffix(name, "_input") {
			continue
		}

		// Extract sensor type (e.g., "fan1" from "fan1_input")
		sensorType := strings.TrimSuffix(name, "_input")

		// Read fan speed value (in RPM)
		inputPath := filepath.Join(devicePath, name)
		fanRPM, ok := readInt64File(inputPath)
		if !ok {
			continue
		}

		// Read label
		label := s.readSensorLabel(devicePath, sensorType)
		if label == "" {
			label = sensorType
		}

		reading := SensorReading{
			Name:  deviceName,
			Label: label,
			Value: float64(fanRPM),
			Unit:  "RPM",
		}

		readings = append(readings, reading)
	}

	return readings, nil
}

// readDeviceName reads the device name from the 'name' file.
func (s *linuxSensorProvider) readDeviceName(devicePath string) string {
	nameBytes, err := os.ReadFile(filepath.Join(devicePath, "name"))
	if err != nil {
		// Fallback: use the hwmon directory name
		return filepath.Base(devicePath)
	}
	return strings.TrimSpace(string(nameBytes))
}

// readSensorLabel reads the sensor label from the '{type}_label' file.
func (s *linuxSensorProvider) readSensorLabel(devicePath, sensorType string) string {
	labelPath := filepath.Join(devicePath, sensorType+"_label")
	labelBytes, err := os.ReadFile(labelPath)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(labelBytes))
}
