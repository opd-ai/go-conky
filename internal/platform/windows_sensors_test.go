// +build windows

package platform

import (
	"testing"
)

func TestWindowsSensorProvider_Temperatures(t *testing.T) {
	provider := newWindowsSensorProvider()

	temps, err := provider.Temperatures()
	if err != nil {
		t.Fatalf("Temperatures() error = %v", err)
	}

	// Should return empty slice for now (not implemented)
	if temps == nil {
		t.Error("Temperatures() returned nil, expected empty slice")
	}
}

func TestWindowsSensorProvider_Fans(t *testing.T) {
	provider := newWindowsSensorProvider()

	fans, err := provider.Fans()
	if err != nil {
		t.Fatalf("Fans() error = %v", err)
	}

	// Should return empty slice for now (not implemented)
	if fans == nil {
		t.Error("Fans() returned nil, expected empty slice")
	}
}
