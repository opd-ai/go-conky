package config

import (
	"image/color"
	"testing"
	"time"
)

func TestNewLegacyParser(t *testing.T) {
	p := NewLegacyParser()
	if p == nil {
		t.Error("NewLegacyParser returned nil")
	}
}

func TestLegacyParserParseBasic(t *testing.T) {
	content := `# Sample legacy config
background no
font DejaVu Sans Mono:size=10
update_interval 1.0
double_buffer yes
own_window yes
own_window_type normal
own_window_transparent yes
own_window_hints undecorated,below,sticky

TEXT
${color grey}System Monitor$color
${hr 2}
`
	p := NewLegacyParser()
	cfg, err := p.Parse([]byte(content))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check parsed values
	if cfg.Display.Background != false {
		t.Errorf("expected background=false, got %v", cfg.Display.Background)
	}
	if cfg.Display.Font != "DejaVu Sans Mono:size=10" {
		t.Errorf("expected font 'DejaVu Sans Mono:size=10', got %q", cfg.Display.Font)
	}
	if cfg.Display.UpdateInterval != time.Second {
		t.Errorf("expected update_interval=1s, got %v", cfg.Display.UpdateInterval)
	}
	if cfg.Display.DoubleBuffer != true {
		t.Errorf("expected double_buffer=true, got %v", cfg.Display.DoubleBuffer)
	}
	if cfg.Window.OwnWindow != true {
		t.Errorf("expected own_window=true, got %v", cfg.Window.OwnWindow)
	}
	if cfg.Window.Type != WindowTypeNormal {
		t.Errorf("expected window type normal, got %v", cfg.Window.Type)
	}
	if cfg.Window.Transparent != true {
		t.Errorf("expected transparent=true, got %v", cfg.Window.Transparent)
	}
	if len(cfg.Window.Hints) != 3 {
		t.Errorf("expected 3 hints, got %d", len(cfg.Window.Hints))
	}
}

func TestLegacyParserParseTextSection(t *testing.T) {
	content := `background yes

TEXT
Line 1
Line 2
Line 3
`
	p := NewLegacyParser()
	cfg, err := p.Parse([]byte(content))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(cfg.Text.Template) != 3 {
		t.Errorf("expected 3 text lines, got %d", len(cfg.Text.Template))
	}
	if cfg.Text.Template[0] != "Line 1" {
		t.Errorf("expected 'Line 1', got %q", cfg.Text.Template[0])
	}
}

func TestLegacyParserPreservesEmptyLinesInText(t *testing.T) {
	content := `background yes

TEXT
Line 1

Line 3
`
	p := NewLegacyParser()
	cfg, err := p.Parse([]byte(content))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(cfg.Text.Template) != 3 {
		t.Errorf("expected 3 text lines, got %d", len(cfg.Text.Template))
	}
	if cfg.Text.Template[1] != "" {
		t.Errorf("expected empty line, got %q", cfg.Text.Template[1])
	}
}

func TestLegacyParserWindowTypes(t *testing.T) {
	tests := []struct {
		input    string
		expected WindowType
	}{
		{"own_window_type normal", WindowTypeNormal},
		{"own_window_type desktop", WindowTypeDesktop},
		{"own_window_type dock", WindowTypeDock},
		{"own_window_type panel", WindowTypePanel},
		{"own_window_type override", WindowTypeOverride},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			p := NewLegacyParser()
			cfg, err := p.Parse([]byte(tt.input))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}
			if cfg.Window.Type != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, cfg.Window.Type)
			}
		})
	}
}

