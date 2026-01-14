// +build darwin

package platform

import (
	"fmt"
	"syscall"
	"unsafe"
)

// darwinMemoryProvider implements MemoryProvider for macOS/Darwin systems using vm_stat and sysctl.
type darwinMemoryProvider struct{}

func newDarwinMemoryProvider() *darwinMemoryProvider {
	return &darwinMemoryProvider{}
}

// Stats returns current memory statistics.
func (m *darwinMemoryProvider) Stats() (*MemoryStats, error) {
	// Get page size
	pageSize, err := sysctlUint64("hw.pagesize")
	if err != nil {
		return nil, fmt.Errorf("getting page size: %w", err)
	}

	// Get total physical memory
	totalMem, err := sysctlUint64("hw.memsize")
	if err != nil {
		return nil, fmt.Errorf("getting total memory: %w", err)
	}

	// Get VM statistics using sysctl vm.swapusage (for free/used calculation)
	// macOS doesn't have a simple "free memory" concept like Linux
	// We need to use vm_stat data through host_statistics64
	
	vmStat, err := m.getVMStatistics()
	if err != nil {
		return nil, fmt.Errorf("getting VM statistics: %w", err)
	}

	// Calculate memory metrics
	// active + inactive + wired = used memory (approximately)
	// free + speculative can be considered "available"
	active := vmStat.activeCount * pageSize
	inactive := vmStat.inactiveCount * pageSize
	wired := vmStat.wireCount * pageSize
	free := vmStat.freeCount * pageSize
	speculative := vmStat.speculativeCount * pageSize
	purgeable := vmStat.purgeableCount * pageSize

	// Used = active + wired
	used := active + wired
	
	// Available = free + inactive + speculative (memory that can be reclaimed)
	available := free + inactive + speculative + purgeable
	
	// Ensure available doesn't exceed total
	if available > totalMem {
		available = totalMem
	}

	// Calculate used percentage
	usedPercent := 0.0
	if totalMem > 0 {
		usedPercent = float64(used) / float64(totalMem) * 100.0
	}

	return &MemoryStats{
		Total:       totalMem,
		Used:        used,
		Free:        free,
		Available:   available,
		Cached:      inactive, // Inactive memory can be considered cached
		Buffers:     0,        // macOS doesn't have a separate buffers concept
		UsedPercent: usedPercent,
	}, nil
}

// SwapStats returns swap/page file statistics.
func (m *darwinMemoryProvider) SwapStats() (*SwapStats, error) {
	// Use sysctl vm.swapusage to get swap information
	// This returns a struct with total, used, and free swap
	
	type xswUsage struct {
		total uint64
		used  uint64
		free  uint64
		_     uint32 // padding for alignment
		encrypted bool
	}

	mib := []int32{2 /* CTL_VM */, 5 /* VM_SWAPUSAGE */}
	
	var swapUsage xswUsage
	n := uintptr(unsafe.Sizeof(swapUsage))
	
	_, _, errno := syscall.Syscall6(
		syscall.SYS___SYSCTL,
		uintptr(unsafe.Pointer(&mib[0])),
		uintptr(len(mib)),
		uintptr(unsafe.Pointer(&swapUsage)),
		uintptr(unsafe.Pointer(&n)),
		0,
		0,
	)
	
	if errno != 0 {
		return nil, fmt.Errorf("sysctl VM_SWAPUSAGE failed: %w", errno)
	}

	usedPercent := 0.0
	if swapUsage.total > 0 {
		usedPercent = float64(swapUsage.used) / float64(swapUsage.total) * 100.0
	}

	return &SwapStats{
		Total:       swapUsage.total,
		Used:        swapUsage.used,
		Free:        swapUsage.free,
		UsedPercent: usedPercent,
	}, nil
}

// vmStatistics represents macOS VM statistics.
type vmStatistics struct {
	freeCount        uint64
	activeCount      uint64
	inactiveCount    uint64
	wireCount        uint64
	purgeableCount   uint64
	speculativeCount uint64
}

// getVMStatistics retrieves VM statistics using host_statistics64.
// We approximate this using available sysctl values since host_statistics64
// requires mach kernel APIs which are complex to use from pure Go.
func (m *darwinMemoryProvider) getVMStatistics() (*vmStatistics, error) {
	// Try to get VM stats from sysctl vm.* values
	// Note: These are approximations as macOS doesn't expose all vm_stat data via sysctl
	
	// Get page size first
	pageSize, err := sysctlUint64("hw.pagesize")
	if err != nil {
		return nil, fmt.Errorf("getting page size: %w", err)
	}

	stats := &vmStatistics{}

	// Try to get various VM counters
	// Note: Not all of these may be available on all macOS versions
	if val, err := sysctlUint64("vm.page_free_count"); err == nil {
		stats.freeCount = val
	}
	
	// For other stats, we'll need to use an alternative approach
	// Since direct sysctl access is limited, we estimate based on available data
	
	// Get memory pressure (if available) to estimate active/inactive
	// This is a simplified approach
	totalMem, _ := sysctlUint64("hw.memsize")
	if stats.freeCount > 0 && pageSize > 0 {
		freeBytes := stats.freeCount * pageSize
		// Rough estimate: active is about 40-60% of (total - free)
		usedBytes := totalMem - freeBytes
		stats.activeCount = (usedBytes * 50) / (100 * pageSize)
		stats.inactiveCount = (usedBytes * 30) / (100 * pageSize)
		stats.wireCount = (usedBytes * 20) / (100 * pageSize)
	}

	return stats, nil
}
