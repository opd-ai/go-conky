//go:build android
// +build android

package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// androidSensorProvider implements SensorProvider for Android systems.
// Android uses hwmon interface similar to Linux, but sensor access may be
// more restricted depending on the device and Android version.
type androidSensorProvider struct {
	hwmonPath     string
	thermalPath   string
	batteryPath   string
}

func newAndroidSensorProvider() *androidSensorProvider {
	return &androidSensorProvider{
		hwmonPath:   "/sys/class/hwmon",
		thermalPath: "/sys/class/thermal",
		batteryPath: "/sys/class/power_supply",
	}
}

func (s *androidSensorProvider) Temperatures() ([]SensorReading, error) {
	var readings []SensorReading

	// Try hwmon first (same as Linux)
	hwmonReadings, _ := s.readHwmonTemperatures()
	readings = append(readings, hwmonReadings...)

	// Android often exposes thermal zones which are more accessible
	thermalReadings, _ := s.readThermalZones()
	readings = append(readings, thermalReadings...)

	// Battery temperature is usually available on Android
	batteryTemp, err := s.readBatteryTemperature()
	if err == nil && batteryTemp.Value != 0 {
		readings = append(readings, batteryTemp)
	}

	return readings, nil
}

func (s *androidSensorProvider) Fans() ([]SensorReading, error) {
	// Android devices typically don't have fan sensors
	// Try hwmon anyway in case of non-standard devices
	return s.readHwmonFans()
}

// readHwmonTemperatures reads temperature sensors from hwmon interface.
func (s *androidSensorProvider) readHwmonTemperatures() ([]SensorReading, error) {
	var readings []SensorReading

	if _, err := os.Stat(s.hwmonPath); os.IsNotExist(err) {
		return readings, nil
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
			continue
		}

		readings = append(readings, deviceReadings...)
	}

	return readings, nil
}

// readThermalZones reads temperature from Android thermal zone interface.
func (s *androidSensorProvider) readThermalZones() ([]SensorReading, error) {
	var readings []SensorReading

	if _, err := os.Stat(s.thermalPath); os.IsNotExist(err) {
		return readings, nil
	}

	entries, err := os.ReadDir(s.thermalPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", s.thermalPath, err)
	}

	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "thermal_zone") {
			continue
		}

		zonePath := filepath.Join(s.thermalPath, entry.Name())

		// Read zone type (name)
		zoneType, _ := readStringFile(filepath.Join(zonePath, "type"))
		if zoneType == "" {
			zoneType = entry.Name()
		}

		// Read temperature (in millidegrees Celsius)
		tempMilliC, ok := readInt64File(filepath.Join(zonePath, "temp"))
		if !ok {
			continue
		}

		reading := SensorReading{
			Name:  "thermal_zone",
			Label: zoneType,
			Value: float64(tempMilliC) / 1000.0,
			Unit:  "°C",
		}

		// Try to read trip points for critical threshold
		for i := 0; i < 10; i++ {
			tripPath := filepath.Join(zonePath, fmt.Sprintf("trip_point_%d_type", i))
			tripType, ok := readStringFile(tripPath)
			if !ok {
				break
			}
			if tripType == "critical" {
				critPath := filepath.Join(zonePath, fmt.Sprintf("trip_point_%d_temp", i))
				critTemp, ok := readInt64File(critPath)
				if ok {
					reading.Critical = float64(critTemp) / 1000.0
				}
				break
			}
		}

		readings = append(readings, reading)
	}

	return readings, nil
}

// readBatteryTemperature reads temperature from battery power supply.
func (s *androidSensorProvider) readBatteryTemperature() (SensorReading, error) {
	reading := SensorReading{
		Name:  "battery",
		Label: "Battery",
		Unit:  "°C",
	}

	entries, err := os.ReadDir(s.batteryPath)
	if err != nil {
		return reading, fmt.Errorf("reading %s: %w", s.batteryPath, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Check if this is a battery
		typePath := filepath.Join(s.batteryPath, entry.Name(), "type")
		typeData, err := os.ReadFile(typePath)
		if err != nil {
			continue
		}

		if strings.TrimSpace(string(typeData)) != "Battery" {
			continue
		}

		// Read temperature (in tenths of degrees Celsius on Android)
		tempPath := filepath.Join(s.batteryPath, entry.Name(), "temp")
		tempTenths, ok := readInt64File(tempPath)
		if ok {
			reading.Value = float64(tempTenths) / 10.0
			return reading, nil
		}
	}

	return reading, fmt.Errorf("battery temperature not found")
}

// readTemperatureSensors reads all temperature sensors from a hwmon device.
func (s *androidSensorProvider) readTemperatureSensors(devicePath string) ([]SensorReading, error) {
	deviceName := s.readDeviceName(devicePath)

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

		sensorType := strings.TrimSuffix(name, "_input")

		inputPath := filepath.Join(devicePath, name)
		tempMilliC, ok := readInt64File(inputPath)
		if !ok {
			continue
		}

		label := s.readSensorLabel(devicePath, sensorType)
		if label == "" {
			label = sensorType
		}

		critPath := filepath.Join(devicePath, sensorType+"_crit")
		critMilliC, hasCrit := readInt64File(critPath)

		reading := SensorReading{
			Name:  deviceName,
			Label: label,
			Value: float64(tempMilliC) / 1000.0,
			Unit:  "°C",
		}

		if hasCrit {
			reading.Critical = float64(critMilliC) / 1000.0
		}

		readings = append(readings, reading)
	}

	return readings, nil
}

// readHwmonFans reads fan sensors from hwmon interface.
func (s *androidSensorProvider) readHwmonFans() ([]SensorReading, error) {
	var readings []SensorReading

	if _, err := os.Stat(s.hwmonPath); os.IsNotExist(err) {
		return readings, nil
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
			continue
		}

		readings = append(readings, deviceReadings...)
	}

	return readings, nil
}

// readFanSensors reads all fan sensors from a hwmon device.
func (s *androidSensorProvider) readFanSensors(devicePath string) ([]SensorReading, error) {
	deviceName := s.readDeviceName(devicePath)

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

		sensorType := strings.TrimSuffix(name, "_input")

		inputPath := filepath.Join(devicePath, name)
		fanRPM, ok := readInt64File(inputPath)
		if !ok {
			continue
		}

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
func (s *androidSensorProvider) readDeviceName(devicePath string) string {
	nameBytes, err := os.ReadFile(filepath.Join(devicePath, "name"))
	if err != nil {
		return filepath.Base(devicePath)
	}
	return strings.TrimSpace(string(nameBytes))
}

// readSensorLabel reads the sensor label from the '{type}_label' file.
func (s *androidSensorProvider) readSensorLabel(devicePath, sensorType string) string {
	labelPath := filepath.Join(devicePath, sensorType+"_label")
	labelBytes, err := os.ReadFile(labelPath)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(labelBytes))
}
