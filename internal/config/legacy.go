// Package config provides configuration parsing for conky-go.
// This file implements the legacy .conkyrc parser for Conky versions prior to 1.10.

package config

import (
	"bufio"
	"fmt"
	"image/color"
	"strconv"
	"strings"
	"time"
)

// LegacyParser parses legacy .conkyrc configuration files.
// The legacy format uses a simple key-value syntax with a TEXT section
// delimiter for template content.
type LegacyParser struct{}

// NewLegacyParser creates a new LegacyParser instance.
func NewLegacyParser() *LegacyParser {
	return &LegacyParser{}
}

// Parse parses a legacy .conkyrc configuration from content bytes.
// It returns a Config with parsed values or an error if parsing fails.
func (p *LegacyParser) Parse(content []byte) (*Config, error) {
	cfg := DefaultConfig()
	scanner := bufio.NewScanner(strings.NewReader(string(content)))

	var inTextSection bool
	var textLines []string
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			if inTextSection {
				textLines = append(textLines, "")
			}
			continue
		}

		// Skip comment lines (starting with #)
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Check for TEXT section marker
		if trimmed == "TEXT" {
			inTextSection = true
			continue
		}

		if inTextSection {
			// Everything after TEXT is template content (preserve original line)
			textLines = append(textLines, line)
		} else {
			// Parse configuration directive
			if err := p.parseDirective(&cfg, trimmed, lineNum); err != nil {
				return nil, err
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading configuration: %w", err)
	}

	cfg.Text.Template = textLines
	return &cfg, nil
}

// parseDirective parses a single configuration directive line.
// Format: "key value" or "key" (for boolean flags).
func (p *LegacyParser) parseDirective(cfg *Config, line string, lineNum int) error {
	// Split into key and value
	parts := strings.SplitN(line, " ", 2)
	key := strings.ToLower(parts[0])

	var value string
	if len(parts) > 1 {
		value = strings.TrimSpace(parts[1])
	}

	switch key {
	// Boolean settings
	case "background":
		cfg.Display.Background = parseBool(value)
	case "double_buffer":
		cfg.Display.DoubleBuffer = parseBool(value)
	case "own_window":
		cfg.Window.OwnWindow = parseBool(value)
	case "own_window_transparent":
		cfg.Window.Transparent = parseBool(value)

	// Window type
	case "own_window_type":
		wt, err := ParseWindowType(value)
		if err != nil {
			return fmt.Errorf("line %d: %w", lineNum, err)
		}
		cfg.Window.Type = wt

	// Window hints (comma-separated list)
	case "own_window_hints":
		hints, err := parseWindowHints(value)
		if err != nil {
			return fmt.Errorf("line %d: %w", lineNum, err)
		}
		cfg.Window.Hints = hints

	// Alignment
	case "alignment":
		a, err := ParseAlignment(value)
		if err != nil {
			return fmt.Errorf("line %d: %w", lineNum, err)
		}
		cfg.Window.Alignment = a

	// Numeric settings
	case "update_interval":
		interval, err := parseFloat(value)
		if err != nil {
			return fmt.Errorf("line %d: invalid update_interval: %w", lineNum, err)
		}
		cfg.Display.UpdateInterval = time.Duration(interval * float64(time.Second))

	case "minimum_width":
		width, err := parseInt(value)
		if err != nil {
			return fmt.Errorf("line %d: invalid minimum_width: %w", lineNum, err)
		}
		cfg.Window.Width = width

	case "minimum_height":
		height, err := parseInt(value)
		if err != nil {
			return fmt.Errorf("line %d: invalid minimum_height: %w", lineNum, err)
		}
		cfg.Window.Height = height

	case "gap_x":
		x, err := parseInt(value)
		if err != nil {
			return fmt.Errorf("line %d: invalid gap_x: %w", lineNum, err)
		}
		cfg.Window.X = x

	case "gap_y":
		y, err := parseInt(value)
		if err != nil {
			return fmt.Errorf("line %d: invalid gap_y: %w", lineNum, err)
		}
		cfg.Window.Y = y

	// Font settings
	case "font":
		cfg.Display.Font = value

	// Color settings
	case "default_color":
		c, err := parseColor(value)
		if err != nil {
			return fmt.Errorf("line %d: invalid default_color: %w", lineNum, err)
		}
		cfg.Colors.Default = c

	case "color0":
		c, err := parseColor(value)
		if err != nil {
			return fmt.Errorf("line %d: invalid color0: %w", lineNum, err)
		}
		cfg.Colors.Color0 = c

	case "color1":
		c, err := parseColor(value)
		if err != nil {
			return fmt.Errorf("line %d: invalid color1: %w", lineNum, err)
		}
		cfg.Colors.Color1 = c

	case "color2":
		c, err := parseColor(value)
		if err != nil {
			return fmt.Errorf("line %d: invalid color2: %w", lineNum, err)
		}
		cfg.Colors.Color2 = c

	case "color3":
		c, err := parseColor(value)
		if err != nil {
			return fmt.Errorf("line %d: invalid color3: %w", lineNum, err)
		}
		cfg.Colors.Color3 = c

	case "color4":
		c, err := parseColor(value)
		if err != nil {
			return fmt.Errorf("line %d: invalid color4: %w", lineNum, err)
		}
		cfg.Colors.Color4 = c

	case "color5":
		c, err := parseColor(value)
		if err != nil {
			return fmt.Errorf("line %d: invalid color5: %w", lineNum, err)
		}
		cfg.Colors.Color5 = c

	case "color6":
		c, err := parseColor(value)
		if err != nil {
			return fmt.Errorf("line %d: invalid color6: %w", lineNum, err)
		}
		cfg.Colors.Color6 = c

	case "color7":
		c, err := parseColor(value)
		if err != nil {
			return fmt.Errorf("line %d: invalid color7: %w", lineNum, err)
		}
		cfg.Colors.Color7 = c

	case "color8":
		c, err := parseColor(value)
		if err != nil {
			return fmt.Errorf("line %d: invalid color8: %w", lineNum, err)
		}
		cfg.Colors.Color8 = c

	case "color9":
		c, err := parseColor(value)
		if err != nil {
			return fmt.Errorf("line %d: invalid color9: %w", lineNum, err)
		}
		cfg.Colors.Color9 = c

	// Unknown directives are silently ignored to maintain forward compatibility
	default:
		// Ignore unknown directives for compatibility with future Conky versions
	}

	return nil
}

