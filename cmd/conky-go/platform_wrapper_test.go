package main

import (
	"context"
	"testing"
	"time"

	"github.com/opd-ai/go-conky/internal/monitor"
	"github.com/opd-ai/go-conky/internal/platform"
)

// mockPlatform implements platform.Platform for testing.
type mockPlatform struct {
	name       string
	cpu        platform.CPUProvider
	memory     platform.MemoryProvider
	network    platform.NetworkProvider
	filesystem platform.FilesystemProvider
	battery    platform.BatteryProvider
	sensors    platform.SensorProvider
}

func (m *mockPlatform) Name() string                              { return m.name }
func (m *mockPlatform) Initialize(_ context.Context) error        { return nil }
func (m *mockPlatform) Close() error                              { return nil }
func (m *mockPlatform) CPU() platform.CPUProvider                 { return m.cpu }
func (m *mockPlatform) Memory() platform.MemoryProvider           { return m.memory }
func (m *mockPlatform) Network() platform.NetworkProvider         { return m.network }
func (m *mockPlatform) Filesystem() platform.FilesystemProvider   { return m.filesystem }
func (m *mockPlatform) Battery() platform.BatteryProvider         { return m.battery }
func (m *mockPlatform) Sensors() platform.SensorProvider          { return m.sensors }

// mockCPUProvider implements platform.CPUProvider for testing.
type mockCPUProvider struct {
	usage     []float64
	total     float64
	frequency []float64
	info      *platform.CPUInfo
	load1     float64
	load5     float64
	load15    float64
}

func (m *mockCPUProvider) Usage() ([]float64, error)                       { return m.usage, nil }
func (m *mockCPUProvider) TotalUsage() (float64, error)                    { return m.total, nil }
func (m *mockCPUProvider) Frequency() ([]float64, error)                   { return m.frequency, nil }
func (m *mockCPUProvider) Info() (*platform.CPUInfo, error)                { return m.info, nil }
func (m *mockCPUProvider) LoadAverage() (float64, float64, float64, error) { return m.load1, m.load5, m.load15, nil }

// mockMemoryProvider implements platform.MemoryProvider for testing.
type mockMemoryProvider struct {
	stats     *platform.MemoryStats
	swapStats *platform.SwapStats
}

func (m *mockMemoryProvider) Stats() (*platform.MemoryStats, error)   { return m.stats, nil }
func (m *mockMemoryProvider) SwapStats() (*platform.SwapStats, error) { return m.swapStats, nil }

// mockNetworkProvider implements platform.NetworkProvider for testing.
type mockNetworkProvider struct {
	interfaces []string
	stats      map[string]*platform.NetworkStats
}

func (m *mockNetworkProvider) Interfaces() ([]string, error) { return m.interfaces, nil }
func (m *mockNetworkProvider) Stats(name string) (*platform.NetworkStats, error) {
	return m.stats[name], nil
}
func (m *mockNetworkProvider) AllStats() (map[string]*platform.NetworkStats, error) {
	return m.stats, nil
}

// mockFilesystemProvider implements platform.FilesystemProvider for testing.
type mockFilesystemProvider struct {
	mounts []platform.MountInfo
	stats  map[string]*platform.FilesystemStats
	diskIO map[string]*platform.DiskIOStats
}

func (m *mockFilesystemProvider) Mounts() ([]platform.MountInfo, error) { return m.mounts, nil }
func (m *mockFilesystemProvider) Stats(mp string) (*platform.FilesystemStats, error) {
	return m.stats[mp], nil
}
func (m *mockFilesystemProvider) DiskIO(dev string) (*platform.DiskIOStats, error) {
	return m.diskIO[dev], nil
}

// mockBatteryProvider implements platform.BatteryProvider for testing.
type mockBatteryProvider struct {
	count int
	stats []*platform.BatteryStats
}

func (m *mockBatteryProvider) Count() int { return m.count }
func (m *mockBatteryProvider) Stats(idx int) (*platform.BatteryStats, error) {
	if idx < 0 || idx >= len(m.stats) {
		return nil, nil
	}
	return m.stats[idx], nil
}

// mockSensorProvider implements platform.SensorProvider for testing.
type mockSensorProvider struct {
	temps []platform.SensorReading
	fans  []platform.SensorReading
}

func (m *mockSensorProvider) Temperatures() ([]platform.SensorReading, error) { return m.temps, nil }
func (m *mockSensorProvider) Fans() ([]platform.SensorReading, error)         { return m.fans, nil }

