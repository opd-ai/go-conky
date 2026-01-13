// Package config provides configuration parsing for conky-go.
// This file implements the modern Lua configuration parser for Conky 1.10+ format.

package config

import (
	"fmt"
	"image/color"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/arnodel/golua/lib"
	rt "github.com/arnodel/golua/runtime"
)

// LuaConfigParser parses modern Lua configuration files (Conky 1.10+ format).
// It uses the Golua runtime to execute Lua code and extract configuration values
// from the conky.config table and conky.text variable.
type LuaConfigParser struct {
	runtime *rt.Runtime
	cleanup func()
	mu      sync.Mutex
}

// NewLuaConfigParser creates a new LuaConfigParser with a fresh Lua runtime.
func NewLuaConfigParser() (*LuaConfigParser, error) {
	return NewLuaConfigParserWithOutput(io.Discard)
}

// NewLuaConfigParserWithOutput creates a LuaConfigParser with custom output.
func NewLuaConfigParserWithOutput(stdout io.Writer) (*LuaConfigParser, error) {
	if stdout == nil {
		stdout = os.Stdout
	}

	runtime := rt.New(stdout)
	cleanup := lib.LoadAll(runtime)

	return &LuaConfigParser{
		runtime: runtime,
		cleanup: cleanup,
	}, nil
}

// Parse parses a Lua configuration from content bytes.
// It executes the Lua code and extracts configuration from conky.config and conky.text.
func (p *LuaConfigParser) Parse(content []byte) (*Config, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Initialize conky global table
	p.initConkyGlobal()

	// Compile and execute the Lua configuration
	closure, err := p.runtime.CompileAndLoadLuaChunk(
		"config",
		content,
		rt.TableValue(p.runtime.GlobalEnv()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to compile Lua configuration: %w", err)
	}

	// Execute with resource limits
	ctx := rt.RuntimeContextDef{
		HardLimits: rt.RuntimeResources{
			Cpu:    10_000_000,
			Memory: 50 * 1024 * 1024, // 50 MB
		},
	}
	p.runtime.PushContext(ctx)
	defer p.runtime.PopContext()

	thread := p.runtime.MainThread()
	_, err = rt.Call1(thread, rt.FunctionValue(closure))
	if err != nil {
		return nil, fmt.Errorf("failed to execute Lua configuration: %w", err)
	}

	// Extract configuration from conky.config table
	return p.extractConfig()
}

// initConkyGlobal initializes the conky global table for configuration parsing.
func (p *LuaConfigParser) initConkyGlobal() {
	conkyTable := rt.NewTable()

	// Initialize empty config table
	configTable := rt.NewTable()
	conkyTable.Set(rt.StringValue("config"), rt.TableValue(configTable))

	// Initialize empty text
	conkyTable.Set(rt.StringValue("text"), rt.StringValue(""))

	p.runtime.GlobalEnv().Set(rt.StringValue("conky"), rt.TableValue(conkyTable))
}

// extractConfig extracts configuration values from the conky global table.
func (p *LuaConfigParser) extractConfig() (*Config, error) {
	cfg := DefaultConfig()

	// Get conky global
	conkyVal := p.runtime.GlobalEnv().Get(rt.StringValue("conky"))
	if conkyVal == rt.NilValue {
		return &cfg, nil // Return defaults if no conky table
	}

	conkyTable, ok := conkyVal.TryTable()
	if !ok {
		return nil, fmt.Errorf("conky is not a table")
	}

	// Extract conky.config table
	configVal := conkyTable.Get(rt.StringValue("config"))
	if configTable, ok := configVal.TryTable(); ok {
		if err := p.extractConfigTable(&cfg, configTable); err != nil {
			return nil, err
		}
	}

	// Extract conky.text
	textVal := conkyTable.Get(rt.StringValue("text"))
	if textStr, ok := textVal.TryString(); ok {
		// Split text into lines, preserving the template format
		lines := strings.Split(textStr, "\n")
		cfg.Text.Template = lines
	}

	return &cfg, nil
}

// extractConfigTable extracts configuration values from the conky.config table.
func (p *LuaConfigParser) extractConfigTable(cfg *Config, table *rt.Table) error {
	// Boolean settings
	if val := getTableBool(table, "background"); val != nil {
		cfg.Display.Background = *val
	}
	if val := getTableBool(table, "double_buffer"); val != nil {
		cfg.Display.DoubleBuffer = *val
	}
	if val := getTableBool(table, "own_window"); val != nil {
		cfg.Window.OwnWindow = *val
	}
	if val := getTableBool(table, "own_window_transparent"); val != nil {
		cfg.Window.Transparent = *val
	}

	// Numeric settings
	if val := getTableFloat(table, "update_interval"); val != nil {
		cfg.Display.UpdateInterval = time.Duration(*val * float64(time.Second))
	}
	if val := getTableInt(table, "minimum_width"); val != nil {
		cfg.Window.Width = *val
	}
	if val := getTableInt(table, "minimum_height"); val != nil {
		cfg.Window.Height = *val
	}
	if val := getTableInt(table, "gap_x"); val != nil {
		cfg.Window.X = *val
	}
	if val := getTableInt(table, "gap_y"); val != nil {
		cfg.Window.Y = *val
	}

	// String settings
	if val := getTableString(table, "font"); val != nil {
		cfg.Display.Font = *val
	}

	// Window type
	if val := getTableString(table, "own_window_type"); val != nil {
		wt, err := ParseWindowType(*val)
		if err != nil {
			return fmt.Errorf("invalid own_window_type: %w", err)
		}
		cfg.Window.Type = wt
	}

	// Window hints (comma-separated string)
	if val := getTableString(table, "own_window_hints"); val != nil {
		hints, err := parseWindowHints(*val)
		if err != nil {
			return fmt.Errorf("invalid own_window_hints: %w", err)
		}
		cfg.Window.Hints = hints
	}

	// Alignment
	if val := getTableString(table, "alignment"); val != nil {
		a, err := ParseAlignment(*val)
		if err != nil {
			return fmt.Errorf("invalid alignment: %w", err)
		}
		cfg.Window.Alignment = a
	}

	// Color settings
	if err := p.extractColors(cfg, table); err != nil {
		return err
	}

	return nil
}

// extractColors extracts color configuration from the table.
func (p *LuaConfigParser) extractColors(cfg *Config, table *rt.Table) error {
	colorFields := []struct {
		key    string
		target *color.RGBA
	}{
		{"default_color", &cfg.Colors.Default},
		{"color0", &cfg.Colors.Color0},
		{"color1", &cfg.Colors.Color1},
		{"color2", &cfg.Colors.Color2},
		{"color3", &cfg.Colors.Color3},
		{"color4", &cfg.Colors.Color4},
		{"color5", &cfg.Colors.Color5},
		{"color6", &cfg.Colors.Color6},
		{"color7", &cfg.Colors.Color7},
		{"color8", &cfg.Colors.Color8},
		{"color9", &cfg.Colors.Color9},
	}

	for _, cf := range colorFields {
		if val := getTableString(table, cf.key); val != nil {
			c, err := parseColor(*val)
			if err != nil {
				return fmt.Errorf("invalid %s: %w", cf.key, err)
			}
			*cf.target = c
		}
	}

	return nil
}

// Close releases resources associated with the parser's Lua runtime.
func (p *LuaConfigParser) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cleanup != nil {
		p.cleanup()
		p.cleanup = nil
	}
	return nil
}

