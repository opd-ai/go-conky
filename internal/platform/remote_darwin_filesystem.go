package platform

import (
	"fmt"
	"strconv"
	"strings"
)

// remoteDarwinFilesystemProvider collects filesystem metrics from remote macOS systems via SSH.
type remoteDarwinFilesystemProvider struct {
	platform *sshPlatform
}

func newRemoteDarwinFilesystemProvider(p *sshPlatform) *remoteDarwinFilesystemProvider {
	return &remoteDarwinFilesystemProvider{
		platform: p,
	}
}

func (f *remoteDarwinFilesystemProvider) Mounts() ([]MountInfo, error) {
	output, err := f.platform.runCommand("mount")
	if err != nil {
		return nil, fmt.Errorf("failed to read mounts: %w", err)
	}

	var mounts []MountInfo
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		// Format: /dev/disk1s1 on / (apfs, local, journaled)
		parts := strings.Split(line, " on ")
		if len(parts) != 2 {
			continue
		}

		device := parts[0]
		rest := parts[1]

		// Split mount point and options
		idx := strings.Index(rest, " (")
		if idx == -1 {
			continue
		}

		mountPoint := rest[:idx]
		optStr := rest[idx+2:]
		optStr = strings.TrimSuffix(optStr, ")")

		// Parse filesystem type and options
		optParts := strings.SplitN(optStr, ",", 2)
		fsType := strings.TrimSpace(optParts[0])

		var options []string
		if len(optParts) > 1 {
			options = strings.Split(optParts[1], ",")
			for i := range options {
				options[i] = strings.TrimSpace(options[i])
			}
		}

		mount := MountInfo{
			Device:     device,
			MountPoint: mountPoint,
			FSType:     fsType,
			Options:    options,
		}

		mounts = append(mounts, mount)
	}

	return mounts, nil
}

func (f *remoteDarwinFilesystemProvider) Stats(mountPoint string) (*FilesystemStats, error) {
	// Use df command with shell-escaped mount point
	cmd := fmt.Sprintf("df -k %s | tail -n 1", shellEscape(mountPoint))
	output, err := f.platform.runCommand(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to get filesystem stats for %s: %w", mountPoint, err)
	}

	fields := strings.Fields(output)
	if len(fields) < 6 {
		return nil, fmt.Errorf("unexpected df output format: %s", output)
	}

	// df -k output format:
	// Filesystem 1K-blocks Used Available Capacity Mounted
	totalKB, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse total size: %w", err)
	}

	usedKB, err := strconv.ParseUint(fields[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse used size: %w", err)
	}

	freeKB, err := strconv.ParseUint(fields[3], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse free size: %w", err)
	}

	stats := &FilesystemStats{
		Total: totalKB * 1024,
		Used:  usedKB * 1024,
		Free:  freeKB * 1024,
	}

	if stats.Total > 0 {
		stats.UsedPercent = float64(stats.Used) / float64(stats.Total) * 100
	}

	// Try to get inode statistics with shell-escaped mount point
	cmd = fmt.Sprintf("df -i %s | tail -n 1", shellEscape(mountPoint))
	output, err = f.platform.runCommand(cmd)
	if err == nil {
		fields = strings.Fields(output)
		if len(fields) >= 6 {
			if inodesTotal, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
				stats.InodesTotal = inodesTotal
			}
			if inodesUsed, err := strconv.ParseUint(fields[2], 10, 64); err == nil {
				stats.InodesUsed = inodesUsed
			}
			if inodesFree, err := strconv.ParseUint(fields[3], 10, 64); err == nil {
				stats.InodesFree = inodesFree
			}
		}
	}

	return stats, nil
}

func (f *remoteDarwinFilesystemProvider) DiskIO(device string) (*DiskIOStats, error) {
	// macOS doesn't provide cumulative disk I/O statistics via simple commands.
	// The iostat command only provides rates (KB/t, tps, MB/s), not absolute counts.
	// To implement this properly would require maintaining state between calls to
	// calculate cumulative values from rates, which is beyond the scope of basic
	// remote monitoring.
	return nil, fmt.Errorf("disk I/O statistics not available on macOS via simple commands")
}
