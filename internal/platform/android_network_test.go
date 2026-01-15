//go:build android
// +build android

package platform

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAndroidNetworkProvider_Interfaces(t *testing.T) {
	tmpDir := t.TempDir()
	netDevPath := filepath.Join(tmpDir, "dev")

	content := `Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
    lo: 1000      10    0    0    0     0          0         0     1000      10    0    0    0     0       0          0
 wlan0: 2000000   5000    1    2    0     0          0         0  1500000   4000    0    1    0     0       0          0
`
	if err := os.WriteFile(netDevPath, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write net/dev file: %v", err)
	}

	provider := &androidNetworkProvider{
		procNetDevPath: netDevPath,
	}

	interfaces, err := provider.Interfaces()
	if err != nil {
		t.Fatalf("Interfaces() failed: %v", err)
	}

	if len(interfaces) != 2 {
		t.Errorf("Interfaces() returned %d interfaces, want 2", len(interfaces))
	}

	found := make(map[string]bool)
	for _, iface := range interfaces {
		found[iface] = true
	}

	if !found["lo"] {
		t.Error("Interface 'lo' not found")
	}
	if !found["wlan0"] {
		t.Error("Interface 'wlan0' not found")
	}
}

func TestAndroidNetworkProvider_Stats(t *testing.T) {
	tmpDir := t.TempDir()
	netDevPath := filepath.Join(tmpDir, "dev")

	content := `Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
 wlan0: 2000000   5000    1    2    0     0          0         0  1500000   4000    3    4    0     0       0          0
`
	if err := os.WriteFile(netDevPath, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write net/dev file: %v", err)
	}

	provider := &androidNetworkProvider{
		procNetDevPath: netDevPath,
	}

	stats, err := provider.Stats("wlan0")
	if err != nil {
		t.Fatalf("Stats() failed: %v", err)
	}

	if stats.BytesRecv != 2000000 {
		t.Errorf("BytesRecv = %v, want 2000000", stats.BytesRecv)
	}
	if stats.BytesSent != 1500000 {
		t.Errorf("BytesSent = %v, want 1500000", stats.BytesSent)
	}
	if stats.PacketsRecv != 5000 {
		t.Errorf("PacketsRecv = %v, want 5000", stats.PacketsRecv)
	}
	if stats.PacketsSent != 4000 {
		t.Errorf("PacketsSent = %v, want 4000", stats.PacketsSent)
	}
	if stats.ErrorsIn != 1 {
		t.Errorf("ErrorsIn = %v, want 1", stats.ErrorsIn)
	}
	if stats.ErrorsOut != 3 {
		t.Errorf("ErrorsOut = %v, want 3", stats.ErrorsOut)
	}
	if stats.DropIn != 2 {
		t.Errorf("DropIn = %v, want 2", stats.DropIn)
	}
	if stats.DropOut != 4 {
		t.Errorf("DropOut = %v, want 4", stats.DropOut)
	}
}

func TestAndroidNetworkProvider_StatsNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	netDevPath := filepath.Join(tmpDir, "dev")

	content := `Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
 wlan0: 2000000   5000    1    2    0     0          0         0  1500000   4000    0    1    0     0       0          0
`
	if err := os.WriteFile(netDevPath, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write net/dev file: %v", err)
	}

	provider := &androidNetworkProvider{
		procNetDevPath: netDevPath,
	}

	_, err := provider.Stats("eth0")
	if err == nil {
		t.Error("Stats() should have returned an error for non-existent interface")
	}
}

func TestAndroidNetworkProvider_AllStats(t *testing.T) {
	tmpDir := t.TempDir()
	netDevPath := filepath.Join(tmpDir, "dev")

	content := `Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
    lo: 1000      10    0    0    0     0          0         0     1000      10    0    0    0     0       0          0
 wlan0: 2000000   5000    1    2    0     0          0         0  1500000   4000    0    1    0     0       0          0
`
	if err := os.WriteFile(netDevPath, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write net/dev file: %v", err)
	}

	provider := &androidNetworkProvider{
		procNetDevPath: netDevPath,
	}

	allStats, err := provider.AllStats()
	if err != nil {
		t.Fatalf("AllStats() failed: %v", err)
	}

	if len(allStats) != 2 {
		t.Errorf("AllStats() returned %d interfaces, want 2", len(allStats))
	}

	if stats, ok := allStats["lo"]; ok {
		if stats.BytesRecv != 1000 {
			t.Errorf("lo BytesRecv = %v, want 1000", stats.BytesRecv)
		}
	} else {
		t.Error("Interface 'lo' not found in AllStats()")
	}

	if stats, ok := allStats["wlan0"]; ok {
		if stats.BytesRecv != 2000000 {
			t.Errorf("wlan0 BytesRecv = %v, want 2000000", stats.BytesRecv)
		}
	} else {
		t.Error("Interface 'wlan0' not found in AllStats()")
	}
}
