//go:build !noebiten

package render

import (
	"sync"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestNewFrameMetrics(t *testing.T) {
	tests := []struct {
		name         string
		updatePeriod time.Duration
		wantPeriod   time.Duration
	}{
		{
			name:         "default period",
			updatePeriod: time.Second,
			wantPeriod:   time.Second,
		},
		{
			name:         "custom period",
			updatePeriod: 500 * time.Millisecond,
			wantPeriod:   500 * time.Millisecond,
		},
		{
			name:         "zero period defaults to 1 second",
			updatePeriod: 0,
			wantPeriod:   time.Second,
		},
		{
			name:         "negative period defaults to 1 second",
			updatePeriod: -time.Second,
			wantPeriod:   time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm := NewFrameMetrics(tt.updatePeriod)
			if fm == nil {
				t.Fatal("NewFrameMetrics returned nil")
			}
			if fm.updatePeriod != tt.wantPeriod {
				t.Errorf("updatePeriod = %v, want %v", fm.updatePeriod, tt.wantPeriod)
			}
		})
	}
}

func TestFrameMetricsRecordFrame(t *testing.T) {
	fm := NewFrameMetrics(time.Second)

	// Record several frames
	frameTime := 16 * time.Millisecond // ~60 FPS
	for i := 0; i < 10; i++ {
		fm.RecordFrame(frameTime)
	}

	// Check last frame time
	if got := fm.LastFrameTime(); got != frameTime {
		t.Errorf("LastFrameTime() = %v, want %v", got, frameTime)
	}

	// Min frame time should be the frame time
	if got := fm.MinFrameTime(); got != frameTime {
		t.Errorf("MinFrameTime() = %v, want %v", got, frameTime)
	}

	// Max frame time should also be the frame time (all same)
	if got := fm.MaxFrameTime(); got != frameTime {
		t.Errorf("MaxFrameTime() = %v, want %v", got, frameTime)
	}
}

func TestFrameMetricsMinMax(t *testing.T) {
	fm := NewFrameMetrics(time.Second)

	// Record frames with different durations
	fm.RecordFrame(10 * time.Millisecond)
	fm.RecordFrame(20 * time.Millisecond)
	fm.RecordFrame(15 * time.Millisecond)

	if got := fm.MinFrameTime(); got != 10*time.Millisecond {
		t.Errorf("MinFrameTime() = %v, want 10ms", got)
	}

	if got := fm.MaxFrameTime(); got != 20*time.Millisecond {
		t.Errorf("MaxFrameTime() = %v, want 20ms", got)
	}
}

func TestFrameMetricsReset(t *testing.T) {
	fm := NewFrameMetrics(time.Second)

	fm.RecordFrame(16 * time.Millisecond)
	fm.Reset()

	if got := fm.LastFrameTime(); got != 0 {
		t.Errorf("LastFrameTime() after reset = %v, want 0", got)
	}
	if got := fm.MaxFrameTime(); got != 0 {
		t.Errorf("MaxFrameTime() after reset = %v, want 0", got)
	}
}

func TestFrameMetricsConcurrency(t *testing.T) {
	fm := NewFrameMetrics(100 * time.Millisecond)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				fm.RecordFrame(time.Duration(j) * time.Millisecond)
			}
		}()
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = fm.FPS()
				_ = fm.LastFrameTime()
				_ = fm.MinFrameTime()
				_ = fm.MaxFrameTime()
			}
		}()
	}

	wg.Wait()
}

func TestDrawOptionsPool(t *testing.T) {
	pool := NewDrawOptionsPool()

	// Get an options instance
	op := pool.Get()
	if op == nil {
		t.Fatal("Get() returned nil")
	}

	// Verify it's reset by checking that applying identity translation has no effect
	// A reset GeoM should have element(0,0) = 1 and element(1,1) = 1 (identity matrix)
	e00 := op.GeoM.Element(0, 0)
	e11 := op.GeoM.Element(1, 1)
	if e00 != 1.0 || e11 != 1.0 {
		t.Errorf("GeoM should be identity after Get(), got diagonal (%v, %v)", e00, e11)
	}

	// Modify and put back
	op.GeoM.Translate(100, 100)
	pool.Put(op)

	// Get again and verify reset
	op2 := pool.Get()
	e00 = op2.GeoM.Element(0, 0)
	e11 = op2.GeoM.Element(1, 1)
	e02 := op2.GeoM.Element(0, 2) // translation x
	e12 := op2.GeoM.Element(1, 2) // translation y
	if e00 != 1.0 || e11 != 1.0 || e02 != 0.0 || e12 != 0.0 {
		t.Error("GeoM should be reset after Get()")
	}
}

func TestDrawOptionsPoolConcurrency(t *testing.T) {
	pool := NewDrawOptionsPool()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			op := pool.Get()
			op.GeoM.Translate(1, 1)
			pool.Put(op)
		}()
	}
	wg.Wait()
}

