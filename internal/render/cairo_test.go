// Package render provides Ebiten-based rendering capabilities for conky-go.
package render

import (
	"image/color"
	"math"
	"os"
	"sync"
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

// TestCairoRenderer_ConcurrentAccess tests that CairoRenderer is thread-safe.
// This test should be run with the race detector: go test -race
func TestCairoRenderer_ConcurrentAccess(t *testing.T) {
	cr := NewCairoRenderer()
	const numGoroutines = 10
	const numOperations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				// Mix of different operations to test concurrent access
				cr.SetSourceRGB(float64(id)/10, float64(j)/100, 0.5)
				cr.SetLineWidth(float64(id + 1))
				cr.SetLineCap(LineCap(id % 3))
				cr.SetLineJoin(LineJoin(j % 3))
				cr.SetAntialias(j%2 == 0)
				cr.NewPath()
				cr.MoveTo(float64(id*10), float64(j*10))
				cr.LineTo(float64(id*10+50), float64(j*10+50))
				cr.Rectangle(float64(id), float64(j), 10, 10)
				cr.GetCurrentPoint()
			}
		}(i)
	}

	wg.Wait()
}

// TestCairoRenderer_ConcurrentColorChanges tests concurrent color changes.
func TestCairoRenderer_ConcurrentColorChanges(t *testing.T) {
	cr := NewCairoRenderer()
	const numGoroutines = 20

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				cr.SetSourceRGBA(float64(id)/20, float64(j)/50, 0.5, 0.8)
				_ = cr.GetCurrentColor()
			}
		}(i)
	}

	wg.Wait()
}

// TestCairoRenderer_ConcurrentPathBuilding tests concurrent path building operations.
func TestCairoRenderer_ConcurrentPathBuilding(t *testing.T) {
	cr := NewCairoRenderer()
	const numGoroutines = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				cr.NewPath()
				cr.MoveTo(float64(id*10), float64(j*10))
				cr.LineTo(float64(id*10+100), float64(j*10+100))
				cr.Arc(50, 50, 25, 0, math.Pi)
				cr.ClosePath()
				cr.GetCurrentPoint()
			}
		}(i)
	}

	wg.Wait()
}

// --- Text Function Tests ---

func TestCairoRenderer_SelectFontFace(t *testing.T) {
	cr := NewCairoRenderer()

	// Test setting font face
	cr.SelectFontFace("GoMono", FontSlantNormal, FontWeightBold)

	// Verify by checking that no panic occurs and renderer is still valid
	if cr == nil {
		t.Error("Renderer should not be nil after SelectFontFace")
	}
}

func TestCairoRenderer_SetFontSize(t *testing.T) {
	cr := NewCairoRenderer()

	// Test setting various font sizes
	testCases := []struct {
		name     string
		size     float64
		expected float64
	}{
		{"normal size", 16.0, 16.0},
		{"large size", 48.0, 48.0},
		{"small size", 8.0, 8.0},
		{"zero size defaults to 14", 0, 14.0},
		{"negative size defaults to 14", -5, 14.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cr.SetFontSize(tc.size)
			if cr.GetFontSize() != tc.expected {
				t.Errorf("Expected font size %f, got %f", tc.expected, cr.GetFontSize())
			}
		})
	}
}

func TestCairoRenderer_ShowText(t *testing.T) {
	cr := NewCairoRenderer()

	// Set up a position
	cr.MoveTo(10, 20)

	// ShowText should not panic even without a screen
	cr.ShowText("Hello, World!")

	// After showing text, current point should have advanced
	x, y, hasPoint := cr.GetCurrentPoint()
	if !hasPoint {
		t.Error("Expected current point after ShowText")
	}
	// X should be > 10 because text advances the cursor
	if x <= 10 {
		t.Errorf("Expected X > 10 after ShowText, got %f", x)
	}
	// Y should remain at 20
	if y != 20 {
		t.Errorf("Expected Y to remain 20, got %f", y)
	}
}

func TestCairoRenderer_TextExtentsResult(t *testing.T) {
	cr := NewCairoRenderer()

	// Set a font size
	cr.SetFontSize(16)

	// Get text extents
	extents := cr.TextExtentsResult("Hello")

	// Width should be positive
	if extents.Width <= 0 {
		t.Errorf("Expected positive width, got %f", extents.Width)
	}

	// Height should be positive
	if extents.Height <= 0 {
		t.Errorf("Expected positive height, got %f", extents.Height)
	}

	// XAdvance should be equal to width for simple text
	if extents.XAdvance != extents.Width {
		t.Errorf("Expected XAdvance (%f) to equal Width (%f)", extents.XAdvance, extents.Width)
	}
}

func TestCairoRenderer_TextExtentsEmptyString(t *testing.T) {
	cr := NewCairoRenderer()

	// Get text extents for empty string
	extents := cr.TextExtentsResult("")

	// Width should be zero for empty string
	if extents.Width != 0 {
		t.Errorf("Expected zero width for empty string, got %f", extents.Width)
	}
}

func TestCairoRenderer_FontSlantConstants(t *testing.T) {
	// Verify font slant constants have correct values
	if FontSlantNormal != 0 {
		t.Errorf("Expected FontSlantNormal = 0, got %d", FontSlantNormal)
	}
	if FontSlantItalic != 1 {
		t.Errorf("Expected FontSlantItalic = 1, got %d", FontSlantItalic)
	}
	if FontSlantOblique != 2 {
		t.Errorf("Expected FontSlantOblique = 2, got %d", FontSlantOblique)
	}
}

func TestCairoRenderer_FontWeightConstants(t *testing.T) {
	// Verify font weight constants have correct values
	if FontWeightNormal != 0 {
		t.Errorf("Expected FontWeightNormal = 0, got %d", FontWeightNormal)
	}
	if FontWeightBold != 1 {
		t.Errorf("Expected FontWeightBold = 1, got %d", FontWeightBold)
	}
}

// --- Transformation Function Tests ---

func TestCairoRenderer_Translate(t *testing.T) {
	cr := NewCairoRenderer()

	// Initial translation should be (0, 0)
	tx, ty := cr.GetTranslate()
	if tx != 0 || ty != 0 {
		t.Errorf("Expected initial translation (0, 0), got (%f, %f)", tx, ty)
	}

	// Apply translation
	cr.Translate(100, 200)
	tx, ty = cr.GetTranslate()
	if tx != 100 || ty != 200 {
		t.Errorf("Expected translation (100, 200), got (%f, %f)", tx, ty)
	}

	// Translations accumulate
	cr.Translate(50, 25)
	tx, ty = cr.GetTranslate()
	if tx != 150 || ty != 225 {
		t.Errorf("Expected accumulated translation (150, 225), got (%f, %f)", tx, ty)
	}
}

func TestCairoRenderer_Rotate(t *testing.T) {
	cr := NewCairoRenderer()

	// Initial rotation should be 0
	if cr.GetRotation() != 0 {
		t.Errorf("Expected initial rotation 0, got %f", cr.GetRotation())
	}

	// Apply rotation (pi/4 = 45 degrees)
	cr.Rotate(math.Pi / 4)
	if math.Abs(cr.GetRotation()-math.Pi/4) > 0.001 {
		t.Errorf("Expected rotation pi/4, got %f", cr.GetRotation())
	}

	// Rotations accumulate
	cr.Rotate(math.Pi / 4)
	if math.Abs(cr.GetRotation()-math.Pi/2) > 0.001 {
		t.Errorf("Expected accumulated rotation pi/2, got %f", cr.GetRotation())
	}
}

