package monitor

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestUptimeReaderWithMockFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create mock /proc/uptime
	// Format: uptime_seconds idle_seconds
	uptimeContent := "12345.67 23456.78\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "uptime"), []byte(uptimeContent), 0o644); err != nil {
		t.Fatalf("failed to write mock uptime: %v", err)
	}

	reader := &uptimeReader{
		procUptimePath: filepath.Join(tmpDir, "uptime"),
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	if stats.Seconds != 12345.67 {
		t.Errorf("Seconds = %v, want 12345.67", stats.Seconds)
	}

	if stats.IdleSeconds != 23456.78 {
		t.Errorf("IdleSeconds = %v, want 23456.78", stats.IdleSeconds)
	}

	expectedDuration := time.Duration(12345.67 * float64(time.Second))
	if stats.Duration != expectedDuration {
		t.Errorf("Duration = %v, want %v", stats.Duration, expectedDuration)
	}
}

func TestUptimeReaderMissingFile(t *testing.T) {
	reader := &uptimeReader{
		procUptimePath: "/nonexistent/uptime",
	}

	_, err := reader.ReadStats()
	if err == nil {
		t.Error("ReadStats() should return error for missing file")
	}
}

func TestUptimeReaderMalformedFile(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "too few fields",
			content: "12345.67\n",
		},
		{
			name:    "invalid uptime value",
			content: "abc 23456.78\n",
		},
		{
			name:    "invalid idle value",
			content: "12345.67 def\n",
		},
		{
			name:    "empty file",
			content: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uptimePath := filepath.Join(tmpDir, "uptime_"+tt.name)
			if err := os.WriteFile(uptimePath, []byte(tt.content), 0o644); err != nil {
				t.Fatalf("failed to write mock uptime: %v", err)
			}

			reader := &uptimeReader{
				procUptimePath: uptimePath,
			}

			_, err := reader.ReadStats()
			if err == nil {
				t.Errorf("ReadStats() should return error for: %s", tt.name)
			}
		})
	}
}

func TestUptimeReaderWithWhitespace(t *testing.T) {
	tmpDir := t.TempDir()

	// Test with extra whitespace
	uptimeContent := "  12345.67   23456.78  \n"
	if err := os.WriteFile(filepath.Join(tmpDir, "uptime"), []byte(uptimeContent), 0o644); err != nil {
		t.Fatalf("failed to write mock uptime: %v", err)
	}

	reader := &uptimeReader{
		procUptimePath: filepath.Join(tmpDir, "uptime"),
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	if stats.Seconds != 12345.67 {
		t.Errorf("Seconds = %v, want 12345.67", stats.Seconds)
	}
}
