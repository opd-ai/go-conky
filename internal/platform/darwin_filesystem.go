//go:build darwin
// +build darwin

package platform

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// darwinFilesystemProvider implements FilesystemProvider for macOS/Darwin systems using statfs.
type darwinFilesystemProvider struct {
	// diskIOCache stores previous iostat readings for rate calculation
	diskIOCache   map[string]*diskIOCacheEntry
	diskIOCacheMu sync.RWMutex
}

// diskIOCacheEntry caches iostat data for a device
type diskIOCacheEntry struct {
	lastUpdate     time.Time
	cumulativeKB   float64 // Cumulative KB transferred
	cumulativeOps  uint64  // Cumulative operations
	lastReadBytes  uint64  // Last calculated read bytes (estimated)
	lastWriteBytes uint64  // Last calculated write bytes (estimated)
}

func newDarwinFilesystemProvider() *darwinFilesystemProvider {
	return &darwinFilesystemProvider{
		diskIOCache: make(map[string]*diskIOCacheEntry),
	}
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
// On macOS, this parses iostat output to get disk I/O metrics.
// Note: macOS iostat doesn't separate read/write, so we estimate based on total throughput.
func (f *darwinFilesystemProvider) DiskIO(device string) (*DiskIOStats, error) {
	// Run iostat to get disk statistics
	// iostat -d -I shows cumulative I/O counts since boot
	// iostat -d shows per-second rates
	cmd := exec.Command("iostat", "-d", "-I")
	output, err := cmd.Output()
	if err != nil {
		// Fall back to zero values if iostat not available
		return &DiskIOStats{}, nil
	}

	stats, err := f.parseIOStat(output, device)
	if err != nil {
		return &DiskIOStats{}, nil
	}

	return stats, nil
}

// parseIOStat parses iostat -d -I output and returns disk statistics.
// The format is:
//
//	disk0               disk1
//	KB/t  xfrs   MB    KB/t  xfrs   MB
//	24.00 12345 289.12 16.00 5678 88.76
func (f *darwinFilesystemProvider) parseIOStat(output []byte, device string) (*DiskIOStats, error) {
	lines := strings.Split(string(output), "\n")
	if len(lines) < 3 {
		return nil, fmt.Errorf("unexpected iostat output format")
	}

	// First line contains device names (or may be empty on some versions)
	// Second line contains column headers
	// Third line contains values

	// Find the header line (contains KB/t)
	var headerLine, dataLine string
	for i, line := range lines {
		if strings.Contains(line, "KB/t") {
			headerLine = line
			if i+1 < len(lines) {
				dataLine = lines[i+1]
			}
			break
		}
	}

	if headerLine == "" || dataLine == "" {
		return nil, fmt.Errorf("could not find iostat data")
	}

	// Find device names line (line before header, or infer from first line)
	var deviceLine string
	for i, line := range lines {
		if strings.Contains(line, "disk") && !strings.Contains(line, "KB/t") {
			deviceLine = line
			break
		}
		// On some macOS versions, the header line is first
		if strings.Contains(lines[i], "KB/t") && i > 0 {
			deviceLine = lines[i-1]
			break
		}
	}

	// Find the device index
	deviceIndex := -1
	if deviceLine != "" {
		devices := strings.Fields(deviceLine)
		for i, d := range devices {
			if d == device {
				deviceIndex = i
				break
			}
		}
	} else {
		// Default to disk0 at index 0
		if device == "disk0" {
			deviceIndex = 0
		}
	}

	if deviceIndex < 0 {
		return nil, fmt.Errorf("device %s not found", device)
	}

	// Parse data values - each device has 3 columns: KB/t, xfrs, MB
	dataFields := strings.Fields(dataLine)
	startIdx := deviceIndex * 3 // 3 columns per device

	if startIdx+2 >= len(dataFields) {
		return nil, fmt.Errorf("insufficient data for device %s", device)
	}

	// Parse cumulative values
	kbPerTransfer, _ := strconv.ParseFloat(dataFields[startIdx], 64)
	transfers, _ := strconv.ParseUint(dataFields[startIdx+1], 10, 64)
	totalMB, _ := strconv.ParseFloat(dataFields[startIdx+2], 64)

	// Calculate bytes from cumulative MB
	totalBytes := uint64(totalMB * 1024 * 1024)

	// Since macOS iostat doesn't separate read/write, we estimate 50/50 split
	// This is a reasonable approximation for general monitoring
	readBytes := totalBytes / 2
	writeBytes := totalBytes / 2

	// Calculate read/write counts (assume similar split)
	readCount := transfers / 2
	writeCount := transfers - readCount

	// Estimate time per operation (based on typical SSD latency ~100us per op)
	avgOpTimeMs := kbPerTransfer / 500.0 // Rough estimate: 500KB/s per ms
	if avgOpTimeMs < 0.01 {
		avgOpTimeMs = 0.01
	}
	readTime := time.Duration(float64(readCount)*avgOpTimeMs) * time.Millisecond
	writeTime := time.Duration(float64(writeCount)*avgOpTimeMs) * time.Millisecond

	return &DiskIOStats{
		ReadBytes:  readBytes,
		WriteBytes: writeBytes,
		ReadCount:  readCount,
		WriteCount: writeCount,
		ReadTime:   readTime,
		WriteTime:  writeTime,
	}, nil
}
