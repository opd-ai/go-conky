package platform

import (
	"errors"
	"sync"
	"testing"
)

// mockSSHPlatform is a mock sshPlatform for testing parsing logic without SSH.
// It implements the commandRunner interface.
type mockSSHPlatform struct {
	commandResults map[string]string
	commandErrors  map[string]error
	mu             sync.RWMutex
}

func newMockSSHPlatform() *mockSSHPlatform {
	return &mockSSHPlatform{
		commandResults: make(map[string]string),
		commandErrors:  make(map[string]error),
	}
}

func (m *mockSSHPlatform) setCommandResult(cmd, result string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.commandResults[cmd] = result
}

func (m *mockSSHPlatform) setCommandError(cmd string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.commandErrors[cmd] = err
}

func (m *mockSSHPlatform) runCommand(cmd string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if err, ok := m.commandErrors[cmd]; ok {
		return "", err
	}
	if result, ok := m.commandResults[cmd]; ok {
		return result, nil
	}
	return "", errors.New("command not mocked: " + cmd)
}

func TestRemoteLinuxMemory_Stats(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteLinuxMemoryProviderWithRunner(mock)

	memInfoContent := `MemTotal:       16384000 kB
MemFree:         4096000 kB
MemAvailable:    8192000 kB
Buffers:         1024000 kB
Cached:          2048000 kB
SwapCached:       512000 kB
`
	mock.setCommandResult("cat /proc/meminfo", memInfoContent)

	stats, err := provider.Stats()
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}

	// All values should be in bytes (multiplied by 1024)
	expectedTotal := uint64(16384000 * 1024)
	expectedFree := uint64(4096000 * 1024)
	expectedAvailable := uint64(8192000 * 1024)
	expectedBuffers := uint64(1024000 * 1024)
	expectedCached := uint64(2048000 * 1024)

	if stats.Total != expectedTotal {
		t.Errorf("Total = %d, want %d", stats.Total, expectedTotal)
	}
	if stats.Free != expectedFree {
		t.Errorf("Free = %d, want %d", stats.Free, expectedFree)
	}
	if stats.Available != expectedAvailable {
		t.Errorf("Available = %d, want %d", stats.Available, expectedAvailable)
	}
	if stats.Buffers != expectedBuffers {
		t.Errorf("Buffers = %d, want %d", stats.Buffers, expectedBuffers)
	}
	if stats.Cached != expectedCached {
		t.Errorf("Cached = %d, want %d", stats.Cached, expectedCached)
	}

	// Used = Total - Free - Buffers - Cached
	expectedUsed := expectedTotal - expectedFree - expectedBuffers - expectedCached
	if stats.Used != expectedUsed {
		t.Errorf("Used = %d, want %d", stats.Used, expectedUsed)
	}

	// UsedPercent should be calculated correctly
	expectedPercent := float64(expectedUsed) / float64(expectedTotal) * 100
	if stats.UsedPercent < expectedPercent-0.1 || stats.UsedPercent > expectedPercent+0.1 {
		t.Errorf("UsedPercent = %f, want ~%f", stats.UsedPercent, expectedPercent)
	}
}

func TestRemoteLinuxMemory_SwapStats(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteLinuxMemoryProviderWithRunner(mock)

	swapContent := `SwapTotal:       8192000 kB
SwapFree:        4096000 kB
`
	mock.setCommandResult("cat /proc/meminfo | grep '^Swap'", swapContent)

	stats, err := provider.SwapStats()
	if err != nil {
		t.Fatalf("SwapStats() error = %v", err)
	}

	expectedTotal := uint64(8192000 * 1024)
	expectedFree := uint64(4096000 * 1024)
	expectedUsed := expectedTotal - expectedFree

	if stats.Total != expectedTotal {
		t.Errorf("Total = %d, want %d", stats.Total, expectedTotal)
	}
	if stats.Free != expectedFree {
		t.Errorf("Free = %d, want %d", stats.Free, expectedFree)
	}
	if stats.Used != expectedUsed {
		t.Errorf("Used = %d, want %d", stats.Used, expectedUsed)
	}

	expectedPercent := float64(expectedUsed) / float64(expectedTotal) * 100
	if stats.UsedPercent < expectedPercent-0.1 || stats.UsedPercent > expectedPercent+0.1 {
		t.Errorf("UsedPercent = %f, want ~%f", stats.UsedPercent, expectedPercent)
	}
}

