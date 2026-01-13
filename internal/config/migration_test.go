package config

import (
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewMigrator(t *testing.T) {
	m := NewMigrator()
	if m == nil {
		t.Fatal("NewMigrator returned nil")
	}
	// Default options
	if !m.includeComments {
		t.Error("expected includeComments to be true by default")
	}
	if m.preserveDefaults {
		t.Error("expected preserveDefaults to be false by default")
	}
}

func TestNewMigratorWithOptions(t *testing.T) {
	m := NewMigrator(
		WithComments(false),
		WithDefaults(true),
	)
	if m.includeComments {
		t.Error("expected includeComments to be false")
	}
	if !m.preserveDefaults {
		t.Error("expected preserveDefaults to be true")
	}
}

func TestMigratorMigrateToLuaNil(t *testing.T) {
	m := NewMigrator()
	_, err := m.MigrateToLua(nil)
	if err == nil {
		t.Error("expected error for nil config")
	}
}

func TestMigratorMigrateToLuaBasic(t *testing.T) {
	m := NewMigrator()
	cfg := DefaultConfig()
	cfg.Display.Background = true
	cfg.Display.UpdateInterval = 2 * time.Second
	cfg.Window.OwnWindow = true
	cfg.Window.Width = 300
	cfg.Window.Height = 200
	cfg.Text.Template = []string{"Line 1", "Line 2"}

	result, err := m.MigrateToLua(&cfg)
	if err != nil {
		t.Fatalf("MigrateToLua failed: %v", err)
	}

	output := string(result)

	// Check header comment
	if !strings.Contains(output, "-- Conky Lua configuration") {
		t.Error("expected header comment")
	}

	// Check conky.config table
	if !strings.Contains(output, "conky.config = {") {
		t.Error("expected conky.config = {")
	}

	// Check settings
	if !strings.Contains(output, "background = true") {
		t.Error("expected background = true")
	}
	if !strings.Contains(output, "update_interval = 2.0") {
		t.Error("expected update_interval = 2.0")
	}
	if !strings.Contains(output, "minimum_width = 300") {
		t.Error("expected minimum_width = 300")
	}
	if !strings.Contains(output, "minimum_height = 200") {
		t.Error("expected minimum_height = 200")
	}

	// Check text section
	if !strings.Contains(output, "conky.text = [[") {
		t.Error("expected conky.text = [[")
	}
	if !strings.Contains(output, "Line 1") {
		t.Error("expected Line 1 in text")
	}
	if !strings.Contains(output, "Line 2") {
		t.Error("expected Line 2 in text")
	}
}

func TestMigratorMigrateToLuaNoComments(t *testing.T) {
	m := NewMigrator(WithComments(false))
	cfg := DefaultConfig()
	cfg.Display.Background = true

	result, err := m.MigrateToLua(&cfg)
	if err != nil {
		t.Fatalf("MigrateToLua failed: %v", err)
	}

	output := string(result)
	if strings.Contains(output, "-- Conky Lua configuration") {
		t.Error("expected no header comment")
	}
	if strings.Contains(output, "-- Display settings") {
		t.Error("expected no section comments")
	}
}

func TestMigratorMigrateToLuaWithDefaults(t *testing.T) {
	m := NewMigrator(WithDefaults(true))
	cfg := DefaultConfig()

	result, err := m.MigrateToLua(&cfg)
	if err != nil {
		t.Fatalf("MigrateToLua failed: %v", err)
	}

	output := string(result)
	// With preserveDefaults, double_buffer should be included even though it's default
	if !strings.Contains(output, "double_buffer = true") {
		t.Error("expected double_buffer = true when preserveDefaults is true")
	}
}

func TestMigratorMigrateToLuaWindowTypes(t *testing.T) {
	// Skip WindowTypeNormal as it's the default and won't be output
	tests := []struct {
		windowType WindowType
		expected   string
	}{
		{WindowTypeDesktop, "own_window_type = 'desktop'"},
		{WindowTypeDock, "own_window_type = 'dock'"},
		{WindowTypePanel, "own_window_type = 'panel'"},
		{WindowTypeOverride, "own_window_type = 'override'"},
	}

	for _, tt := range tests {
		t.Run(tt.windowType.String(), func(t *testing.T) {
			m := NewMigrator()
			cfg := DefaultConfig()
			cfg.Window.Type = tt.windowType

			result, err := m.MigrateToLua(&cfg)
			if err != nil {
				t.Fatalf("MigrateToLua failed: %v", err)
			}

			if !strings.Contains(string(result), tt.expected) {
				t.Errorf("expected %s in output", tt.expected)
			}
		})
	}
}

func TestMigratorMigrateToLuaWindowTypeNormalWithPreserveDefaults(t *testing.T) {
	// Test that WindowTypeNormal is output when preserveDefaults is true
	m := NewMigrator(WithDefaults(true))
	cfg := DefaultConfig()
	cfg.Window.Type = WindowTypeNormal

	result, err := m.MigrateToLua(&cfg)
	if err != nil {
		t.Fatalf("MigrateToLua failed: %v", err)
	}

	if !strings.Contains(string(result), "own_window_type = 'normal'") {
		t.Error("expected own_window_type = 'normal' in output when preserveDefaults is true")
	}
}

func TestMigratorMigrateToLuaWindowHints(t *testing.T) {
	m := NewMigrator()
	cfg := DefaultConfig()
	cfg.Window.Hints = []WindowHint{
		WindowHintUndecorated,
		WindowHintBelow,
		WindowHintSticky,
	}

	result, err := m.MigrateToLua(&cfg)
	if err != nil {
		t.Fatalf("MigrateToLua failed: %v", err)
	}

	output := string(result)
	if !strings.Contains(output, "own_window_hints = 'undecorated,below,sticky'") {
		t.Errorf("expected window hints, got: %s", output)
	}
}

func TestMigratorMigrateToLuaAlignment(t *testing.T) {
	tests := []struct {
		alignment Alignment
		expected  string
	}{
		{AlignmentTopRight, "alignment = 'top_right'"},
		{AlignmentBottomLeft, "alignment = 'bottom_left'"},
		{AlignmentMiddleMiddle, "alignment = 'middle_middle'"},
	}

	for _, tt := range tests {
		t.Run(tt.alignment.String(), func(t *testing.T) {
			m := NewMigrator()
			cfg := DefaultConfig()
			cfg.Window.Alignment = tt.alignment

			result, err := m.MigrateToLua(&cfg)
			if err != nil {
				t.Fatalf("MigrateToLua failed: %v", err)
			}

			if !strings.Contains(string(result), tt.expected) {
				t.Errorf("expected %s in output", tt.expected)
			}
		})
	}
}

func TestMigratorMigrateToLuaColors(t *testing.T) {
	m := NewMigrator()
	cfg := DefaultConfig()
	cfg.Colors.Default = color.RGBA{R: 255, G: 0, B: 0, A: 255}
	cfg.Colors.Color1 = color.RGBA{R: 0, G: 128, B: 0, A: 255} // Not a named color

	result, err := m.MigrateToLua(&cfg)
	if err != nil {
		t.Fatalf("MigrateToLua failed: %v", err)
	}

	output := string(result)
	// Red is a named color
	if !strings.Contains(output, "default_color = 'red'") {
		t.Errorf("expected default_color = 'red', got: %s", output)
	}
	// 008000 is not a named color, should use hex
	if !strings.Contains(output, "color1 = '008000'") {
		t.Errorf("expected color1 = '008000', got: %s", output)
	}
}

func TestMigratorMigrateToLuaNamedColors(t *testing.T) {
	// Test non-default colors only, since default colors won't be output
	tests := []struct {
		name     string
		color    color.RGBA
		expected string
	}{
		{"black", color.RGBA{R: 0, G: 0, B: 0, A: 255}, "default_color = 'black'"},
		{"grey", color.RGBA{R: 128, G: 128, B: 128, A: 255}, "default_color = 'grey'"}, // or 'gray'
		{"orange", color.RGBA{R: 255, G: 165, B: 0, A: 255}, "default_color = 'orange'"},
		{"red", color.RGBA{R: 255, G: 0, B: 0, A: 255}, "default_color = 'red'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMigrator()
			cfg := DefaultConfig()
			cfg.Colors.Default = tt.color

			result, err := m.MigrateToLua(&cfg)
			if err != nil {
				t.Fatalf("MigrateToLua failed: %v", err)
			}

			output := string(result)
			if !strings.Contains(output, tt.expected) {
				// Check for alternative spelling (gray vs grey)
				if tt.name == "grey" && strings.Contains(output, "default_color = 'gray'") {
					return // acceptable
				}
				t.Errorf("expected %s, got: %s", tt.expected, output)
			}
		})
	}
}

