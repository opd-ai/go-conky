package config

import (
	"image/color"
	"strings"
	"testing"
	"time"
)

func TestNewLuaConfigParser(t *testing.T) {
	p, err := NewLuaConfigParser()
	if err != nil {
		t.Fatalf("NewLuaConfigParser failed: %v", err)
	}
	defer p.Close()

	if p == nil {
		t.Error("NewLuaConfigParser returned nil")
	}
}

func TestLuaConfigParserParseBasic(t *testing.T) {
	p, err := NewLuaConfigParser()
	if err != nil {
		t.Fatalf("NewLuaConfigParser failed: %v", err)
	}
	defer p.Close()

	content := `
conky.config = {
    background = false,
    font = 'DejaVu Sans Mono:size=10',
    update_interval = 1.0,
    double_buffer = true,
    own_window = true,
    own_window_type = 'normal',
    own_window_transparent = true,
}

conky.text = [[
${color1}System Monitor$color
${hr 2}
${color1}CPU Usage:$color $cpu%
]]
`
	cfg, err := p.Parse([]byte(content))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify parsed values
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
}

func TestLuaConfigParserParseText(t *testing.T) {
	p, err := NewLuaConfigParser()
	if err != nil {
		t.Fatalf("NewLuaConfigParser failed: %v", err)
	}
	defer p.Close()

	content := `
conky.config = {}
conky.text = [[
Line 1
Line 2
Line 3
]]
`
	cfg, err := p.Parse([]byte(content))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Account for leading newline in [[...]]
	expectedLines := 4 // empty, Line 1, Line 2, Line 3
	if len(cfg.Text.Template) != expectedLines {
		t.Errorf("expected %d text lines, got %d: %v", expectedLines, len(cfg.Text.Template), cfg.Text.Template)
	}
}

func TestLuaConfigParserWindowTypes(t *testing.T) {
	tests := []struct {
		input    string
		expected WindowType
	}{
		{"own_window_type = 'normal'", WindowTypeNormal},
		{"own_window_type = 'desktop'", WindowTypeDesktop},
		{"own_window_type = 'dock'", WindowTypeDock},
		{"own_window_type = 'panel'", WindowTypePanel},
		{"own_window_type = 'override'", WindowTypeOverride},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			p, err := NewLuaConfigParser()
			if err != nil {
				t.Fatalf("NewLuaConfigParser failed: %v", err)
			}
			defer p.Close()

			content := "conky.config = { " + tt.input + " }\nconky.text = ''"
			cfg, err := p.Parse([]byte(content))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}
			if cfg.Window.Type != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, cfg.Window.Type)
			}
		})
	}
}

func TestLuaConfigParserWindowHints(t *testing.T) {
	p, err := NewLuaConfigParser()
	if err != nil {
		t.Fatalf("NewLuaConfigParser failed: %v", err)
	}
	defer p.Close()

	content := `
conky.config = {
    own_window_hints = 'undecorated,below,sticky,skip_taskbar,skip_pager',
}
conky.text = ''
`
	cfg, err := p.Parse([]byte(content))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	expected := []WindowHint{
		WindowHintUndecorated,
		WindowHintBelow,
		WindowHintSticky,
		WindowHintSkipTaskbar,
		WindowHintSkipPager,
	}

	if len(cfg.Window.Hints) != len(expected) {
		t.Errorf("expected %d hints, got %d", len(expected), len(cfg.Window.Hints))
		return
	}

	for i, hint := range cfg.Window.Hints {
		if hint != expected[i] {
			t.Errorf("hint %d: expected %v, got %v", i, expected[i], hint)
		}
	}
}

func TestLuaConfigParserAlignment(t *testing.T) {
	tests := []struct {
		input    string
		expected Alignment
	}{
		{"alignment = 'top_left'", AlignmentTopLeft},
		{"alignment = 'top_right'", AlignmentTopRight},
		{"alignment = 'bottom_left'", AlignmentBottomLeft},
		{"alignment = 'bottom_right'", AlignmentBottomRight},
		{"alignment = 'center'", AlignmentMiddleMiddle},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			p, err := NewLuaConfigParser()
			if err != nil {
				t.Fatalf("NewLuaConfigParser failed: %v", err)
			}
			defer p.Close()

			content := "conky.config = { " + tt.input + " }\nconky.text = ''"
			cfg, err := p.Parse([]byte(content))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}
			if cfg.Window.Alignment != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, cfg.Window.Alignment)
			}
		})
	}
}

