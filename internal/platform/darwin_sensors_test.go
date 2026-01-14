//go:build darwin
// +build darwin

package platform

import (
	"testing"
)

func TestDarwinSensorProvider_Temperatures(t *testing.T) {
	provider := newDarwinSensorProvider()

	temps, err := provider.Temperatures()
	if err != nil {
		t.Fatalf("Temperatures() failed: %v", err)
	}

	// Temperature sensors may not be available without root privileges
	// So we just check that it doesn't crash and returns valid data if available
	for _, temp := range temps {
		if temp.Value < -273.15 {
			t.Errorf("Temperature %s is below absolute zero: %f", temp.Name, temp.Value)
		}

		if temp.Value > 200 {
			t.Errorf("Temperature %s is unreasonably high: %f", temp.Name, temp.Value)
		}

		if temp.Unit != "°C" {
			t.Errorf("Expected temperature unit to be °C, got %s", temp.Unit)
		}
	}

	if len(temps) > 0 {
		t.Logf("Found %d temperature sensors", len(temps))
	} else {
		t.Log("No temperature sensors found (may require root privileges)")
	}
}

func TestDarwinSensorProvider_Fans(t *testing.T) {
	provider := newDarwinSensorProvider()

	fans, err := provider.Fans()
	if err != nil {
		t.Fatalf("Fans() failed: %v", err)
	}

	// Fan sensors may not be available without root privileges
	// So we just check that it doesn't crash and returns valid data if available
	for _, fan := range fans {
		if fan.Value < 0 {
			t.Errorf("Fan speed %s should be non-negative: %f", fan.Name, fan.Value)
		}

		if fan.Value > 10000 {
			t.Errorf("Fan speed %s is unreasonably high: %f RPM", fan.Name, fan.Value)
		}

		if fan.Unit != "RPM" {
			t.Errorf("Expected fan unit to be RPM, got %s", fan.Unit)
		}
	}

	if len(fans) > 0 {
		t.Logf("Found %d fan sensors", len(fans))
	} else {
		t.Log("No fan sensors found (may require root privileges)")
	}
}

func TestParseTemperatureLine(t *testing.T) {
	provider := newDarwinSensorProvider()

	tests := []struct {
		input    string
		expected float64
		wantNil  bool
	}{
		{"CPU die temperature: 45.00 C", 45.00, false},
		{"GPU temperature: 60.5 C", 60.5, false},
		{"Invalid line", 0, true},
		{"Temperature without value:", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			reading := provider.parseTemperatureLine(tt.input)
			if tt.wantNil {
				if reading != nil {
					t.Errorf("Expected nil reading for input %s", tt.input)
				}
			} else {
				if reading == nil {
					t.Errorf("Expected non-nil reading for input %s", tt.input)
					return
				}
				if reading.Value != tt.expected {
					t.Errorf("Expected value %f, got %f", tt.expected, reading.Value)
				}
			}
		})
	}
}

func TestParseFanLine(t *testing.T) {
	provider := newDarwinSensorProvider()

	tests := []struct {
		input    string
		expected float64
		wantNil  bool
	}{
		{"Fan: 2000 rpm", 2000, false},
		{"Left fan: 1500 rpm", 1500, false},
		{"Invalid line", 0, true},
		{"Fan without value:", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			reading := provider.parseFanLine(tt.input)
			if tt.wantNil {
				if reading != nil {
					t.Errorf("Expected nil reading for input %s", tt.input)
				}
			} else {
				if reading == nil {
					t.Errorf("Expected non-nil reading for input %s", tt.input)
					return
				}
				if reading.Value != tt.expected {
					t.Errorf("Expected value %f, got %f", tt.expected, reading.Value)
				}
			}
		})
	}
}
