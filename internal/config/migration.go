// Package config provides configuration parsing and migration for conky-go.
// This file implements migration tools for converting legacy .conkyrc configurations
// to the modern Lua configuration format (Conky 1.10+).
package config

import (
	"bytes"
	"fmt"
	"image/color"
	"os"
	"strings"
)

// Migrator provides tools for converting legacy .conkyrc configurations
// to the modern Lua configuration format.
type Migrator struct {
	// includeComments adds explanatory comments to the output.
	includeComments bool
	// preserveDefaults includes settings even when they match defaults.
	preserveDefaults bool
}

// MigratorOption is a functional option for configuring a Migrator.
type MigratorOption func(*Migrator)

// WithComments enables adding explanatory comments to the Lua output.
func WithComments(include bool) MigratorOption {
	return func(m *Migrator) {
		m.includeComments = include
	}
}

// WithDefaults includes settings that match default values in the output.
func WithDefaults(preserve bool) MigratorOption {
	return func(m *Migrator) {
		m.preserveDefaults = preserve
	}
}

// NewMigrator creates a new Migrator with the given options.
func NewMigrator(opts ...MigratorOption) *Migrator {
	m := &Migrator{
		includeComments:  true,
		preserveDefaults: false,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// MigrateToLua converts a Config to modern Lua configuration format.
// Returns the Lua configuration as bytes suitable for writing to a file.
func (m *Migrator) MigrateToLua(cfg *Config) ([]byte, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}

	var buf bytes.Buffer

	// Write header comment
	if m.includeComments {
		buf.WriteString("-- Conky Lua configuration\n")
		buf.WriteString("-- Converted from legacy .conkyrc format by conky-go\n")
		buf.WriteString("-- See https://github.com/brndnmtthws/conky/wiki/Configuration-Settings\n\n")
	}

	// Write conky.config table
	buf.WriteString("conky.config = {\n")
	m.writeConfigTable(&buf, cfg)
	buf.WriteString("}\n\n")

	// Write conky.text
	m.writeTextSection(&buf, cfg)

	return buf.Bytes(), nil
}

// writeConfigTable writes the conky.config table contents.
func (m *Migrator) writeConfigTable(buf *bytes.Buffer, cfg *Config) {
	defaults := DefaultConfig()

	// Write display settings
	if m.includeComments {
		buf.WriteString("    -- Display settings\n")
	}
	if m.preserveDefaults || cfg.Display.Background != defaults.Display.Background {
		m.writeBool(buf, "background", cfg.Display.Background)
	}
	if m.preserveDefaults || cfg.Display.DoubleBuffer != defaults.Display.DoubleBuffer {
		m.writeBool(buf, "double_buffer", cfg.Display.DoubleBuffer)
	}
	if m.preserveDefaults || cfg.Display.UpdateInterval != defaults.Display.UpdateInterval {
		m.writeFloat(buf, "update_interval", cfg.Display.UpdateInterval.Seconds())
	}
	if cfg.Display.Font != "" && (m.preserveDefaults || cfg.Display.Font != defaults.Display.Font) {
		m.writeString(buf, "font", cfg.Display.Font)
	}

	// Write window settings
	if m.includeComments {
		buf.WriteString("\n    -- Window settings\n")
	}
	if m.preserveDefaults || cfg.Window.OwnWindow != defaults.Window.OwnWindow {
		m.writeBool(buf, "own_window", cfg.Window.OwnWindow)
	}
	if m.preserveDefaults || cfg.Window.Type != defaults.Window.Type {
		m.writeString(buf, "own_window_type", cfg.Window.Type.String())
	}
	if m.preserveDefaults || cfg.Window.Transparent != defaults.Window.Transparent {
		m.writeBool(buf, "own_window_transparent", cfg.Window.Transparent)
	}
	if len(cfg.Window.Hints) > 0 {
		m.writeWindowHints(buf, cfg.Window.Hints)
	}
	if m.preserveDefaults || cfg.Window.Alignment != defaults.Window.Alignment {
		m.writeString(buf, "alignment", cfg.Window.Alignment.String())
	}

	// Write window dimensions
	if m.includeComments {
		buf.WriteString("\n    -- Window dimensions\n")
	}
	if m.preserveDefaults || cfg.Window.Width != defaults.Window.Width {
		m.writeInt(buf, "minimum_width", cfg.Window.Width)
	}
	if m.preserveDefaults || cfg.Window.Height != defaults.Window.Height {
		m.writeInt(buf, "minimum_height", cfg.Window.Height)
	}
	if m.preserveDefaults || cfg.Window.X != defaults.Window.X {
		m.writeInt(buf, "gap_x", cfg.Window.X)
	}
	if m.preserveDefaults || cfg.Window.Y != defaults.Window.Y {
		m.writeInt(buf, "gap_y", cfg.Window.Y)
	}

	// Write color settings
	if m.hasNonDefaultColors(cfg, defaults) {
		if m.includeComments {
			buf.WriteString("\n    -- Colors\n")
		}
		m.writeColors(buf, cfg, defaults)
	}
}

// hasNonDefaultColors checks if any color settings differ from defaults.
func (m *Migrator) hasNonDefaultColors(cfg *Config, defaults Config) bool {
	if m.preserveDefaults {
		return true
	}
	return cfg.Colors.Default != defaults.Colors.Default ||
		cfg.Colors.Color0 != defaults.Colors.Color0 ||
		cfg.Colors.Color1 != defaults.Colors.Color1 ||
		cfg.Colors.Color2 != defaults.Colors.Color2 ||
		cfg.Colors.Color3 != defaults.Colors.Color3 ||
		cfg.Colors.Color4 != defaults.Colors.Color4 ||
		cfg.Colors.Color5 != defaults.Colors.Color5 ||
		cfg.Colors.Color6 != defaults.Colors.Color6 ||
		cfg.Colors.Color7 != defaults.Colors.Color7 ||
		cfg.Colors.Color8 != defaults.Colors.Color8 ||
		cfg.Colors.Color9 != defaults.Colors.Color9
}

// writeColors writes color settings to the buffer.
func (m *Migrator) writeColors(buf *bytes.Buffer, cfg *Config, defaults Config) {
	colorFields := []struct {
		name  string
		value color.RGBA
		def   color.RGBA
	}{
		{"default_color", cfg.Colors.Default, defaults.Colors.Default},
		{"color0", cfg.Colors.Color0, defaults.Colors.Color0},
		{"color1", cfg.Colors.Color1, defaults.Colors.Color1},
		{"color2", cfg.Colors.Color2, defaults.Colors.Color2},
		{"color3", cfg.Colors.Color3, defaults.Colors.Color3},
		{"color4", cfg.Colors.Color4, defaults.Colors.Color4},
		{"color5", cfg.Colors.Color5, defaults.Colors.Color5},
		{"color6", cfg.Colors.Color6, defaults.Colors.Color6},
		{"color7", cfg.Colors.Color7, defaults.Colors.Color7},
		{"color8", cfg.Colors.Color8, defaults.Colors.Color8},
		{"color9", cfg.Colors.Color9, defaults.Colors.Color9},
	}

	for _, cf := range colorFields {
		if m.preserveDefaults || cf.value != cf.def {
			m.writeColor(buf, cf.name, cf.value)
		}
	}
}

// writeTextSection writes the conky.text section.
func (m *Migrator) writeTextSection(buf *bytes.Buffer, cfg *Config) {
	if m.includeComments {
		buf.WriteString("-- Text template (uses Conky variables)\n")
	}

	buf.WriteString("conky.text = [[\n")
	for _, line := range cfg.Text.Template {
		buf.WriteString(line)
		buf.WriteString("\n")
	}
	buf.WriteString("]]\n")
}

// writeBool writes a boolean setting to the buffer.
func (m *Migrator) writeBool(buf *bytes.Buffer, name string, value bool) {
	fmt.Fprintf(buf, "    %s = %t,\n", name, value)
}

// writeString writes a string setting to the buffer.
func (m *Migrator) writeString(buf *bytes.Buffer, name, value string) {
	// Escape single quotes in the value
	escaped := strings.ReplaceAll(value, "'", "\\'")
	fmt.Fprintf(buf, "    %s = '%s',\n", name, escaped)
}

// writeInt writes an integer setting to the buffer.
func (m *Migrator) writeInt(buf *bytes.Buffer, name string, value int) {
	fmt.Fprintf(buf, "    %s = %d,\n", name, value)
}

// writeFloat writes a float setting to the buffer.
func (m *Migrator) writeFloat(buf *bytes.Buffer, name string, value float64) {
	// Format with minimal decimal places
	if value == float64(int(value)) {
		fmt.Fprintf(buf, "    %s = %.1f,\n", name, value)
	} else {
		fmt.Fprintf(buf, "    %s = %g,\n", name, value)
	}
}

// writeWindowHints writes window hints as a comma-separated string.
func (m *Migrator) writeWindowHints(buf *bytes.Buffer, hints []WindowHint) {
	if len(hints) == 0 {
		return
	}

	hintStrs := make([]string, len(hints))
	for i, hint := range hints {
		hintStrs[i] = hint.String()
	}
	fmt.Fprintf(buf, "    own_window_hints = '%s',\n", strings.Join(hintStrs, ","))
}

// writeColor writes a color setting to the buffer.
func (m *Migrator) writeColor(buf *bytes.Buffer, name string, c color.RGBA) {
	// Try to find a named color first
	colorName := colorToName(c)
	if colorName != "" {
		m.writeString(buf, name, colorName)
		return
	}

	// Fall back to hex format
	hex := fmt.Sprintf("%02x%02x%02x", c.R, c.G, c.B)
	m.writeString(buf, name, hex)
}

// reverseColorNames provides a deterministic mapping from RGBA colors to names.
// This avoids non-deterministic iteration order of the colorNames map.
// When multiple names map to the same color (e.g., "grey" and "gray"),
// we prefer the first entry in this list for consistent output.
var reverseColorNames = map[color.RGBA]string{
	{R: 255, G: 255, B: 255, A: 255}: "white",
	{R: 0, G: 0, B: 0, A: 255}:       "black",
	{R: 255, G: 0, B: 0, A: 255}:     "red",
	{R: 0, G: 255, B: 0, A: 255}:     "green",
	{R: 0, G: 0, B: 255, A: 255}:     "blue",
	{R: 255, G: 255, B: 0, A: 255}:   "yellow",
	{R: 0, G: 255, B: 255, A: 255}:   "cyan",
	{R: 255, G: 0, B: 255, A: 255}:   "magenta",
	{R: 128, G: 128, B: 128, A: 255}: "grey", // prefer "grey" over "gray"
	{R: 255, G: 165, B: 0, A: 255}:   "orange",
}

// colorToName converts an RGBA color to a named color if possible.
// Uses a reverse lookup map for deterministic and efficient lookups.
func colorToName(c color.RGBA) string {
	if name, ok := reverseColorNames[c]; ok {
		return name
	}
	return ""
}

// MigrateLegacyFile reads a legacy .conkyrc file and converts it to Lua format.
// This is a convenience function that combines parsing and migration.
func MigrateLegacyFile(path string, opts ...MigratorOption) ([]byte, error) {
	parser := NewLegacyParser()
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	cfg, err := parser.Parse(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse legacy config: %w", err)
	}

	migrator := NewMigrator(opts...)
	return migrator.MigrateToLua(cfg)
}

// MigrateLegacyContent converts legacy .conkyrc content to Lua format.
func MigrateLegacyContent(content []byte, opts ...MigratorOption) ([]byte, error) {
	parser := NewLegacyParser()
	cfg, err := parser.Parse(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse legacy config: %w", err)
	}

	migrator := NewMigrator(opts...)
	return migrator.MigrateToLua(cfg)
}
