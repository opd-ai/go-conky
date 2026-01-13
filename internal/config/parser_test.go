package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewParser(t *testing.T) {
	p, err := NewParser()
	if err != nil {
		t.Fatalf("NewParser failed: %v", err)
	}
	defer p.Close()

	if p == nil {
		t.Error("NewParser returned nil")
	}
}

func TestParserParseLegacy(t *testing.T) {
	p, err := NewParser()
	if err != nil {
		t.Fatalf("NewParser failed: %v", err)
	}
	defer p.Close()

	content := `# Legacy config
background no
font DejaVu Sans Mono:size=10
update_interval 1.0
double_buffer yes
own_window yes

TEXT
${color grey}Test$color
`
	cfg, err := p.Parse([]byte(content))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if cfg.Display.Background != false {
		t.Error("expected background=false")
	}
	if cfg.Display.Font != "DejaVu Sans Mono:size=10" {
		t.Errorf("font mismatch: got %q", cfg.Display.Font)
	}
}

func TestParserParseLua(t *testing.T) {
	p, err := NewParser()
	if err != nil {
		t.Fatalf("NewParser failed: %v", err)
	}
	defer p.Close()

	content := `
conky.config = {
    background = false,
    font = 'DejaVu Sans Mono:size=10',
    update_interval = 1.0,
    double_buffer = true,
}
conky.text = [[
${color grey}Test$color
]]
`
	cfg, err := p.Parse([]byte(content))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if cfg.Display.Background != false {
		t.Error("expected background=false")
	}
	if cfg.Display.Font != "DejaVu Sans Mono:size=10" {
		t.Errorf("font mismatch: got %q", cfg.Display.Font)
	}
}

func TestParserAutoDetection(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		isLua    bool
	}{
		{
			"legacy with TEXT",
			"background yes\nTEXT\n$cpu",
			false,
		},
		{
			"legacy with comments",
			"# comment\nbackground yes",
			false,
		},
		{
			"lua with conky.config",
			"conky.config = {}\nconky.text = ''",
			true,
		},
		{
			"lua with comments",
			"-- comment\nconky.config = {}",
			true,
		},
	}

	p, err := NewParser()
	if err != nil {
		t.Fatalf("NewParser failed: %v", err)
	}
	defer p.Close()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detected := isLuaConfig([]byte(tt.content))
			if detected != tt.isLua {
				t.Errorf("isLuaConfig() = %v, want %v", detected, tt.isLua)
			}

			// Verify parsing works
			cfg, err := p.Parse([]byte(tt.content))
			if err != nil {
				t.Errorf("Parse failed: %v", err)
				return
			}
			if cfg == nil {
				t.Error("Parse returned nil config")
			}
		})
	}
}

func TestParserParseFile(t *testing.T) {
	p, err := NewParser()
	if err != nil {
		t.Fatalf("NewParser failed: %v", err)
	}
	defer p.Close()

	// Create temp file with legacy config
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.conkyrc")
	content := `background yes
update_interval 2.0

TEXT
Test
`
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	cfg, err := p.ParseFile(tmpFile)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if cfg.Display.Background != true {
		t.Error("expected background=true")
	}
	if cfg.Display.UpdateInterval != 2*time.Second {
		t.Errorf("expected update_interval=2s, got %v", cfg.Display.UpdateInterval)
	}
}

func TestParserParseFileLua(t *testing.T) {
	p, err := NewParser()
	if err != nil {
		t.Fatalf("NewParser failed: %v", err)
	}
	defer p.Close()

	// Create temp file with Lua config
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.lua")
	content := `
conky.config = {
    background = true,
    update_interval = 2.0,
}
conky.text = [[Test]]
`
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	cfg, err := p.ParseFile(tmpFile)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if cfg.Display.Background != true {
		t.Error("expected background=true")
	}
	if cfg.Display.UpdateInterval != 2*time.Second {
		t.Errorf("expected update_interval=2s, got %v", cfg.Display.UpdateInterval)
	}
}

func TestParserParseFileNotFound(t *testing.T) {
	p, err := NewParser()
	if err != nil {
		t.Fatalf("NewParser failed: %v", err)
	}
	defer p.Close()

	_, err = p.ParseFile("/nonexistent/path/to/config")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestParserClose(t *testing.T) {
	p, err := NewParser()
	if err != nil {
		t.Fatalf("NewParser failed: %v", err)
	}

	if err := p.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Closing again should be safe
	if err := p.Close(); err != nil {
		t.Errorf("Second Close failed: %v", err)
	}
}

func TestIsLuaConfig(t *testing.T) {
	tests := []struct {
		content string
		isLua   bool
	}{
		{"conky.config = {}", true},
		{"-- comment\nconky.config = {}", true},
		{"   conky.config = {}", true},
		{"background yes", false},
		{"TEXT\n$cpu", false},
		{"# comment\nbackground yes", false},
		{"", false},
		{"conky.text = [[]]", false}, // No conky.config
	}

	for _, tt := range tests {
		t.Run(tt.content, func(t *testing.T) {
			if got := isLuaConfig([]byte(tt.content)); got != tt.isLua {
				t.Errorf("isLuaConfig(%q) = %v, want %v", tt.content, got, tt.isLua)
			}
		})
	}
}

func TestParseActualTestConfigs(t *testing.T) {
	p, err := NewParser()
	if err != nil {
		t.Fatalf("NewParser failed: %v", err)
	}
	defer p.Close()

	// Test parsing the actual test configuration files
	testFiles := []struct {
		path    string
		isLua   bool
	}{
		{"../../test/configs/basic.conkyrc", false},
		{"../../test/configs/basic_lua.conkyrc", true},
	}

	for _, tf := range testFiles {
		t.Run(tf.path, func(t *testing.T) {
			cfg, err := p.ParseFile(tf.path)
			if err != nil {
				t.Fatalf("ParseFile failed: %v", err)
			}

			// Common validations for both config formats
			if cfg.Window.OwnWindow != true {
				t.Error("expected own_window=true")
			}
			if cfg.Window.Type != WindowTypeNormal {
				t.Error("expected window type normal")
			}
			if cfg.Window.Transparent != true {
				t.Error("expected transparent=true")
			}
			if len(cfg.Text.Template) == 0 {
				t.Error("expected non-empty text template")
			}
		})
	}
}
