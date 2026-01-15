//go:build android
// +build android

package platform

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAndroidFilesystemProvider_Mounts(t *testing.T) {
	tmpDir := t.TempDir()
	mountsPath := filepath.Join(tmpDir, "mounts")

	content := `/dev/block/sda1 / ext4 rw,relatime 0 0
/dev/block/sda2 /data ext4 rw,nosuid,nodev 0 0
proc /proc proc rw,nosuid,nodev,noexec 0 0
sysfs /sys sysfs rw,nosuid,nodev,noexec 0 0
tmpfs /tmp tmpfs rw,nosuid,nodev 0 0
`
	if err := os.WriteFile(mountsPath, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write mounts file: %v", err)
	}

	provider := &androidFilesystemProvider{
		procMountsPath: mountsPath,
	}

	mounts, err := provider.Mounts()
	if err != nil {
		t.Fatalf("Mounts() failed: %v", err)
	}

	// Should only return non-virtual filesystems (ext4)
	if len(mounts) != 2 {
		t.Errorf("Mounts() returned %d mounts, want 2 (non-virtual only)", len(mounts))
	}

	foundRoot := false
	foundData := false
	for _, mount := range mounts {
		if mount.MountPoint == "/" {
			foundRoot = true
			if mount.FSType != "ext4" {
				t.Errorf("Root FSType = %v, want ext4", mount.FSType)
			}
		}
		if mount.MountPoint == "/data" {
			foundData = true
		}
	}

	if !foundRoot {
		t.Error("Root mount not found")
	}
	if !foundData {
		t.Error("/data mount not found")
	}
}

func TestAndroidFilesystemProvider_MountsWithEscapedPaths(t *testing.T) {
	tmpDir := t.TempDir()
	mountsPath := filepath.Join(tmpDir, "mounts")

	// Path with space escaped as \040
	content := `/dev/sda1 /mnt/my\040drive ext4 rw 0 0
`
	if err := os.WriteFile(mountsPath, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write mounts file: %v", err)
	}

	provider := &androidFilesystemProvider{
		procMountsPath: mountsPath,
	}

	mounts, err := provider.Mounts()
	if err != nil {
		t.Fatalf("Mounts() failed: %v", err)
	}

	if len(mounts) != 1 {
		t.Fatalf("Mounts() returned %d mounts, want 1", len(mounts))
	}

	expected := "/mnt/my drive"
	if mounts[0].MountPoint != expected {
		t.Errorf("MountPoint = %v, want %v", mounts[0].MountPoint, expected)
	}
}

func TestAndroidFilesystemProvider_IsVirtualFS(t *testing.T) {
	testCases := []struct {
		fsType   string
		expected bool
	}{
		{"ext4", false},
		{"f2fs", false},
		{"proc", true},
		{"sysfs", true},
		{"tmpfs", true},
		{"devpts", true},
		{"cgroup", true},
		{"selinuxfs", true}, // Android-specific
		{"functionfs", true}, // Android-specific
	}

	for _, tc := range testCases {
		t.Run(tc.fsType, func(t *testing.T) {
			result := isVirtualFS(tc.fsType)
			if result != tc.expected {
				t.Errorf("isVirtualFS(%s) = %v, want %v", tc.fsType, result, tc.expected)
			}
		})
	}
}

func TestAndroidFilesystemProvider_UnescapeMountPath(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"/mnt/drive", "/mnt/drive"},
		{"/mnt/my\\040drive", "/mnt/my drive"},
		{"/mnt/tab\\011here", "/mnt/tab\there"},
		{"/mnt/normal", "/mnt/normal"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := unescapeMountPath(tc.input)
			if result != tc.expected {
				t.Errorf("unescapeMountPath(%s) = %v, want %v", tc.input, result, tc.expected)
			}
		})
	}
}