func TestCairoRenderer_Scale(t *testing.T) {
	cr := NewCairoRenderer()

	// Initial scale should be (1, 1)
	sx, sy := cr.GetScale()
	if sx != 1 || sy != 1 {
		t.Errorf("Expected initial scale (1, 1), got (%f, %f)", sx, sy)
	}

	// Apply scale
	cr.Scale(2, 3)
	sx, sy = cr.GetScale()
	if sx != 2 || sy != 3 {
		t.Errorf("Expected scale (2, 3), got (%f, %f)", sx, sy)
	}

	// Scales multiply
	cr.Scale(0.5, 2)
	sx, sy = cr.GetScale()
	if sx != 1 || sy != 6 {
		t.Errorf("Expected accumulated scale (1, 6), got (%f, %f)", sx, sy)
	}
}

func TestCairoRenderer_SaveRestore(t *testing.T) {
	cr := NewCairoRenderer()

	// Set initial state
	cr.SetSourceRGB(1, 0, 0)
	cr.SetLineWidth(5)
	cr.Translate(100, 200)
	cr.SetFontSize(24)

	// Save state
	cr.Save()

	// Modify state
	cr.SetSourceRGB(0, 1, 0)
	cr.SetLineWidth(10)
	cr.Translate(50, 50)
	cr.SetFontSize(48)

	// Verify modified state
	if cr.GetLineWidth() != 10 {
		t.Errorf("Expected modified line width 10, got %f", cr.GetLineWidth())
	}

	// Restore state
	cr.Restore()

	// Verify restored state
	clr := cr.GetCurrentColor()
	if clr.R != 255 || clr.G != 0 || clr.B != 0 {
		t.Errorf("Expected restored color (255, 0, 0), got (%d, %d, %d)", clr.R, clr.G, clr.B)
	}
	if cr.GetLineWidth() != 5 {
		t.Errorf("Expected restored line width 5, got %f", cr.GetLineWidth())
	}
	tx, ty := cr.GetTranslate()
	if tx != 100 || ty != 200 {
		t.Errorf("Expected restored translation (100, 200), got (%f, %f)", tx, ty)
	}
	if cr.GetFontSize() != 24 {
		t.Errorf("Expected restored font size 24, got %f", cr.GetFontSize())
	}
}

func TestCairoRenderer_SaveRestoreMultipleLevels(t *testing.T) {
	cr := NewCairoRenderer()

	// Set state and save multiple times
	cr.SetLineWidth(1)
	cr.Save()

	cr.SetLineWidth(2)
	cr.Save()

	cr.SetLineWidth(3)
	cr.Save()

	cr.SetLineWidth(4)

	// Verify current state
	if cr.GetLineWidth() != 4 {
		t.Errorf("Expected line width 4, got %f", cr.GetLineWidth())
	}

	// Restore and verify each level
	cr.Restore()
	if cr.GetLineWidth() != 3 {
		t.Errorf("Expected line width 3 after first restore, got %f", cr.GetLineWidth())
	}

	cr.Restore()
	if cr.GetLineWidth() != 2 {
		t.Errorf("Expected line width 2 after second restore, got %f", cr.GetLineWidth())
	}

	cr.Restore()
	if cr.GetLineWidth() != 1 {
		t.Errorf("Expected line width 1 after third restore, got %f", cr.GetLineWidth())
	}
}

func TestCairoRenderer_RestoreEmptyStack(t *testing.T) {
	cr := NewCairoRenderer()

	// Set some state
	cr.SetLineWidth(5)
	cr.Translate(100, 100)

	// Restore on empty stack should do nothing (not panic)
	cr.Restore()

	// State should be unchanged
	if cr.GetLineWidth() != 5 {
		t.Errorf("Expected line width 5 unchanged, got %f", cr.GetLineWidth())
	}
	tx, ty := cr.GetTranslate()
	if tx != 100 || ty != 100 {
		t.Errorf("Expected translation (100, 100) unchanged, got (%f, %f)", tx, ty)
	}
}

func TestCairoRenderer_IdentityMatrix(t *testing.T) {
	cr := NewCairoRenderer()

	// Apply transformations
	cr.Translate(100, 200)
	cr.Rotate(math.Pi)
	cr.Scale(2, 3)

	// Reset with IdentityMatrix
	cr.IdentityMatrix()

	// Verify all transformations are reset
	tx, ty := cr.GetTranslate()
	if tx != 0 || ty != 0 {
		t.Errorf("Expected translation (0, 0) after identity, got (%f, %f)", tx, ty)
	}
	if cr.GetRotation() != 0 {
		t.Errorf("Expected rotation 0 after identity, got %f", cr.GetRotation())
	}
	sx, sy := cr.GetScale()
	if sx != 1 || sy != 1 {
		t.Errorf("Expected scale (1, 1) after identity, got (%f, %f)", sx, sy)
	}
}

func TestCairoRenderer_TransformPointIntegration(t *testing.T) {
	cr := NewCairoRenderer()

	// Apply transformations and verify they affect text placement
	cr.Translate(50, 50)
	cr.MoveTo(10, 10)
	cr.ShowText("Test")

	// The text should have been drawn at transformed coordinates
	// (We can't directly verify the drawing, but we can check state)
	x, y, hasPoint := cr.GetCurrentPoint()
	if !hasPoint {
		t.Error("Expected current point after ShowText")
	}
	// Position should still be in local coordinates
	// (transformation is applied during drawing)
	if y != 10 {
		t.Errorf("Expected Y to remain 10, got %f", y)
	}
	if x <= 10 {
		t.Errorf("Expected X > 10 after text advance, got %f", x)
	}
}

// TestCairoRenderer_ConcurrentTransformations tests concurrent transformation access.
func TestCairoRenderer_ConcurrentTransformations(t *testing.T) {
	cr := NewCairoRenderer()
	const numGoroutines = 10
	const numOperations = 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				cr.Save()
				cr.Translate(float64(id*10), float64(j*10))
				cr.Rotate(float64(id) / 10)
				cr.Scale(1.0+float64(id)/100, 1.0+float64(j)/100)
				cr.GetTranslate()
				cr.GetRotation()
				cr.GetScale()
				cr.Restore()
			}
		}(i)
	}

	wg.Wait()
}

// TestCairoRenderer_ConcurrentTextOperations tests concurrent text operations.
func TestCairoRenderer_ConcurrentTextOperations(t *testing.T) {
	cr := NewCairoRenderer()
	const numGoroutines = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				cr.SelectFontFace("GoMono", FontSlant(j%3), FontWeight(j%2))
				cr.SetFontSize(float64(12 + id + j))
				cr.GetFontSize()
				cr.TextExtentsResult("Test string")
			}
		}(i)
	}

	wg.Wait()
}

// --- Surface Management Tests ---

func TestNewCairoSurface(t *testing.T) {
	surface := NewCairoSurface(100, 200)
	if surface == nil {
		t.Fatal("NewCairoSurface returned nil")
	}

	if surface.Width() != 100 {
		t.Errorf("Expected width 100, got %d", surface.Width())
	}
	if surface.Height() != 200 {
		t.Errorf("Expected height 200, got %d", surface.Height())
	}
	if surface.IsDestroyed() {
		t.Error("Surface should not be destroyed initially")
	}
	if surface.Image() == nil {
		t.Error("Surface image should not be nil")
	}
}