func TestMigratorMigrateToLuaDefaultColorWithPreserve(t *testing.T) {
	// Test that white (default) is output when preserveDefaults is true
	m := NewMigrator(WithDefaults(true))
	cfg := DefaultConfig()
	// Default color is already white

	result, err := m.MigrateToLua(&cfg)
	if err != nil {
		t.Fatalf("MigrateToLua failed: %v", err)
	}

	if !strings.Contains(string(result), "default_color = 'white'") {
		t.Error("expected default_color = 'white' when preserveDefaults is true")
	}
}

func TestMigratorMigrateToLuaFont(t *testing.T) {
	m := NewMigrator()
	cfg := DefaultConfig()
	cfg.Display.Font = "DejaVu Sans Mono:size=12"

	result, err := m.MigrateToLua(&cfg)
	if err != nil {
		t.Fatalf("MigrateToLua failed: %v", err)
	}

	if !strings.Contains(string(result), "font = 'DejaVu Sans Mono:size=12'") {
		t.Error("expected font setting in output")
	}
}

func TestMigratorMigrateToLuaFontWithQuotes(t *testing.T) {
	m := NewMigrator()
	cfg := DefaultConfig()
	cfg.Display.Font = "Font's Name"

	result, err := m.MigrateToLua(&cfg)
	if err != nil {
		t.Fatalf("MigrateToLua failed: %v", err)
	}

	if !strings.Contains(string(result), "font = 'Font\\'s Name'") {
		t.Errorf("expected escaped quote in font name, got: %s", string(result))
	}
}

