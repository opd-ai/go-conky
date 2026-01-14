// Package platform provides cross-platform system monitoring abstractions for go-conky.
//
// The platform package defines interfaces and types for collecting system metrics
// across different operating systems. It provides a unified API for monitoring CPU,
// memory, network, filesystem, battery, and hardware sensors, with platform-specific
// implementations for Linux, Windows, macOS, and Android.
//
// # Architecture
//
// The package uses a provider pattern where each platform implements the Platform
// interface and provides concrete implementations of various provider interfaces
// (CPUProvider, MemoryProvider, etc.). This design allows for:
//
//   - Platform-agnostic monitoring code
//   - Easy testing with mock implementations
//   - Future support for remote monitoring via SSH
//   - Clean separation between interface and implementation
//
// # Usage
//
// Creating a platform for the current OS:
//
//	p, err := platform.NewPlatform()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer p.Close()
//
//	if err := p.Initialize(context.Background()); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Access system metrics
//	cpuUsage, _ := p.CPU().TotalUsage()
//	memStats, _ := p.Memory().Stats()
//	fmt.Printf("CPU: %.1f%%, Memory: %.1f%%\n", cpuUsage, memStats.UsedPercent)
//
// # Supported Platforms
//
// Currently supported:
//   - Linux (fully implemented via /proc filesystem)
//
// Planned for future implementation:
//   - Windows (WMI/PDH APIs)
//   - macOS (sysctl/IOKit)
//   - Android (Android APIs + /proc)
//   - Remote systems via SSH
//
// # Thread Safety
//
// All Platform and Provider implementations are safe for concurrent use from
// multiple goroutines unless otherwise documented.
package platform
