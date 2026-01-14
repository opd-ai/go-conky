package platform

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLinuxFilesystemProvider_Mounts(t *testing.T) {
	// Create a temporary /proc/mounts file
	tmpDir := t.TempDir()
	mountsPath := filepath.Join(tmpDir, "mounts")

	content := `/dev/sda1 / ext4 rw,relatime 0 0
/dev/sda2 /home ext4 rw,relatime 0 0
tmpfs /tmp tmpfs rw,nosuid,nodev 0 0
proc /proc proc rw,nosuid,nodev,noexec,relatime 0 0
sysfs /sys sysfs rw,nosuid,nodev,noexec,relatime 0 0
`
	if err := os.WriteFile(mountsPath, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write mounts file: %v", err)
	}

	provider := &linuxFilesystemProvider{
		procMountsPath: mountsPath,
	}

	mounts, err := provider.Mounts()
	if err != nil {
		t.Fatalf("Mounts() failed: %v", err)
	}

	// Should filter out virtual filesystems (tmpfs, proc, sysfs)
	if len(mounts) != 2 {
		t.Errorf("Mounts() returned %d mounts, want 2 (virtual filesystems should be filtered)", len(mounts))
	}

	// Check first mount
	found := false
	for _, mount := range mounts {
		if mount.Device == "/dev/sda1" {
			found = true
			if mount.MountPoint != "/" {
				t.Errorf("Mount point = %s, want /", mount.MountPoint)
			}
			if mount.FSType != "ext4" {
				t.Errorf("FSType = %s, want ext4", mount.FSType)
			}
			if len(mount.Options) == 0 {
				t.Error("Options should not be empty")
			}
		}
	}
	if !found {
		t.Error("Mount /dev/sda1 not found")
	}
}

func TestLinuxFilesystemProvider_Mounts_OctalEscape(t *testing.T) {
	// Create a temporary /proc/mounts file with octal-escaped mount point
	tmpDir := t.TempDir()
	mountsPath := filepath.Join(tmpDir, "mounts")

	// \040 is the octal escape for a space character
	content := `/dev/sdb1 /mnt/my\040drive ext4 rw,relatime 0 0
`
	if err := os.WriteFile(mountsPath, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write mounts file: %v", err)
	}

	provider := &linuxFilesystemProvider{
		procMountsPath: mountsPath,
	}

	mounts, err := provider.Mounts()
	if err != nil {
		t.Fatalf("Mounts() failed: %v", err)
	}

	if len(mounts) != 1 {
		t.Fatalf("Mounts() returned %d mounts, want 1", len(mounts))
	}

	// Check that octal escape is properly decoded
	if mounts[0].MountPoint != "/mnt/my drive" {
		t.Errorf("Mount point = %q, want %q", mounts[0].MountPoint, "/mnt/my drive")
	}
}

func TestUnescapeMountPath(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no escapes",
			input: "/mnt/data",
			want:  "/mnt/data",
		},
		{
			name:  "space escape",
			input: "/mnt/my\\040drive",
			want:  "/mnt/my drive",
		},
		{
			name:  "multiple escapes",
			input: "/mnt/my\\040test\\040drive",
			want:  "/mnt/my test drive",
		},
		{
			name:  "tab escape",
			input: "/mnt/my\\011drive",
			want:  "/mnt/my\tdrive",
		},
		{
			name:  "invalid escape",
			input: "/mnt/my\\xyz",
			want:  "/mnt/my\\xyz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := unescapeMountPath(tt.input)
			if got != tt.want {
				t.Errorf("unescapeMountPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsVirtualFS(t *testing.T) {
	tests := []struct {
		fsType string
		want   bool
	}{
		{"ext4", false},
		{"xfs", false},
		{"btrfs", false},
		{"proc", true},
		{"sysfs", true},
		{"tmpfs", true},
		{"devtmpfs", true},
		{"devpts", true},
		{"cgroup", true},
		{"cgroup2", true},
		{"debugfs", true},
	}

	for _, tt := range tests {
		t.Run(tt.fsType, func(t *testing.T) {
			got := isVirtualFS(tt.fsType)
			if got != tt.want {
				t.Errorf("isVirtualFS(%q) = %v, want %v", tt.fsType, got, tt.want)
			}
		})
	}
}

func TestLinuxFilesystemProvider_Stats(t *testing.T) {
	// We can't easily test Stats() without actually mounting filesystems,
	// but we can at least test that it returns an error for non-existent paths
	provider := &linuxFilesystemProvider{
		procMountsPath: "/proc/mounts",
	}

	_, err := provider.Stats("/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Error("Stats() should fail for non-existent path")
	}
}