func TestLuaConfigParserNumericSettings(t *testing.T) {
	p, err := NewLuaConfigParser()
	if err != nil {
		t.Fatalf("NewLuaConfigParser failed: %v", err)
	}
	defer p.Close()

	content := `
conky.config = {
    minimum_width = 300,
    minimum_height = 200,
    gap_x = 10,
    gap_y = 20,
    update_interval = 2.5,
}
conky.text = ''
`
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

func TestLuaConfigParserColors(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected color.RGBA
	}{
		{"default_color white", "default_color = 'white'", color.RGBA{R: 255, G: 255, B: 255, A: 255}},
		{"color0 white", "color0 = 'white'", color.RGBA{R: 255, G: 255, B: 255, A: 255}},
		{"color1 grey", "color1 = 'grey'", color.RGBA{R: 128, G: 128, B: 128, A: 255}},
		{"hex color", "default_color = 'ff0000'", color.RGBA{R: 255, G: 0, B: 0, A: 255}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewLuaConfigParser()
			if err != nil {
				t.Fatalf("NewLuaConfigParser failed: %v", err)
			}
			defer p.Close()

			content := "conky.config = { " + tt.input + " }\nconky.text = ''"
			cfg, err := p.Parse([]byte(content))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			// Check the appropriate color field
			var actual color.RGBA
			switch {
			case strings.Contains(tt.input, "default_color"):
				actual = cfg.Colors.Default
			case strings.Contains(tt.input, "color0"):
				actual = cfg.Colors.Color0
			case strings.Contains(tt.input, "color1"):
				actual = cfg.Colors.Color1
			}

			if actual != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, actual)
			}
		})
	}
}

func TestLuaConfigParserEmptyConfig(t *testing.T) {
	p, err := NewLuaConfigParser()
	if err != nil {
		t.Fatalf("NewLuaConfigParser failed: %v", err)
	}
	defer p.Close()

	content := `
conky.config = {}
conky.text = ''
`
	cfg, err := p.Parse([]byte(content))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should return defaults
	defaults := DefaultConfig()
	if cfg.Window.Width != defaults.Window.Width {
		t.Errorf("expected default width %d, got %d", defaults.Window.Width, cfg.Window.Width)
	}
}

func TestLuaConfigParserNoConkyTable(t *testing.T) {
	p, err := NewLuaConfigParser()
	if err != nil {
		t.Fatalf("NewLuaConfigParser failed: %v", err)
	}
	defer p.Close()

	// Empty Lua code - conky table should be initialized by parser
	content := "-- Empty config"
	cfg, err := p.Parse([]byte(content))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should return defaults
	if cfg == nil {
		t.Fatal("Expected non-nil config")
	}
}