func TestNewCairoSurface_ZeroDimensions(t *testing.T) {
	// Zero dimensions should be clamped to 1
	surface := NewCairoSurface(0, 0)
	if surface == nil {
		t.Fatal("NewCairoSurface returned nil for zero dimensions")
	}
	if surface.Width() != 1 || surface.Height() != 1 {
		t.Errorf("Expected dimensions (1,1) for zero input, got (%d,%d)", surface.Width(), surface.Height())
	}
}

func TestNewCairoSurface_NegativeDimensions(t *testing.T) {
	// Negative dimensions should be clamped to 1
	surface := NewCairoSurface(-10, -20)
	if surface == nil {
		t.Fatal("NewCairoSurface returned nil for negative dimensions")
	}
	if surface.Width() != 1 || surface.Height() != 1 {
		t.Errorf("Expected dimensions (1,1) for negative input, got (%d,%d)", surface.Width(), surface.Height())
	}
}

func TestNewCairoXlibSurface(t *testing.T) {
	// Test that NewCairoXlibSurface creates a valid surface
	// The display, drawable, visual parameters are for compatibility only
	surface := NewCairoXlibSurface(0, 0, 0, 640, 480)
	if surface == nil {
		t.Fatal("NewCairoXlibSurface returned nil")
	}

	if surface.Width() != 640 {
		t.Errorf("Expected width 640, got %d", surface.Width())
	}
	if surface.Height() != 480 {
		t.Errorf("Expected height 480, got %d", surface.Height())
	}
}

func TestCairoSurface_Destroy(t *testing.T) {
	surface := NewCairoSurface(100, 100)

	// Surface should not be destroyed initially
	if surface.IsDestroyed() {
		t.Error("Surface should not be destroyed initially")
	}

	// Destroy the surface
	surface.Destroy()

	// Surface should be marked as destroyed
	if !surface.IsDestroyed() {
		t.Error("Surface should be destroyed after Destroy()")
	}

	// Image should be nil after destruction
	if surface.Image() != nil {
		t.Error("Image should be nil after Destroy()")
	}

	// Calling Destroy again should not panic
	surface.Destroy()
}

func TestNewCairoContext(t *testing.T) {
	surface := NewCairoSurface(200, 150)
	ctx := NewCairoContext(surface)

	if ctx == nil {
		t.Fatal("NewCairoContext returned nil")
	}

	if ctx.Renderer() == nil {
		t.Error("Context renderer should not be nil")
	}

	if ctx.Surface() != surface {
		t.Error("Context surface should match the input surface")
	}

	if ctx.IsDestroyed() {
		t.Error("Context should not be destroyed initially")
	}
}

func TestNewCairoContext_NilSurface(t *testing.T) {
	ctx := NewCairoContext(nil)
	if ctx != nil {
		t.Error("NewCairoContext should return nil for nil surface")
	}
}

func TestNewCairoContext_DestroyedSurface(t *testing.T) {
	surface := NewCairoSurface(100, 100)
	surface.Destroy()

	ctx := NewCairoContext(surface)
	if ctx != nil {
		t.Error("NewCairoContext should return nil for destroyed surface")
	}
}

func TestCairoContext_Destroy(t *testing.T) {
	surface := NewCairoSurface(100, 100)
	ctx := NewCairoContext(surface)

	// Context should not be destroyed initially
	if ctx.IsDestroyed() {
		t.Error("Context should not be destroyed initially")
	}

	// Destroy the context
	ctx.Destroy()

	// Context should be marked as destroyed
	if !ctx.IsDestroyed() {
		t.Error("Context should be destroyed after Destroy()")
	}

	// Renderer should be nil after destruction
	if ctx.Renderer() != nil {
		t.Error("Renderer should be nil after Destroy()")
	}

	// Surface should NOT be destroyed (that's handled separately)
	if surface.IsDestroyed() {
		t.Error("Surface should not be destroyed when context is destroyed")
	}

	// Calling Destroy again should not panic
	ctx.Destroy()
}

func TestCairoContext_DrawingOperations(t *testing.T) {
	surface := NewCairoSurface(200, 200)
	ctx := NewCairoContext(surface)

	// Get the renderer and perform some drawing operations
	renderer := ctx.Renderer()
	if renderer == nil {
		t.Fatal("Renderer should not be nil")
	}

	// These operations should not panic
	renderer.SetSourceRGB(1, 0, 0)
	renderer.SetLineWidth(2)
	renderer.MoveTo(10, 10)
	renderer.LineTo(50, 50)
	renderer.Rectangle(60, 60, 30, 30)
	renderer.Stroke()

	renderer.SetSourceRGBA(0, 1, 0, 0.5)
	renderer.Rectangle(100, 100, 50, 50)
	renderer.Fill()

	// Clean up
	ctx.Destroy()
	surface.Destroy()
}

// TestCairoSurface_ConcurrentAccess tests thread safety of surface operations.
func TestCairoSurface_ConcurrentAccess(t *testing.T) {
	surface := NewCairoSurface(100, 100)
	const numGoroutines = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				surface.Width()
				surface.Height()
				surface.IsDestroyed()
				surface.Image()
			}
		}()
	}

	wg.Wait()
	surface.Destroy()
}

// TestCairoContext_ConcurrentAccess tests thread safety of context operations.
func TestCairoContext_ConcurrentAccess(t *testing.T) {
	surface := NewCairoSurface(100, 100)
	ctx := NewCairoContext(surface)
	const numGoroutines = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				ctx.IsDestroyed()
				ctx.Renderer()
				ctx.Surface()
			}
		}()
	}

	wg.Wait()
	ctx.Destroy()
	surface.Destroy()
}

// --- PNG Surface Loading/Saving Tests ---

func TestNewCairoSurfaceFromPNG(t *testing.T) {
	// Create a temporary PNG file for testing
	tmpDir := t.TempDir()
	pngPath := tmpDir + "/test.png"

	// First, create a simple surface and save it as PNG
	surface := NewCairoSurface(50, 30)
	ctx := NewCairoContext(surface)
	renderer := ctx.Renderer()

	// Draw something on the surface
	renderer.SetSourceRGB(1, 0, 0) // Red
	renderer.Rectangle(0, 0, 50, 30)
	renderer.Fill()

	err := surface.WriteToPNG(pngPath)
	if err != nil {
		t.Fatalf("Failed to create test PNG: %v", err)
	}
	ctx.Destroy()
	surface.Destroy()

	// Now test loading the PNG
	loadedSurface, err := NewCairoSurfaceFromPNG(pngPath)
	if err != nil {
		t.Fatalf("NewCairoSurfaceFromPNG failed: %v", err)
	}
	if loadedSurface == nil {
		t.Fatal("NewCairoSurfaceFromPNG returned nil surface")
	}

	if loadedSurface.Width() != 50 {
		t.Errorf("Expected width 50, got %d", loadedSurface.Width())
	}
	if loadedSurface.Height() != 30 {
		t.Errorf("Expected height 30, got %d", loadedSurface.Height())
	}
	if loadedSurface.IsDestroyed() {
		t.Error("Loaded surface should not be destroyed")
	}
	if loadedSurface.Image() == nil {
		t.Error("Loaded surface image should not be nil")
	}

	loadedSurface.Destroy()
}

