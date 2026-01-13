// Package config provides configuration parsing for conky-go.
// This file implements the unified parser that auto-detects the configuration format.

package config

import (
	"fmt"
	"os"
	"strings"
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
// It uses the presence of "conky.config" to detect Lua format.
func (p *Parser) Parse(content []byte) (*Config, error) {
	if isLuaConfig(content) {
		return p.luaParser.Parse(content)
	}
	return p.legacyParser.Parse(content)
}

// isLuaConfig determines if the content is a Lua configuration.
// It checks for the presence of "conky.config" which is the modern Lua format marker.
func isLuaConfig(content []byte) bool {
	s := string(content)
	return strings.Contains(s, "conky.config")
}

// Close releases resources associated with the parser.
func (p *Parser) Close() error {
	if p.luaParser != nil {
		return p.luaParser.Close()
	}
	return nil
}
