package conky

import (
	"testing"

	"github.com/opd-ai/go-conky/internal/config"
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
				config.WindowHintBelow,  // Not supported by Ebiten
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
