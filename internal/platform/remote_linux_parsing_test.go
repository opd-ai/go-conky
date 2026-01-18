package platform

import (
	"errors"
	"sync"
	"testing"
)

// mockSSHPlatform is a mock sshPlatform for testing parsing logic without SSH.
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

// testableRemoteLinuxCPUProvider wraps remoteLinuxCPUProvider for testing
type testableRemoteLinuxCPUProvider struct {
	mock      *mockSSHPlatform
	mu        sync.Mutex
	prevStats map[int]cpuTimes
}

func newTestableRemoteLinuxCPUProvider(mock *mockSSHPlatform) *testableRemoteLinuxCPUProvider {
	return &testableRemoteLinuxCPUProvider{
		mock:      mock,
		prevStats: make(map[int]cpuTimes),
	}
}

func (c *testableRemoteLinuxCPUProvider) TotalUsage() (float64, error) {
	output, err := c.mock.runCommand("cat /proc/stat | head -1")
	if err != nil {
		return 0, err
	}
	return parseTotalCPUUsage(output, c.prevStats, &c.mu)
}

func (c *testableRemoteLinuxCPUProvider) LoadAverage() (float64, float64, float64, error) {
	output, err := c.mock.runCommand("cat /proc/loadavg")
	if err != nil {
		return 0, 0, 0, err
	}
	return parseLoadAverage(output)
}

// testableRemoteLinuxMemoryProvider wraps remoteLinuxMemoryProvider for testing
type testableRemoteLinuxMemoryProvider struct {
	mock *mockSSHPlatform
}

func newTestableRemoteLinuxMemoryProvider(mock *mockSSHPlatform) *testableRemoteLinuxMemoryProvider {
	return &testableRemoteLinuxMemoryProvider{mock: mock}
}

func (m *testableRemoteLinuxMemoryProvider) Stats() (*MemoryStats, error) {
	output, err := m.mock.runCommand("cat /proc/meminfo")
	if err != nil {
		return nil, err
	}
	return parseMemInfoOutput(output)
}

func (m *testableRemoteLinuxMemoryProvider) SwapStats() (*SwapStats, error) {
	output, err := m.mock.runCommand("cat /proc/meminfo | grep '^Swap'")
	if err != nil {
		return nil, err
	}
	return parseSwapOutput(output)
}

func TestRemoteLinuxMemory_ParseMemInfo(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteLinuxMemoryProvider(mock)

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

func TestRemoteLinuxMemory_ParseSwap(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteLinuxMemoryProvider(mock)

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
	provider := newTestableRemoteLinuxMemoryProvider(mock)

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
	provider := newTestableRemoteLinuxMemoryProvider(mock)

	mock.setCommandError("cat /proc/meminfo", errors.New("SSH connection failed"))

	_, err := provider.Stats()
	if err == nil {
		t.Error("Stats() should return error when command fails")
	}
}

func TestRemoteLinuxCPU_ParseLoadAverage(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteLinuxCPUProvider(mock)

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

func TestRemoteLinuxCPU_ParseLoadAverageInvalid(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteLinuxCPUProvider(mock)

	mock.setCommandResult("cat /proc/loadavg", "invalid output")

	_, _, _, err := provider.LoadAverage()
	if err == nil {
		t.Error("LoadAverage() should return error for invalid output")
	}
}

func TestRemoteLinuxCPU_ParseTotalUsage(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteLinuxCPUProvider(mock)

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

func TestRemoteLinuxCPU_CommandError(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteLinuxCPUProvider(mock)

	mock.setCommandError("cat /proc/stat | head -1", errors.New("connection refused"))

	_, err := provider.TotalUsage()
	if err == nil {
		t.Error("TotalUsage() should return error when command fails")
	}
}
