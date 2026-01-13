package monitor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsVirtualFS(t *testing.T) {
	tests := []struct {
		name     string
		fsType   string
		expected bool
	}{
		{"sysfs", "sysfs", true},
		{"proc", "proc", true},
		{"tmpfs", "tmpfs", true},
		{"cgroup2", "cgroup2", true},
		{"ext4", "ext4", false},
		{"xfs", "xfs", false},
		{"ntfs", "ntfs", false},
		{"btrfs", "btrfs", false},
		{"vfat", "vfat", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isVirtualFS(tt.fsType)
			if got != tt.expected {
				t.Errorf("isVirtualFS(%q) = %v, want %v", tt.fsType, got, tt.expected)
			}
		})
	}
}

func TestUnescapeMountPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"no escape", "/home/user", "/home/user"},
		{"space escape", "/home/my\\040documents", "/home/my documents"},
		{"tab escape", "/home/my\\011tabs", "/home/my\ttabs"},
		{"newline escape", "/home/my\\012line", "/home/my\nline"},
		{"backslash escape", "/home/my\\134path", "/home/my\\path"},
		{"multiple escapes", "/home/my\\040test\\040folder", "/home/my test folder"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := unescapeMountPath(tt.input)
			if got != tt.expected {
				t.Errorf("unescapeMountPath(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestFilesystemReaderReadProcMounts(t *testing.T) {
	tmpDir := t.TempDir()

	mountsContent := `/dev/sda1 / ext4 rw,relatime 0 0
/dev/sda2 /home ext4 rw,relatime 0 0
/dev/sdb1 /mnt/data xfs rw,relatime 0 0
proc /proc proc rw,nosuid,nodev,noexec,relatime 0 0
sysfs /sys sysfs rw,nosuid,nodev,noexec,relatime 0 0
tmpfs /run tmpfs rw,nosuid,nodev,mode=755 0 0
`
	if err := os.WriteFile(filepath.Join(tmpDir, "mounts"), []byte(mountsContent), 0o644); err != nil {
		t.Fatalf("failed to write mock mounts: %v", err)
	}

	reader := &filesystemReader{
		procMountsPath: filepath.Join(tmpDir, "mounts"),
	}

	mounts, err := reader.readProcMounts()
	if err != nil {
		t.Fatalf("readProcMounts() error = %v", err)
	}

	// Should have 6 mounts
	if len(mounts) != 6 {
		t.Errorf("got %d mounts, want 6", len(mounts))
	}

	// Verify first mount
	if mounts[0].device != "/dev/sda1" {
		t.Errorf("first mount device = %q, want /dev/sda1", mounts[0].device)
	}
	if mounts[0].mountPoint != "/" {
		t.Errorf("first mount point = %q, want /", mounts[0].mountPoint)
	}
	if mounts[0].fsType != "ext4" {
		t.Errorf("first mount fsType = %q, want ext4", mounts[0].fsType)
	}
}

func TestFilesystemReaderReadProcMountsWithEscapes(t *testing.T) {
	tmpDir := t.TempDir()

	// In /proc/mounts, spaces are escaped as \040
	// Using regular string to properly represent the escape sequence
	mountsContent := "/dev/sda1 /home/my\\040documents ext4 rw,relatime 0 0\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "mounts"), []byte(mountsContent), 0o644); err != nil {
		t.Fatalf("failed to write mock mounts: %v", err)
	}

	reader := &filesystemReader{
		procMountsPath: filepath.Join(tmpDir, "mounts"),
	}

	mounts, err := reader.readProcMounts()
	if err != nil {
		t.Fatalf("readProcMounts() error = %v", err)
	}

	if len(mounts) != 1 {
		t.Fatalf("got %d mounts, want 1", len(mounts))
	}

	expected := "/home/my documents"
	if mounts[0].mountPoint != expected {
		t.Errorf("mount point = %q, want %q", mounts[0].mountPoint, expected)
	}
}

func TestFilesystemReaderMissingFile(t *testing.T) {
	reader := &filesystemReader{
		procMountsPath: "/nonexistent/mounts",
	}

	_, err := reader.ReadStats()
	if err == nil {
		t.Error("ReadStats() should return error for missing file")
	}
}

func TestFilesystemReaderEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmpDir, "mounts"), []byte(""), 0o644); err != nil {
		t.Fatalf("failed to write mock mounts: %v", err)
	}

	reader := &filesystemReader{
		procMountsPath: filepath.Join(tmpDir, "mounts"),
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	if len(stats.Mounts) != 0 {
		t.Errorf("got %d mounts from empty file, want 0", len(stats.Mounts))
	}
}

func TestFilesystemReaderMalformedLines(t *testing.T) {
	tmpDir := t.TempDir()

	mountsContent := `/dev/sda1 / ext4 rw,relatime 0 0
malformed line
/dev/sda2 /home ext4 rw,relatime 0 0
another bad
`
	if err := os.WriteFile(filepath.Join(tmpDir, "mounts"), []byte(mountsContent), 0o644); err != nil {
		t.Fatalf("failed to write mock mounts: %v", err)
	}

	reader := &filesystemReader{
		procMountsPath: filepath.Join(tmpDir, "mounts"),
	}

	mounts, err := reader.readProcMounts()
	if err != nil {
		t.Fatalf("readProcMounts() error = %v", err)
	}

	// Should only have 2 valid mounts
	if len(mounts) != 2 {
		t.Errorf("got %d mounts, want 2", len(mounts))
	}
}

func TestNewFilesystemReader(t *testing.T) {
	reader := newFilesystemReader()

	if reader.procMountsPath != "/proc/mounts" {
		t.Errorf("procMountsPath = %q, want %q", reader.procMountsPath, "/proc/mounts")
	}
}

func TestFilesystemReaderFiltersVirtualFS(t *testing.T) {
	// This test needs a system with /proc/mounts available
	// Skip if running on non-Linux or in a container without /proc
	if _, err := os.Stat("/proc/mounts"); os.IsNotExist(err) {
		t.Skip("Skipping test: /proc/mounts not available")
	}

	reader := newFilesystemReader()
	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	// Check that virtual filesystems are filtered out
	for mountPoint, mount := range stats.Mounts {
		if isVirtualFS(mount.FSType) {
			t.Errorf("virtual filesystem %q (type %q) should be filtered", mountPoint, mount.FSType)
		}
	}
}
