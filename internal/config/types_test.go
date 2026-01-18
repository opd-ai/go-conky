package config

import (
	"image/color"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Test window defaults
	if cfg.Window.Width != DefaultWidth {
		t.Errorf("expected window width %d, got %d", DefaultWidth, cfg.Window.Width)
	}
	if cfg.Window.Height != DefaultHeight {
		t.Errorf("expected window height %d, got %d", DefaultHeight, cfg.Window.Height)
	}
	if cfg.Window.Type != WindowTypeNormal {
		t.Errorf("expected window type %v, got %v", WindowTypeNormal, cfg.Window.Type)
	}
	if !cfg.Window.OwnWindow {
		t.Error("expected OwnWindow to be true")
	}

	// Test display defaults
	if cfg.Display.UpdateInterval != DefaultUpdateInterval {
		t.Errorf("expected update interval %v, got %v", DefaultUpdateInterval, cfg.Display.UpdateInterval)
	}
	if cfg.Display.Font != DefaultFont {
		t.Errorf("expected font %q, got %q", DefaultFont, cfg.Display.Font)
	}
	if cfg.Display.FontSize != DefaultFontSize {
		t.Errorf("expected font size %f, got %f", DefaultFontSize, cfg.Display.FontSize)
	}
	if !cfg.Display.DoubleBuffer {
		t.Error("expected DoubleBuffer to be true")
	}

	// Test color defaults
	if cfg.Colors.Default != DefaultTextColor {
		t.Errorf("expected default color %v, got %v", DefaultTextColor, cfg.Colors.Default)
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name:    "valid default config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "negative window width",
			config: Config{
				Window: WindowConfig{Width: -1},
			},
			wantErr: true,
		},
		{
			name: "negative window height",
			config: Config{
				Window: WindowConfig{Height: -1},
			},
			wantErr: true,
		},
		{
			name: "negative update interval",
			config: Config{
				Display: DisplayConfig{UpdateInterval: -time.Second},
			},
			wantErr: true,
		},
		{
			name: "zero dimensions valid",
			config: Config{
				Window:  WindowConfig{Width: 0, Height: 0},
				Display: DisplayConfig{UpdateInterval: 0},
			},
			wantErr: false,
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

func TestWindowTypeString(t *testing.T) {
	tests := []struct {
		wt   WindowType
		want string
	}{
		{WindowTypeNormal, "normal"},
		{WindowTypeDesktop, "desktop"},
		{WindowTypeDock, "dock"},
		{WindowTypePanel, "panel"},
		{WindowTypeOverride, "override"},
		{WindowType(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.wt.String(); got != tt.want {
				t.Errorf("WindowType.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseWindowType(t *testing.T) {
	tests := []struct {
		input   string
		want    WindowType
		wantErr bool
	}{
		{"normal", WindowTypeNormal, false},
		{"desktop", WindowTypeDesktop, false},
		{"dock", WindowTypeDock, false},
		{"panel", WindowTypePanel, false},
		{"override", WindowTypeOverride, false},
		{"invalid", WindowTypeNormal, true},
		{"", WindowTypeNormal, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseWindowType(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseWindowType(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseWindowType(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestWindowHintString(t *testing.T) {
	tests := []struct {
		wh   WindowHint
		want string
	}{
		{WindowHintUndecorated, "undecorated"},
		{WindowHintBelow, "below"},
		{WindowHintAbove, "above"},
		{WindowHintSticky, "sticky"},
		{WindowHintSkipTaskbar, "skip_taskbar"},
		{WindowHintSkipPager, "skip_pager"},
		{WindowHint(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.wh.String(); got != tt.want {
				t.Errorf("WindowHint.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseWindowHint(t *testing.T) {
	tests := []struct {
		input   string
		want    WindowHint
		wantErr bool
	}{
		{"undecorated", WindowHintUndecorated, false},
		{"below", WindowHintBelow, false},
		{"above", WindowHintAbove, false},
		{"sticky", WindowHintSticky, false},
		{"skip_taskbar", WindowHintSkipTaskbar, false},
		{"skip_pager", WindowHintSkipPager, false},
		{"invalid", WindowHintUndecorated, true},
		{"", WindowHintUndecorated, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseWindowHint(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseWindowHint(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseWindowHint(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestAlignmentString(t *testing.T) {
	tests := []struct {
		a    Alignment
		want string
	}{
		{AlignmentTopLeft, "top_left"},
		{AlignmentTopMiddle, "top_middle"},
		{AlignmentTopRight, "top_right"},
		{AlignmentMiddleLeft, "middle_left"},
		{AlignmentMiddleMiddle, "middle_middle"},
		{AlignmentMiddleRight, "middle_right"},
		{AlignmentBottomLeft, "bottom_left"},
		{AlignmentBottomMiddle, "bottom_middle"},
		{AlignmentBottomRight, "bottom_right"},
		{Alignment(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.a.String(); got != tt.want {
				t.Errorf("Alignment.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseAlignment(t *testing.T) {
	tests := []struct {
		input   string
		want    Alignment
		wantErr bool
	}{
		{"top_left", AlignmentTopLeft, false},
		{"tl", AlignmentTopLeft, false},
		{"top_middle", AlignmentTopMiddle, false},
		{"tm", AlignmentTopMiddle, false},
		{"top_center", AlignmentTopMiddle, false},
		{"tc", AlignmentTopMiddle, false},
		{"top_right", AlignmentTopRight, false},
		{"tr", AlignmentTopRight, false},
		{"middle_left", AlignmentMiddleLeft, false},
		{"ml", AlignmentMiddleLeft, false},
		{"middle_middle", AlignmentMiddleMiddle, false},
		{"mm", AlignmentMiddleMiddle, false},
		{"middle_center", AlignmentMiddleMiddle, false},
		{"mc", AlignmentMiddleMiddle, false},
		{"center", AlignmentMiddleMiddle, false},
		{"c", AlignmentMiddleMiddle, false},
		{"middle_right", AlignmentMiddleRight, false},
		{"mr", AlignmentMiddleRight, false},
		{"bottom_left", AlignmentBottomLeft, false},
		{"bl", AlignmentBottomLeft, false},
		{"bottom_middle", AlignmentBottomMiddle, false},
		{"bm", AlignmentBottomMiddle, false},
		{"bottom_center", AlignmentBottomMiddle, false},
		{"bc", AlignmentBottomMiddle, false},
		{"bottom_right", AlignmentBottomRight, false},
		{"br", AlignmentBottomRight, false},
		{"invalid", AlignmentTopLeft, true},
		{"", AlignmentTopLeft, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseAlignment(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseAlignment(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseAlignment(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestDefaultWindowConfig(t *testing.T) {
	wc := DefaultWindowConfig()
	dc := DefaultConfig()

	// Compare individual fields since struct contains slice
	if wc.Width != dc.Window.Width {
		t.Error("DefaultWindowConfig().Width should match DefaultConfig().Window.Width")
	}
	if wc.Height != dc.Window.Height {
		t.Error("DefaultWindowConfig().Height should match DefaultConfig().Window.Height")
	}
	if wc.Type != dc.Window.Type {
		t.Error("DefaultWindowConfig().Type should match DefaultConfig().Window.Type")
	}
	if wc.OwnWindow != dc.Window.OwnWindow {
		t.Error("DefaultWindowConfig().OwnWindow should match DefaultConfig().Window.OwnWindow")
	}
	if wc.Transparent != dc.Window.Transparent {
		t.Error("DefaultWindowConfig().Transparent should match DefaultConfig().Window.Transparent")
	}
}

func TestDefaultDisplayConfig(t *testing.T) {
	dc := DefaultDisplayConfig()
	cfg := DefaultConfig()

	if dc != cfg.Display {
		t.Error("DefaultDisplayConfig() should match DefaultConfig().Display")
	}
}

func TestDefaultColorConfig(t *testing.T) {
	cc := DefaultColorConfig()
	cfg := DefaultConfig()

	if cc != cfg.Colors {
		t.Error("DefaultColorConfig() should match DefaultConfig().Colors")
	}
}

func TestColorDefaults(t *testing.T) {
	// Test that default colors are properly initialized
	if DefaultTextColor != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
		t.Error("DefaultTextColor should be white")
	}
	if DefaultGrey != (color.RGBA{R: 128, G: 128, B: 128, A: 255}) {
		t.Error("DefaultGrey should be grey")
	}
	if TransparentColor != (color.RGBA{R: 0, G: 0, B: 0, A: 0}) {
		t.Error("TransparentColor should be transparent")
	}
}

func TestConfigCopy(t *testing.T) {
	// Test that Config can be safely copied
	original := DefaultConfig()
	original.Window.Width = 500
	original.Display.Font = "Custom Font"
	original.Text.Template = []string{"line1", "line2"}

	copied := original

	// Modify copied to ensure independence
	copied.Window.Width = 600
	copied.Display.Font = "Other Font"

	if original.Window.Width != 500 {
		t.Error("Original config should not be affected by copy modification (Width)")
	}
	if original.Display.Font != "Custom Font" {
		t.Error("Original config should not be affected by copy modification (Font)")
	}

	// Note: slices are not deep copied by value copy
	// This is expected Go behavior
}

func TestBackgroundModeString(t *testing.T) {
	tests := []struct {
		mode BackgroundMode
		want string
	}{
		{BackgroundModeSolid, "solid"},
		{BackgroundModeNone, "none"},
		{BackgroundModeTransparent, "transparent"},
		{BackgroundModeGradient, "gradient"},
		{BackgroundModePseudo, "pseudo"},
		{BackgroundMode(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.mode.String(); got != tt.want {
				t.Errorf("BackgroundMode.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseBackgroundMode(t *testing.T) {
	tests := []struct {
		input   string
		want    BackgroundMode
		wantErr bool
	}{
		{"solid", BackgroundModeSolid, false},
		{"Solid", BackgroundModeSolid, false},
		{"SOLID", BackgroundModeSolid, false},
		{"none", BackgroundModeNone, false},
		{"None", BackgroundModeNone, false},
		{"NONE", BackgroundModeNone, false},
		{"transparent", BackgroundModeTransparent, false},
		{"Transparent", BackgroundModeTransparent, false},
		{"gradient", BackgroundModeGradient, false},
		{"Gradient", BackgroundModeGradient, false},
		{"GRADIENT", BackgroundModeGradient, false},
		{"pseudo", BackgroundModePseudo, false},
		{"Pseudo", BackgroundModePseudo, false},
		{"PSEUDO", BackgroundModePseudo, false},
		{"pseudo-transparent", BackgroundModePseudo, false},
		{"pseudo_transparent", BackgroundModePseudo, false},
		{"", BackgroundModeSolid, false}, // Empty defaults to solid
		{"invalid", BackgroundModeSolid, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseBackgroundMode(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseBackgroundMode(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseBackgroundMode(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestGradientDirectionString(t *testing.T) {
	tests := []struct {
		dir  GradientDirection
		want string
	}{
		{GradientDirectionVertical, "vertical"},
		{GradientDirectionHorizontal, "horizontal"},
		{GradientDirectionDiagonal, "diagonal"},
		{GradientDirectionRadial, "radial"},
		{GradientDirection(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.dir.String(); got != tt.want {
				t.Errorf("GradientDirection.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseGradientDirection(t *testing.T) {
	tests := []struct {
		input   string
		want    GradientDirection
		wantErr bool
	}{
		{"vertical", GradientDirectionVertical, false},
		{"Vertical", GradientDirectionVertical, false},
		{"v", GradientDirectionVertical, false},
		{"", GradientDirectionVertical, false}, // Empty defaults to vertical
		{"horizontal", GradientDirectionHorizontal, false},
		{"Horizontal", GradientDirectionHorizontal, false},
		{"h", GradientDirectionHorizontal, false},
		{"diagonal", GradientDirectionDiagonal, false},
		{"Diagonal", GradientDirectionDiagonal, false},
		{"d", GradientDirectionDiagonal, false},
		{"radial", GradientDirectionRadial, false},
		{"Radial", GradientDirectionRadial, false},
		{"r", GradientDirectionRadial, false},
		{"invalid", GradientDirectionVertical, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseGradientDirection(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseGradientDirection(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseGradientDirection(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestGradientConfig(t *testing.T) {
	startColor := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	endColor := color.RGBA{R: 0, G: 0, B: 255, A: 255}

	gc := GradientConfig{
		StartColor: startColor,
		EndColor:   endColor,
		Direction:  GradientDirectionVertical,
	}

	if gc.StartColor != startColor {
		t.Errorf("StartColor = %v, want %v", gc.StartColor, startColor)
	}
	if gc.EndColor != endColor {
		t.Errorf("EndColor = %v, want %v", gc.EndColor, endColor)
	}
	if gc.Direction != GradientDirectionVertical {
		t.Errorf("Direction = %v, want GradientDirectionVertical", gc.Direction)
	}
}

func TestDefaultConfigHasBackgroundMode(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Window.BackgroundMode != BackgroundModeSolid {
		t.Errorf("Default BackgroundMode = %v, want BackgroundModeSolid", cfg.Window.BackgroundMode)
	}
	if cfg.Window.BackgroundColour != DefaultBackgroundColour {
		t.Errorf("Default BackgroundColour = %v, want %v", cfg.Window.BackgroundColour, DefaultBackgroundColour)
	}
}