func TestRemoteLinuxMemory_SwapNoSwap(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteLinuxMemoryProviderWithRunner(mock)

	// No swap configured
	swapContent := `SwapTotal:              0 kB
SwapFree:               0 kB
`
	mock.setCommandResult("cat /proc/meminfo | grep '^Swap'", swapContent)

	stats, err := provider.SwapStats()
	if err != nil {
		t.Fatalf("SwapStats() error = %v", err)
	}

	if stats.Total != 0 {
		t.Errorf("Total = %d, want 0", stats.Total)
	}
	if stats.UsedPercent != 0 {
		t.Errorf("UsedPercent = %f, want 0", stats.UsedPercent)
	}
}

func TestRemoteLinuxMemory_CommandError(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteLinuxMemoryProviderWithRunner(mock)

	mock.setCommandError("cat /proc/meminfo", errors.New("SSH connection failed"))

	_, err := provider.Stats()
	if err == nil {
		t.Error("Stats() should return error when command fails")
	}
}

func TestRemoteLinuxCPU_LoadAverage(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteLinuxCPUProviderWithRunner(mock)

	mock.setCommandResult("cat /proc/loadavg", "0.15 0.25 0.35 1/234 5678")

	load1, load5, load15, err := provider.LoadAverage()
	if err != nil {
		t.Fatalf("LoadAverage() error = %v", err)
	}

	if load1 != 0.15 {
		t.Errorf("load1 = %f, want 0.15", load1)
	}
	if load5 != 0.25 {
		t.Errorf("load5 = %f, want 0.25", load5)
	}
	if load15 != 0.35 {
		t.Errorf("load15 = %f, want 0.35", load15)
	}
}

func TestRemoteLinuxCPU_LoadAverageInvalid(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteLinuxCPUProviderWithRunner(mock)

	mock.setCommandResult("cat /proc/loadavg", "invalid output")

	_, _, _, err := provider.LoadAverage()
	if err == nil {
		t.Error("LoadAverage() should return error for invalid output")
	}
}

func TestRemoteLinuxCPU_TotalUsage(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteLinuxCPUProviderWithRunner(mock)

	// First call establishes baseline (returns 0)
	mock.setCommandResult("cat /proc/stat | head -1", "cpu  100 50 150 500 10 5 3 2 0 0")

	usage1, err := provider.TotalUsage()
	if err != nil {
		t.Fatalf("TotalUsage() first call error = %v", err)
	}
	if usage1 != 0 {
		t.Logf("First call returned %f (expected 0 for baseline)", usage1)
	}

	// Second call with increased values shows usage
	mock.setCommandResult("cat /proc/stat | head -1", "cpu  200 100 300 600 20 10 6 4 0 0")

	usage2, err := provider.TotalUsage()
	if err != nil {
		t.Fatalf("TotalUsage() second call error = %v", err)
	}

	// Usage should be non-zero now
	if usage2 < 0 || usage2 > 100 {
		t.Errorf("Usage = %f, should be between 0-100", usage2)
	}
}

func TestRemoteLinuxCPU_TotalUsageCommandError(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteLinuxCPUProviderWithRunner(mock)

	mock.setCommandError("cat /proc/stat | head -1", errors.New("connection refused"))

	_, err := provider.TotalUsage()
	if err == nil {
		t.Error("TotalUsage() should return error when command fails")
	}
}

