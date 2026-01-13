// Package config provides comprehensive configuration test suite.
// This file implements integration tests for parsing, validation, and migration
// of Conky configuration files in both legacy and Lua formats.
package config

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestConfigSuiteRealWorldConfigs tests parsing of real-world configuration files
// located in the test/configs directory.
func TestConfigSuiteRealWorldConfigs(t *testing.T) {
	tests := []struct {
		name              string
		path              string
		isLua             bool
		expectBackground  bool
		expectOwnWindow   bool
		expectWindowType  WindowType
		expectTransparent bool
		minTextLines      int
	}{
		{
			name:              "basic legacy config",
			path:              "../../test/configs/basic.conkyrc",
			isLua:             false,
			expectBackground:  false,
			expectOwnWindow:   true,
			expectWindowType:  WindowTypeNormal,
			expectTransparent: true,
			minTextLines:      3,
		},
		{
			name:              "basic Lua config",
			path:              "../../test/configs/basic_lua.conkyrc",
			isLua:             true,
			expectBackground:  false,
			expectOwnWindow:   true,
			expectWindowType:  WindowTypeNormal,
			expectTransparent: true,
			minTextLines:      3,
		},
		{
			name:              "advanced legacy config",
			path:              "../../test/configs/advanced.conkyrc",
			isLua:             false,
			expectBackground:  true,
			expectOwnWindow:   true,
			expectWindowType:  WindowTypeDesktop,
			expectTransparent: false,
			minTextLines:      10,
		},
		{
			name:              "advanced Lua config",
			path:              "../../test/configs/advanced_lua.conkyrc",
			isLua:             true,
			expectBackground:  true,
			expectOwnWindow:   true,
			expectWindowType:  WindowTypeDesktop,
			expectTransparent: false,
			minTextLines:      10,
		},
		{
			name:              "minimal legacy config",
			path:              "../../test/configs/minimal.conkyrc",
			isLua:             false,
			expectBackground:  false, // default
			expectOwnWindow:   true,  // default
			expectWindowType:  WindowTypeNormal,
			expectTransparent: false,
			minTextLines:      1,
		},
		{
			name:              "minimal Lua config",
			path:              "../../test/configs/minimal_lua.conkyrc",
			isLua:             true,
			expectBackground:  false, // default
			expectOwnWindow:   true,  // default
			expectWindowType:  WindowTypeNormal,
			expectTransparent: false,
			minTextLines:      1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser()
			if err != nil {
				t.Fatalf("NewParser failed: %v", err)
			}
			defer parser.Close()

			cfg, err := parser.ParseFile(tt.path)
			if err != nil {
				t.Fatalf("ParseFile failed: %v", err)
			}

			// Validate format detection
			content, err := os.ReadFile(tt.path)
			if err != nil {
				t.Fatalf("Failed to read file: %v", err)
			}
			if isLuaConfig(content) != tt.isLua {
				t.Errorf("Format detection mismatch: expected isLua=%v", tt.isLua)
			}

			// Validate parsed values
			if cfg.Display.Background != tt.expectBackground {
				t.Errorf("Background: expected %v, got %v", tt.expectBackground, cfg.Display.Background)
			}
			if cfg.Window.OwnWindow != tt.expectOwnWindow {
				t.Errorf("OwnWindow: expected %v, got %v", tt.expectOwnWindow, cfg.Window.OwnWindow)
			}
			if cfg.Window.Type != tt.expectWindowType {
				t.Errorf("WindowType: expected %v, got %v", tt.expectWindowType, cfg.Window.Type)
			}
			if cfg.Window.Transparent != tt.expectTransparent {
				t.Errorf("Transparent: expected %v, got %v", tt.expectTransparent, cfg.Window.Transparent)
			}
			if len(cfg.Text.Template) < tt.minTextLines {
				t.Errorf("Text.Template: expected at least %d lines, got %d", tt.minTextLines, len(cfg.Text.Template))
			}

			// Validate config is valid
			if err := cfg.Validate(); err != nil {
				t.Errorf("Validate failed: %v", err)
			}
		})
	}
}

