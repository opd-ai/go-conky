// Package render provides Ebiten-based rendering capabilities for conky-go.
package render

import (
	"image/color"
	"math"
	"testing"
)

func TestNewCairoRenderer(t *testing.T) {
	cr := NewCairoRenderer()
	if cr == nil {
		t.Fatal("NewCairoRenderer returned nil")
	}

	// Check default values
	clr := cr.GetCurrentColor()
	if clr.R != 0 || clr.G != 0 || clr.B != 0 || clr.A != 255 {
		t.Errorf("Expected default color black (0,0,0,255), got (%d,%d,%d,%d)",
			clr.R, clr.G, clr.B, clr.A)
	}

	if cr.GetLineWidth() != 1.0 {
		t.Errorf("Expected default line width 1.0, got %f", cr.GetLineWidth())
	}

	if cr.GetLineCap() != LineCapButt {
		t.Errorf("Expected default line cap LineCapButt, got %v", cr.GetLineCap())
	}

	if cr.GetLineJoin() != LineJoinMiter {
		t.Errorf("Expected default line join LineJoinMiter, got %v", cr.GetLineJoin())
	}

	if !cr.GetAntialias() {
		t.Error("Expected antialias to be enabled by default")
	}
}

func TestCairoRenderer_SetSourceRGB(t *testing.T) {
	tests := []struct {
		name     string
		r, g, b  float64
		expected color.RGBA
	}{
		{
			name: "black",
			r:    0, g: 0, b: 0,
			expected: color.RGBA{R: 0, G: 0, B: 0, A: 255},
		},
		{
			name: "white",
			r:    1, g: 1, b: 1,
			expected: color.RGBA{R: 255, G: 255, B: 255, A: 255},
		},
		{
			name: "red",
			r:    1, g: 0, b: 0,
			expected: color.RGBA{R: 255, G: 0, B: 0, A: 255},
		},
		{
			name: "half gray",
			r:    0.5, g: 0.5, b: 0.5,
			expected: color.RGBA{R: 127, G: 127, B: 127, A: 255},
		},
		{
			name: "clamped negative",
			r:    -0.5, g: 0, b: 0,
			expected: color.RGBA{R: 0, G: 0, B: 0, A: 255},
		},
		{
			name: "clamped over 1",
			r:    1.5, g: 0, b: 0,
			expected: color.RGBA{R: 255, G: 0, B: 0, A: 255},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cr := NewCairoRenderer()
			cr.SetSourceRGB(tt.r, tt.g, tt.b)
			got := cr.GetCurrentColor()

			if got.R != tt.expected.R || got.G != tt.expected.G ||
				got.B != tt.expected.B || got.A != tt.expected.A {
				t.Errorf("SetSourceRGB(%f,%f,%f) = (%d,%d,%d,%d), want (%d,%d,%d,%d)",
					tt.r, tt.g, tt.b,
					got.R, got.G, got.B, got.A,
					tt.expected.R, tt.expected.G, tt.expected.B, tt.expected.A)
			}
		})
	}
}

func TestCairoRenderer_SetSourceRGBA(t *testing.T) {
	tests := []struct {
		name       string
		r, g, b, a float64
		expected   color.RGBA
	}{
		{
			name: "fully opaque black",
			r:    0, g: 0, b: 0, a: 1,
			expected: color.RGBA{R: 0, G: 0, B: 0, A: 255},
		},
		{
			name: "fully transparent",
			r:    1, g: 1, b: 1, a: 0,
			expected: color.RGBA{R: 255, G: 255, B: 255, A: 0},
		},
		{
			name: "half transparent red",
			r:    1, g: 0, b: 0, a: 0.5,
			expected: color.RGBA{R: 255, G: 0, B: 0, A: 127},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cr := NewCairoRenderer()
			cr.SetSourceRGBA(tt.r, tt.g, tt.b, tt.a)
			got := cr.GetCurrentColor()

			if got.R != tt.expected.R || got.G != tt.expected.G ||
				got.B != tt.expected.B || got.A != tt.expected.A {
				t.Errorf("SetSourceRGBA(%f,%f,%f,%f) = (%d,%d,%d,%d), want (%d,%d,%d,%d)",
					tt.r, tt.g, tt.b, tt.a,
					got.R, got.G, got.B, got.A,
					tt.expected.R, tt.expected.G, tt.expected.B, tt.expected.A)
			}
		})
	}
}

