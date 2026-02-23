package monitor

import (
	"testing"
	"time"
)

// mockCPUProvider is a test implementation of CPUProviderInterface.
type mockCPUProvider struct {
	usage      []float64
	totalUsage float64
	frequency  []float64
	info       *PlatformCPUInfo
	loadAvg    [3]float64
	err        error
}

func (m *mockCPUProvider) Usage() ([]float64, error)       { return m.usage, m.err }
func (m *mockCPUProvider) TotalUsage() (float64, error)    { return m.totalUsage, m.err }
func (m *mockCPUProvider) Frequency() ([]float64, error)   { return m.frequency, m.err }
func (m *mockCPUProvider) Info() (*PlatformCPUInfo, error) { return m.info, m.err }
func (m *mockCPUProvider) LoadAverage() (float64, float64, float64, error) {
	return m.loadAvg[0], m.loadAvg[1], m.loadAvg[2], m.err
}

// mockMemoryProvider is a test implementation of MemoryProviderInterface.
type mockMemoryProvider struct {
	stats     *PlatformMemoryStats
	swapStats *PlatformSwapStats
	err       error
}

func (m *mockMemoryProvider) Stats() (*PlatformMemoryStats, error)   { return m.stats, m.err }
func (m *mockMemoryProvider) SwapStats() (*PlatformSwapStats, error) { return m.swapStats, m.err }

// mockNetworkProvider is a test implementation of NetworkProviderInterface.
type mockNetworkProvider struct {
	interfaces []string
	stats      map[string]*PlatformNetworkStats
	err        error
}

func (m *mockNetworkProvider) Interfaces() ([]string, error) { return m.interfaces, m.err }
func (m *mockNetworkProvider) Stats(name string) (*PlatformNetworkStats, error) {
	return m.stats[name], m.err
}

func (m *mockNetworkProvider) AllStats() (map[string]*PlatformNetworkStats, error) {
	return m.stats, m.err
}

// mockFilesystemProvider is a test implementation of FilesystemProviderInterface.
type mockFilesystemProvider struct {
	mounts []PlatformMountInfo
	stats  map[string]*PlatformFilesystemStats
	diskIO map[string]*PlatformDiskIOStats
	err    error
}

func (m *mockFilesystemProvider) Mounts() ([]PlatformMountInfo, error) { return m.mounts, m.err }
func (m *mockFilesystemProvider) Stats(mp string) (*PlatformFilesystemStats, error) {
	return m.stats[mp], m.err
}

func (m *mockFilesystemProvider) DiskIO(dev string) (*PlatformDiskIOStats, error) {
	return m.diskIO[dev], m.err
}

// mockBatteryProvider is a test implementation of BatteryProviderInterface.
type mockBatteryProvider struct {
	count int
	stats []*PlatformBatteryStats
	err   error
}

func (m *mockBatteryProvider) Count() int { return m.count }
func (m *mockBatteryProvider) Stats(idx int) (*PlatformBatteryStats, error) {
	if idx >= 0 && idx < len(m.stats) {
		return m.stats[idx], m.err
	}
	return nil, m.err
}

// mockSensorProvider is a test implementation of SensorProviderInterface.
type mockSensorProvider struct {
	temps []PlatformSensorReading
	fans  []PlatformSensorReading
	err   error
}

func (m *mockSensorProvider) Temperatures() ([]PlatformSensorReading, error) { return m.temps, m.err }
func (m *mockSensorProvider) Fans() ([]PlatformSensorReading, error)         { return m.fans, m.err }

// mockPlatform is a test implementation of PlatformInterface.
type mockPlatform struct {
	name       string
	cpu        CPUProviderInterface
	memory     MemoryProviderInterface
	network    NetworkProviderInterface
	filesystem FilesystemProviderInterface
	battery    BatteryProviderInterface
	sensors    SensorProviderInterface
}

func (m *mockPlatform) Name() string                            { return m.name }
func (m *mockPlatform) CPU() CPUProviderInterface               { return m.cpu }
func (m *mockPlatform) Memory() MemoryProviderInterface         { return m.memory }
func (m *mockPlatform) Network() NetworkProviderInterface       { return m.network }
func (m *mockPlatform) Filesystem() FilesystemProviderInterface { return m.filesystem }
func (m *mockPlatform) Battery() BatteryProviderInterface       { return m.battery }
func (m *mockPlatform) Sensors() SensorProviderInterface        { return m.sensors }

