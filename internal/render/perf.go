// Package render provides Ebiten-based rendering capabilities for conky-go.
// This file implements performance optimization utilities including frame rate
// monitoring, object pooling, and dirty region tracking for 60fps rendering.
package render

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// FrameMetrics tracks frame timing and performance statistics.
// It provides real-time FPS monitoring and frame time analysis
// for performance optimization and debugging.
type FrameMetrics struct {
	frameCount    atomic.Int64
	lastFPS       atomic.Int64  // Stored as int64 (FPS * 1000 for precision)
	lastFrameTime atomic.Int64  // Stored as nanoseconds
	minFrameTime  atomic.Int64  // Minimum frame time in nanoseconds
	maxFrameTime  atomic.Int64  // Maximum frame time in nanoseconds
	totalTime     atomic.Int64  // Total time accumulated in nanoseconds
	lastUpdate    atomic.Int64  // Last update time as Unix nano
	updatePeriod  time.Duration // How often to recalculate FPS
	mu            sync.RWMutex
}

// NewFrameMetrics creates a new FrameMetrics instance.
// The updatePeriod determines how often FPS is recalculated (default: 1 second).
func NewFrameMetrics(updatePeriod time.Duration) *FrameMetrics {
	if updatePeriod <= 0 {
		updatePeriod = time.Second
	}
	fm := &FrameMetrics{
		updatePeriod: updatePeriod,
	}
	fm.lastUpdate.Store(time.Now().UnixNano())
	fm.minFrameTime.Store(int64(time.Hour)) // Start with a large value
	return fm
}

// RecordFrame records a new frame with its duration.
// This should be called once per frame in the Update or Draw loop.
func (fm *FrameMetrics) RecordFrame(frameTime time.Duration) {
	frameNanos := frameTime.Nanoseconds()

	// Update atomic counters
	fm.frameCount.Add(1)
	fm.lastFrameTime.Store(frameNanos)
	fm.totalTime.Add(frameNanos)

	// Update min/max using compare-and-swap
	for {
		currentMin := fm.minFrameTime.Load()
		if frameNanos >= currentMin || fm.minFrameTime.CompareAndSwap(currentMin, frameNanos) {
			break
		}
	}
	for {
		currentMax := fm.maxFrameTime.Load()
		if frameNanos <= currentMax || fm.maxFrameTime.CompareAndSwap(currentMax, frameNanos) {
			break
		}
	}

	// Check if we need to recalculate FPS
	now := time.Now().UnixNano()
	lastUpdate := fm.lastUpdate.Load()
	elapsed := time.Duration(now - lastUpdate)

	if elapsed >= fm.updatePeriod {
		if fm.lastUpdate.CompareAndSwap(lastUpdate, now) {
			// Calculate FPS based on frames in this period
			frames := fm.frameCount.Swap(0)
			if elapsed > 0 {
				fps := float64(frames) / elapsed.Seconds()
				fm.lastFPS.Store(int64(fps * 1000)) // Store with 3 decimal precision
			}
		}
	}
}

// FPS returns the current frames per second.
func (fm *FrameMetrics) FPS() float64 {
	return float64(fm.lastFPS.Load()) / 1000.0
}

// LastFrameTime returns the duration of the last frame.
func (fm *FrameMetrics) LastFrameTime() time.Duration {
	return time.Duration(fm.lastFrameTime.Load())
}

// MinFrameTime returns the minimum frame time recorded.
func (fm *FrameMetrics) MinFrameTime() time.Duration {
	return time.Duration(fm.minFrameTime.Load())
}

// MaxFrameTime returns the maximum frame time recorded.
func (fm *FrameMetrics) MaxFrameTime() time.Duration {
	return time.Duration(fm.maxFrameTime.Load())
}

// AverageFrameTime returns the average frame time.
func (fm *FrameMetrics) AverageFrameTime() time.Duration {
	total := fm.totalTime.Load()
	count := fm.frameCount.Load()
	if count == 0 {
		return 0
	}
	return time.Duration(total / count)
}

// Reset clears all metrics to their initial state.
func (fm *FrameMetrics) Reset() {
	fm.frameCount.Store(0)
	fm.lastFPS.Store(0)
	fm.lastFrameTime.Store(0)
	fm.minFrameTime.Store(int64(time.Hour))
	fm.maxFrameTime.Store(0)
	fm.totalTime.Store(0)
	fm.lastUpdate.Store(time.Now().UnixNano())
}

// DrawOptionsPool provides pooling for ebiten.DrawImageOptions to reduce allocations.
// This is particularly useful in high-frequency draw operations.
type DrawOptionsPool struct {
	pool sync.Pool
}