func TestNewCairoSurfaceFromPNG_NonexistentFile(t *testing.T) {
	surface, err := NewCairoSurfaceFromPNG("/nonexistent/path/to/image.png")
	if err == nil {
		t.Error("Expected error for nonexistent file")
		if surface != nil {
			surface.Destroy()
		}
	}
	if surface != nil {
		t.Error("Expected nil surface for nonexistent file")
	}
}

func TestNewCairoSurfaceFromPNG_InvalidPNG(t *testing.T) {
	// Create a temporary file with invalid PNG data
	tmpDir := t.TempDir()
	invalidPath := tmpDir + "/invalid.png"

	// Write non-PNG data
	err := os.WriteFile(invalidPath, []byte("not a png file"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	surface, err := NewCairoSurfaceFromPNG(invalidPath)
	if err == nil {
		t.Error("Expected error for invalid PNG file")
		if surface != nil {
			surface.Destroy()
		}
	}
	if surface != nil {
		t.Error("Expected nil surface for invalid PNG")
	}
}

func TestCairoSurface_WriteToPNG(t *testing.T) {
	tmpDir := t.TempDir()
	pngPath := tmpDir + "/output.png"

	surface := NewCairoSurface(100, 80)
	ctx := NewCairoContext(surface)
	renderer := ctx.Renderer()

	// Draw a simple pattern
	renderer.SetSourceRGB(0, 0, 1) // Blue background
	renderer.Rectangle(0, 0, 100, 80)
	renderer.Fill()

	renderer.SetSourceRGB(1, 1, 0) // Yellow circle
	renderer.Arc(50, 40, 20, 0, 2*3.14159)
	renderer.Fill()

	err := surface.WriteToPNG(pngPath)
	if err != nil {
		t.Fatalf("WriteToPNG failed: %v", err)
	}

	// Verify the file exists and has content
	info, err := os.Stat(pngPath)
	if err != nil {
		t.Fatalf("Failed to stat output file: %v", err)
	}
	if info.Size() == 0 {
		t.Error("Output PNG file is empty")
	}

	// Verify we can load the file back
	loadedSurface, err := NewCairoSurfaceFromPNG(pngPath)
	if err != nil {
		t.Fatalf("Failed to reload saved PNG: %v", err)
	}
	if loadedSurface.Width() != 100 || loadedSurface.Height() != 80 {
		t.Errorf("Loaded dimensions mismatch: expected (100,80), got (%d,%d)",
			loadedSurface.Width(), loadedSurface.Height())
	}

	loadedSurface.Destroy()
	ctx.Destroy()
	surface.Destroy()
}

func TestCairoSurface_WriteToPNG_DestroyedSurface(t *testing.T) {
	tmpDir := t.TempDir()
	pngPath := tmpDir + "/destroyed.png"

	surface := NewCairoSurface(50, 50)
	surface.Destroy()

	err := surface.WriteToPNG(pngPath)
	if err == nil {
		t.Error("Expected error when writing destroyed surface to PNG")
	}
}

func TestCairoSurface_WriteToPNG_InvalidPath(t *testing.T) {
	surface := NewCairoSurface(50, 50)
	defer surface.Destroy()

	err := surface.WriteToPNG("/nonexistent/directory/output.png")
	if err == nil {
		t.Error("Expected error for invalid output path")
	}
}

func TestCairoSurfaceFromPNG_RoundTrip(t *testing.T) {
	// Test that a surface can be saved and loaded without data loss
	tmpDir := t.TempDir()
	pngPath := tmpDir + "/roundtrip.png"

	// Create original surface with specific dimensions
	original := NewCairoSurface(123, 456)
	ctx := NewCairoContext(original)
	renderer := ctx.Renderer()

	// Draw something identifiable
	renderer.SetSourceRGBA(0.5, 0.25, 0.75, 1.0)
	renderer.Rectangle(10, 20, 50, 60)
	renderer.Fill()

	// Save
	err := original.WriteToPNG(pngPath)
	if err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Load
	loaded, err := NewCairoSurfaceFromPNG(pngPath)
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	// Verify dimensions match
	if loaded.Width() != original.Width() || loaded.Height() != original.Height() {
		t.Errorf("Dimension mismatch: original (%d,%d), loaded (%d,%d)",
			original.Width(), original.Height(), loaded.Width(), loaded.Height())
	}

	loaded.Destroy()
	ctx.Destroy()
	original.Destroy()
}

// --- Relative Path Function Tests ---

func TestCairoRenderer_RelMoveTo(t *testing.T) {
	cr := NewCairoRenderer()

	// First move to an absolute position
	cr.MoveTo(100, 100)

	// Verify initial position
	x, y, hasPoint := cr.GetCurrentPoint()
	if !hasPoint {
		t.Fatal("Expected current point after MoveTo")
	}
	if x != 100 || y != 100 {
		t.Errorf("Expected initial position (100, 100), got (%f, %f)", x, y)
	}

	// Now do a relative move
	cr.RelMoveTo(50, 25)

	// Verify relative move
	x, y, hasPoint = cr.GetCurrentPoint()
	if !hasPoint {
		t.Fatal("Expected current point after RelMoveTo")
	}
	if x != 150 || y != 125 {
		t.Errorf("Expected position (150, 125) after RelMoveTo, got (%f, %f)", x, y)
	}
}

func TestCairoRenderer_RelMoveToNoPath(t *testing.T) {
	cr := NewCairoRenderer()

	// Relative move without a current point should do nothing
	cr.RelMoveTo(50, 25)

	// Verify no current point
	_, _, hasPoint := cr.GetCurrentPoint()
	if hasPoint {
		t.Error("Expected no current point after RelMoveTo without initial point")
	}
}

func TestCairoRenderer_RelLineTo(t *testing.T) {
	cr := NewCairoRenderer()

	// First move to an absolute position
	cr.MoveTo(100, 100)

	// Now do a relative line
	cr.RelLineTo(50, 25)

	// Verify relative line end point
	x, y, hasPoint := cr.GetCurrentPoint()
	if !hasPoint {
		t.Fatal("Expected current point after RelLineTo")
	}
	if x != 150 || y != 125 {
		t.Errorf("Expected position (150, 125) after RelLineTo, got (%f, %f)", x, y)
	}
}

func TestCairoRenderer_RelLineToNoPath(t *testing.T) {
	cr := NewCairoRenderer()

	// Relative line without a current point should do nothing
	cr.RelLineTo(50, 25)

	// Verify no current point
	_, _, hasPoint := cr.GetCurrentPoint()
	if hasPoint {
		t.Error("Expected no current point after RelLineTo without initial point")
	}
}

func TestCairoRenderer_RelCurveTo(t *testing.T) {
	cr := NewCairoRenderer()

	// First move to an absolute position
	cr.MoveTo(100, 100)

	// Now do a relative curve
	cr.RelCurveTo(10, 20, 30, 40, 50, 60)

	// Verify relative curve end point
	x, y, hasPoint := cr.GetCurrentPoint()
	if !hasPoint {
		t.Fatal("Expected current point after RelCurveTo")
	}
	// End point should be (100+50, 100+60) = (150, 160)
	if x != 150 || y != 160 {
		t.Errorf("Expected position (150, 160) after RelCurveTo, got (%f, %f)", x, y)
	}
}

func TestCairoRenderer_RelCurveToNoPath(t *testing.T) {
	cr := NewCairoRenderer()

	// Relative curve without a current point should do nothing
	cr.RelCurveTo(10, 20, 30, 40, 50, 60)

	// Verify no current point
	_, _, hasPoint := cr.GetCurrentPoint()
	if hasPoint {
		t.Error("Expected no current point after RelCurveTo without initial point")
	}
}

func TestCairoRenderer_RelativePathChain(t *testing.T) {
	cr := NewCairoRenderer()

	// Create a square using relative moves
	cr.MoveTo(0, 0)
	cr.RelLineTo(100, 0)  // right
	cr.RelLineTo(0, 100)  // down
	cr.RelLineTo(-100, 0) // left
	cr.RelLineTo(0, -100) // up (back to start)

	// Verify we're back at the origin
	x, y, hasPoint := cr.GetCurrentPoint()
	if !hasPoint {
		t.Fatal("Expected current point after relative path chain")
	}
	if x != 0 || y != 0 {
		t.Errorf("Expected position (0, 0) after relative square, got (%f, %f)", x, y)
	}
}

// --- Clipping Function Tests ---

func TestCairoRenderer_Clip(t *testing.T) {
	cr := NewCairoRenderer()

	// Create a clip path
	cr.Rectangle(10, 10, 100, 100)

	// Verify we have a path before clip
	_, _, hasPointBefore := cr.GetCurrentPoint()
	if !hasPointBefore {
		t.Fatal("Expected current point before clip")
	}

	// Apply clip
	cr.Clip()

	// Verify clip is set and path is cleared
	if !cr.HasClip() {
		t.Error("Expected clip to be set after Clip()")
	}

	// Path should be cleared after clip
	_, _, hasPointAfter := cr.GetCurrentPoint()
	if hasPointAfter {
		t.Error("Expected path to be cleared after Clip()")
	}
}

func TestCairoRenderer_ClipNoPath(t *testing.T) {
	cr := NewCairoRenderer()

	// Clip without a path should do nothing
	cr.Clip()

	// Verify clip is not set
	if cr.HasClip() {
		t.Error("Expected no clip after Clip() without path")
	}
}

func TestCairoRenderer_ClipPreserve(t *testing.T) {
	cr := NewCairoRenderer()

	// Create a clip path
	cr.Rectangle(10, 10, 100, 100)

	// Apply clip preserve
	cr.ClipPreserve()

	// Verify clip is set and path is preserved
	if !cr.HasClip() {
		t.Error("Expected clip to be set after ClipPreserve()")
	}

	// Path should be preserved after clip preserve
	_, _, hasPoint := cr.GetCurrentPoint()
	if !hasPoint {
		t.Error("Expected path to be preserved after ClipPreserve()")
	}
}

func TestCairoRenderer_ResetClip(t *testing.T) {
	cr := NewCairoRenderer()

	// Set clip
	cr.Rectangle(10, 10, 100, 100)
	cr.Clip()

	// Verify clip is set
	if !cr.HasClip() {
		t.Fatal("Expected clip to be set before reset")
	}

	// Reset clip
	cr.ResetClip()

	// Verify clip is reset
	if cr.HasClip() {
		t.Error("Expected clip to be reset after ResetClip()")
	}
}

func TestCairoRenderer_ClipSaveRestore(t *testing.T) {
	cr := NewCairoRenderer()

	// Save initial state (no clip)
	cr.Save()

	// Set clip
	cr.Rectangle(10, 10, 100, 100)
	cr.Clip()

	// Verify clip is set
	if !cr.HasClip() {
		t.Fatal("Expected clip to be set after Clip()")
	}

	// Restore state
	cr.Restore()

	// Verify clip is restored to no-clip state
	if cr.HasClip() {
		t.Error("Expected clip to be restored (no clip) after Restore()")
	}
}

func TestCairoRenderer_ClipSaveRestoreWithClip(t *testing.T) {
	cr := NewCairoRenderer()

	// Set initial clip
	cr.Rectangle(0, 0, 50, 50)
	cr.Clip()

	// Save state with clip
	cr.Save()

	// Reset clip
	cr.ResetClip()

	// Verify no clip
	if cr.HasClip() {
		t.Fatal("Expected no clip after ResetClip()")
	}

	// Restore state
	cr.Restore()

	// Verify clip is restored
	if !cr.HasClip() {
		t.Error("Expected clip to be restored after Restore()")
	}
}

func TestCairoRenderer_HasClipInitialState(t *testing.T) {
	cr := NewCairoRenderer()

	// Verify initial state has no clip
	if cr.HasClip() {
		t.Error("Expected no clip in initial state")
	}
}

// --- Tests for Path Query Functions ---

func TestCairoRenderer_HasCurrentPoint(t *testing.T) {
	cr := NewCairoRenderer()

	// Initially no current point
	if cr.HasCurrentPoint() {
		t.Error("Expected no current point in initial state")
	}

	// After MoveTo, should have current point
	cr.MoveTo(10, 20)
	if !cr.HasCurrentPoint() {
		t.Error("Expected current point after MoveTo")
	}

	// After NewPath, should have no current point
	cr.NewPath()
	if cr.HasCurrentPoint() {
		t.Error("Expected no current point after NewPath")
	}
}

func TestCairoRenderer_PathExtents(t *testing.T) {
	cr := NewCairoRenderer()

	// No path - should return zeros
	x1, y1, x2, y2 := cr.PathExtents()
	if x1 != 0 || y1 != 0 || x2 != 0 || y2 != 0 {
		t.Errorf("Expected (0,0,0,0) for empty path, got (%f,%f,%f,%f)", x1, y1, x2, y2)
	}

	// Simple rectangle
	cr.Rectangle(10, 20, 100, 50)
	x1, y1, x2, y2 = cr.PathExtents()
	if x1 != 10 || y1 != 20 || x2 != 110 || y2 != 70 {
		t.Errorf("Expected (10,20,110,70), got (%f,%f,%f,%f)", x1, y1, x2, y2)
	}

	// After NewPath, bounds should reset
	cr.NewPath()
	x1, y1, x2, y2 = cr.PathExtents()
	if x1 != 0 || y1 != 0 || x2 != 0 || y2 != 0 {
		t.Errorf("Expected (0,0,0,0) after NewPath, got (%f,%f,%f,%f)", x1, y1, x2, y2)
	}
}

func TestCairoRenderer_PathExtentsWithLines(t *testing.T) {
	cr := NewCairoRenderer()

	// Build a path with lines
	cr.MoveTo(0, 0)
	cr.LineTo(100, 50)
	cr.LineTo(50, 100)

	x1, y1, x2, y2 := cr.PathExtents()
	if x1 != 0 || y1 != 0 || x2 != 100 || y2 != 100 {
		t.Errorf("Expected (0,0,100,100), got (%f,%f,%f,%f)", x1, y1, x2, y2)
	}
}

func TestCairoRenderer_PathExtentsWithCurve(t *testing.T) {
	cr := NewCairoRenderer()

	// Build a path with a curve
	cr.MoveTo(0, 0)
	cr.CurveTo(20, 30, 40, 50, 60, 70)

	x1, y1, x2, y2 := cr.PathExtents()
	// Bounds should include control points and endpoints
	if x1 != 0 || y1 != 0 || x2 != 60 || y2 != 70 {
		t.Errorf("Expected (0,0,60,70), got (%f,%f,%f,%f)", x1, y1, x2, y2)
	}
}

func TestCairoRenderer_ClipExtents(t *testing.T) {
	cr := NewCairoRenderer()

	// No clip, no screen - should return zeros
	x1, y1, x2, y2 := cr.ClipExtents()
	if x1 != 0 || y1 != 0 || x2 != 0 || y2 != 0 {
		t.Errorf("Expected (0,0,0,0) with no clip and no screen, got (%f,%f,%f,%f)", x1, y1, x2, y2)
	}

	// Set a clip region
	cr.Rectangle(10, 20, 100, 50)
	cr.Clip()

	x1, y1, x2, y2 = cr.ClipExtents()
	if x1 != 10 || y1 != 20 || x2 != 110 || y2 != 70 {
		t.Errorf("Expected (10,20,110,70), got (%f,%f,%f,%f)", x1, y1, x2, y2)
	}

	// Reset clip
	cr.ResetClip()
	x1, y1, x2, y2 = cr.ClipExtents()
	if x1 != 0 || y1 != 0 || x2 != 0 || y2 != 0 {
		t.Errorf("Expected (0,0,0,0) after ResetClip, got (%f,%f,%f,%f)", x1, y1, x2, y2)
	}
}

func TestCairoRenderer_InClip(t *testing.T) {
	cr := NewCairoRenderer()

	// No clip - all points should be in clip
	if !cr.InClip(50, 50) {
		t.Error("Expected point to be in clip when no clip is set")
	}

	// Set a clip region
	cr.Rectangle(10, 20, 100, 50)
	cr.Clip()

	// Point inside clip
	if !cr.InClip(50, 40) {
		t.Error("Expected point (50,40) to be inside clip (10,20,110,70)")
	}

	// Point outside clip
	if cr.InClip(5, 40) {
		t.Error("Expected point (5,40) to be outside clip (10,20,110,70)")
	}
	if cr.InClip(50, 15) {
		t.Error("Expected point (50,15) to be outside clip (10,20,110,70)")
	}
	if cr.InClip(200, 40) {
		t.Error("Expected point (200,40) to be outside clip (10,20,110,70)")
	}

	// Point on boundary should be inside
	if !cr.InClip(10, 20) {
		t.Error("Expected point (10,20) on boundary to be inside clip")
	}
	if !cr.InClip(110, 70) {
		t.Error("Expected point (110,70) on boundary to be inside clip")
	}
}

func TestCairoRenderer_ClipBoundsSaveRestore(t *testing.T) {
	cr := NewCairoRenderer()

	// Set initial clip
	cr.Rectangle(10, 20, 100, 50)
	cr.Clip()

	// Save state
	cr.Save()

	// Set a different clip
	cr.Rectangle(0, 0, 200, 200)
	cr.Clip()

	x1, y1, x2, y2 := cr.ClipExtents()
	if x1 != 0 || y1 != 0 || x2 != 200 || y2 != 200 {
		t.Errorf("Expected (0,0,200,200), got (%f,%f,%f,%f)", x1, y1, x2, y2)
	}

	// Restore state
	cr.Restore()

	// Should have original clip bounds
	x1, y1, x2, y2 = cr.ClipExtents()
	if x1 != 10 || y1 != 20 || x2 != 110 || y2 != 70 {
		t.Errorf("Expected (10,20,110,70) after restore, got (%f,%f,%f,%f)", x1, y1, x2, y2)
	}
}

// --- CairoMatrix Tests ---

func TestNewIdentityMatrix(t *testing.T) {
	m := NewIdentityMatrix()
	if m.XX != 1 || m.XY != 0 || m.YX != 0 || m.YY != 1 || m.X0 != 0 || m.Y0 != 0 {
		t.Errorf("Expected identity matrix, got %+v", m)
	}
}

func TestNewTranslateMatrix(t *testing.T) {
	m := NewTranslateMatrix(10, 20)
	x, y := m.TransformPoint(0, 0)
	if x != 10 || y != 20 {
		t.Errorf("Expected (10, 20), got (%f, %f)", x, y)
	}
}

func TestNewScaleMatrix(t *testing.T) {
	m := NewScaleMatrix(2, 3)
	x, y := m.TransformPoint(5, 4)
	if x != 10 || y != 12 {
		t.Errorf("Expected (10, 12), got (%f, %f)", x, y)
	}
}

func TestNewRotateMatrix(t *testing.T) {
	m := NewRotateMatrix(math.Pi / 2) // 90 degrees
	x, y := m.TransformPoint(1, 0)
	// After 90 degree rotation, (1, 0) -> (0, 1)
	if math.Abs(x) > 1e-10 || math.Abs(y-1) > 1e-10 {
		t.Errorf("Expected (0, 1), got (%f, %f)", x, y)
	}
}

func TestCairoMatrix_Translate(t *testing.T) {
	m := NewIdentityMatrix()
	m.Translate(10, 20)
	x, y := m.TransformPoint(0, 0)
	if x != 10 || y != 20 {
		t.Errorf("Expected (10, 20), got (%f, %f)", x, y)
	}
}

func TestCairoMatrix_Scale(t *testing.T) {
	m := NewIdentityMatrix()
	m.Scale(2, 3)
	x, y := m.TransformPoint(5, 4)
	if x != 10 || y != 12 {
		t.Errorf("Expected (10, 12), got (%f, %f)", x, y)
	}
}

func TestCairoMatrix_Rotate(t *testing.T) {
	m := NewIdentityMatrix()
	m.Rotate(math.Pi / 2) // 90 degrees
	x, y := m.TransformPoint(1, 0)
	if math.Abs(x) > 1e-10 || math.Abs(y-1) > 1e-10 {
		t.Errorf("Expected (0, 1), got (%f, %f)", x, y)
	}
}

func TestCairoMatrix_Multiply(t *testing.T) {
	// Create a translation then a scale
	translate := NewTranslateMatrix(10, 20)
	scale := NewScaleMatrix(2, 2)

	// Combine: scale * translate means translate first, then scale
	translate.Multiply(scale)

	// (0, 0) -> translate by (10, 20) -> (10, 20) -> scale by 2 -> (20, 40)
	x, y := translate.TransformPoint(0, 0)
	if x != 20 || y != 40 {
		t.Errorf("Expected (20, 40), got (%f, %f)", x, y)
	}
}

func TestCairoMatrix_TransformDistance(t *testing.T) {
	m := NewScaleMatrix(2, 3)
	m.Translate(100, 100) // Translation shouldn't affect distance

	dx, dy := m.TransformDistance(5, 4)
	if dx != 10 || dy != 12 {
		t.Errorf("Expected (10, 12), got (%f, %f)", dx, dy)
	}
}

func TestCairoMatrix_Invert(t *testing.T) {
	m := NewTranslateMatrix(10, 20)
	if !m.Invert() {
		t.Fatal("Expected inversion to succeed")
	}
	x, y := m.TransformPoint(10, 20)
	if math.Abs(x) > 1e-10 || math.Abs(y) > 1e-10 {
		t.Errorf("Expected (0, 0), got (%f, %f)", x, y)
	}
}

func TestCairoMatrix_InvertSingular(t *testing.T) {
	// Create a singular matrix (all zeros)
	m := &CairoMatrix{XX: 0, XY: 0, YX: 0, YY: 0, X0: 0, Y0: 0}
	if m.Invert() {
		t.Error("Expected inversion to fail for singular matrix")
	}
}

func TestCairoMatrix_Copy(t *testing.T) {
	m := NewTranslateMatrix(10, 20)
	copy := m.Copy()

	// Modify original
	m.Translate(5, 5)

	// Copy should be unchanged
	if copy.X0 != 10 || copy.Y0 != 20 {
		t.Errorf("Expected copy to be (10, 20), got (%f, %f)", copy.X0, copy.Y0)
	}
}

func TestCairoRenderer_GetSetMatrix(t *testing.T) {
	cr := NewCairoRenderer()

	// Get initial matrix (should be identity)
	m := cr.GetMatrix()
	if m.XX != 1 || m.XY != 0 || m.YX != 0 || m.YY != 1 || m.X0 != 0 || m.Y0 != 0 {
		t.Errorf("Expected identity matrix, got %+v", m)
	}

	// Set a new matrix
	newMatrix := NewTranslateMatrix(100, 200)
	cr.SetMatrix(newMatrix)

	m = cr.GetMatrix()
	if m.X0 != 100 || m.Y0 != 200 {
		t.Errorf("Expected translation (100, 200), got (%f, %f)", m.X0, m.Y0)
	}
}

func TestCairoRenderer_Transform(t *testing.T) {
	cr := NewCairoRenderer()

	// Apply a translation
	translate := NewTranslateMatrix(10, 20)
	cr.Transform(translate)

	m := cr.GetMatrix()
	if m.X0 != 10 || m.Y0 != 20 {
		t.Errorf("Expected translation (10, 20), got (%f, %f)", m.X0, m.Y0)
	}
}

func TestCairoRenderer_MatrixSaveRestore(t *testing.T) {
	cr := NewCairoRenderer()

	// Set a matrix
	cr.SetMatrix(NewTranslateMatrix(100, 200))

	// Save
	cr.Save()

	// Modify matrix
	cr.SetMatrix(NewTranslateMatrix(500, 600))

	m := cr.GetMatrix()
	if m.X0 != 500 || m.Y0 != 600 {
		t.Errorf("Expected (500, 600), got (%f, %f)", m.X0, m.Y0)
	}

	// Restore
	cr.Restore()

	m = cr.GetMatrix()
	if m.X0 != 100 || m.Y0 != 200 {
		t.Errorf("Expected (100, 200) after restore, got (%f, %f)", m.X0, m.Y0)
	}
}

// --- Pattern Extend Tests ---

func TestCairoPattern_SetGetExtend(t *testing.T) {
	tests := []struct {
		name   string
		extend PatternExtend
	}{
		{"none", PatternExtendNone},
		{"repeat", PatternExtendRepeat},
		{"reflect", PatternExtendReflect},
		{"pad", PatternExtendPad},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewLinearPattern(0, 0, 100, 0)
			p.SetExtend(tt.extend)
			if got := p.GetExtend(); got != tt.extend {
				t.Errorf("Expected extend %v, got %v", tt.extend, got)
			}
		})
	}
}

