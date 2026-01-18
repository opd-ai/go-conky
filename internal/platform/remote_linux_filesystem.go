package platform

import (
	"fmt"
	"strconv"
	"strings"
)

// remoteLinuxFilesystemProvider collects filesystem metrics from remote Linux systems via SSH.
type remoteLinuxFilesystemProvider struct {
	runner commandRunner
}

func newRemoteLinuxFilesystemProvider(p *sshPlatform) *remoteLinuxFilesystemProvider {
	return &remoteLinuxFilesystemProvider{
		runner: p,
	}
}

// newTestableRemoteLinuxFilesystemProviderWithRunner creates a provider with an injectable runner for testing.
func newTestableRemoteLinuxFilesystemProviderWithRunner(runner commandRunner) *remoteLinuxFilesystemProvider {
	return &remoteLinuxFilesystemProvider{
		runner: runner,
	}
}

func (f *remoteLinuxFilesystemProvider) Mounts() ([]MountInfo, error) {
	output, err := f.runner.runCommand("cat /proc/mounts")
	if err != nil {
		return nil, fmt.Errorf("failed to read /proc/mounts: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	mounts := make([]MountInfo, 0, len(lines))

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		mount := MountInfo{
			Device:     fields[0],
			MountPoint: fields[1],
			FSType:     fields[2],
		}

		if len(fields) > 3 {
			mount.Options = strings.Split(fields[3], ",")
		}

		mounts = append(mounts, mount)
	}

	return mounts, nil
}

func (f *remoteLinuxFilesystemProvider) Stats(mountPoint string) (*FilesystemStats, error) {
	// Use df command with shell-escaped mount point
	cmd := fmt.Sprintf("df -B1 %s | tail -n 1", shellEscape(mountPoint))
	output, err := f.runner.runCommand(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to get filesystem stats for %s: %w", mountPoint, err)
	}

	fields := strings.Fields(output)
	if len(fields) < 6 {
		return nil, fmt.Errorf("unexpected df output format: %s", output)
	}

	// df output format:
	// Filesystem 1B-blocks Used Available Use% Mounted
	total, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse total size: %w", err)
	}

	used, err := strconv.ParseUint(fields[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse used size: %w", err)
	}

	free, err := strconv.ParseUint(fields[3], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse free size: %w", err)
	}

	stats := &FilesystemStats{
		Total: total,
		Used:  used,
		Free:  free,
	}

	if total > 0 {
		stats.UsedPercent = float64(used) / float64(total) * 100
	}

	// Try to get inode statistics with shell-escaped mount point
	cmd = fmt.Sprintf("df -i %s | tail -n 1", shellEscape(mountPoint))
	output, err = f.runner.runCommand(cmd)
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

func (f *remoteLinuxFilesystemProvider) DiskIO(device string) (*DiskIOStats, error) {
	// Read all disk stats and search for the requested device to avoid command injection
	output, err := f.runner.runCommand("cat /proc/diskstats")
	if err != nil {
		return nil, fmt.Errorf("failed to read disk stats for %s: %w", device, err)
	}

	var line string
	for _, l := range strings.Split(strings.TrimSpace(output), "\n") {
		fields := strings.Fields(l)
		// /proc/diskstats format:
		// major minor name reads reads_merged sectors_read time_reading writes writes_merged sectors_written time_writing ...
		if len(fields) < 3 {
			continue
		}
		if fields[2] == device {
			line = l
			break
		}
	}

	if strings.TrimSpace(line) == "" {
		return nil, fmt.Errorf("device %s not found in /proc/diskstats", device)
	}

	// /proc/diskstats format:
	// major minor name reads reads_merged sectors_read time_reading writes writes_merged sectors_written time_writing ...
	fields := strings.Fields(line)
	if len(fields) < 14 {
		return nil, fmt.Errorf("unexpected /proc/diskstats format: %s", line)
	}

	stats := &DiskIOStats{
		ReadCount:  parseUint64(fields[3]),
		ReadBytes:  parseUint64(fields[5]) * 512, // sectors to bytes
		WriteCount: parseUint64(fields[7]),
		WriteBytes: parseUint64(fields[9]) * 512, // sectors to bytes
	}

	return stats, nil
}