func TestLegacyParserWindowHints(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []WindowHint
	}{
		{
			"single hint",
			"own_window_hints undecorated",
			[]WindowHint{WindowHintUndecorated},
		},
		{
			"multiple hints",
			"own_window_hints undecorated,below,sticky",
			[]WindowHint{WindowHintUndecorated, WindowHintBelow, WindowHintSticky},
		},
		{
			"all hints",
			"own_window_hints undecorated,below,above,sticky,skip_taskbar,skip_pager",
			[]WindowHint{
				WindowHintUndecorated,
				WindowHintBelow,
				WindowHintAbove,
				WindowHintSticky,
				WindowHintSkipTaskbar,
				WindowHintSkipPager,
			},
		},
		{
			"hints with spaces",
			"own_window_hints undecorated, below, sticky",
			[]WindowHint{WindowHintUndecorated, WindowHintBelow, WindowHintSticky},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewLegacyParser()
			cfg, err := p.Parse([]byte(tt.input))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}
			if len(cfg.Window.Hints) != len(tt.expected) {
				t.Errorf("expected %d hints, got %d", len(tt.expected), len(cfg.Window.Hints))
				return
			}
			for i, hint := range cfg.Window.Hints {
				if hint != tt.expected[i] {
					t.Errorf("hint %d: expected %v, got %v", i, tt.expected[i], hint)
				}
			}
		})
	}
}

func TestLegacyParserAlignment(t *testing.T) {
	tests := []struct {
		input    string
		expected Alignment
	}{
		{"alignment top_left", AlignmentTopLeft},
		{"alignment tl", AlignmentTopLeft},
		{"alignment top_right", AlignmentTopRight},
		{"alignment tr", AlignmentTopRight},
		{"alignment bottom_left", AlignmentBottomLeft},
		{"alignment bl", AlignmentBottomLeft},
		{"alignment bottom_right", AlignmentBottomRight},
		{"alignment br", AlignmentBottomRight},
		{"alignment center", AlignmentMiddleMiddle},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			p := NewLegacyParser()
			cfg, err := p.Parse([]byte(tt.input))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}
			if cfg.Window.Alignment != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, cfg.Window.Alignment)
			}
		})
	}
}

func TestLegacyParserNumericSettings(t *testing.T) {
	content := `minimum_width 300
minimum_height 200
gap_x 10
gap_y 20
update_interval 2.5
`
	p := NewLegacyParser()
	cfg, err := p.Parse([]byte(content))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if cfg.Window.Width != 300 {
		t.Errorf("expected width=300, got %d", cfg.Window.Width)
	}
	if cfg.Window.Height != 200 {
		t.Errorf("expected height=200, got %d", cfg.Window.Height)
	}
	if cfg.Window.X != 10 {
		t.Errorf("expected X=10, got %d", cfg.Window.X)
	}
	if cfg.Window.Y != 20 {
		t.Errorf("expected Y=20, got %d", cfg.Window.Y)
	}
	if cfg.Display.UpdateInterval != 2500*time.Millisecond {
		t.Errorf("expected update_interval=2.5s, got %v", cfg.Display.UpdateInterval)
	}
}

func TestLegacyParserColors(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		field    string
		expected color.RGBA
	}{
		{"named white", "default_color white", "default", color.RGBA{R: 255, G: 255, B: 255, A: 255}},
		{"named black", "default_color black", "default", color.RGBA{R: 0, G: 0, B: 0, A: 255}},
		{"named red", "default_color red", "default", color.RGBA{R: 255, G: 0, B: 0, A: 255}},
		{"named green", "default_color green", "default", color.RGBA{R: 0, G: 255, B: 0, A: 255}},
		{"named blue", "default_color blue", "default", color.RGBA{R: 0, G: 0, B: 255, A: 255}},
		{"named grey", "default_color grey", "default", color.RGBA{R: 128, G: 128, B: 128, A: 255}},
		{"named gray", "default_color gray", "default", color.RGBA{R: 128, G: 128, B: 128, A: 255}},
		{"hex no hash", "default_color ff0000", "default", color.RGBA{R: 255, G: 0, B: 0, A: 255}},
		{"hex with hash", "default_color #00ff00", "default", color.RGBA{R: 0, G: 255, B: 0, A: 255}},
		{"color0", "color0 white", "color0", color.RGBA{R: 255, G: 255, B: 255, A: 255}},
		{"color1", "color1 grey", "color1", color.RGBA{R: 128, G: 128, B: 128, A: 255}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewLegacyParser()
			cfg, err := p.Parse([]byte(tt.input))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			var actual color.RGBA
			switch tt.field {
			case "default":
				actual = cfg.Colors.Default
			case "color0":
				actual = cfg.Colors.Color0
			case "color1":
				actual = cfg.Colors.Color1
			}

			if actual != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, actual)
			}
		})
	}
}