// parseBool parses a boolean value from common string representations.
// Accepts: yes, no, true, false, 1, 0
func parseBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "yes", "true", "1":
		return true
	default:
		return false
	}
}

// parseFloat parses a float64 from a string.
func parseFloat(s string) (float64, error) {
	s = strings.TrimSpace(s)
	return strconv.ParseFloat(s, 64)
}

// parseInt parses an int from a string.
func parseInt(s string) (int, error) {
	s = strings.TrimSpace(s)
	return strconv.Atoi(s)
}

// parseWindowHints parses a comma-separated list of window hints.
func parseWindowHints(s string) ([]WindowHint, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}

	parts := strings.Split(s, ",")
	hints := make([]WindowHint, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		hint, err := ParseWindowHint(part)
		if err != nil {
			return nil, err
		}
		hints = append(hints, hint)
	}

	return hints, nil
}

// colorNames maps common color names to RGBA values.
var colorNames = map[string]color.RGBA{
	"white":   {R: 255, G: 255, B: 255, A: 255},
	"black":   {R: 0, G: 0, B: 0, A: 255},
	"red":     {R: 255, G: 0, B: 0, A: 255},
	"green":   {R: 0, G: 255, B: 0, A: 255},
	"blue":    {R: 0, G: 0, B: 255, A: 255},
	"yellow":  {R: 255, G: 255, B: 0, A: 255},
	"cyan":    {R: 0, G: 255, B: 255, A: 255},
	"magenta": {R: 255, G: 0, B: 255, A: 255},
	"grey":    {R: 128, G: 128, B: 128, A: 255},
	"gray":    {R: 128, G: 128, B: 128, A: 255},
	"orange":  {R: 255, G: 165, B: 0, A: 255},
}

// parseColor parses a color from a name or hex value.
// Hex values can be in format: RRGGBB or #RRGGBB
func parseColor(s string) (color.RGBA, error) {
	s = strings.TrimSpace(strings.ToLower(s))

	// Check named colors first
	if c, ok := colorNames[s]; ok {
		return c, nil
	}

	// Handle hex color
	hex := strings.TrimPrefix(s, "#")
	if len(hex) != 6 {
		return color.RGBA{}, fmt.Errorf("invalid color format: %s", s)
	}

	r, err := strconv.ParseUint(hex[0:2], 16, 8)
	if err != nil {
		return color.RGBA{}, fmt.Errorf("invalid red component in color: %s", s)
	}

	g, err := strconv.ParseUint(hex[2:4], 16, 8)
	if err != nil {
		return color.RGBA{}, fmt.Errorf("invalid green component in color: %s", s)
	}

	b, err := strconv.ParseUint(hex[4:6], 16, 8)
	if err != nil {
		return color.RGBA{}, fmt.Errorf("invalid blue component in color: %s", s)
	}

	return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}, nil
}
