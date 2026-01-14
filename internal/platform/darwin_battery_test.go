// +build darwin

package platform

import (
	"testing"
)

func TestDarwinBatteryProvider_Count(t *testing.T) {
	provider := newDarwinBatteryProvider()

	count := provider.Count()
	
	// macOS systems have either 0 (desktop) or 1 (laptop) battery
	if count < 0 || count > 1 {
		t.Errorf("Expected battery count to be 0 or 1, got %d", count)
	}
}

func TestDarwinBatteryProvider_Stats(t *testing.T) {
	provider := newDarwinBatteryProvider()

	count := provider.Count()
	if count == 0 {
		t.Skip("No battery found (likely a desktop system)")
	}

	stats, err := provider.Stats(0)
	if err != nil {
		t.Fatalf("Stats(0) failed: %v", err)
	}

	// Percentage should be between 0 and 100
	if stats.Percent < 0 || stats.Percent > 100 {
		t.Errorf("Battery percentage should be between 0 and 100, got %f", stats.Percent)
	}

	// TimeRemaining can be 0 if not available or fully charged
	if stats.TimeRemaining < 0 {
		t.Errorf("TimeRemaining should be non-negative, got %v", stats.TimeRemaining)
	}

	t.Logf("Battery stats: %.1f%%, charging: %v, time remaining: %v", 
		stats.Percent, stats.Charging, stats.TimeRemaining)
}

func TestDarwinBatteryProvider_Stats_InvalidIndex(t *testing.T) {
	provider := newDarwinBatteryProvider()

	_, err := provider.Stats(1)
	if err == nil {
		t.Error("Expected error for invalid battery index")
	}

	_, err = provider.Stats(-1)
	if err == nil {
		t.Error("Expected error for negative battery index")
	}
}

func TestParseTimeRemaining(t *testing.T) {
	tests := []struct {
		input    string
		expected int // in minutes
		wantErr  bool
	}{
		{"5:23", 5*60 + 23, false},
		{"0:45", 45, false},
		{"10:00", 10 * 60, false},
		{"invalid", 0, true},
		{"5", 0, true},
		{"5:60", 5*60 + 60, false}, // Invalid but parsed
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			duration, err := parseTimeRemaining(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for input %s", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for input %s: %v", tt.input, err)
				}
				expectedDuration := tt.expected * 60 // convert to seconds
				if int(duration.Seconds()) != expectedDuration {
					t.Errorf("Expected %d seconds, got %d", expectedDuration, int(duration.Seconds()))
				}
			}
		})
	}
}
