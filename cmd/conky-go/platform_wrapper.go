// Package main provides platform wrapper types that bridge the internal/platform
// package with the internal/monitor package, enabling cross-platform monitoring.
package main

import (
	"context"
	"time"

	"github.com/opd-ai/go-conky/internal/monitor"
	"github.com/opd-ai/go-conky/internal/platform"
)

// PlatformWrapper adapts a platform.Platform to monitor.PlatformInterface.
// This bridges the gap between the platform abstraction layer and the monitor
// package, enabling cross-platform system monitoring.
type PlatformWrapper struct {
	plat platform.Platform
}

// WrapPlatform creates a new PlatformWrapper for the given platform.
// Returns nil if plat is nil.
func WrapPlatform(plat platform.Platform) *PlatformWrapper {
	if plat == nil {
		return nil
	}
	return &PlatformWrapper{plat: plat}
}

// Name returns the platform identifier.
func (pw *PlatformWrapper) Name() string {
	return pw.plat.Name()
}

// CPU returns the CPU metrics provider wrapped as a monitor interface.
func (pw *PlatformWrapper) CPU() monitor.CPUProviderInterface {
	cpu := pw.plat.CPU()
	if cpu == nil {
		return nil
	}
	return &cpuProviderWrapper{cpu: cpu}
}

// Memory returns the memory metrics provider wrapped as a monitor interface.
func (pw *PlatformWrapper) Memory() monitor.MemoryProviderInterface {
	mem := pw.plat.Memory()
	if mem == nil {
		return nil
	}
	return &memoryProviderWrapper{mem: mem}
}

// Network returns the network metrics provider wrapped as a monitor interface.
func (pw *PlatformWrapper) Network() monitor.NetworkProviderInterface {
	net := pw.plat.Network()
	if net == nil {
		return nil
	}
	return &networkProviderWrapper{net: net}
}

// Filesystem returns the filesystem metrics provider wrapped as a monitor interface.
func (pw *PlatformWrapper) Filesystem() monitor.FilesystemProviderInterface {
	fs := pw.plat.Filesystem()
	if fs == nil {
		return nil
	}
	return &filesystemProviderWrapper{fs: fs}
}

// Battery returns the battery metrics provider wrapped as a monitor interface.
func (pw *PlatformWrapper) Battery() monitor.BatteryProviderInterface {
	bat := pw.plat.Battery()
	if bat == nil {
		return nil
	}
	return &batteryProviderWrapper{bat: bat}
}

// Sensors returns the sensors provider wrapped as a monitor interface.
func (pw *PlatformWrapper) Sensors() monitor.SensorProviderInterface {
	sensors := pw.plat.Sensors()
	if sensors == nil {
		return nil
	}
	return &sensorProviderWrapper{sensors: sensors}
}

// Platform returns the underlying platform.Platform.
func (pw *PlatformWrapper) Platform() platform.Platform {
	return pw.plat
}

// Close closes the underlying platform.
func (pw *PlatformWrapper) Close() error {
	return pw.plat.Close()
}

// cpuProviderWrapper adapts platform.CPUProvider to monitor.CPUProviderInterface.
type cpuProviderWrapper struct {
	cpu platform.CPUProvider
}

func (w *cpuProviderWrapper) Usage() ([]float64, error) {
	return w.cpu.Usage()
}

func (w *cpuProviderWrapper) TotalUsage() (float64, error) {
	return w.cpu.TotalUsage()
}

func (w *cpuProviderWrapper) Frequency() ([]float64, error) {
	return w.cpu.Frequency()
}

func (w *cpuProviderWrapper) Info() (*monitor.PlatformCPUInfo, error) {
	info, err := w.cpu.Info()
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, nil
	}
	return &monitor.PlatformCPUInfo{
		Model:     info.Model,
		Vendor:    info.Vendor,
		Cores:     info.Cores,
		Threads:   info.Threads,
		CacheSize: info.CacheSize,
	}, nil
}

