//go:build android
// +build android

package platform

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAndroidSensorProvider_Temperatures(t *testing.T) {
	tmpDir := t.TempDir()

	// Create thermal zone structure
	thermalPath := filepath.Join(tmpDir, "thermal")
	zone0 := filepath.Join(thermalPath, "thermal_zone0")
	if err := os.MkdirAll(zone0, 0o755); err != nil {
		t.Fatalf("Failed to create thermal_zone0 directory: %v", err)
	}

	files := map[string]string{
		"type": "cpu-thermal\n",
		"temp": "45000\n", // 45°C in millidegrees
	}

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(zone0, name), []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to write %s file: %v", name, err)
		}
	}

	provider := &androidSensorProvider{
		hwmonPath:   filepath.Join(tmpDir, "hwmon"), // Non-existent
		thermalPath: thermalPath,
		batteryPath: filepath.Join(tmpDir, "power_supply"), // Non-existent
	}

	temps, err := provider.Temperatures()
	if err != nil {
		t.Fatalf("Temperatures() failed: %v", err)
	}

	if len(temps) != 1 {
		t.Fatalf("Temperatures() returned %d readings, want 1", len(temps))
	}

	if temps[0].Label != "cpu-thermal" {
		t.Errorf("Label = %v, want cpu-thermal", temps[0].Label)
	}

	if temps[0].Value != 45.0 {
		t.Errorf("Value = %v, want 45.0", temps[0].Value)
	}

	if temps[0].Unit != "°C" {
		t.Errorf("Unit = %v, want °C", temps[0].Unit)
	}
}

func TestAndroidSensorProvider_BatteryTemperature(t *testing.T) {
	tmpDir := t.TempDir()

	// Create battery power supply structure
	batteryPath := filepath.Join(tmpDir, "power_supply")
	bat0 := filepath.Join(batteryPath, "BAT0")
	if err := os.MkdirAll(bat0, 0o755); err != nil {
		t.Fatalf("Failed to create BAT0 directory: %v", err)
	}

	files := map[string]string{
		"type": "Battery\n",
		"temp": "320\n", // 32.0°C in tenths of degrees
	}

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(bat0, name), []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to write %s file: %v", name, err)
		}
	}

	provider := &androidSensorProvider{
		hwmonPath:   filepath.Join(tmpDir, "hwmon"),
		thermalPath: filepath.Join(tmpDir, "thermal"),
		batteryPath: batteryPath,
	}

	temps, err := provider.Temperatures()
	if err != nil {
		t.Fatalf("Temperatures() failed: %v", err)
	}

	if len(temps) != 1 {
		t.Fatalf("Temperatures() returned %d readings, want 1", len(temps))
	}

	if temps[0].Label != "Battery" {
		t.Errorf("Label = %v, want Battery", temps[0].Label)
	}

	if temps[0].Value != 32.0 {
		t.Errorf("Value = %v, want 32.0", temps[0].Value)
	}
}

func TestAndroidSensorProvider_HwmonTemperatures(t *testing.T) {
	tmpDir := t.TempDir()

	// Create hwmon structure
	hwmonPath := filepath.Join(tmpDir, "hwmon")
	hwmon0 := filepath.Join(hwmonPath, "hwmon0")
	if err := os.MkdirAll(hwmon0, 0o755); err != nil {
		t.Fatalf("Failed to create hwmon0 directory: %v", err)
	}

	files := map[string]string{
		"name":        "coretemp\n",
		"temp1_input": "50000\n", // 50°C in millidegrees
		"temp1_label": "Core 0\n",
		"temp1_crit":  "100000\n", // 100°C critical
	}

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(hwmon0, name), []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to write %s file: %v", name, err)
		}
	}

	provider := &androidSensorProvider{
		hwmonPath:   hwmonPath,
		thermalPath: filepath.Join(tmpDir, "thermal"),
		batteryPath: filepath.Join(tmpDir, "power_supply"),
	}

	temps, err := provider.Temperatures()
	if err != nil {
		t.Fatalf("Temperatures() failed: %v", err)
	}

	if len(temps) != 1 {
		t.Fatalf("Temperatures() returned %d readings, want 1", len(temps))
	}

	if temps[0].Name != "coretemp" {
		t.Errorf("Name = %v, want coretemp", temps[0].Name)
	}

	if temps[0].Label != "Core 0" {
		t.Errorf("Label = %v, want Core 0", temps[0].Label)
	}

	if temps[0].Value != 50.0 {
		t.Errorf("Value = %v, want 50.0", temps[0].Value)
	}

	if temps[0].Critical != 100.0 {
		t.Errorf("Critical = %v, want 100.0", temps[0].Critical)
	}
}