func TestWrapPlatform_Nil(t *testing.T) {
	wrapper := WrapPlatform(nil)
	if wrapper != nil {
		t.Error("WrapPlatform(nil) should return nil")
	}
}

func TestWrapPlatform_Name(t *testing.T) {
	mock := &mockPlatform{name: "test-platform"}
	wrapper := WrapPlatform(mock)

	if wrapper == nil {
		t.Fatal("WrapPlatform should not return nil for valid platform")
	}

	if got := wrapper.Name(); got != "test-platform" {
		t.Errorf("Name() = %q, want %q", got, "test-platform")
	}
}

func TestWrapPlatform_NilProviders(t *testing.T) {
	mock := &mockPlatform{name: "empty"}
	wrapper := WrapPlatform(mock)

	if wrapper.CPU() != nil {
		t.Error("CPU() should return nil when platform.CPU() is nil")
	}
	if wrapper.Memory() != nil {
		t.Error("Memory() should return nil when platform.Memory() is nil")
	}
	if wrapper.Network() != nil {
		t.Error("Network() should return nil when platform.Network() is nil")
	}
	if wrapper.Filesystem() != nil {
		t.Error("Filesystem() should return nil when platform.Filesystem() is nil")
	}
	if wrapper.Battery() != nil {
		t.Error("Battery() should return nil when platform.Battery() is nil")
	}
	if wrapper.Sensors() != nil {
		t.Error("Sensors() should return nil when platform.Sensors() is nil")
	}
}

func TestCPUProviderWrapper(t *testing.T) {
	mockCPU := &mockCPUProvider{
		usage:     []float64{10.5, 20.3, 15.7, 8.2},
		total:     13.7,
		frequency: []float64{3200.0, 3400.0, 3100.0, 3300.0},
		info: &platform.CPUInfo{
			Model:     "Test CPU",
			Vendor:    "TestVendor",
			Cores:     4,
			Threads:   8,
			CacheSize: 8388608,
		},
		load1:  0.5,
		load5:  1.2,
		load15: 0.8,
	}

	mock := &mockPlatform{name: "cpu-test", cpu: mockCPU}
	wrapper := WrapPlatform(mock)
	cpuWrapper := wrapper.CPU()

	if cpuWrapper == nil {
		t.Fatal("CPU() should not return nil")
	}

	// Test Usage
	usage, err := cpuWrapper.Usage()
	if err != nil {
		t.Errorf("Usage() error = %v", err)
	}
	if len(usage) != 4 {
		t.Errorf("Usage() returned %d values, want 4", len(usage))
	}

	// Test TotalUsage
	total, err := cpuWrapper.TotalUsage()
	if err != nil {
		t.Errorf("TotalUsage() error = %v", err)
	}
	if total != 13.7 {
		t.Errorf("TotalUsage() = %v, want %v", total, 13.7)
	}

	// Test Info
	info, err := cpuWrapper.Info()
	if err != nil {
		t.Errorf("Info() error = %v", err)
	}
	if info == nil {
		t.Fatal("Info() returned nil")
	}
	if info.Model != "Test CPU" {
		t.Errorf("Info().Model = %q, want %q", info.Model, "Test CPU")
	}
	if info.Cores != 4 {
		t.Errorf("Info().Cores = %d, want %d", info.Cores, 4)
	}

	// Test LoadAverage
	l1, l5, l15, err := cpuWrapper.LoadAverage()
	if err != nil {
		t.Errorf("LoadAverage() error = %v", err)
	}
	if l1 != 0.5 || l5 != 1.2 || l15 != 0.8 {
		t.Errorf("LoadAverage() = (%v, %v, %v), want (0.5, 1.2, 0.8)", l1, l5, l15)
	}
}

