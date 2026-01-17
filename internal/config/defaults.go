package config

import (
	"image/color"
	"time"
)

// Default values for configuration options.
const (
	// DefaultUpdateInterval is the default time between updates (1 second).
	DefaultUpdateInterval = time.Second
	// DefaultWidth is the default window width in pixels.
	DefaultWidth = 200
	// DefaultHeight is the default window height in pixels.
	DefaultHeight = 100
	// DefaultFont is the default font specification.
	DefaultFont = "DejaVu Sans Mono"
	// DefaultFontSize is the default font size in points.
	DefaultFontSize = 10.0
)

// Default colors.
var (
	// DefaultTextColor is the default text color (white).
	DefaultTextColor = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	// DefaultGrey is the default grey color used for labels.
	DefaultGrey = color.RGBA{R: 128, G: 128, B: 128, A: 255}
	// TransparentColor represents fully transparent.
	TransparentColor = color.RGBA{R: 0, G: 0, B: 0, A: 0}
)

// defaultWindowConfig returns a WindowConfig with sensible default values.
// It is used by DefaultConfig and DefaultWindowConfig to avoid duplicating logic.
func defaultWindowConfig() WindowConfig {
	return WindowConfig{
		OwnWindow:   true,
		Type:        WindowTypeNormal,
		Transparent: false,
		Hints:       nil,
		Width:       DefaultWidth,
		Height:      DefaultHeight,
		X:           0,
		Y:           0,
		Alignment:   AlignmentTopLeft,
	}
}

// defaultDisplayConfig returns a DisplayConfig with sensible default values.
// It is used by DefaultConfig and DefaultDisplayConfig to avoid duplicating logic.
func defaultDisplayConfig() DisplayConfig {
	return DisplayConfig{
		Background:        false,
		DoubleBuffer:      true,
		UpdateInterval:    DefaultUpdateInterval,
		Font:              DefaultFont,
		FontSize:          DefaultFontSize,
		DrawBorders:       false,
		DrawOutline:       false,
		DrawShades:        true, // Conky defaults to shades enabled
		BorderWidth:       1,
		BorderInnerMargin: 5,
		BorderOuterMargin: 5,
		StippledBorders:   false,
	}
}

// defaultColorConfig returns a ColorConfig with sensible default values.
// It is used by DefaultConfig and DefaultColorConfig to avoid duplicating logic.
func defaultColorConfig() ColorConfig {
	return ColorConfig{
		Default: DefaultTextColor,
		Color0:  DefaultTextColor,
		Color1:  DefaultGrey,
		Color2:  TransparentColor,
		Color3:  TransparentColor,
		Color4:  TransparentColor,
		Color5:  TransparentColor,
		Color6:  TransparentColor,
		Color7:  TransparentColor,
		Color8:  TransparentColor,
		Color9:  TransparentColor,
	}
}

// DefaultConfig returns a Config with sensible default values.
// These defaults mirror typical Conky configuration defaults.
func DefaultConfig() Config {
	return Config{
		Window:  defaultWindowConfig(),
		Display: defaultDisplayConfig(),
		Text: TextConfig{
			Template: nil,
		},
		Colors: defaultColorConfig(),
	}
}

// DefaultWindowConfig returns a WindowConfig with default values.
func DefaultWindowConfig() WindowConfig {
	return defaultWindowConfig()
}

// DefaultDisplayConfig returns a DisplayConfig with default values.
func DefaultDisplayConfig() DisplayConfig {
	return defaultDisplayConfig()
}

// DefaultColorConfig returns a ColorConfig with default values.
func DefaultColorConfig() ColorConfig {
	return defaultColorConfig()
}