// --- Surface Functions Tests ---

func TestCairoSurface_FlushAndMarkDirty(t *testing.T) {
	s := NewCairoSurface(100, 100)
	defer s.Destroy()

	// These should not panic (they are no-ops in Ebiten)
	s.Flush()
	s.MarkDirty()
	s.MarkDirtyRectangle(0, 0, 50, 50)
}

func TestCairoSurface_FlushAfterDestroy(t *testing.T) {
	s := NewCairoSurface(100, 100)
	s.Destroy()

	// Should not panic even after destroy
	s.Flush()
	s.MarkDirty()
}

// --- Matrix Concurrency Tests ---

func TestCairoRenderer_MatrixConcurrency(t *testing.T) {
	cr := NewCairoRenderer()
	var wg sync.WaitGroup

	// Run multiple goroutines modifying and reading the matrix
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				m := NewTranslateMatrix(float64(j), float64(j))
				cr.SetMatrix(m)
				_ = cr.GetMatrix()
				cr.Transform(NewScaleMatrix(1.01, 1.01))
			}
		}()
	}

	wg.Wait()
}

// --- Dash Pattern Tests ---

func TestCairoRenderer_SetGetDash(t *testing.T) {
	cr := NewCairoRenderer()

	// Initially empty dash
	dashes, offset := cr.GetDash()
	if len(dashes) != 0 {
		t.Errorf("Expected empty dash pattern initially, got %v", dashes)
	}
	if offset != 0 {
		t.Errorf("Expected offset 0 initially, got %f", offset)
	}

	// Set a dash pattern
	cr.SetDash([]float64{10, 5, 3, 5}, 2.5)

	dashes, offset = cr.GetDash()
	if len(dashes) != 4 {
		t.Errorf("Expected 4 dash elements, got %d", len(dashes))
	}
	if offset != 2.5 {
		t.Errorf("Expected offset 2.5, got %f", offset)
	}

	// Verify dash count
	if cr.GetDashCount() != 4 {
		t.Errorf("Expected dash count 4, got %d", cr.GetDashCount())
	}
}