// TestConfigSuiteFormatCompatibility verifies that equivalent legacy and Lua configs
// produce the same parsed configuration values.
func TestConfigSuiteFormatCompatibility(t *testing.T) {
	compatTests := []struct {
		name       string
		legacyPath string
		luaPath    string
	}{
		{
			name:       "basic configs",
			legacyPath: "../../test/configs/basic.conkyrc",
			luaPath:    "../../test/configs/basic_lua.conkyrc",
		},
		{
			name:       "advanced configs",
			legacyPath: "../../test/configs/advanced.conkyrc",
			luaPath:    "../../test/configs/advanced_lua.conkyrc",
		},
		{
			name:       "minimal configs",
			legacyPath: "../../test/configs/minimal.conkyrc",
			luaPath:    "../../test/configs/minimal_lua.conkyrc",
		},
	}

	for _, tt := range compatTests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := NewParser()
			if err != nil {
				t.Fatalf("NewParser failed: %v", err)
			}
			defer parser.Close()

			legacyCfg, err := parser.ParseFile(tt.legacyPath)
			if err != nil {
				t.Fatalf("Parse legacy failed: %v", err)
			}

			luaCfg, err := parser.ParseFile(tt.luaPath)
			if err != nil {
				t.Fatalf("Parse Lua failed: %v", err)
			}

			// Compare key configuration values
			if legacyCfg.Display.Background != luaCfg.Display.Background {
				t.Errorf("Background mismatch: legacy=%v, lua=%v",
					legacyCfg.Display.Background, luaCfg.Display.Background)
			}
			if legacyCfg.Window.Type != luaCfg.Window.Type {
				t.Errorf("WindowType mismatch: legacy=%v, lua=%v",
					legacyCfg.Window.Type, luaCfg.Window.Type)
			}
			if legacyCfg.Window.OwnWindow != luaCfg.Window.OwnWindow {
				t.Errorf("OwnWindow mismatch: legacy=%v, lua=%v",
					legacyCfg.Window.OwnWindow, luaCfg.Window.OwnWindow)
			}
			if legacyCfg.Window.Alignment != luaCfg.Window.Alignment {
				t.Errorf("Alignment mismatch: legacy=%v, lua=%v",
					legacyCfg.Window.Alignment, luaCfg.Window.Alignment)
			}

			// Compare update interval (with tolerance for floating point)
			if legacyCfg.Display.UpdateInterval != luaCfg.Display.UpdateInterval {
				t.Errorf("UpdateInterval mismatch: legacy=%v, lua=%v",
					legacyCfg.Display.UpdateInterval, luaCfg.Display.UpdateInterval)
			}

			// Compare dimensions (only if explicitly set in configs)
			if legacyCfg.Window.Width != luaCfg.Window.Width {
				t.Errorf("Width mismatch: legacy=%d, lua=%d",
					legacyCfg.Window.Width, luaCfg.Window.Width)
			}
			if legacyCfg.Window.Height != luaCfg.Window.Height {
				t.Errorf("Height mismatch: legacy=%d, lua=%d",
					legacyCfg.Window.Height, luaCfg.Window.Height)
			}
		})
	}
}

// TestConfigSuiteRoundTrip tests that configs can be parsed, migrated, and re-parsed.
func TestConfigSuiteRoundTrip(t *testing.T) {
	legacyConfigs := []string{
		"../../test/configs/basic.conkyrc",
		"../../test/configs/advanced.conkyrc",
		"../../test/configs/minimal.conkyrc",
	}

	for _, path := range legacyConfigs {
		t.Run(filepath.Base(path), func(t *testing.T) {
			// Parse original legacy config
			parser, err := NewParser()
			if err != nil {
				t.Fatalf("NewParser failed: %v", err)
			}
			defer parser.Close()

			originalCfg, err := parser.ParseFile(path)
			if err != nil {
				t.Fatalf("ParseFile failed: %v", err)
			}

			// Migrate to Lua
			luaContent, err := MigrateLegacyFile(path)
			if err != nil {
				t.Fatalf("MigrateLegacyFile failed: %v", err)
			}

			// Re-parse the migrated Lua content
			parser2, err := NewParser()
			if err != nil {
				t.Fatalf("NewParser (second) failed: %v", err)
			}
			defer parser2.Close()

			migratedCfg, err := parser2.Parse(luaContent)
			if err != nil {
				t.Fatalf("Parse migrated content failed: %v\nContent:\n%s", err, string(luaContent))
			}

			// Compare key values - they should match after round-trip
			if originalCfg.Display.Background != migratedCfg.Display.Background {
				t.Errorf("Background mismatch after round-trip: original=%v, migrated=%v",
					originalCfg.Display.Background, migratedCfg.Display.Background)
			}
			if originalCfg.Window.OwnWindow != migratedCfg.Window.OwnWindow {
				t.Errorf("OwnWindow mismatch after round-trip: original=%v, migrated=%v",
					originalCfg.Window.OwnWindow, migratedCfg.Window.OwnWindow)
			}
			if originalCfg.Window.Type != migratedCfg.Window.Type {
				t.Errorf("WindowType mismatch after round-trip: original=%v, migrated=%v",
					originalCfg.Window.Type, migratedCfg.Window.Type)
			}
			if originalCfg.Window.Transparent != migratedCfg.Window.Transparent {
				t.Errorf("Transparent mismatch after round-trip: original=%v, migrated=%v",
					originalCfg.Window.Transparent, migratedCfg.Window.Transparent)
			}

			// Verify migrated config is valid
			if err := migratedCfg.Validate(); err != nil {
				t.Errorf("Migrated config validation failed: %v", err)
			}
		})
	}
}

