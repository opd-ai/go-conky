//go:build linux && !android
// +build linux,!android

package platform

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLinuxNetworkProvider_Interfaces(t *testing.T) {
	// Create a temporary /proc/net/dev file
	tmpDir := t.TempDir()
	netDevPath := filepath.Join(tmpDir, "dev")

	content := `Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
    lo: 1234567       100    0    0    0     0          0         0  1234567       100    0    0    0     0       0          0
  eth0: 9876543210    5000    5    2    0     0          0         0  1234567890   4000    1    0    0     0       0          0
  wlan0: 5555555555    3000    0    0    0     0          0         0  4444444444   2500    0    0    0     0       0          0
`
	if err := os.WriteFile(netDevPath, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write net/dev file: %v", err)
	}

	provider := &linuxNetworkProvider{
		procNetDevPath: netDevPath,
	}

	interfaces, err := provider.Interfaces()
	if err != nil {
		t.Fatalf("Interfaces() failed: %v", err)
	}

	if len(interfaces) != 3 {
		t.Errorf("Interfaces() returned %d interfaces, want 3", len(interfaces))
	}

	// Check that all expected interfaces are present
	expectedInterfaces := map[string]bool{"lo": false, "eth0": false, "wlan0": false}
	for _, iface := range interfaces {
		if _, ok := expectedInterfaces[iface]; ok {
			expectedInterfaces[iface] = true
		}
	}

	for iface, found := range expectedInterfaces {
		if !found {
			t.Errorf("Interface %s not found in results", iface)
		}
	}
}

func TestLinuxNetworkProvider_Stats(t *testing.T) {
	// Create a temporary /proc/net/dev file
	tmpDir := t.TempDir()
	netDevPath := filepath.Join(tmpDir, "dev")

	content := `Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
    lo: 1234567       100    0    0    0     0          0         0  1234567       100    0    0    0     0       0          0
  eth0: 9876543210    5000    5    2    0     0          0         0  1234567890   4000    1    0    0     0       0          0
`
	if err := os.WriteFile(netDevPath, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write net/dev file: %v", err)
	}

	provider := &linuxNetworkProvider{
		procNetDevPath: netDevPath,
	}

	stats, err := provider.Stats("eth0")
	if err != nil {
		t.Fatalf("Stats('eth0') failed: %v", err)
	}

	if stats.BytesRecv != 9876543210 {
		t.Errorf("BytesRecv = %d, want 9876543210", stats.BytesRecv)
	}
	if stats.BytesSent != 1234567890 {
		t.Errorf("BytesSent = %d, want 1234567890", stats.BytesSent)
	}
	if stats.PacketsRecv != 5000 {
		t.Errorf("PacketsRecv = %d, want 5000", stats.PacketsRecv)
	}
	if stats.PacketsSent != 4000 {
		t.Errorf("PacketsSent = %d, want 4000", stats.PacketsSent)
	}
	if stats.ErrorsIn != 5 {
		t.Errorf("ErrorsIn = %d, want 5", stats.ErrorsIn)
	}
	if stats.ErrorsOut != 1 {
		t.Errorf("ErrorsOut = %d, want 1", stats.ErrorsOut)
	}
	if stats.DropIn != 2 {
		t.Errorf("DropIn = %d, want 2", stats.DropIn)
	}
	if stats.DropOut != 0 {
		t.Errorf("DropOut = %d, want 0", stats.DropOut)
	}
}

func TestLinuxNetworkProvider_Stats_NotFound(t *testing.T) {
	// Create a temporary /proc/net/dev file
	tmpDir := t.TempDir()
	netDevPath := filepath.Join(tmpDir, "dev")

	content := `Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
    lo: 1234567       100    0    0    0     0          0         0  1234567       100    0    0    0     0       0          0
`
	if err := os.WriteFile(netDevPath, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write net/dev file: %v", err)
	}

	provider := &linuxNetworkProvider{
		procNetDevPath: netDevPath,
	}

	_, err := provider.Stats("eth999")
	if err == nil {
		t.Error("Stats('eth999') should have failed for non-existent interface")
	}
}

func TestLinuxNetworkProvider_AllStats(t *testing.T) {
	// Create a temporary /proc/net/dev file
	tmpDir := t.TempDir()
	netDevPath := filepath.Join(tmpDir, "dev")

	content := `Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
    lo: 1234567       100    0    0    0     0          0         0  1234567       100    0    0    0     0       0          0
  eth0: 9876543210    5000    5    2    0     0          0         0  1234567890   4000    1    0    0     0       0          0
`
	if err := os.WriteFile(netDevPath, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write net/dev file: %v", err)
	}

	provider := &linuxNetworkProvider{
		procNetDevPath: netDevPath,
	}

	allStats, err := provider.AllStats()
	if err != nil {
		t.Fatalf("AllStats() failed: %v", err)
	}

	if len(allStats) != 2 {
		t.Errorf("AllStats() returned %d interfaces, want 2", len(allStats))
	}

	// Check lo interface
	loStats, ok := allStats["lo"]
	if !ok {
		t.Error("AllStats() missing 'lo' interface")
	} else if loStats.BytesRecv != 1234567 {
		t.Errorf("lo BytesRecv = %d, want 1234567", loStats.BytesRecv)
	}

	// Check eth0 interface
	eth0Stats, ok := allStats["eth0"]
	if !ok {
		t.Error("AllStats() missing 'eth0' interface")
	} else if eth0Stats.BytesRecv != 9876543210 {
		t.Errorf("eth0 BytesRecv = %d, want 9876543210", eth0Stats.BytesRecv)
	}
}

func TestLinuxNetworkProvider_ConcurrentAccess(t *testing.T) {
	// Create a temporary /proc/net/dev file
	tmpDir := t.TempDir()
	netDevPath := filepath.Join(tmpDir, "dev")

	content := `Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
    lo: 1234567       100    0    0    0     0          0         0  1234567       100    0    0    0     0       0          0
  eth0: 9876543210    5000    5    2    0     0          0         0  1234567890   4000    1    0    0     0       0          0
`
	if err := os.WriteFile(netDevPath, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write net/dev file: %v", err)
	}

	provider := &linuxNetworkProvider{
		procNetDevPath: netDevPath,
	}

	// Test concurrent access to provider methods
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			_, _ = provider.Interfaces()
			_, _ = provider.Stats("eth0")
			_, _ = provider.AllStats()
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