func TestMemoryProviderWrapper(t *testing.T) {
	mockMem := &mockMemoryProvider{
		stats: &platform.MemoryStats{
			Total:       16000000000,
			Used:        8000000000,
			Free:        4000000000,
			Available:   6000000000,
			Cached:      3000000000,
			Buffers:     500000000,
			UsedPercent: 50.0,
		},
		swapStats: &platform.SwapStats{
			Total:       4000000000,
			Used:        1000000000,
			Free:        3000000000,
			UsedPercent: 25.0,
		},
	}

	mock := &mockPlatform{name: "mem-test", memory: mockMem}
	wrapper := WrapPlatform(mock)
	memWrapper := wrapper.Memory()

	if memWrapper == nil {
		t.Fatal("Memory() should not return nil")
	}

	// Test Stats
	stats, err := memWrapper.Stats()
	if err != nil {
		t.Errorf("Stats() error = %v", err)
	}
	if stats == nil {
		t.Fatal("Stats() returned nil")
	}
	if stats.Total != 16000000000 {
		t.Errorf("Stats().Total = %d, want %d", stats.Total, 16000000000)
	}
	if stats.UsedPercent != 50.0 {
		t.Errorf("Stats().UsedPercent = %v, want %v", stats.UsedPercent, 50.0)
	}

	// Test SwapStats
	swap, err := memWrapper.SwapStats()
	if err != nil {
		t.Errorf("SwapStats() error = %v", err)
	}
	if swap == nil {
		t.Fatal("SwapStats() returned nil")
	}
	if swap.UsedPercent != 25.0 {
		t.Errorf("SwapStats().UsedPercent = %v, want %v", swap.UsedPercent, 25.0)
	}
}

func TestNetworkProviderWrapper(t *testing.T) {
	mockNet := &mockNetworkProvider{
		interfaces: []string{"eth0", "lo"},
		stats: map[string]*platform.NetworkStats{
			"eth0": {
				BytesRecv:   1000000,
				BytesSent:   500000,
				PacketsRecv: 10000,
				PacketsSent: 5000,
				ErrorsIn:    0,
				ErrorsOut:   0,
				DropIn:      0,
				DropOut:     0,
			},
		},
	}

	mock := &mockPlatform{name: "net-test", network: mockNet}
	wrapper := WrapPlatform(mock)
	netWrapper := wrapper.Network()

	if netWrapper == nil {
		t.Fatal("Network() should not return nil")
	}

	// Test Interfaces
	ifaces, err := netWrapper.Interfaces()
	if err != nil {
		t.Errorf("Interfaces() error = %v", err)
	}
	if len(ifaces) != 2 {
		t.Errorf("Interfaces() returned %d, want 2", len(ifaces))
	}

	// Test Stats
	stats, err := netWrapper.Stats("eth0")
	if err != nil {
		t.Errorf("Stats() error = %v", err)
	}
	if stats == nil {
		t.Fatal("Stats() returned nil")
	}
	if stats.BytesRecv != 1000000 {
		t.Errorf("Stats().BytesRecv = %d, want %d", stats.BytesRecv, 1000000)
	}

	// Test AllStats
	allStats, err := netWrapper.AllStats()
	if err != nil {
		t.Errorf("AllStats() error = %v", err)
	}
	if allStats == nil {
		t.Fatal("AllStats() returned nil")
	}
	if len(allStats) != 1 {
		t.Errorf("AllStats() returned %d entries, want 1", len(allStats))
	}
}

func TestFilesystemProviderWrapper(t *testing.T) {
	mockFS := &mockFilesystemProvider{
		mounts: []platform.MountInfo{
			{Device: "/dev/sda1", MountPoint: "/", FSType: "ext4", Options: []string{"rw", "relatime"}},
		},
		stats: map[string]*platform.FilesystemStats{
			"/": {
				Total:       500000000000,
				Used:        250000000000,
				Free:        250000000000,
				UsedPercent: 50.0,
				InodesTotal: 30000000,
				InodesUsed:  1000000,
				InodesFree:  29000000,
			},
		},
		diskIO: map[string]*platform.DiskIOStats{
			"/dev/sda": {
				ReadBytes:  10000000,
				WriteBytes: 5000000,
				ReadCount:  1000,
				WriteCount: 500,
				ReadTime:   100 * time.Millisecond,
				WriteTime:  50 * time.Millisecond,
			},
		},
	}

	mock := &mockPlatform{name: "fs-test", filesystem: mockFS}
	wrapper := WrapPlatform(mock)
	fsWrapper := wrapper.Filesystem()

	if fsWrapper == nil {
		t.Fatal("Filesystem() should not return nil")
	}

	// Test Mounts
	mounts, err := fsWrapper.Mounts()
	if err != nil {
		t.Errorf("Mounts() error = %v", err)
	}
	if len(mounts) != 1 {
		t.Errorf("Mounts() returned %d, want 1", len(mounts))
	}
	if mounts[0].MountPoint != "/" {
		t.Errorf("Mounts()[0].MountPoint = %q, want %q", mounts[0].MountPoint, "/")
	}

	// Test Stats
	stats, err := fsWrapper.Stats("/")
	if err != nil {
		t.Errorf("Stats() error = %v", err)
	}
	if stats == nil {
		t.Fatal("Stats() returned nil")
	}
	if stats.UsedPercent != 50.0 {
		t.Errorf("Stats().UsedPercent = %v, want %v", stats.UsedPercent, 50.0)
	}

	// Test DiskIO
	io, err := fsWrapper.DiskIO("/dev/sda")
	if err != nil {
		t.Errorf("DiskIO() error = %v", err)
	}
	if io == nil {
		t.Fatal("DiskIO() returned nil")
	}
	if io.ReadBytes != 10000000 {
		t.Errorf("DiskIO().ReadBytes = %d, want %d", io.ReadBytes, 10000000)
	}
}

