// Package profiling provides CPU and memory profiling support for conky-go.
// This file implements memory leak detection and prevention utilities.
package profiling

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

// Byte size constants for memory formatting
const (
	KB = 1024
	MB = KB * 1024
	GB = MB * 1024
)

// MemorySnapshot represents a point-in-time memory measurement.
type MemorySnapshot struct {
	Timestamp      time.Time
	HeapAlloc      uint64 // Bytes of allocated heap objects
	HeapSys        uint64 // Bytes of heap memory obtained from OS
	HeapObjects    uint64 // Number of allocated heap objects
	StackInuse     uint64 // Bytes in stack spans
	GoroutineCount int    // Number of active goroutines
	NumGC          uint32 // Number of GCs completed
}

// MemoryGrowth tracks memory growth between snapshots.
type MemoryGrowth struct {
	Duration         time.Duration
	HeapAllocDelta   int64   // Positive means growth, negative means shrink
	HeapObjectsDelta int64   // Change in number of heap objects
	GoroutineDelta   int     // Change in goroutine count
	GrowthRatePerSec float64 // Bytes per second growth rate
	PotentialLeak    bool    // Indicates possible memory leak
	LeakReason       string  // Explanation if PotentialLeak is true
}

// LeakDetectorConfig configures the memory leak detector.
type LeakDetectorConfig struct {
	// SampleInterval is the interval between memory snapshots.
	// Shorter intervals provide more granular data but increase overhead.
	SampleInterval time.Duration

	// MaxSnapshots is the maximum number of snapshots to retain.
	// Older snapshots are discarded when this limit is reached.
	MaxSnapshots int

	// LeakThresholdBytes is the minimum bytes per second growth
	// to consider as a potential leak.
	LeakThresholdBytes int64

	// GoroutineLeakThreshold is the number of goroutines that can
	// be added before considering it a potential leak.
	GoroutineLeakThreshold int
}

// DefaultLeakDetectorConfig returns a LeakDetectorConfig with sensible defaults.
func DefaultLeakDetectorConfig() LeakDetectorConfig {
	return LeakDetectorConfig{
		SampleInterval:         time.Second * 10,
		MaxSnapshots:           100,
		LeakThresholdBytes:     1024 * 1024, // 1MB per second sustained growth
		GoroutineLeakThreshold: 10,          // 10 goroutines net increase
	}
}

// MemoryLeakDetector monitors memory usage over time to detect potential leaks.
// It provides snapshots, growth analysis, and goroutine monitoring.
type MemoryLeakDetector struct {
	config    LeakDetectorConfig
	snapshots []MemorySnapshot
	running   bool
	stopChan  chan struct{}
	doneChan  chan struct{}
	mu        sync.RWMutex
	onLeak    func(growth MemoryGrowth) // Optional callback when leak detected
}

// NewMemoryLeakDetector creates a new MemoryLeakDetector with the given configuration.
func NewMemoryLeakDetector(config LeakDetectorConfig) *MemoryLeakDetector {
	if config.SampleInterval <= 0 {
		config.SampleInterval = DefaultLeakDetectorConfig().SampleInterval
	}
	if config.MaxSnapshots <= 0 {
		config.MaxSnapshots = DefaultLeakDetectorConfig().MaxSnapshots
	}
	if config.LeakThresholdBytes <= 0 {
		config.LeakThresholdBytes = DefaultLeakDetectorConfig().LeakThresholdBytes
	}
	if config.GoroutineLeakThreshold <= 0 {
		config.GoroutineLeakThreshold = DefaultLeakDetectorConfig().GoroutineLeakThreshold
	}

	return &MemoryLeakDetector{
		config:    config,
		snapshots: make([]MemorySnapshot, 0, config.MaxSnapshots),
	}
}

// TakeSnapshot captures the current memory state.
func (d *MemoryLeakDetector) TakeSnapshot() MemorySnapshot {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	snapshot := MemorySnapshot{
		Timestamp:      time.Now(),
		HeapAlloc:      memStats.HeapAlloc,
		HeapSys:        memStats.HeapSys,
		HeapObjects:    memStats.HeapObjects,
		StackInuse:     memStats.StackInuse,
		GoroutineCount: runtime.NumGoroutine(),
		NumGC:          memStats.NumGC,
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	// Add snapshot and maintain max size
	d.snapshots = append(d.snapshots, snapshot)
	if len(d.snapshots) > d.config.MaxSnapshots {
		// Remove oldest snapshot
		d.snapshots = d.snapshots[1:]
	}

	return snapshot
}

// Snapshots returns a copy of all stored snapshots.
func (d *MemoryLeakDetector) Snapshots() []MemorySnapshot {
	d.mu.RLock()
	defer d.mu.RUnlock()

	result := make([]MemorySnapshot, len(d.snapshots))
	copy(result, d.snapshots)
	return result
}

// SnapshotCount returns the number of stored snapshots.
func (d *MemoryLeakDetector) SnapshotCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.snapshots)
}

// ClearSnapshots removes all stored snapshots.
func (d *MemoryLeakDetector) ClearSnapshots() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.snapshots = d.snapshots[:0]
}

// AnalyzeGrowth compares the oldest and newest snapshots to determine memory growth.
// Returns nil if there are fewer than 2 snapshots.
func (d *MemoryLeakDetector) AnalyzeGrowth() *MemoryGrowth {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if len(d.snapshots) < 2 {
		return nil
	}

	first := d.snapshots[0]
	last := d.snapshots[len(d.snapshots)-1]

	return d.analyzeGrowthBetween(first, last)
}