func TestVertexPool(t *testing.T) {
	pool := NewVertexPool()

	// Get a slice
	vertices := pool.Get()
	if vertices == nil {
		t.Fatal("Get() returned nil")
	}
	if len(vertices) != 0 {
		t.Errorf("len = %d, want 0", len(vertices))
	}
	if cap(vertices) < 4 {
		t.Errorf("cap = %d, want >= 4", cap(vertices))
	}

	// Append and put back
	vertices = append(vertices, ebiten.Vertex{DstX: 1, DstY: 2})
	pool.Put(vertices)

	// Get again
	vertices2 := pool.Get()
	if len(vertices2) != 0 {
		t.Errorf("len after get = %d, want 0", len(vertices2))
	}
}

func TestIndexPool(t *testing.T) {
	pool := NewIndexPool()

	// Get a slice
	indices := pool.Get()
	if indices == nil {
		t.Fatal("Get() returned nil")
	}
	if len(indices) != 0 {
		t.Errorf("len = %d, want 0", len(indices))
	}
	if cap(indices) < 6 {
		t.Errorf("cap = %d, want >= 6", cap(indices))
	}

	// Append and put back
	indices = append(indices, 0, 1, 2)
	pool.Put(indices)

	// Get again
	indices2 := pool.Get()
	if len(indices2) != 0 {
		t.Errorf("len after get = %d, want 0", len(indices2))
	}
}

func TestDirtyRegionContains(t *testing.T) {
	r := DirtyRegion{X: 10, Y: 20, Width: 100, Height: 50}

	tests := []struct {
		x, y float64
		want bool
	}{
		{15, 25, true},      // Inside
		{10, 20, true},      // Top-left corner
		{109, 69, true},     // Just inside bottom-right
		{110, 70, false},    // At bottom-right edge (exclusive)
		{5, 25, false},      // Left of region
		{115, 25, false},    // Right of region
		{15, 15, false},     // Above region
		{15, 75, false},     // Below region
	}

	for _, tt := range tests {
		if got := r.Contains(tt.x, tt.y); got != tt.want {
			t.Errorf("Contains(%v, %v) = %v, want %v", tt.x, tt.y, got, tt.want)
		}
	}
}