// NewDrawOptionsPool creates a new DrawOptionsPool.
func NewDrawOptionsPool() *DrawOptionsPool {
	return &DrawOptionsPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &ebiten.DrawImageOptions{}
			},
		},
	}
}

// Get retrieves a DrawImageOptions from the pool.
// The returned options are reset to default values.
func (p *DrawOptionsPool) Get() *ebiten.DrawImageOptions {
	op := p.pool.Get().(*ebiten.DrawImageOptions)
	// Reset to default state
	op.GeoM.Reset()
	op.ColorScale.Reset()
	op.Blend = ebiten.BlendSourceOver
	op.Filter = ebiten.FilterNearest
	return op
}

// Put returns a DrawImageOptions to the pool.
func (p *DrawOptionsPool) Put(op *ebiten.DrawImageOptions) {
	if op != nil {
		p.pool.Put(op)
	}
}

// VertexPool provides pooling for vertex slices used in triangle drawing.
type VertexPool struct {
	pool sync.Pool
}

// NewVertexPool creates a new VertexPool.
func NewVertexPool() *VertexPool {
	return &VertexPool{
		pool: sync.Pool{
			New: func() interface{} {
				// Pre-allocate with common size (4 vertices for a quad)
				return make([]ebiten.Vertex, 0, 4)
			},
		},
	}
}

// Get retrieves a vertex slice from the pool.
// The returned slice has length 0 but may have capacity > 0.
func (p *VertexPool) Get() []ebiten.Vertex {
	return p.pool.Get().([]ebiten.Vertex)[:0]
}

// Put returns a vertex slice to the pool.
func (p *VertexPool) Put(vertices []ebiten.Vertex) {
	if vertices != nil {
		p.pool.Put(vertices[:0])
	}
}

// IndexPool provides pooling for index slices used in triangle drawing.
type IndexPool struct {
	pool sync.Pool
}

// NewIndexPool creates a new IndexPool.
func NewIndexPool() *IndexPool {
	return &IndexPool{
		pool: sync.Pool{
			New: func() interface{} {
				// Pre-allocate with common size (6 indices for 2 triangles)
				return make([]uint16, 0, 6)
			},
		},
	}
}

// Get retrieves an index slice from the pool.
// The returned slice has length 0 but may have capacity > 0.
func (p *IndexPool) Get() []uint16 {
	return p.pool.Get().([]uint16)[:0]
}

// Put returns an index slice to the pool.
func (p *IndexPool) Put(indices []uint16) {
	if indices != nil {
		p.pool.Put(indices[:0])
	}
}

// DirtyRegion represents a rectangular area that needs redrawing.
type DirtyRegion struct {
	X, Y          float64
	Width, Height float64
}

// Contains returns true if the region contains the given point.
func (r DirtyRegion) Contains(x, y float64) bool {
	return x >= r.X && x < r.X+r.Width && y >= r.Y && y < r.Y+r.Height
}

// Intersects returns true if this region overlaps with another.
func (r DirtyRegion) Intersects(other DirtyRegion) bool {
	return r.X < other.X+other.Width &&
		r.X+r.Width > other.X &&
		r.Y < other.Y+other.Height &&
		r.Y+r.Height > other.Y
}

// Union returns a region that encompasses both regions.
func (r DirtyRegion) Union(other DirtyRegion) DirtyRegion {
	minX := r.X
	if other.X < minX {
		minX = other.X
	}
	minY := r.Y
	if other.Y < minY {
		minY = other.Y
	}
	maxX := r.X + r.Width
	if other.X+other.Width > maxX {
		maxX = other.X + other.Width
	}
	maxY := r.Y + r.Height
	if other.Y+other.Height > maxY {
		maxY = other.Y + other.Height
	}
	return DirtyRegion{
		X:      minX,
		Y:      minY,
		Width:  maxX - minX,
		Height: maxY - minY,
	}
}

// DirtyTracker tracks regions of the screen that need redrawing.
// This enables partial screen updates for improved performance when
// only portions of the display change between frames.
type DirtyTracker struct {
	regions     []DirtyRegion
	fullRedraw  bool
	screenW     int
	screenH     int
	mergeThresh float64 // Threshold for merging regions (0.0-1.0)
	mu          sync.Mutex
}

// NewDirtyTracker creates a new DirtyTracker for the given screen dimensions.
func NewDirtyTracker(screenWidth, screenHeight int) *DirtyTracker {
	return &DirtyTracker{
		regions:     make([]DirtyRegion, 0, 16),
		screenW:     screenWidth,
		screenH:     screenHeight,
		mergeThresh: 0.5, // Merge if overlap is > 50%
	}
}

