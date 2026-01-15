//go:build darwin
// +build darwin

package platform

import (
	"fmt"
	"runtime"
	"sync"
	"syscall"
	"unsafe"
)

// Sysctl MIB constants for Darwin
const (
	ctlKern    = 1  // CTL_KERN
	ctlVM      = 2  // CTL_VM
	kernCPTime = 82 // KERN_CP_TIME
	vmLoadavg  = 2  // VM_LOADAVG
)

// darwinCPUProvider implements CPUProvider for macOS/Darwin systems using sysctl and mach APIs.
type darwinCPUProvider struct {
	mu        sync.Mutex
	prevStats cpuTimes
}

func newDarwinCPUProvider() *darwinCPUProvider {
	return &darwinCPUProvider{}
}

// Usage returns CPU usage percentages for all cores.
// Note: Per-core CPU usage is not easily available on macOS without mach APIs,
// so we return the total usage for all cores.
func (c *darwinCPUProvider) Usage() ([]float64, error) {
	// macOS doesn't provide per-CPU usage easily via sysctl
	// Return total usage for each logical CPU
	totalUsage, err := c.TotalUsage()
	if err != nil {
		return nil, err
	}

	numCPU := runtime.NumCPU()
	usages := make([]float64, numCPU)
	for i := 0; i < numCPU; i++ {
		usages[i] = totalUsage
	}

	return usages, nil
}

// TotalUsage returns the aggregate CPU usage percentage.
func (c *darwinCPUProvider) TotalUsage() (float64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	current, err := c.getHostCPULoadInfo()
	if err != nil {
		return 0, err
	}

	prev := c.prevStats
	c.prevStats = current

	// On first call, we don't have previous stats
	if prev.user == 0 && prev.system == 0 && prev.idle == 0 {
		return 0, nil
	}

	totalDelta := float64(
		(current.user - prev.user) +
			(current.system - prev.system) +
			(current.idle - prev.idle) +
			(current.nice - prev.nice))

	idleDelta := float64(current.idle - prev.idle)

	if totalDelta > 0 {
		return 100 * (1 - idleDelta/totalDelta), nil
	}

	return 0, nil
}

// Frequency returns CPU frequencies in MHz for all cores.
// macOS doesn't expose per-core frequencies easily, so we return the base frequency for all cores.
func (c *darwinCPUProvider) Frequency() ([]float64, error) {
	freq, err := sysctlUint64("hw.cpufrequency")
	if err != nil {
		// Fallback to hw.cpufrequency_max if hw.cpufrequency is not available
		freq, err = sysctlUint64("hw.cpufrequency_max")
		if err != nil {
			return nil, fmt.Errorf("getting CPU frequency: %w", err)
		}
	}

	// Convert Hz to MHz
	freqMHz := float64(freq) / 1000000.0

	numCPU := runtime.NumCPU()
	frequencies := make([]float64, numCPU)
	for i := 0; i < numCPU; i++ {
		frequencies[i] = freqMHz
	}

	return frequencies, nil
}

// Info returns static CPU information (model, cores, etc.).
func (c *darwinCPUProvider) Info() (*CPUInfo, error) {
	brand, err := sysctlString("machdep.cpu.brand_string")
	if err != nil {
		return nil, fmt.Errorf("getting CPU brand: %w", err)
	}

	vendor, err := sysctlString("machdep.cpu.vendor")
	if err != nil {
		// Vendor may not be available on all systems
		vendor = "Unknown"
	}

	cores, err := sysctlUint64("hw.physicalcpu")
	if err != nil {
		return nil, fmt.Errorf("getting physical CPU count: %w", err)
	}

	threads, err := sysctlUint64("hw.logicalcpu")
	if err != nil {
		return nil, fmt.Errorf("getting logical CPU count: %w", err)
	}

	cacheSize, err := sysctlUint64("hw.l3cachesize")
	if err != nil {
		// L3 cache may not be available, try L2
		cacheSize, err = sysctlUint64("hw.l2cachesize")
		if err != nil {
			// If neither is available, set to 0
			cacheSize = 0
		}
	}

	return &CPUInfo{
		Model:     brand,
		Vendor:    vendor,
		Cores:     int(cores),
		Threads:   int(threads),
		CacheSize: int64(cacheSize),
	}, nil
}

// LoadAverage returns 1, 5, and 15 minute load averages.
func (c *darwinCPUProvider) LoadAverage() (float64, float64, float64, error) {
	type loadavg struct {
		load  [3]uint32
		scale int
	}

	mib := []int32{ctlKern, vmLoadavg}

	var la loadavg
	n := uintptr(unsafe.Sizeof(la))

	_, _, errno := syscall.Syscall6(
		syscall.SYS___SYSCTL,
		uintptr(unsafe.Pointer(&mib[0])),
		uintptr(len(mib)),
		uintptr(unsafe.Pointer(&la)),
		uintptr(unsafe.Pointer(&n)),
		0,
		0,
	)

	if errno != 0 {
		return 0, 0, 0, fmt.Errorf("sysctl VM_LOADAVG failed: %w", errno)
	}

	scale := float64(la.scale)
	if scale == 0 {
		scale = 2048.0 // Default LOADAVG_SCALE on macOS
	}

	load1 := float64(la.load[0]) / scale
	load5 := float64(la.load[1]) / scale
	load15 := float64(la.load[2]) / scale

	return load1, load5, load15, nil
}

// getHostCPULoadInfo retrieves CPU time statistics using sysctl.
// This uses the host_statistics64 mach call via sysctl.
func (c *darwinCPUProvider) getHostCPULoadInfo() (cpuTimes, error) {
	// Try to get CPU times using sysctl kern.cp_time
	// This is an array of [user, nice, system, idle]
	mib := []int32{ctlKern, kernCPTime}

	var times [4]uint64
	n := uintptr(unsafe.Sizeof(times))

	_, _, errno := syscall.Syscall6(
		syscall.SYS___SYSCTL,
		uintptr(unsafe.Pointer(&mib[0])),
		uintptr(len(mib)),
		uintptr(unsafe.Pointer(&times[0])),
		uintptr(unsafe.Pointer(&n)),
		0,
		0,
	)

	if errno != 0 {
		return cpuTimes{}, fmt.Errorf("sysctl KERN_CP_TIME failed: %w", errno)
	}

	return cpuTimes{
		user:   times[0],
		nice:   times[1],
		system: times[2],
		idle:   times[3],
	}, nil
}

// sysctlUint64 retrieves a uint64 value from sysctl by name.
func sysctlUint64(name string) (uint64, error) {
	// Use syscall.Sysctl to get the value as a string, then parse it
	// This is more compatible than syscall.SysctlUint64 which may not be available
	valueStr, err := syscall.Sysctl(name)
	if err != nil {
		return 0, fmt.Errorf("sysctl %s: %w", name, err)
	}

	// Try to parse as uint64
	var value uint64
	_, err = fmt.Sscanf(valueStr, "%d", &value)
	if err != nil {
		return 0, fmt.Errorf("parsing sysctl %s value %q: %w", name, valueStr, err)
	}
	return value, nil
}

// sysctlString retrieves a string value from sysctl by name.
func sysctlString(name string) (string, error) {
	value, err := syscall.Sysctl(name)
	if err != nil {
		return "", fmt.Errorf("sysctl %s: %w", name, err)
	}
	return value, nil
}