func TestLuaConfigParserErrorHandling(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			"invalid Lua syntax",
			"conky.config = { this is not valid lua",
			true,
		},
		{
			"invalid window type",
			"conky.config = { own_window_type = 'invalid' }\nconky.text = ''",
			true,
		},
		{
			"invalid window hint",
			"conky.config = { own_window_hints = 'invalid_hint' }\nconky.text = ''",
			true,
		},
		{
			"invalid alignment",
			"conky.config = { alignment = 'invalid_alignment' }\nconky.text = ''",
			true,
		},
		{
			"invalid color",
			"conky.config = { default_color = 'gggggg' }\nconky.text = ''",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewLuaConfigParser()
			if err != nil {
				t.Fatalf("NewLuaConfigParser failed: %v", err)
			}
			defer p.Close()

			_, err = p.Parse([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLuaConfigParserFullConfig(t *testing.T) {
	p, err := NewLuaConfigParser()
	if err != nil {
		t.Fatalf("NewLuaConfigParser failed: %v", err)
	}
	defer p.Close()

	content := `
-- Sample Lua configuration for testing
conky.config = {
    background = false,
    font = 'DejaVu Sans Mono:size=10',
    update_interval = 1.0,
    double_buffer = true,
    own_window = true,
    own_window_type = 'normal',
    own_window_transparent = true,
    own_window_hints = 'undecorated,below,sticky,skip_taskbar,skip_pager',
    alignment = 'top_right',
    minimum_width = 300,
    minimum_height = 200,
    gap_x = 10,
    gap_y = 10,
    default_color = 'white',
    color0 = 'white',
    color1 = 'grey',
}

conky.text = [[
${color1}System Monitor$color
${hr 2}
${color1}CPU Usage:$color $cpu%
${color1}RAM Usage:$color $mem/$memmax ($memperc%)
${color1}Uptime:$color $uptime
]]
`
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
	if len(cfg.Text.Template) == 0 {
		t.Error("expected non-empty text template")
	}
}

func TestLuaConfigParserWithExistingRuntime(t *testing.T) {
	runtime, err := NewLuaConfigParser()
	if err != nil {
		t.Fatalf("NewLuaConfigParser failed: %v", err)
	}
	defer runtime.Close()

	// Test that the parser works
	content := `
conky.config = { background = true }
conky.text = ''
`
	cfg, err := runtime.Parse([]byte(content))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if cfg.Display.Background != true {
		t.Error("expected background=true")
	}
}

func TestLuaConfigParserClose(t *testing.T) {
	p, err := NewLuaConfigParser()
	if err != nil {
		t.Fatalf("NewLuaConfigParser failed: %v", err)
	}

	// Close should not error
	if err := p.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Closing again should be safe
	if err := p.Close(); err != nil {
		t.Errorf("Second Close failed: %v", err)
	}
}

// TestLuaParserTemplates tests parsing of template0-template9 in Lua config.
func TestLuaParserTemplates(t *testing.T) {
content := []byte(`
conky.config = {
    template0 = "Hello World",
    template1 = "Core \\1 usage: ${cpu \\1}%",
    template2 = "FS \\1 is \\2% full",
    template9 = "Last template with arg \\1",
}
conky.text = [[Test]]
`)

parser, err := NewLuaConfigParser()
if err != nil {
t.Fatalf("NewLuaConfigParser failed: %v", err)
}
defer parser.Close()

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

// TestLuaParserDisplayDirectives tests parsing of display/rendering directives in Lua format.
func TestLuaParserDisplayDirectives(t *testing.T) {
	content := []byte(`
conky.config = {
    draw_borders = true,
    draw_outline = true,
    draw_shades = false,
    stippled_borders = true,
    border_width = 3,
    border_inner_margin = 10,
    border_outer_margin = 8,
}
conky.text = [[Test line]]
`)

	parser, err := NewLuaConfigParser()
	if err != nil {
		t.Fatalf("NewLuaConfigParser failed: %v", err)
	}
	defer parser.Close()

	cfg, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"DrawBorders", cfg.Display.DrawBorders, true},
		{"DrawOutline", cfg.Display.DrawOutline, true},
		{"DrawShades", cfg.Display.DrawShades, false},
		{"StippledBorders", cfg.Display.StippledBorders, true},
		{"BorderWidth", cfg.Display.BorderWidth, 3},
		{"BorderInnerMargin", cfg.Display.BorderInnerMargin, 10},
		{"BorderOuterMargin", cfg.Display.BorderOuterMargin, 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

// TestLuaParserDisplayDirectivesDefaults tests default values for display directives.
func TestLuaParserDisplayDirectivesDefaults(t *testing.T) {
	content := []byte(`
conky.config = {
    background = false,
}
conky.text = [[Test line]]
`)

	parser, err := NewLuaConfigParser()
	if err != nil {
		t.Fatalf("NewLuaConfigParser failed: %v", err)
	}
	defer parser.Close()

	cfg, err := parser.Parse(content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if cfg.Display.DrawBorders != false {
		t.Errorf("DrawBorders default = %v, want false", cfg.Display.DrawBorders)
	}
	if cfg.Display.DrawOutline != false {
		t.Errorf("DrawOutline default = %v, want false", cfg.Display.DrawOutline)
	}
	if cfg.Display.DrawShades != true {
		t.Errorf("DrawShades default = %v, want true", cfg.Display.DrawShades)
	}
	if cfg.Display.StippledBorders != false {
		t.Errorf("StippledBorders default = %v, want false", cfg.Display.StippledBorders)
	}
	if cfg.Display.BorderWidth != 1 {
		t.Errorf("BorderWidth default = %v, want 1", cfg.Display.BorderWidth)
	}
	if cfg.Display.BorderInnerMargin != 5 {
		t.Errorf("BorderInnerMargin default = %v, want 5", cfg.Display.BorderInnerMargin)
	}
	if cfg.Display.BorderOuterMargin != 5 {
		t.Errorf("BorderOuterMargin default = %v, want 5", cfg.Display.BorderOuterMargin)
	}
}
