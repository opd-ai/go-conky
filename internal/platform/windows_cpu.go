//go:build windows
// +build windows

package platform

import (
	"fmt"
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

var (
	modPdh                          = syscall.NewLazyDLL("pdh.dll")
	procPdhOpenQueryW               = modPdh.NewProc("PdhOpenQueryW")
	procPdhAddCounterW              = modPdh.NewProc("PdhAddCounterW")
	procPdhCollectQueryData         = modPdh.NewProc("PdhCollectQueryData")
	procPdhGetFormattedCounterValue = modPdh.NewProc("PdhGetFormattedCounterValue")
	procPdhCloseQuery               = modPdh.NewProc("PdhCloseQuery")
)

const (
	PDH_FMT_DOUBLE     = 0x00000200
	PDH_INVALID_DATA   = 0xC0000BC6
	PDH_INVALID_HANDLE = 0xC0000BBC
	PDH_NO_DATA        = 0x800007D5
)

// pdhCounterValue matches the Windows PDH_FMT_COUNTERVALUE structure for double values
type pdhCounterValue struct {
	CStatus     uint32
	DoubleValue float64
}

// windowsCPUProvider implements CPUProvider for Windows systems using PDH API
type windowsCPUProvider struct {
	mu             sync.Mutex
	query          uintptr
	counterTotal   uintptr
	countersPerCPU []uintptr
	lastSample     time.Time
	initialized    bool
}

func newWindowsCPUProvider() *windowsCPUProvider {
	c := &windowsCPUProvider{}
	// Set a finalizer to ensure PDH queries are cleaned up
	runtime.SetFinalizer(c, func(provider *windowsCPUProvider) {
		provider.Close()
	})
	return c
}

// Close releases PDH query resources
func (c *windowsCPUProvider) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closeQuery()
}

// initialize sets up the PDH query and counters
func (c *windowsCPUProvider) initialize() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.initialized {
		return nil
	}

	// Open PDH query
	var query uintptr
	ret, _, _ := procPdhOpenQueryW.Call(0, 0, uintptr(unsafe.Pointer(&query)))
	if ret != 0 {
		return fmt.Errorf("PdhOpenQuery failed with status 0x%x", ret)
	}
	c.query = query

	// Add total CPU counter
	counterPath, err := syscall.UTF16PtrFromString("\\Processor(_Total)\\% Processor Time")
	if err != nil {
		c.closeQuery()
		return fmt.Errorf("failed to create counter path string: %w", err)
	}
	var counterTotal uintptr
	ret, _, _ = procPdhAddCounterW.Call(query, uintptr(unsafe.Pointer(counterPath)), 0, uintptr(unsafe.Pointer(&counterTotal)))
	if ret != 0 {
		c.closeQuery()
		return fmt.Errorf("PdhAddCounter for total CPU failed with status 0x%x", ret)
	}
	c.counterTotal = counterTotal

	// Add per-CPU counters
	numCPU := runtime.NumCPU()
	c.countersPerCPU = make([]uintptr, numCPU)
	for i := 0; i < numCPU; i++ {
		path := fmt.Sprintf("\\Processor(%d)\\%% Processor Time", i)
		counterPath, err := syscall.UTF16PtrFromString(path)
		if err != nil {
			c.closeQuery()
			return fmt.Errorf("failed to create counter path string for CPU %d: %w", i, err)
		}
		var counter uintptr
		ret, _, _ = procPdhAddCounterW.Call(query, uintptr(unsafe.Pointer(counterPath)), 0, uintptr(unsafe.Pointer(&counter)))
		if ret != 0 {
			c.closeQuery()
			return fmt.Errorf("PdhAddCounter for CPU %d failed with status 0x%x", i, ret)
		}
		c.countersPerCPU[i] = counter
	}

	// Collect initial sample (needed for subsequent samples to be valid)
	ret, _, _ = procPdhCollectQueryData.Call(query)
	if ret != 0 {
		c.closeQuery()
		return fmt.Errorf("initial PdhCollectQueryData failed with status 0x%x", ret)
	}

	c.lastSample = time.Now()
	c.initialized = true
	return nil
}

func (c *windowsCPUProvider) closeQuery() {
	if c.query != 0 {
		procPdhCloseQuery.Call(c.query)
		c.query = 0
		c.initialized = false
	}
}

// collectSample collects a new PDH sample if enough time has passed
func (c *windowsCPUProvider) collectSample() error {
	// Wait at least 100ms between samples to get meaningful data. If called
	// more frequently, reuse the previous sample instead of blocking.
	// Skip if we have a recent sample (within 100ms)
	if !c.lastSample.IsZero() && time.Since(c.lastSample) < 100*time.Millisecond {
		return nil
	}

	ret, _, _ := procPdhCollectQueryData.Call(c.query)
	if ret != 0 {
		return fmt.Errorf("PdhCollectQueryData failed with status 0x%x", ret)
	}
	c.lastSample = time.Now()
	return nil
}

func (c *windowsCPUProvider) getCounterValue(counter uintptr) (float64, error) {
	var value pdhCounterValue
	ret, _, _ := procPdhGetFormattedCounterValue.Call(counter, PDH_FMT_DOUBLE, 0, uintptr(unsafe.Pointer(&value)))
	if ret != 0 {
		return 0, fmt.Errorf("PdhGetFormattedCounterValue failed with status 0x%x", ret)
	}
	return value.DoubleValue, nil
}

func (c *windowsCPUProvider) Usage() ([]float64, error) {
	if err := c.initialize(); err != nil {
		return nil, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.collectSample(); err != nil {
		return nil, err
	}

	usage := make([]float64, len(c.countersPerCPU))
	for i, counter := range c.countersPerCPU {
		val, err := c.getCounterValue(counter)
		if err != nil {
			return nil, fmt.Errorf("getting CPU %d usage: %w", i, err)
		}
		usage[i] = val
	}

	return usage, nil
}

func (c *windowsCPUProvider) TotalUsage() (float64, error) {
	if err := c.initialize(); err != nil {
		return 0, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.collectSample(); err != nil {
		return 0, err
	}

	return c.getCounterValue(c.counterTotal)
}

func (c *windowsCPUProvider) Frequency() ([]float64, error) {
	// CPU frequency is not easily available via PDH
	// Would need to use WMI or registry queries
	// Return empty slice for now
	numCPU := runtime.NumCPU()
	return make([]float64, numCPU), nil
}

func (c *windowsCPUProvider) Info() (*CPUInfo, error) {
	// Read CPU info from registry
	// For now, return basic info from runtime
	return &CPUInfo{
		Model:     "Windows CPU",
		Vendor:    "Unknown",
		Cores:     runtime.NumCPU(),
		Threads:   runtime.NumCPU(),
		CacheSize: 0,
	}, nil
}

func (c *windowsCPUProvider) LoadAverage() (float64, float64, float64, error) {
	// Windows does not have a direct equivalent to Unix load average
	return 0, 0, 0, fmt.Errorf("load average not available on Windows")
}
