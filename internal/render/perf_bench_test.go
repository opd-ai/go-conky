//go:build !noebiten

package render

import (
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// Benchmark for DrawOptionsPool
func BenchmarkDrawOptionsPoolGet(b *testing.B) {
	pool := NewDrawOptionsPool()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		op := pool.Get()
		pool.Put(op)
	}
}

func BenchmarkDrawOptionsNewAllocation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = &ebiten.DrawImageOptions{}
	}
}

// Benchmark for VertexPool
func BenchmarkVertexPoolGet(b *testing.B) {
	pool := NewVertexPool()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v := pool.Get()
		v = append(v, ebiten.Vertex{}, ebiten.Vertex{}, ebiten.Vertex{}, ebiten.Vertex{})
		pool.Put(v)
	}
}

func BenchmarkVertexNewAllocation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v := make([]ebiten.Vertex, 0, 4)
		v = append(v, ebiten.Vertex{}, ebiten.Vertex{}, ebiten.Vertex{}, ebiten.Vertex{})
		_ = v
	}
}

// Benchmark for IndexPool
func BenchmarkIndexPoolGet(b *testing.B) {
	pool := NewIndexPool()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		indices := pool.Get()
		indices = append(indices, 0, 1, 2, 0, 2, 3)
		pool.Put(indices)
	}
}

func BenchmarkIndexNewAllocation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		indices := make([]uint16, 0, 6)
		indices = append(indices, 0, 1, 2, 0, 2, 3)
		_ = indices
	}
}

// Benchmark for FrameMetrics
func BenchmarkFrameMetricsRecordFrame(b *testing.B) {
	fm := NewFrameMetrics(time.Second)
	frameTime := 16 * time.Millisecond

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fm.RecordFrame(frameTime)
	}
}

func BenchmarkFrameMetricsFPS(b *testing.B) {
	fm := NewFrameMetrics(time.Second)
	for i := 0; i < 1000; i++ {
		fm.RecordFrame(16 * time.Millisecond)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = fm.FPS()
	}
}

// Benchmark for DirtyTracker
func BenchmarkDirtyTrackerMarkDirty(b *testing.B) {
	dt := NewDirtyTracker(1920, 1080)
	region := DirtyRegion{X: 100, Y: 100, Width: 200, Height: 200}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dt.MarkDirty(region)
		if i%100 == 0 {
			dt.Clear()
		}
	}
}

func BenchmarkDirtyTrackerDirtyRegions(b *testing.B) {
	dt := NewDirtyTracker(1920, 1080)
	// Add several non-overlapping regions
	for i := 0; i < 10; i++ {
		dt.MarkDirty(DirtyRegion{
			X:      float64(i * 100),
			Y:      float64(i * 50),
			Width:  80,
			Height: 40,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = dt.DirtyRegions()
	}
}

// Benchmark for RenderStats
func BenchmarkRenderStatsRecordDrawCall(b *testing.B) {
	rs := NewRenderStats()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rs.RecordDrawCall(4)
	}
}

// Benchmark for DirtyRegion operations
func BenchmarkDirtyRegionIntersects(b *testing.B) {
	r1 := DirtyRegion{X: 10, Y: 10, Width: 100, Height: 100}
	r2 := DirtyRegion{X: 50, Y: 50, Width: 100, Height: 100}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r1.Intersects(r2)
	}
}

func BenchmarkDirtyRegionUnion(b *testing.B) {
	r1 := DirtyRegion{X: 10, Y: 10, Width: 100, Height: 100}
	r2 := DirtyRegion{X: 50, Y: 50, Width: 100, Height: 100}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r1.Union(r2)
	}
}

// Benchmark for PerformanceManager
func BenchmarkPerformanceManagerGetDrawOptions(b *testing.B) {
	config := DefaultPerformanceConfig()
	pm := NewPerformanceManager(config, 1920, 1080)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		op := pm.GetDrawOptions()
		pm.PutDrawOptions(op)
	}
}

func BenchmarkPerformanceManagerRecordFrame(b *testing.B) {
	config := DefaultPerformanceConfig()
	pm := NewPerformanceManager(config, 1920, 1080)
	frameTime := 16 * time.Millisecond

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pm.RecordFrame(frameTime)
	}
}

// Concurrent benchmarks
func BenchmarkDrawOptionsPoolConcurrent(b *testing.B) {
	pool := NewDrawOptionsPool()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			op := pool.Get()
			op.GeoM.Translate(1, 1)
			pool.Put(op)
		}
	})
}

func BenchmarkFrameMetricsConcurrent(b *testing.B) {
	fm := NewFrameMetrics(time.Second)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			fm.RecordFrame(16 * time.Millisecond)
			_ = fm.FPS()
		}
	})
}

func BenchmarkDirtyTrackerConcurrent(b *testing.B) {
	dt := NewDirtyTracker(1920, 1080)

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			dt.MarkDirty(DirtyRegion{
				X:      float64(i % 1000),
				Y:      float64(i % 500),
				Width:  100,
				Height: 100,
			})
			if i%100 == 0 {
				dt.Clear()
			}
			i++
		}
	})
}