func TestPlatformAdapterReadCPUStats(t *testing.T) {
	plat := &mockPlatform{
		name: "test",
		cpu: &mockCPUProvider{
			usage:      []float64{10.0, 20.0, 30.0, 40.0},
			totalUsage: 25.0,
			frequency:  []float64{3500.0, 3600.0, 3500.0, 3600.0},
			info:       &PlatformCPUInfo{Model: "Test CPU", Vendor: "TestVendor", Cores: 4, Threads: 8},
		},
	}

	adapter := NewPlatformAdapter(plat)
	if adapter == nil {
		t.Fatal("NewPlatformAdapter returned nil")
	}

	stats, err := adapter.ReadCPUStats()
	if err != nil {
		t.Fatalf("ReadCPUStats failed: %v", err)
	}

	if stats.UsagePercent != 25.0 {
		t.Errorf("expected UsagePercent=25.0, got %v", stats.UsagePercent)
	}
	if len(stats.Cores) != 4 {
		t.Errorf("expected 4 cores, got %d", len(stats.Cores))
	}
	if stats.ModelName != "Test CPU" {
		t.Errorf("expected ModelName='Test CPU', got %q", stats.ModelName)
	}
	if stats.Frequency != 3500.0 {
		t.Errorf("expected Frequency=3500.0, got %v", stats.Frequency)
	}
}

func TestPlatformAdapterReadMemoryStats(t *testing.T) {
	plat := &mockPlatform{
		name: "test",
		memory: &mockMemoryProvider{
			stats: &PlatformMemoryStats{
				Total:       16 * 1024 * 1024 * 1024,
				Used:        8 * 1024 * 1024 * 1024,
				Free:        4 * 1024 * 1024 * 1024,
				Available:   6 * 1024 * 1024 * 1024,
				UsedPercent: 50.0,
			},
			swapStats: &PlatformSwapStats{
				Total:       8 * 1024 * 1024 * 1024,
				Used:        1 * 1024 * 1024 * 1024,
				Free:        7 * 1024 * 1024 * 1024,
				UsedPercent: 12.5,
			},
		},
	}

	adapter := NewPlatformAdapter(plat)
	stats, err := adapter.ReadMemoryStats()
	if err != nil {
		t.Fatalf("ReadMemoryStats failed: %v", err)
	}

	if stats.UsagePercent != 50.0 {
		t.Errorf("expected UsagePercent=50.0, got %v", stats.UsagePercent)
	}
	if stats.SwapPercent != 12.5 {
		t.Errorf("expected SwapPercent=12.5, got %v", stats.SwapPercent)
	}
}

func TestPlatformAdapterReadNetworkStats(t *testing.T) {
	plat := &mockPlatform{
		name: "test",
		network: &mockNetworkProvider{
			interfaces: []string{"eth0", "lo"},
			stats: map[string]*PlatformNetworkStats{
				"eth0": {BytesRecv: 1000, BytesSent: 2000, PacketsRecv: 10, PacketsSent: 20},
				"lo":   {BytesRecv: 500, BytesSent: 500, PacketsRecv: 5, PacketsSent: 5},
			},
		},
	}

	adapter := NewPlatformAdapter(plat)
	stats, err := adapter.ReadNetworkStats()
	if err != nil {
		t.Fatalf("ReadNetworkStats failed: %v", err)
	}

	if len(stats.Interfaces) != 2 {
		t.Errorf("expected 2 interfaces, got %d", len(stats.Interfaces))
	}
	if stats.TotalRxBytes != 1500 {
		t.Errorf("expected TotalRxBytes=1500, got %d", stats.TotalRxBytes)
	}
	if stats.TotalTxBytes != 2500 {
		t.Errorf("expected TotalTxBytes=2500, got %d", stats.TotalTxBytes)
	}
}

func TestPlatformAdapterReadFilesystemStats(t *testing.T) {
	plat := &mockPlatform{
		name: "test",
		filesystem: &mockFilesystemProvider{
			mounts: []PlatformMountInfo{
				{Device: "/dev/sda1", MountPoint: "/", FSType: "ext4"},
				{Device: "/dev/sda2", MountPoint: "/home", FSType: "ext4"},
			},
			stats: map[string]*PlatformFilesystemStats{
				"/":     {Total: 100 * 1024 * 1024 * 1024, Used: 50 * 1024 * 1024 * 1024, Free: 50 * 1024 * 1024 * 1024, UsedPercent: 50.0},
				"/home": {Total: 200 * 1024 * 1024 * 1024, Used: 100 * 1024 * 1024 * 1024, Free: 100 * 1024 * 1024 * 1024, UsedPercent: 50.0},
			},
		},
	}

	adapter := NewPlatformAdapter(plat)
	stats, err := adapter.ReadFilesystemStats()
	if err != nil {
		t.Fatalf("ReadFilesystemStats failed: %v", err)
	}

	if len(stats.Mounts) != 2 {
		t.Errorf("expected 2 mounts, got %d", len(stats.Mounts))
	}

	rootMount, ok := stats.Mounts["/"]
	if !ok {
		t.Fatal("expected root mount")
	}
	if rootMount.FSType != "ext4" {
		t.Errorf("expected FSType=ext4, got %q", rootMount.FSType)
	}
}

