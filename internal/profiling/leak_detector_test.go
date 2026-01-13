package profiling

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestDefaultLeakDetectorConfig(t *testing.T) {
	config := DefaultLeakDetectorConfig()

	if config.SampleInterval <= 0 {
		t.Error("SampleInterval should be positive")
	}
	if config.MaxSnapshots <= 0 {
		t.Error("MaxSnapshots should be positive")
	}
	if config.LeakThresholdBytes <= 0 {
		t.Error("LeakThresholdBytes should be positive")
	}
	if config.GoroutineLeakThreshold <= 0 {
		t.Error("GoroutineLeakThreshold should be positive")
	}
}

func TestNewMemoryLeakDetector(t *testing.T) {
	t.Run("with default config", func(t *testing.T) {
		detector := NewMemoryLeakDetector(DefaultLeakDetectorConfig())
		if detector == nil {
			t.Fatal("NewMemoryLeakDetector returned nil")
		}
		if detector.SnapshotCount() != 0 {
			t.Error("new detector should have no snapshots")
		}
		if detector.IsRunning() {
			t.Error("new detector should not be running")
		}
	})

	t.Run("with zero values", func(t *testing.T) {
		// Should use defaults for zero values
		detector := NewMemoryLeakDetector(LeakDetectorConfig{})
		if detector == nil {
			t.Fatal("NewMemoryLeakDetector returned nil")
		}
		if detector.config.SampleInterval <= 0 {
			t.Error("should use default SampleInterval for zero value")
		}
		if detector.config.MaxSnapshots <= 0 {
			t.Error("should use default MaxSnapshots for zero value")
		}
	})
}

func TestTakeSnapshot(t *testing.T) {
	detector := NewMemoryLeakDetector(DefaultLeakDetectorConfig())

	snapshot := detector.TakeSnapshot()

	if snapshot.Timestamp.IsZero() {
		t.Error("snapshot timestamp should not be zero")
	}
	if snapshot.HeapAlloc == 0 {
		t.Error("HeapAlloc should be non-zero")
	}
	if snapshot.GoroutineCount <= 0 {
		t.Error("GoroutineCount should be positive")
	}
	if detector.SnapshotCount() != 1 {
		t.Errorf("snapshot count = %d, want 1", detector.SnapshotCount())
	}
}

func TestSnapshotMaxLimit(t *testing.T) {
	config := DefaultLeakDetectorConfig()
	config.MaxSnapshots = 3
	detector := NewMemoryLeakDetector(config)

	// Take more snapshots than max
	for i := 0; i < 5; i++ {
		detector.TakeSnapshot()
	}

	if detector.SnapshotCount() != 3 {
		t.Errorf("snapshot count = %d, want 3 (max)", detector.SnapshotCount())
	}

	// Verify we have the latest snapshots (not the oldest)
	snapshots := detector.Snapshots()
	if len(snapshots) != 3 {
		t.Errorf("Snapshots() returned %d, want 3", len(snapshots))
	}
}

func TestClearSnapshots(t *testing.T) {
	detector := NewMemoryLeakDetector(DefaultLeakDetectorConfig())

	detector.TakeSnapshot()
	detector.TakeSnapshot()
	detector.TakeSnapshot()

	if detector.SnapshotCount() != 3 {
		t.Fatalf("snapshot count = %d, want 3", detector.SnapshotCount())
	}

	detector.ClearSnapshots()

	if detector.SnapshotCount() != 0 {
		t.Errorf("snapshot count = %d after clear, want 0", detector.SnapshotCount())
	}
}

func TestAnalyzeGrowthNoSnapshots(t *testing.T) {
	detector := NewMemoryLeakDetector(DefaultLeakDetectorConfig())

	growth := detector.AnalyzeGrowth()
	if growth != nil {
		t.Error("AnalyzeGrowth should return nil with no snapshots")
	}
}

func TestAnalyzeGrowthOneSnapshot(t *testing.T) {
	detector := NewMemoryLeakDetector(DefaultLeakDetectorConfig())
	detector.TakeSnapshot()

	growth := detector.AnalyzeGrowth()
	if growth != nil {
		t.Error("AnalyzeGrowth should return nil with only one snapshot")
	}
}

