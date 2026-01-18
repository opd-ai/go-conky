// Package config provides configuration parsing for conky-go.
// This file contains fuzzing tests for the configuration parsers to ensure
// robustness against malformed or unexpected input.

package config

import (
	"testing"
)

// FuzzLegacyParser tests the legacy .conkyrc parser with arbitrary input.
// It ensures the parser handles malformed configuration gracefully without panicking.
func FuzzLegacyParser(f *testing.F) {
	// Add seed corpus with valid configurations
	f.Add([]byte(`background yes
own_window yes
TEXT
CPU: ${cpu}%`))

	f.Add([]byte(`# comment line
alignment top_left
update_interval 1.0
minimum_width 200
TEXT
${mem}/${memmax}`))

	// Edge cases
	f.Add([]byte(""))           // empty
	f.Add([]byte("TEXT"))       // only TEXT marker
	f.Add([]byte("TEXT\n"))     // TEXT with newline
	f.Add([]byte("\n\n\n"))     // only newlines
	f.Add([]byte("# comment"))  // only comment
	f.Add([]byte("key"))        // key without value
	f.Add([]byte("key value"))  // minimal directive

	// Malformed inputs
	f.Add([]byte("update_interval not_a_number"))
	f.Add([]byte("minimum_width -999999999999"))
	f.Add([]byte("default_color invalid_color_name"))
	f.Add([]byte("own_window_argb_value 99999"))
	f.Add([]byte("alignment invalid_alignment"))
	f.Add([]byte("own_window_type bad_type"))

	f.Fuzz(func(t *testing.T, data []byte) {
		parser := NewLegacyParser()
		// Parse should not panic
		cfg, err := parser.Parse(data)

		if err == nil && cfg == nil {
			t.Error("Parse returned nil config with nil error")
		}
	})
}

// FuzzLuaParser tests the Lua configuration parser with arbitrary input.
// It ensures the parser handles malformed Lua code gracefully without panicking.
func FuzzLuaParser(f *testing.F) {
	// Add seed corpus with valid Lua configurations
	f.Add([]byte(`conky.config = {
    background = true,
    own_window = true,
    update_interval = 1.0,
}

conky.text = [[
CPU: ${cpu}%
]]`))

	f.Add([]byte(`conky.config = {
    alignment = 'top_left',
    minimum_width = 200,
    minimum_height = 100,
    gap_x = 10,
    gap_y = 10,
}

conky.text = ''`))

	// Edge cases
	f.Add([]byte(""))                   // empty
	f.Add([]byte("conky.config = {}"))  // minimal valid config
	f.Add([]byte("conky.text = ''"))    // only text
	f.Add([]byte("-- comment only"))    // Lua comment
	f.Add([]byte("local x = 1"))        // valid Lua but no conky table

	// Malformed Lua
	f.Add([]byte("conky.config = {"))   // unclosed brace
	f.Add([]byte("conky.config = nil")) // nil config
	f.Add([]byte("error('test')"))      // Lua error

	// Edge case values
	f.Add([]byte(`conky.config = { update_interval = -1 }`))
	f.Add([]byte(`conky.config = { minimum_width = 999999999 }`))
	f.Add([]byte(`conky.config = { alignment = 'invalid' }`))

	f.Fuzz(func(t *testing.T, data []byte) {
		parser, err := NewLuaConfigParser()
		if err != nil {
			t.Skip("failed to create Lua parser")
		}
		defer parser.Close()

		// Parse should not panic
		cfg, err := parser.Parse(data)

		if err == nil && cfg == nil {
			t.Error("Parse returned nil config with nil error")
		}
	})
}

