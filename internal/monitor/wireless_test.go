package monitor

import (
	"os"
	"path/filepath"
	"testing"
)

// TestWirelessReader tests the wireless reader functionality.
func TestWirelessReader(t *testing.T) {
	t.Run("NewWirelessReader", func(t *testing.T) {
		reader := newWirelessReader()
		if reader == nil {
			t.Fatal("expected non-nil reader")
		}
		if reader.procWirelessPath != "/proc/net/wireless" {
			t.Errorf("unexpected procWirelessPath: %s", reader.procWirelessPath)
		}
		if reader.sysNetPath != "/sys/class/net" {
			t.Errorf("unexpected sysNetPath: %s", reader.sysNetPath)
		}
	})
}

// TestParseProcWirelessLine tests parsing of /proc/net/wireless lines.
func TestParseProcWirelessLine(t *testing.T) {
	reader := newWirelessReader()

	tests := []struct {
		name            string
		line            string
		wantIface       string
		wantLinkQuality int
		wantSignal      int
		wantNoise       int
		wantErr         bool
	}{
		{
			name:            "standard line with dots",
			line:            "wlan0: 0000   70.  -40.  -95.        0      0      0     0      0        0",
			wantIface:       "wlan0",
			wantLinkQuality: 70,
			wantSignal:      -40,
			wantNoise:       -95,
			wantErr:         false,
		},
		{
			name:            "line without dots",
			line:            "wlan0: 0000   70  -40  -95        0      0      0     0      0        0",
			wantIface:       "wlan0",
			wantLinkQuality: 70,
			wantSignal:      -40,
			wantNoise:       -95,
			wantErr:         false,
		},
		{
			name:            "high quality",
			line:            "wlp3s0: 0000   100.  -30.  -90.        0      0      0     0      0        0",
			wantIface:       "wlp3s0",
			wantLinkQuality: 100,
			wantSignal:      -30,
			wantNoise:       -90,
			wantErr:         false,
		},
		{
			name:            "low quality",
			line:            "wlan1: 0000   10.  -80.  -95.        0      0      0     0      0        0",
			wantIface:       "wlan1",
			wantLinkQuality: 10,
			wantSignal:      -80,
			wantNoise:       -95,
			wantErr:         false,
		},
		{
			name:    "too few fields",
			line:    "wlan0: 0000",
			wantErr: true,
		},
		{
			name:    "empty line",
			line:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, iface, err := reader.parseProcWirelessLine(tt.line)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if iface != tt.wantIface {
				t.Errorf("interface = %q, want %q", iface, tt.wantIface)
			}
			if info.LinkQuality != tt.wantLinkQuality {
				t.Errorf("LinkQuality = %d, want %d", info.LinkQuality, tt.wantLinkQuality)
			}
			if info.SignalLevel != tt.wantSignal {
				t.Errorf("SignalLevel = %d, want %d", info.SignalLevel, tt.wantSignal)
			}
			if info.NoiseLevel != tt.wantNoise {
				t.Errorf("NoiseLevel = %d, want %d", info.NoiseLevel, tt.wantNoise)
			}
			if !info.IsWireless {
				t.Error("IsWireless should be true")
			}
		})
	}
}

// TestWirelessReaderWithMockFile tests reading from a mock /proc/net/wireless file.
func TestWirelessReaderWithMockFile(t *testing.T) {
	// Create a temp directory for mock files
	tmpDir := t.TempDir()

	// Create mock /proc/net/wireless content
	procWirelessContent := `Inter-| sta-|   Quality        |   Discarded packets               | Missed | WE
 face | tus | link level noise |  nwid  crypt   frag  retry   misc | beacon | 22
wlan0: 0000   70.  -40.  -95.        0      0      0     0      0        0
wlan1: 0000   50.  -55.  -92.        0      0      0     0      0        0
`
	procPath := filepath.Join(tmpDir, "wireless")
	if err := os.WriteFile(procPath, []byte(procWirelessContent), 0o644); err != nil {
		t.Fatalf("failed to write mock file: %v", err)
	}

	// Create reader with mock path
	reader := newWirelessReader()
	reader.procWirelessPath = procPath
	reader.sysNetPath = tmpDir // For enhanceWithSysInfo

	// Read stats
	stats, err := reader.ReadWirelessStats()
	if err != nil {
		t.Fatalf("ReadWirelessStats failed: %v", err)
	}

	// Verify we got both interfaces
	if len(stats) != 2 {
		t.Errorf("expected 2 interfaces, got %d", len(stats))
	}

	// Check wlan0
	wlan0, ok := stats["wlan0"]
	if !ok {
		t.Error("wlan0 not found in stats")
	} else {
		if wlan0.LinkQuality != 70 {
			t.Errorf("wlan0 LinkQuality = %d, want 70", wlan0.LinkQuality)
		}
		if wlan0.SignalLevel != -40 {
			t.Errorf("wlan0 SignalLevel = %d, want -40", wlan0.SignalLevel)
		}
		if !wlan0.IsWireless {
			t.Error("wlan0 IsWireless should be true")
		}
	}

	// Check wlan1
	wlan1, ok := stats["wlan1"]
	if !ok {
		t.Error("wlan1 not found in stats")
	} else {
		if wlan1.LinkQuality != 50 {
			t.Errorf("wlan1 LinkQuality = %d, want 50", wlan1.LinkQuality)
		}
	}
}