// TestConfigSuiteEdgeCases tests edge cases and boundary conditions.
func TestConfigSuiteEdgeCases(t *testing.T) {
	t.Run("empty text section", func(t *testing.T) {
		p := NewLegacyParser()
		content := `background yes
own_window yes

TEXT
`
		cfg, err := p.Parse([]byte(content))
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}
		if cfg.Display.Background != true {
			t.Error("expected background=true")
		}
		// Empty text section is valid
		if err := cfg.Validate(); err != nil {
			t.Errorf("Validate failed for empty text section: %v", err)
		}
	})

	t.Run("no TEXT section", func(t *testing.T) {
		p := NewLegacyParser()
		content := `background yes
own_window yes
`
		cfg, err := p.Parse([]byte(content))
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}
		if cfg.Display.Background != true {
			t.Error("expected background=true")
		}
		if len(cfg.Text.Template) != 0 {
			t.Errorf("expected empty template, got %d lines", len(cfg.Text.Template))
		}
	})

	t.Run("very long text template", func(t *testing.T) {
		p := NewLegacyParser()
		var builder strings.Builder
		builder.WriteString("background yes\n\nTEXT\n")
		for i := range 100 {
			builder.WriteString("${cpu} Line ")
			builder.WriteString(string(rune('0' + i%10)))
			builder.WriteString("\n")
		}
		content := builder.String()

		cfg, err := p.Parse([]byte(content))
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}
		if len(cfg.Text.Template) != 100 {
			t.Errorf("expected 100 text lines, got %d", len(cfg.Text.Template))
		}
	})

	t.Run("update interval precision", func(t *testing.T) {
		tests := []struct {
			input    string
			expected time.Duration
		}{
			{"update_interval 1.0", time.Second},
			{"update_interval 0.5", 500 * time.Millisecond},
			{"update_interval 0.1", 100 * time.Millisecond},
			{"update_interval 2.5", 2500 * time.Millisecond},
			{"update_interval 60", 60 * time.Second},
		}

		for _, tt := range tests {
			t.Run(tt.input, func(t *testing.T) {
				p := NewLegacyParser()
				cfg, err := p.Parse([]byte(tt.input))
				if err != nil {
					t.Fatalf("Parse failed: %v", err)
				}
				if cfg.Display.UpdateInterval != tt.expected {
					t.Errorf("expected %v, got %v", tt.expected, cfg.Display.UpdateInterval)
				}
			})
		}
	})

	t.Run("all window hints combined", func(t *testing.T) {
		p := NewLegacyParser()
		content := `own_window_hints undecorated,below,above,sticky,skip_taskbar,skip_pager`
		cfg, err := p.Parse([]byte(content))
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}
		if len(cfg.Window.Hints) != 6 {
			t.Errorf("expected 6 hints, got %d", len(cfg.Window.Hints))
		}
	})

	t.Run("all alignments", func(t *testing.T) {
		alignments := []struct {
			input    string
			expected Alignment
		}{
			{"alignment top_left", AlignmentTopLeft},
			{"alignment top_middle", AlignmentTopMiddle},
			{"alignment top_right", AlignmentTopRight},
			{"alignment middle_left", AlignmentMiddleLeft},
			{"alignment middle_middle", AlignmentMiddleMiddle},
			{"alignment middle_right", AlignmentMiddleRight},
			{"alignment bottom_left", AlignmentBottomLeft},
			{"alignment bottom_middle", AlignmentBottomMiddle},
			{"alignment bottom_right", AlignmentBottomRight},
		}

		for _, tt := range alignments {
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
	})

	t.Run("hex colors various formats", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected color.RGBA
		}{
			{"lowercase hex", "default_color aabbcc", color.RGBA{R: 0xaa, G: 0xbb, B: 0xcc, A: 255}},
			{"uppercase hex", "default_color AABBCC", color.RGBA{R: 0xaa, G: 0xbb, B: 0xcc, A: 255}},
			{"mixed case hex", "default_color AaBbCc", color.RGBA{R: 0xaa, G: 0xbb, B: 0xcc, A: 255}},
			{"with hash", "default_color #aabbcc", color.RGBA{R: 0xaa, G: 0xbb, B: 0xcc, A: 255}},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				p := NewLegacyParser()
				cfg, err := p.Parse([]byte(tt.input))
				if err != nil {
					t.Fatalf("Parse failed: %v", err)
				}
				if cfg.Colors.Default != tt.expected {
					t.Errorf("expected %v, got %v", tt.expected, cfg.Colors.Default)
				}
			})
		}
	})
}

