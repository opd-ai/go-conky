package render

import (
	"image/color"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Width != 400 {
		t.Errorf("Width = %d, want 400", config.Width)
	}
	if config.Height != 300 {
		t.Errorf("Height = %d, want 300", config.Height)
	}
	if config.Title != "conky-go" {
		t.Errorf("Title = %q, want %q", config.Title, "conky-go")
	}
	if config.UpdateInterval != time.Second {
		t.Errorf("UpdateInterval = %v, want %v", config.UpdateInterval, time.Second)
	}
	if config.BackgroundColor.A != 200 {
		t.Errorf("BackgroundColor.A = %d, want 200", config.BackgroundColor.A)
	}
}

func TestTextLineStruct(t *testing.T) {
	line := TextLine{
		Text:  "Hello World",
		X:     10.5,
		Y:     20.5,
		Color: color.RGBA{R: 255, G: 128, B: 64, A: 255},
	}

	if line.Text != "Hello World" {
		t.Errorf("Text = %q, want %q", line.Text, "Hello World")
	}
	if line.X != 10.5 {
		t.Errorf("X = %v, want 10.5", line.X)
	}
	if line.Y != 20.5 {
		t.Errorf("Y = %v, want 20.5", line.Y)
	}
	if line.Color.R != 255 {
		t.Errorf("Color.R = %d, want 255", line.Color.R)
	}
}

func TestConfigCustomValues(t *testing.T) {
	config := Config{
		Width:           1920,
		Height:          1080,
		Title:           "Custom Title",
		UpdateInterval:  500 * time.Millisecond,
		BackgroundColor: color.RGBA{R: 50, G: 50, B: 50, A: 255},
	}

	if config.Width != 1920 {
		t.Errorf("Width = %d, want 1920", config.Width)
	}
	if config.Height != 1080 {
		t.Errorf("Height = %d, want 1080", config.Height)
	}
	if config.Title != "Custom Title" {
		t.Errorf("Title = %q, want %q", config.Title, "Custom Title")
	}
	if config.UpdateInterval != 500*time.Millisecond {
		t.Errorf("UpdateInterval = %v, want 500ms", config.UpdateInterval)
	}
}

func TestConfigZeroValues(t *testing.T) {
	config := Config{}

	// Verify zero values don't cause issues
	if config.Width != 0 {
		t.Errorf("Width = %d, want 0", config.Width)
	}
	if config.Height != 0 {
		t.Errorf("Height = %d, want 0", config.Height)
	}
	if config.Title != "" {
		t.Errorf("Title = %q, want empty", config.Title)
	}
}