func TestCairoRenderer_DashSaveRestore(t *testing.T) {
	cr := NewCairoRenderer()

	// Set dash pattern
	cr.SetDash([]float64{10, 5}, 0)

	// Save state
	cr.Save()

	// Modify dash
	cr.SetDash([]float64{20, 10, 5, 10}, 5.0)

	// Verify modification
	if cr.GetDashCount() != 4 {
		t.Errorf("Expected dash count 4 after modification, got %d", cr.GetDashCount())
	}

	// Restore should bring back original dash
	cr.Restore()

	if cr.GetDashCount() != 2 {
		t.Errorf("Expected dash count 2 after restore, got %d", cr.GetDashCount())
	}
}

// --- Miter Limit Tests ---

func TestCairoRenderer_SetGetMiterLimit(t *testing.T) {
	cr := NewCairoRenderer()

	// Default miter limit is 0
	if cr.GetMiterLimit() != 0 {
		t.Errorf("Expected default miter limit 0, got %f", cr.GetMiterLimit())
	}

	// Set miter limit
	cr.SetMiterLimit(10.0)
	if cr.GetMiterLimit() != 10.0 {
		t.Errorf("Expected miter limit 10.0, got %f", cr.GetMiterLimit())
	}

	// Set another value
	cr.SetMiterLimit(5.5)
	if cr.GetMiterLimit() != 5.5 {
		t.Errorf("Expected miter limit 5.5, got %f", cr.GetMiterLimit())
	}
}

