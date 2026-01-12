package monitor

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// TempSensor represents a single temperature sensor reading.
type TempSensor struct {
	// Label is the sensor label (e.g., "Core 0", "Package id 0").
	Label string
	// Input is the current temperature in millidegrees Celsius.
	Input int64
	// InputCelsius is the current temperature in degrees Celsius.
	InputCelsius float64
	// Max is the maximum temperature threshold in millidegrees Celsius.
	Max int64
	// MaxCelsius is the maximum temperature threshold in degrees Celsius.
	MaxCelsius float64
	// Crit is the critical temperature threshold in millidegrees Celsius.
	Crit int64
	// CritCelsius is the critical temperature threshold in degrees Celsius.
	CritCelsius float64
	// Type is the sensor type identifier (e.g., "temp1", "temp2").
	Type string
}

// HwmonDevice represents a single hwmon device with its sensors.
type HwmonDevice struct {
	// Name is the device name (e.g., "coretemp", "acpitz").
	Name string
	// Path is the sysfs path to the hwmon device.
	Path string
	// Temps contains temperature sensor readings keyed by sensor type.
	Temps map[string]TempSensor
}

// HwmonStats contains hardware monitoring statistics.
type HwmonStats struct {
	// Devices contains hwmon device information keyed by device name.
	Devices map[string]HwmonDevice
	// TempSensors is a flat list of all temperature sensors for convenience.
	TempSensors []TempSensor
}

// hwmonReader reads hardware monitoring data from /sys/class/hwmon.
type hwmonReader struct {
	mu        sync.Mutex
	hwmonPath string
}

// newHwmonReader creates a new hwmonReader with default paths.
func newHwmonReader() *hwmonReader {
	return &hwmonReader{
		hwmonPath: "/sys/class/hwmon",
	}
}

// ReadStats reads current hardware monitoring statistics.
func (r *hwmonReader) ReadStats() (HwmonStats, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	stats := HwmonStats{
		Devices:     make(map[string]HwmonDevice),
		TempSensors: make([]TempSensor, 0),
	}

	// Check if hwmon directory exists
	if _, err := os.Stat(r.hwmonPath); os.IsNotExist(err) {
		return stats, nil // No hwmon support, return empty stats
	}

	entries, err := os.ReadDir(r.hwmonPath)
	if err != nil {
		return stats, fmt.Errorf("reading %s: %w", r.hwmonPath, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "hwmon") {
			continue
		}

		devicePath := filepath.Join(r.hwmonPath, entry.Name())
		device, err := r.readDevice(devicePath)
		if err != nil {
			// Skip devices that fail to read
			continue
		}

		stats.Devices[device.Name] = device
		for _, temp := range device.Temps {
			stats.TempSensors = append(stats.TempSensors, temp)
		}
	}

	return stats, nil
}

// readDevice reads information for a single hwmon device.
func (r *hwmonReader) readDevice(devicePath string) (HwmonDevice, error) {
	device := HwmonDevice{
		Path:  devicePath,
		Temps: make(map[string]TempSensor),
	}

	// Read device name
	nameBytes, err := os.ReadFile(filepath.Join(devicePath, "name"))
	if err != nil {
		// Try device symlink for older kernels
		linkPath, linkErr := os.Readlink(filepath.Join(devicePath, "device"))
		if linkErr != nil {
			device.Name = filepath.Base(devicePath)
		} else {
			device.Name = filepath.Base(linkPath)
		}
	} else {
		device.Name = strings.TrimSpace(string(nameBytes))
	}

	// Find all temperature sensors (temp1_input, temp2_input, etc.)
	entries, err := os.ReadDir(devicePath)
	if err != nil {
		return device, fmt.Errorf("reading device directory: %w", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, "temp") || !strings.HasSuffix(name, "_input") {
			continue
		}

		// Extract sensor type (e.g., "temp1" from "temp1_input")
		sensorType := strings.TrimSuffix(name, "_input")
		sensor, err := r.readTempSensor(devicePath, sensorType)
		if err != nil {
			continue
		}

		device.Temps[sensorType] = sensor
	}

	return device, nil
}

// readTempSensor reads a single temperature sensor.
func (r *hwmonReader) readTempSensor(devicePath, sensorType string) (TempSensor, error) {
	sensor := TempSensor{
		Type: sensorType,
	}

	// Read temperature input (required)
	inputPath := filepath.Join(devicePath, sensorType+"_input")
	inputBytes, err := os.ReadFile(inputPath)
	if err != nil {
		return sensor, fmt.Errorf("reading input: %w", err)
	}
	input, err := strconv.ParseInt(strings.TrimSpace(string(inputBytes)), 10, 64)
	if err != nil {
		return sensor, fmt.Errorf("parsing input: %w", err)
	}
	sensor.Input = input
	sensor.InputCelsius = float64(input) / 1000.0

	// Read label (optional)
	labelPath := filepath.Join(devicePath, sensorType+"_label")
	if labelBytes, err := os.ReadFile(labelPath); err == nil {
		sensor.Label = strings.TrimSpace(string(labelBytes))
	} else {
		sensor.Label = sensorType
	}

	// Read max temperature (optional)
	maxPath := filepath.Join(devicePath, sensorType+"_max")
	if maxBytes, err := os.ReadFile(maxPath); err == nil {
		if maxVal, err := strconv.ParseInt(strings.TrimSpace(string(maxBytes)), 10, 64); err == nil {
			sensor.Max = maxVal
			sensor.MaxCelsius = float64(maxVal) / 1000.0
		}
	}

	// Read critical temperature (optional)
	critPath := filepath.Join(devicePath, sensorType+"_crit")
	if critBytes, err := os.ReadFile(critPath); err == nil {
		if critVal, err := strconv.ParseInt(strings.TrimSpace(string(critBytes)), 10, 64); err == nil {
			sensor.Crit = critVal
			sensor.CritCelsius = float64(critVal) / 1000.0
		}
	}

	return sensor, nil
}
