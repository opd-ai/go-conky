// Package config provides configuration parsing for conky-go.
// This file implements the unified parser that auto-detects the configuration format.

package config

import (
	"fmt"
	"os"
	"regexp"
)

// Parser provides a unified interface for parsing Conky configuration files.
// It automatically detects whether a file uses legacy (.conkyrc) or modern (Lua) format.
type Parser struct {
	legacyParser *LegacyParser
	luaParser    *LuaConfigParser
}

// NewParser creates a new Parser that can handle both legacy and Lua configurations.
func NewParser() (*Parser, error) {
	luaParser, err := NewLuaConfigParser()
	if err != nil {
		return nil, fmt.Errorf("failed to create Lua parser: %w", err)
	}

	return &Parser{
		legacyParser: NewLegacyParser(),
		luaParser:    luaParser,
	}, nil
}

// ParseFile reads and parses a configuration file, auto-detecting the format.
// Returns a Config on success or an error if parsing fails.
func (p *Parser) ParseFile(path string) (*Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	return p.Parse(content)
}

// Parse parses configuration content, auto-detecting the format.
// It uses the presence of "conky.config = " pattern to detect Lua format.
func (p *Parser) Parse(content []byte) (*Config, error) {
	if isLuaConfig(content) {
		return p.luaParser.Parse(content)
	}
	return p.legacyParser.Parse(content)
}

// luaConfigPattern matches "conky.config" followed by optional whitespace and "="
// at the start of a line (not inside a comment).
// This pattern identifies modern Lua configuration format and reduces false positives
// from comments in legacy configs that might mention "conky.config".
var luaConfigPattern = regexp.MustCompile(`(?m)^\s*conky\.config\s*=`)

// isLuaConfig determines if the content is a Lua configuration.
// It uses a regex pattern to match "conky.config =" at the start of a line,
// which is the Lua format marker.
func isLuaConfig(content []byte) bool {
	return luaConfigPattern.Match(content)
}

// Close releases resources associated with the parser.
func (p *Parser) Close() error {
	if p.luaParser != nil {
		return p.luaParser.Close()
	}
	return nil
}
