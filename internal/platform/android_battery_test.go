//go:build android
// +build android

package platform

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAndroidBatteryProvider_Count(t *testing.T) {
	tmpDir := t.TempDir()
	powerSupplyPath := tmpDir

	// Create a battery directory
	batteryDir := filepath.Join(powerSupplyPath, "BAT0")
	if err := os.MkdirAll(batteryDir, 0o755); err != nil {
		t.Fatalf("Failed to create battery directory: %v", err)
	}

	// Create type file
	if err := os.WriteFile(filepath.Join(batteryDir, "type"), []byte("Battery\n"), 0o644); err != nil {
		t.Fatalf("Failed to write type file: %v", err)
	}

	// Create a non-battery directory (AC adapter)
	acDir := filepath.Join(powerSupplyPath, "AC0")
	if err := os.MkdirAll(acDir, 0o755); err != nil {
		t.Fatalf("Failed to create AC directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(acDir, "type"), []byte("Mains\n"), 0o644); err != nil {
		t.Fatalf("Failed to write AC type file: %v", err)
	}

	provider := &androidBatteryProvider{
		powerSupplyPath: powerSupplyPath,
	}

	count := provider.Count()
	if count != 1 {
		t.Errorf("Count() = %v, want 1", count)
	}
}

func TestAndroidBatteryProvider_Stats(t *testing.T) {
	tmpDir := t.TempDir()
	powerSupplyPath := tmpDir

	// Create a battery directory
	batteryDir := filepath.Join(powerSupplyPath, "BAT0")
	if err := os.MkdirAll(batteryDir, 0o755); err != nil {
		t.Fatalf("Failed to create battery directory: %v", err)
	}

	// Create required files
	files := map[string]string{
		"type":        "Battery\n",
		"capacity":    "75\n",
		"status":      "Discharging\n",
		"voltage_now": "4200000\n", // 4.2V in µV
		"energy_now":  "50000000\n",
		"energy_full": "100000000\n",
		"power_now":   "10000000\n",
	}

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(batteryDir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to write %s file: %v", name, err)
		}
	}

	provider := &androidBatteryProvider{
		powerSupplyPath: powerSupplyPath,
	}

	stats, err := provider.Stats(0)
	if err != nil {
		t.Fatalf("Stats() failed: %v", err)
	}

	if stats.Percent != 75 {
		t.Errorf("Percent = %v, want 75", stats.Percent)
	}

	if stats.Charging {
		t.Error("Charging = true, want false")
	}

	expectedVoltage := 4.2
	if stats.Voltage < expectedVoltage-0.01 || stats.Voltage > expectedVoltage+0.01 {
		t.Errorf("Voltage = %v, want ~%v", stats.Voltage, expectedVoltage)
	}
}

func TestAndroidBatteryProvider_StatsCharging(t *testing.T) {
	tmpDir := t.TempDir()
	powerSupplyPath := tmpDir

	batteryDir := filepath.Join(powerSupplyPath, "BAT0")
	if err := os.MkdirAll(batteryDir, 0o755); err != nil {
		t.Fatalf("Failed to create battery directory: %v", err)
	}

	files := map[string]string{
		"type":     "Battery\n",
		"capacity": "50\n",
		"status":   "Charging\n",
	}

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(batteryDir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to write %s file: %v", name, err)
		}
	}

	provider := &androidBatteryProvider{
		powerSupplyPath: powerSupplyPath,
	}

	stats, err := provider.Stats(0)
	if err != nil {
		t.Fatalf("Stats() failed: %v", err)
	}

	if !stats.Charging {
		t.Error("Charging = false, want true")
	}
}

func TestAndroidBatteryProvider_StatsOutOfRange(t *testing.T) {
	tmpDir := t.TempDir()
	powerSupplyPath := tmpDir

	batteryDir := filepath.Join(powerSupplyPath, "BAT0")
	if err := os.MkdirAll(batteryDir, 0o755); err != nil {
		t.Fatalf("Failed to create battery directory: %v", err)
	}

	if err := os.WriteFile(filepath.Join(batteryDir, "type"), []byte("Battery\n"), 0o644); err != nil {
		t.Fatalf("Failed to write type file: %v", err)
	}

	provider := &androidBatteryProvider{
		powerSupplyPath: powerSupplyPath,
	}

	_, err := provider.Stats(1)
	if err == nil {
		t.Error("Stats(1) should have returned an error for out of range index")
	}

	_, err = provider.Stats(-1)
	if err == nil {
		t.Error("Stats(-1) should have returned an error for negative index")
	}
}