func TestDirtyRegionIntersects(t *testing.T) {
	r := DirtyRegion{X: 10, Y: 10, Width: 50, Height: 50}

	tests := []struct {
		name  string
		other DirtyRegion
		want  bool
	}{
		{
			name:  "overlapping",
			other: DirtyRegion{X: 40, Y: 40, Width: 50, Height: 50},
			want:  true,
		},
		{
			name:  "contained",
			other: DirtyRegion{X: 20, Y: 20, Width: 10, Height: 10},
			want:  true,
		},
		{
			name:  "containing",
			other: DirtyRegion{X: 0, Y: 0, Width: 100, Height: 100},
			want:  true,
		},
		{
			name:  "to the right",
			other: DirtyRegion{X: 100, Y: 10, Width: 50, Height: 50},
			want:  false,
		},
		{
			name:  "to the left",
			other: DirtyRegion{X: 0, Y: 10, Width: 5, Height: 50},
			want:  false,
		},
		{
			name:  "above",
			other: DirtyRegion{X: 10, Y: 0, Width: 50, Height: 5},
			want:  false,
		},
		{
			name:  "below",
			other: DirtyRegion{X: 10, Y: 100, Width: 50, Height: 50},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := r.Intersects(tt.other); got != tt.want {
				t.Errorf("Intersects() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDirtyRegionUnion(t *testing.T) {
	r1 := DirtyRegion{X: 10, Y: 10, Width: 50, Height: 50}
	r2 := DirtyRegion{X: 40, Y: 40, Width: 50, Height: 50}

	union := r1.Union(r2)

	if union.X != 10 {
		t.Errorf("Union X = %v, want 10", union.X)
	}
	if union.Y != 10 {
		t.Errorf("Union Y = %v, want 10", union.Y)
	}
	if union.Width != 80 {
		t.Errorf("Union Width = %v, want 80", union.Width)
	}
	if union.Height != 80 {
		t.Errorf("Union Height = %v, want 80", union.Height)
	}
}

func TestNewDirtyTracker(t *testing.T) {
	dt := NewDirtyTracker(800, 600)
	if dt == nil {
		t.Fatal("NewDirtyTracker returned nil")
	}
	if dt.screenW != 800 || dt.screenH != 600 {
		t.Errorf("screen size = (%d, %d), want (800, 600)", dt.screenW, dt.screenH)
	}
	if !dt.IsEmpty() {
		t.Error("new tracker should be empty")
	}
}

func TestDirtyTrackerMarkDirty(t *testing.T) {
	dt := NewDirtyTracker(800, 600)

	dt.MarkDirty(DirtyRegion{X: 10, Y: 10, Width: 50, Height: 50})

	if dt.IsEmpty() {
		t.Error("tracker should not be empty after marking dirty")
	}
	if dt.RegionCount() != 1 {
		t.Errorf("RegionCount() = %d, want 1", dt.RegionCount())
	}
}

func TestDirtyTrackerMarkDirtyMerge(t *testing.T) {
	dt := NewDirtyTracker(800, 600)

	// Mark two overlapping regions
	dt.MarkDirty(DirtyRegion{X: 10, Y: 10, Width: 50, Height: 50})
	dt.MarkDirty(DirtyRegion{X: 40, Y: 40, Width: 50, Height: 50})

	// Should be merged into one
	if dt.RegionCount() != 1 {
		t.Errorf("RegionCount() = %d, want 1 (merged)", dt.RegionCount())
	}
}

func TestDirtyTrackerMarkDirtyClamp(t *testing.T) {
	dt := NewDirtyTracker(100, 100)

	// Mark region outside screen bounds
	dt.MarkDirty(DirtyRegion{X: -10, Y: -10, Width: 50, Height: 50})

	regions := dt.DirtyRegions()
	if len(regions) != 1 {
		t.Fatalf("RegionCount() = %d, want 1", len(regions))
	}

	r := regions[0]
	if r.X != 0 || r.Y != 0 {
		t.Errorf("Region should be clamped to (0,0), got (%v, %v)", r.X, r.Y)
	}
}

func TestDirtyTrackerMarkDirtyEmpty(t *testing.T) {
	dt := NewDirtyTracker(100, 100)

	// Mark empty region
	dt.MarkDirty(DirtyRegion{X: 50, Y: 50, Width: 0, Height: 0})

	if !dt.IsEmpty() {
		t.Error("empty regions should not be tracked")
	}
}

func TestDirtyTrackerMarkFullRedraw(t *testing.T) {
	dt := NewDirtyTracker(800, 600)

	dt.MarkDirty(DirtyRegion{X: 10, Y: 10, Width: 50, Height: 50})
	dt.MarkFullRedraw()

	if !dt.NeedsFullRedraw() {
		t.Error("NeedsFullRedraw() should be true")
	}

	regions := dt.DirtyRegions()
	if len(regions) != 1 {
		t.Fatalf("len(DirtyRegions()) = %d, want 1", len(regions))
	}
	if regions[0].Width != 800 || regions[0].Height != 600 {
		t.Errorf("full redraw region = %v, want full screen", regions[0])
	}
}

func TestDirtyTrackerClear(t *testing.T) {
	dt := NewDirtyTracker(800, 600)

	dt.MarkDirty(DirtyRegion{X: 10, Y: 10, Width: 50, Height: 50})
	dt.MarkFullRedraw()
	dt.Clear()

	if !dt.IsEmpty() {
		t.Error("tracker should be empty after clear")
	}
	if dt.NeedsFullRedraw() {
		t.Error("NeedsFullRedraw should be false after clear")
	}
}

func TestDirtyTrackerSetScreenSize(t *testing.T) {
	dt := NewDirtyTracker(800, 600)

	dt.SetScreenSize(1920, 1080)

	if !dt.NeedsFullRedraw() {
		t.Error("screen size change should trigger full redraw")
	}
}

func TestDirtyTrackerConcurrency(t *testing.T) {
	dt := NewDirtyTracker(800, 600)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				dt.MarkDirty(DirtyRegion{
					X:      float64(n * 10),
					Y:      float64(j),
					Width:  10,
					Height: 10,
				})
			}
		}(i)
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = dt.DirtyRegions()
				_ = dt.IsEmpty()
				_ = dt.RegionCount()
			}
		}()
	}

	wg.Wait()
}

func TestRenderStats(t *testing.T) {
	rs := NewRenderStats()

	rs.RecordDrawCall(4)
	rs.RecordDrawCall(6)
	rs.RecordTextDraw()

	draws, vertices, texts := rs.Stats()

	if draws != 2 {
		t.Errorf("DrawCalls = %d, want 2", draws)
	}
	if vertices != 10 {
		t.Errorf("VertexCount = %d, want 10", vertices)
	}
	if texts != 1 {
		t.Errorf("TextDrawCalls = %d, want 1", texts)
	}
}

func TestRenderStatsReset(t *testing.T) {
	rs := NewRenderStats()

	rs.RecordDrawCall(4)
	rs.RecordTextDraw()
	rs.Reset()

	draws, vertices, texts := rs.Stats()

	if draws != 0 || vertices != 0 || texts != 0 {
		t.Errorf("Stats after reset = (%d, %d, %d), want (0, 0, 0)", draws, vertices, texts)
	}
}