// MarkDirty marks a region as needing redraw.
func (dt *DirtyTracker) MarkDirty(region DirtyRegion) {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	// Clamp to screen bounds
	if region.X < 0 {
		region.Width += region.X
		region.X = 0
	}
	if region.Y < 0 {
		region.Height += region.Y
		region.Y = 0
	}
	if region.X+region.Width > float64(dt.screenW) {
		region.Width = float64(dt.screenW) - region.X
	}
	if region.Y+region.Height > float64(dt.screenH) {
		region.Height = float64(dt.screenH) - region.Y
	}

	// Skip empty regions
	if region.Width <= 0 || region.Height <= 0 {
		return
	}

	// Try to merge with existing regions
	for i := range dt.regions {
		if dt.regions[i].Intersects(region) {
			dt.regions[i] = dt.regions[i].Union(region)
			return
		}
	}

	dt.regions = append(dt.regions, region)
}

// MarkFullRedraw marks the entire screen as needing redraw.
func (dt *DirtyTracker) MarkFullRedraw() {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.fullRedraw = true
	dt.regions = dt.regions[:0]
}

// NeedsFullRedraw returns true if the entire screen needs redrawing.
func (dt *DirtyTracker) NeedsFullRedraw() bool {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	return dt.fullRedraw
}

// DirtyRegions returns a copy of the current dirty regions.
func (dt *DirtyTracker) DirtyRegions() []DirtyRegion {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	if dt.fullRedraw {
		return []DirtyRegion{{
			X: 0, Y: 0,
			Width:  float64(dt.screenW),
			Height: float64(dt.screenH),
		}}
	}

	result := make([]DirtyRegion, len(dt.regions))
	copy(result, dt.regions)
	return result
}

// Clear resets the dirty tracker for the next frame.
func (dt *DirtyTracker) Clear() {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.fullRedraw = false
	dt.regions = dt.regions[:0]
}

// IsEmpty returns true if no regions are marked dirty.
func (dt *DirtyTracker) IsEmpty() bool {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	return !dt.fullRedraw && len(dt.regions) == 0
}

// RegionCount returns the number of dirty regions.
func (dt *DirtyTracker) RegionCount() int {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	if dt.fullRedraw {
		return 1
	}
	return len(dt.regions)
}

// SetScreenSize updates the screen dimensions.
func (dt *DirtyTracker) SetScreenSize(width, height int) {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.screenW = width
	dt.screenH = height
	dt.fullRedraw = true
}

// RenderStats holds rendering performance statistics.
type RenderStats struct {
	DrawCalls     atomic.Int64
	VertexCount   atomic.Int64
	TextDrawCalls atomic.Int64
	LastResetTime atomic.Int64
}

// NewRenderStats creates a new RenderStats instance.
func NewRenderStats() *RenderStats {
	rs := &RenderStats{}
	rs.LastResetTime.Store(time.Now().UnixNano())
	return rs
}

// RecordDrawCall records a draw call with its vertex count.
func (rs *RenderStats) RecordDrawCall(vertices int) {
	rs.DrawCalls.Add(1)
	rs.VertexCount.Add(int64(vertices))
}

// RecordTextDraw records a text drawing operation.
func (rs *RenderStats) RecordTextDraw() {
	rs.TextDrawCalls.Add(1)
}

// Stats returns the current statistics.
func (rs *RenderStats) Stats() (drawCalls, vertices, textDraws int64) {
	return rs.DrawCalls.Load(), rs.VertexCount.Load(), rs.TextDrawCalls.Load()
}

// Reset clears all statistics.
func (rs *RenderStats) Reset() {
	rs.DrawCalls.Store(0)
	rs.VertexCount.Store(0)
	rs.TextDrawCalls.Store(0)
	rs.LastResetTime.Store(time.Now().UnixNano())
}

// TimeSinceReset returns the duration since the last reset.
func (rs *RenderStats) TimeSinceReset() time.Duration {
	return time.Since(time.Unix(0, rs.LastResetTime.Load()))
}

// PerformanceConfig holds performance tuning settings.
type PerformanceConfig struct {
	// TargetFPS is the target frame rate (default: 60).
	TargetFPS int
	// EnableDirtyTracking enables partial screen updates.
	EnableDirtyTracking bool
	// EnableObjectPooling enables object pooling for allocations.
	EnableObjectPooling bool
	// EnableMetrics enables frame timing metrics collection.
	EnableMetrics bool
	// MetricsUpdatePeriod is how often FPS is recalculated.
	MetricsUpdatePeriod time.Duration
}

