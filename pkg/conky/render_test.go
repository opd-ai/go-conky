package conky

import (
	"testing"

	"github.com/opd-ai/go-conky/internal/config"
	"github.com/opd-ai/go-conky/internal/render"
)

func TestParseWindowHints(t *testing.T) {
	tests := []struct {
		name            string
		hints           []config.WindowHint
		wantUndecorated bool
		wantFloating    bool
		wantSkipTaskbar bool
		wantSkipPager   bool
	}{
		{
			name:            "empty hints",
			hints:           nil,
			wantUndecorated: false,
			wantFloating:    false,
			wantSkipTaskbar: false,
			wantSkipPager:   false,
		},
		{
			name:            "undecorated only",
			hints:           []config.WindowHint{config.WindowHintUndecorated},
			wantUndecorated: true,
			wantFloating:    false,
			wantSkipTaskbar: false,
			wantSkipPager:   false,
		},
		{
			name:            "above (floating) only",
			hints:           []config.WindowHint{config.WindowHintAbove},
			wantUndecorated: false,
			wantFloating:    true,
			wantSkipTaskbar: false,
			wantSkipPager:   false,
		},
		{
			name:            "skip_taskbar only",
			hints:           []config.WindowHint{config.WindowHintSkipTaskbar},
			wantUndecorated: false,
			wantFloating:    false,
			wantSkipTaskbar: true,
			wantSkipPager:   false,
		},
		{
			name:            "skip_pager only",
			hints:           []config.WindowHint{config.WindowHintSkipPager},
			wantUndecorated: false,
			wantFloating:    false,
			wantSkipTaskbar: false,
			wantSkipPager:   true,
		},
		{
			name: "desktop widget style",
			hints: []config.WindowHint{
				config.WindowHintUndecorated,
				config.WindowHintAbove,
				config.WindowHintSkipTaskbar,
				config.WindowHintSkipPager,
			},
			wantUndecorated: true,
			wantFloating:    true,
			wantSkipTaskbar: true,
			wantSkipPager:   true,
		},
		{
			name: "all hints including unsupported",
			hints: []config.WindowHint{
				config.WindowHintUndecorated,
				config.WindowHintBelow, // Not supported by Ebiten
				config.WindowHintAbove,
				config.WindowHintSticky, // Not supported by Ebiten
				config.WindowHintSkipTaskbar,
				config.WindowHintSkipPager,
			},
			wantUndecorated: true,
			wantFloating:    true,
			wantSkipTaskbar: true,
			wantSkipPager:   true,
		},
		{
			name: "below hint ignored",
			hints: []config.WindowHint{
				config.WindowHintBelow, // Should be ignored (not supported)
			},
			wantUndecorated: false,
			wantFloating:    false,
			wantSkipTaskbar: false,
			wantSkipPager:   false,
		},
		{
			name: "sticky hint ignored",
			hints: []config.WindowHint{
				config.WindowHintSticky, // Should be ignored (not supported)
			},
			wantUndecorated: false,
			wantFloating:    false,
			wantSkipTaskbar: false,
			wantSkipPager:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			undecorated, floating, skipTaskbar, skipPager := parseWindowHints(tt.hints)

			if undecorated != tt.wantUndecorated {
				t.Errorf("undecorated = %v, want %v", undecorated, tt.wantUndecorated)
			}
			if floating != tt.wantFloating {
				t.Errorf("floating = %v, want %v", floating, tt.wantFloating)
			}
			if skipTaskbar != tt.wantSkipTaskbar {
				t.Errorf("skipTaskbar = %v, want %v", skipTaskbar, tt.wantSkipTaskbar)
			}
			if skipPager != tt.wantSkipPager {
				t.Errorf("skipPager = %v, want %v", skipPager, tt.wantSkipPager)
			}
		})
	}
}

func TestConfigToRenderBackgroundMode(t *testing.T) {
	tests := []struct {
		name     string
		input    config.BackgroundMode
		expected render.BackgroundMode
	}{
		{
			name:     "solid mode",
			input:    config.BackgroundModeSolid,
			expected: render.BackgroundModeSolid,
		},
		{
			name:     "none mode maps to none",
			input:    config.BackgroundModeNone,
			expected: render.BackgroundModeNone,
		},
		{
			name:     "transparent mode maps to none",
			input:    config.BackgroundModeTransparent,
			expected: render.BackgroundModeNone,
		},
		{
			name:     "gradient mode maps to solid (not fully implemented in render)",
			input:    config.BackgroundModeGradient,
			expected: render.BackgroundModeSolid,
		},
		{
			name:     "pseudo mode maps to solid (fallback)",
			input:    config.BackgroundModePseudo,
			expected: render.BackgroundModeSolid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := configToRenderBackgroundMode(tt.input)
			if result != tt.expected {
				t.Errorf("configToRenderBackgroundMode(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNewGameRunner(t *testing.T) {
	gr := newGameRunner()
	if gr == nil {
		t.Error("newGameRunner() returned nil")
	}
	// Initially game should be nil (set during run)
	if gr.game != nil {
		t.Error("newGameRunner() should have nil game initially")
	}
}