// TestWirelessReaderNonexistentFile tests handling of missing /proc/net/wireless.
func TestWirelessReaderNonexistentFile(t *testing.T) {
	reader := newWirelessReader()
	reader.procWirelessPath = "/nonexistent/path/wireless"

	stats, err := reader.ReadWirelessStats()
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if len(stats) != 0 {
		t.Errorf("expected empty stats, got %d interfaces", len(stats))
	}
}

// TestWirelessInfoLinkQualityPercent tests the percentage calculation.
func TestWirelessInfoLinkQualityPercent(t *testing.T) {
	tests := []struct {
		name     string
		info     WirelessInfo
		expected int
	}{
		{
			name:     "full quality",
			info:     WirelessInfo{LinkQuality: 100, LinkQualityMax: 100},
			expected: 100,
		},
		{
			name:     "half quality",
			info:     WirelessInfo{LinkQuality: 50, LinkQualityMax: 100},
			expected: 50,
		},
		{
			name:     "zero quality",
			info:     WirelessInfo{LinkQuality: 0, LinkQualityMax: 100},
			expected: 0,
		},
		{
			name:     "quality over max (capped)",
			info:     WirelessInfo{LinkQuality: 150, LinkQualityMax: 100},
			expected: 100,
		},
		{
			name:     "different max",
			info:     WirelessInfo{LinkQuality: 35, LinkQualityMax: 70},
			expected: 50,
		},
		{
			name:     "zero max (avoid division by zero)",
			info:     WirelessInfo{LinkQuality: 50, LinkQualityMax: 0},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.info.LinkQualityPercent()
			if result != tt.expected {
				t.Errorf("LinkQualityPercent() = %d, want %d", result, tt.expected)
			}
		})
	}
}

// TestWirelessInfoBitRateString tests the bit rate formatting.
func TestWirelessInfoBitRateString(t *testing.T) {
	tests := []struct {
		name     string
		info     WirelessInfo
		expected string
	}{
		{
			name:     "zero rate",
			info:     WirelessInfo{BitRate: 0},
			expected: "0Mb/s",
		},
		{
			name:     "typical wifi",
			info:     WirelessInfo{BitRate: 54},
			expected: "54Mb/s",
		},
		{
			name:     "modern wifi",
			info:     WirelessInfo{BitRate: 300},
			expected: "300Mb/s",
		},
		{
			name:     "gigabit",
			info:     WirelessInfo{BitRate: 1000},
			expected: "1.0Gb/s",
		},
		{
			name:     "wifi 6",
			info:     WirelessInfo{BitRate: 2400},
			expected: "2.4Gb/s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.info.BitRateString()
			if result != tt.expected {
				t.Errorf("BitRateString() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestGetWirelessInfo tests the cached lookup method.
func TestGetWirelessInfo(t *testing.T) {
	tmpDir := t.TempDir()

	// Create mock /proc/net/wireless content
	procWirelessContent := `Inter-| sta-|   Quality        |   Discarded packets               | Missed | WE
 face | tus | link level noise |  nwid  crypt   frag  retry   misc | beacon | 22
wlan0: 0000   70.  -40.  -95.        0      0      0     0      0        0
`
	procPath := filepath.Join(tmpDir, "wireless")
	if err := os.WriteFile(procPath, []byte(procWirelessContent), 0o644); err != nil {
		t.Fatalf("failed to write mock file: %v", err)
	}

	reader := newWirelessReader()
	reader.procWirelessPath = procPath
	reader.sysNetPath = tmpDir

	// First call to populate cache
	_, err := reader.ReadWirelessStats()
	if err != nil {
		t.Fatalf("ReadWirelessStats failed: %v", err)
	}

	// Test cache lookup
	info, ok := reader.GetWirelessInfo("wlan0")
	if !ok {
		t.Error("expected wlan0 to be found in cache")
	}
	if info.LinkQuality != 70 {
		t.Errorf("cached LinkQuality = %d, want 70", info.LinkQuality)
	}

	// Test lookup for non-existent interface
	_, ok = reader.GetWirelessInfo("eth0")
	if ok {
		t.Error("expected eth0 to not be found in cache")
	}
}

// TestIsWirelessInterface tests the wireless interface detection.
func TestIsWirelessInterface(t *testing.T) {
	tmpDir := t.TempDir()

	// Create mock /proc/net/wireless
	procWirelessContent := `Inter-| sta-|   Quality        |   Discarded packets               | Missed | WE
 face | tus | link level noise |  nwid  crypt   frag  retry   misc | beacon | 22
wlan0: 0000   70.  -40.  -95.        0      0      0     0      0        0
`
	procPath := filepath.Join(tmpDir, "wireless")
	if err := os.WriteFile(procPath, []byte(procWirelessContent), 0o644); err != nil {
		t.Fatalf("failed to write mock file: %v", err)
	}

	// Create mock /sys/class/net/<iface>/wireless directory
	wirelessDir := filepath.Join(tmpDir, "wlan0", "wireless")
	if err := os.MkdirAll(wirelessDir, 0o755); err != nil {
		t.Fatalf("failed to create mock wireless dir: %v", err)
	}

	reader := newWirelessReader()
	reader.procWirelessPath = procPath
	reader.sysNetPath = tmpDir

	// Populate cache
	_, err := reader.ReadWirelessStats()
	if err != nil {
		t.Fatalf("ReadWirelessStats failed: %v", err)
	}

	// Test detection methods
	if !reader.IsWirelessInterface("wlan0") {
		t.Error("expected wlan0 to be detected as wireless")
	}

	if reader.IsWirelessInterface("eth0") {
		t.Error("expected eth0 to not be detected as wireless")
	}
}