// DefaultPerformanceConfig returns a PerformanceConfig with sensible defaults.
func DefaultPerformanceConfig() PerformanceConfig {
	return PerformanceConfig{
		TargetFPS:           60,
		EnableDirtyTracking: true,
		EnableObjectPooling: true,
		EnableMetrics:       true,
		MetricsUpdatePeriod: time.Second,
	}
}

// PerformanceManager coordinates performance optimization features.
type PerformanceManager struct {
	config         PerformanceConfig
	metrics        *FrameMetrics
	stats          *RenderStats
	dirtyTracker   *DirtyTracker
	drawOptionsPool *DrawOptionsPool
	vertexPool     *VertexPool
	indexPool      *IndexPool
	mu             sync.RWMutex
}

// NewPerformanceManager creates a new PerformanceManager with the given config.
func NewPerformanceManager(config PerformanceConfig, screenWidth, screenHeight int) *PerformanceManager {
	pm := &PerformanceManager{
		config: config,
	}

	if config.EnableMetrics {
		pm.metrics = NewFrameMetrics(config.MetricsUpdatePeriod)
	}

	pm.stats = NewRenderStats()

	if config.EnableDirtyTracking {
		pm.dirtyTracker = NewDirtyTracker(screenWidth, screenHeight)
	}

	if config.EnableObjectPooling {
		pm.drawOptionsPool = NewDrawOptionsPool()
		pm.vertexPool = NewVertexPool()
		pm.indexPool = NewIndexPool()
	}

	return pm
}

// RecordFrame records a frame for metrics tracking.
func (pm *PerformanceManager) RecordFrame(frameTime time.Duration) {
	if pm.metrics != nil {
		pm.metrics.RecordFrame(frameTime)
	}
}

// Metrics returns the frame metrics, or nil if disabled.
func (pm *PerformanceManager) Metrics() *FrameMetrics {
	return pm.metrics
}

// Stats returns the render statistics.
func (pm *PerformanceManager) Stats() *RenderStats {
	return pm.stats
}

// DirtyTracker returns the dirty region tracker, or nil if disabled.
func (pm *PerformanceManager) DirtyTracker() *DirtyTracker {
	return pm.dirtyTracker
}

// GetDrawOptions retrieves a DrawImageOptions from the pool.
// If pooling is disabled, returns a new instance.
func (pm *PerformanceManager) GetDrawOptions() *ebiten.DrawImageOptions {
	if pm.drawOptionsPool != nil {
		return pm.drawOptionsPool.Get()
	}
	return &ebiten.DrawImageOptions{}
}

// PutDrawOptions returns a DrawImageOptions to the pool.
// Does nothing if pooling is disabled.
func (pm *PerformanceManager) PutDrawOptions(op *ebiten.DrawImageOptions) {
	if pm.drawOptionsPool != nil {
		pm.drawOptionsPool.Put(op)
	}
}

// GetVertices retrieves a vertex slice from the pool.
func (pm *PerformanceManager) GetVertices() []ebiten.Vertex {
	if pm.vertexPool != nil {
		return pm.vertexPool.Get()
	}
	return make([]ebiten.Vertex, 0, 4)
}

// PutVertices returns a vertex slice to the pool.
func (pm *PerformanceManager) PutVertices(vertices []ebiten.Vertex) {
	if pm.vertexPool != nil {
		pm.vertexPool.Put(vertices)
	}
}

// GetIndices retrieves an index slice from the pool.
func (pm *PerformanceManager) GetIndices() []uint16 {
	if pm.indexPool != nil {
		return pm.indexPool.Get()
	}
	return make([]uint16, 0, 6)
}

// PutIndices returns an index slice to the pool.
func (pm *PerformanceManager) PutIndices(indices []uint16) {
	if pm.indexPool != nil {
		pm.indexPool.Put(indices)
	}
}

// Config returns the current performance configuration.
func (pm *PerformanceManager) Config() PerformanceConfig {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.config
}

// SetScreenSize updates the screen size for dirty tracking.
func (pm *PerformanceManager) SetScreenSize(width, height int) {
	if pm.dirtyTracker != nil {
		pm.dirtyTracker.SetScreenSize(width, height)
	}
}

// ResetStats resets all performance statistics.
func (pm *PerformanceManager) ResetStats() {
	if pm.metrics != nil {
		pm.metrics.Reset()
	}
	if pm.stats != nil {
		pm.stats.Reset()
	}
}
