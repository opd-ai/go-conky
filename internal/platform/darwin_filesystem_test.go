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

	// DiskIO returns zero values on macOS (requires IOKit)
	stats, err := provider.DiskIO("disk0")
	if err != nil {
		t.Fatalf("DiskIO() failed: %v", err)
	}

	// On macOS without IOKit, all values should be 0
	if stats.ReadBytes != 0 || stats.WriteBytes != 0 {
		t.Logf("Note: DiskIO returned non-zero values (IOKit may be available)")
	}
}
