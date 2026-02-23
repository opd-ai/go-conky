// Package monitor provides system monitoring functionality.
// This file provides integration with the platform abstraction layer,
// enabling cross-platform system monitoring.
//
// # Platform Integration
//
// The monitor package can use the platform abstraction layer (internal/platform)
// for cross-platform system monitoring. Due to import cycles, a wrapper is needed
// to connect the two packages.
//
// Example usage from pkg/conky or cmd/conky-go:
//
//	// 1. Import both packages
//	import (
//	    "github.com/opd-ai/go-conky/internal/monitor"
//	    "github.com/opd-ai/go-conky/internal/platform"
//	)
//
//	// 2. Create a wrapper that implements monitor.PlatformInterface
//	// (see the interfaces defined below for required methods)
//
//	// 3. Create monitor with platform support
//	plat, _ := platform.NewPlatform()
//	plat.Initialize(ctx)
//	wrapped := myPlatformWrapper{plat}
//	sm := monitor.NewSystemMonitorWithPlatform(time.Second, &wrapped)
//
// The wrapper simply delegates calls from monitor.*Interface types to
// platform.* types, converting between the two type systems.
package monitor

import (
	"time"
)

// PlatformInterface defines the minimal interface needed for cross-platform monitoring.
// This mirrors the platform.Platform interface but is defined locally to avoid
// import cycles (platform -> pkg/conky -> monitor).
type PlatformInterface interface {
	// Name returns the platform identifier.
	Name() string

	// CPU returns the CPU metrics provider.
	CPU() CPUProviderInterface

	// Memory returns the memory metrics provider.
	Memory() MemoryProviderInterface

	// Network returns the network metrics provider.
	Network() NetworkProviderInterface

	// Filesystem returns the filesystem metrics provider.
	Filesystem() FilesystemProviderInterface

	// Battery returns the battery metrics provider (may be nil).
	Battery() BatteryProviderInterface

	// Sensors returns the sensors provider (may be nil).
	Sensors() SensorProviderInterface
}

// CPUProviderInterface mirrors platform.CPUProvider.
type CPUProviderInterface interface {
	Usage() ([]float64, error)
	TotalUsage() (float64, error)
	Frequency() ([]float64, error)
	Info() (*PlatformCPUInfo, error)
	LoadAverage() (float64, float64, float64, error)
}

// PlatformCPUInfo mirrors platform.CPUInfo.
type PlatformCPUInfo struct {
	Model     string
	Vendor    string
	Cores     int
	Threads   int
	CacheSize int64
}

// MemoryProviderInterface mirrors platform.MemoryProvider.
type MemoryProviderInterface interface {
	Stats() (*PlatformMemoryStats, error)
	SwapStats() (*PlatformSwapStats, error)
}

// PlatformMemoryStats mirrors platform.MemoryStats.
type PlatformMemoryStats struct {
	Total       uint64
	Used        uint64
	Free        uint64
	Available   uint64
	Cached      uint64
	Buffers     uint64
	UsedPercent float64
}

// PlatformSwapStats mirrors platform.SwapStats.
type PlatformSwapStats struct {
	Total       uint64
	Used        uint64
	Free        uint64
	UsedPercent float64
}

// NetworkProviderInterface mirrors platform.NetworkProvider.
type NetworkProviderInterface interface {
	Interfaces() ([]string, error)
	Stats(interfaceName string) (*PlatformNetworkStats, error)
	AllStats() (map[string]*PlatformNetworkStats, error)
}

// PlatformNetworkStats mirrors platform.NetworkStats.
type PlatformNetworkStats struct {
	BytesRecv   uint64
	BytesSent   uint64
	PacketsRecv uint64
	PacketsSent uint64
	ErrorsIn    uint64
	ErrorsOut   uint64
	DropIn      uint64
	DropOut     uint64
}

// FilesystemProviderInterface mirrors platform.FilesystemProvider.
type FilesystemProviderInterface interface {
	Mounts() ([]PlatformMountInfo, error)
	Stats(mountPoint string) (*PlatformFilesystemStats, error)
	DiskIO(device string) (*PlatformDiskIOStats, error)
}

// PlatformMountInfo mirrors platform.MountInfo.
type PlatformMountInfo struct {
	Device     string
	MountPoint string
	FSType     string
	Options    []string
}

// PlatformFilesystemStats mirrors platform.FilesystemStats.
type PlatformFilesystemStats struct {
	Total       uint64
	Used        uint64
	Free        uint64
	UsedPercent float64
	InodesTotal uint64
	InodesUsed  uint64
	InodesFree  uint64
}

