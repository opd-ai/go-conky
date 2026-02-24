package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	if Version == "" {
		t.Error("Version should not be empty")
	}
}

func TestParseFlags(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantConfig  string
		wantVer     bool
		wantCPU     string
		wantMem     string
		wantConv    string
		wantWatch   bool
		wantErr     bool
	}{
		{
			name:    "no flags",
			args:    []string{},
			wantErr: false,
		},
		{
			name:       "config flag",
			args:       []string{"-c", "/path/to/config"},
			wantConfig: "/path/to/config",
		},
		{
			name:    "version flag",
			args:    []string{"-v"},
			wantVer: true,
		},
		{
			name:    "cpu profile flag",
			args:    []string{"-cpuprofile", "cpu.prof"},
			wantCPU: "cpu.prof",
		},
		{
			name:    "mem profile flag",
			args:    []string{"-memprofile", "mem.prof"},
			wantMem: "mem.prof",
		},
		{
			name:     "convert flag",
			args:     []string{"-convert", "old.conkyrc"},
			wantConv: "old.conkyrc",
		},
		{
			name:      "watch flag",
			args:      []string{"-w"},
			wantWatch: true,
		},
		{
			name:       "all flags",
			args:       []string{"-c", "cfg", "-v", "-cpuprofile", "c.prof", "-memprofile", "m.prof", "-w"},
			wantConfig: "cfg",
			wantVer:    true,
			wantCPU:    "c.prof",
			wantMem:    "m.prof",
			wantWatch:  true,
		},
		{
			name:    "unknown flag",
			args:    []string{"-unknown"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags, err := parseFlags(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if flags.configPath != tt.wantConfig {
				t.Errorf("configPath = %q, want %q", flags.configPath, tt.wantConfig)
			}
			if flags.version != tt.wantVer {
				t.Errorf("version = %v, want %v", flags.version, tt.wantVer)
			}
			if flags.cpuProfile != tt.wantCPU {
				t.Errorf("cpuProfile = %q, want %q", flags.cpuProfile, tt.wantCPU)
			}
			if flags.memProfile != tt.wantMem {
				t.Errorf("memProfile = %q, want %q", flags.memProfile, tt.wantMem)
			}
			if flags.convert != tt.wantConv {
				t.Errorf("convert = %q, want %q", flags.convert, tt.wantConv)
			}
			if flags.watchConfig != tt.wantWatch {
				t.Errorf("watchConfig = %v, want %v", flags.watchConfig, tt.wantWatch)
			}
		})
	}
}

func TestRunWithArgsVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := runWithArgs([]string{"-v"}, &stdout, &stderr)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	output := stdout.String()
	if !strings.Contains(output, "conky-go version") {
		t.Errorf("expected version output, got: %s", output)
	}
	if !strings.Contains(output, Version) {
		t.Errorf("expected version %s in output, got: %s", Version, output)
	}
}

func TestRunWithArgsNoConfig(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := runWithArgs([]string{}, &stdout, &stderr)

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	errOutput := stderr.String()
	if !strings.Contains(errOutput, "No configuration file specified") {
		t.Errorf("expected error about missing config, got: %s", errOutput)
	}
}

func TestRunWithArgsNonexistentConfig(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := runWithArgs([]string{"-c", "/nonexistent/path/to/config"}, &stdout, &stderr)

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	errOutput := stderr.String()
	if !strings.Contains(errOutput, "Configuration file not found") {
		t.Errorf("expected 'not found' error, got: %s", errOutput)
	}
}

func TestRunWithArgsInvalidFlags(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := runWithArgs([]string{"-invalid-flag"}, &stdout, &stderr)

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	errOutput := stderr.String()
	if !strings.Contains(errOutput, "Error parsing flags") {
		t.Errorf("expected parsing error, got: %s", errOutput)
	}
}

func TestRunWithArgsConvert(t *testing.T) {
	// Create a temporary legacy config file
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

	var stdout, stderr bytes.Buffer
	exitCode := runWithArgs([]string{"-convert", tmpFile}, &stdout, &stderr)

	if exitCode != 0 {
		t.Errorf("runWithArgs returned non-zero exit code: %d, stderr: %s", exitCode, stderr.String())
	}

	output := stdout.String()
	expectedContents := []string{
		"conky.config = {",
		"background = true",
		"font = 'Test Font'",
		"conky.text = [[",
		"Test line",
	}
	for _, expected := range expectedContents {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, got: %s", expected, output)
		}
	}
}

func TestRunWithArgsConvertNonexistent(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := runWithArgs([]string{"-convert", "/nonexistent/config"}, &stdout, &stderr)

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	errOutput := stderr.String()
	if !strings.Contains(errOutput, "Configuration file not found") {
		t.Errorf("expected 'not found' error, got: %s", errOutput)
	}
}

func TestRunWithArgsConvertInvalidContent(t *testing.T) {
	// Create a temporary file with invalid content
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "invalid.conkyrc")
	content := `own_window_type invalid_type`
	if err := os.WriteFile(tmpFile, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	var stdout, stderr bytes.Buffer
	exitCode := runWithArgs([]string{"-convert", tmpFile}, &stdout, &stderr)

	if exitCode != 1 {
		t.Errorf("expected exit code 1 for invalid content, got: %d", exitCode)
	}
}

// TestRunConvert tests the legacy runConvert function for backwards compatibility.
func TestRunConvert(t *testing.T) {
	// Create a temporary legacy config file
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

	// Save and restore stdout
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	os.Stdout = w

	exitCode := runConvert(tmpFile)

	w.Close()
	os.Stdout = oldStdout

	if exitCode != 0 {
		t.Errorf("runConvert returned non-zero exit code: %d", exitCode)
	}

	// Read captured output
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("Failed to read captured output: %v", err)
	}
	output := buf.String()

	expectedContents := []string{
		"conky.config = {",
		"background = true",
		"font = 'Test Font'",
	}
	for _, expected := range expectedContents {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, got: %s", expected, output)
		}
	}
}

func TestRunConvertNonexistentFile(t *testing.T) {
	exitCode := runConvert("/nonexistent/path/to/config")
	if exitCode != 1 {
		t.Errorf("runConvert should return 1 for nonexistent file, got: %d", exitCode)
	}
}

func TestRunConvertInvalidContent(t *testing.T) {
	// Create a temporary file with invalid content
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "invalid.conkyrc")
	content := `own_window_type invalid_type`
	if err := os.WriteFile(tmpFile, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	exitCode := runConvert(tmpFile)
	if exitCode != 1 {
		t.Errorf("runConvert should return 1 for invalid content, got: %d", exitCode)
	}
}

func TestRunConvertWithWriterAccessError(t *testing.T) {
	// Create a directory instead of a file to trigger a non-NotExist error
	tmpDir := t.TempDir()
	dirPath := filepath.Join(tmpDir, "testdir")
	if err := os.Mkdir(dirPath, 0o755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	// Make directory unreadable to trigger access error (only works on Unix)
	if err := os.Chmod(dirPath, 0o000); err != nil {
		t.Skipf("Cannot change permissions: %v", err)
	}
	defer os.Chmod(dirPath, 0o755) //nolint:errcheck // cleanup

	// Try to convert a file inside the unreadable directory
	filePath := filepath.Join(dirPath, "test.conkyrc")
	var stdout, stderr bytes.Buffer
	exitCode := runConvertWithWriter(filePath, &stdout, &stderr)

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}