func TestCairoRenderer_LineWidth(t *testing.T) {
	tests := []struct {
		name     string
		width    float64
		expected float64
	}{
		{"positive", 2.5, 2.5},
		{"zero becomes 1", 0, 1.0},
		{"negative becomes 1", -5, 1.0},
		{"large value", 100.0, 100.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cr := NewCairoRenderer()
			cr.SetLineWidth(tt.width)
			got := cr.GetLineWidth()

			if got != tt.expected {
				t.Errorf("SetLineWidth(%f) = %f, want %f", tt.width, got, tt.expected)
			}
		})
	}
}

func TestCairoRenderer_LineCap(t *testing.T) {
	tests := []struct {
		cap LineCap
	}{
		{LineCapButt},
		{LineCapRound},
		{LineCapSquare},
	}

	for _, tt := range tests {
		cr := NewCairoRenderer()
		cr.SetLineCap(tt.cap)
		got := cr.GetLineCap()

		if got != tt.cap {
			t.Errorf("SetLineCap(%v) = %v, want %v", tt.cap, got, tt.cap)
		}
	}
}

func TestCairoRenderer_LineJoin(t *testing.T) {
	tests := []struct {
		join LineJoin
	}{
		{LineJoinMiter},
		{LineJoinRound},
		{LineJoinBevel},
	}

	for _, tt := range tests {
		cr := NewCairoRenderer()
		cr.SetLineJoin(tt.join)
		got := cr.GetLineJoin()

		if got != tt.join {
			t.Errorf("SetLineJoin(%v) = %v, want %v", tt.join, got, tt.join)
		}
	}
}

func TestCairoRenderer_Antialias(t *testing.T) {
	cr := NewCairoRenderer()

	// Default is enabled
	if !cr.GetAntialias() {
		t.Error("Expected antialias to be enabled by default")
	}

	// Disable
	cr.SetAntialias(false)
	if cr.GetAntialias() {
		t.Error("Expected antialias to be disabled after SetAntialias(false)")
	}

	// Re-enable
	cr.SetAntialias(true)
	if !cr.GetAntialias() {
		t.Error("Expected antialias to be enabled after SetAntialias(true)")
	}
}

func TestCairoRenderer_PathBuilding(t *testing.T) {
	cr := NewCairoRenderer()

	// Initially no current point
	_, _, hasPoint := cr.GetCurrentPoint()
	if hasPoint {
		t.Error("Expected no current point before MoveTo")
	}

	// MoveTo sets current point
	cr.MoveTo(10, 20)
	x, y, hasPoint := cr.GetCurrentPoint()
	if !hasPoint {
		t.Error("Expected current point after MoveTo")
	}
	if x != 10 || y != 20 {
		t.Errorf("Expected current point (10,20), got (%f,%f)", x, y)
	}

	// LineTo updates current point
	cr.LineTo(30, 40)
	x, y, _ = cr.GetCurrentPoint()
	if x != 30 || y != 40 {
		t.Errorf("Expected current point (30,40), got (%f,%f)", x, y)
	}

	// NewPath clears path
	cr.NewPath()
	_, _, hasPoint = cr.GetCurrentPoint()
	if hasPoint {
		t.Error("Expected no current point after NewPath")
	}
}

func TestCairoRenderer_Rectangle(t *testing.T) {
	cr := NewCairoRenderer()
	cr.Rectangle(10, 20, 100, 50)

	x, y, hasPoint := cr.GetCurrentPoint()
	if !hasPoint {
		t.Error("Expected current point after Rectangle")
	}
	// After a rectangle, current point should be at the starting corner
	if x != 10 || y != 20 {
		t.Errorf("Expected current point (10,20), got (%f,%f)", x, y)
	}
}

func TestCairoRenderer_Arc(t *testing.T) {
	cr := NewCairoRenderer()

	// Full circle
	cr.Arc(100, 100, 50, 0, 2*math.Pi)

	x, y, hasPoint := cr.GetCurrentPoint()
	if !hasPoint {
		t.Error("Expected current point after Arc")
	}
	// End point should be at angle 2*Pi (same as 0) from center
	expectedX := 100 + 50*math.Cos(2*math.Pi)
	expectedY := 100 + 50*math.Sin(2*math.Pi)
	const epsilon = 0.001
	if math.Abs(x-expectedX) > epsilon || math.Abs(y-expectedY) > epsilon {
		t.Errorf("Expected current point (%f,%f), got (%f,%f)", expectedX, expectedY, x, y)
	}
}

func TestCairoRenderer_CurveTo(t *testing.T) {
	cr := NewCairoRenderer()
	cr.MoveTo(0, 0)
	cr.CurveTo(10, 20, 30, 40, 50, 60)

	x, y, hasPoint := cr.GetCurrentPoint()
	if !hasPoint {
		t.Error("Expected current point after CurveTo")
	}
	if x != 50 || y != 60 {
		t.Errorf("Expected current point (50,60), got (%f,%f)", x, y)
	}
}

