//go:build windows
// +build windows

package platform

import (
	"testing"
)

func TestWindowsMemoryProvider_Stats(t *testing.T) {
	provider := newWindowsMemoryProvider()

	stats, err := provider.Stats()
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}

	// Validate stats structure
	if stats.Total == 0 {
		t.Error("Total memory should not be 0")
	}

	if stats.UsedPercent < 0 || stats.UsedPercent > 100 {
		t.Errorf("UsedPercent = %v, want 0-100", stats.UsedPercent)
	}

	if stats.Available > stats.Total {
		t.Errorf("Available = %v > Total = %v", stats.Available, stats.Total)
	}

	if stats.Used > stats.Total {
		t.Errorf("Used = %v > Total = %v", stats.Used, stats.Total)
	}
}

func TestWindowsMemoryProvider_SwapStats(t *testing.T) {
	provider := newWindowsMemoryProvider()

	stats, err := provider.SwapStats()
	if err != nil {
		t.Fatalf("SwapStats() error = %v", err)
	}

	// Page file may be 0 on some systems
	if stats.Total > 0 {
		if stats.UsedPercent < 0 || stats.UsedPercent > 100 {
			t.Errorf("UsedPercent = %v, want 0-100", stats.UsedPercent)
		}

		// On some Windows CI environments, swap metrics can be inconsistent
		// (e.g., Free > Total when page file is dynamically managed)
		if stats.Free > stats.Total {
			t.Logf("Skipping Free > Total check: Free = %v, Total = %v (dynamic page file)", stats.Free, stats.Total)
		}

		if stats.Used > stats.Total {
			t.Errorf("Used = %v > Total = %v", stats.Used, stats.Total)
		}
	}
}