func (w *cpuProviderWrapper) LoadAverage() (float64, float64, float64, error) {
	return w.cpu.LoadAverage()
}

// memoryProviderWrapper adapts platform.MemoryProvider to monitor.MemoryProviderInterface.
type memoryProviderWrapper struct {
	mem platform.MemoryProvider
}

func (w *memoryProviderWrapper) Stats() (*monitor.PlatformMemoryStats, error) {
	stats, err := w.mem.Stats()
	if err != nil {
		return nil, err
	}
	if stats == nil {
		return nil, nil
	}
	return &monitor.PlatformMemoryStats{
		Total:       stats.Total,
		Used:        stats.Used,
		Free:        stats.Free,
		Available:   stats.Available,
		Cached:      stats.Cached,
		Buffers:     stats.Buffers,
		UsedPercent: stats.UsedPercent,
	}, nil
}

func (w *memoryProviderWrapper) SwapStats() (*monitor.PlatformSwapStats, error) {
	stats, err := w.mem.SwapStats()
	if err != nil {
		return nil, err
	}
	if stats == nil {
		return nil, nil
	}
	return &monitor.PlatformSwapStats{
		Total:       stats.Total,
		Used:        stats.Used,
		Free:        stats.Free,
		UsedPercent: stats.UsedPercent,
	}, nil
}

// networkProviderWrapper adapts platform.NetworkProvider to monitor.NetworkProviderInterface.
type networkProviderWrapper struct {
	net platform.NetworkProvider
}

func (w *networkProviderWrapper) Interfaces() ([]string, error) {
	return w.net.Interfaces()
}

func (w *networkProviderWrapper) Stats(interfaceName string) (*monitor.PlatformNetworkStats, error) {
	stats, err := w.net.Stats(interfaceName)
	if err != nil {
		return nil, err
	}
	if stats == nil {
		return nil, nil
	}
	return &monitor.PlatformNetworkStats{
		BytesRecv:   stats.BytesRecv,
		BytesSent:   stats.BytesSent,
		PacketsRecv: stats.PacketsRecv,
		PacketsSent: stats.PacketsSent,
		ErrorsIn:    stats.ErrorsIn,
		ErrorsOut:   stats.ErrorsOut,
		DropIn:      stats.DropIn,
		DropOut:     stats.DropOut,
	}, nil
}

func (w *networkProviderWrapper) AllStats() (map[string]*monitor.PlatformNetworkStats, error) {
	allStats, err := w.net.AllStats()
	if err != nil {
		return nil, err
	}
	if allStats == nil {
		return nil, nil
	}
	result := make(map[string]*monitor.PlatformNetworkStats, len(allStats))
	for name, stats := range allStats {
		result[name] = &monitor.PlatformNetworkStats{
			BytesRecv:   stats.BytesRecv,
			BytesSent:   stats.BytesSent,
			PacketsRecv: stats.PacketsRecv,
			PacketsSent: stats.PacketsSent,
			ErrorsIn:    stats.ErrorsIn,
			ErrorsOut:   stats.ErrorsOut,
			DropIn:      stats.DropIn,
			DropOut:     stats.DropOut,
		}
	}
	return result, nil
}

// filesystemProviderWrapper adapts platform.FilesystemProvider to monitor.FilesystemProviderInterface.
type filesystemProviderWrapper struct {
	fs platform.FilesystemProvider
}

func (w *filesystemProviderWrapper) Mounts() ([]monitor.PlatformMountInfo, error) {
	mounts, err := w.fs.Mounts()
	if err != nil {
		return nil, err
	}
	result := make([]monitor.PlatformMountInfo, len(mounts))
	for i, m := range mounts {
		result[i] = monitor.PlatformMountInfo{
			Device:     m.Device,
			MountPoint: m.MountPoint,
			FSType:     m.FSType,
			Options:    m.Options,
		}
	}
	return result, nil
}

