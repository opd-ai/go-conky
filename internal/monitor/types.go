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
	// IPv4Addrs contains the IPv4 addresses assigned to this interface.
	IPv4Addrs []string
	// IPv6Addrs contains the IPv6 addresses assigned to this interface.
	IPv6Addrs []string
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
	// GatewayIP is the default gateway IP address.
	GatewayIP string
	// GatewayInterface is the interface name for the default gateway.
	GatewayInterface string
	// Nameservers contains the DNS nameserver addresses from /etc/resolv.conf.
	Nameservers []string
}

// MountStats contains statistics for a single mounted filesystem.
type MountStats struct {
	// MountPoint is the filesystem mount path (e.g., "/", "/home").
	MountPoint string
	// Device is the block device path (e.g., "/dev/sda1").
	Device string
	// FSType is the filesystem type (e.g., "ext4", "xfs").
	FSType string
	// Total is the total filesystem size in bytes.
	Total uint64
	// Used is the used space in bytes.
	Used uint64
	// Free is the free space in bytes.
	Free uint64
	// Available is the available space for non-root users in bytes.
	Available uint64
	// UsagePercent is the usage as a percentage (0-100).
	UsagePercent float64
	// InodesTotal is the total number of inodes.
	InodesTotal uint64
	// InodesUsed is the number of used inodes.
	InodesUsed uint64
	// InodesFree is the number of free inodes.
	InodesFree uint64
	// InodesPercent is the inode usage as a percentage (0-100).
	InodesPercent float64
}

// FilesystemStats contains statistics for all mounted filesystems.
type FilesystemStats struct {
	// Mounts is a map of mount point to mount statistics.
	Mounts map[string]MountStats
}

// DiskStats contains I/O statistics for a single disk device.
type DiskStats struct {
	// Name is the device name (e.g., "sda", "nvme0n1").
	Name string
	// ReadsCompleted is the total number of reads completed.
	ReadsCompleted uint64
	// ReadsMerged is the number of reads merged.
	ReadsMerged uint64
	// SectorsRead is the total number of sectors read.
	SectorsRead uint64
	// ReadTimeMs is the total time spent reading in milliseconds.
	ReadTimeMs uint64
	// WritesCompleted is the total number of writes completed.
	WritesCompleted uint64
	// WritesMerged is the number of writes merged.
	WritesMerged uint64
	// SectorsWritten is the total number of sectors written.
	SectorsWritten uint64
	// WriteTimeMs is the total time spent writing in milliseconds.
	WriteTimeMs uint64
	// IOInProgress is the number of I/Os currently in progress.
	IOInProgress uint64
	// IOTimeMs is the total time spent doing I/Os in milliseconds.
	IOTimeMs uint64
	// WeightedIOTimeMs is the weighted time spent doing I/Os in milliseconds.
	WeightedIOTimeMs uint64
	// ReadBytesPerSec is the read rate in bytes per second.
	ReadBytesPerSec float64
	// WriteBytesPerSec is the write rate in bytes per second.
	WriteBytesPerSec float64
	// ReadsPerSec is the read operations per second.
	ReadsPerSec float64
	// WritesPerSec is the write operations per second.
	WritesPerSec float64
}

// DiskIOStats contains I/O statistics for all disk devices.
type DiskIOStats struct {
	// Disks is a map of device name to disk statistics.
	Disks map[string]DiskStats
}

// ProcessInfo contains information about a single process.
type ProcessInfo struct {
	// PID is the process identifier.
	PID int
	// Name is the process name (command name).
	Name string
	// State is the process state (R, S, D, Z, T, etc.).
	State string
	// CPUPercent is the process CPU usage as a system-wide percentage (0-100),
	// where 100 represents full utilization of all logical CPU cores.
	// On multi-core systems, a process fully utilizing a single core will
	// typically report approximately 100 / CPUCount.
	CPUPercent float64
	// MemPercent is the memory usage as a percentage (0-100).
	MemPercent float64
	// MemBytes is the resident set size (RSS) in bytes.
	MemBytes uint64
	// VirtBytes is the virtual memory size in bytes.
	VirtBytes uint64
	// Threads is the number of threads in the process.
	Threads int
	// StartTime is the process start time in jiffies since system boot.
	StartTime uint64
}

