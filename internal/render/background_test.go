package render

import (
	"image/color"
	"testing"
)

func TestNewSolidBackground(t *testing.T) {
	c := color.RGBA{R: 100, G: 150, B: 200, A: 255}
	bg := NewSolidBackground(c)

	if bg.Mode() != BackgroundModeSolid {
		t.Errorf("Mode() = %v, want BackgroundModeSolid", bg.Mode())
	}

	if bg.Color() != c {
		t.Errorf("Color() = %v, want %v", bg.Color(), c)
	}
}

func TestSolidBackgroundWithARGB(t *testing.T) {
	tests := []struct {
		name          string
		initialColor  color.RGBA
		argbEnabled   bool
		argbValue     int
		wantStoredVal int
	}{
		{
			name:          "ARGB disabled",
			initialColor:  color.RGBA{R: 0, G: 0, B: 0, A: 200},
			argbEnabled:   false,
			argbValue:     100,
			wantStoredVal: 100, // value stored, but ARGB is disabled
		},
		{
			name:          "ARGB enabled with valid value",
			initialColor:  color.RGBA{R: 0, G: 0, B: 0, A: 200},
			argbEnabled:   true,
			argbValue:     128,
			wantStoredVal: 128,
		},
		{
			name:          "ARGB enabled with zero (fully transparent)",
			initialColor:  color.RGBA{R: 0, G: 0, B: 0, A: 200},
			argbEnabled:   true,
			argbValue:     0,
			wantStoredVal: 0,
		},
		{
			name:          "ARGB enabled with max (fully opaque)",
			initialColor:  color.RGBA{R: 0, G: 0, B: 0, A: 200},
			argbEnabled:   true,
			argbValue:     255,
			wantStoredVal: 255,
		},
		{
			name:          "ARGB enabled with negative value (clamped to 0)",
			initialColor:  color.RGBA{R: 0, G: 0, B: 0, A: 200},
			argbEnabled:   true,
			argbValue:     -50,
			wantStoredVal: 0, // clamped
		},
		{
			name:          "ARGB enabled with value over 255 (clamped to 255)",
			initialColor:  color.RGBA{R: 0, G: 0, B: 0, A: 200},
			argbEnabled:   true,
			argbValue:     500,
			wantStoredVal: 255, // clamped
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bg := NewSolidBackground(tt.initialColor).WithARGB(tt.argbEnabled, tt.argbValue)

			// Verify mode is correct
			if bg.Mode() != BackgroundModeSolid {
				t.Errorf("Mode() = %v, want BackgroundModeSolid", bg.Mode())
			}

			// Verify the stored values
			if bg.argbValue != tt.wantStoredVal {
				t.Errorf("argbValue = %d, want %d", bg.argbValue, tt.wantStoredVal)
			}
			if bg.argbOn != tt.argbEnabled {
				t.Errorf("argbOn = %v, want %v", bg.argbOn, tt.argbEnabled)
			}
		})
	}
}

func TestNewNoneBackground(t *testing.T) {
	bg := NewNoneBackground()

	if bg.Mode() != BackgroundModeNone {
		t.Errorf("Mode() = %v, want BackgroundModeNone", bg.Mode())
	}
}

func TestNewBackgroundRenderer(t *testing.T) {
	tests := []struct {
		name       string
		mode       BackgroundMode
		color      color.RGBA
		argbVisual bool
		argbValue  int
		wantMode   BackgroundMode
	}{
		{
			name:       "solid mode",
			mode:       BackgroundModeSolid,
			color:      color.RGBA{R: 100, G: 100, B: 100, A: 200},
			argbVisual: false,
			argbValue:  255,
			wantMode:   BackgroundModeSolid,
		},
		{
			name:       "none mode",
			mode:       BackgroundModeNone,
			color:      color.RGBA{R: 100, G: 100, B: 100, A: 200},
			argbVisual: false,
			argbValue:  255,
			wantMode:   BackgroundModeNone,
		},
		{
			name:       "solid mode with ARGB",
			mode:       BackgroundModeSolid,
			color:      color.RGBA{R: 0, G: 0, B: 0, A: 200},
			argbVisual: true,
			argbValue:  128,
			wantMode:   BackgroundModeSolid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := NewBackgroundRenderer(tt.mode, tt.color, tt.argbVisual, tt.argbValue)
			if renderer.Mode() != tt.wantMode {
				t.Errorf("Mode() = %v, want %v", renderer.Mode(), tt.wantMode)
			}
		})
	}
}