// TestConfigSuiteValidationIntegration tests validation with parsed configs.
func TestConfigSuiteValidationIntegration(t *testing.T) {
	t.Run("valid config passes validation", func(t *testing.T) {
		parser, err := NewParser()
		if err != nil {
			t.Fatalf("NewParser failed: %v", err)
		}
		defer parser.Close()

		cfg, err := parser.ParseFile("../../test/configs/basic.conkyrc")
		if err != nil {
			t.Fatalf("ParseFile failed: %v", err)
		}

		// Non-strict validation should pass
		if err := ValidateConfig(cfg); err != nil {
			t.Errorf("ValidateConfig failed: %v", err)
		}

		// Strict validation should also pass for known variables
		result := NewValidator().WithStrictMode(true).Validate(cfg)
		if !result.IsValid() {
			t.Errorf("Strict validation failed with errors: %v", result.Errors)
		}
	})

	t.Run("config with unknown variables generates warnings", func(t *testing.T) {
		p := NewLegacyParser()
		content := `background yes

TEXT
${unknown_variable}
${cpu}
${another_unknown}
`
		cfg, err := p.Parse([]byte(content))
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}

		// Non-strict validation should pass
		result := NewValidator().WithStrictMode(false).Validate(cfg)
		if !result.IsValid() {
			t.Errorf("Expected valid result in non-strict mode, got errors: %v", result.Errors)
		}
		if len(result.Warnings) != 2 {
			t.Errorf("Expected 2 warnings for unknown variables, got %d", len(result.Warnings))
		}

		// Strict validation should fail
		resultStrict := NewValidator().WithStrictMode(true).Validate(cfg)
		if resultStrict.IsValid() {
			t.Error("Expected validation to fail in strict mode with unknown variables")
		}
		if len(resultStrict.Errors) != 2 {
			t.Errorf("Expected 2 errors in strict mode, got %d", len(resultStrict.Errors))
		}
	})

	t.Run("parse error propagation", func(t *testing.T) {
		p := NewLegacyParser()
		invalidConfigs := []struct {
			name    string
			content string
		}{
			{"invalid window type", "own_window_type invalid"},
			{"invalid window hint", "own_window_hints invalid"},
			{"invalid alignment", "alignment invalid"},
			{"invalid color", "default_color gggggg"},
			{"invalid width", "minimum_width not_a_number"},
		}

		for _, tt := range invalidConfigs {
			t.Run(tt.name, func(t *testing.T) {
				_, err := p.Parse([]byte(tt.content))
				if err == nil {
					t.Errorf("Expected error for %s, got nil", tt.name)
				}
			})
		}
	})
}