// PlatformDiskIOStats mirrors platform.DiskIOStats.
type PlatformDiskIOStats struct {
	ReadBytes  uint64
	WriteBytes uint64
	ReadCount  uint64
	WriteCount uint64
	ReadTime   time.Duration
	WriteTime  time.Duration
}

// BatteryProviderInterface mirrors platform.BatteryProvider.
type BatteryProviderInterface interface {
	Count() int
	Stats(index int) (*PlatformBatteryStats, error)
}

// PlatformBatteryStats mirrors platform.BatteryStats.
type PlatformBatteryStats struct {
	Percent       float64
	TimeRemaining time.Duration
	Charging      bool
	FullCapacity  uint64
	Current       uint64
	Voltage       float64
}

// SensorProviderInterface mirrors platform.SensorProvider.
type SensorProviderInterface interface {
	Temperatures() ([]PlatformSensorReading, error)
	Fans() ([]PlatformSensorReading, error)
}

// PlatformSensorReading mirrors platform.SensorReading.
type PlatformSensorReading struct {
	Name     string
	Label    string
	Value    float64
	Unit     string
	Critical float64
}

// PlatformAdapter wraps a PlatformInterface and provides conversion methods
// to translate between platform types and monitor types.
type PlatformAdapter struct {
	plat PlatformInterface
}

// NewPlatformAdapter creates a new PlatformAdapter wrapping the given platform.
func NewPlatformAdapter(plat PlatformInterface) *PlatformAdapter {
	if plat == nil {
		return nil
	}
	return &PlatformAdapter{plat: plat}
}

// ReadCPUStats reads CPU statistics from the platform and converts to monitor types.
func (pa *PlatformAdapter) ReadCPUStats() (CPUStats, error) {
	cpu := pa.plat.CPU()
	if cpu == nil {
		return CPUStats{}, nil
	}

	stats := CPUStats{}

	// Get total usage
	totalUsage, err := cpu.TotalUsage()
	if err == nil {
		stats.UsagePercent = totalUsage
	}

	// Get per-core usage
	coreUsage, err := cpu.Usage()
	if err == nil {
		stats.Cores = coreUsage
		stats.CPUCount = len(coreUsage)
	}

	// Get CPU info
	info, err := cpu.Info()
	if err == nil && info != nil {
		stats.ModelName = info.Model
		if stats.CPUCount == 0 {
			stats.CPUCount = info.Threads
		}
	}

	// Get frequency (use first core's frequency as representative)
	freqs, err := cpu.Frequency()
	if err == nil && len(freqs) > 0 {
		stats.Frequency = freqs[0]
	}

	return stats, nil
}

// ReadMemoryStats reads memory statistics from the platform and converts to monitor types.
func (pa *PlatformAdapter) ReadMemoryStats() (MemoryStats, error) {
	mem := pa.plat.Memory()
	if mem == nil {
		return MemoryStats{}, nil
	}

	stats := MemoryStats{}

	// Get memory stats
	memStats, err := mem.Stats()
	if err == nil && memStats != nil {
		stats.Total = memStats.Total
		stats.Used = memStats.Used
		stats.Free = memStats.Free
		stats.Available = memStats.Available
		stats.Buffers = memStats.Buffers
		stats.Cached = memStats.Cached
		stats.UsagePercent = memStats.UsedPercent
	}

	// Get swap stats
	swapStats, err := mem.SwapStats()
	if err == nil && swapStats != nil {
		stats.SwapTotal = swapStats.Total
		stats.SwapUsed = swapStats.Used
		stats.SwapFree = swapStats.Free
		stats.SwapPercent = swapStats.UsedPercent
	}

	return stats, nil
}

// ReadNetworkStats reads network statistics from the platform and converts to monitor types.
func (pa *PlatformAdapter) ReadNetworkStats() (NetworkStats, error) {
	net := pa.plat.Network()
	if net == nil {
		return NetworkStats{Interfaces: make(map[string]InterfaceStats)}, nil
	}

	stats := NetworkStats{
		Interfaces: make(map[string]InterfaceStats),
	}

	// Get all interface stats
	allStats, err := net.AllStats()
	if err == nil && allStats != nil {
		for name, netStats := range allStats {
			ifStats := InterfaceStats{
				Name:      name,
				RxBytes:   netStats.BytesRecv,
				TxBytes:   netStats.BytesSent,
				RxPackets: netStats.PacketsRecv,
				TxPackets: netStats.PacketsSent,
				RxErrors:  netStats.ErrorsIn,
				TxErrors:  netStats.ErrorsOut,
				RxDropped: netStats.DropIn,
				TxDropped: netStats.DropOut,
			}
			stats.Interfaces[name] = ifStats
			stats.TotalRxBytes += netStats.BytesRecv
			stats.TotalTxBytes += netStats.BytesSent
		}
	}

	return stats, nil
}