func TestCairoRenderer_Screen(t *testing.T) {
	cr := NewCairoRenderer()

	// Initially nil
	if cr.Screen() != nil {
		t.Error("Expected nil screen initially")
	}

	// Stroke and Fill should not panic with nil screen
	cr.MoveTo(0, 0)
	cr.LineTo(10, 10)
	cr.Stroke() // Should not panic

	cr.Rectangle(0, 0, 10, 10)
	cr.Fill() // Should not panic

	cr.Paint() // Should not panic
}

func TestClampToByte(t *testing.T) {
	tests := []struct {
		input    float64
		expected uint8
	}{
		{0, 0},
		{1, 255},
		{0.5, 127},
		{-1, 0},
		{2, 255},
		{0.25, 63},
	}

	for _, tt := range tests {
		got := clampToByte(tt.input)
		if got != tt.expected {
			t.Errorf("clampToByte(%f) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestCairoRenderer_ConvenienceFunctions(t *testing.T) {
	// Test that convenience functions don't panic
	cr := NewCairoRenderer()

	// These should not panic even without a screen set
	cr.DrawLine(0, 0, 10, 10)
	cr.DrawRectangle(0, 0, 10, 10)
	cr.FillRectangle(0, 0, 10, 10)
	cr.DrawCircle(50, 50, 25)
	cr.FillCircle(50, 50, 25)
}

func TestCairoRenderer_PreserveFunctions(t *testing.T) {
	cr := NewCairoRenderer()

	// Build a path
	cr.Rectangle(0, 0, 10, 10)

	// Check that StrokePreserve keeps the path
	// (we can only verify it doesn't panic without a screen)
	cr.StrokePreserve()

	// Path should still be valid - we can check by calling GetCurrentPoint
	_, _, hasPoint := cr.GetCurrentPoint()
	// Note: After StrokePreserve without a screen, the path state is unchanged
	if !hasPoint {
		t.Error("Expected path to be preserved after StrokePreserve")
	}

	// Same for FillPreserve
	cr.NewPath()
	cr.Rectangle(0, 0, 10, 10)
	cr.FillPreserve()
	_, _, hasPoint = cr.GetCurrentPoint()
	if !hasPoint {
		t.Error("Expected path to be preserved after FillPreserve")
	}
}

func TestCairoRenderer_PaintWithAlpha(t *testing.T) {
	cr := NewCairoRenderer()

	// Should not panic without screen
	cr.PaintWithAlpha(0.5)
}

func TestCairoRenderer_LineToWithoutMoveTo(t *testing.T) {
	cr := NewCairoRenderer()

	// LineTo without MoveTo should start at the given point
	cr.LineTo(50, 50)

	x, y, hasPoint := cr.GetCurrentPoint()
	if !hasPoint {
		t.Error("Expected current point after LineTo")
	}
	if x != 50 || y != 50 {
		t.Errorf("Expected current point (50,50), got (%f,%f)", x, y)
	}
}

func TestCairoRenderer_CurveToWithoutMoveTo(t *testing.T) {
	cr := NewCairoRenderer()

	// CurveTo without MoveTo should handle gracefully
	cr.CurveTo(10, 20, 30, 40, 50, 60)

	x, y, hasPoint := cr.GetCurrentPoint()
	if !hasPoint {
		t.Error("Expected current point after CurveTo")
	}
	if x != 50 || y != 60 {
		t.Errorf("Expected current point (50,60), got (%f,%f)", x, y)
	}
}

func TestCairoRenderer_ClosePath(t *testing.T) {
	cr := NewCairoRenderer()

	// ClosePath without a path should not panic
	cr.ClosePath()

	// ClosePath with a path should work
	cr.MoveTo(0, 0)
	cr.LineTo(10, 0)
	cr.LineTo(10, 10)
	cr.ClosePath() // Should draw line back to (0,0)
}

func TestCairoRenderer_ArcNegative(t *testing.T) {
	cr := NewCairoRenderer()

	// ArcNegative should create arc in counter-clockwise direction
	cr.ArcNegative(100, 100, 50, math.Pi, 0)

	x, y, hasPoint := cr.GetCurrentPoint()
	if !hasPoint {
		t.Error("Expected current point after ArcNegative")
	}
	// End point should be at angle 0 from center
	expectedX := 100 + 50*math.Cos(0)
	expectedY := 100 + 50*math.Sin(0)
	const epsilon = 0.001
	if math.Abs(x-expectedX) > epsilon || math.Abs(y-expectedY) > epsilon {
		t.Errorf("Expected current point (%f,%f), got (%f,%f)", expectedX, expectedY, x, y)
	}
}