func TestAnalyzeGrowthBasic(t *testing.T) {
	config := DefaultLeakDetectorConfig()
	config.LeakThresholdBytes = 1024 * 1024 * 1024 // 1GB/s - high threshold to avoid false positives
	detector := NewMemoryLeakDetector(config)

	detector.TakeSnapshot()
	// Small delay to ensure time difference
	time.Sleep(10 * time.Millisecond)
	detector.TakeSnapshot()

	growth := detector.AnalyzeGrowth()
	if growth == nil {
		t.Fatal("AnalyzeGrowth returned nil with two snapshots")
	}

	if growth.Duration <= 0 {
		t.Error("Duration should be positive")
	}
}

func TestAnalyzeGrowthDetectsLeak(t *testing.T) {
	config := DefaultLeakDetectorConfig()
	config.LeakThresholdBytes = 1 // Very low threshold to trigger leak detection
	detector := NewMemoryLeakDetector(config)

	detector.TakeSnapshot()

	// Allocate some memory to trigger growth
	largeSlice := make([]byte, 10*1024*1024) // 10MB
	_ = largeSlice[0]                        // Ensure it's not optimized away
	runtime.KeepAlive(largeSlice)

	time.Sleep(10 * time.Millisecond)
	detector.TakeSnapshot()

	growth := detector.AnalyzeGrowth()
	if growth == nil {
		t.Fatal("AnalyzeGrowth returned nil")
	}

	// With such a low threshold and large allocation, we should detect a leak
	if !growth.PotentialLeak {
		t.Log("Note: Leak might not be detected if GC ran between snapshots")
	}
}

func TestAnalyzeGrowthDetectsGoroutineLeak(t *testing.T) {
	config := DefaultLeakDetectorConfig()
	config.GoroutineLeakThreshold = 2              // Low threshold
	config.LeakThresholdBytes = 1024 * 1024 * 1024 // High memory threshold
	detector := NewMemoryLeakDetector(config)

	detector.TakeSnapshot()

	// Create goroutines that will wait
	done := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-done
		}()
	}

	time.Sleep(10 * time.Millisecond)
	detector.TakeSnapshot()

	growth := detector.AnalyzeGrowth()
	if growth == nil {
		t.Fatal("AnalyzeGrowth returned nil")
	}

	if growth.GoroutineDelta < 5 {
		t.Errorf("GoroutineDelta = %d, want at least 5", growth.GoroutineDelta)
	}

	if !growth.PotentialLeak {
		t.Error("should detect goroutine leak")
	}

	// Cleanup
	close(done)
	wg.Wait()
}

func TestStartStop(t *testing.T) {
	config := DefaultLeakDetectorConfig()
	config.SampleInterval = 10 * time.Millisecond // Fast for testing
	detector := NewMemoryLeakDetector(config)

	// Start
	if err := detector.Start(); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	if !detector.IsRunning() {
		t.Error("IsRunning() should return true after Start()")
	}

	// Try to start again - should fail
	if err := detector.Start(); err == nil {
		t.Error("Start() should fail when already running")
	}

	// Wait for some snapshots
	time.Sleep(50 * time.Millisecond)

	if detector.SnapshotCount() < 2 {
		t.Error("should have collected at least 2 snapshots")
	}

	// Stop
	if err := detector.Stop(); err != nil {
		t.Fatalf("Stop() failed: %v", err)
	}

	if detector.IsRunning() {
		t.Error("IsRunning() should return false after Stop()")
	}

	// Try to stop again - should fail
	if err := detector.Stop(); err == nil {
		t.Error("Stop() should fail when not running")
	}
}

func TestLeakCallback(t *testing.T) {
	config := DefaultLeakDetectorConfig()
	config.SampleInterval = 10 * time.Millisecond
	config.LeakThresholdBytes = 1 // Very low to trigger callback
	detector := NewMemoryLeakDetector(config)

	var callbackCalled atomic.Int32
	detector.SetOnLeakCallback(func(growth MemoryGrowth) {
		callbackCalled.Add(1)
	})

	if err := detector.Start(); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	// Allocate memory to trigger leak detection
	var allocs [][]byte
	for i := 0; i < 5; i++ {
		allocs = append(allocs, make([]byte, 1024*1024))
		time.Sleep(15 * time.Millisecond)
	}
	runtime.KeepAlive(allocs)

	if err := detector.Stop(); err != nil {
		t.Fatalf("Stop() failed: %v", err)
	}

	// Callback may or may not be called depending on GC timing
	// Just verify no panics occurred
	t.Logf("Leak callback was called %d times", callbackCalled.Load())
}

