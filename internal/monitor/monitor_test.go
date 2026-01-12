package monitor

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewSystemData(t *testing.T) {
	sd := NewSystemData()
	if sd == nil {
		t.Fatal("NewSystemData() returned nil")
	}
}

func TestSystemDataConcurrency(t *testing.T) {
	sd := NewSystemData()

	// Test concurrent access to CPU stats
	done := make(chan bool)
	go func() {
		for i := 0; i < 100; i++ {
			sd.setCPU(CPUStats{UsagePercent: float64(i)})
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			_ = sd.GetCPU()
		}
		done <- true
	}()

	<-done
	<-done
}

func TestSystemDataGettersSetters(t *testing.T) {
	sd := NewSystemData()

	// Test CPU
	cpuStats := CPUStats{
		UsagePercent: 50.5,
		CPUCount:     4,
		ModelName:    "Test CPU",
	}
	sd.setCPU(cpuStats)
	got := sd.GetCPU()
	if got.UsagePercent != cpuStats.UsagePercent {
		t.Errorf("GetCPU().UsagePercent = %v, want %v", got.UsagePercent, cpuStats.UsagePercent)
	}

	// Test Memory
	memStats := MemoryStats{
		Total:        8192,
		Used:         4096,
		UsagePercent: 50.0,
	}
	sd.setMemory(memStats)
	gotMem := sd.GetMemory()
	if gotMem.Total != memStats.Total {
		t.Errorf("GetMemory().Total = %v, want %v", gotMem.Total, memStats.Total)
	}

	// Test Uptime
	uptimeStats := UptimeStats{
		Seconds:  12345.67,
		Duration: time.Duration(12345.67 * float64(time.Second)),
	}
	sd.setUptime(uptimeStats)
	gotUptime := sd.GetUptime()
	if gotUptime.Seconds != uptimeStats.Seconds {
		t.Errorf("GetUptime().Seconds = %v, want %v", gotUptime.Seconds, uptimeStats.Seconds)
	}

	// Test Network
	networkStats := NetworkStats{
		Interfaces: map[string]InterfaceStats{
			"eth0": {
				Name:    "eth0",
				RxBytes: 1000,
				TxBytes: 2000,
			},
		},
		TotalRxBytes: 1000,
		TotalTxBytes: 2000,
	}
	sd.setNetwork(networkStats)
	gotNetwork := sd.GetNetwork()
	if gotNetwork.TotalRxBytes != networkStats.TotalRxBytes {
		t.Errorf("GetNetwork().TotalRxBytes = %v, want %v", gotNetwork.TotalRxBytes, networkStats.TotalRxBytes)
	}
	if len(gotNetwork.Interfaces) != 1 {
		t.Errorf("GetNetwork().Interfaces length = %d, want 1", len(gotNetwork.Interfaces))
	}
	eth0, ok := gotNetwork.Interfaces["eth0"]
	if !ok {
		t.Error("GetNetwork().Interfaces should contain eth0")
	} else if eth0.RxBytes != 1000 {
		t.Errorf("GetNetwork().Interfaces[eth0].RxBytes = %v, want 1000", eth0.RxBytes)
	}

	// Test Filesystem
	filesystemStats := FilesystemStats{
		Mounts: map[string]MountStats{
			"/": {
				MountPoint:   "/",
				Device:       "/dev/sda1",
				FSType:       "ext4",
				Total:        1000000,
				Used:         500000,
				UsagePercent: 50.0,
			},
		},
	}
	sd.setFilesystem(filesystemStats)
	gotFilesystem := sd.GetFilesystem()
	if len(gotFilesystem.Mounts) != 1 {
		t.Errorf("GetFilesystem().Mounts length = %d, want 1", len(gotFilesystem.Mounts))
	}
	rootMount, ok := gotFilesystem.Mounts["/"]
	if !ok {
		t.Error("GetFilesystem().Mounts should contain /")
	} else if rootMount.Total != 1000000 {
		t.Errorf("GetFilesystem().Mounts[/].Total = %v, want 1000000", rootMount.Total)
	}

	// Test DiskIO
	diskIOStats := DiskIOStats{
		Disks: map[string]DiskStats{
			"sda": {
				Name:            "sda",
				ReadsCompleted:  1000,
				WritesCompleted: 500,
			},
		},
	}
	sd.setDiskIO(diskIOStats)
	gotDiskIO := sd.GetDiskIO()
	if len(gotDiskIO.Disks) != 1 {
		t.Errorf("GetDiskIO().Disks length = %d, want 1", len(gotDiskIO.Disks))
	}
	sda, ok := gotDiskIO.Disks["sda"]
	if !ok {
		t.Error("GetDiskIO().Disks should contain sda")
	} else if sda.ReadsCompleted != 1000 {
		t.Errorf("GetDiskIO().Disks[sda].ReadsCompleted = %v, want 1000", sda.ReadsCompleted)
	}
}

