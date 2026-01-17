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