func TestSolidBackgroundPreservesRGB(t *testing.T) {
	// Test that color is preserved when stored
	c := color.RGBA{R: 100, G: 150, B: 200, A: 255}
	bg := NewSolidBackground(c)

	got := bg.Color()
	if got.R != c.R || got.G != c.G || got.B != c.B {
		t.Errorf("Color() RGB = (%d, %d, %d), want (%d, %d, %d)",
			got.R, got.G, got.B, c.R, c.G, c.B)
	}
}

func TestBackgroundModeConstants(t *testing.T) {
	// Verify mode constants have expected values
	if BackgroundModeSolid != 0 {
		t.Errorf("BackgroundModeSolid = %v, want 0", BackgroundModeSolid)
	}
	if BackgroundModeNone != 1 {
		t.Errorf("BackgroundModeNone = %v, want 1", BackgroundModeNone)
	}
	if BackgroundModeGradient != 2 {
		t.Errorf("BackgroundModeGradient = %v, want 2", BackgroundModeGradient)
	}
}

func TestSolidBackgroundWithARGBChaining(t *testing.T) {
	// Test method chaining returns same instance
	c := color.RGBA{R: 0, G: 0, B: 0, A: 200}
	bg := NewSolidBackground(c)
	returned := bg.WithARGB(true, 128)

	if bg != returned {
		t.Error("WithARGB() should return same instance for chaining")
	}
}

func TestNoneBackgroundMode(t *testing.T) {
	// Verify NoneBackground reports correct mode
	bg := NewNoneBackground()
	if bg.Mode() != BackgroundModeNone {
		t.Errorf("NoneBackground.Mode() = %v, want BackgroundModeNone", bg.Mode())
	}
}

func TestNewGradientBackground(t *testing.T) {
	startColor := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	endColor := color.RGBA{R: 0, G: 0, B: 255, A: 255}

	bg := NewGradientBackground(startColor, endColor, GradientDirectionVertical)

	if bg.Mode() != BackgroundModeGradient {
		t.Errorf("Mode() = %v, want BackgroundModeGradient", bg.Mode())
	}
	if bg.StartColor() != startColor {
		t.Errorf("StartColor() = %v, want %v", bg.StartColor(), startColor)
	}
	if bg.EndColor() != endColor {
		t.Errorf("EndColor() = %v, want %v", bg.EndColor(), endColor)
	}
	if bg.Direction() != GradientDirectionVertical {
		t.Errorf("Direction() = %v, want GradientDirectionVertical", bg.Direction())
	}
}

func TestGradientBackgroundDirections(t *testing.T) {
	tests := []struct {
		name      string
		direction GradientDirection
	}{
		{"vertical", GradientDirectionVertical},
		{"horizontal", GradientDirectionHorizontal},
		{"diagonal", GradientDirectionDiagonal},
		{"radial", GradientDirectionRadial},
	}

	startColor := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	endColor := color.RGBA{R: 0, G: 0, B: 255, A: 255}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bg := NewGradientBackground(startColor, endColor, tt.direction)
			if bg.Direction() != tt.direction {
				t.Errorf("Direction() = %v, want %v", bg.Direction(), tt.direction)
			}
		})
	}
}