func TestBatteryProviderWrapper(t *testing.T) {
	mockBat := &mockBatteryProvider{
		count: 1,
		stats: []*platform.BatteryStats{
			{
				Percent:       75.0,
				TimeRemaining: 2 * time.Hour,
				Charging:      false,
				FullCapacity:  50000,
				Current:       37500,
				Voltage:       11.4,
			},
		},
	}

	mock := &mockPlatform{name: "bat-test", battery: mockBat}
	wrapper := WrapPlatform(mock)
	batWrapper := wrapper.Battery()

	if batWrapper == nil {
		t.Fatal("Battery() should not return nil")
	}

	// Test Count
	if count := batWrapper.Count(); count != 1 {
		t.Errorf("Count() = %d, want 1", count)
	}

	// Test Stats
	stats, err := batWrapper.Stats(0)
	if err != nil {
		t.Errorf("Stats() error = %v", err)
	}
	if stats == nil {
		t.Fatal("Stats() returned nil")
	}
	if stats.Percent != 75.0 {
		t.Errorf("Stats().Percent = %v, want %v", stats.Percent, 75.0)
	}
	if stats.Charging {
		t.Error("Stats().Charging should be false")
	}
}

func TestSensorProviderWrapper(t *testing.T) {
	mockSensors := &mockSensorProvider{
		temps: []platform.SensorReading{
			{Name: "coretemp", Label: "Core 0", Value: 45.0, Unit: "°C", Critical: 100.0},
			{Name: "coretemp", Label: "Core 1", Value: 47.0, Unit: "°C", Critical: 100.0},
		},
		fans: []platform.SensorReading{
			{Name: "thinkpad", Label: "Fan1", Value: 2500.0, Unit: "RPM", Critical: 0},
		},
	}

	mock := &mockPlatform{name: "sensor-test", sensors: mockSensors}
	wrapper := WrapPlatform(mock)
	sensorWrapper := wrapper.Sensors()

	if sensorWrapper == nil {
		t.Fatal("Sensors() should not return nil")
	}

	// Test Temperatures
	temps, err := sensorWrapper.Temperatures()
	if err != nil {
		t.Errorf("Temperatures() error = %v", err)
	}
	if len(temps) != 2 {
		t.Errorf("Temperatures() returned %d, want 2", len(temps))
	}
	if temps[0].Value != 45.0 {
		t.Errorf("Temperatures()[0].Value = %v, want %v", temps[0].Value, 45.0)
	}

	// Test Fans
	fans, err := sensorWrapper.Fans()
	if err != nil {
		t.Errorf("Fans() error = %v", err)
	}
	if len(fans) != 1 {
		t.Errorf("Fans() returned %d, want 1", len(fans))
	}
	if fans[0].Value != 2500.0 {
		t.Errorf("Fans()[0].Value = %v, want %v", fans[0].Value, 2500.0)
	}
}

func TestPlatformWrapperImplementsInterface(t *testing.T) {
	mock := &mockPlatform{name: "interface-test"}
	wrapper := WrapPlatform(mock)

	// Verify that wrapper implements monitor.PlatformInterface
	var _ monitor.PlatformInterface = wrapper
}

func TestPlatformWrapperPlatformGetter(t *testing.T) {
	mock := &mockPlatform{name: "getter-test"}
	wrapper := WrapPlatform(mock)

	if got := wrapper.Platform(); got != mock {
		t.Error("Platform() should return the underlying platform")
	}
}

func TestPlatformWrapperClose(t *testing.T) {
	mock := &mockPlatform{name: "close-test"}
	wrapper := WrapPlatform(mock)

	if err := wrapper.Close(); err != nil {
		t.Errorf("Close() should not return error, got: %v", err)
	}
}