func TestRemoteLinuxCPU_Info(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteLinuxCPUProviderWithRunner(mock)

	cpuInfoContent := `processor	: 0
vendor_id	: GenuineIntel
model name	: Intel(R) Core(TM) i7-8700 CPU @ 3.20GHz
cpu cores	: 6
cache size	: 12288 KB

processor	: 1
vendor_id	: GenuineIntel
model name	: Intel(R) Core(TM) i7-8700 CPU @ 3.20GHz
cpu cores	: 6
cache size	: 12288 KB
`
	mock.setCommandResult("cat /proc/cpuinfo", cpuInfoContent)

	info, err := provider.Info()
	if err != nil {
		t.Fatalf("Info() error = %v", err)
	}

	if info.Model != "Intel(R) Core(TM) i7-8700 CPU @ 3.20GHz" {
		t.Errorf("Model = %s, want Intel(R) Core(TM) i7-8700 CPU @ 3.20GHz", info.Model)
	}
	if info.Vendor != "GenuineIntel" {
		t.Errorf("Vendor = %s, want GenuineIntel", info.Vendor)
	}
	if info.Cores != 6 {
		t.Errorf("Cores = %d, want 6", info.Cores)
	}
	if info.Threads != 2 {
		t.Errorf("Threads = %d, want 2", info.Threads)
	}
	// 12288 KB = 12288 * 1024 bytes
	expectedCache := int64(12288 * 1024)
	if info.CacheSize != expectedCache {
		t.Errorf("CacheSize = %d, want %d", info.CacheSize, expectedCache)
	}
}

func TestRemoteLinuxCPU_Frequency(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteLinuxCPUProviderWithRunner(mock)

	mock.setCommandResult("cat /proc/cpuinfo | grep 'cpu MHz'", `cpu MHz		: 3200.000
cpu MHz		: 3100.500
cpu MHz		: 3300.250
`)

	freqs, err := provider.Frequency()
	if err != nil {
		t.Fatalf("Frequency() error = %v", err)
	}

	if len(freqs) != 3 {
		t.Fatalf("len(Frequency()) = %d, want 3", len(freqs))
	}

	if freqs[0] != 3200.0 {
		t.Errorf("freqs[0] = %f, want 3200.0", freqs[0])
	}
	if freqs[1] != 3100.5 {
		t.Errorf("freqs[1] = %f, want 3100.5", freqs[1])
	}
	if freqs[2] != 3300.25 {
		t.Errorf("freqs[2] = %f, want 3300.25", freqs[2])
	}
}

func TestRemoteLinuxCPU_FrequencyEmpty(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteLinuxCPUProviderWithRunner(mock)

	mock.setCommandResult("cat /proc/cpuinfo | grep 'cpu MHz'", "")

	_, err := provider.Frequency()
	if err == nil {
		t.Error("Frequency() should return error when no frequency info available")
	}
}

func TestRemoteLinuxCPU_Usage(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteLinuxCPUProviderWithRunner(mock)

	// First call establishes baseline
	mock.setCommandResult("cat /proc/stat | grep '^cpu[0-9]'", `cpu0 100 50 150 500 10 5 3 2
cpu1 200 100 300 600 20 10 6 4`)

	usage1, err := provider.Usage()
	if err != nil {
		t.Fatalf("Usage() first call error = %v", err)
	}
	if len(usage1) != 2 {
		t.Fatalf("len(Usage()) = %d, want 2", len(usage1))
	}

	// Second call with increased values
	mock.setCommandResult("cat /proc/stat | grep '^cpu[0-9]'", `cpu0 200 100 300 600 20 10 6 4
cpu1 400 200 600 800 40 20 12 8`)

	usage2, err := provider.Usage()
	if err != nil {
		t.Fatalf("Usage() second call error = %v", err)
	}
	if len(usage2) != 2 {
		t.Fatalf("len(Usage()) = %d, want 2", len(usage2))
	}

	// Usage should be between 0-100
	for i, u := range usage2 {
		if u < 0 || u > 100 {
			t.Errorf("usage[%d] = %f, should be between 0-100", i, u)
		}
	}
}

// Network provider tests

