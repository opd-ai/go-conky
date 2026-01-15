//go:build android
// +build android

package platform

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAndroidMemoryProvider_Stats(t *testing.T) {
	tmpDir := t.TempDir()
	memInfoPath := filepath.Join(tmpDir, "meminfo")

	content := `MemTotal:        8000000 kB
MemFree:         2000000 kB
MemAvailable:    5000000 kB
Buffers:          500000 kB
Cached:          2000000 kB
SwapTotal:       2000000 kB
SwapFree:        1500000 kB
`
	if err := os.WriteFile(memInfoPath, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write meminfo file: %v", err)
	}

	provider := &androidMemoryProvider{
		procMemInfoPath: memInfoPath,
	}

	stats, err := provider.Stats()
	if err != nil {
		t.Fatalf("Stats() failed: %v", err)
	}

	expectedTotal := uint64(8000000) * 1024
	if stats.Total != expectedTotal {
		t.Errorf("Total = %v, want %v", stats.Total, expectedTotal)
	}

	expectedFree := uint64(2000000) * 1024
	if stats.Free != expectedFree {
		t.Errorf("Free = %v, want %v", stats.Free, expectedFree)
	}

	expectedAvailable := uint64(5000000) * 1024
	if stats.Available != expectedAvailable {
		t.Errorf("Available = %v, want %v", stats.Available, expectedAvailable)
	}

	expectedBuffers := uint64(500000) * 1024
	if stats.Buffers != expectedBuffers {
		t.Errorf("Buffers = %v, want %v", stats.Buffers, expectedBuffers)
	}

	expectedCached := uint64(2000000) * 1024
	if stats.Cached != expectedCached {
		t.Errorf("Cached = %v, want %v", stats.Cached, expectedCached)
	}

	// Used = Total - Free - Buffers - Cached
	expectedUsed := expectedTotal - expectedFree - expectedBuffers - expectedCached
	if stats.Used != expectedUsed {
		t.Errorf("Used = %v, want %v", stats.Used, expectedUsed)
	}
}

func TestAndroidMemoryProvider_SwapStats(t *testing.T) {
	tmpDir := t.TempDir()
	memInfoPath := filepath.Join(tmpDir, "meminfo")

	content := `MemTotal:        8000000 kB
MemFree:         2000000 kB
SwapTotal:       2000000 kB
SwapFree:        1500000 kB
`
	if err := os.WriteFile(memInfoPath, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write meminfo file: %v", err)
	}

	provider := &androidMemoryProvider{
		procMemInfoPath: memInfoPath,
	}

	stats, err := provider.SwapStats()
	if err != nil {
		t.Fatalf("SwapStats() failed: %v", err)
	}

	expectedTotal := uint64(2000000) * 1024
	if stats.Total != expectedTotal {
		t.Errorf("Total = %v, want %v", stats.Total, expectedTotal)
	}

	expectedFree := uint64(1500000) * 1024
	if stats.Free != expectedFree {
		t.Errorf("Free = %v, want %v", stats.Free, expectedFree)
	}

	expectedUsed := expectedTotal - expectedFree
	if stats.Used != expectedUsed {
		t.Errorf("Used = %v, want %v", stats.Used, expectedUsed)
	}

	expectedPercent := float64(expectedUsed) / float64(expectedTotal) * 100.0
	if stats.UsedPercent < expectedPercent-0.1 || stats.UsedPercent > expectedPercent+0.1 {
		t.Errorf("UsedPercent = %v, want ~%v", stats.UsedPercent, expectedPercent)
	}
}

func TestAndroidMemoryProvider_EmptySwap(t *testing.T) {
	tmpDir := t.TempDir()
	memInfoPath := filepath.Join(tmpDir, "meminfo")

	content := `MemTotal:        8000000 kB
MemFree:         2000000 kB
SwapTotal:             0 kB
SwapFree:              0 kB
`
	if err := os.WriteFile(memInfoPath, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write meminfo file: %v", err)
	}

	provider := &androidMemoryProvider{
		procMemInfoPath: memInfoPath,
	}

	stats, err := provider.SwapStats()
	if err != nil {
		t.Fatalf("SwapStats() failed: %v", err)
	}

	if stats.Total != 0 {
		t.Errorf("Total = %v, want 0", stats.Total)
	}
	if stats.Used != 0 {
		t.Errorf("Used = %v, want 0", stats.Used)
	}
	if stats.UsedPercent != 0 {
		t.Errorf("UsedPercent = %v, want 0", stats.UsedPercent)
	}
}