func TestNewSystemMonitor(t *testing.T) {
	sm := NewSystemMonitor(time.Second)
	if sm == nil {
		t.Fatal("NewSystemMonitor() returned nil")
	}
	if sm.interval != time.Second {
		t.Errorf("interval = %v, want %v", sm.interval, time.Second)
	}
	if sm.IsRunning() {
		t.Error("newly created monitor should not be running")
	}
}

func TestSystemMonitorStartStop(t *testing.T) {
	// Create a monitor with mock files
	tmpDir := t.TempDir()
	setupMockProcFiles(t, tmpDir)

	sm := createTestMonitor(tmpDir, 100*time.Millisecond)

	// Test double start
	if err := sm.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if !sm.IsRunning() {
		t.Error("monitor should be running after Start()")
	}

	if err := sm.Start(); err == nil {
		t.Error("Start() should return error when already running")
	}

	// Test stop
	sm.Stop()
	if sm.IsRunning() {
		t.Error("monitor should not be running after Stop()")
	}

	// Test double stop (should not panic)
	sm.Stop()
}

func TestSystemMonitorUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	setupMockProcFiles(t, tmpDir)

	sm := createTestMonitor(tmpDir, time.Second)

	err := sm.Update()
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify data was updated
	cpu := sm.CPU()
	if cpu.CPUCount != 2 {
		t.Errorf("CPU().CPUCount = %d, want 2", cpu.CPUCount)
	}

	mem := sm.Memory()
	if mem.Total == 0 {
		t.Error("Memory().Total should not be 0")
	}

	uptime := sm.Uptime()
	if uptime.Seconds == 0 {
		t.Error("Uptime().Seconds should not be 0")
	}

	network := sm.Network()
	if len(network.Interfaces) != 2 {
		t.Errorf("Network().Interfaces count = %d, want 2", len(network.Interfaces))
	}
	if network.TotalRxBytes != 6000 { // 1000 + 5000
		t.Errorf("Network().TotalRxBytes = %d, want 6000", network.TotalRxBytes)
	}

	// Test Filesystem stats
	filesystem := sm.Filesystem()
	// Note: statfs may fail on the mock mount path, so we just verify the method doesn't panic
	// In a real test environment with access to filesystem, we'd check the actual values
	_ = filesystem

	// Test DiskIO stats
	diskIO := sm.DiskIO()
	if len(diskIO.Disks) != 1 {
		t.Errorf("DiskIO().Disks count = %d, want 1", len(diskIO.Disks))
	}
	sda, ok := diskIO.Disks["sda"]
	if !ok {
		t.Error("DiskIO().Disks should contain sda")
	} else if sda.ReadsCompleted != 1000 {
		t.Errorf("DiskIO().Disks[sda].ReadsCompleted = %d, want 1000", sda.ReadsCompleted)
	}
}

func TestSystemMonitorData(t *testing.T) {
	tmpDir := t.TempDir()
	setupMockProcFiles(t, tmpDir)

	sm := createTestMonitor(tmpDir, time.Second)
	_ = sm.Update()

	data := sm.Data()
	// Verify we got a valid snapshot with non-zero CPU count
	if data.CPU.CPUCount == 0 {
		t.Error("Data() should return valid snapshot with CPU data")
	}
	// Verify network data is included in snapshot
	if len(data.Network.Interfaces) == 0 {
		t.Error("Data() should return valid snapshot with Network data")
	}
}

func TestSystemMonitorPeriodicUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	setupMockProcFiles(t, tmpDir)

	sm := createTestMonitor(tmpDir, 50*time.Millisecond)

	if err := sm.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer sm.Stop()

	// Wait for at least one update cycle
	time.Sleep(100 * time.Millisecond)

	// Verify data is available
	cpu := sm.CPU()
	if cpu.CPUCount == 0 {
		t.Error("CPU data should be available after update")
	}
}

// Helper functions for creating test monitor with mock files

func setupMockProcFiles(t *testing.T, tmpDir string) {
	t.Helper()

	// Create mock /proc/stat
	statContent := `cpu  100 10 50 500 20 5 3 2
cpu0 50 5 25 250 10 2 1 1
cpu1 50 5 25 250 10 3 2 1
intr 12345
`
	if err := os.WriteFile(filepath.Join(tmpDir, "stat"), []byte(statContent), 0644); err != nil {
		t.Fatalf("failed to write mock stat: %v", err)
	}

	// Create mock /proc/cpuinfo
	cpuinfoContent := `processor	: 0
model name	: Test CPU Model
cpu MHz		: 2400.123

processor	: 1
model name	: Test CPU Model
cpu MHz		: 2400.456
`
	if err := os.WriteFile(filepath.Join(tmpDir, "cpuinfo"), []byte(cpuinfoContent), 0644); err != nil {
		t.Fatalf("failed to write mock cpuinfo: %v", err)
	}

	// Create mock /proc/meminfo
	meminfoContent := `MemTotal:        8192000 kB
MemFree:         2048000 kB
MemAvailable:    4096000 kB
Buffers:          512000 kB
Cached:          1024000 kB
SwapTotal:       4096000 kB
SwapFree:        3072000 kB
`
	if err := os.WriteFile(filepath.Join(tmpDir, "meminfo"), []byte(meminfoContent), 0644); err != nil {
		t.Fatalf("failed to write mock meminfo: %v", err)
	}

	// Create mock /proc/uptime
	uptimeContent := "12345.67 23456.78\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "uptime"), []byte(uptimeContent), 0644); err != nil {
		t.Fatalf("failed to write mock uptime: %v", err)
	}

	// Create mock /proc/net/dev
	netDevContent := `Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
    lo: 1000   100    0    0    0     0          0         0  1000   100    0    0    0     0       0          0
  eth0: 5000   500    0    0    0     0          0         0  2000   200    0    0    0     0       0          0
`
	if err := os.WriteFile(filepath.Join(tmpDir, "net_dev"), []byte(netDevContent), 0644); err != nil {
		t.Fatalf("failed to write mock net_dev: %v", err)
	}

	// Create mock /proc/mounts
	mountsContent := `/dev/sda1 / ext4 rw,relatime 0 0
proc /proc proc rw,nosuid,nodev,noexec,relatime 0 0
`
	if err := os.WriteFile(filepath.Join(tmpDir, "mounts"), []byte(mountsContent), 0644); err != nil {
		t.Fatalf("failed to write mock mounts: %v", err)
	}

	// Create mock /proc/diskstats
	diskstatsContent := `   8       0 sda 1000 100 2000 50 500 50 1000 25 0 100 125
   8       1 sda1 500 50 1000 25 250 25 500 12 0 50 62
`
	if err := os.WriteFile(filepath.Join(tmpDir, "diskstats"), []byte(diskstatsContent), 0644); err != nil {
		t.Fatalf("failed to write mock diskstats: %v", err)
	}
}

func createTestMonitor(tmpDir string, interval time.Duration) *SystemMonitor {
	sm := NewSystemMonitor(interval)
	sm.cpuReader.procStatPath = filepath.Join(tmpDir, "stat")
	sm.cpuReader.procInfoPath = filepath.Join(tmpDir, "cpuinfo")
	sm.memReader.procMemInfoPath = filepath.Join(tmpDir, "meminfo")
	sm.uptimeReader.procUptimePath = filepath.Join(tmpDir, "uptime")
	sm.networkReader.procNetDevPath = filepath.Join(tmpDir, "net_dev")
	sm.filesystemReader.procMountsPath = filepath.Join(tmpDir, "mounts")
	sm.diskIOReader.procDiskstatsPath = filepath.Join(tmpDir, "diskstats")
	return sm
}