// analyzeGrowthBetween calculates growth between two snapshots.
func (d *MemoryLeakDetector) analyzeGrowthBetween(first, last MemorySnapshot) *MemoryGrowth {
	duration := last.Timestamp.Sub(first.Timestamp)
	if duration <= 0 {
		return nil
	}

	heapDelta := int64(last.HeapAlloc) - int64(first.HeapAlloc)
	objectsDelta := int64(last.HeapObjects) - int64(first.HeapObjects)
	goroutineDelta := last.GoroutineCount - first.GoroutineCount

	growthRate := float64(heapDelta) / duration.Seconds()

	growth := &MemoryGrowth{
		Duration:         duration,
		HeapAllocDelta:   heapDelta,
		HeapObjectsDelta: objectsDelta,
		GoroutineDelta:   goroutineDelta,
		GrowthRatePerSec: growthRate,
	}

	// Determine if this is a potential leak
	if growthRate > float64(d.config.LeakThresholdBytes) {
		growth.PotentialLeak = true
		growth.LeakReason = fmt.Sprintf(
			"sustained memory growth of %.2f KB/s exceeds threshold of %.2f KB/s",
			growthRate/KB, float64(d.config.LeakThresholdBytes)/KB,
		)
	} else if goroutineDelta > d.config.GoroutineLeakThreshold {
		growth.PotentialLeak = true
		growth.LeakReason = fmt.Sprintf(
			"goroutine count increased by %d (threshold: %d), possible goroutine leak",
			goroutineDelta, d.config.GoroutineLeakThreshold,
		)
	}

	return growth
}

// Start begins automatic snapshot collection at the configured interval.
// Returns an error if the detector is already running.
func (d *MemoryLeakDetector) Start() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.running {
		return fmt.Errorf("leak detector is already running")
	}

	d.stopChan = make(chan struct{})
	d.doneChan = make(chan struct{})
	d.running = true

	go d.collectLoop()

	return nil
}

// Stop halts automatic snapshot collection.
// Returns an error if the detector is not running.
func (d *MemoryLeakDetector) Stop() error {
	d.mu.Lock()
	if !d.running {
		d.mu.Unlock()
		return fmt.Errorf("leak detector is not running")
	}
	close(d.stopChan)
	d.running = false
	d.mu.Unlock()

	// Wait for the goroutine to finish
	<-d.doneChan

	return nil
}

// IsRunning returns true if the detector is actively collecting snapshots.
func (d *MemoryLeakDetector) IsRunning() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.running
}

// SetOnLeakCallback sets a function to be called when a potential leak is detected.
// Set to nil to disable callbacks.
func (d *MemoryLeakDetector) SetOnLeakCallback(callback func(growth MemoryGrowth)) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.onLeak = callback
}

// collectLoop runs the automatic snapshot collection.
func (d *MemoryLeakDetector) collectLoop() {
	defer close(d.doneChan)

	ticker := time.NewTicker(d.config.SampleInterval)
	defer ticker.Stop()

	// Take initial snapshot
	d.TakeSnapshot()

	for {
		select {
		case <-d.stopChan:
			return
		case <-ticker.C:
			d.TakeSnapshot()

			// Check for leaks if we have enough data
			if growth := d.AnalyzeGrowth(); growth != nil && growth.PotentialLeak {
				d.mu.RLock()
				callback := d.onLeak
				d.mu.RUnlock()

				if callback != nil {
					callback(*growth)
				}
			}
		}
	}
}

// CurrentMemoryStats returns the current memory statistics without storing a snapshot.
func CurrentMemoryStats() MemorySnapshot {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return MemorySnapshot{
		Timestamp:      time.Now(),
		HeapAlloc:      memStats.HeapAlloc,
		HeapSys:        memStats.HeapSys,
		HeapObjects:    memStats.HeapObjects,
		StackInuse:     memStats.StackInuse,
		GoroutineCount: runtime.NumGoroutine(),
		NumGC:          memStats.NumGC,
	}
}

// FormatBytes formats a byte count as a human-readable string.
func FormatBytes(bytes uint64) string {
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// MemoryReport generates a human-readable memory usage report.
func (s MemorySnapshot) String() string {
	return fmt.Sprintf(
		"Memory Report at %s:\n"+
			"  Heap Allocated: %s\n"+
			"  Heap System:    %s\n"+
			"  Heap Objects:   %d\n"+
			"  Stack In Use:   %s\n"+
			"  Goroutines:     %d\n"+
			"  GC Cycles:      %d",
		s.Timestamp.Format(time.RFC3339),
		FormatBytes(s.HeapAlloc),
		FormatBytes(s.HeapSys),
		s.HeapObjects,
		FormatBytes(s.StackInuse),
		s.GoroutineCount,
		s.NumGC,
	)
}

// String returns a human-readable representation of the growth analysis.
func (g MemoryGrowth) String() string {
	direction := "increased"
	if g.HeapAllocDelta < 0 {
		direction = "decreased"
	}

	leakStatus := "No leak detected"
	if g.PotentialLeak {
		leakStatus = fmt.Sprintf("POTENTIAL LEAK: %s", g.LeakReason)
	}

	return fmt.Sprintf(
		"Memory Growth Analysis (over %s):\n"+
			"  Heap %s by %s (%.2f KB/s)\n"+
			"  Heap Objects: %+d\n"+
			"  Goroutines:   %+d\n"+
			"  Status:       %s",
		g.Duration.Round(time.Second),
		direction,
		FormatBytes(uint64(abs(g.HeapAllocDelta))),
		g.GrowthRatePerSec/KB,
		g.HeapObjectsDelta,
		g.GoroutineDelta,
		leakStatus,
	)
}

// abs returns the absolute value of an int64.
func abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