// TestConfigSuiteLuaSpecificFeatures tests Lua-specific parsing features.
func TestConfigSuiteLuaSpecificFeatures(t *testing.T) {
	t.Run("Lua comments are ignored", func(t *testing.T) {
		p, err := NewLuaConfigParser()
		if err != nil {
			t.Fatalf("NewLuaConfigParser failed: %v", err)
		}
		defer p.Close()

		content := `
-- This is a comment
conky.config = {
    -- This is also a comment
    background = true,  -- inline comment
}
--[[ Multi-line comment
     that spans multiple lines ]]
conky.text = 'test'
`
		cfg, err := p.Parse([]byte(content))
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}
		if cfg.Display.Background != true {
			t.Error("expected background=true")
		}
	})

	t.Run("Lua multi-line strings", func(t *testing.T) {
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
		// The [[...]] syntax includes the newline after [[
		if len(cfg.Text.Template) < 3 {
			t.Errorf("expected at least 3 lines, got %d: %v", len(cfg.Text.Template), cfg.Text.Template)
		}
	})

	t.Run("Lua table with trailing comma", func(t *testing.T) {
		p, err := NewLuaConfigParser()
		if err != nil {
			t.Fatalf("NewLuaConfigParser failed: %v", err)
		}
		defer p.Close()

		content := `
conky.config = {
    background = true,
    own_window = true,
}
conky.text = ''
`
		cfg, err := p.Parse([]byte(content))
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}
		if cfg.Display.Background != true {
			t.Error("expected background=true")
		}
		if cfg.Window.OwnWindow != true {
			t.Error("expected own_window=true")
		}
	})

	t.Run("Lua single quoted and double quoted strings", func(t *testing.T) {
		p, err := NewLuaConfigParser()
		if err != nil {
			t.Fatalf("NewLuaConfigParser failed: %v", err)
		}
		defer p.Close()

		content := `
conky.config = {
    font = 'Single Quoted Font',
    default_color = "ff0000",
}
conky.text = ""
`
		cfg, err := p.Parse([]byte(content))
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}
		if cfg.Display.Font != "Single Quoted Font" {
			t.Errorf("expected 'Single Quoted Font', got %q", cfg.Display.Font)
		}
		expectedColor := color.RGBA{R: 255, G: 0, B: 0, A: 255}
		if cfg.Colors.Default != expectedColor {
			t.Errorf("expected %v, got %v", expectedColor, cfg.Colors.Default)
		}
	})
}

// TestConfigSuiteMigrationCompleteness tests that migration produces complete Lua configs.
func TestConfigSuiteMigrationCompleteness(t *testing.T) {
	t.Run("migration includes all non-default values", func(t *testing.T) {
		// Note: own_window = yes is the default, so it won't be in the output
		// unless preserveDefaults is enabled
		content := `background yes
font CustomFont:size=14
update_interval 3.0
double_buffer yes
own_window yes
own_window_type dock
own_window_transparent yes
own_window_hints undecorated,below
alignment bottom_left
minimum_width 500
minimum_height 400
gap_x 30
gap_y 40
default_color red
color0 green
color1 blue

TEXT
Test line
`
		result, err := MigrateLegacyContent([]byte(content))
		if err != nil {
			t.Fatalf("MigrateLegacyContent failed: %v", err)
		}

		output := string(result)
		// Only include settings that differ from defaults
		// own_window is default=true, so it won't be output
		// double_buffer is default=true, so it won't be output
		expectedSettings := []string{
			"background = true",
			"font = 'CustomFont:size=14'",
			"update_interval = 3.0",
			"own_window_type = 'dock'",
			"own_window_transparent = true",
			"own_window_hints = 'undecorated,below'",
			"alignment = 'bottom_left'",
			"minimum_width = 500",
			"minimum_height = 400",
			"gap_x = 30",
			"gap_y = 40",
			"default_color = 'red'",
			"color0 = 'green'",
			"color1 = 'blue'",
			"Test line",
		}

		for _, expected := range expectedSettings {
			if !strings.Contains(output, expected) {
				t.Errorf("expected %q in output, got:\n%s", expected, output)
			}
		}
	})

	t.Run("migrator options are respected", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Display.Background = true

		// With comments (default)
		m1 := NewMigrator(WithComments(true))
		result1, err := m1.MigrateToLua(&cfg)
		if err != nil {
			t.Fatalf("MigrateToLua failed: %v", err)
		}
		if !strings.Contains(string(result1), "-- ") {
			t.Error("expected comments in output with WithComments(true)")
		}

		// Without comments
		m2 := NewMigrator(WithComments(false))
		result2, err := m2.MigrateToLua(&cfg)
		if err != nil {
			t.Fatalf("MigrateToLua failed: %v", err)
		}
		// Should start with conky.config, not a comment
		trimmed := strings.TrimSpace(string(result2))
		if strings.HasPrefix(trimmed, "--") {
			t.Error("expected no comments in output with WithComments(false)")
		}
	})
}

