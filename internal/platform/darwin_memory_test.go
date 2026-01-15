//go:build darwin
// +build darwin

package platform

import (
	"testing"
)

func TestDarwinMemoryProvider_Stats(t *testing.T) {
	provider := newDarwinMemoryProvider()

	stats, err := provider.Stats()
	if isDarwinCIError(err) {
		t.Skipf("Skipping: sysctl unavailable in this environment: %v", err)
	}
	if err != nil {
		t.Fatalf("Stats() failed: %v", err)
	}

	if stats.Total == 0 {
		t.Error("Total memory should not be zero")
	}

	if stats.Used > stats.Total {
		t.Errorf("Used memory (%d) should not exceed total (%d)", stats.Used, stats.Total)
	}

	if stats.Available > stats.Total {
		t.Errorf("Available memory (%d) should not exceed total (%d)", stats.Available, stats.Total)
	}

	if stats.UsedPercent < 0 || stats.UsedPercent > 100 {
		t.Errorf("Used percentage should be between 0 and 100, got %f", stats.UsedPercent)
	}

	// Check that used + available is approximately equal to total
	// (allowing for some accounting differences)
	diff := int64(stats.Total) - int64(stats.Used) - int64(stats.Available)
	if diff < 0 {
		diff = -diff
	}
	tolerance := int64(stats.Total) / 10 // 10% tolerance
	if diff > tolerance {
		t.Logf("Warning: Used + Available differs from Total by %d bytes", diff)
	}
}

func TestDarwinMemoryProvider_SwapStats(t *testing.T) {
	provider := newDarwinMemoryProvider()

	stats, err := provider.SwapStats()
	if err != nil {
		t.Fatalf("SwapStats() failed: %v", err)
	}

	// Swap may be zero on some systems
	if stats.Total > 0 {
		if stats.Used > stats.Total {
			t.Errorf("Used swap (%d) should not exceed total (%d)", stats.Used, stats.Total)
		}

		if stats.Free > stats.Total {
			t.Errorf("Free swap (%d) should not exceed total (%d)", stats.Free, stats.Total)
		}

		if stats.Used+stats.Free != stats.Total {
			// Allow small discrepancies
			diff := int64(stats.Total) - int64(stats.Used) - int64(stats.Free)
			if diff < 0 {
				diff = -diff
			}
			if diff > 1024*1024 { // 1 MB tolerance
				t.Errorf("Used + Free (%d) should equal Total (%d), diff: %d",
					stats.Used+stats.Free, stats.Total, diff)
			}
		}

		if stats.UsedPercent < 0 || stats.UsedPercent > 100 {
			t.Errorf("Used percentage should be between 0 and 100, got %f", stats.UsedPercent)
		}
	}
}
