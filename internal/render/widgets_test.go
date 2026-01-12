//go:build !noebiten

package render

import (
	"image/color"
	"math"
	"sync"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestDefaultWidgetStyle(t *testing.T) {
	style := DefaultWidgetStyle()

	if style.FillColor.A == 0 {
		t.Error("FillColor should have non-zero alpha")
	}
	if style.BackgroundColor.A == 0 {
		t.Error("BackgroundColor should have non-zero alpha")
	}
	if style.BorderColor.A == 0 {
		t.Error("BorderColor should have non-zero alpha")
	}
	if style.BorderWidth <= 0 {
		t.Error("BorderWidth should be positive")
	}
	if !style.ShowBorder {
		t.Error("ShowBorder should be true by default")
	}
	if !style.ShowBackground {
		t.Error("ShowBackground should be true by default")
	}
}

// ProgressBar tests

func TestNewProgressBar(t *testing.T) {
	pb := NewProgressBar(10, 20, 100, 25)

	if pb.x != 10 || pb.y != 20 {
		t.Errorf("position = (%v, %v), want (10, 20)", pb.x, pb.y)
	}
	if pb.width != 100 || pb.height != 25 {
		t.Errorf("size = (%v, %v), want (100, 25)", pb.width, pb.height)
	}
	if pb.value != 0 {
		t.Errorf("initial value = %v, want 0", pb.value)
	}
	if pb.minValue != 0 || pb.maxValue != 100 {
		t.Errorf("range = (%v, %v), want (0, 100)", pb.minValue, pb.maxValue)
	}
	if pb.vertical {
		t.Error("vertical should be false by default")
	}
	if pb.reversed {
		t.Error("reversed should be false by default")
	}
}

func TestProgressBarSetStyle(t *testing.T) {
	pb := NewProgressBar(0, 0, 100, 20)
	style := WidgetStyle{
		FillColor:   color.RGBA{R: 255, G: 0, B: 0, A: 255},
		BorderWidth: 2.0,
	}
	pb.SetStyle(style)

	pb.mu.RLock()
	defer pb.mu.RUnlock()
	if pb.style.FillColor.R != 255 {
		t.Error("style was not set correctly")
	}
	if pb.style.BorderWidth != 2.0 {
		t.Error("BorderWidth was not set correctly")
	}
}

func TestProgressBarSetPosition(t *testing.T) {
	pb := NewProgressBar(0, 0, 100, 20)
	pb.SetPosition(50, 60)

	pb.mu.RLock()
	defer pb.mu.RUnlock()
	if pb.x != 50 || pb.y != 60 {
		t.Errorf("position = (%v, %v), want (50, 60)", pb.x, pb.y)
	}
}

func TestProgressBarSetSize(t *testing.T) {
	pb := NewProgressBar(0, 0, 100, 20)
	pb.SetSize(200, 40)

	pb.mu.RLock()
	defer pb.mu.RUnlock()
	if pb.width != 200 || pb.height != 40 {
		t.Errorf("size = (%v, %v), want (200, 40)", pb.width, pb.height)
	}
}

func TestProgressBarSetValue(t *testing.T) {
	pb := NewProgressBar(0, 0, 100, 20)
	pb.SetValue(75)

	if pb.Value() != 75 {
		t.Errorf("value = %v, want 75", pb.Value())
	}
}

func TestProgressBarSetRange(t *testing.T) {
	pb := NewProgressBar(0, 0, 100, 20)
	pb.SetRange(10, 90)

	pb.mu.RLock()
	defer pb.mu.RUnlock()
	if pb.minValue != 10 || pb.maxValue != 90 {
		t.Errorf("range = (%v, %v), want (10, 90)", pb.minValue, pb.maxValue)
	}
}

func TestProgressBarSetRangeInvalid(t *testing.T) {
	pb := NewProgressBar(0, 0, 100, 20)

	// Test with min > max (should swap)
	pb.SetRange(90, 10)
	pb.mu.RLock()
	if pb.minValue != 10 || pb.maxValue != 90 {
		t.Errorf("range = (%v, %v), want (10, 90) after swap", pb.minValue, pb.maxValue)
	}
	pb.mu.RUnlock()

	// Test with min == max (should add 1 to max)
	pb.SetRange(50, 50)
	pb.mu.RLock()
	if pb.minValue != 50 || pb.maxValue != 51 {
		t.Errorf("range = (%v, %v), want (50, 51) for equal values", pb.minValue, pb.maxValue)
	}
	pb.mu.RUnlock()
}

func TestProgressBarSetVertical(t *testing.T) {
	pb := NewProgressBar(0, 0, 100, 100)
	pb.SetVertical(true)

	pb.mu.RLock()
	defer pb.mu.RUnlock()
	if !pb.vertical {
		t.Error("vertical should be true")
	}
}

func TestProgressBarSetReversed(t *testing.T) {
	pb := NewProgressBar(0, 0, 100, 20)
	pb.SetReversed(true)

	pb.mu.RLock()
	defer pb.mu.RUnlock()
	if !pb.reversed {
		t.Error("reversed should be true")
	}
}

func TestProgressBarPercentage(t *testing.T) {
	tests := []struct {
		name     string
		value    float64
		minVal   float64
		maxVal   float64
		expected float64
	}{
		{"zero value", 0, 0, 100, 0},
		{"max value", 100, 0, 100, 100},
		{"mid value", 50, 0, 100, 50},
		{"custom range", 75, 50, 100, 50},
		{"below min", -10, 0, 100, 0},
		{"above max", 150, 0, 100, 100},
		{"negative range", -25, -50, 0, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pb := NewProgressBar(0, 0, 100, 20)
			pb.SetRange(tt.minVal, tt.maxVal)
			pb.SetValue(tt.value)

			got := pb.Percentage()
			if got != tt.expected {
				t.Errorf("Percentage() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestProgressBarPercentageZeroRange(t *testing.T) {
	pb := NewProgressBar(0, 0, 100, 20)
	pb.mu.Lock()
	pb.minValue = 50
	pb.maxValue = 50 // Force zero range
	pb.mu.Unlock()

	if pb.Percentage() != 0 {
		t.Errorf("Percentage() with zero range = %v, want 0", pb.Percentage())
	}
}

func TestProgressBarDrawHorizontal(t *testing.T) {
	pb := NewProgressBar(10, 10, 100, 20)
	pb.SetValue(50)

	screen := ebiten.NewImage(200, 100)

	// Should not panic
	pb.Draw(screen)

	// Test with zero value
	pb.SetValue(0)
	pb.Draw(screen)

	// Test with full value
	pb.SetValue(100)
	pb.Draw(screen)
}

func TestProgressBarDrawVertical(t *testing.T) {
	pb := NewProgressBar(10, 10, 20, 100)
	pb.SetVertical(true)
	pb.SetValue(75)

	screen := ebiten.NewImage(100, 200)

	// Should not panic
	pb.Draw(screen)
}

func TestProgressBarDrawReversed(t *testing.T) {
	screen := ebiten.NewImage(200, 200)

	// Horizontal reversed
	pb := NewProgressBar(10, 10, 100, 20)
	pb.SetReversed(true)
	pb.SetValue(50)
	pb.Draw(screen)

	// Vertical reversed
	pb2 := NewProgressBar(10, 50, 20, 100)
	pb2.SetVertical(true)
	pb2.SetReversed(true)
	pb2.SetValue(50)
	pb2.Draw(screen)
}

func TestProgressBarDrawNoBorder(t *testing.T) {
	pb := NewProgressBar(10, 10, 100, 20)
	pb.SetValue(50)
	style := pb.style
	style.ShowBorder = false
	pb.SetStyle(style)

	screen := ebiten.NewImage(200, 100)
	pb.Draw(screen)
}

func TestProgressBarDrawNoBackground(t *testing.T) {
	pb := NewProgressBar(10, 10, 100, 20)
	pb.SetValue(50)
	style := pb.style
	style.ShowBackground = false
	pb.SetStyle(style)

	screen := ebiten.NewImage(200, 100)
	pb.Draw(screen)
}

// Gauge tests

func TestNewGauge(t *testing.T) {
	g := NewGauge(100, 100, 50)

	if g.x != 100 || g.y != 100 {
		t.Errorf("position = (%v, %v), want (100, 100)", g.x, g.y)
	}
	if g.radius != 50 {
		t.Errorf("radius = %v, want 50", g.radius)
	}
	if g.value != 0 {
		t.Errorf("initial value = %v, want 0", g.value)
	}
	if g.minValue != 0 || g.maxValue != 100 {
		t.Errorf("range = (%v, %v), want (0, 100)", g.minValue, g.maxValue)
	}
	if g.thickness != 10 {
		t.Errorf("thickness = %v, want 10", g.thickness)
	}
	if !g.clockwise {
		t.Error("clockwise should be true by default")
	}
}

func TestGaugeSetStyle(t *testing.T) {
	g := NewGauge(50, 50, 30)
	style := WidgetStyle{
		FillColor:   color.RGBA{R: 0, G: 255, B: 0, A: 255},
		BorderWidth: 3.0,
	}
	g.SetStyle(style)

	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.style.FillColor.G != 255 {
		t.Error("style was not set correctly")
	}
}

func TestGaugeSetPosition(t *testing.T) {
	g := NewGauge(0, 0, 50)
	g.SetPosition(75, 85)

	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.x != 75 || g.y != 85 {
		t.Errorf("position = (%v, %v), want (75, 85)", g.x, g.y)
	}
}

func TestGaugeSetRadius(t *testing.T) {
	g := NewGauge(50, 50, 30)
	g.SetRadius(60)

	g.mu.RLock()
	if g.radius != 60 {
		t.Errorf("radius = %v, want 60", g.radius)
	}
	g.mu.RUnlock()

	// Should ignore non-positive values
	g.SetRadius(0)
	g.mu.RLock()
	if g.radius != 60 {
		t.Error("radius should not change for zero value")
	}
	g.mu.RUnlock()

	g.SetRadius(-10)
	g.mu.RLock()
	if g.radius != 60 {
		t.Error("radius should not change for negative value")
	}
	g.mu.RUnlock()
}

func TestGaugeSetValue(t *testing.T) {
	g := NewGauge(50, 50, 30)
	g.SetValue(80)

	if g.Value() != 80 {
		t.Errorf("value = %v, want 80", g.Value())
	}
}

func TestGaugeSetRange(t *testing.T) {
	g := NewGauge(50, 50, 30)
	g.SetRange(20, 80)

	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.minValue != 20 || g.maxValue != 80 {
		t.Errorf("range = (%v, %v), want (20, 80)", g.minValue, g.maxValue)
	}
}

func TestGaugeSetRangeInvalid(t *testing.T) {
	g := NewGauge(50, 50, 30)

	// Test with min > max (should swap)
	g.SetRange(100, 0)
	g.mu.RLock()
	if g.minValue != 0 || g.maxValue != 100 {
		t.Errorf("range = (%v, %v), want (0, 100) after swap", g.minValue, g.maxValue)
	}
	g.mu.RUnlock()

	// Test with min == max (should add 1 to max)
	g.SetRange(25, 25)
	g.mu.RLock()
	if g.minValue != 25 || g.maxValue != 26 {
		t.Errorf("range = (%v, %v), want (25, 26) for equal values", g.minValue, g.maxValue)
	}
	g.mu.RUnlock()
}

func TestGaugeSetAngles(t *testing.T) {
	g := NewGauge(50, 50, 30)
	g.SetAngles(0, math.Pi)

	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.startAngle != 0 || g.endAngle != math.Pi {
		t.Errorf("angles = (%v, %v), want (0, π)", g.startAngle, g.endAngle)
	}
}

func TestGaugeSetThickness(t *testing.T) {
	g := NewGauge(50, 50, 30)
	g.SetThickness(15)

	g.mu.RLock()
	if g.thickness != 15 {
		t.Errorf("thickness = %v, want 15", g.thickness)
	}
	g.mu.RUnlock()

	// Should ignore non-positive values
	g.SetThickness(0)
	g.mu.RLock()
	if g.thickness != 15 {
		t.Error("thickness should not change for zero value")
	}
	g.mu.RUnlock()

	g.SetThickness(-5)
	g.mu.RLock()
	if g.thickness != 15 {
		t.Error("thickness should not change for negative value")
	}
	g.mu.RUnlock()
}

func TestGaugeSetClockwise(t *testing.T) {
	g := NewGauge(50, 50, 30)
	g.SetClockwise(false)

	g.mu.RLock()
	defer g.mu.RUnlock()
	if g.clockwise {
		t.Error("clockwise should be false")
	}
}

func TestGaugePercentage(t *testing.T) {
	tests := []struct {
		name     string
		value    float64
		minVal   float64
		maxVal   float64
		expected float64
	}{
		{"zero value", 0, 0, 100, 0},
		{"max value", 100, 0, 100, 100},
		{"mid value", 50, 0, 100, 50},
		{"custom range", 75, 50, 100, 50},
		{"below min", -10, 0, 100, 0},
		{"above max", 150, 0, 100, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewGauge(50, 50, 30)
			g.SetRange(tt.minVal, tt.maxVal)
			g.SetValue(tt.value)

			got := g.Percentage()
			if got != tt.expected {
				t.Errorf("Percentage() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGaugePercentageZeroRange(t *testing.T) {
	g := NewGauge(50, 50, 30)
	g.mu.Lock()
	g.minValue = 50
	g.maxValue = 50 // Force zero range
	g.mu.Unlock()

	if g.Percentage() != 0 {
		t.Errorf("Percentage() with zero range = %v, want 0", g.Percentage())
	}
}

func TestGaugeDraw(t *testing.T) {
	g := NewGauge(100, 100, 50)
	g.SetValue(75)

	screen := ebiten.NewImage(200, 200)

	// Should not panic
	g.Draw(screen)

	// Test with zero value
	g.SetValue(0)
	g.Draw(screen)

	// Test with full value
	g.SetValue(100)
	g.Draw(screen)
}

func TestGaugeDrawCounterClockwise(t *testing.T) {
	g := NewGauge(100, 100, 50)
	g.SetClockwise(false)
	g.SetValue(50)

	screen := ebiten.NewImage(200, 200)

	// Should not panic
	g.Draw(screen)
}

func TestGaugeDrawNoBackground(t *testing.T) {
	g := NewGauge(100, 100, 50)
	g.SetValue(50)
	style := g.style
	style.ShowBackground = false
	g.SetStyle(style)

	screen := ebiten.NewImage(200, 200)
	g.Draw(screen)
}

func TestGaugeDrawZeroRadius(t *testing.T) {
	g := NewGauge(100, 100, 50)
	g.mu.Lock()
	g.radius = 0 // Force zero radius
	g.mu.Unlock()
	g.SetValue(50)

	screen := ebiten.NewImage(200, 200)

	// Should not panic (early return)
	g.Draw(screen)
}

func TestGaugeDrawZeroThickness(t *testing.T) {
	g := NewGauge(100, 100, 50)
	g.mu.Lock()
	g.thickness = 0 // Force zero thickness
	g.mu.Unlock()
	g.SetValue(50)

	screen := ebiten.NewImage(200, 200)

	// Should not panic (early return)
	g.Draw(screen)
}

func TestGaugeDrawFullCircle(t *testing.T) {
	g := NewGauge(100, 100, 50)
	g.SetAngles(0, 2*math.Pi)
	g.SetValue(75)

	screen := ebiten.NewImage(200, 200)

	// Should not panic
	g.Draw(screen)
}

func TestGaugeDrawSmallArc(t *testing.T) {
	g := NewGauge(100, 100, 50)
	g.SetAngles(0, math.Pi/4) // 45 degree arc
	g.SetValue(50)

	screen := ebiten.NewImage(200, 200)

	// Should not panic
	g.Draw(screen)
}

func TestGaugeDrawThickArc(t *testing.T) {
	g := NewGauge(100, 100, 50)
	g.SetThickness(60) // Thicker than radius (creates filled circle effect)
	g.SetValue(50)

	screen := ebiten.NewImage(200, 200)

	// Should not panic
	g.Draw(screen)
}

// Concurrent access tests

func TestProgressBarConcurrentAccess(t *testing.T) {
	pb := NewProgressBar(0, 0, 100, 20)
	screen := ebiten.NewImage(200, 100)

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			pb.SetValue(float64(i))
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			pb.Draw(screen)
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = pb.Percentage()
			_ = pb.Value()
		}
	}()

	wg.Wait()
}

func TestGaugeConcurrentAccess(t *testing.T) {
	g := NewGauge(100, 100, 50)
	screen := ebiten.NewImage(200, 200)

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			g.SetValue(float64(i))
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			g.Draw(screen)
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = g.Percentage()
			_ = g.Value()
		}
	}()

	wg.Wait()
}

// Edge case tests

func TestProgressBarValueClamping(t *testing.T) {
	pb := NewProgressBar(0, 0, 100, 20)
	pb.SetRange(0, 100)

	// Value below minimum
	pb.SetValue(-50)
	pct := pb.Percentage()
	if pct != 0 {
		t.Errorf("Percentage for value below min = %v, want 0", pct)
	}

	// Value above maximum
	pb.SetValue(200)
	pct = pb.Percentage()
	if pct != 100 {
		t.Errorf("Percentage for value above max = %v, want 100", pct)
	}
}

func TestGaugeValueClamping(t *testing.T) {
	g := NewGauge(50, 50, 30)
	g.SetRange(0, 100)

	// Value below minimum
	g.SetValue(-50)
	pct := g.Percentage()
	if pct != 0 {
		t.Errorf("Percentage for value below min = %v, want 0", pct)
	}

	// Value above maximum
	g.SetValue(200)
	pct = g.Percentage()
	if pct != 100 {
		t.Errorf("Percentage for value above max = %v, want 100", pct)
	}
}

// Interface compliance tests

func TestProgressBarImplementsWidget(t *testing.T) {
	var _ Widget = (*ProgressBar)(nil)
}

func TestGaugeImplementsWidget(t *testing.T) {
	var _ Widget = (*Gauge)(nil)
}