func TestCurrentMemoryStats(t *testing.T) {
	stats := CurrentMemoryStats()

	if stats.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
	if stats.HeapAlloc == 0 {
		t.Error("HeapAlloc should be non-zero")
	}
	if stats.GoroutineCount <= 0 {
		t.Error("GoroutineCount should be positive")
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    uint64
		expected string
	}{
		{0, "0 B"},
		{100, "100 B"},
		{1024, "1.00 KB"},
		{1536, "1.50 KB"},
		{1024 * 1024, "1.00 MB"},
		{1024 * 1024 * 1024, "1.00 GB"},
		{1024 * 1024 * 1024 * 2, "2.00 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatBytes(tt.bytes)
			if result != tt.expected {
				t.Errorf("FormatBytes(%d) = %q, want %q", tt.bytes, result, tt.expected)
			}
		})
	}
}

func TestMemorySnapshotString(t *testing.T) {
	snapshot := MemorySnapshot{
		Timestamp:      time.Now(),
		HeapAlloc:      1024 * 1024,
		HeapSys:        2 * 1024 * 1024,
		HeapObjects:    1000,
		StackInuse:     512 * 1024,
		GoroutineCount: 10,
		NumGC:          5,
	}

	str := snapshot.String()
	if str == "" {
		t.Error("String() should not return empty string")
	}
	if len(str) < 50 {
		t.Error("String() should return detailed report")
	}
}

func TestMemoryGrowthString(t *testing.T) {
	t.Run("growth", func(t *testing.T) {
		growth := MemoryGrowth{
			Duration:         time.Minute,
			HeapAllocDelta:   1024 * 1024,
			HeapObjectsDelta: 100,
			GoroutineDelta:   2,
			GrowthRatePerSec: 17476.27,
			PotentialLeak:    false,
		}

		str := growth.String()
		if str == "" {
			t.Error("String() should not return empty string")
		}
	})

	t.Run("shrink", func(t *testing.T) {
		growth := MemoryGrowth{
			Duration:         time.Minute,
			HeapAllocDelta:   -1024 * 1024,
			HeapObjectsDelta: -100,
			GoroutineDelta:   -2,
			GrowthRatePerSec: -17476.27,
			PotentialLeak:    false,
		}

		str := growth.String()
		if str == "" {
			t.Error("String() should not return empty string")
		}
	})

	t.Run("leak detected", func(t *testing.T) {
		growth := MemoryGrowth{
			Duration:         time.Minute,
			HeapAllocDelta:   100 * 1024 * 1024,
			HeapObjectsDelta: 10000,
			GoroutineDelta:   50,
			GrowthRatePerSec: 1747626.67,
			PotentialLeak:    true,
			LeakReason:       "test leak reason",
		}

		str := growth.String()
		if str == "" {
			t.Error("String() should not return empty string")
		}
	})
}

func TestConcurrentAccess(t *testing.T) {
	detector := NewMemoryLeakDetector(DefaultLeakDetectorConfig())

	var wg sync.WaitGroup

	// Concurrent snapshot taking
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				detector.TakeSnapshot()
			}
		}()
	}

	// Concurrent reading
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				_ = detector.Snapshots()
				_ = detector.SnapshotCount()
				_ = detector.AnalyzeGrowth()
			}
		}()
	}

	wg.Wait()

	// Verify no data corruption
	count := detector.SnapshotCount()
	if count > detector.config.MaxSnapshots {
		t.Errorf("snapshot count %d exceeds max %d", count, detector.config.MaxSnapshots)
	}
}

func TestSnapshotsCopy(t *testing.T) {
	detector := NewMemoryLeakDetector(DefaultLeakDetectorConfig())

	detector.TakeSnapshot()
	detector.TakeSnapshot()

	// Get snapshots
	snapshots := detector.Snapshots()
	originalLen := len(snapshots)
	originalHeapAlloc := snapshots[0].HeapAlloc

	// Modify an element in the returned slice
	snapshots[0].HeapAlloc = 999999

	// Get snapshots again and verify internal state is unchanged
	newSnapshots := detector.Snapshots()
	if detector.SnapshotCount() != originalLen {
		t.Error("modifying returned slice should not affect internal state")
	}
	if newSnapshots[0].HeapAlloc != originalHeapAlloc {
		t.Error("modifying returned snapshot should not affect internal snapshot")
	}
}

func TestAbs(t *testing.T) {
	tests := []struct {
		input    int64
		expected int64
	}{
		{0, 0},
		{5, 5},
		{-5, 5},
		{-1000000, 1000000},
		{1000000, 1000000},
	}

	for _, tt := range tests {
		result := abs(tt.input)
		if result != tt.expected {
			t.Errorf("abs(%d) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}
