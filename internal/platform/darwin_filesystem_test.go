//go:build darwin
// +build darwin

package platform

import (
	"testing"
)

func TestDarwinFilesystemProvider_Mounts(t *testing.T) {
	provider := newDarwinFilesystemProvider()

	mounts, err := provider.Mounts()
	if err != nil {
		t.Fatalf("Mounts() failed: %v", err)
	}

	// Should have at least one mount (root filesystem)
	if len(mounts) == 0 {
		t.Error("Expected at least one mounted filesystem")
	}

	// Check for root filesystem
	foundRoot := false
	for _, mount := range mounts {
		if mount.MountPoint == "/" {
			foundRoot = true
			if mount.Device == "" {
				t.Error("Root filesystem device should not be empty")
			}
			if mount.FSType == "" {
				t.Error("Root filesystem type should not be empty")
			}
		}

		// Basic validation
		if mount.MountPoint == "" {
			t.Error("Mount point should not be empty")
		}
	}

	if !foundRoot {
		t.Error("Root filesystem (/) not found in mounts")
	}
}

func TestDarwinFilesystemProvider_Stats(t *testing.T) {
	provider := newDarwinFilesystemProvider()

	// Test with root filesystem
	stats, err := provider.Stats("/")
	if err != nil {
		t.Fatalf("Stats('/') failed: %v", err)
	}

	if stats.Total == 0 {
		t.Error("Total space should not be zero")
	}

	if stats.Used > stats.Total {
		t.Errorf("Used space (%d) should not exceed total (%d)", stats.Used, stats.Total)
	}

	if stats.Free > stats.Total {
		t.Errorf("Free space (%d) should not exceed total (%d)", stats.Free, stats.Total)
	}

	if stats.UsedPercent < 0 || stats.UsedPercent > 100 {
		t.Errorf("Used percentage should be between 0 and 100, got %f", stats.UsedPercent)
	}

	// Check inode stats
	if stats.InodesTotal > 0 {
		if stats.InodesUsed > stats.InodesTotal {
			t.Errorf("Used inodes (%d) should not exceed total (%d)",
				stats.InodesUsed, stats.InodesTotal)
		}

		if stats.InodesFree > stats.InodesTotal {
			t.Errorf("Free inodes (%d) should not exceed total (%d)",
				stats.InodesFree, stats.InodesTotal)
		}
	}
}

func TestDarwinFilesystemProvider_Stats_InvalidPath(t *testing.T) {
	provider := newDarwinFilesystemProvider()

	_, err := provider.Stats("/nonexistent/path/xyz123")
	if err == nil {
		t.Error("Expected error for invalid path")
	}
}

func TestDarwinFilesystemProvider_DiskIO(t *testing.T) {
	provider := newDarwinFilesystemProvider()

	// DiskIO now parses iostat output on macOS
	stats, err := provider.DiskIO("disk0")
	if err != nil {
		t.Fatalf("DiskIO() failed: %v", err)
	}

	// Stats should be non-nil
	if stats == nil {
		t.Fatal("DiskIO() returned nil stats")
	}

	// Log the values for inspection
	t.Logf("DiskIO stats for disk0: ReadBytes=%d, WriteBytes=%d, ReadCount=%d, WriteCount=%d",
		stats.ReadBytes, stats.WriteBytes, stats.ReadCount, stats.WriteCount)
}

func TestDarwinFilesystemProvider_DiskIO_UnknownDevice(t *testing.T) {
	provider := newDarwinFilesystemProvider()

	// Unknown device should return zero stats without error
	stats, err := provider.DiskIO("disk999")
	if err != nil {
		t.Fatalf("DiskIO() for unknown device should not error: %v", err)
	}

	// Should return empty stats
	if stats == nil {
		t.Fatal("DiskIO() returned nil stats")
	}
}

func TestDarwinFilesystemProvider_parseIOStat(t *testing.T) {
	provider := newDarwinFilesystemProvider()

	// Sample iostat -d -I output
	sampleOutput := `          disk0               disk1 
    KB/t  xfrs   MB      KB/t  xfrs   MB 
   24.00 12345  289.12   16.00  5678   88.76
`

	stats, err := provider.parseIOStat([]byte(sampleOutput), "disk0")
	if err != nil {
		t.Fatalf("parseIOStat() failed: %v", err)
	}

	// Total MB is 289.12, so total bytes ~= 303,173,222
	expectedTotalBytes := uint64(289.12 * 1024 * 1024)
	actualTotalBytes := stats.ReadBytes + stats.WriteBytes

	// Allow some tolerance for floating point
	tolerance := uint64(1024) // 1KB tolerance
	if actualTotalBytes < expectedTotalBytes-tolerance || actualTotalBytes > expectedTotalBytes+tolerance {
		t.Errorf("Total bytes mismatch: expected ~%d, got %d", expectedTotalBytes, actualTotalBytes)
	}

	// Transfers should sum to original
	expectedTransfers := uint64(12345)
	actualTransfers := stats.ReadCount + stats.WriteCount
	if actualTransfers != expectedTransfers {
		t.Errorf("Transfer count mismatch: expected %d, got %d", expectedTransfers, actualTransfers)
	}

	// Test disk1
	stats2, err := provider.parseIOStat([]byte(sampleOutput), "disk1")
	if err != nil {
		t.Fatalf("parseIOStat() for disk1 failed: %v", err)
	}

	expectedTotalBytes2 := uint64(88.76 * 1024 * 1024)
	actualTotalBytes2 := stats2.ReadBytes + stats2.WriteBytes

	if actualTotalBytes2 < expectedTotalBytes2-tolerance || actualTotalBytes2 > expectedTotalBytes2+tolerance {
		t.Errorf("disk1 total bytes mismatch: expected ~%d, got %d", expectedTotalBytes2, actualTotalBytes2)
	}
}

func TestDarwinFilesystemProvider_parseIOStat_InvalidFormat(t *testing.T) {
	provider := newDarwinFilesystemProvider()

	// Empty output
	_, err := provider.parseIOStat([]byte(""), "disk0")
	if err == nil {
		t.Error("Expected error for empty output")
	}

	// Malformed output
	_, err = provider.parseIOStat([]byte("invalid data"), "disk0")
	if err == nil {
		t.Error("Expected error for malformed output")
	}
}