func TestCairoRenderer_MiterLimitSaveRestore(t *testing.T) {
	cr := NewCairoRenderer()

	// Set miter limit
	cr.SetMiterLimit(10.0)

	// Save state
	cr.Save()

	// Modify miter limit
	cr.SetMiterLimit(20.0)

	if cr.GetMiterLimit() != 20.0 {
		t.Errorf("Expected miter limit 20.0 after modification, got %f", cr.GetMiterLimit())
	}

	// Restore should bring back original
	cr.Restore()

	if cr.GetMiterLimit() != 10.0 {
		t.Errorf("Expected miter limit 10.0 after restore, got %f", cr.GetMiterLimit())
	}
}

// --- Fill Rule and Operator Tests ---

func TestCairoRenderer_FillRule(t *testing.T) {
	cr := NewCairoRenderer()

	// These are no-ops but should not panic
	cr.SetFillRule(0) // WINDING
	if cr.GetFillRule() != 0 {
		t.Errorf("Expected fill rule 0, got %d", cr.GetFillRule())
	}

	cr.SetFillRule(1) // EVEN_ODD
	// Still returns 0 as it's a no-op
	if cr.GetFillRule() != 0 {
		t.Errorf("Expected fill rule 0 (no-op), got %d", cr.GetFillRule())
	}
}