func TestLegacyParserBooleanValues(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"yes", "background yes", true},
		{"no", "background no", false},
		{"true", "background true", true},
		{"false", "background false", false},
		{"1", "background 1", true},
		{"0", "background 0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewLegacyParser()
			cfg, err := p.Parse([]byte(tt.input))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}
			if cfg.Display.Background != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, cfg.Display.Background)
			}
		})
	}
}

func TestLegacyParserCommentsIgnored(t *testing.T) {
	content := `# This is a comment
background yes
# Another comment
font Test Font
`
	p := NewLegacyParser()
	cfg, err := p.Parse([]byte(content))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if cfg.Display.Background != true {
		t.Error("expected background=true")
	}
	if cfg.Display.Font != "Test Font" {
		t.Errorf("expected font 'Test Font', got %q", cfg.Display.Font)
	}
}

func TestLegacyParserUnknownDirectivesIgnored(t *testing.T) {
	content := `unknown_setting value
future_setting another_value
background yes
`
	p := NewLegacyParser()
	cfg, err := p.Parse([]byte(content))
	if err != nil {
		t.Fatalf("Parse failed unexpectedly: %v", err)
	}

	// Should still parse known settings
	if cfg.Display.Background != true {
		t.Error("expected background=true")
	}
}

func TestLegacyParserErrorHandling(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			"invalid window type",
			"own_window_type invalid_type",
			true,
		},
		{
			"invalid window hint",
			"own_window_hints invalid_hint",
			true,
		},
		{
			"invalid alignment",
			"alignment invalid_alignment",
			true,
		},
		{
			"invalid width",
			"minimum_width not_a_number",
			true,
		},
		{
			"invalid height",
			"minimum_height not_a_number",
			true,
		},
		{
			"invalid gap_x",
			"gap_x not_a_number",
			true,
		},
		{
			"invalid gap_y",
			"gap_y not_a_number",
			true,
		},
		{
			"invalid update_interval",
			"update_interval not_a_number",
			true,
		},
		{
			"invalid color hex",
			"default_color gggggg",
			true,
		},
		{
			"invalid color short hex",
			"default_color fff",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewLegacyParser()
			_, err := p.Parse([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLegacyParserFullConfig(t *testing.T) {
	content := `# Full test configuration
background no
font DejaVu Sans Mono:size=10
update_interval 1.0
double_buffer yes
own_window yes
own_window_type normal
own_window_transparent yes
own_window_hints undecorated,below,sticky,skip_taskbar,skip_pager
alignment top_right
minimum_width 300
minimum_height 200
gap_x 10
gap_y 10
default_color white
color0 white
color1 grey

TEXT
${color grey}System Monitor$color
${hr 2}
${color grey}CPU Usage:$color $cpu%
${color grey}RAM Usage:$color $mem/$memmax ($memperc%)
${color grey}Uptime:$color $uptime
`
	p := NewLegacyParser()
	cfg, err := p.Parse([]byte(content))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Validate all parsed values
	if cfg.Display.Background != false {
		t.Error("expected background=false")
	}
	if cfg.Display.Font != "DejaVu Sans Mono:size=10" {
		t.Error("font mismatch")
	}
	if cfg.Display.UpdateInterval != time.Second {
		t.Error("update_interval mismatch")
	}
	if cfg.Display.DoubleBuffer != true {
		t.Error("double_buffer mismatch")
	}
	if cfg.Window.OwnWindow != true {
		t.Error("own_window mismatch")
	}
	if cfg.Window.Type != WindowTypeNormal {
		t.Error("window type mismatch")
	}
	if cfg.Window.Transparent != true {
		t.Error("transparent mismatch")
	}
	if len(cfg.Window.Hints) != 5 {
		t.Error("hints count mismatch")
	}
	if cfg.Window.Alignment != AlignmentTopRight {
		t.Error("alignment mismatch")
	}
	if cfg.Window.Width != 300 {
		t.Error("width mismatch")
	}
	if cfg.Window.Height != 200 {
		t.Error("height mismatch")
	}
	if cfg.Window.X != 10 {
		t.Error("gap_x mismatch")
	}
	if cfg.Window.Y != 10 {
		t.Error("gap_y mismatch")
	}
	if len(cfg.Text.Template) != 5 {
		t.Errorf("expected 5 text lines, got %d", len(cfg.Text.Template))
	}
}

func TestParseBool(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"yes", true},
		{"YES", true},
		{"Yes", true},
		{"no", false},
		{"NO", false},
		{"No", false},
		{"true", true},
		{"TRUE", true},
		{"false", false},
		{"FALSE", false},
		{"1", true},
		{"0", false},
		{"", false},
		{"invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := parseBool(tt.input); got != tt.expected {
				t.Errorf("parseBool(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParseColor(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected color.RGBA
		wantErr  bool
	}{
		{"white", "white", color.RGBA{R: 255, G: 255, B: 255, A: 255}, false},
		{"black", "black", color.RGBA{R: 0, G: 0, B: 0, A: 255}, false},
		{"hex red", "ff0000", color.RGBA{R: 255, G: 0, B: 0, A: 255}, false},
		{"hex with hash", "#00ff00", color.RGBA{R: 0, G: 255, B: 0, A: 255}, false},
		{"uppercase hex", "FFFFFF", color.RGBA{R: 255, G: 255, B: 255, A: 255}, false},
		{"invalid hex", "gggggg", color.RGBA{}, true},
		{"short hex", "fff", color.RGBA{}, true},
		{"unknown name", "unknowncolor", color.RGBA{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseColor(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseColor(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.expected {
				t.Errorf("parseColor(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParseWindowHints(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []WindowHint
		wantErr  bool
	}{
		{"empty", "", nil, false},
		{"single", "undecorated", []WindowHint{WindowHintUndecorated}, false},
		{"multiple", "undecorated,below", []WindowHint{WindowHintUndecorated, WindowHintBelow}, false},
		{"with spaces", "undecorated, below", []WindowHint{WindowHintUndecorated, WindowHintBelow}, false},
		{"invalid", "invalid", nil, true},
		{"mixed invalid", "undecorated,invalid", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseWindowHints(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseWindowHints(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(got) != len(tt.expected) {
					t.Errorf("parseWindowHints(%q) = %v, want %v", tt.input, got, tt.expected)
				}
			}
		})
	}
}

// TestLegacyParserTemplates tests parsing of template0-template9 directives.
func TestLegacyParserTemplates(t *testing.T) {
content := []byte(`
template0 Hello World
template1 Core \1 usage: ${cpu \1}%
template2 FS \1 is \2% full
template9 Last template with arg \1

TEXT
Test line
`)

parser := NewLegacyParser()
cfg, err := parser.Parse(content)
if err != nil {
t.Fatalf("Parse failed: %v", err)
}

tests := []struct {
index    int
expected string
}{
{0, "Hello World"},
{1, "Core \\1 usage: ${cpu \\1}%"},
{2, "FS \\1 is \\2% full"},
{3, ""},
{4, ""},
{5, ""},
{6, ""},
{7, ""},
{8, ""},
{9, "Last template with arg \\1"},
}

for _, tt := range tests {
t.Run("template"+string(rune('0'+tt.index)), func(t *testing.T) {
if cfg.Text.Templates[tt.index] != tt.expected {
t.Errorf("Templates[%d] = %q, want %q", tt.index, cfg.Text.Templates[tt.index], tt.expected)
}
})
}
}
