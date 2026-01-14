// +build windows

package platform

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	modKernel32              = syscall.NewLazyDLL("kernel32.dll")
	procGlobalMemoryStatusEx = modKernel32.NewProc("GlobalMemoryStatusEx")
)

// memoryStatusEx matches the Windows MEMORYSTATUSEX structure
type memoryStatusEx struct {
	dwLength                uint32
	dwMemoryLoad            uint32
	ullTotalPhys            uint64
	ullAvailPhys            uint64
	ullTotalPageFile        uint64
	ullAvailPageFile        uint64
	ullTotalVirtual         uint64
	ullAvailVirtual         uint64
	ullAvailExtendedVirtual uint64
}

// windowsMemoryProvider implements MemoryProvider for Windows systems
// using GlobalMemoryStatusEx API
type windowsMemoryProvider struct{}

func newWindowsMemoryProvider() *windowsMemoryProvider {
	return &windowsMemoryProvider{}
}

// getMemoryStatus retrieves the current memory status from Windows API
func (m *windowsMemoryProvider) getMemoryStatus() (*memoryStatusEx, error) {
	var memStatus memoryStatusEx
	memStatus.dwLength = uint32(unsafe.Sizeof(memStatus))

	ret, _, err := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&memStatus)))
	if ret == 0 {
		return nil, fmt.Errorf("GlobalMemoryStatusEx failed: %w", err)
	}

	return &memStatus, nil
}

func (m *windowsMemoryProvider) Stats() (*MemoryStats, error) {
	memStatus, err := m.getMemoryStatus()
	if err != nil {
		return nil, err
	}

	total := memStatus.ullTotalPhys
	available := memStatus.ullAvailPhys
	
	// Calculate used memory
	var used uint64
	if total >= available {
		used = total - available
	}

	return &MemoryStats{
		Total:       total,
		Used:        used,
		Free:        available,
		Available:   available,
		Cached:      0, // Windows doesn't expose cached memory directly
		Buffers:     0, // Windows doesn't expose buffer memory directly
		UsedPercent: float64(memStatus.dwMemoryLoad),
	}, nil
}

func (m *windowsMemoryProvider) SwapStats() (*SwapStats, error) {
	memStatus, err := m.getMemoryStatus()
	if err != nil {
		return nil, err
	}

	// Calculate page file (swap) size
	// Page file total includes physical memory, so we subtract it
	var pageFileTotal, pageFileAvail uint64
	if memStatus.ullTotalPageFile > memStatus.ullTotalPhys &&
		memStatus.ullAvailPageFile > memStatus.ullAvailPhys {
		pageFileTotal = memStatus.ullTotalPageFile - memStatus.ullTotalPhys
		pageFileAvail = memStatus.ullAvailPageFile - memStatus.ullAvailPhys
	} else {
		// Fallback: use total page file values when subtraction would underflow
		pageFileTotal = memStatus.ullTotalPageFile
		pageFileAvail = memStatus.ullAvailPageFile
	}

	// Ensure we don't underflow on used calculation
	var used uint64
	if pageFileTotal > pageFileAvail {
		used = pageFileTotal - pageFileAvail
	}

	var usedPercent float64
	if pageFileTotal > 0 {
		usedPercent = float64(used) / float64(pageFileTotal) * 100.0
	}

	return &SwapStats{
		Total:       pageFileTotal,
		Used:        used,
		Free:        pageFileAvail,
		UsedPercent: usedPercent,
	}, nil
}
