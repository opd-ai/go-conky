//go:build windows
// +build windows

package platform

import (
	"testing"
)

func TestWindowsFilesystemProvider_Mounts(t *testing.T) {
	provider := newWindowsFilesystemProvider()

	mounts, err := provider.Mounts()
	if err != nil {
		t.Fatalf("Mounts() error = %v", err)
	}

	// Most Windows systems have at least C:\ drive
	if len(mounts) == 0 {
		t.Error("Mounts() returned empty slice, expected at least one drive")
	}

	// Validate mount structure
	for _, mount := range mounts {
		if mount.Device == "" {
			t.Error("Mount device is empty")
		}
		if mount.MountPoint == "" {
			t.Error("Mount point is empty")
		}
		if mount.FSType == "" {
			t.Error("FSType is empty")
		}
	}
}

func TestWindowsFilesystemProvider_Stats(t *testing.T) {
	provider := newWindowsFilesystemProvider()

	// Get list of mounts
	mounts, err := provider.Mounts()
	if err != nil {
		t.Fatalf("Mounts() error = %v", err)
	}

	if len(mounts) == 0 {
		t.Skip("No filesystem mounts available for testing")
	}

	// Test stats for first mount
	stats, err := provider.Stats(mounts[0].MountPoint)
	if err != nil {
		t.Fatalf("Stats(%s) error = %v", mounts[0].MountPoint, err)
	}

	if stats.Total == 0 {
		t.Error("Total space should not be 0")
	}

	if stats.UsedPercent < 0 || stats.UsedPercent > 100 {
		t.Errorf("UsedPercent = %v, want 0-100", stats.UsedPercent)
	}

	if stats.Free > stats.Total {
		t.Errorf("Free = %v > Total = %v", stats.Free, stats.Total)
	}

	if stats.Used > stats.Total {
		t.Errorf("Used = %v > Total = %v", stats.Used, stats.Total)
	}
}

func TestWindowsFilesystemProvider_StatsWithInvalidPath(t *testing.T) {
	provider := newWindowsFilesystemProvider()

	_, err := provider.Stats("InvalidPath12345")
	if err == nil {
		t.Error("Stats() with invalid path should return error")
	}
}

func TestWindowsFilesystemProvider_DiskIO(t *testing.T) {
	provider := newWindowsFilesystemProvider()

	// DiskIO returns empty stats for now (not implemented)
	stats, err := provider.DiskIO("C:")
	if err != nil {
		t.Fatalf("DiskIO() error = %v", err)
	}

	if stats == nil {
		t.Fatal("DiskIO returned nil")
	}
}
