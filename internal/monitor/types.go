// Package monitor provides system monitoring functionality for Linux systems.
// It collects CPU, memory, uptime, and other system statistics by parsing
// data from the /proc filesystem.
package monitor

import (
	"sync"
	"time"
)

// CPUStats contains CPU usage statistics.
type CPUStats struct {
	// UsagePercent is the overall CPU usage as a percentage (0-100).
	UsagePercent float64
	// Cores contains per-core usage percentages.
	Cores []float64
	// CPUCount is the number of logical CPU cores.
	CPUCount int
	// ModelName is the CPU model name from /proc/cpuinfo.
	ModelName string
	// Frequency is the current CPU frequency in MHz.
	Frequency float64
}

// MemoryStats contains memory usage statistics.
type MemoryStats struct {
	// Total is the total physical memory in bytes.
	Total uint64
	// Used is the used memory in bytes.
	Used uint64
	// Free is the free memory in bytes.
	Free uint64
	// Available is the available memory in bytes.
	Available uint64
	// Buffers is the memory used for buffers in bytes.
	Buffers uint64
	// Cached is the memory used for cache in bytes.
	Cached uint64
	// SwapTotal is the total swap memory in bytes.
	SwapTotal uint64
	// SwapUsed is the used swap memory in bytes.
	SwapUsed uint64
	// SwapFree is the free swap memory in bytes.
	SwapFree uint64
	// UsagePercent is the memory usage as a percentage (0-100).
	UsagePercent float64
	// SwapPercent is the swap usage as a percentage (0-100).
	SwapPercent float64
}

// UptimeStats contains system uptime information.
type UptimeStats struct {
	// Duration is the system uptime as a time.Duration.
	Duration time.Duration
	// Seconds is the system uptime in seconds.
	Seconds float64
	// IdleSeconds is the cumulative idle time of all CPUs in seconds.
	IdleSeconds float64
}

// InterfaceStats contains statistics for a single network interface.
type InterfaceStats struct {
	// Name is the interface name (e.g., "eth0", "lo").
	Name string
	// RxBytes is the total bytes received.
	RxBytes uint64
	// RxPackets is the total packets received.
	RxPackets uint64
	// RxErrors is the total receive errors.
	RxErrors uint64
	// RxDropped is the total receive packets dropped.
	RxDropped uint64
	// TxBytes is the total bytes transmitted.
	TxBytes uint64
	// TxPackets is the total packets transmitted.
	TxPackets uint64
	// TxErrors is the total transmit errors.
	TxErrors uint64
	// TxDropped is the total transmit packets dropped.
	TxDropped uint64
	// RxBytesPerSec is the receive rate in bytes per second.
	RxBytesPerSec float64
	// TxBytesPerSec is the transmit rate in bytes per second.
	TxBytesPerSec float64
}

// NetworkStats contains network interface statistics.
type NetworkStats struct {
	// Interfaces is a map of interface name to interface statistics.
	Interfaces map[string]InterfaceStats
	// TotalRxBytes is the sum of RxBytes across all interfaces.
	TotalRxBytes uint64
	// TotalTxBytes is the sum of TxBytes across all interfaces.
	TotalTxBytes uint64
	// TotalRxBytesPerSec is the sum of RxBytesPerSec across all interfaces.
	TotalRxBytesPerSec float64
	// TotalTxBytesPerSec is the sum of TxBytesPerSec across all interfaces.
	TotalTxBytesPerSec float64
}

// SystemData aggregates all system monitoring data.
type SystemData struct {
	CPU     CPUStats
	Memory  MemoryStats
	Uptime  UptimeStats
	Network NetworkStats
	mu      sync.RWMutex
}

// NewSystemData creates a new SystemData instance.
func NewSystemData() *SystemData {
	return &SystemData{}
}

// GetCPU returns a copy of the CPU statistics with proper locking.
func (sd *SystemData) GetCPU() CPUStats {
	sd.mu.RLock()
	defer sd.mu.RUnlock()
	return sd.CPU
}

// GetMemory returns a copy of the memory statistics with proper locking.
func (sd *SystemData) GetMemory() MemoryStats {
	sd.mu.RLock()
	defer sd.mu.RUnlock()
	return sd.Memory
}

// GetUptime returns a copy of the uptime statistics with proper locking.
func (sd *SystemData) GetUptime() UptimeStats {
	sd.mu.RLock()
	defer sd.mu.RUnlock()
	return sd.Uptime
}

// GetNetwork returns a copy of the network statistics with proper locking.
func (sd *SystemData) GetNetwork() NetworkStats {
	sd.mu.RLock()
	defer sd.mu.RUnlock()
	// Return a deep copy of the interfaces map
	result := NetworkStats{
		Interfaces:         make(map[string]InterfaceStats, len(sd.Network.Interfaces)),
		TotalRxBytes:       sd.Network.TotalRxBytes,
		TotalTxBytes:       sd.Network.TotalTxBytes,
		TotalRxBytesPerSec: sd.Network.TotalRxBytesPerSec,
		TotalTxBytesPerSec: sd.Network.TotalTxBytesPerSec,
	}
	for k, v := range sd.Network.Interfaces {
		result.Interfaces[k] = v
	}
	return result
}

// setCPU updates the CPU statistics with proper locking.
func (sd *SystemData) setCPU(cpu CPUStats) {
	sd.mu.Lock()
	defer sd.mu.Unlock()
	sd.CPU = cpu
}

// setMemory updates the memory statistics with proper locking.
func (sd *SystemData) setMemory(mem MemoryStats) {
	sd.mu.Lock()
	defer sd.mu.Unlock()
	sd.Memory = mem
}

// setUptime updates the uptime statistics with proper locking.
func (sd *SystemData) setUptime(uptime UptimeStats) {
	sd.mu.Lock()
	defer sd.mu.Unlock()
	sd.Uptime = uptime
}

// setNetwork updates the network statistics with proper locking.
func (sd *SystemData) setNetwork(network NetworkStats) {
	sd.mu.Lock()
	defer sd.mu.Unlock()
	sd.Network = network
}
