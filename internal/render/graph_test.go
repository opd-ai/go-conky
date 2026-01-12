//go:build !noebiten

package render

import (
	"image/color"
	"sync"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestDefaultGraphStyle(t *testing.T) {
	style := DefaultGraphStyle()

	if style.FillColor.A == 0 {
		t.Error("FillColor should have non-zero alpha")
	}
	if style.StrokeColor.A == 0 {
		t.Error("StrokeColor should have non-zero alpha")
	}
	if style.StrokeWidth <= 0 {
		t.Error("StrokeWidth should be positive")
	}
	if !style.ShowBackground {
		t.Error("ShowBackground should be true by default")
	}
}

// LineGraph tests

func TestNewLineGraph(t *testing.T) {
	lg := NewLineGraph(10, 20, 100, 50)

	if lg.x != 10 || lg.y != 20 {
		t.Errorf("position = (%v, %v), want (10, 20)", lg.x, lg.y)
	}
	if lg.width != 100 || lg.height != 50 {
		t.Errorf("size = (%v, %v), want (100, 50)", lg.width, lg.height)
	}
	if lg.maxPoints != 100 {
		t.Errorf("maxPoints = %d, want 100", lg.maxPoints)
	}
	if !lg.autoScale {
		t.Error("autoScale should be true by default")
	}
}

func TestLineGraphSetStyle(t *testing.T) {
	lg := NewLineGraph(0, 0, 100, 100)
	style := GraphStyle{
		FillColor:   color.RGBA{R: 255, G: 0, B: 0, A: 255},
		StrokeWidth: 2.0,
	}
	lg.SetStyle(style)

	lg.mu.RLock()
	defer lg.mu.RUnlock()
	if lg.style.FillColor.R != 255 {
		t.Error("style was not set correctly")
	}
}

func TestLineGraphSetPosition(t *testing.T) {
	lg := NewLineGraph(0, 0, 100, 100)
	lg.SetPosition(50, 60)

	lg.mu.RLock()
	defer lg.mu.RUnlock()
	if lg.x != 50 || lg.y != 60 {
		t.Errorf("position = (%v, %v), want (50, 60)", lg.x, lg.y)
	}
}

func TestLineGraphSetSize(t *testing.T) {
	lg := NewLineGraph(0, 0, 100, 100)
	lg.SetSize(200, 150)

	lg.mu.RLock()
	defer lg.mu.RUnlock()
	if lg.width != 200 || lg.height != 150 {
		t.Errorf("size = (%v, %v), want (200, 150)", lg.width, lg.height)
	}
}

func TestLineGraphSetMaxPoints(t *testing.T) {
	lg := NewLineGraph(0, 0, 100, 100)
	lg.SetMaxPoints(50)

	lg.mu.RLock()
	if lg.maxPoints != 50 {
		t.Errorf("maxPoints = %d, want 50", lg.maxPoints)
	}
	lg.mu.RUnlock()

	// Should ignore non-positive values
	lg.SetMaxPoints(0)
	lg.mu.RLock()
	if lg.maxPoints != 50 {
		t.Error("maxPoints should not change for zero value")
	}
	lg.mu.RUnlock()
}

func TestLineGraphSetRange(t *testing.T) {
	lg := NewLineGraph(0, 0, 100, 100)
	lg.SetRange(10, 90)

	lg.mu.RLock()
	defer lg.mu.RUnlock()
	if lg.minValue != 10 || lg.maxValue != 90 {
		t.Errorf("range = (%v, %v), want (10, 90)", lg.minValue, lg.maxValue)
	}
	if lg.autoScale {
		t.Error("autoScale should be false after SetRange")
	}
}

func TestLineGraphSetAutoScale(t *testing.T) {
	lg := NewLineGraph(0, 0, 100, 100)
	lg.SetAutoScale(false)

	lg.mu.RLock()
	if lg.autoScale {
		t.Error("autoScale should be false")
	}
	lg.mu.RUnlock()

	lg.SetAutoScale(true)
	lg.mu.RLock()
	if !lg.autoScale {
		t.Error("autoScale should be true")
	}
	lg.mu.RUnlock()
}

func TestLineGraphAddPoint(t *testing.T) {
	lg := NewLineGraph(0, 0, 100, 100)
	lg.SetMaxPoints(5)

	for i := 0; i < 10; i++ {
		lg.AddPoint(float64(i))
	}

	lg.mu.RLock()
	defer lg.mu.RUnlock()
	if len(lg.data) != 5 {
		t.Errorf("data length = %d, want 5", len(lg.data))
	}
	// Should contain last 5 values (5, 6, 7, 8, 9)
	if lg.data[0] != 5 {
		t.Errorf("first value = %v, want 5", lg.data[0])
	}
}

func TestLineGraphSetData(t *testing.T) {
	lg := NewLineGraph(0, 0, 100, 100)
	data := []float64{1, 2, 3, 4, 5}
	lg.SetData(data)

	lg.mu.RLock()
	defer lg.mu.RUnlock()
	if len(lg.data) != 5 {
		t.Errorf("data length = %d, want 5", len(lg.data))
	}

	// Verify data is copied
	data[0] = 999
	if lg.data[0] != 1 {
		t.Error("SetData should copy data, not share reference")
	}
}

func TestLineGraphSetDataWithMaxPoints(t *testing.T) {
	lg := NewLineGraph(0, 0, 100, 100)
	lg.SetMaxPoints(3)
	lg.SetData([]float64{1, 2, 3, 4, 5})

	lg.mu.RLock()
	defer lg.mu.RUnlock()
	if len(lg.data) != 3 {
		t.Errorf("data length = %d, want 3", len(lg.data))
	}
	if lg.data[0] != 3 {
		t.Errorf("first value = %v, want 3", lg.data[0])
	}
}

func TestLineGraphClearData(t *testing.T) {
	lg := NewLineGraph(0, 0, 100, 100)
	lg.SetData([]float64{1, 2, 3})
	lg.ClearData()

	lg.mu.RLock()
	defer lg.mu.RUnlock()
	if len(lg.data) != 0 {
		t.Errorf("data length = %d, want 0", len(lg.data))
	}
}

func TestLineGraphDraw(t *testing.T) {
	lg := NewLineGraph(10, 10, 100, 50)
	lg.SetData([]float64{10, 20, 30, 40, 50})

	screen := ebiten.NewImage(200, 100)

	// Should not panic
	lg.Draw(screen)

	// Test with empty data
	lg.ClearData()
	lg.Draw(screen)

	// Test with single point
	lg.SetData([]float64{50})
	lg.Draw(screen)

	// Test without background
	lg.SetData([]float64{10, 20, 30})
	style := lg.style
	style.ShowBackground = false
	lg.SetStyle(style)
	lg.Draw(screen)
}

func TestLineGraphDrawAutoScale(t *testing.T) {
	lg := NewLineGraph(10, 10, 100, 50)
	lg.SetAutoScale(true)
	lg.SetData([]float64{50, 50, 50}) // All same values

	screen := ebiten.NewImage(200, 100)

	// Should handle zero value range gracefully
	lg.Draw(screen)
}

// BarGraph tests

func TestNewBarGraph(t *testing.T) {
	bg := NewBarGraph(10, 20, 100, 50)

	if bg.x != 10 || bg.y != 20 {
		t.Errorf("position = (%v, %v), want (10, 20)", bg.x, bg.y)
	}
	if bg.width != 100 || bg.height != 50 {
		t.Errorf("size = (%v, %v), want (100, 50)", bg.width, bg.height)
	}
	if bg.horizontal {
		t.Error("horizontal should be false by default")
	}
}

func TestBarGraphSetStyle(t *testing.T) {
	bg := NewBarGraph(0, 0, 100, 100)
	style := GraphStyle{
		FillColor: color.RGBA{R: 0, G: 255, B: 0, A: 255},
	}
	bg.SetStyle(style)

	bg.mu.RLock()
	defer bg.mu.RUnlock()
	if bg.style.FillColor.G != 255 {
		t.Error("style was not set correctly")
	}
}

func TestBarGraphSetPosition(t *testing.T) {
	bg := NewBarGraph(0, 0, 100, 100)
	bg.SetPosition(30, 40)

	bg.mu.RLock()
	defer bg.mu.RUnlock()
	if bg.x != 30 || bg.y != 40 {
		t.Errorf("position = (%v, %v), want (30, 40)", bg.x, bg.y)
	}
}

func TestBarGraphSetSize(t *testing.T) {
	bg := NewBarGraph(0, 0, 100, 100)
	bg.SetSize(150, 75)

	bg.mu.RLock()
	defer bg.mu.RUnlock()
	if bg.width != 150 || bg.height != 75 {
		t.Errorf("size = (%v, %v), want (150, 75)", bg.width, bg.height)
	}
}

func TestBarGraphSetBarSpacing(t *testing.T) {
	bg := NewBarGraph(0, 0, 100, 100)
	bg.SetBarSpacing(5)

	bg.mu.RLock()
	if bg.barSpacing != 5 {
		t.Errorf("barSpacing = %v, want 5", bg.barSpacing)
	}
	bg.mu.RUnlock()

	// Should ignore negative values
	bg.SetBarSpacing(-1)
	bg.mu.RLock()
	if bg.barSpacing != 5 {
		t.Error("barSpacing should not change for negative value")
	}
	bg.mu.RUnlock()
}

func TestBarGraphSetHorizontal(t *testing.T) {
	bg := NewBarGraph(0, 0, 100, 100)
	bg.SetHorizontal(true)

	bg.mu.RLock()
	defer bg.mu.RUnlock()
	if !bg.horizontal {
		t.Error("horizontal should be true")
	}
}

func TestBarGraphSetRange(t *testing.T) {
	bg := NewBarGraph(0, 0, 100, 100)
	bg.SetRange(5, 95)

	bg.mu.RLock()
	defer bg.mu.RUnlock()
	if bg.minValue != 5 || bg.maxValue != 95 {
		t.Errorf("range = (%v, %v), want (5, 95)", bg.minValue, bg.maxValue)
	}
	if bg.autoScale {
		t.Error("autoScale should be false after SetRange")
	}
}

func TestBarGraphSetData(t *testing.T) {
	bg := NewBarGraph(0, 0, 100, 100)
	data := []float64{10, 20, 30}
	bg.SetData(data)

	bg.mu.RLock()
	defer bg.mu.RUnlock()
	if len(bg.data) != 3 {
		t.Errorf("data length = %d, want 3", len(bg.data))
	}

	// Verify data is copied
	data[0] = 999
	if bg.data[0] != 10 {
		t.Error("SetData should copy data, not share reference")
	}
}

func TestBarGraphSetLabels(t *testing.T) {
	bg := NewBarGraph(0, 0, 100, 100)
	labels := []string{"A", "B", "C"}
	bg.SetLabels(labels)

	bg.mu.RLock()
	defer bg.mu.RUnlock()
	if len(bg.labels) != 3 {
		t.Errorf("labels length = %d, want 3", len(bg.labels))
	}

	// Verify labels are copied
	labels[0] = "X"
	if bg.labels[0] != "A" {
		t.Error("SetLabels should copy labels, not share reference")
	}
}

func TestBarGraphClearData(t *testing.T) {
	bg := NewBarGraph(0, 0, 100, 100)
	bg.SetData([]float64{1, 2, 3})
	bg.SetLabels([]string{"A", "B", "C"})
	bg.ClearData()

	bg.mu.RLock()
	defer bg.mu.RUnlock()
	if len(bg.data) != 0 {
		t.Errorf("data length = %d, want 0", len(bg.data))
	}
	if len(bg.labels) != 0 {
		t.Errorf("labels length = %d, want 0", len(bg.labels))
	}
}

func TestBarGraphDrawVertical(t *testing.T) {
	bg := NewBarGraph(10, 10, 100, 50)
	bg.SetData([]float64{25, 50, 75, 100})

	screen := ebiten.NewImage(200, 100)

	// Should not panic
	bg.Draw(screen)

	// Test with empty data
	bg.ClearData()
	bg.Draw(screen)
}

func TestBarGraphDrawHorizontal(t *testing.T) {
	bg := NewBarGraph(10, 10, 100, 50)
	bg.SetHorizontal(true)
	bg.SetData([]float64{25, 50, 75, 100})

	screen := ebiten.NewImage(200, 100)

	// Should not panic
	bg.Draw(screen)
}

func TestBarGraphDrawAutoScale(t *testing.T) {
	bg := NewBarGraph(10, 10, 100, 50)
	bg.SetAutoScale(true)
	bg.SetData([]float64{0, 0, 0}) // All zero values

	screen := ebiten.NewImage(200, 100)

	// Should handle zero max value gracefully
	bg.Draw(screen)
}

func TestBarGraphDrawOutOfRange(t *testing.T) {
	bg := NewBarGraph(10, 10, 100, 50)
	bg.SetRange(0, 50)
	bg.SetData([]float64{-10, 100}) // Values outside range

	screen := ebiten.NewImage(200, 100)

	// Should clamp values
	bg.Draw(screen)
}

// Histogram tests

func TestNewHistogram(t *testing.T) {
	h := NewHistogram(10, 20, 100, 50)

	if h.x != 10 || h.y != 20 {
		t.Errorf("position = (%v, %v), want (10, 20)", h.x, h.y)
	}
	if h.width != 100 || h.height != 50 {
		t.Errorf("size = (%v, %v), want (100, 50)", h.width, h.height)
	}
	if h.binCount != 10 {
		t.Errorf("binCount = %d, want 10", h.binCount)
	}
	if !h.autoRange {
		t.Error("autoRange should be true by default")
	}
}

func TestHistogramSetStyle(t *testing.T) {
	h := NewHistogram(0, 0, 100, 100)
	style := GraphStyle{
		FillColor: color.RGBA{R: 0, G: 0, B: 255, A: 255},
	}
	h.SetStyle(style)

	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.style.FillColor.B != 255 {
		t.Error("style was not set correctly")
	}
}

func TestHistogramSetPosition(t *testing.T) {
	h := NewHistogram(0, 0, 100, 100)
	h.SetPosition(25, 35)

	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.x != 25 || h.y != 35 {
		t.Errorf("position = (%v, %v), want (25, 35)", h.x, h.y)
	}
}

func TestHistogramSetSize(t *testing.T) {
	h := NewHistogram(0, 0, 100, 100)
	h.SetSize(120, 80)

	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.width != 120 || h.height != 80 {
		t.Errorf("size = (%v, %v), want (120, 80)", h.width, h.height)
	}
}

func TestHistogramSetBinCount(t *testing.T) {
	h := NewHistogram(0, 0, 100, 100)
	h.SetBinCount(20)

	h.mu.RLock()
	if h.binCount != 20 {
		t.Errorf("binCount = %d, want 20", h.binCount)
	}
	h.mu.RUnlock()

	// Should ignore non-positive values
	h.SetBinCount(0)
	h.mu.RLock()
	if h.binCount != 20 {
		t.Error("binCount should not change for zero value")
	}
	h.mu.RUnlock()
}

func TestHistogramSetRange(t *testing.T) {
	h := NewHistogram(0, 0, 100, 100)
	h.SetRange(0, 50)

	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.minValue != 0 || h.maxValue != 50 {
		t.Errorf("range = (%v, %v), want (0, 50)", h.minValue, h.maxValue)
	}
	if h.autoRange {
		t.Error("autoRange should be false after SetRange")
	}
}

func TestHistogramSetAutoRange(t *testing.T) {
	h := NewHistogram(0, 0, 100, 100)
	h.SetAutoRange(false)

	h.mu.RLock()
	if h.autoRange {
		t.Error("autoRange should be false")
	}
	h.mu.RUnlock()
}

func TestHistogramSetData(t *testing.T) {
	h := NewHistogram(0, 0, 100, 100)
	data := []float64{1, 2, 3, 4, 5}
	h.SetData(data)

	h.mu.RLock()
	defer h.mu.RUnlock()
	if len(h.data) != 5 {
		t.Errorf("data length = %d, want 5", len(h.data))
	}

	// Verify data is copied
	data[0] = 999
	if h.data[0] != 1 {
		t.Error("SetData should copy data, not share reference")
	}
}

func TestHistogramAddValue(t *testing.T) {
	h := NewHistogram(0, 0, 100, 100)
	h.AddValue(10)
	h.AddValue(20)
	h.AddValue(30)

	h.mu.RLock()
	defer h.mu.RUnlock()
	if len(h.data) != 3 {
		t.Errorf("data length = %d, want 3", len(h.data))
	}
}

func TestHistogramClearData(t *testing.T) {
	h := NewHistogram(0, 0, 100, 100)
	h.SetData([]float64{1, 2, 3})
	h.ClearData()

	h.mu.RLock()
	defer h.mu.RUnlock()
	if len(h.data) != 0 {
		t.Errorf("data length = %d, want 0", len(h.data))
	}
}

func TestHistogramDraw(t *testing.T) {
	h := NewHistogram(10, 10, 100, 50)
	h.SetBinCount(5)
	h.SetData([]float64{10, 15, 20, 25, 30, 35, 40, 45, 50})

	screen := ebiten.NewImage(200, 100)

	// Should not panic
	h.Draw(screen)

	// Test with empty data
	h.ClearData()
	h.Draw(screen)

	// Test without background
	h.SetData([]float64{10, 20, 30})
	style := h.style
	style.ShowBackground = false
	h.SetStyle(style)
	h.Draw(screen)
}

func TestHistogramDrawAutoRange(t *testing.T) {
	h := NewHistogram(10, 10, 100, 50)
	h.SetAutoRange(true)
	h.SetData([]float64{50, 50, 50}) // All same values

	screen := ebiten.NewImage(200, 100)

	// Should handle zero range gracefully
	h.Draw(screen)
}

func TestHistogramDrawZeroBinCount(t *testing.T) {
	h := NewHistogram(10, 10, 100, 50)
	h.mu.Lock()
	h.binCount = 0 // Force zero bin count
	h.mu.Unlock()
	h.SetData([]float64{10, 20, 30})

	screen := ebiten.NewImage(200, 100)

	// Should not panic
	h.Draw(screen)
}

// Interface compliance tests

func TestLineGraphImplementsGraphWidget(t *testing.T) {
	var _ GraphWidget = (*LineGraph)(nil)
}

func TestBarGraphImplementsGraphWidget(t *testing.T) {
	var _ GraphWidget = (*BarGraph)(nil)
}

func TestHistogramImplementsGraphWidget(t *testing.T) {
	var _ GraphWidget = (*Histogram)(nil)
}

// Concurrent access tests

func TestLineGraphConcurrentAccess(t *testing.T) {
	lg := NewLineGraph(0, 0, 100, 100)
	screen := ebiten.NewImage(200, 100)

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			lg.AddPoint(float64(i))
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			lg.Draw(screen)
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			lg.SetStyle(DefaultGraphStyle())
		}
	}()

	wg.Wait()
}

func TestBarGraphConcurrentAccess(t *testing.T) {
	bg := NewBarGraph(0, 0, 100, 100)
	screen := ebiten.NewImage(200, 100)

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			bg.SetData([]float64{float64(i), float64(i + 1)})
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			bg.Draw(screen)
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			bg.SetHorizontal(i%2 == 0)
		}
	}()

	wg.Wait()
}

func TestHistogramConcurrentAccess(t *testing.T) {
	h := NewHistogram(0, 0, 100, 100)
	screen := ebiten.NewImage(200, 100)

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			h.AddValue(float64(i))
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			h.Draw(screen)
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			h.SetBinCount(5 + (i % 10))
		}
	}()

	wg.Wait()
}
