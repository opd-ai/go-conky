package main

import (
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

	// Capture stdout by redirecting it
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	exitCode := runConvert(tmpFile)

	w.Close()
	os.Stdout = oldStdout

	if exitCode != 0 {
		t.Errorf("runConvert returned non-zero exit code: %d", exitCode)
	}

	// Read captured output
	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Verify the output contains expected Lua content
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