// TestConfigSuiteDefaultsPreservation tests that default values are preserved correctly.
func TestConfigSuiteDefaultsPreservation(t *testing.T) {
	t.Run("empty config uses defaults", func(t *testing.T) {
		p := NewLegacyParser()
		cfg, err := p.Parse([]byte(""))
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}

		defaults := DefaultConfig()

		// Check defaults are applied
		if cfg.Window.Width != defaults.Window.Width {
			t.Errorf("Width: expected %d, got %d", defaults.Window.Width, cfg.Window.Width)
		}
		if cfg.Window.Height != defaults.Window.Height {
			t.Errorf("Height: expected %d, got %d", defaults.Window.Height, cfg.Window.Height)
		}
		if cfg.Display.UpdateInterval != defaults.Display.UpdateInterval {
			t.Errorf("UpdateInterval: expected %v, got %v", defaults.Display.UpdateInterval, cfg.Display.UpdateInterval)
		}
		if cfg.Display.Font != defaults.Display.Font {
			t.Errorf("Font: expected %q, got %q", defaults.Display.Font, cfg.Display.Font)
		}
	})

	t.Run("partial config preserves unset defaults", func(t *testing.T) {
		p := NewLegacyParser()
		// Only set a few values, others should use defaults
		content := `background yes
update_interval 2.0`
		cfg, err := p.Parse([]byte(content))
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}

		defaults := DefaultConfig()

		// Check overridden values
		if cfg.Display.Background != true {
			t.Error("expected background=true")
		}
		if cfg.Display.UpdateInterval != 2*time.Second {
			t.Errorf("expected update_interval=2s, got %v", cfg.Display.UpdateInterval)
		}

		// Check defaults are preserved for unset values
		if cfg.Window.Width != defaults.Window.Width {
			t.Errorf("Width should default: expected %d, got %d", defaults.Window.Width, cfg.Window.Width)
		}
		if cfg.Display.Font != defaults.Display.Font {
			t.Errorf("Font should default: expected %q, got %q", defaults.Display.Font, cfg.Display.Font)
		}
	})
}

// TestConfigSuiteLegacyParserConcurrentParsing tests that the legacy parser is safe for concurrent use.
// Note: Lua parser has a data race in the third-party golua library, so we only test
// the legacy parser here.
func TestConfigSuiteLegacyParserConcurrentParsing(t *testing.T) {
	content := `background yes
own_window yes
update_interval 1.0

TEXT
${cpu}
`
	const numGoroutines = 10

	done := make(chan bool, numGoroutines)
	errors := make(chan error, numGoroutines)

	for range numGoroutines {
		go func() {
			parser := NewLegacyParser()

			cfg, err := parser.Parse([]byte(content))
			if err != nil {
				errors <- err
				done <- true
				return
			}

			if cfg.Display.Background != true {
				errors <- fmt.Errorf("background expected true, got %v", cfg.Display.Background)
				done <- true
				return
			}

			done <- true
		}()
	}

	// Wait for all goroutines
	for range numGoroutines {
		<-done
	}

	// Check for errors
	close(errors)
	for err := range errors {
		t.Errorf("Concurrent parsing error: %v", err)
	}
}

// TestConfigSuiteAllColorVariants tests all color0-9 parsing.
func TestConfigSuiteAllColorVariants(t *testing.T) {
	colors := []struct {
		directive string
		getColor  func(*ColorConfig) color.RGBA
	}{
		{"color0", func(c *ColorConfig) color.RGBA { return c.Color0 }},
		{"color1", func(c *ColorConfig) color.RGBA { return c.Color1 }},
		{"color2", func(c *ColorConfig) color.RGBA { return c.Color2 }},
		{"color3", func(c *ColorConfig) color.RGBA { return c.Color3 }},
		{"color4", func(c *ColorConfig) color.RGBA { return c.Color4 }},
		{"color5", func(c *ColorConfig) color.RGBA { return c.Color5 }},
		{"color6", func(c *ColorConfig) color.RGBA { return c.Color6 }},
		{"color7", func(c *ColorConfig) color.RGBA { return c.Color7 }},
		{"color8", func(c *ColorConfig) color.RGBA { return c.Color8 }},
		{"color9", func(c *ColorConfig) color.RGBA { return c.Color9 }},
	}

	for _, tt := range colors {
		t.Run(tt.directive, func(t *testing.T) {
			p := NewLegacyParser()
			content := tt.directive + " ff0000"
			cfg, err := p.Parse([]byte(content))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			expected := color.RGBA{R: 255, G: 0, B: 0, A: 255}
			if tt.getColor(&cfg.Colors) != expected {
				t.Errorf("expected %v, got %v", expected, tt.getColor(&cfg.Colors))
			}
		})
	}
}
