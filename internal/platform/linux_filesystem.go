package platform

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// linuxFilesystemProvider implements FilesystemProvider for Linux systems.
type linuxFilesystemProvider struct {
	procMountsPath string
}

func newLinuxFilesystemProvider() *linuxFilesystemProvider {
	return &linuxFilesystemProvider{
		procMountsPath: "/proc/mounts",
	}
}

func (f *linuxFilesystemProvider) Mounts() ([]MountInfo, error) {
	file, err := os.Open(f.procMountsPath)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", f.procMountsPath, err)
	}
	defer file.Close()

	var mounts []MountInfo
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		device := fields[0]
		mountPoint := unescapeMountPath(fields[1])
		fsType := fields[2]
		options := strings.Split(fields[3], ",")

		// Skip virtual filesystems
		if isVirtualFS(fsType) {
			continue
		}

		mounts = append(mounts, MountInfo{
			Device:     device,
			MountPoint: mountPoint,
			FSType:     fsType,
			Options:    options,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning %s: %w", f.procMountsPath, err)
	}

	return mounts, nil
}

func (f *linuxFilesystemProvider) Stats(mountPoint string) (*FilesystemStats, error) {
	var statfs syscall.Statfs_t
	if err := syscall.Statfs(mountPoint, &statfs); err != nil {
		return nil, fmt.Errorf("statfs %s: %w", mountPoint, err)
	}

	blockSize := uint64(statfs.Bsize)
	total := statfs.Blocks * blockSize
	free := statfs.Bfree * blockSize

	// Calculate used
	var used uint64
	if total >= free {
		used = total - free
	}

	// Calculate usage percentage
	var usedPercent float64
	if total > 0 {
		usedPercent = float64(used) / float64(total) * 100.0
	}

	// Inode statistics
	inodesTotal := statfs.Files
	inodesFree := statfs.Ffree
	var inodesUsed uint64
	if inodesTotal >= inodesFree {
		inodesUsed = inodesTotal - inodesFree
	}

	return &FilesystemStats{
		Total:       total,
		Used:        used,
		Free:        free,
		UsedPercent: usedPercent,
		InodesTotal: inodesTotal,
		InodesUsed:  inodesUsed,
		InodesFree:  inodesFree,
	}, nil
}

func (f *linuxFilesystemProvider) DiskIO(device string) (*DiskIOStats, error) {
	// Read from /proc/diskstats
	file, err := os.Open("/proc/diskstats")
	if err != nil {
		return nil, fmt.Errorf("opening /proc/diskstats: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 14 {
			continue
		}

		// Device name is at index 2
		if fields[2] != device {
			continue
		}

		// Parse disk I/O statistics
		// Field layout: https://www.kernel.org/doc/Documentation/ABI/testing/procfs-diskstats
		readsCompleted := parseUint64(fields[3])
		sectorsRead := parseUint64(fields[5])
		readTimeMs := parseUint64(fields[6])
		writesCompleted := parseUint64(fields[7])
		sectorsWritten := parseUint64(fields[9])
		writeTimeMs := parseUint64(fields[10])

		// Convert sectors to bytes (512 bytes per sector)
		const sectorSize = 512
		readBytes := sectorsRead * sectorSize
		writeBytes := sectorsWritten * sectorSize

		return &DiskIOStats{
			ReadBytes:  readBytes,
			WriteBytes: writeBytes,
			ReadCount:  readsCompleted,
			WriteCount: writesCompleted,
			ReadTime:   timeFromMillis(readTimeMs),
			WriteTime:  timeFromMillis(writeTimeMs),
		}, nil
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning /proc/diskstats: %w", err)
	}

	return nil, fmt.Errorf("device %s not found in /proc/diskstats", device)
}

// unescapeMountPath unescapes octal sequences in mount paths from /proc/mounts.
func unescapeMountPath(path string) string {
	// /proc/mounts escapes spaces and other characters as octal sequences (e.g., \040 for space)
	result := strings.Builder{}
	for i := 0; i < len(path); i++ {
		if path[i] == '\\' && i+3 < len(path) {
			// Try to parse octal sequence
			octalStr := path[i+1 : i+4]
			if val, err := strconv.ParseInt(octalStr, 8, 32); err == nil {
				result.WriteByte(byte(val))
				i += 3
				continue
			}
		}
		result.WriteByte(path[i])
	}
	return result.String()
}

// isVirtualFS checks if a filesystem type is virtual (no physical backing).
func isVirtualFS(fsType string) bool {
	virtualFS := map[string]bool{
		"proc":       true,
		"sysfs":      true,
		"devtmpfs":   true,
		"devpts":     true,
		"tmpfs":      true,
		"cgroup":     true,
		"cgroup2":    true,
		"pstore":     true,
		"bpf":        true,
		"debugfs":    true,
		"tracefs":    true,
		"securityfs": true,
		"fusectl":    true,
		"configfs":   true,
		"mqueue":     true,
		"hugetlbfs":  true,
		"autofs":     true,
		"rpc_pipefs": true,
	}
	return virtualFS[fsType]
}

// timeFromMillis converts milliseconds to time.Duration.
func timeFromMillis(ms uint64) time.Duration {
	return time.Duration(ms) * time.Millisecond
}
