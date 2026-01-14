// +build windows

package platform

import (
	"testing"
)

func TestWindowsBatteryProvider_Count(t *testing.T) {
	provider := newWindowsBatteryProvider()

	count := provider.Count()

	// Count should be 0 or 1 (Windows API reports at most 1 battery)
	if count < 0 || count > 1 {
		t.Errorf("Count = %v, want 0 or 1", count)
	}
}

func TestWindowsBatteryProvider_Stats(t *testing.T) {
	provider := newWindowsBatteryProvider()

	count := provider.Count()
	if count == 0 {
		t.Skip("No battery present for testing")
	}

	stats, err := provider.Stats(0)
	if err != nil {
		t.Fatalf("Stats(0) error = %v", err)
	}

	if stats.Percent < 0 || stats.Percent > 100 {
		t.Errorf("Percent = %v, want 0-100", stats.Percent)
	}

	// TimeRemaining can be 0 if unknown
	if stats.TimeRemaining < 0 {
		t.Errorf("TimeRemaining = %v, want >= 0", stats.TimeRemaining)
	}
}

func TestWindowsBatteryProvider_StatsInvalidIndex(t *testing.T) {
	provider := newWindowsBatteryProvider()

	// Index 1 should always be invalid (Windows only reports 1 battery max)
	_, err := provider.Stats(1)
	if err == nil {
		t.Error("Stats(1) should return error for invalid index")
	}
}
