//go:build darwin
// +build darwin

package platform

import (
	"testing"
)

func TestDarwinNetworkProvider_Interfaces(t *testing.T) {
	provider := newDarwinNetworkProvider()

	interfaces, err := provider.Interfaces()
	if err != nil {
		t.Fatalf("Interfaces() failed: %v", err)
	}

	// Should have at least one non-loopback interface
	// (even if just a virtual interface)
	if len(interfaces) == 0 {
		t.Log("Warning: No network interfaces found (expected at least one)")
	}

	// Check for duplicates
	seen := make(map[string]bool)
	for _, name := range interfaces {
		if seen[name] {
			t.Errorf("Duplicate interface name: %s", name)
		}
		seen[name] = true
	}
}

func TestDarwinNetworkProvider_Stats(t *testing.T) {
	provider := newDarwinNetworkProvider()

	interfaces, err := provider.Interfaces()
	if err != nil {
		t.Fatalf("Interfaces() failed: %v", err)
	}

	if len(interfaces) == 0 {
		t.Skip("No network interfaces found")
	}

	// Test getting stats for the first interface
	ifaceName := interfaces[0]
	stats, err := provider.Stats(ifaceName)
	if err != nil {
		t.Fatalf("Stats(%s) failed: %v", ifaceName, err)
	}

	// Basic sanity checks - values should be non-negative
	if stats.BytesRecv < 0 {
		t.Errorf("BytesRecv should be non-negative, got %d", stats.BytesRecv)
	}

	if stats.BytesSent < 0 {
		t.Errorf("BytesSent should be non-negative, got %d", stats.BytesSent)
	}

	if stats.PacketsRecv < 0 {
		t.Errorf("PacketsRecv should be non-negative, got %d", stats.PacketsRecv)
	}

	if stats.PacketsSent < 0 {
		t.Errorf("PacketsSent should be non-negative, got %d", stats.PacketsSent)
	}
}

func TestDarwinNetworkProvider_Stats_InvalidInterface(t *testing.T) {
	provider := newDarwinNetworkProvider()

	_, err := provider.Stats("invalid_interface_xyz123")
	if err == nil {
		t.Error("Expected error for invalid interface name")
	}
}

func TestDarwinNetworkProvider_AllStats(t *testing.T) {
	provider := newDarwinNetworkProvider()

	allStats, err := provider.AllStats()
	if err != nil {
		t.Fatalf("AllStats() failed: %v", err)
	}

	// Should have at least one interface
	if len(allStats) == 0 {
		t.Log("Warning: No network interface stats found")
	}

	// Verify all stats are valid
	for ifaceName, stats := range allStats {
		if stats == nil {
			t.Errorf("Stats for %s should not be nil", ifaceName)
			continue
		}

		if stats.BytesRecv < 0 {
			t.Errorf("%s: BytesRecv should be non-negative", ifaceName)
		}

		if stats.BytesSent < 0 {
			t.Errorf("%s: BytesSent should be non-negative", ifaceName)
		}
	}
}