// ProcessStats contains process-related statistics.
type ProcessStats struct {
	// TotalProcesses is the total number of processes.
	TotalProcesses int
	// RunningProcesses is the number of processes in running state.
	RunningProcesses int
	// SleepingProcesses is the number of processes in sleeping state.
	SleepingProcesses int
	// ZombieProcesses is the number of zombie processes.
	ZombieProcesses int
	// StoppedProcesses is the number of stopped processes.
	StoppedProcesses int
	// TotalThreads is the total number of threads across all processes.
	TotalThreads int
	// TopCPU contains the top processes by CPU usage.
	TopCPU []ProcessInfo
	// TopMem contains the top processes by memory usage.
	TopMem []ProcessInfo
}

// SystemData aggregates all system monitoring data.
type SystemData struct {
	CPU        CPUStats
	Memory     MemoryStats
	Uptime     UptimeStats
	Network    NetworkStats
	Filesystem FilesystemStats
	DiskIO     DiskIOStats
	Hwmon      HwmonStats
	Process    ProcessStats
	Battery    BatteryStats
	Audio      AudioStats
	SysInfo    SystemInfo
	mu         sync.RWMutex
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
	return sd.copyNetwork()
}

// copyNetwork returns a deep copy of the network statistics.
// Caller must hold at least a read lock on sd.mu.
func (sd *SystemData) copyNetwork() NetworkStats {
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

// GetFilesystem returns a copy of the filesystem statistics with proper locking.
func (sd *SystemData) GetFilesystem() FilesystemStats {
	sd.mu.RLock()
	defer sd.mu.RUnlock()
	return sd.copyFilesystem()
}

// copyFilesystem returns a deep copy of the filesystem statistics.
// Caller must hold at least a read lock on sd.mu.
func (sd *SystemData) copyFilesystem() FilesystemStats {
	result := FilesystemStats{
		Mounts: make(map[string]MountStats, len(sd.Filesystem.Mounts)),
	}
	for k, v := range sd.Filesystem.Mounts {
		result.Mounts[k] = v
	}
	return result
}

// setFilesystem updates the filesystem statistics with proper locking.
func (sd *SystemData) setFilesystem(fs FilesystemStats) {
	sd.mu.Lock()
	defer sd.mu.Unlock()
	sd.Filesystem = fs
}

// GetDiskIO returns a copy of the disk I/O statistics with proper locking.
func (sd *SystemData) GetDiskIO() DiskIOStats {
	sd.mu.RLock()
	defer sd.mu.RUnlock()
	return sd.copyDiskIO()
}

// copyDiskIO returns a deep copy of the disk I/O statistics.
// Caller must hold at least a read lock on sd.mu.
func (sd *SystemData) copyDiskIO() DiskIOStats {
	result := DiskIOStats{
		Disks: make(map[string]DiskStats, len(sd.DiskIO.Disks)),
	}
	for k, v := range sd.DiskIO.Disks {
		result.Disks[k] = v
	}
	return result
}

// setDiskIO updates the disk I/O statistics with proper locking.
func (sd *SystemData) setDiskIO(diskIO DiskIOStats) {
	sd.mu.Lock()
	defer sd.mu.Unlock()
	sd.DiskIO = diskIO
}

// GetHwmon returns a copy of the hardware monitoring statistics with proper locking.
func (sd *SystemData) GetHwmon() HwmonStats {
	sd.mu.RLock()
	defer sd.mu.RUnlock()
	return sd.copyHwmon()
}

// copyHwmon returns a deep copy of the hardware monitoring statistics.
// Caller must hold at least a read lock on sd.mu.
func (sd *SystemData) copyHwmon() HwmonStats {
	result := HwmonStats{
		Devices:     make(map[string]HwmonDevice, len(sd.Hwmon.Devices)),
		TempSensors: make([]TempSensor, len(sd.Hwmon.TempSensors)),
	}
	for k, v := range sd.Hwmon.Devices {
		// Deep copy the device including its temps map
		deviceCopy := HwmonDevice{
			Name:  v.Name,
			Path:  v.Path,
			Temps: make(map[string]TempSensor, len(v.Temps)),
		}
		for tk, tv := range v.Temps {
			deviceCopy.Temps[tk] = tv
		}
		result.Devices[k] = deviceCopy
	}
	copy(result.TempSensors, sd.Hwmon.TempSensors)
	return result
}

// setHwmon updates the hardware monitoring statistics with proper locking.
func (sd *SystemData) setHwmon(hwmon HwmonStats) {
	sd.mu.Lock()
	defer sd.mu.Unlock()
	sd.Hwmon = hwmon
}

// GetProcess returns a copy of the process statistics with proper locking.
func (sd *SystemData) GetProcess() ProcessStats {
	sd.mu.RLock()
	defer sd.mu.RUnlock()
	return sd.copyProcess()
}

// copyProcess returns a deep copy of the process statistics.
// Caller must hold at least a read lock on sd.mu.
func (sd *SystemData) copyProcess() ProcessStats {
	result := ProcessStats{
		TotalProcesses:    sd.Process.TotalProcesses,
		RunningProcesses:  sd.Process.RunningProcesses,
		SleepingProcesses: sd.Process.SleepingProcesses,
		ZombieProcesses:   sd.Process.ZombieProcesses,
		StoppedProcesses:  sd.Process.StoppedProcesses,
		TotalThreads:      sd.Process.TotalThreads,
		TopCPU:            make([]ProcessInfo, len(sd.Process.TopCPU)),
		TopMem:            make([]ProcessInfo, len(sd.Process.TopMem)),
	}
	copy(result.TopCPU, sd.Process.TopCPU)
	copy(result.TopMem, sd.Process.TopMem)
	return result
}

// setProcess updates the process statistics with proper locking.
func (sd *SystemData) setProcess(process ProcessStats) {
	sd.mu.Lock()
	defer sd.mu.Unlock()
	sd.Process = process
}

// GetBattery returns a copy of the battery statistics with proper locking.
func (sd *SystemData) GetBattery() BatteryStats {
	sd.mu.RLock()
	defer sd.mu.RUnlock()
	return sd.copyBattery()
}

// copyBattery returns a deep copy of the battery statistics.
// Caller must hold at least a read lock on sd.mu.
func (sd *SystemData) copyBattery() BatteryStats {
	result := BatteryStats{
		Batteries:       make(map[string]BatteryInfo, len(sd.Battery.Batteries)),
		ACAdapters:      make(map[string]ACAdapterInfo, len(sd.Battery.ACAdapters)),
		ACOnline:        sd.Battery.ACOnline,
		TotalCapacity:   sd.Battery.TotalCapacity,
		TotalEnergyNow:  sd.Battery.TotalEnergyNow,
		TotalEnergyFull: sd.Battery.TotalEnergyFull,
		IsCharging:      sd.Battery.IsCharging,
		IsDischarging:   sd.Battery.IsDischarging,
	}
	for k, v := range sd.Battery.Batteries {
		result.Batteries[k] = v
	}
	for k, v := range sd.Battery.ACAdapters {
		result.ACAdapters[k] = v
	}
	return result
}

// setBattery updates the battery statistics with proper locking.
func (sd *SystemData) setBattery(battery BatteryStats) {
	sd.mu.Lock()
	defer sd.mu.Unlock()
	sd.Battery = battery
}

// GetAudio returns a copy of the audio statistics with proper locking.
func (sd *SystemData) GetAudio() AudioStats {
	sd.mu.RLock()
	defer sd.mu.RUnlock()
	return sd.copyAudio()
}

// copyAudio returns a deep copy of the audio statistics.
// Caller must hold at least a read lock on sd.mu.
func (sd *SystemData) copyAudio() AudioStats {
	result := AudioStats{
		Cards:        make(map[int]AudioCard, len(sd.Audio.Cards)),
		DefaultCard:  sd.Audio.DefaultCard,
		MasterVolume: sd.Audio.MasterVolume,
		MasterMuted:  sd.Audio.MasterMuted,
		HasAudio:     sd.Audio.HasAudio,
	}
	for k, v := range sd.Audio.Cards {
		// Deep copy the card including its mixers map
		cardCopy := AudioCard{
			Index:  v.Index,
			ID:     v.ID,
			Name:   v.Name,
			Driver: v.Driver,
			Mixers: make(map[string]MixerInfo, len(v.Mixers)),
		}
		for mk, mv := range v.Mixers {
			cardCopy.Mixers[mk] = mv
		}
		result.Cards[k] = cardCopy
	}
	return result
}

// setAudio updates the audio statistics with proper locking.
func (sd *SystemData) setAudio(audio AudioStats) {
	sd.mu.Lock()
	defer sd.mu.Unlock()
	sd.Audio = audio
}

// GetSysInfo returns a copy of the system info with proper locking.
func (sd *SystemData) GetSysInfo() SystemInfo {
	sd.mu.RLock()
	defer sd.mu.RUnlock()
	return sd.SysInfo
}

// setSysInfo updates the system info with proper locking.
func (sd *SystemData) setSysInfo(sysInfo SystemInfo) {
	sd.mu.Lock()
	defer sd.mu.Unlock()
	sd.SysInfo = sysInfo
}

// copySysInfo returns a copy of the system info.
// Caller must hold at least a read lock on sd.mu.
func (sd *SystemData) copySysInfo() SystemInfo {
	return sd.SysInfo
}