func TestGradientBackgroundWithARGB(t *testing.T) {
	tests := []struct {
		name          string
		argbEnabled   bool
		argbValue     int
		wantStoredVal int
	}{
		{
			name:          "ARGB disabled",
			argbEnabled:   false,
			argbValue:     100,
			wantStoredVal: 100,
		},
		{
			name:          "ARGB enabled with valid value",
			argbEnabled:   true,
			argbValue:     128,
			wantStoredVal: 128,
		},
		{
			name:          "ARGB enabled with zero",
			argbEnabled:   true,
			argbValue:     0,
			wantStoredVal: 0,
		},
		{
			name:          "ARGB enabled with max",
			argbEnabled:   true,
			argbValue:     255,
			wantStoredVal: 255,
		},
		{
			name:          "ARGB enabled with negative (clamped)",
			argbEnabled:   true,
			argbValue:     -50,
			wantStoredVal: 0,
		},
		{
			name:          "ARGB enabled with over 255 (clamped)",
			argbEnabled:   true,
			argbValue:     500,
			wantStoredVal: 255,
		},
	}

	startColor := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	endColor := color.RGBA{R: 0, G: 0, B: 255, A: 255}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bg := NewGradientBackground(startColor, endColor, GradientDirectionVertical).
				WithARGB(tt.argbEnabled, tt.argbValue)

			if bg.argbValue != tt.wantStoredVal {
				t.Errorf("argbValue = %d, want %d", bg.argbValue, tt.wantStoredVal)
			}
			if bg.argbOn != tt.argbEnabled {
				t.Errorf("argbOn = %v, want %v", bg.argbOn, tt.argbEnabled)
			}
		})
	}
}

func TestGradientBackgroundWithARGBChaining(t *testing.T) {
	startColor := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	endColor := color.RGBA{R: 0, G: 0, B: 255, A: 255}

	bg := NewGradientBackground(startColor, endColor, GradientDirectionVertical)
	returned := bg.WithARGB(true, 128)

	if bg != returned {
		t.Error("WithARGB() should return same instance for chaining")
	}
}

func TestGradientBackgroundLerpColor(t *testing.T) {
	startColor := color.RGBA{R: 0, G: 0, B: 0, A: 0}
	endColor := color.RGBA{R: 200, G: 100, B: 50, A: 250}

	bg := NewGradientBackground(startColor, endColor, GradientDirectionVertical)

	tests := []struct {
		name  string
		t     float64
		wantR uint8
		wantG uint8
		wantB uint8
		wantA uint8
	}{
		{"start", 0.0, 0, 0, 0, 0},
		{"end", 1.0, 200, 100, 50, 250},
		{"middle", 0.5, 100, 50, 25, 125},
		{"quarter", 0.25, 50, 25, 12, 62},
		{"clamped below", -0.5, 0, 0, 0, 0},
		{"clamped above", 1.5, 200, 100, 50, 250},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bg.lerpColor(tt.t)
			// Allow small rounding differences
			if absDiffU8(got.R, tt.wantR) > 1 || absDiffU8(got.G, tt.wantG) > 1 ||
				absDiffU8(got.B, tt.wantB) > 1 || absDiffU8(got.A, tt.wantA) > 1 {
				t.Errorf("lerpColor(%v) = (%d, %d, %d, %d), want (%d, %d, %d, %d)",
					tt.t, got.R, got.G, got.B, got.A, tt.wantR, tt.wantG, tt.wantB, tt.wantA)
			}
		})
	}
}

func TestGradientBackgroundLerpColorWithARGB(t *testing.T) {
	startColor := color.RGBA{R: 0, G: 0, B: 0, A: 255}
	endColor := color.RGBA{R: 255, G: 255, B: 255, A: 255}

	bg := NewGradientBackground(startColor, endColor, GradientDirectionVertical).
		WithARGB(true, 128)

	// With ARGB enabled, alpha should always be the ARGB value
	for _, t_val := range []float64{0.0, 0.5, 1.0} {
		got := bg.lerpColor(t_val)
		if got.A != 128 {
			t.Errorf("lerpColor(%v).A = %d, want 128 (ARGB override)", t_val, got.A)
		}
	}
}

