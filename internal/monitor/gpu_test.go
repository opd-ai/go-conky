package monitor

import (
	"testing"
)

func TestGPUReader(t *testing.T) {
	t.Run("NewGPUReader", func(t *testing.T) {
		r := newGPUReader()
		if r == nil {
			t.Fatal("expected non-nil reader")
		}
		if r.cacheDuration == 0 {
			t.Error("expected non-zero cache duration")
		}
	})
}

func TestParseNvidiaSmiOutput(t *testing.T) {
	r := newGPUReader()

	tests := []struct {
		name      string
		output    string
		wantName  string
		wantTemp  int
		wantUtil  int
		wantAvail bool
	}{
		{
			name:      "valid output",
			output:    "NVIDIA GeForce RTX 3080, 535.154.05, 45, 30, 25, 2048, 10240, 8192, 55, 150.5, 320.0",
			wantName:  "NVIDIA GeForce RTX 3080",
			wantTemp:  45,
			wantUtil:  30,
			wantAvail: true,
		},
		{
			name:      "empty output",
			output:    "",
			wantAvail: false,
		},
		{
			name:      "insufficient fields",
			output:    "NVIDIA, 535",
			wantAvail: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats, _ := r.parseNvidiaSmiOutput(tt.output)
			if stats.Available != tt.wantAvail {
				t.Errorf("Available = %v, want %v", stats.Available, tt.wantAvail)
			}
			if tt.wantAvail {
				if stats.Name != tt.wantName {
					t.Errorf("Name = %q, want %q", stats.Name, tt.wantName)
				}
				if stats.Temperature != tt.wantTemp {
					t.Errorf("Temperature = %d, want %d", stats.Temperature, tt.wantTemp)
				}
				if stats.UtilGPU != tt.wantUtil {
					t.Errorf("UtilGPU = %d, want %d", stats.UtilGPU, tt.wantUtil)
				}
			}
		})
	}
}

func TestGPUStatsGetField(t *testing.T) {
	stats := GPUStats{
		Name:        "RTX 3080",
		Temperature: 65,
		UtilGPU:     45,
		UtilMem:     30,
		MemUsed:     2 * 1024 * 1024 * 1024,
		MemTotal:    10 * 1024 * 1024 * 1024,
		FanSpeed:    50,
		PowerDraw:   200.5,
		Available:   true,
	}

	tests := []struct {
		field    string
		expected string
	}{
		{"gpuutil", "45%"},
		{"temp", "65°C"},
		{"name", "RTX 3080"},
		{"fan", "50%"},
		{"power", "200.5W"},
		{"memperc", "20.0%"},
	}

	for _, tt := range tests {
		result := stats.GetField(tt.field)
		if result != tt.expected {
			t.Errorf("GetField(%q) = %q, want %q", tt.field, result, tt.expected)
		}
	}

	// Test unavailable GPU
	unavail := GPUStats{Available: false}
	if unavail.GetField("temp") != "N/A" {
		t.Errorf("expected N/A for unavailable GPU")
	}
}
