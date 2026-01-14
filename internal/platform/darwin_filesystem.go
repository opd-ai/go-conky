//go:build darwin
// +build darwin

package platform

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
)

// darwinFilesystemProvider implements FilesystemProvider for macOS/Darwin systems using statfs.
type darwinFilesystemProvider struct{}

func newDarwinFilesystemProvider() *darwinFilesystemProvider {
	return &darwinFilesystemProvider{}
}

// Mounts returns a list of mounted filesystems.
func (f *darwinFilesystemProvider) Mounts() ([]MountInfo, error) {
	// Use the mount command to get mount information
	// This is more reliable than parsing /etc/fstab or using getmntinfo
	cmd := exec.Command("mount")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("running mount command: %w", err)
	}

	var mounts []MountInfo
	scanner := bufio.NewScanner(bytes.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()
		// Format: /dev/disk1s1 on / (apfs, local, journaled)
		// or: map auto_home on /System/Volumes/Data/home (autofs, automounted, nobrowse)

		parts := strings.Split(line, " on ")
		if len(parts) != 2 {
			continue
		}

		device := parts[0]

		// Split the rest by " ("
		rest := strings.Split(parts[1], " (")
		if len(rest) != 2 {
			continue
		}

		mountPoint := rest[0]

		// Extract filesystem type and options
		optsStr := strings.TrimSuffix(rest[1], ")")
		opts := strings.Split(optsStr, ", ")

		fsType := ""
		if len(opts) > 0 {
			fsType = opts[0]
			opts = opts[1:]
		}

		mounts = append(mounts, MountInfo{
			Device:     device,
			MountPoint: mountPoint,
			FSType:     fsType,
			Options:    opts,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("parsing mount output: %w", err)
	}

	return mounts, nil
}

// Stats returns filesystem statistics for a specific mount point.
func (f *darwinFilesystemProvider) Stats(mountPoint string) (*FilesystemStats, error) {
	var stat syscall.Statfs_t
	err := syscall.Statfs(mountPoint, &stat)
	if err != nil {
		return nil, fmt.Errorf("statfs on %s: %w", mountPoint, err)
	}

	// Calculate sizes
	blockSize := uint64(stat.Bsize)
	total := blockSize * stat.Blocks
	free := blockSize * stat.Bfree
	available := blockSize * stat.Bavail
	used := total - free

	usedPercent := 0.0
	if total > 0 {
		usedPercent = float64(used) / float64(total) * 100.0
	}

	return &FilesystemStats{
		Total:       total,
		Used:        used,
		Free:        free,
		UsedPercent: usedPercent,
		InodesTotal: stat.Files,
		InodesUsed:  stat.Files - stat.Ffree,
		InodesFree:  stat.Ffree,
	}, nil
}

// DiskIO returns disk I/O statistics for a specific device.
// Note: macOS doesn't provide easy access to per-device I/O stats without IOKit.
// This is a simplified implementation that returns aggregate stats.
func (f *darwinFilesystemProvider) DiskIO(device string) (*DiskIOStats, error) {
	// macOS requires IOKit to get detailed disk I/O statistics
	// For a pure Go implementation, we would need to use CGO to access IOKit
	// As a fallback, we return zero values with a note that detailed stats are unavailable

	// TODO: Implement using iostat parsing or IOKit (requires CGO)
	return &DiskIOStats{
		ReadBytes:  0,
		WriteBytes: 0,
		ReadCount:  0,
		WriteCount: 0,
		ReadTime:   0,
		WriteTime:  0,
	}, nil
}