// FuzzParseColor tests the color parsing function with arbitrary input.
// It ensures parseColor handles malformed color values gracefully.
func FuzzParseColor(f *testing.F) {
	// Valid colors
	f.Add("white")
	f.Add("black")
	f.Add("red")
	f.Add("#FF0000")
	f.Add("FF0000")
	f.Add("#ffffff")
	f.Add("000000")

	// Edge cases
	f.Add("")
	f.Add("#")
	f.Add("###")
	f.Add("#FFF")                // 3-char hex (not supported)
	f.Add("#FFFFFFFF")           // 8-char hex (with alpha)
	f.Add("notacolor")
	f.Add("RED")                 // uppercase
	f.Add("  white  ")           // whitespace
	f.Add("#GGG000")             // invalid hex chars

	f.Fuzz(func(t *testing.T, data string) {
		// parseColor should not panic
		_, _ = parseColor(data)
	})
}

// FuzzParseWindowHints tests the window hints parsing function.
func FuzzParseWindowHints(f *testing.F) {
	// Valid hints
	f.Add("undecorated")
	f.Add("below,sticky")
	f.Add("above, skip_taskbar, skip_pager")

	// Edge cases
	f.Add("")
	f.Add(",")
	f.Add(",,")
	f.Add("  ")
	f.Add("undecorated,")
	f.Add(",undecorated")
	f.Add("invalid_hint")
	f.Add("below, invalid, sticky")

	f.Fuzz(func(t *testing.T, data string) {
		// parseWindowHints should not panic
		_, _ = parseWindowHints(data)
	})
}

// FuzzParseAlignment tests the alignment parsing function.
func FuzzParseAlignment(f *testing.F) {
	// Valid alignments
	f.Add("top_left")
	f.Add("top_middle")
	f.Add("top_right")
	f.Add("middle_left")
	f.Add("middle_middle")
	f.Add("middle_right")
	f.Add("bottom_left")
	f.Add("bottom_middle")
	f.Add("bottom_right")
	f.Add("tl")
	f.Add("tm")
	f.Add("tr")

	// Edge cases
	f.Add("")
	f.Add("invalid")
	f.Add("TOP_LEFT")
	f.Add("  top_left  ")
	f.Add("top")
	f.Add("left")

	f.Fuzz(func(t *testing.T, data string) {
		// ParseAlignment should not panic
		_, _ = ParseAlignment(data)
	})
}

// FuzzParseWindowType tests the window type parsing function.
func FuzzParseWindowType(f *testing.F) {
	// Valid window types
	f.Add("normal")
	f.Add("desktop")
	f.Add("dock")
	f.Add("panel")
	f.Add("override")

	// Edge cases
	f.Add("")
	f.Add("invalid")
	f.Add("NORMAL")
	f.Add("  desktop  ")

	f.Fuzz(func(t *testing.T, data string) {
		// ParseWindowType should not panic
		_, _ = ParseWindowType(data)
	})
}

// FuzzParseBackgroundMode tests the background mode parsing function.
func FuzzParseBackgroundMode(f *testing.F) {
	// Valid background modes
	f.Add("solid")
	f.Add("none")
	f.Add("transparent")
	f.Add("gradient")
	f.Add("pseudo")
	f.Add("pseudo-transparent")
	f.Add("pseudo_transparent")

	// Edge cases
	f.Add("")
	f.Add("invalid")
	f.Add("SOLID")
	f.Add("  none  ")

	f.Fuzz(func(t *testing.T, data string) {
		// ParseBackgroundMode should not panic
		_, _ = ParseBackgroundMode(data)
	})
}

// FuzzIsLuaConfig tests the format detection function.
func FuzzIsLuaConfig(f *testing.F) {
	// Lua format
	f.Add([]byte("conky.config = {}"))
	f.Add([]byte("  conky.config = {}"))
	f.Add([]byte("\nconky.config = {}"))
	f.Add([]byte("-- comment\nconky.config = {}"))

	// Legacy format
	f.Add([]byte("background yes\nTEXT"))
	f.Add([]byte("# comment with conky.config"))
	f.Add([]byte(""))

	// Edge cases
	f.Add([]byte("conky.config"))        // no equals
	f.Add([]byte("conky.config={}"))     // no space
	f.Add([]byte("CONKY.CONFIG = {}"))   // uppercase

	f.Fuzz(func(t *testing.T, data []byte) {
		// isLuaConfig should not panic
		_ = isLuaConfig(data)
	})
}