func TestAndroidSensorProvider_Fans(t *testing.T) {
	tmpDir := t.TempDir()

	// Create hwmon structure with fan sensor
	hwmonPath := filepath.Join(tmpDir, "hwmon")
	hwmon0 := filepath.Join(hwmonPath, "hwmon0")
	if err := os.MkdirAll(hwmon0, 0o755); err != nil {
		t.Fatalf("Failed to create hwmon0 directory: %v", err)
	}

	files := map[string]string{
		"name":       "thinkpad\n",
		"fan1_input": "3200\n", // 3200 RPM
		"fan1_label": "CPU Fan\n",
	}

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(hwmon0, name), []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to write %s file: %v", name, err)
		}
	}

	provider := &androidSensorProvider{
		hwmonPath:   hwmonPath,
		thermalPath: filepath.Join(tmpDir, "thermal"),
		batteryPath: filepath.Join(tmpDir, "power_supply"),
	}

	fans, err := provider.Fans()
	if err != nil {
		t.Fatalf("Fans() failed: %v", err)
	}

	if len(fans) != 1 {
		t.Fatalf("Fans() returned %d readings, want 1", len(fans))
	}

	if fans[0].Name != "thinkpad" {
		t.Errorf("Name = %v, want thinkpad", fans[0].Name)
	}

	if fans[0].Label != "CPU Fan" {
		t.Errorf("Label = %v, want CPU Fan", fans[0].Label)
	}

	if fans[0].Value != 3200 {
		t.Errorf("Value = %v, want 3200", fans[0].Value)
	}

	if fans[0].Unit != "RPM" {
		t.Errorf("Unit = %v, want RPM", fans[0].Unit)
	}
}

func TestAndroidSensorProvider_NoSensors(t *testing.T) {
	tmpDir := t.TempDir()

	provider := &androidSensorProvider{
		hwmonPath:   filepath.Join(tmpDir, "nonexistent_hwmon"),
		thermalPath: filepath.Join(tmpDir, "nonexistent_thermal"),
		batteryPath: filepath.Join(tmpDir, "nonexistent_power_supply"),
	}

	temps, err := provider.Temperatures()
	if err != nil {
		t.Fatalf("Temperatures() failed: %v", err)
	}

	// Should return empty slice, not error
	if len(temps) != 0 {
		t.Errorf("Temperatures() returned %d readings, want 0", len(temps))
	}

	fans, err := provider.Fans()
	if err != nil {
		t.Fatalf("Fans() failed: %v", err)
	}

	if len(fans) != 0 {
		t.Errorf("Fans() returned %d readings, want 0", len(fans))
	}
}

func TestAndroidSensorProvider_ThermalZoneWithCriticalTrip(t *testing.T) {
	tmpDir := t.TempDir()

	thermalPath := filepath.Join(tmpDir, "thermal")
	zone0 := filepath.Join(thermalPath, "thermal_zone0")
	if err := os.MkdirAll(zone0, 0o755); err != nil {
		t.Fatalf("Failed to create thermal_zone0 directory: %v", err)
	}

	files := map[string]string{
		"type":              "gpu-thermal\n",
		"temp":              "55000\n",
		"trip_point_0_type": "passive\n",
		"trip_point_0_temp": "85000\n",
		"trip_point_1_type": "critical\n",
		"trip_point_1_temp": "105000\n",
	}

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(zone0, name), []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to write %s file: %v", name, err)
		}
	}

	provider := &androidSensorProvider{
		hwmonPath:   filepath.Join(tmpDir, "hwmon"),
		thermalPath: thermalPath,
		batteryPath: filepath.Join(tmpDir, "power_supply"),
	}

	temps, err := provider.Temperatures()
	if err != nil {
		t.Fatalf("Temperatures() failed: %v", err)
	}

	if len(temps) != 1 {
		t.Fatalf("Temperatures() returned %d readings, want 1", len(temps))
	}

	if temps[0].Critical != 105.0 {
		t.Errorf("Critical = %v, want 105.0", temps[0].Critical)
	}
}