func TestTextLineZeroValues(t *testing.T) {
	line := TextLine{}

	// Verify zero values don't cause issues
	if line.Text != "" {
		t.Errorf("Text = %q, want empty", line.Text)
	}
	if line.X != 0 {
		t.Errorf("X = %v, want 0", line.X)
	}
	if line.Y != 0 {
		t.Errorf("Y = %v, want 0", line.Y)
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name:    "zero width",
			config:  Config{Width: 0, Height: 300},
			wantErr: true,
		},
		{
			name:    "negative width",
			config:  Config{Width: -100, Height: 300},
			wantErr: true,
		},
		{
			name:    "zero height",
			config:  Config{Width: 400, Height: 0},
			wantErr: true,
		},
		{
			name:    "negative height",
			config:  Config{Width: 400, Height: -50},
			wantErr: true,
		},
		{
			name:    "both zero",
			config:  Config{Width: 0, Height: 0},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultConfigDisplayEffects(t *testing.T) {
	config := DefaultConfig()

	// Test new display effect defaults
	if config.DrawBorders {
		t.Error("DrawBorders should be false by default")
	}
	if config.DrawOutline {
		t.Error("DrawOutline should be false by default")
	}
	if config.DrawShades {
		t.Error("DrawShades should be false by default")
	}
	if config.BorderWidth != 1 {
		t.Errorf("BorderWidth = %d, want 1", config.BorderWidth)
	}
	if config.BorderInnerMargin != 5 {
		t.Errorf("BorderInnerMargin = %d, want 5", config.BorderInnerMargin)
	}
	if config.BorderOuterMargin != 5 {
		t.Errorf("BorderOuterMargin = %d, want 5", config.BorderOuterMargin)
	}
	if config.StippledBorders {
		t.Error("StippledBorders should be false by default")
	}

	// Test default colors
	if config.BorderColor != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
		t.Errorf("BorderColor = %v, want white", config.BorderColor)
	}
	if config.OutlineColor != (color.RGBA{R: 0, G: 0, B: 0, A: 255}) {
		t.Errorf("OutlineColor = %v, want black", config.OutlineColor)
	}
	if config.ShadeColor != (color.RGBA{R: 0, G: 0, B: 0, A: 128}) {
		t.Errorf("ShadeColor = %v, want dark gray with 50%% alpha", config.ShadeColor)
	}
}

func TestConfigDisplayEffectsCustomValues(t *testing.T) {
	config := Config{
		Width:             800,
		Height:            600,
		DrawBorders:       true,
		DrawOutline:       true,
		DrawShades:        true,
		BorderWidth:       3,
		BorderInnerMargin: 10,
		BorderOuterMargin: 15,
		StippledBorders:   true,
		BorderColor:       color.RGBA{R: 255, G: 0, B: 0, A: 255},
		OutlineColor:      color.RGBA{R: 0, G: 255, B: 0, A: 255},
		ShadeColor:        color.RGBA{R: 0, G: 0, B: 255, A: 200},
	}

	if !config.DrawBorders {
		t.Error("DrawBorders should be true")
	}
	if !config.DrawOutline {
		t.Error("DrawOutline should be true")
	}
	if !config.DrawShades {
		t.Error("DrawShades should be true")
	}
	if config.BorderWidth != 3 {
		t.Errorf("BorderWidth = %d, want 3", config.BorderWidth)
	}
	if config.BorderInnerMargin != 10 {
		t.Errorf("BorderInnerMargin = %d, want 10", config.BorderInnerMargin)
	}
	if config.BorderOuterMargin != 15 {
		t.Errorf("BorderOuterMargin = %d, want 15", config.BorderOuterMargin)
	}
	if !config.StippledBorders {
		t.Error("StippledBorders should be true")
	}
	if config.BorderColor.R != 255 {
		t.Errorf("BorderColor.R = %d, want 255", config.BorderColor.R)
	}
	if config.OutlineColor.G != 255 {
		t.Errorf("OutlineColor.G = %d, want 255", config.OutlineColor.G)
	}
	if config.ShadeColor.B != 255 {
		t.Errorf("ShadeColor.B = %d, want 255", config.ShadeColor.B)
	}
}

func TestDefaultConfigARGBSettings(t *testing.T) {
	config := DefaultConfig()

	// Test ARGB transparency defaults
	if config.Transparent {
		t.Error("Transparent should be false by default")
	}
	if config.ARGBVisual {
		t.Error("ARGBVisual should be false by default")
	}
	if config.ARGBValue != 255 {
		t.Errorf("ARGBValue = %d, want 255 (fully opaque)", config.ARGBValue)
	}
}

func TestConfigARGBCustomValues(t *testing.T) {
	tests := []struct {
		name        string
		transparent bool
		argbVisual  bool
		argbValue   int
	}{
		{
			name:        "transparency enabled",
			transparent: true,
			argbVisual:  false,
			argbValue:   255,
		},
		{
			name:        "argb visual enabled",
			transparent: false,
			argbVisual:  true,
			argbValue:   128,
		},
		{
			name:        "both enabled with semi-transparent",
			transparent: true,
			argbVisual:  true,
			argbValue:   100,
		},
		{
			name:        "fully transparent",
			transparent: true,
			argbVisual:  true,
			argbValue:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				Width:       400,
				Height:      300,
				Transparent: tt.transparent,
				ARGBVisual:  tt.argbVisual,
				ARGBValue:   tt.argbValue,
			}

			if config.Transparent != tt.transparent {
				t.Errorf("Transparent = %v, want %v", config.Transparent, tt.transparent)
			}
			if config.ARGBVisual != tt.argbVisual {
				t.Errorf("ARGBVisual = %v, want %v", config.ARGBVisual, tt.argbVisual)
			}
			if config.ARGBValue != tt.argbValue {
				t.Errorf("ARGBValue = %d, want %d", config.ARGBValue, tt.argbValue)
			}
		})
	}
}
