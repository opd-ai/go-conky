package monitor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewHwmonReader(t *testing.T) {
	reader := newHwmonReader()
	if reader == nil {
		t.Fatal("newHwmonReader() returned nil")
	}
	if reader.hwmonPath != "/sys/class/hwmon" {
		t.Errorf("hwmonPath = %q, want %q", reader.hwmonPath, "/sys/class/hwmon")
	}
}

func TestHwmonReaderMissingDirectory(t *testing.T) {
	reader := &hwmonReader{
		hwmonPath: "/nonexistent/hwmon",
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Errorf("ReadStats() error = %v, want nil for missing directory", err)
	}
	if len(stats.Devices) != 0 {
		t.Errorf("Devices count = %d, want 0 for missing directory", len(stats.Devices))
	}
	if len(stats.TempSensors) != 0 {
		t.Errorf("TempSensors count = %d, want 0 for missing directory", len(stats.TempSensors))
	}
}

func TestHwmonReaderWithMockDevice(t *testing.T) {
	tmpDir := t.TempDir()

	// Create mock hwmon0 device (coretemp)
	hwmon0 := filepath.Join(tmpDir, "hwmon0")
	if err := os.MkdirAll(hwmon0, 0o755); err != nil {
		t.Fatalf("failed to create hwmon0 directory: %v", err)
	}

	// Write device name
	if err := os.WriteFile(filepath.Join(hwmon0, "name"), []byte("coretemp\n"), 0o644); err != nil {
		t.Fatalf("failed to write name: %v", err)
	}

	// Write temp1 sensor (Package)
	if err := os.WriteFile(filepath.Join(hwmon0, "temp1_input"), []byte("45000\n"), 0o644); err != nil {
		t.Fatalf("failed to write temp1_input: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hwmon0, "temp1_label"), []byte("Package id 0\n"), 0o644); err != nil {
		t.Fatalf("failed to write temp1_label: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hwmon0, "temp1_max"), []byte("100000\n"), 0o644); err != nil {
		t.Fatalf("failed to write temp1_max: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hwmon0, "temp1_crit"), []byte("110000\n"), 0o644); err != nil {
		t.Fatalf("failed to write temp1_crit: %v", err)
	}

	// Write temp2 sensor (Core 0)
	if err := os.WriteFile(filepath.Join(hwmon0, "temp2_input"), []byte("42000\n"), 0o644); err != nil {
		t.Fatalf("failed to write temp2_input: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hwmon0, "temp2_label"), []byte("Core 0\n"), 0o644); err != nil {
		t.Fatalf("failed to write temp2_label: %v", err)
	}

	reader := &hwmonReader{
		hwmonPath: tmpDir,
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	if len(stats.Devices) != 1 {
		t.Errorf("Devices count = %d, want 1", len(stats.Devices))
	}

	device, ok := stats.Devices["coretemp"]
	if !ok {
		t.Fatal("coretemp device not found")
	}

	if device.Name != "coretemp" {
		t.Errorf("Device.Name = %q, want %q", device.Name, "coretemp")
	}

	if len(device.Temps) != 2 {
		t.Errorf("Temps count = %d, want 2", len(device.Temps))
	}

	// Verify temp1 (Package)
	temp1, ok := device.Temps["temp1"]
	if !ok {
		t.Fatal("temp1 sensor not found")
	}
	if temp1.Label != "Package id 0" {
		t.Errorf("temp1.Label = %q, want %q", temp1.Label, "Package id 0")
	}
	if temp1.Input != 45000 {
		t.Errorf("temp1.Input = %d, want 45000", temp1.Input)
	}
	if temp1.InputCelsius != 45.0 {
		t.Errorf("temp1.InputCelsius = %f, want 45.0", temp1.InputCelsius)
	}
	if temp1.Max != 100000 {
		t.Errorf("temp1.Max = %d, want 100000", temp1.Max)
	}
	if temp1.MaxCelsius != 100.0 {
		t.Errorf("temp1.MaxCelsius = %f, want 100.0", temp1.MaxCelsius)
	}
	if temp1.Crit != 110000 {
		t.Errorf("temp1.Crit = %d, want 110000", temp1.Crit)
	}
	if temp1.CritCelsius != 110.0 {
		t.Errorf("temp1.CritCelsius = %f, want 110.0", temp1.CritCelsius)
	}

	// Verify temp2 (Core 0)
	temp2, ok := device.Temps["temp2"]
	if !ok {
		t.Fatal("temp2 sensor not found")
	}
	if temp2.Label != "Core 0" {
		t.Errorf("temp2.Label = %q, want %q", temp2.Label, "Core 0")
	}
	if temp2.InputCelsius != 42.0 {
		t.Errorf("temp2.InputCelsius = %f, want 42.0", temp2.InputCelsius)
	}

	// Verify TempSensors list
	if len(stats.TempSensors) != 2 {
		t.Errorf("TempSensors count = %d, want 2", len(stats.TempSensors))
	}
}

func TestHwmonReaderWithMultipleDevices(t *testing.T) {
	tmpDir := t.TempDir()

	// Create hwmon0 (coretemp)
	hwmon0 := filepath.Join(tmpDir, "hwmon0")
	if err := os.MkdirAll(hwmon0, 0o755); err != nil {
		t.Fatalf("failed to create hwmon0: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hwmon0, "name"), []byte("coretemp\n"), 0o644); err != nil {
		t.Fatalf("failed to write name: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hwmon0, "temp1_input"), []byte("50000\n"), 0o644); err != nil {
		t.Fatalf("failed to write temp1_input: %v", err)
	}

	// Create hwmon1 (acpitz)
	hwmon1 := filepath.Join(tmpDir, "hwmon1")
	if err := os.MkdirAll(hwmon1, 0o755); err != nil {
		t.Fatalf("failed to create hwmon1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hwmon1, "name"), []byte("acpitz\n"), 0o644); err != nil {
		t.Fatalf("failed to write name: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hwmon1, "temp1_input"), []byte("40000\n"), 0o644); err != nil {
		t.Fatalf("failed to write temp1_input: %v", err)
	}

	reader := &hwmonReader{
		hwmonPath: tmpDir,
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	if len(stats.Devices) != 2 {
		t.Errorf("Devices count = %d, want 2", len(stats.Devices))
	}

	if _, ok := stats.Devices["coretemp"]; !ok {
		t.Error("coretemp device not found")
	}
	if _, ok := stats.Devices["acpitz"]; !ok {
		t.Error("acpitz device not found")
	}

	if len(stats.TempSensors) != 2 {
		t.Errorf("TempSensors count = %d, want 2", len(stats.TempSensors))
	}
}

func TestHwmonReaderNoName(t *testing.T) {
	tmpDir := t.TempDir()

	// Create hwmon0 without a name file
	hwmon0 := filepath.Join(tmpDir, "hwmon0")
	if err := os.MkdirAll(hwmon0, 0o755); err != nil {
		t.Fatalf("failed to create hwmon0: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hwmon0, "temp1_input"), []byte("50000\n"), 0o644); err != nil {
		t.Fatalf("failed to write temp1_input: %v", err)
	}

	reader := &hwmonReader{
		hwmonPath: tmpDir,
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	if len(stats.Devices) != 1 {
		t.Errorf("Devices count = %d, want 1", len(stats.Devices))
	}

	// Should use directory name as fallback
	if _, ok := stats.Devices["hwmon0"]; !ok {
		t.Error("hwmon0 device not found (fallback name)")
	}
}

func TestHwmonReaderInvalidTempInput(t *testing.T) {
	tmpDir := t.TempDir()

	hwmon0 := filepath.Join(tmpDir, "hwmon0")
	if err := os.MkdirAll(hwmon0, 0o755); err != nil {
		t.Fatalf("failed to create hwmon0: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hwmon0, "name"), []byte("test\n"), 0o644); err != nil {
		t.Fatalf("failed to write name: %v", err)
	}
	// Write invalid temperature value
	if err := os.WriteFile(filepath.Join(hwmon0, "temp1_input"), []byte("not_a_number\n"), 0o644); err != nil {
		t.Fatalf("failed to write temp1_input: %v", err)
	}

	reader := &hwmonReader{
		hwmonPath: tmpDir,
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	device := stats.Devices["test"]
	// Invalid sensor should be skipped
	if len(device.Temps) != 0 {
		t.Errorf("Temps count = %d, want 0 for invalid input", len(device.Temps))
	}
}

func TestHwmonReaderNoLabel(t *testing.T) {
	tmpDir := t.TempDir()

	hwmon0 := filepath.Join(tmpDir, "hwmon0")
	if err := os.MkdirAll(hwmon0, 0o755); err != nil {
		t.Fatalf("failed to create hwmon0: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hwmon0, "name"), []byte("test\n"), 0o644); err != nil {
		t.Fatalf("failed to write name: %v", err)
	}
	// Only temp input, no label
	if err := os.WriteFile(filepath.Join(hwmon0, "temp1_input"), []byte("45000\n"), 0o644); err != nil {
		t.Fatalf("failed to write temp1_input: %v", err)
	}

	reader := &hwmonReader{
		hwmonPath: tmpDir,
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	device := stats.Devices["test"]
	temp1 := device.Temps["temp1"]
	// Should use sensor type as label fallback
	if temp1.Label != "temp1" {
		t.Errorf("Label = %q, want %q (fallback)", temp1.Label, "temp1")
	}
}

func TestHwmonReaderEmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	reader := &hwmonReader{
		hwmonPath: tmpDir,
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	if len(stats.Devices) != 0 {
		t.Errorf("Devices count = %d, want 0", len(stats.Devices))
	}
}

func TestHwmonReaderNonHwmonDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a directory that doesn't match hwmon pattern
	otherDir := filepath.Join(tmpDir, "other_device")
	if err := os.MkdirAll(otherDir, 0o755); err != nil {
		t.Fatalf("failed to create other_device: %v", err)
	}

	reader := &hwmonReader{
		hwmonPath: tmpDir,
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	// Should skip non-hwmon directories
	if len(stats.Devices) != 0 {
		t.Errorf("Devices count = %d, want 0", len(stats.Devices))
	}
}

func TestHwmonReaderNegativeTemperature(t *testing.T) {
	tmpDir := t.TempDir()

	hwmon0 := filepath.Join(tmpDir, "hwmon0")
	if err := os.MkdirAll(hwmon0, 0o755); err != nil {
		t.Fatalf("failed to create hwmon0: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hwmon0, "name"), []byte("test\n"), 0o644); err != nil {
		t.Fatalf("failed to write name: %v", err)
	}
	// Negative temperature (can happen with some sensors)
	if err := os.WriteFile(filepath.Join(hwmon0, "temp1_input"), []byte("-5000\n"), 0o644); err != nil {
		t.Fatalf("failed to write temp1_input: %v", err)
	}

	reader := &hwmonReader{
		hwmonPath: tmpDir,
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	device := stats.Devices["test"]
	temp1 := device.Temps["temp1"]
	if temp1.Input != -5000 {
		t.Errorf("Input = %d, want -5000", temp1.Input)
	}
	if temp1.InputCelsius != -5.0 {
		t.Errorf("InputCelsius = %f, want -5.0", temp1.InputCelsius)
	}
}

func TestTempSensorStruct(t *testing.T) {
	sensor := TempSensor{
		Label:        "Core 0",
		Input:        50000,
		InputCelsius: 50.0,
		Max:          100000,
		MaxCelsius:   100.0,
		Crit:         110000,
		CritCelsius:  110.0,
		Type:         "temp1",
	}

	if sensor.Label != "Core 0" {
		t.Errorf("Label = %q, want %q", sensor.Label, "Core 0")
	}
	if sensor.InputCelsius != 50.0 {
		t.Errorf("InputCelsius = %f, want 50.0", sensor.InputCelsius)
	}
	if sensor.Input != 50000 {
		t.Errorf("Input = %d, want 50000", sensor.Input)
	}
	if sensor.Max != 100000 {
		t.Errorf("Max = %d, want 100000", sensor.Max)
	}
	if sensor.MaxCelsius != 100.0 {
		t.Errorf("MaxCelsius = %f, want 100.0", sensor.MaxCelsius)
	}
	if sensor.Crit != 110000 {
		t.Errorf("Crit = %d, want 110000", sensor.Crit)
	}
	if sensor.CritCelsius != 110.0 {
		t.Errorf("CritCelsius = %f, want 110.0", sensor.CritCelsius)
	}
	if sensor.Type != "temp1" {
		t.Errorf("Type = %q, want %q", sensor.Type, "temp1")
	}
}

func TestHwmonDeviceStruct(t *testing.T) {
	device := HwmonDevice{
		Name:  "coretemp",
		Path:  "/sys/class/hwmon/hwmon0",
		Temps: make(map[string]TempSensor),
	}

	device.Temps["temp1"] = TempSensor{
		Label:        "Package id 0",
		InputCelsius: 45.0,
	}

	if device.Name != "coretemp" {
		t.Errorf("Name = %q, want %q", device.Name, "coretemp")
	}
	if device.Path != "/sys/class/hwmon/hwmon0" {
		t.Errorf("Path = %q, want %q", device.Path, "/sys/class/hwmon/hwmon0")
	}
	if len(device.Temps) != 1 {
		t.Errorf("Temps count = %d, want 1", len(device.Temps))
	}
}

func TestHwmonStatsStruct(t *testing.T) {
	stats := HwmonStats{
		Devices:     make(map[string]HwmonDevice),
		TempSensors: make([]TempSensor, 0),
	}

	stats.Devices["test"] = HwmonDevice{Name: "test"}
	stats.TempSensors = append(stats.TempSensors, TempSensor{Label: "Test Sensor"})

	if len(stats.Devices) != 1 {
		t.Errorf("Devices count = %d, want 1", len(stats.Devices))
	}
	if len(stats.TempSensors) != 1 {
		t.Errorf("TempSensors count = %d, want 1", len(stats.TempSensors))
	}
}