func TestPlatformAdapterReadBatteryStats(t *testing.T) {
	plat := &mockPlatform{
		name: "test",
		battery: &mockBatteryProvider{
			count: 1,
			stats: []*PlatformBatteryStats{
				{
					Percent:       75.0,
					TimeRemaining: 2 * time.Hour,
					Charging:      true,
					FullCapacity:  50000,
					Current:       37500,
					Voltage:       12.6,
				},
			},
		},
	}

	adapter := NewPlatformAdapter(plat)
	stats, err := adapter.ReadBatteryStats()
	if err != nil {
		t.Fatalf("ReadBatteryStats failed: %v", err)
	}

	if len(stats.Batteries) != 1 {
		t.Errorf("expected 1 battery, got %d", len(stats.Batteries))
	}
	if !stats.IsCharging {
		t.Error("expected IsCharging=true")
	}
	if stats.TotalCapacity != 75.0 {
		t.Errorf("expected TotalCapacity=75.0, got %v", stats.TotalCapacity)
	}
}

func TestPlatformAdapterReadSensorStats(t *testing.T) {
	plat := &mockPlatform{
		name: "test",
		sensors: &mockSensorProvider{
			temps: []PlatformSensorReading{
				{Name: "coretemp", Label: "Core 0", Value: 45.0, Critical: 100.0},
				{Name: "coretemp", Label: "Core 1", Value: 47.0, Critical: 100.0},
			},
		},
	}

	adapter := NewPlatformAdapter(plat)
	stats, err := adapter.ReadSensorStats()
	if err != nil {
		t.Fatalf("ReadSensorStats failed: %v", err)
	}

	if len(stats.TempSensors) != 2 {
		t.Errorf("expected 2 temp sensors, got %d", len(stats.TempSensors))
	}
	if stats.TempSensors[0].InputCelsius != 45.0 {
		t.Errorf("expected InputCelsius=45.0, got %v", stats.TempSensors[0].InputCelsius)
	}
}

func TestPlatformAdapterNilProviders(t *testing.T) {
	// Test with all nil providers
	plat := &mockPlatform{name: "test"}
	adapter := NewPlatformAdapter(plat)

	// These should not panic and return empty stats
	cpuStats, _ := adapter.ReadCPUStats()
	if cpuStats.CPUCount != 0 {
		t.Errorf("expected empty CPU stats")
	}

	memStats, _ := adapter.ReadMemoryStats()
	if memStats.Total != 0 {
		t.Errorf("expected empty memory stats")
	}

	netStats, _ := adapter.ReadNetworkStats()
	if len(netStats.Interfaces) != 0 {
		t.Errorf("expected empty network stats")
	}

	fsStats, _ := adapter.ReadFilesystemStats()
	if len(fsStats.Mounts) != 0 {
		t.Errorf("expected empty filesystem stats")
	}

	batStats, _ := adapter.ReadBatteryStats()
	if len(batStats.Batteries) != 0 {
		t.Errorf("expected empty battery stats")
	}

	sensorStats, _ := adapter.ReadSensorStats()
	if len(sensorStats.TempSensors) != 0 {
		t.Errorf("expected empty sensor stats")
	}
}

func TestNewPlatformAdapterNil(t *testing.T) {
	adapter := NewPlatformAdapter(nil)
	if adapter != nil {
		t.Error("expected nil adapter for nil platform")
	}
}

func TestNewSystemMonitorWithPlatform(t *testing.T) {
	plat := &mockPlatform{
		name: "test",
		cpu: &mockCPUProvider{
			usage:      []float64{25.0},
			totalUsage: 25.0,
		},
		memory: &mockMemoryProvider{
			stats: &PlatformMemoryStats{Total: 1024, Used: 512, UsedPercent: 50.0},
		},
	}

	sm := NewSystemMonitorWithPlatform(time.Second, plat)
	if sm == nil {
		t.Fatal("NewSystemMonitorWithPlatform returned nil")
	}

	if !sm.UsesPlatform() {
		t.Error("expected UsesPlatform=true")
	}

	if sm.Platform() == nil {
		t.Error("expected non-nil Platform()")
	}

	if sm.Platform().PlatformName() != "test" {
		t.Errorf("expected PlatformName='test', got %q", sm.Platform().PlatformName())
	}
}

func TestSystemMonitorUsesPlatformFalseForDefault(t *testing.T) {
	sm := NewSystemMonitor(time.Second)
	if sm.UsesPlatform() {
		t.Error("expected UsesPlatform=false for default monitor")
	}
	if sm.Platform() != nil {
		t.Error("expected Platform()=nil for default monitor")
	}
}
