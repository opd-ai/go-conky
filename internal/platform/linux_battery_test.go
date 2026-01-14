package platform

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLinuxBatteryProvider_Count(t *testing.T) {
	// Create a temporary /sys/class/power_supply directory
	tmpDir := t.TempDir()
	powerSupplyPath := filepath.Join(tmpDir, "power_supply")
	if err := os.MkdirAll(powerSupplyPath, 0755); err != nil {
		t.Fatalf("Failed to create power_supply directory: %v", err)
	}

	// Create battery directories
	bat0Path := filepath.Join(powerSupplyPath, "BAT0")
	bat1Path := filepath.Join(powerSupplyPath, "BAT1")
	acPath := filepath.Join(powerSupplyPath, "AC")

	for _, path := range []string{bat0Path, bat1Path, acPath} {
		if err := os.MkdirAll(path, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", path, err)
		}
	}

	// Create type files
	if err := os.WriteFile(filepath.Join(bat0Path, "type"), []byte("Battery\n"), 0644); err != nil {
		t.Fatalf("Failed to write type file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bat1Path, "type"), []byte("Battery\n"), 0644); err != nil {
		t.Fatalf("Failed to write type file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(acPath, "type"), []byte("Mains\n"), 0644); err != nil {
		t.Fatalf("Failed to write type file: %v", err)
	}

	provider := &linuxBatteryProvider{
		powerSupplyPath: powerSupplyPath,
	}

	count := provider.Count()
	if count != 2 {
		t.Errorf("Count() = %d, want 2 (should only count Battery types)", count)
	}
}

func TestLinuxBatteryProvider_Stats_EnergyBased(t *testing.T) {
	// Create a temporary battery directory
	tmpDir := t.TempDir()
	powerSupplyPath := filepath.Join(tmpDir, "power_supply")
	bat0Path := filepath.Join(powerSupplyPath, "BAT0")
	if err := os.MkdirAll(bat0Path, 0755); err != nil {
		t.Fatalf("Failed to create battery directory: %v", err)
	}

	// Create type file
	if err := os.WriteFile(filepath.Join(bat0Path, "type"), []byte("Battery\n"), 0644); err != nil {
		t.Fatalf("Failed to write type file: %v", err)
	}

	// Create battery stat files (energy-based)
	files := map[string]string{
		"capacity":    "75",
		"status":      "Discharging",
		"energy_now":  "30000000", // 30 Wh in µWh
		"energy_full": "40000000", // 40 Wh in µWh
		"voltage_now": "12000000", // 12V in µV
		"power_now":   "10000000", // 10W in µW
	}

	for filename, content := range files {
		path := filepath.Join(bat0Path, filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write %s: %v", filename, err)
		}
	}

	provider := &linuxBatteryProvider{
		powerSupplyPath: powerSupplyPath,
	}

	stats, err := provider.Stats(0)
	if err != nil {
		t.Fatalf("Stats(0) failed: %v", err)
	}

	if stats.Percent != 75 {
		t.Errorf("Percent = %v, want 75", stats.Percent)
	}
	if stats.Charging {
		t.Error("Charging should be false for Discharging status")
	}
	if stats.Current != 30000000 {
		t.Errorf("Current = %v, want 30000000", stats.Current)
	}
	if stats.FullCapacity != 40000000 {
		t.Errorf("FullCapacity = %v, want 40000000", stats.FullCapacity)
	}
	if stats.Voltage < 11.9 || stats.Voltage > 12.1 {
		t.Errorf("Voltage = %v, want ~12.0", stats.Voltage)
	}

	// Check time remaining (should be approximately 3 hours = 10800 seconds)
	// energy_now = 30 Wh, power_now = 10 W, so 30/10 = 3 hours
	expectedTime := 3 * time.Hour
	if stats.TimeRemaining < expectedTime-time.Minute || stats.TimeRemaining > expectedTime+time.Minute {
		t.Errorf("TimeRemaining = %v, want ~%v", stats.TimeRemaining, expectedTime)
	}
}

func TestLinuxBatteryProvider_Stats_ChargeBased(t *testing.T) {
	// Create a temporary battery directory
	tmpDir := t.TempDir()
	powerSupplyPath := filepath.Join(tmpDir, "power_supply")
	bat0Path := filepath.Join(powerSupplyPath, "BAT0")
	if err := os.MkdirAll(bat0Path, 0755); err != nil {
		t.Fatalf("Failed to create battery directory: %v", err)
	}

	// Create type file
	if err := os.WriteFile(filepath.Join(bat0Path, "type"), []byte("Battery\n"), 0644); err != nil {
		t.Fatalf("Failed to write type file: %v", err)
	}

	// Create battery stat files (charge-based, no energy_* files)
	files := map[string]string{
		"capacity":    "50",
		"status":      "Charging",
		"charge_now":  "2500000",  // 2.5 Ah in µAh
		"charge_full": "5000000",  // 5 Ah in µAh
		"voltage_now": "12000000", // 12V in µV
	}

	for filename, content := range files {
		path := filepath.Join(bat0Path, filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write %s: %v", filename, err)
		}
	}

	provider := &linuxBatteryProvider{
		powerSupplyPath: powerSupplyPath,
	}

	stats, err := provider.Stats(0)
	if err != nil {
		t.Fatalf("Stats(0) failed: %v", err)
	}

	if stats.Percent != 50 {
		t.Errorf("Percent = %v, want 50", stats.Percent)
	}
	if !stats.Charging {
		t.Error("Charging should be true for Charging status")
	}

	// Charge-based battery: energy = charge * voltage
	// 2.5 Ah * 12 V = 30 Wh = 30000000 µWh
	expectedCurrent := uint64(2500000 * 12000000 / 1000000)
	if stats.Current != expectedCurrent {
		t.Errorf("Current = %v, want %v", stats.Current, expectedCurrent)
	}
}

func TestLinuxBatteryProvider_Stats_OutOfRange(t *testing.T) {
	tmpDir := t.TempDir()
	provider := &linuxBatteryProvider{
		powerSupplyPath: tmpDir,
	}

	_, err := provider.Stats(999)
	if err == nil {
		t.Error("Stats(999) should fail for out-of-range index")
	}
}