func TestGradientBackgroundInterpolationFactor(t *testing.T) {
	startColor := color.RGBA{R: 0, G: 0, B: 0, A: 255}
	endColor := color.RGBA{R: 255, G: 255, B: 255, A: 255}

	tests := []struct {
		name      string
		direction GradientDirection
		x, y      int
		w, h      int
		wantT     float64
		tolerance float64
	}{
		// Vertical: t = y / (h-1)
		{"vertical top", GradientDirectionVertical, 50, 0, 100, 100, 0.0, 0.01},
		{"vertical bottom", GradientDirectionVertical, 50, 99, 100, 100, 1.0, 0.01},
		{"vertical middle", GradientDirectionVertical, 50, 50, 100, 101, 0.5, 0.01},

		// Horizontal: t = x / (w-1)
		{"horizontal left", GradientDirectionHorizontal, 0, 50, 100, 100, 0.0, 0.01},
		{"horizontal right", GradientDirectionHorizontal, 99, 50, 100, 100, 1.0, 0.01},
		{"horizontal middle", GradientDirectionHorizontal, 50, 50, 101, 100, 0.5, 0.01},

		// Diagonal: t = (x/(w-1) + y/(h-1)) / 2
		{"diagonal top-left", GradientDirectionDiagonal, 0, 0, 100, 100, 0.0, 0.01},
		{"diagonal bottom-right", GradientDirectionDiagonal, 99, 99, 100, 100, 1.0, 0.01},
		{"diagonal center", GradientDirectionDiagonal, 50, 50, 101, 101, 0.5, 0.01},

		// Radial: t = distance from center / max distance
		{"radial center", GradientDirectionRadial, 50, 50, 100, 100, 0.0, 0.01},
		{"radial corner", GradientDirectionRadial, 0, 0, 100, 100, 1.0, 0.01},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bg := NewGradientBackground(startColor, endColor, tt.direction)
			got := bg.interpolationFactor(tt.x, tt.y, tt.w, tt.h)

			if absDiffFloat(got-tt.wantT) > tt.tolerance {
				t.Errorf("interpolationFactor(%d, %d, %d, %d) = %f, want %f (±%f)",
					tt.x, tt.y, tt.w, tt.h, got, tt.wantT, tt.tolerance)
			}
		})
	}
}

func TestNewGradientBackgroundRenderer(t *testing.T) {
	startColor := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	endColor := color.RGBA{R: 0, G: 0, B: 255, A: 255}

	renderer := NewGradientBackgroundRenderer(startColor, endColor, GradientDirectionVertical, true, 128)

	if renderer.Mode() != BackgroundModeGradient {
		t.Errorf("Mode() = %v, want BackgroundModeGradient", renderer.Mode())
	}

	// Verify it's a GradientBackground
	gb, ok := renderer.(*GradientBackground)
	if !ok {
		t.Fatal("expected *GradientBackground")
	}
	if gb.argbOn != true {
		t.Error("expected argbOn = true")
	}
	if gb.argbValue != 128 {
		t.Errorf("argbValue = %d, want 128", gb.argbValue)
	}
}

func TestGradientDirectionConstants(t *testing.T) {
	if GradientDirectionVertical != 0 {
		t.Errorf("GradientDirectionVertical = %v, want 0", GradientDirectionVertical)
	}
	if GradientDirectionHorizontal != 1 {
		t.Errorf("GradientDirectionHorizontal = %v, want 1", GradientDirectionHorizontal)
	}
	if GradientDirectionDiagonal != 2 {
		t.Errorf("GradientDirectionDiagonal = %v, want 2", GradientDirectionDiagonal)
	}
	if GradientDirectionRadial != 3 {
		t.Errorf("GradientDirectionRadial = %v, want 3", GradientDirectionRadial)
	}
}

// Helper functions for tests
func absDiffU8(a, b uint8) uint8 {
	if a > b {
		return a - b
	}
	return b - a
}

func absDiffFloat(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