func TestCairoRenderer_Operator(t *testing.T) {
	cr := NewCairoRenderer()

	// These are no-ops but should not panic
	cr.SetOperator(1) // SOURCE
	if cr.GetOperator() != 2 {
		t.Errorf("Expected operator 2 (OVER default), got %d", cr.GetOperator())
	}

	cr.SetOperator(0) // CLEAR
	// Still returns 2 as it's a no-op
	if cr.GetOperator() != 2 {
		t.Errorf("Expected operator 2 (no-op), got %d", cr.GetOperator())
	}
}

// --- Hit Testing Tests ---

func TestCairoRenderer_InFill(t *testing.T) {
	cr := NewCairoRenderer()

	// Empty path should return false
	if cr.InFill(50, 50) {
		t.Error("Expected InFill to return false for empty path")
	}

	// Create a rectangle path
	cr.Rectangle(0, 0, 100, 100)

	// Point inside should return true
	if !cr.InFill(50, 50) {
		t.Error("Expected InFill to return true for point inside rectangle")
	}

	// Point outside should return false
	if cr.InFill(150, 150) {
		t.Error("Expected InFill to return false for point outside rectangle")
	}
}

func TestCairoRenderer_InStroke(t *testing.T) {
	cr := NewCairoRenderer()
	cr.SetLineWidth(10)

	// Empty path should return false
	if cr.InStroke(50, 50) {
		t.Error("Expected InStroke to return false for empty path")
	}

	// Create a rectangle path
	cr.Rectangle(0, 0, 100, 100)

	// Point on edge (within line width) should return true
	if !cr.InStroke(0, 50) {
		t.Error("Expected InStroke to return true for point on edge")
	}

	// Point far outside should return false
	if cr.InStroke(200, 200) {
		t.Error("Expected InStroke to return false for point far outside")
	}
}

// --- Path Extent Tests ---

func TestCairoRenderer_StrokeExtents(t *testing.T) {
	cr := NewCairoRenderer()
	cr.SetLineWidth(10)

	cr.Rectangle(10, 10, 80, 80)

	x1, y1, x2, y2 := cr.StrokeExtents()

	// With line width of 10, extents should be expanded by 5 on each side
	if x1 > 10 || y1 > 10 {
		t.Errorf("StrokeExtents min should include line width: got (%f, %f)", x1, y1)
	}
	if x2 < 90 || y2 < 90 {
		t.Errorf("StrokeExtents max should include line width: got (%f, %f)", x2, y2)
	}
}

func TestCairoRenderer_FillExtents(t *testing.T) {
	cr := NewCairoRenderer()

	cr.Rectangle(10, 20, 80, 60)

	x1, y1, x2, y2 := cr.FillExtents()

	if x1 != 10 || y1 != 20 {
		t.Errorf("FillExtents min expected (10, 20), got (%f, %f)", x1, y1)
	}
	if x2 != 90 || y2 != 80 {
		t.Errorf("FillExtents max expected (90, 80), got (%f, %f)", x2, y2)
	}
}

// --- Font Extent Tests ---

func TestCairoRenderer_FontExtents(t *testing.T) {
	cr := NewCairoRenderer()
	cr.SetFontSize(20)

	extents := cr.FontExtents()

	if extents.Ascent <= 0 {
		t.Error("Expected positive font ascent")
	}
	if extents.Descent <= 0 {
		t.Error("Expected positive font descent")
	}
	if extents.Height <= 0 {
		t.Error("Expected positive font height")
	}
	if extents.Height < extents.Ascent+extents.Descent {
		t.Error("Font height should be >= ascent + descent")
	}
}

func TestCairoRenderer_GetFontInfo(t *testing.T) {
	cr := NewCairoRenderer()

	// Test default font face
	if cr.GetFontFace() != "GoMono" {
		t.Errorf("Expected default font 'GoMono', got %q", cr.GetFontFace())
	}

	// Set and get font size
	cr.SetFontSize(24)
	if cr.GetFontSize() != 24 {
		t.Errorf("Expected font size 24, got %f", cr.GetFontSize())
	}

	// Test font slant and weight
	if cr.GetFontSlant() != FontSlantNormal {
		t.Errorf("Expected FontSlantNormal, got %v", cr.GetFontSlant())
	}
	if cr.GetFontWeight() != FontWeightNormal {
		t.Errorf("Expected FontWeightNormal, got %v", cr.GetFontWeight())
	}
}

// --- Coordinate Transform Tests ---

func TestCairoRenderer_UserToDevice(t *testing.T) {
	cr := NewCairoRenderer()

	// Without any transforms, user = device
	dx, dy := cr.UserToDevice(10, 20)
	if dx != 10 || dy != 20 {
		t.Errorf("Without transforms, expected (10, 20), got (%f, %f)", dx, dy)
	}

	// With translation
	cr.Translate(100, 50)
	dx, dy = cr.UserToDevice(10, 20)
	if dx != 110 || dy != 70 {
		t.Errorf("With translation, expected (110, 70), got (%f, %f)", dx, dy)
	}
}

func TestCairoRenderer_DeviceToUser(t *testing.T) {
	cr := NewCairoRenderer()

	// With translation
	cr.Translate(100, 50)
	x, y := cr.DeviceToUser(110, 70)
	if math.Abs(x-10) > 0.001 || math.Abs(y-20) > 0.001 {
		t.Errorf("With translation, expected (10, 20), got (%f, %f)", x, y)
	}
}

// --- Sub-path Tests ---

func TestCairoRenderer_NewSubPath(t *testing.T) {
	cr := NewCairoRenderer()

	cr.MoveTo(10, 10)
	cr.LineTo(100, 100)

	// NewSubPath should reset current point
	cr.NewSubPath()

	// This should not panic even without current point
	cr.MoveTo(200, 200)
}

// --- Path Copy/Append Tests ---

func TestCairoRenderer_CopyPath(t *testing.T) {
	cr := NewCairoRenderer()

	// Create a simple path
	cr.MoveTo(0, 0)
	cr.LineTo(100, 100)

	// CopyPath returns empty for now (stub implementation)
	segments := cr.CopyPath()
	if segments == nil {
		t.Error("CopyPath should return non-nil slice")
	}
}

func TestCairoRenderer_AppendPath(t *testing.T) {
	cr := NewCairoRenderer()

	// Create some segments
	segments := []PathSegment{
		{Type: PathMoveTo, X: 10, Y: 10},
		{Type: PathLineTo, X: 100, Y: 100},
		{Type: PathClose},
	}

	// AppendPath should not panic
	cr.AppendPath(segments)

	// Check path was built
	x1, y1, x2, y2 := cr.FillExtents()
	if x1 == 0 && y1 == 0 && x2 == 0 && y2 == 0 {
		t.Error("AppendPath should have built a path with extents")
	}
}
