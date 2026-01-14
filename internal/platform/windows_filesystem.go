//go:build windows
// +build windows

package platform

import (
	"fmt"
	"path/filepath"
	"syscall"
	"unsafe"
)

var (
	modKernel32GetDiskFreeSpaceEx   = modKernel32.NewProc("GetDiskFreeSpaceExW")
	modKernel32GetLogicalDrives     = modKernel32.NewProc("GetLogicalDrives")
	modKernel32GetVolumeInformation = modKernel32.NewProc("GetVolumeInformationW")
)

// windowsFilesystemProvider implements FilesystemProvider for Windows systems
type windowsFilesystemProvider struct{}

func newWindowsFilesystemProvider() *windowsFilesystemProvider {
	return &windowsFilesystemProvider{}
}

// getLogicalDrives returns a list of available drive letters
func (f *windowsFilesystemProvider) getLogicalDrives() ([]string, error) {
	ret, _, err := modKernel32GetLogicalDrives.Call()
	if ret == 0 {
		return nil, fmt.Errorf("GetLogicalDrives failed: %w", err)
	}

	drives := make([]string, 0)
	for i := 0; i < 26; i++ {
		if ret&(1<<uint(i)) != 0 {
			drive := string(rune('A'+i)) + ":\\"
			drives = append(drives, drive)
		}
	}

	return drives, nil
}

// getVolumeInformation retrieves filesystem type for a drive
func (f *windowsFilesystemProvider) getVolumeInformation(drive string) (string, error) {
	drivePtr, err := syscall.UTF16PtrFromString(drive)
	if err != nil {
		return "", err
	}

	var volumeName [syscall.MAX_PATH + 1]uint16
	var serialNumber uint32
	var maxComponentLength uint32
	var fileSystemFlags uint32
	var fileSystemName [syscall.MAX_PATH + 1]uint16

	ret, _, err := modKernel32GetVolumeInformation.Call(
		uintptr(unsafe.Pointer(drivePtr)),
		uintptr(unsafe.Pointer(&volumeName[0])),
		uintptr(len(volumeName)),
		uintptr(unsafe.Pointer(&serialNumber)),
		uintptr(unsafe.Pointer(&maxComponentLength)),
		uintptr(unsafe.Pointer(&fileSystemFlags)),
		uintptr(unsafe.Pointer(&fileSystemName[0])),
		uintptr(len(fileSystemName)),
	)

	if ret == 0 {
		return "", fmt.Errorf("GetVolumeInformation failed for %s: %w", drive, err)
	}

	return syscall.UTF16ToString(fileSystemName[:]), nil
}

func (f *windowsFilesystemProvider) Mounts() ([]MountInfo, error) {
	drives, err := f.getLogicalDrives()
	if err != nil {
		return nil, err
	}

	mounts := make([]MountInfo, 0, len(drives))
	for _, drive := range drives {
		fsType, err := f.getVolumeInformation(drive)
		if err != nil {
			// Skip drives that can't be queried (e.g., empty CD drives)
			continue
		}

		mounts = append(mounts, MountInfo{
			Device:     drive,
			MountPoint: drive,
			FSType:     fsType,
			Options:    []string{},
		})
	}

	return mounts, nil
}

func (f *windowsFilesystemProvider) Stats(mountPoint string) (*FilesystemStats, error) {
	// Normalize the path to a drive letter
	// On Windows, mount points are drive letters like "C:" or "C:\"
	drive := filepath.VolumeName(mountPoint)
	if drive == "" {
		return nil, fmt.Errorf("invalid mount point: %s (Windows mount points must be drive letters like C: or C:\\)", mountPoint)
	}

	// Ensure drive ends with backslash
	if drive[len(drive)-1] != '\\' {
		drive += "\\"
	}

	drivePtr, err := syscall.UTF16PtrFromString(drive)
	if err != nil {
		return nil, err
	}

	var freeBytesAvailable uint64
	var totalBytes uint64
	var totalFreeBytes uint64

	ret, _, err := modKernel32GetDiskFreeSpaceEx.Call(
		uintptr(unsafe.Pointer(drivePtr)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&totalFreeBytes)),
	)

	if ret == 0 {
		return nil, fmt.Errorf("GetDiskFreeSpaceEx failed for %s: %w", drive, err)
	}

	var used uint64
	if totalBytes >= totalFreeBytes {
		used = totalBytes - totalFreeBytes
	}

	var usedPercent float64
	if totalBytes > 0 {
		usedPercent = float64(used) / float64(totalBytes) * 100.0
	}

	return &FilesystemStats{
		Total:       totalBytes,
		Used:        used,
		Free:        totalFreeBytes,
		UsedPercent: usedPercent,
		InodesTotal: 0, // Windows doesn't expose inode information
		InodesUsed:  0,
		InodesFree:  0,
	}, nil
}

func (f *windowsFilesystemProvider) DiskIO(device string) (*DiskIOStats, error) {
	// Disk I/O statistics would require performance counters or WMI
	// Return empty stats for now
	return &DiskIOStats{
		ReadBytes:  0,
		WriteBytes: 0,
		ReadCount:  0,
		WriteCount: 0,
		ReadTime:   0,
		WriteTime:  0,
	}, nil
}