// getTableBool retrieves a boolean value from a Lua table.
// Returns nil if the key doesn't exist or is not a boolean.
func getTableBool(table *rt.Table, key string) *bool {
	val := table.Get(rt.StringValue(key))
	if val == rt.NilValue {
		return nil
	}

	// Handle actual booleans
	if b, ok := val.TryBool(); ok {
		return &b
	}

	// Handle string "true"/"false" for compatibility
	if s, ok := val.TryString(); ok {
		b := parseBool(s)
		return &b
	}

	return nil
}

// getTableString retrieves a string value from a Lua table.
// Returns nil if the key doesn't exist or is not a string.
func getTableString(table *rt.Table, key string) *string {
	val := table.Get(rt.StringValue(key))
	if val == rt.NilValue {
		return nil
	}

	if s, ok := val.TryString(); ok {
		return &s
	}

	return nil
}

// getTableFloat retrieves a float64 value from a Lua table.
// Returns nil if the key doesn't exist or is not a number.
func getTableFloat(table *rt.Table, key string) *float64 {
	val := table.Get(rt.StringValue(key))
	if val == rt.NilValue {
		return nil
	}

	if n, ok := val.TryFloat(); ok {
		return &n
	}

	// Try int conversion
	if n, ok := val.TryInt(); ok {
		f := float64(n)
		return &f
	}

	return nil
}

// getTableInt retrieves an int value from a Lua table.
// Returns nil if the key doesn't exist or is not a number.
func getTableInt(table *rt.Table, key string) *int {
	val := table.Get(rt.StringValue(key))
	if val == rt.NilValue {
		return nil
	}

	if n, ok := val.TryInt(); ok {
		i := int(n)
		return &i
	}

	// Try float conversion (truncate)
	if f, ok := val.TryFloat(); ok {
		i := int(f)
		return &i
	}

	return nil
}