func TestMigratorMigrateToLuaGapSettings(t *testing.T) {
	m := NewMigrator()
	cfg := DefaultConfig()
	cfg.Window.X = 10
	cfg.Window.Y = 20

	result, err := m.MigrateToLua(&cfg)
	if err != nil {
		t.Fatalf("MigrateToLua failed: %v", err)
	}

	output := string(result)
	if !strings.Contains(output, "gap_x = 10") {
		t.Error("expected gap_x = 10")
	}
	if !strings.Contains(output, "gap_y = 20") {
		t.Error("expected gap_y = 20")
	}
}

func TestMigratorMigrateToLuaTextPreserved(t *testing.T) {
	m := NewMigrator()
	cfg := DefaultConfig()
	cfg.Text.Template = []string{
		"${color grey}System Monitor$color",
		"${hr 2}",
		"${color grey}CPU Usage:$color $cpu%",
		"",
		"${color grey}RAM:$color $mem/$memmax",
	}

	result, err := m.MigrateToLua(&cfg)
	if err != nil {
		t.Fatalf("MigrateToLua failed: %v", err)
	}

	output := string(result)
	for _, line := range cfg.Text.Template {
		if !strings.Contains(output, line) {
			t.Errorf("expected line %q in output", line)
		}
	}
}

func TestMigrateLegacyContent(t *testing.T) {
	content := []byte(`# Sample legacy config
background yes
font DejaVu Sans Mono:size=10
update_interval 1.5
double_buffer yes
own_window yes
own_window_type desktop
own_window_hints undecorated,below,sticky
alignment top_right
minimum_width 300
minimum_height 200
gap_x 10
gap_y 20
default_color red

TEXT
${color grey}System Monitor$color
${hr 2}
${color grey}CPU:$color $cpu%
`)

	result, err := MigrateLegacyContent(content)
	if err != nil {
		t.Fatalf("MigrateLegacyContent failed: %v", err)
	}

	output := string(result)

	// Verify key settings are present (only non-default values are output)
	expectedSettings := []string{
		"conky.config = {",
		"background = true",
		"font = 'DejaVu Sans Mono:size=10'",
		"update_interval = 1.5",
		"own_window_type = 'desktop'",
		"own_window_hints = 'undecorated,below,sticky'",
		"alignment = 'top_right'",
		"minimum_width = 300",
		"minimum_height = 200",
		"gap_x = 10",
		"gap_y = 20",
		"default_color = 'red'",
		"conky.text = [[",
		"${color grey}System Monitor$color",
	}

	for _, expected := range expectedSettings {
		if !strings.Contains(output, expected) {
			t.Errorf("expected %q in output", expected)
		}
	}
}

func TestMigrateLegacyContentInvalidContent(t *testing.T) {
	content := []byte(`own_window_type invalid_type`)

	_, err := MigrateLegacyContent(content)
	if err == nil {
		t.Error("expected error for invalid content")
	}
}