func (w *filesystemProviderWrapper) Stats(mountPoint string) (*monitor.PlatformFilesystemStats, error) {
	stats, err := w.fs.Stats(mountPoint)
	if err != nil {
		return nil, err
	}
	if stats == nil {
		return nil, nil
	}
	return &monitor.PlatformFilesystemStats{
		Total:       stats.Total,
		Used:        stats.Used,
		Free:        stats.Free,
		UsedPercent: stats.UsedPercent,
		InodesTotal: stats.InodesTotal,
		InodesUsed:  stats.InodesUsed,
		InodesFree:  stats.InodesFree,
	}, nil
}

func (w *filesystemProviderWrapper) DiskIO(device string) (*monitor.PlatformDiskIOStats, error) {
	stats, err := w.fs.DiskIO(device)
	if err != nil {
		return nil, err
	}
	if stats == nil {
		return nil, nil
	}
	return &monitor.PlatformDiskIOStats{
		ReadBytes:  stats.ReadBytes,
		WriteBytes: stats.WriteBytes,
		ReadCount:  stats.ReadCount,
		WriteCount: stats.WriteCount,
		ReadTime:   stats.ReadTime,
		WriteTime:  stats.WriteTime,
	}, nil
}

// batteryProviderWrapper adapts platform.BatteryProvider to monitor.BatteryProviderInterface.
type batteryProviderWrapper struct {
	bat platform.BatteryProvider
}

func (w *batteryProviderWrapper) Count() int {
	return w.bat.Count()
}

func (w *batteryProviderWrapper) Stats(index int) (*monitor.PlatformBatteryStats, error) {
	stats, err := w.bat.Stats(index)
	if err != nil {
		return nil, err
	}
	if stats == nil {
		return nil, nil
	}
	return &monitor.PlatformBatteryStats{
		Percent:       stats.Percent,
		TimeRemaining: stats.TimeRemaining,
		Charging:      stats.Charging,
		FullCapacity:  stats.FullCapacity,
		Current:       stats.Current,
		Voltage:       stats.Voltage,
	}, nil
}

// sensorProviderWrapper adapts platform.SensorProvider to monitor.SensorProviderInterface.
type sensorProviderWrapper struct {
	sensors platform.SensorProvider
}

func (w *sensorProviderWrapper) Temperatures() ([]monitor.PlatformSensorReading, error) {
	readings, err := w.sensors.Temperatures()
	if err != nil {
		return nil, err
	}
	result := make([]monitor.PlatformSensorReading, len(readings))
	for i, r := range readings {
		result[i] = monitor.PlatformSensorReading{
			Name:     r.Name,
			Label:    r.Label,
			Value:    r.Value,
			Unit:     r.Unit,
			Critical: r.Critical,
		}
	}
	return result, nil
}

func (w *sensorProviderWrapper) Fans() ([]monitor.PlatformSensorReading, error) {
	readings, err := w.sensors.Fans()
	if err != nil {
		return nil, err
	}
	result := make([]monitor.PlatformSensorReading, len(readings))
	for i, r := range readings {
		result[i] = monitor.PlatformSensorReading{
			Name:     r.Name,
			Label:    r.Label,
			Value:    r.Value,
			Unit:     r.Unit,
			Critical: r.Critical,
		}
	}
	return result, nil
}

// initializePlatform creates and initializes the local platform for the current OS.
// Returns nil if platform initialization fails (caller should fall back to Linux readers).
func initializePlatform(ctx context.Context) *PlatformWrapper {
	plat, err := platform.NewPlatform()
	if err != nil {
		return nil
	}

	// Initialize with a timeout
	initCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := plat.Initialize(initCtx); err != nil {
		// Close the platform on initialization failure
		_ = plat.Close()
		return nil
	}

	return WrapPlatform(plat)
}