// ReadFilesystemStats reads filesystem statistics from the platform and converts to monitor types.
func (pa *PlatformAdapter) ReadFilesystemStats() (FilesystemStats, error) {
	fs := pa.plat.Filesystem()
	if fs == nil {
		return FilesystemStats{Mounts: make(map[string]MountStats)}, nil
	}

	stats := FilesystemStats{
		Mounts: make(map[string]MountStats),
	}

	// Get mount info
	mounts, err := fs.Mounts()
	if err != nil {
		return stats, nil
	}

	for _, mount := range mounts {
		fsStats, err := fs.Stats(mount.MountPoint)
		if err != nil {
			continue
		}

		mountStats := MountStats{
			MountPoint:   mount.MountPoint,
			Device:       mount.Device,
			FSType:       mount.FSType,
			Total:        fsStats.Total,
			Used:         fsStats.Used,
			Free:         fsStats.Free,
			Available:    fsStats.Free, // Platform uses Free, monitor has Available
			UsagePercent: fsStats.UsedPercent,
			InodesTotal:  fsStats.InodesTotal,
			InodesUsed:   fsStats.InodesUsed,
			InodesFree:   fsStats.InodesFree,
		}

		// Calculate inode percentage
		if mountStats.InodesTotal > 0 {
			mountStats.InodesPercent = float64(mountStats.InodesUsed) / float64(mountStats.InodesTotal) * 100.0
		}

		stats.Mounts[mount.MountPoint] = mountStats
	}

	return stats, nil
}

// ReadBatteryStats reads battery statistics from the platform and converts to monitor types.
func (pa *PlatformAdapter) ReadBatteryStats() (BatteryStats, error) {
	bat := pa.plat.Battery()
	if bat == nil {
		return BatteryStats{
			Batteries:  make(map[string]BatteryInfo),
			ACAdapters: make(map[string]ACAdapterInfo),
		}, nil
	}

	stats := BatteryStats{
		Batteries:  make(map[string]BatteryInfo),
		ACAdapters: make(map[string]ACAdapterInfo),
	}

	count := bat.Count()
	for i := 0; i < count; i++ {
		batStats, err := bat.Stats(i)
		if err != nil {
			continue
		}

		name := "BAT" + string(rune('0'+i))
		info := BatteryInfo{
			Name:             name,
			Present:          true,
			Capacity:         int(batStats.Percent),
			TimeToEmpty:      batStats.TimeRemaining.Seconds(),
			EnergyNow:        batStats.Current,
			EnergyFull:       batStats.FullCapacity,
			EnergyFullDesign: batStats.FullCapacity,
			VoltageNow:       uint64(batStats.Voltage * 1000000), // Convert V to µV
		}

		if batStats.Charging {
			info.Status = "Charging"
			stats.IsCharging = true
		} else {
			info.Status = "Discharging"
			stats.IsDischarging = true
		}

		stats.Batteries[name] = info
		stats.TotalCapacity += batStats.Percent
		stats.TotalEnergyNow += batStats.Current
		stats.TotalEnergyFull += batStats.FullCapacity
	}

	if count > 0 {
		stats.TotalCapacity /= float64(count)
	}

	return stats, nil
}

// ReadSensorStats reads sensor statistics from the platform and converts to hwmon types.
func (pa *PlatformAdapter) ReadSensorStats() (HwmonStats, error) {
	sensors := pa.plat.Sensors()
	if sensors == nil {
		return HwmonStats{
			Devices:     make(map[string]HwmonDevice),
			TempSensors: []TempSensor{},
		}, nil
	}

	stats := HwmonStats{
		Devices:     make(map[string]HwmonDevice),
		TempSensors: []TempSensor{},
	}

	// Get temperature sensors
	temps, err := sensors.Temperatures()
	if err == nil {
		for _, reading := range temps {
			sensor := TempSensor{
				Label:        reading.Label,
				DeviceName:   reading.Name,
				InputCelsius: reading.Value,
				CritCelsius:  reading.Critical,
				MaxCelsius:   reading.Critical,
				// Convert to millidegrees for raw values
				Input: int64(reading.Value * 1000),
				Crit:  int64(reading.Critical * 1000),
				Max:   int64(reading.Critical * 1000),
			}
			stats.TempSensors = append(stats.TempSensors, sensor)
		}
	}

	return stats, nil
}

// PlatformName returns the name of the underlying platform.
func (pa *PlatformAdapter) PlatformName() string {
	return pa.plat.Name()
}