func TestMigrateLegacyFile(t *testing.T) {
	// Create a temporary file with legacy content
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.conkyrc")
	content := `background yes
font Test Font
update_interval 2.0

TEXT
Test line
`
	if err := os.WriteFile(tmpFile, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	result, err := MigrateLegacyFile(tmpFile)
	if err != nil {
		t.Fatalf("MigrateLegacyFile failed: %v", err)
	}

	output := string(result)
	if !strings.Contains(output, "background = true") {
		t.Error("expected background = true")
	}
	if !strings.Contains(output, "font = 'Test Font'") {
		t.Error("expected font setting")
	}
	if !strings.Contains(output, "Test line") {
		t.Error("expected text line")
	}
}

func TestMigrateLegacyFileNotFound(t *testing.T) {
	_, err := MigrateLegacyFile("/nonexistent/path/to/config")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestColorToName(t *testing.T) {
	tests := []struct {
		name     string
		color    color.RGBA
		expected string
	}{
		{"white", color.RGBA{R: 255, G: 255, B: 255, A: 255}, "white"},
		{"black", color.RGBA{R: 0, G: 0, B: 0, A: 255}, "black"},
		{"red", color.RGBA{R: 255, G: 0, B: 0, A: 255}, "red"},
		{"unknown", color.RGBA{R: 123, G: 45, B: 67, A: 255}, ""},
		{"transparent", color.RGBA{R: 255, G: 255, B: 255, A: 0}, ""}, // Not full opacity
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := colorToName(tt.color)
			if got != tt.expected {
				// gray and grey might both match
				if (tt.expected == "grey" || tt.expected == "gray") &&
					(got == "grey" || got == "gray") {
					return
				}
				t.Errorf("colorToName(%v) = %q, want %q", tt.color, got, tt.expected)
			}
		})
	}
}

func TestMigratorMigrateToLuaRoundTrip(t *testing.T) {
	// Parse a legacy config, migrate it, and verify the Lua config can be parsed
	legacyContent := []byte(`background yes
font DejaVu Sans Mono:size=10
update_interval 1.0
double_buffer yes
own_window yes
own_window_type normal
own_window_transparent yes
alignment top_right
minimum_width 200
minimum_height 100

TEXT
${color grey}Test$color
`)

	// Migrate to Lua
	luaContent, err := MigrateLegacyContent(legacyContent)
	if err != nil {
		t.Fatalf("MigrateLegacyContent failed: %v", err)
	}

	// Parse the Lua content
	parser, err := NewParser()
	if err != nil {
		t.Fatalf("NewParser failed: %v", err)
	}
	defer parser.Close()

	cfg, err := parser.Parse(luaContent)
	if err != nil {
		t.Fatalf("Failed to parse migrated Lua config: %v", err)
	}

	// Verify the parsed config matches expectations
	if cfg.Display.Background != true {
		t.Error("expected background=true")
	}
	if cfg.Display.Font != "DejaVu Sans Mono:size=10" {
		t.Errorf("font mismatch: got %q", cfg.Display.Font)
	}
	if cfg.Display.UpdateInterval != time.Second {
		t.Errorf("update_interval mismatch: got %v", cfg.Display.UpdateInterval)
	}
	if cfg.Display.DoubleBuffer != true {
		t.Error("expected double_buffer=true")
	}
	if cfg.Window.OwnWindow != true {
		t.Error("expected own_window=true")
	}
	if cfg.Window.Type != WindowTypeNormal {
		t.Errorf("window type mismatch: got %v", cfg.Window.Type)
	}
	if cfg.Window.Transparent != true {
		t.Error("expected transparent=true")
	}
	if cfg.Window.Alignment != AlignmentTopRight {
		t.Errorf("alignment mismatch: got %v", cfg.Window.Alignment)
	}
	if cfg.Window.Width != 200 {
		t.Errorf("width mismatch: got %d", cfg.Window.Width)
	}
	if cfg.Window.Height != 100 {
		t.Errorf("height mismatch: got %d", cfg.Window.Height)
	}
}

func TestMigrateActualTestConfig(t *testing.T) {
	// Test migration of the actual test config file
	result, err := MigrateLegacyFile("../../test/configs/basic.conkyrc")
	if err != nil {
		t.Fatalf("MigrateLegacyFile failed: %v", err)
	}

	output := string(result)

	// Verify the output is valid Lua that can be parsed
	parser, err := NewParser()
	if err != nil {
		t.Fatalf("NewParser failed: %v", err)
	}
	defer parser.Close()

	cfg, err := parser.Parse(result)
	if err != nil {
		t.Fatalf("Failed to parse migrated config: %v\nOutput was:\n%s", err, output)
	}

	// Verify basic properties
	if cfg.Window.OwnWindow != true {
		t.Error("expected own_window=true")
	}
	if cfg.Window.Type != WindowTypeNormal {
		t.Error("expected window type normal")
	}
	if len(cfg.Text.Template) == 0 {
		t.Error("expected non-empty text template")
	}
}
