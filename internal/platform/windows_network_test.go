// +build windows

package platform

import (
	"testing"
)

func TestWindowsNetworkProvider_Interfaces(t *testing.T) {
	provider := newWindowsNetworkProvider()

	interfaces, err := provider.Interfaces()
	if err != nil {
		t.Fatalf("Interfaces() error = %v", err)
	}

	// Most systems have at least one network interface
	if len(interfaces) == 0 {
		t.Error("Interfaces() returned empty slice, expected at least one interface")
	}
}

func TestWindowsNetworkProvider_AllStats(t *testing.T) {
	provider := newWindowsNetworkProvider()

	stats, err := provider.AllStats()
	if err != nil {
		t.Fatalf("AllStats() error = %v", err)
	}

	if len(stats) == 0 {
		t.Error("AllStats() returned empty map, expected at least one interface")
	}

	// Validate each interface's stats
	for name, stat := range stats {
		if stat == nil {
			t.Errorf("Stats for interface %s is nil", name)
			continue
		}

		// Stats should be non-negative
		if stat.BytesRecv < 0 {
			t.Errorf("Interface %s: BytesRecv < 0", name)
		}
		if stat.BytesSent < 0 {
			t.Errorf("Interface %s: BytesSent < 0", name)
		}
	}
}

func TestWindowsNetworkProvider_Stats(t *testing.T) {
	provider := newWindowsNetworkProvider()

	// Get list of interfaces
	interfaces, err := provider.Interfaces()
	if err != nil {
		t.Fatalf("Interfaces() error = %v", err)
	}

	if len(interfaces) == 0 {
		t.Skip("No network interfaces available for testing")
	}

	// Test stats for first interface
	stats, err := provider.Stats(interfaces[0])
	if err != nil {
		t.Fatalf("Stats(%s) error = %v", interfaces[0], err)
	}

	if stats == nil {
		t.Fatal("Stats returned nil")
	}

	// Test with non-existent interface
	_, err = provider.Stats("NonExistentInterface12345")
	if err == nil {
		t.Error("Stats() with invalid interface should return error")
	}
}