func TestRemoteLinuxNetwork_Interfaces(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteLinuxNetworkProviderWithRunner(mock)

	mock.setCommandResult("cat /proc/net/dev | tail -n +3", `    lo:  123456  100    0    0    0     0          0         0   123456  100    0    0    0     0       0          0
  eth0: 9876543210  654321    0    0    0     0          0         0 1234567890  543210    0    0    0     0       0          0
docker0:       0       0    0    0    0     0          0         0        0       0    0    0    0     0       0          0`)

	interfaces, err := provider.Interfaces()
	if err != nil {
		t.Fatalf("Interfaces() error = %v", err)
	}

	if len(interfaces) != 3 {
		t.Fatalf("len(Interfaces()) = %d, want 3", len(interfaces))
	}

	expected := []string{"lo", "eth0", "docker0"}
	for i, iface := range interfaces {
		if iface != expected[i] {
			t.Errorf("interfaces[%d] = %s, want %s", i, iface, expected[i])
		}
	}
}

func TestRemoteLinuxNetwork_Stats(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteLinuxNetworkProviderWithRunner(mock)

	mock.setCommandResult("cat /proc/net/dev | tail -n +3", `    lo:  123456  100    0    0    0     0          0         0   123456  100    0    0    0     0       0          0
  eth0: 9876543210  654321    5    3    0     0          0         0 1234567890  543210    2    1    0     0       0          0`)

	stats, err := provider.Stats("eth0")
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}

	if stats.BytesRecv != 9876543210 {
		t.Errorf("BytesRecv = %d, want 9876543210", stats.BytesRecv)
	}
	if stats.PacketsRecv != 654321 {
		t.Errorf("PacketsRecv = %d, want 654321", stats.PacketsRecv)
	}
	if stats.ErrorsIn != 5 {
		t.Errorf("ErrorsIn = %d, want 5", stats.ErrorsIn)
	}
	if stats.DropIn != 3 {
		t.Errorf("DropIn = %d, want 3", stats.DropIn)
	}
	if stats.BytesSent != 1234567890 {
		t.Errorf("BytesSent = %d, want 1234567890", stats.BytesSent)
	}
	if stats.PacketsSent != 543210 {
		t.Errorf("PacketsSent = %d, want 543210", stats.PacketsSent)
	}
	if stats.ErrorsOut != 2 {
		t.Errorf("ErrorsOut = %d, want 2", stats.ErrorsOut)
	}
	if stats.DropOut != 1 {
		t.Errorf("DropOut = %d, want 1", stats.DropOut)
	}
}

func TestRemoteLinuxNetwork_StatsNotFound(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteLinuxNetworkProviderWithRunner(mock)

	mock.setCommandResult("cat /proc/net/dev | tail -n +3", `    lo:  123456  100    0    0    0     0          0         0   123456  100    0    0    0     0       0          0`)

	_, err := provider.Stats("nonexistent")
	if err == nil {
		t.Error("Stats() should return error for non-existent interface")
	}
}

func TestRemoteLinuxNetwork_AllStats(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteLinuxNetworkProviderWithRunner(mock)

	mock.setCommandResult("cat /proc/net/dev | tail -n +3", `    lo:  123456  100    0    0    0     0          0         0   654321  200    0    0    0     0       0          0
  eth0: 9876543210  654321    0    0    0     0          0         0 1234567890  543210    0    0    0     0       0          0`)

	allStats, err := provider.AllStats()
	if err != nil {
		t.Fatalf("AllStats() error = %v", err)
	}

	if len(allStats) != 2 {
		t.Fatalf("len(AllStats()) = %d, want 2", len(allStats))
	}

	if _, ok := allStats["lo"]; !ok {
		t.Error("AllStats() should include 'lo' interface")
	}
	if _, ok := allStats["eth0"]; !ok {
		t.Error("AllStats() should include 'eth0' interface")
	}
}

// Filesystem provider tests

func TestRemoteLinuxFilesystem_Mounts(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteLinuxFilesystemProviderWithRunner(mock)

	mock.setCommandResult("cat /proc/mounts", `/dev/sda1 / ext4 rw,relatime 0 0
tmpfs /tmp tmpfs rw,nosuid,nodev 0 0
/dev/sdb1 /home xfs rw,relatime,inode64 0 0`)

	mounts, err := provider.Mounts()
	if err != nil {
		t.Fatalf("Mounts() error = %v", err)
	}

	if len(mounts) != 3 {
		t.Fatalf("len(Mounts()) = %d, want 3", len(mounts))
	}

	// Check first mount
	if mounts[0].Device != "/dev/sda1" {
		t.Errorf("mounts[0].Device = %s, want /dev/sda1", mounts[0].Device)
	}
	if mounts[0].MountPoint != "/" {
		t.Errorf("mounts[0].MountPoint = %s, want /", mounts[0].MountPoint)
	}
	if mounts[0].FSType != "ext4" {
		t.Errorf("mounts[0].FSType = %s, want ext4", mounts[0].FSType)
	}
	if len(mounts[0].Options) < 1 || mounts[0].Options[0] != "rw" {
		t.Errorf("mounts[0].Options first element should be 'rw'")
	}
}

