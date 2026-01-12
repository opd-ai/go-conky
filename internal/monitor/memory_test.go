package monitor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMemoryReaderWithMockFile(t *testing.T) {
	tmpDir := t.TempDir()

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

	reader := &memoryReader{
		procMemInfoPath: filepath.Join(tmpDir, "meminfo"),
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	// Verify values (converted from kB to bytes)
	expectedTotal := uint64(8192000 * 1024)
	if stats.Total != expectedTotal {
		t.Errorf("Total = %d, want %d", stats.Total, expectedTotal)
	}

	expectedFree := uint64(2048000 * 1024)
	if stats.Free != expectedFree {
		t.Errorf("Free = %d, want %d", stats.Free, expectedFree)
	}

	expectedAvailable := uint64(4096000 * 1024)
	if stats.Available != expectedAvailable {
		t.Errorf("Available = %d, want %d", stats.Available, expectedAvailable)
	}

	expectedBuffers := uint64(512000 * 1024)
	if stats.Buffers != expectedBuffers {
		t.Errorf("Buffers = %d, want %d", stats.Buffers, expectedBuffers)
	}

	expectedCached := uint64(1024000 * 1024)
	if stats.Cached != expectedCached {
		t.Errorf("Cached = %d, want %d", stats.Cached, expectedCached)
	}

	expectedSwapTotal := uint64(4096000 * 1024)
	if stats.SwapTotal != expectedSwapTotal {
		t.Errorf("SwapTotal = %d, want %d", stats.SwapTotal, expectedSwapTotal)
	}

	expectedSwapFree := uint64(3072000 * 1024)
	if stats.SwapFree != expectedSwapFree {
		t.Errorf("SwapFree = %d, want %d", stats.SwapFree, expectedSwapFree)
	}

	// Verify calculated values
	// Used = Total - Free - Buffers - Cached
	// = 8192000 - 2048000 - 512000 - 1024000 = 4608000 kB
	expectedUsed := uint64(4608000 * 1024)
	if stats.Used != expectedUsed {
		t.Errorf("Used = %d, want %d", stats.Used, expectedUsed)
	}

	// SwapUsed = SwapTotal - SwapFree
	// = 4096000 - 3072000 = 1024000 kB
	expectedSwapUsed := uint64(1024000 * 1024)
	if stats.SwapUsed != expectedSwapUsed {
		t.Errorf("SwapUsed = %d, want %d", stats.SwapUsed, expectedSwapUsed)
	}

	// UsagePercent = Used / Total * 100
	// = 4608000 / 8192000 * 100 = 56.25%
	expectedUsagePercent := 56.25
	if stats.UsagePercent != expectedUsagePercent {
		t.Errorf("UsagePercent = %v, want %v", stats.UsagePercent, expectedUsagePercent)
	}

	// SwapPercent = SwapUsed / SwapTotal * 100
	// = 1024000 / 4096000 * 100 = 25%
	expectedSwapPercent := 25.0
	if stats.SwapPercent != expectedSwapPercent {
		t.Errorf("SwapPercent = %v, want %v", stats.SwapPercent, expectedSwapPercent)
	}
}

func TestMemoryReaderNoSwap(t *testing.T) {
	tmpDir := t.TempDir()

	// Create mock /proc/meminfo without swap
	meminfoContent := `MemTotal:        8192000 kB
MemFree:         2048000 kB
MemAvailable:    4096000 kB
Buffers:          512000 kB
Cached:          1024000 kB
SwapTotal:             0 kB
SwapFree:              0 kB
`
	if err := os.WriteFile(filepath.Join(tmpDir, "meminfo"), []byte(meminfoContent), 0644); err != nil {
		t.Fatalf("failed to write mock meminfo: %v", err)
	}

	reader := &memoryReader{
		procMemInfoPath: filepath.Join(tmpDir, "meminfo"),
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	if stats.SwapTotal != 0 {
		t.Errorf("SwapTotal = %d, want 0", stats.SwapTotal)
	}
	if stats.SwapPercent != 0.0 {
		t.Errorf("SwapPercent = %v, want 0.0", stats.SwapPercent)
	}
}

func TestMemoryReaderSwapUnderflowProtection(t *testing.T) {
	tmpDir := t.TempDir()

	// Create mock /proc/meminfo with SwapFree > SwapTotal (inconsistent data)
	meminfoContent := `MemTotal:        8192000 kB
MemFree:         2048000 kB
MemAvailable:    4096000 kB
Buffers:          512000 kB
Cached:          1024000 kB
SwapTotal:       1000000 kB
SwapFree:        2000000 kB
`
	if err := os.WriteFile(filepath.Join(tmpDir, "meminfo"), []byte(meminfoContent), 0644); err != nil {
		t.Fatalf("failed to write mock meminfo: %v", err)
	}

	reader := &memoryReader{
		procMemInfoPath: filepath.Join(tmpDir, "meminfo"),
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	// SwapUsed should be 0 due to underflow protection
	if stats.SwapUsed != 0 {
		t.Errorf("SwapUsed = %d, want 0 (underflow protection)", stats.SwapUsed)
	}
}

func TestMemoryReaderMissingFile(t *testing.T) {
	reader := &memoryReader{
		procMemInfoPath: "/nonexistent/meminfo",
	}

	_, err := reader.ReadStats()
	if err == nil {
		t.Error("ReadStats() should return error for missing file")
	}
}

func TestMemoryReaderMalformedFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a malformed /proc/meminfo (missing colon separators)
	meminfoContent := `This is not valid
MemTotal 8192000 kB
`
	if err := os.WriteFile(filepath.Join(tmpDir, "meminfo"), []byte(meminfoContent), 0644); err != nil {
		t.Fatalf("failed to write mock meminfo: %v", err)
	}

	reader := &memoryReader{
		procMemInfoPath: filepath.Join(tmpDir, "meminfo"),
	}

	// Should not error, just return zeros for missing values
	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	if stats.Total != 0 {
		t.Errorf("Total = %d, want 0 for malformed input", stats.Total)
	}
}

func TestSafeMultiply(t *testing.T) {
	tests := []struct {
		name     string
		a        uint64
		b        uint64
		expected uint64
	}{
		{
			name:     "normal multiplication",
			a:        100,
			b:        200,
			expected: 20000,
		},
		{
			name:     "multiply by zero",
			a:        100,
			b:        0,
			expected: 0,
		},
		{
			name:     "zero times value",
			a:        0,
			b:        100,
			expected: 0,
		},
		{
			name:     "large values no overflow",
			a:        1000000,
			b:        1000000,
			expected: 1000000000000,
		},
		{
			name:     "overflow protection",
			a:        ^uint64(0),
			b:        2,
			expected: ^uint64(0),
		},
		{
			name:     "near overflow protection",
			a:        ^uint64(0) / 2,
			b:        3,
			expected: ^uint64(0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := safeMultiply(tt.a, tt.b)
			if got != tt.expected {
				t.Errorf("safeMultiply(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.expected)
			}
		})
	}
}
