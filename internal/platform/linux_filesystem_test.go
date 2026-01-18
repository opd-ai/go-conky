//go:build linux && !android
// +build linux,!android

package platform

import (
	"os"
	"path/filepath"
	"testing"
	"time"
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

func TestLinuxFilesystemProvider_DiskIO(t *testing.T) {
	// Create a temporary /proc/diskstats file
	tmpDir := t.TempDir()
	diskstatsPath := filepath.Join(tmpDir, "diskstats")

	// Example diskstats content (from Linux kernel documentation)
	// Format: major minor name reads reads_merged sectors_read read_time writes ...
	content := `   8       0 sda 139631 11053 4146832 45912 189264 51212 5248152 167292 0 96316 213204
   8       1 sda1 1234 567 24680 500 890 123 9876 250 0 700 750
   8      16 sdb 50000 1000 1000000 10000 25000 500 500000 5000 0 15000 15000
  11       0 sr0 0 0 0 0 0 0 0 0 0 0 0
`
	if err := os.WriteFile(diskstatsPath, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write diskstats file: %v", err)
	}

	provider := &linuxFilesystemProvider{
		procDiskstatsPath: diskstatsPath,
	}

	tests := []struct {
		name       string
		device     string
		wantErr    bool
		wantReads  uint64
		wantWrites uint64
	}{
		{
			name:       "sda device",
			device:     "sda",
			wantErr:    false,
			wantReads:  139631,
			wantWrites: 189264,
		},
		{
			name:       "sda1 partition",
			device:     "sda1",
			wantErr:    false,
			wantReads:  1234,
			wantWrites: 890,
		},
		{
			name:       "sdb device",
			device:     "sdb",
			wantErr:    false,
			wantReads:  50000,
			wantWrites: 25000,
		},
		{
			name:    "non-existent device",
			device:  "nvme0n1",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats, err := provider.DiskIO(tt.device)
			if tt.wantErr {
				if err == nil {
					t.Error("DiskIO() should have returned an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("DiskIO() failed: %v", err)
			}
			if stats.ReadCount != tt.wantReads {
				t.Errorf("ReadCount = %d, want %d", stats.ReadCount, tt.wantReads)
			}
			if stats.WriteCount != tt.wantWrites {
				t.Errorf("WriteCount = %d, want %d", stats.WriteCount, tt.wantWrites)
			}
		})
	}
}

func TestLinuxFilesystemProvider_DiskIO_FileNotFound(t *testing.T) {
	provider := &linuxFilesystemProvider{
		procDiskstatsPath: "/nonexistent/path/diskstats",
	}

	_, err := provider.DiskIO("sda")
	if err == nil {
		t.Error("DiskIO() should fail when diskstats file doesn't exist")
	}
}

func TestLinuxFilesystemProvider_DiskIO_BytesCalculation(t *testing.T) {
	// Test that sector-to-byte conversion is correct (512 bytes per sector)
	tmpDir := t.TempDir()
	diskstatsPath := filepath.Join(tmpDir, "diskstats")

	// 1000 sectors read, 2000 sectors written
	// Should be 512000 bytes read, 1024000 bytes written
	content := `   8       0 sda 100 0 1000 50 200 0 2000 100 0 150 150
`
	if err := os.WriteFile(diskstatsPath, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write diskstats file: %v", err)
	}

	provider := &linuxFilesystemProvider{
		procDiskstatsPath: diskstatsPath,
	}

	stats, err := provider.DiskIO("sda")
	if err != nil {
		t.Fatalf("DiskIO() failed: %v", err)
	}

	expectedReadBytes := uint64(1000 * 512)
	expectedWriteBytes := uint64(2000 * 512)

	if stats.ReadBytes != expectedReadBytes {
		t.Errorf("ReadBytes = %d, want %d", stats.ReadBytes, expectedReadBytes)
	}
	if stats.WriteBytes != expectedWriteBytes {
		t.Errorf("WriteBytes = %d, want %d", stats.WriteBytes, expectedWriteBytes)
	}
}

func TestLinuxFilesystemProvider_DiskIO_TimeConversion(t *testing.T) {
	// Test that millisecond-to-duration conversion is correct
	tmpDir := t.TempDir()
	diskstatsPath := filepath.Join(tmpDir, "diskstats")

	// 1500ms read time, 2500ms write time
	content := `   8       0 sda 100 0 1000 1500 200 0 2000 2500 0 4000 4000
`
	if err := os.WriteFile(diskstatsPath, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write diskstats file: %v", err)
	}

	provider := &linuxFilesystemProvider{
		procDiskstatsPath: diskstatsPath,
	}

	stats, err := provider.DiskIO("sda")
	if err != nil {
		t.Fatalf("DiskIO() failed: %v", err)
	}

	expectedReadTime := time.Duration(1500) * time.Millisecond
	expectedWriteTime := time.Duration(2500) * time.Millisecond

	if stats.ReadTime != expectedReadTime {
		t.Errorf("ReadTime = %v, want %v", stats.ReadTime, expectedReadTime)
	}
	if stats.WriteTime != expectedWriteTime {
		t.Errorf("WriteTime = %v, want %v", stats.WriteTime, expectedWriteTime)
	}
}

func TestTimeFromMillis(t *testing.T) {
	tests := []struct {
		name string
		ms   uint64
		want time.Duration
	}{
		{
			name: "zero",
			ms:   0,
			want: 0,
		},
		{
			name: "one millisecond",
			ms:   1,
			want: time.Millisecond,
		},
		{
			name: "one second",
			ms:   1000,
			want: time.Second,
		},
		{
			name: "one minute",
			ms:   60000,
			want: time.Minute,
		},
		{
			name: "mixed",
			ms:   1500,
			want: 1500 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := timeFromMillis(tt.ms)
			if got != tt.want {
				t.Errorf("timeFromMillis(%d) = %v, want %v", tt.ms, got, tt.want)
			}
		})
	}
}
