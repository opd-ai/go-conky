// Package platform provides cross-platform system monitoring abstractions.
// It defines interfaces for OS-specific implementations and provides a factory
// for creating the appropriate platform implementation based on the runtime OS.
package platform

import (
	"context"
	"time"
)

// Platform defines the interface for OS-specific system monitoring.
// Each supported operating system implements this interface to provide
// unified access to system metrics.
type Platform interface {
	// Name returns the platform identifier (e.g., "linux", "windows", "darwin", "android").
	Name() string

	// Initialize prepares the platform for data collection.
	// Returns an error if the platform cannot be initialized.
	Initialize(ctx context.Context) error

	// Close releases any platform-specific resources.
	Close() error

	// CPU returns the CPU metrics provider for this platform.
	CPU() CPUProvider

	// Memory returns the memory metrics provider for this platform.
	Memory() MemoryProvider

	// Network returns the network metrics provider for this platform.
	Network() NetworkProvider

	// Filesystem returns the filesystem metrics provider for this platform.
	Filesystem() FilesystemProvider

	// Battery returns the battery metrics provider for this platform.
	// Returns nil if battery monitoring is not supported.
	Battery() BatteryProvider

	// Sensors returns the hardware sensors provider for this platform.
	// Returns nil if sensor monitoring is not supported.
	Sensors() SensorProvider
}

// CPUProvider defines the interface for CPU metrics collection.
type CPUProvider interface {
	// Usage returns CPU usage percentages for all cores.
	Usage() ([]float64, error)

	// TotalUsage returns the aggregate CPU usage percentage.
	TotalUsage() (float64, error)

	// Frequency returns CPU frequencies in MHz for all cores.
	Frequency() ([]float64, error)

	// Info returns static CPU information (model, cores, etc.).
	Info() (*CPUInfo, error)

	// LoadAverage returns 1, 5, and 15 minute load averages.
	// Returns an error on platforms that don't support load average (Windows).
	LoadAverage() (float64, float64, float64, error)
}

// MemoryProvider defines the interface for memory metrics collection.
type MemoryProvider interface {
	// Stats returns current memory statistics.
	Stats() (*MemoryStats, error)

	// SwapStats returns swap/page file statistics.
	SwapStats() (*SwapStats, error)
}

// NetworkProvider defines the interface for network metrics collection.
type NetworkProvider interface {
	// Interfaces returns a list of network interface names.
	Interfaces() ([]string, error)

	// Stats returns network statistics for a specific interface.
	Stats(interfaceName string) (*NetworkStats, error)

	// AllStats returns network statistics for all interfaces.
	AllStats() (map[string]*NetworkStats, error)
}

// FilesystemProvider defines the interface for filesystem metrics collection.
type FilesystemProvider interface {
	// Mounts returns a list of mounted filesystems.
	Mounts() ([]MountInfo, error)

	// Stats returns filesystem statistics for a specific mount point.
	Stats(mountPoint string) (*FilesystemStats, error)

	// DiskIO returns disk I/O statistics for a specific device.
	DiskIO(device string) (*DiskIOStats, error)
}

// BatteryProvider defines the interface for battery metrics collection.
type BatteryProvider interface {
	// Count returns the number of batteries in the system.
	Count() int

	// Stats returns battery statistics for a specific battery index.
	Stats(index int) (*BatteryStats, error)
}

// SensorProvider defines the interface for hardware sensor metrics collection.
type SensorProvider interface {
	// Temperatures returns all temperature sensor readings.
	Temperatures() ([]SensorReading, error)

	// Fans returns all fan speed sensor readings.
	Fans() ([]SensorReading, error)
}

// CPUInfo contains static CPU information.
type CPUInfo struct {
	Model     string
	Vendor    string
	Cores     int
	Threads   int
	CacheSize int64 // in bytes
}

// MemoryStats contains memory usage statistics.
type MemoryStats struct {
	Total       uint64
	Used        uint64
	Free        uint64
	Available   uint64
	Cached      uint64
	Buffers     uint64
	UsedPercent float64
}

// SwapStats contains swap/page file statistics.
type SwapStats struct {
	Total       uint64
	Used        uint64
	Free        uint64
	UsedPercent float64
}

// NetworkStats contains network interface statistics.
type NetworkStats struct {
	BytesRecv   uint64
	BytesSent   uint64
	PacketsRecv uint64
	PacketsSent uint64
	ErrorsIn    uint64
	ErrorsOut   uint64
	DropIn      uint64
	DropOut     uint64
}

// MountInfo contains filesystem mount information.
type MountInfo struct {
	Device     string
	MountPoint string
	FSType     string
	Options    []string
}

// FilesystemStats contains filesystem usage statistics.
type FilesystemStats struct {
	Total       uint64
	Used        uint64
	Free        uint64
	UsedPercent float64
	InodesTotal uint64
	InodesUsed  uint64
	InodesFree  uint64
}

// DiskIOStats contains disk I/O statistics.
type DiskIOStats struct {
	ReadBytes  uint64
	WriteBytes uint64
	ReadCount  uint64
	WriteCount uint64
	ReadTime   time.Duration
	WriteTime  time.Duration
}

// BatteryStats contains battery status information.
type BatteryStats struct {
	Percent       float64
	TimeRemaining time.Duration
	Charging      bool
	FullCapacity  uint64
	Current       uint64
	Voltage       float64
}

// SensorReading contains a sensor reading with metadata.
type SensorReading struct {
	Name     string
	Label    string
	Value    float64
	Unit     string
	Critical float64 // threshold value (0 if not available)
}
