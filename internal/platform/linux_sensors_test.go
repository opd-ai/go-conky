//go:build linux && !android
// +build linux,!android

package platform

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLinuxSensorProvider_Temperatures(t *testing.T) {
	// Create a temporary /sys/class/hwmon directory
	tmpDir := t.TempDir()
	hwmonPath := filepath.Join(tmpDir, "hwmon")
	if err := os.MkdirAll(hwmonPath, 0o755); err != nil {
		t.Fatalf("Failed to create hwmon directory: %v", err)
	}

	// Create hwmon0 device
	hwmon0Path := filepath.Join(hwmonPath, "hwmon0")
	if err := os.MkdirAll(hwmon0Path, 0o755); err != nil {
		t.Fatalf("Failed to create hwmon0 directory: %v", err)
	}

	// Create device name file
	if err := os.WriteFile(filepath.Join(hwmon0Path, "name"), []byte("coretemp\n"), 0o644); err != nil {
		t.Fatalf("Failed to write name file: %v", err)
	}

	// Create temperature sensor files
	if err := os.WriteFile(filepath.Join(hwmon0Path, "temp1_input"), []byte("45000\n"), 0o644); err != nil {
		t.Fatalf("Failed to write temp1_input: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hwmon0Path, "temp1_label"), []byte("Core 0\n"), 0o644); err != nil {
		t.Fatalf("Failed to write temp1_label: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hwmon0Path, "temp1_crit"), []byte("100000\n"), 0o644); err != nil {
		t.Fatalf("Failed to write temp1_crit: %v", err)
	}

	if err := os.WriteFile(filepath.Join(hwmon0Path, "temp2_input"), []byte("52000\n"), 0o644); err != nil {
		t.Fatalf("Failed to write temp2_input: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hwmon0Path, "temp2_label"), []byte("Core 1\n"), 0o644); err != nil {
		t.Fatalf("Failed to write temp2_label: %v", err)
	}

	provider := &linuxSensorProvider{
		hwmonPath: hwmonPath,
	}

	temps, err := provider.Temperatures()
	if err != nil {
		t.Fatalf("Temperatures() failed: %v", err)
	}

	if len(temps) != 2 {
		t.Fatalf("Temperatures() returned %d readings, want 2", len(temps))
	}

	// Check first temperature sensor
	found := false
	for _, temp := range temps {
		if temp.Label != "Core 0" {
			continue
		}
		found = true
		if temp.Name != "coretemp" {
			t.Errorf("Name = %s, want coretemp", temp.Name)
		}
		if temp.Value != 45.0 {
			t.Errorf("Value = %v, want 45.0", temp.Value)
		}
		if temp.Unit != "°C" {
			t.Errorf("Unit = %s, want °C", temp.Unit)
		}
		if temp.Critical != 100.0 {
			t.Errorf("Critical = %v, want 100.0", temp.Critical)
		}
	}
	if !found {
		t.Error("Core 0 temperature sensor not found")
	}

	// Check second temperature sensor
	found = false
	for _, temp := range temps {
		if temp.Label == "Core 1" {
			found = true
			if temp.Value != 52.0 {
				t.Errorf("Value = %v, want 52.0", temp.Value)
			}
			if temp.Critical != 0 {
				t.Errorf("Critical = %v, want 0 (no crit file)", temp.Critical)
			}
		}
	}
	if !found {
		t.Error("Core 1 temperature sensor not found")
	}
}

func TestLinuxSensorProvider_Fans(t *testing.T) {
	// Create a temporary /sys/class/hwmon directory
	tmpDir := t.TempDir()
	hwmonPath := filepath.Join(tmpDir, "hwmon")
	if err := os.MkdirAll(hwmonPath, 0o755); err != nil {
		t.Fatalf("Failed to create hwmon directory: %v", err)
	}

	// Create hwmon1 device
	hwmon1Path := filepath.Join(hwmonPath, "hwmon1")
	if err := os.MkdirAll(hwmon1Path, 0o755); err != nil {
		t.Fatalf("Failed to create hwmon1 directory: %v", err)
	}

	// Create device name file
	if err := os.WriteFile(filepath.Join(hwmon1Path, "name"), []byte("it8792\n"), 0o644); err != nil {
		t.Fatalf("Failed to write name file: %v", err)
	}

	// Create fan sensor files
	if err := os.WriteFile(filepath.Join(hwmon1Path, "fan1_input"), []byte("1200\n"), 0o644); err != nil {
		t.Fatalf("Failed to write fan1_input: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hwmon1Path, "fan1_label"), []byte("CPU Fan\n"), 0o644); err != nil {
		t.Fatalf("Failed to write fan1_label: %v", err)
	}

	if err := os.WriteFile(filepath.Join(hwmon1Path, "fan2_input"), []byte("800\n"), 0o644); err != nil {
		t.Fatalf("Failed to write fan2_input: %v", err)
	}

	provider := &linuxSensorProvider{
		hwmonPath: hwmonPath,
	}

	fans, err := provider.Fans()
	if err != nil {
		t.Fatalf("Fans() failed: %v", err)
	}

	if len(fans) != 2 {
		t.Fatalf("Fans() returned %d readings, want 2", len(fans))
	}

	// Check first fan sensor
	found := false
	for _, fan := range fans {
		if fan.Label == "CPU Fan" {
			found = true
			if fan.Name != "it8792" {
				t.Errorf("Name = %s, want it8792", fan.Name)
			}
			if fan.Value != 1200 {
				t.Errorf("Value = %v, want 1200", fan.Value)
			}
			if fan.Unit != "RPM" {
				t.Errorf("Unit = %s, want RPM", fan.Unit)
			}
		}
	}
	if !found {
		t.Error("CPU Fan sensor not found")
	}

	// Check second fan sensor (without label)
	found = false
	for _, fan := range fans {
		if fan.Label == "fan2" { // Should use sensor type as label when no label file exists
			found = true
			if fan.Value != 800 {
				t.Errorf("Value = %v, want 800", fan.Value)
			}
		}
	}
	if !found {
		t.Error("fan2 sensor not found")
	}
}

func TestLinuxSensorProvider_NoHwmon(t *testing.T) {
	// Test when /sys/class/hwmon doesn't exist
	tmpDir := t.TempDir()
	nonExistentPath := filepath.Join(tmpDir, "nonexistent")

	provider := &linuxSensorProvider{
		hwmonPath: nonExistentPath,
	}

	temps, err := provider.Temperatures()
	if err != nil {
		t.Fatalf("Temperatures() failed: %v", err)
	}
	if len(temps) != 0 {
		t.Errorf("Temperatures() returned %d readings, want 0 when hwmon doesn't exist", len(temps))
	}

	fans, err := provider.Fans()
	if err != nil {
		t.Fatalf("Fans() failed: %v", err)
	}
	if len(fans) != 0 {
		t.Errorf("Fans() returned %d readings, want 0 when hwmon doesn't exist", len(fans))
	}
}

func TestLinuxSensorProvider_MultipleDevices(t *testing.T) {
	// Create a temporary /sys/class/hwmon directory
	tmpDir := t.TempDir()
	hwmonPath := filepath.Join(tmpDir, "hwmon")
	if err := os.MkdirAll(hwmonPath, 0o755); err != nil {
		t.Fatalf("Failed to create hwmon directory: %v", err)
	}

	// Create two hwmon devices
	for i := 0; i < 2; i++ {
		devicePath := filepath.Join(hwmonPath, "hwmon"+string(rune('0'+i)))
		if err := os.MkdirAll(devicePath, 0o755); err != nil {
			t.Fatalf("Failed to create device directory: %v", err)
		}

		// Create device name file
		deviceName := "device" + string(rune('0'+i)) + "\n"
		if err := os.WriteFile(filepath.Join(devicePath, "name"), []byte(deviceName), 0o644); err != nil {
			t.Fatalf("Failed to write name file: %v", err)
		}

		// Create temperature sensor
		tempValue := string(rune('4'+i)) + "0000\n"
		if err := os.WriteFile(filepath.Join(devicePath, "temp1_input"), []byte(tempValue), 0o644); err != nil {
			t.Fatalf("Failed to write temp1_input: %v", err)
		}
	}

	provider := &linuxSensorProvider{
		hwmonPath: hwmonPath,
	}

	temps, err := provider.Temperatures()
	if err != nil {
		t.Fatalf("Temperatures() failed: %v", err)
	}

	// Should get sensors from both devices
	if len(temps) < 2 {
		t.Errorf("Temperatures() returned %d readings, want at least 2 from multiple devices", len(temps))
	}
}