func TestRenderStatsConcurrency(t *testing.T) {
	rs := NewRenderStats()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				rs.RecordDrawCall(4)
				rs.RecordTextDraw()
			}
		}()
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_, _, _ = rs.Stats()
			}
		}()
	}

	wg.Wait()
}

func TestDefaultPerformanceConfig(t *testing.T) {
	config := DefaultPerformanceConfig()

	if config.TargetFPS != 60 {
		t.Errorf("TargetFPS = %d, want 60", config.TargetFPS)
	}
	if !config.EnableDirtyTracking {
		t.Error("EnableDirtyTracking should be true by default")
	}
	if !config.EnableObjectPooling {
		t.Error("EnableObjectPooling should be true by default")
	}
	if !config.EnableMetrics {
		t.Error("EnableMetrics should be true by default")
	}
	if config.MetricsUpdatePeriod != time.Second {
		t.Errorf("MetricsUpdatePeriod = %v, want 1s", config.MetricsUpdatePeriod)
	}
}

func TestNewPerformanceManager(t *testing.T) {
	config := DefaultPerformanceConfig()
	pm := NewPerformanceManager(config, 800, 600)

	if pm == nil {
		t.Fatal("NewPerformanceManager returned nil")
	}
	if pm.Metrics() == nil {
		t.Error("Metrics() should not be nil with EnableMetrics=true")
	}
	if pm.Stats() == nil {
		t.Error("Stats() should not be nil")
	}
	if pm.DirtyTracker() == nil {
		t.Error("DirtyTracker() should not be nil with EnableDirtyTracking=true")
	}
}

func TestPerformanceManagerDisabled(t *testing.T) {
	config := PerformanceConfig{
		TargetFPS:           60,
		EnableDirtyTracking: false,
		EnableObjectPooling: false,
		EnableMetrics:       false,
	}
	pm := NewPerformanceManager(config, 800, 600)

	if pm.Metrics() != nil {
		t.Error("Metrics() should be nil with EnableMetrics=false")
	}
	if pm.DirtyTracker() != nil {
		t.Error("DirtyTracker() should be nil with EnableDirtyTracking=false")
	}
}

func TestPerformanceManagerObjectPooling(t *testing.T) {
	config := DefaultPerformanceConfig()
	pm := NewPerformanceManager(config, 800, 600)

	// Test draw options pool
	op := pm.GetDrawOptions()
	if op == nil {
		t.Fatal("GetDrawOptions() returned nil")
	}
	pm.PutDrawOptions(op)

	// Test vertex pool
	vertices := pm.GetVertices()
	if vertices == nil {
		t.Fatal("GetVertices() returned nil")
	}
	pm.PutVertices(vertices)

	// Test index pool
	indices := pm.GetIndices()
	if indices == nil {
		t.Fatal("GetIndices() returned nil")
	}
	pm.PutIndices(indices)
}

func TestPerformanceManagerPoolingDisabled(t *testing.T) {
	config := PerformanceConfig{
		TargetFPS:           60,
		EnableObjectPooling: false,
	}
	pm := NewPerformanceManager(config, 800, 600)

	// Should still work, just creates new instances
	op := pm.GetDrawOptions()
	if op == nil {
		t.Fatal("GetDrawOptions() returned nil even with pooling disabled")
	}
	pm.PutDrawOptions(op) // Should not panic

	vertices := pm.GetVertices()
	if vertices == nil {
		t.Fatal("GetVertices() returned nil even with pooling disabled")
	}
	pm.PutVertices(vertices) // Should not panic
}

func TestPerformanceManagerRecordFrame(t *testing.T) {
	config := DefaultPerformanceConfig()
	pm := NewPerformanceManager(config, 800, 600)

	pm.RecordFrame(16 * time.Millisecond)

	if pm.Metrics().LastFrameTime() != 16*time.Millisecond {
		t.Errorf("LastFrameTime() = %v, want 16ms", pm.Metrics().LastFrameTime())
	}
}

func TestPerformanceManagerSetScreenSize(t *testing.T) {
	config := DefaultPerformanceConfig()
	pm := NewPerformanceManager(config, 800, 600)

	pm.SetScreenSize(1920, 1080)

	if !pm.DirtyTracker().NeedsFullRedraw() {
		t.Error("screen size change should trigger full redraw")
	}
}

func TestPerformanceManagerResetStats(t *testing.T) {
	config := DefaultPerformanceConfig()
	pm := NewPerformanceManager(config, 800, 600)

	pm.RecordFrame(16 * time.Millisecond)
	pm.Stats().RecordDrawCall(4)
	pm.ResetStats()

	if pm.Metrics().LastFrameTime() != 0 {
		t.Error("metrics should be reset")
	}
	draws, _, _ := pm.Stats().Stats()
	if draws != 0 {
		t.Error("stats should be reset")
	}
}
