package platform

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLinuxMemoryProvider_Stats(t *testing.T) {
	// Create a temporary /proc/meminfo file
	tmpDir := t.TempDir()
	meminfoPath := filepath.Join(tmpDir, "meminfo")

	content := `MemTotal:       16384000 kB
MemFree:         8192000 kB
MemAvailable:   10240000 kB
Buffers:          512000 kB
Cached:          2048000 kB
SwapCached:            0 kB
Active:          4096000 kB
Inactive:        2048000 kB
`
	if err := os.WriteFile(meminfoPath, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write meminfo file: %v", err)
	}

	provider := &linuxMemoryProvider{
		procMemInfoPath: meminfoPath,
	}

	stats, err := provider.Stats()
	if err != nil {
		t.Fatalf("Stats() failed: %v", err)
	}

	// Expected values in bytes (kB * 1024)
	expectedTotal := uint64(16384000 * 1024)
	expectedFree := uint64(8192000 * 1024)
	expectedAvailable := uint64(10240000 * 1024)
	expectedBuffers := uint64(512000 * 1024)
	expectedCached := uint64(2048000 * 1024)

	if stats.Total != expectedTotal {
		t.Errorf("Total = %v, want %v", stats.Total, expectedTotal)
	}
	if stats.Free != expectedFree {
		t.Errorf("Free = %v, want %v", stats.Free, expectedFree)
	}
	if stats.Available != expectedAvailable {
		t.Errorf("Available = %v, want %v", stats.Available, expectedAvailable)
	}
	if stats.Buffers != expectedBuffers {
		t.Errorf("Buffers = %v, want %v", stats.Buffers, expectedBuffers)
	}
	if stats.Cached != expectedCached {
		t.Errorf("Cached = %v, want %v", stats.Cached, expectedCached)
	}

	// Check that Used is calculated correctly
	// Used = Total - Free - Buffers - Cached
	expectedUsed := expectedTotal - expectedFree - expectedBuffers - expectedCached
	if stats.Used != expectedUsed {
		t.Errorf("Used = %v, want %v", stats.Used, expectedUsed)
	}

	// Check usage percentage
	expectedUsagePercent := float64(stats.Used) / float64(stats.Total) * 100.0
	if stats.UsedPercent < expectedUsagePercent-0.1 || stats.UsedPercent > expectedUsagePercent+0.1 {
		t.Errorf("UsedPercent = %v, want ~%v", stats.UsedPercent, expectedUsagePercent)
	}
}

func TestLinuxMemoryProvider_SwapStats(t *testing.T) {
	// Create a temporary /proc/meminfo file
	tmpDir := t.TempDir()
	meminfoPath := filepath.Join(tmpDir, "meminfo")

	content := `MemTotal:       16384000 kB
MemFree:         8192000 kB
SwapTotal:       4096000 kB
SwapFree:        3072000 kB
`
	if err := os.WriteFile(meminfoPath, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write meminfo file: %v", err)
	}

	provider := &linuxMemoryProvider{
		procMemInfoPath: meminfoPath,
	}

	stats, err := provider.SwapStats()
	if err != nil {
		t.Fatalf("SwapStats() failed: %v", err)
	}

	// Expected values in bytes (kB * 1024)
	expectedTotal := uint64(4096000 * 1024)
	expectedFree := uint64(3072000 * 1024)
	expectedUsed := expectedTotal - expectedFree

	if stats.Total != expectedTotal {
		t.Errorf("Total = %v, want %v", stats.Total, expectedTotal)
	}
	if stats.Free != expectedFree {
		t.Errorf("Free = %v, want %v", stats.Free, expectedFree)
	}
	if stats.Used != expectedUsed {
		t.Errorf("Used = %v, want %v", stats.Used, expectedUsed)
	}

	// Check usage percentage
	expectedUsagePercent := float64(expectedUsed) / float64(expectedTotal) * 100.0
	if stats.UsedPercent < expectedUsagePercent-0.1 || stats.UsedPercent > expectedUsagePercent+0.1 {
		t.Errorf("UsedPercent = %v, want ~%v", stats.UsedPercent, expectedUsagePercent)
	}
}

func TestLinuxMemoryProvider_Stats_NoSwap(t *testing.T) {
	// Create a temporary /proc/meminfo file without swap
	tmpDir := t.TempDir()
	meminfoPath := filepath.Join(tmpDir, "meminfo")

	content := `MemTotal:       16384000 kB
MemFree:         8192000 kB
MemAvailable:   10240000 kB
Buffers:          512000 kB
Cached:          2048000 kB
SwapTotal:              0 kB
SwapFree:               0 kB
`
	if err := os.WriteFile(meminfoPath, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write meminfo file: %v", err)
	}

	provider := &linuxMemoryProvider{
		procMemInfoPath: meminfoPath,
	}

	swapStats, err := provider.SwapStats()
	if err != nil {
		t.Fatalf("SwapStats() failed: %v", err)
	}

	if swapStats.Total != 0 {
		t.Errorf("SwapStats.Total = %v, want 0", swapStats.Total)
	}
	if swapStats.Used != 0 {
		t.Errorf("SwapStats.Used = %v, want 0", swapStats.Used)
	}
	if swapStats.UsedPercent != 0 {
		t.Errorf("SwapStats.UsedPercent = %v, want 0", swapStats.UsedPercent)
	}
}
