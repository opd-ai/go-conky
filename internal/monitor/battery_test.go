package monitor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewBatteryReader(t *testing.T) {
	reader := newBatteryReader()
	if reader == nil {
		t.Fatal("newBatteryReader() returned nil")
	}
	if reader.powerSupplyPath != "/sys/class/power_supply" {
		t.Errorf("powerSupplyPath = %q, want %q", reader.powerSupplyPath, "/sys/class/power_supply")
	}
}

func TestBatteryReaderMissingDirectory(t *testing.T) {
	reader := &batteryReader{
		powerSupplyPath: "/nonexistent/power_supply",
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Errorf("ReadStats() error = %v, want nil for missing directory", err)
	}
	if len(stats.Batteries) != 0 {
		t.Errorf("Batteries count = %d, want 0 for missing directory", len(stats.Batteries))
	}
	if len(stats.ACAdapters) != 0 {
		t.Errorf("ACAdapters count = %d, want 0 for missing directory", len(stats.ACAdapters))
	}
}

func TestBatteryReaderWithMockBattery(t *testing.T) {
	tmpDir := t.TempDir()

	// Create mock BAT0 battery
	bat0 := filepath.Join(tmpDir, "BAT0")
	if err := os.MkdirAll(bat0, 0o755); err != nil {
		t.Fatalf("failed to create BAT0 directory: %v", err)
	}

	// Write battery attributes
	writeFile(t, bat0, "type", "Battery")
	writeFile(t, bat0, "present", "1")
	writeFile(t, bat0, "status", "Discharging")
	writeFile(t, bat0, "capacity", "75")
	writeFile(t, bat0, "capacity_level", "Normal")
	writeFile(t, bat0, "energy_now", "30000000")         // 30 Wh in µWh
	writeFile(t, bat0, "energy_full", "40000000")        // 40 Wh in µWh
	writeFile(t, bat0, "energy_full_design", "45000000") // 45 Wh in µWh
	writeFile(t, bat0, "power_now", "15000000")          // 15 W in µW
	writeFile(t, bat0, "voltage_now", "12000000")        // 12 V in µV
	writeFile(t, bat0, "voltage_min_design", "11100000") // 11.1 V in µV
	writeFile(t, bat0, "technology", "Li-ion")
	writeFile(t, bat0, "manufacturer", "TestMfr")
	writeFile(t, bat0, "model_name", "TestModel")
	writeFile(t, bat0, "serial_number", "12345")
	writeFile(t, bat0, "cycle_count", "100")

	reader := &batteryReader{
		powerSupplyPath: tmpDir,
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	if len(stats.Batteries) != 1 {
		t.Errorf("Batteries count = %d, want 1", len(stats.Batteries))
	}

	battery, ok := stats.Batteries["BAT0"]
	if !ok {
		t.Fatal("BAT0 battery not found")
	}

	// Verify basic attributes
	if battery.Name != "BAT0" {
		t.Errorf("Name = %q, want %q", battery.Name, "BAT0")
	}
	if !battery.Present {
		t.Error("Present = false, want true")
	}
	if battery.Status != "Discharging" {
		t.Errorf("Status = %q, want %q", battery.Status, "Discharging")
	}
	if battery.Capacity != 75 {
		t.Errorf("Capacity = %d, want 75", battery.Capacity)
	}
	if battery.CapacityLevel != "Normal" {
		t.Errorf("CapacityLevel = %q, want %q", battery.CapacityLevel, "Normal")
	}

	// Verify energy values
	if battery.EnergyNow != 30000000 {
		t.Errorf("EnergyNow = %d, want 30000000", battery.EnergyNow)
	}
	if battery.EnergyFull != 40000000 {
		t.Errorf("EnergyFull = %d, want 40000000", battery.EnergyFull)
	}
	if battery.EnergyFullDesign != 45000000 {
		t.Errorf("EnergyFullDesign = %d, want 45000000", battery.EnergyFullDesign)
	}

	// Verify power and voltage
	if battery.PowerNow != 15000000 {
		t.Errorf("PowerNow = %d, want 15000000", battery.PowerNow)
	}
	if battery.VoltageNow != 12000000 {
		t.Errorf("VoltageNow = %d, want 12000000", battery.VoltageNow)
	}
	if battery.VoltageMinDesign != 11100000 {
		t.Errorf("VoltageMinDesign = %d, want 11100000", battery.VoltageMinDesign)
	}

	// Verify info
	if battery.Technology != "Li-ion" {
		t.Errorf("Technology = %q, want %q", battery.Technology, "Li-ion")
	}
	if battery.Manufacturer != "TestMfr" {
		t.Errorf("Manufacturer = %q, want %q", battery.Manufacturer, "TestMfr")
	}
	if battery.ModelName != "TestModel" {
		t.Errorf("ModelName = %q, want %q", battery.ModelName, "TestModel")
	}
	if battery.SerialNumber != "12345" {
		t.Errorf("SerialNumber = %q, want %q", battery.SerialNumber, "12345")
	}
	if battery.CycleCount != 100 {
		t.Errorf("CycleCount = %d, want 100", battery.CycleCount)
	}

	// Verify calculated health (EnergyFull/EnergyFullDesign * 100)
	expectedHealth := float64(40000000) / float64(45000000) * 100
	if battery.Health < expectedHealth-0.1 || battery.Health > expectedHealth+0.1 {
		t.Errorf("Health = %f, want approximately %f", battery.Health, expectedHealth)
	}

	// Verify time to empty calculation (should be ~2 hours = 7200 seconds)
	expectedTimeToEmpty := float64(30000000) / float64(15000000) * 3600 // ~7200 seconds
	if battery.TimeToEmpty < expectedTimeToEmpty-1 || battery.TimeToEmpty > expectedTimeToEmpty+1 {
		t.Errorf("TimeToEmpty = %f, want approximately %f", battery.TimeToEmpty, expectedTimeToEmpty)
	}

	// Verify aggregate stats
	if !stats.IsDischarging {
		t.Error("IsDischarging = false, want true")
	}
	if stats.IsCharging {
		t.Error("IsCharging = true, want false")
	}
	if stats.TotalEnergyNow != 30000000 {
		t.Errorf("TotalEnergyNow = %d, want 30000000", stats.TotalEnergyNow)
	}
	if stats.TotalEnergyFull != 40000000 {
		t.Errorf("TotalEnergyFull = %d, want 40000000", stats.TotalEnergyFull)
	}
	expectedCapacity := float64(30000000) / float64(40000000) * 100
	if stats.TotalCapacity < expectedCapacity-0.1 || stats.TotalCapacity > expectedCapacity+0.1 {
		t.Errorf("TotalCapacity = %f, want approximately %f", stats.TotalCapacity, expectedCapacity)
	}
}

func TestBatteryReaderWithACAdapter(t *testing.T) {
	tmpDir := t.TempDir()

	// Create mock AC adapter
	ac0 := filepath.Join(tmpDir, "AC0")
	if err := os.MkdirAll(ac0, 0o755); err != nil {
		t.Fatalf("failed to create AC0 directory: %v", err)
	}

	writeFile(t, ac0, "type", "Mains")
	writeFile(t, ac0, "online", "1")

	reader := &batteryReader{
		powerSupplyPath: tmpDir,
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	if len(stats.ACAdapters) != 1 {
		t.Errorf("ACAdapters count = %d, want 1", len(stats.ACAdapters))
	}

	adapter, ok := stats.ACAdapters["AC0"]
	if !ok {
		t.Fatal("AC0 adapter not found")
	}

	if adapter.Name != "AC0" {
		t.Errorf("Name = %q, want %q", adapter.Name, "AC0")
	}
	if !adapter.Online {
		t.Error("Online = false, want true")
	}
	if !stats.ACOnline {
		t.Error("ACOnline = false, want true")
	}
}

func TestBatteryReaderACOffline(t *testing.T) {
	tmpDir := t.TempDir()

	ac0 := filepath.Join(tmpDir, "AC0")
	if err := os.MkdirAll(ac0, 0o755); err != nil {
		t.Fatalf("failed to create AC0 directory: %v", err)
	}

	writeFile(t, ac0, "type", "Mains")
	writeFile(t, ac0, "online", "0")

	reader := &batteryReader{
		powerSupplyPath: tmpDir,
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	adapter := stats.ACAdapters["AC0"]
	if adapter.Online {
		t.Error("Online = true, want false")
	}
	if stats.ACOnline {
		t.Error("ACOnline = true, want false")
	}
}

func TestBatteryReaderCharging(t *testing.T) {
	tmpDir := t.TempDir()

	bat0 := filepath.Join(tmpDir, "BAT0")
	if err := os.MkdirAll(bat0, 0o755); err != nil {
		t.Fatalf("failed to create BAT0 directory: %v", err)
	}

	writeFile(t, bat0, "type", "Battery")
	writeFile(t, bat0, "status", "Charging")
	writeFile(t, bat0, "capacity", "50")
	writeFile(t, bat0, "energy_now", "20000000")
	writeFile(t, bat0, "energy_full", "40000000")
	writeFile(t, bat0, "power_now", "20000000") // 20W charging rate

	reader := &batteryReader{
		powerSupplyPath: tmpDir,
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	battery := stats.Batteries["BAT0"]
	if battery.Status != "Charging" {
		t.Errorf("Status = %q, want %q", battery.Status, "Charging")
	}

	// Verify time to full calculation
	// Remaining: 40000000 - 20000000 = 20000000 µWh
	// At 20000000 µW = 1 hour = 3600 seconds
	expectedTimeToFull := float64(20000000) / float64(20000000) * 3600
	if battery.TimeToFull < expectedTimeToFull-1 || battery.TimeToFull > expectedTimeToFull+1 {
		t.Errorf("TimeToFull = %f, want approximately %f", battery.TimeToFull, expectedTimeToFull)
	}

	if !stats.IsCharging {
		t.Error("IsCharging = false, want true")
	}
}

func TestBatteryReaderChargeBased(t *testing.T) {
	// Test charge-based battery (µAh instead of µWh)
	tmpDir := t.TempDir()

	bat0 := filepath.Join(tmpDir, "BAT0")
	if err := os.MkdirAll(bat0, 0o755); err != nil {
		t.Fatalf("failed to create BAT0 directory: %v", err)
	}

	writeFile(t, bat0, "type", "Battery")
	writeFile(t, bat0, "status", "Discharging")
	writeFile(t, bat0, "capacity", "50")
	writeFile(t, bat0, "charge_now", "2500000")         // 2500 mAh in µAh
	writeFile(t, bat0, "charge_full", "5000000")        // 5000 mAh in µAh
	writeFile(t, bat0, "charge_full_design", "5500000") // 5500 mAh in µAh
	writeFile(t, bat0, "current_now", "1000000")        // 1000 mA in µA
	writeFile(t, bat0, "voltage_now", "12000000")       // 12V in µV

	reader := &batteryReader{
		powerSupplyPath: tmpDir,
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	battery := stats.Batteries["BAT0"]

	// Verify charge values
	if battery.ChargeNow != 2500000 {
		t.Errorf("ChargeNow = %d, want 2500000", battery.ChargeNow)
	}
	if battery.ChargeFull != 5000000 {
		t.Errorf("ChargeFull = %d, want 5000000", battery.ChargeFull)
	}
	if battery.ChargeFullDesign != 5500000 {
		t.Errorf("ChargeFullDesign = %d, want 5500000", battery.ChargeFullDesign)
	}
	if battery.CurrentNow != 1000000 {
		t.Errorf("CurrentNow = %d, want 1000000", battery.CurrentNow)
	}

	// Verify energy conversion (charge * voltage / 1000000)
	// EnergyNow = 2500000 * 12000000 / 1000000 = 30000000000 (30 Wh in µWh)
	expectedEnergy := uint64(2500000 * 12000000 / 1000000)
	if battery.EnergyNow != expectedEnergy {
		t.Errorf("EnergyNow = %d, want %d (converted from charge)", battery.EnergyNow, expectedEnergy)
	}
}

func TestBatteryReaderMultipleBatteries(t *testing.T) {
	tmpDir := t.TempDir()

	// Create BAT0
	bat0 := filepath.Join(tmpDir, "BAT0")
	if err := os.MkdirAll(bat0, 0o755); err != nil {
		t.Fatalf("failed to create BAT0: %v", err)
	}
	writeFile(t, bat0, "type", "Battery")
	writeFile(t, bat0, "capacity", "80")
	writeFile(t, bat0, "energy_now", "32000000")
	writeFile(t, bat0, "energy_full", "40000000")

	// Create BAT1
	bat1 := filepath.Join(tmpDir, "BAT1")
	if err := os.MkdirAll(bat1, 0o755); err != nil {
		t.Fatalf("failed to create BAT1: %v", err)
	}
	writeFile(t, bat1, "type", "Battery")
	writeFile(t, bat1, "capacity", "60")
	writeFile(t, bat1, "energy_now", "24000000")
	writeFile(t, bat1, "energy_full", "40000000")

	reader := &batteryReader{
		powerSupplyPath: tmpDir,
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	if len(stats.Batteries) != 2 {
		t.Errorf("Batteries count = %d, want 2", len(stats.Batteries))
	}

	// Verify aggregated values
	expectedTotalEnergy := uint64(32000000 + 24000000)
	if stats.TotalEnergyNow != expectedTotalEnergy {
		t.Errorf("TotalEnergyNow = %d, want %d", stats.TotalEnergyNow, expectedTotalEnergy)
	}

	expectedTotalFull := uint64(40000000 + 40000000)
	if stats.TotalEnergyFull != expectedTotalFull {
		t.Errorf("TotalEnergyFull = %d, want %d", stats.TotalEnergyFull, expectedTotalFull)
	}

	// Total capacity = (32000000 + 24000000) / (40000000 + 40000000) * 100 = 70%
	expectedCapacity := float64(expectedTotalEnergy) / float64(expectedTotalFull) * 100
	if stats.TotalCapacity < expectedCapacity-0.1 || stats.TotalCapacity > expectedCapacity+0.1 {
		t.Errorf("TotalCapacity = %f, want approximately %f", stats.TotalCapacity, expectedCapacity)
	}
}

func TestBatteryReaderEmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	reader := &batteryReader{
		powerSupplyPath: tmpDir,
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	if len(stats.Batteries) != 0 {
		t.Errorf("Batteries count = %d, want 0", len(stats.Batteries))
	}
	if len(stats.ACAdapters) != 0 {
		t.Errorf("ACAdapters count = %d, want 0", len(stats.ACAdapters))
	}
}

func TestBatteryReaderUnknownType(t *testing.T) {
	tmpDir := t.TempDir()

	// Create device with unknown type
	unknown := filepath.Join(tmpDir, "UNKNOWN0")
	if err := os.MkdirAll(unknown, 0o755); err != nil {
		t.Fatalf("failed to create UNKNOWN0: %v", err)
	}
	writeFile(t, unknown, "type", "USB")

	reader := &batteryReader{
		powerSupplyPath: tmpDir,
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	// Unknown type should be skipped
	if len(stats.Batteries) != 0 {
		t.Errorf("Batteries count = %d, want 0", len(stats.Batteries))
	}
	if len(stats.ACAdapters) != 0 {
		t.Errorf("ACAdapters count = %d, want 0", len(stats.ACAdapters))
	}
}

func TestBatteryReaderNoTypeFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create device without type file
	noType := filepath.Join(tmpDir, "NOTYPE")
	if err := os.MkdirAll(noType, 0o755); err != nil {
		t.Fatalf("failed to create NOTYPE: %v", err)
	}
	writeFile(t, noType, "capacity", "50")

	reader := &batteryReader{
		powerSupplyPath: tmpDir,
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	// Device without type should be skipped
	if len(stats.Batteries) != 0 {
		t.Errorf("Batteries count = %d, want 0", len(stats.Batteries))
	}
}

func TestBatteryReaderUPSType(t *testing.T) {
	tmpDir := t.TempDir()

	// Create UPS device (should be treated as AC adapter)
	ups := filepath.Join(tmpDir, "UPS0")
	if err := os.MkdirAll(ups, 0o755); err != nil {
		t.Fatalf("failed to create UPS0: %v", err)
	}
	writeFile(t, ups, "type", "UPS")
	writeFile(t, ups, "online", "1")

	reader := &batteryReader{
		powerSupplyPath: tmpDir,
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	if len(stats.ACAdapters) != 1 {
		t.Errorf("ACAdapters count = %d, want 1", len(stats.ACAdapters))
	}
	if !stats.ACOnline {
		t.Error("ACOnline = false, want true")
	}
}

func TestBatteryReaderBatteryNotPresent(t *testing.T) {
	tmpDir := t.TempDir()

	bat0 := filepath.Join(tmpDir, "BAT0")
	if err := os.MkdirAll(bat0, 0o755); err != nil {
		t.Fatalf("failed to create BAT0: %v", err)
	}
	writeFile(t, bat0, "type", "Battery")
	writeFile(t, bat0, "present", "0")

	reader := &batteryReader{
		powerSupplyPath: tmpDir,
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	battery := stats.Batteries["BAT0"]
	if battery.Present {
		t.Error("Present = true, want false")
	}
}

func TestBatteryReaderFullStatus(t *testing.T) {
	tmpDir := t.TempDir()

	bat0 := filepath.Join(tmpDir, "BAT0")
	if err := os.MkdirAll(bat0, 0o755); err != nil {
		t.Fatalf("failed to create BAT0: %v", err)
	}
	writeFile(t, bat0, "type", "Battery")
	writeFile(t, bat0, "status", "Full")
	writeFile(t, bat0, "capacity", "100")

	reader := &batteryReader{
		powerSupplyPath: tmpDir,
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	battery := stats.Batteries["BAT0"]
	if battery.Status != "Full" {
		t.Errorf("Status = %q, want %q", battery.Status, "Full")
	}
	if battery.Capacity != 100 {
		t.Errorf("Capacity = %d, want 100", battery.Capacity)
	}

	// Full status is neither charging nor discharging
	if stats.IsCharging {
		t.Error("IsCharging = true, want false for Full status")
	}
	if stats.IsDischarging {
		t.Error("IsDischarging = true, want false for Full status")
	}
}

func TestBatteryInfoStruct(t *testing.T) {
	info := BatteryInfo{
		Name:          "BAT0",
		Present:       true,
		Status:        "Discharging",
		Capacity:      75,
		CapacityLevel: "Normal",
		Technology:    "Li-ion",
		Manufacturer:  "TestMfr",
		ModelName:     "TestModel",
		SerialNumber:  "12345",
		CycleCount:    100,
		Health:        88.9,
		TimeToEmpty:   7200.0,
	}

	if info.Name != "BAT0" {
		t.Errorf("Name = %q, want %q", info.Name, "BAT0")
	}
	if info.Capacity != 75 {
		t.Errorf("Capacity = %d, want 75", info.Capacity)
	}
	if info.Health < 88.8 || info.Health > 89.0 {
		t.Errorf("Health = %f, want approximately 88.9", info.Health)
	}
}

func TestACAdapterInfoStruct(t *testing.T) {
	info := ACAdapterInfo{
		Name:   "AC0",
		Online: true,
	}

	if info.Name != "AC0" {
		t.Errorf("Name = %q, want %q", info.Name, "AC0")
	}
	if !info.Online {
		t.Error("Online = false, want true")
	}
}

func TestBatteryStatsStruct(t *testing.T) {
	stats := BatteryStats{
		Batteries:       make(map[string]BatteryInfo),
		ACAdapters:      make(map[string]ACAdapterInfo),
		ACOnline:        true,
		TotalCapacity:   75.0,
		TotalEnergyNow:  30000000,
		TotalEnergyFull: 40000000,
		IsCharging:      false,
		IsDischarging:   true,
	}

	stats.Batteries["BAT0"] = BatteryInfo{Name: "BAT0"}
	stats.ACAdapters["AC0"] = ACAdapterInfo{Name: "AC0"}

	if len(stats.Batteries) != 1 {
		t.Errorf("Batteries count = %d, want 1", len(stats.Batteries))
	}
	if len(stats.ACAdapters) != 1 {
		t.Errorf("ACAdapters count = %d, want 1", len(stats.ACAdapters))
	}
	if !stats.ACOnline {
		t.Error("ACOnline = false, want true")
	}
	if stats.TotalCapacity != 75.0 {
		t.Errorf("TotalCapacity = %f, want 75.0", stats.TotalCapacity)
	}
}

func TestSystemDataBatteryGetSet(t *testing.T) {
	sd := NewSystemData()

	battery := BatteryStats{
		Batteries:     make(map[string]BatteryInfo),
		ACAdapters:    make(map[string]ACAdapterInfo),
		ACOnline:      true,
		TotalCapacity: 80.0,
	}
	battery.Batteries["BAT0"] = BatteryInfo{Name: "BAT0", Capacity: 80}
	battery.ACAdapters["AC0"] = ACAdapterInfo{Name: "AC0", Online: true}

	sd.setBattery(battery)

	got := sd.GetBattery()
	if got.TotalCapacity != 80.0 {
		t.Errorf("TotalCapacity = %f, want 80.0", got.TotalCapacity)
	}
	if !got.ACOnline {
		t.Error("ACOnline = false, want true")
	}
	if len(got.Batteries) != 1 {
		t.Errorf("Batteries count = %d, want 1", len(got.Batteries))
	}
	if len(got.ACAdapters) != 1 {
		t.Errorf("ACAdapters count = %d, want 1", len(got.ACAdapters))
	}

	// Verify deep copy - modifying returned value shouldn't affect stored value
	got.Batteries["BAT1"] = BatteryInfo{Name: "BAT1"}
	stored := sd.GetBattery()
	if len(stored.Batteries) != 1 {
		t.Errorf("Stored Batteries count = %d after modification, want 1", len(stored.Batteries))
	}
}

// writeFile is a test helper to write content to a file
func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content+"\n"), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", name, err)
	}
}