func TestRemoteLinuxFilesystem_Stats(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteLinuxFilesystemProviderWithRunner(mock)

	// df -B1 output
	mock.setCommandResult("df -B1 '/' | tail -n 1", "/dev/sda1      107374182400  53687091200  53687091200  50% /")
	// df -i output
	mock.setCommandResult("df -i '/' | tail -n 1", "/dev/sda1       6553600  1638400  4915200  26% /")

	stats, err := provider.Stats("/")
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}

	if stats.Total != 107374182400 {
		t.Errorf("Total = %d, want 107374182400", stats.Total)
	}
	if stats.Used != 53687091200 {
		t.Errorf("Used = %d, want 53687091200", stats.Used)
	}
	if stats.Free != 53687091200 {
		t.Errorf("Free = %d, want 53687091200", stats.Free)
	}
	if stats.UsedPercent < 49.9 || stats.UsedPercent > 50.1 {
		t.Errorf("UsedPercent = %f, want ~50", stats.UsedPercent)
	}
	if stats.InodesTotal != 6553600 {
		t.Errorf("InodesTotal = %d, want 6553600", stats.InodesTotal)
	}
	if stats.InodesUsed != 1638400 {
		t.Errorf("InodesUsed = %d, want 1638400", stats.InodesUsed)
	}
	if stats.InodesFree != 4915200 {
		t.Errorf("InodesFree = %d, want 4915200", stats.InodesFree)
	}
}

func TestRemoteLinuxFilesystem_DiskIO(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteLinuxFilesystemProviderWithRunner(mock)

	// /proc/diskstats format:
	// major minor name reads reads_merged sectors_read time_reading writes writes_merged sectors_written time_writing io_ops io_time weighted_time
	mock.setCommandResult("cat /proc/diskstats", `   8       0 sda 123456 0 246912 100000 654321 0 1308642 200000 0 50000 300000
   8       1 sda1 100000 0 200000 80000 500000 0 1000000 150000 0 40000 230000
   8      16 sdb 50000 0 100000 40000 100000 0 200000 50000 0 20000 90000`)

	stats, err := provider.DiskIO("sda")
	if err != nil {
		t.Fatalf("DiskIO() error = %v", err)
	}

	if stats.ReadCount != 123456 {
		t.Errorf("ReadCount = %d, want 123456", stats.ReadCount)
	}
	// sectors_read (246912) * 512 = 126418944
	expectedReadBytes := uint64(246912 * 512)
	if stats.ReadBytes != expectedReadBytes {
		t.Errorf("ReadBytes = %d, want %d", stats.ReadBytes, expectedReadBytes)
	}
	if stats.WriteCount != 654321 {
		t.Errorf("WriteCount = %d, want 654321", stats.WriteCount)
	}
	// sectors_written (1308642) * 512 = 670024704
	expectedWriteBytes := uint64(1308642 * 512)
	if stats.WriteBytes != expectedWriteBytes {
		t.Errorf("WriteBytes = %d, want %d", stats.WriteBytes, expectedWriteBytes)
	}
}

func TestRemoteLinuxFilesystem_DiskIONotFound(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteLinuxFilesystemProviderWithRunner(mock)

	mock.setCommandResult("cat /proc/diskstats", `   8       0 sda 123456 0 246912 100000 654321 0 1308642 200000 0 50000 300000`)

	_, err := provider.DiskIO("nonexistent")
	if err == nil {
		t.Error("DiskIO() should return error for non-existent device")
	}
}
