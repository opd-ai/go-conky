package monitor

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"
)

// filesystemReader reads filesystem statistics from /proc/mounts and statfs (via syscall.Statfs).
type filesystemReader struct {
	procMountsPath string
}

// newFilesystemReader creates a new filesystemReader with default paths.
func newFilesystemReader() *filesystemReader {
	return &filesystemReader{
		procMountsPath: "/proc/mounts",
	}
}

// ReadStats reads current filesystem statistics from /proc/mounts.
func (r *filesystemReader) ReadStats() (FilesystemStats, error) {
	mounts, err := r.readProcMounts()
	if err != nil {
		return FilesystemStats{}, err
	}

	stats := FilesystemStats{
		Mounts: make(map[string]MountStats, len(mounts)),
	}

	for _, mount := range mounts {
		// Skip virtual filesystems
		if isVirtualFS(mount.fsType) {
			continue
		}

		mountStats, err := r.getStatfs(mount)
		if err != nil {
			// Skip mounts that fail statfs (e.g., disconnected network shares)
			continue
		}

		stats.Mounts[mount.mountPoint] = mountStats
	}

	return stats, nil
}

// mountInfo represents a parsed line from /proc/mounts.
type mountInfo struct {
	device     string
	mountPoint string
	fsType     string
}

// readProcMounts parses /proc/mounts and returns mount information.
func (r *filesystemReader) readProcMounts() ([]mountInfo, error) {
	file, err := os.Open(r.procMountsPath)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", r.procMountsPath, err)
	}
	defer file.Close()

	var mounts []mountInfo
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		mounts = append(mounts, mountInfo{
			device:     fields[0],
			mountPoint: unescapeMountPath(fields[1]),
			fsType:     fields[2],
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning %s: %w", r.procMountsPath, err)
	}

	return mounts, nil
}

// getStatfs calls statfs on a mount point and returns statistics.
func (r *filesystemReader) getStatfs(mount mountInfo) (MountStats, error) {
	var statfs syscall.Statfs_t
	if err := syscall.Statfs(mount.mountPoint, &statfs); err != nil {
		return MountStats{}, fmt.Errorf("statfs %s: %w", mount.mountPoint, err)
	}

	// Calculate disk space with overflow protection
	blockSize := uint64(statfs.Bsize)
	total := safeMultiply(statfs.Blocks, blockSize)
	free := safeMultiply(statfs.Bfree, blockSize)
	available := safeMultiply(statfs.Bavail, blockSize)
	used := safeSubtract(total, free)

	// Calculate usage percentage
	var usagePercent float64
	if total > 0 {
		usagePercent = float64(used) / float64(total) * 100.0
	}

	// Calculate inode statistics
	inodesTotal := statfs.Files
	inodesFree := statfs.Ffree
	inodesUsed := safeSubtract(inodesTotal, inodesFree)

	var inodesPercent float64
	if inodesTotal > 0 {
		inodesPercent = float64(inodesUsed) / float64(inodesTotal) * 100.0
	}

	return MountStats{
		MountPoint:    mount.mountPoint,
		Device:        mount.device,
		FSType:        mount.fsType,
		Total:         total,
		Used:          used,
		Free:          free,
		Available:     available,
		UsagePercent:  usagePercent,
		InodesTotal:   inodesTotal,
		InodesUsed:    inodesUsed,
		InodesFree:    inodesFree,
		InodesPercent: inodesPercent,
	}, nil
}

// isVirtualFS returns true if the filesystem type is virtual/pseudo.
func isVirtualFS(fsType string) bool {
	virtualTypes := map[string]bool{
		"sysfs":       true,
		"proc":        true,
		"devtmpfs":    true,
		"devpts":      true,
		"tmpfs":       true,
		"securityfs":  true,
		"cgroup":      true,
		"cgroup2":     true,
		"pstore":      true,
		"efivarfs":    true,
		"bpf":         true,
		"debugfs":     true,
		"tracefs":     true,
		"hugetlbfs":   true,
		"mqueue":      true,
		"fusectl":     true,
		"configfs":    true,
		"selinuxfs":   true,
		"autofs":      true,
		"rpc_pipefs":  true,
		"nsfs":        true,
		"ramfs":       true,
		"overlay":     true,
		"fuse.portal": true,
	}
	return virtualTypes[fsType]
}

// unescapeMountPath handles escaped characters in mount paths.
// /proc/mounts uses octal escapes for special characters (e.g., \040 for space).
func unescapeMountPath(path string) string {
	// Common octal escapes in mount paths
	result := strings.ReplaceAll(path, "\\040", " ")
	result = strings.ReplaceAll(result, "\\011", "\t")
	result = strings.ReplaceAll(result, "\\012", "\n")
	result = strings.ReplaceAll(result, "\\134", "\\")
	return result
}
