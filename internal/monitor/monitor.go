package monitor

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// SystemMonitor provides centralized system monitoring capabilities.
// It supports cross-platform monitoring via the platform abstraction layer,
// with fallback to Linux-specific /proc filesystem readers.
type SystemMonitor struct {
	data              *SystemData
	interval          time.Duration
	platformAdapter   *PlatformAdapter
	cpuReader         *cpuReader
	memReader         *memoryReader
	uptimeReader      *uptimeReader
	networkReader     *networkReader
	networkAddrReader *networkAddressReader
	wirelessReader    *wirelessReader
	filesystemReader  *filesystemReader
	diskIOReader      *diskIOReader
	hwmonReader       *hwmonReader
	processReader     *processReader
	batteryReader     *batteryReader
	audioReader       *audioReader
	sysInfoReader     *sysInfoReader
	tcpReader         *tcpReader
	gpuReader         *gpuReader
	mailReader        *mailReader
	weatherReader     *weatherReader
	mpdReader         *mpdReader
	ctx               context.Context
	cancel            context.CancelFunc
	wg                sync.WaitGroup
	mu                sync.RWMutex
	running           bool
}

// NewSystemMonitor creates a new SystemMonitor with the specified update interval.
// This uses Linux-specific readers for system monitoring.
func NewSystemMonitor(interval time.Duration) *SystemMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	return &SystemMonitor{
		data:              NewSystemData(),
		interval:          interval,
		cpuReader:         newCPUReader(),
		memReader:         newMemoryReader(),
		uptimeReader:      newUptimeReader(),
		networkReader:     newNetworkReader(),
		networkAddrReader: newNetworkAddressReader(),
		wirelessReader:    newWirelessReader(),
		filesystemReader:  newFilesystemReader(),
		diskIOReader:      newDiskIOReader(),
		hwmonReader:       newHwmonReader(),
		processReader:     newProcessReader(),
		batteryReader:     newBatteryReader(),
		audioReader:       newAudioReader(),
		sysInfoReader:     newSysInfoReader(),
		tcpReader:         newTCPReader(),
		gpuReader:         newGPUReader(),
		mailReader:        newMailReader(),
		weatherReader:     newWeatherReader(),
		mpdReader:         newMPDReader(),
		ctx:               ctx,
		cancel:            cancel,
	}
}

// NewSystemMonitorWithPlatform creates a new SystemMonitor that uses the platform
// abstraction layer for cross-platform system monitoring. The platform must be
// initialized before passing to this function.
//
// When a platform is provided, CPU, memory, network, filesystem, battery, and
// sensor readings will use the platform providers. Other readers (uptime, process,
// audio, GPU, mail, weather) will fall back to Linux-specific implementations.
//
// The platform parameter must implement the PlatformInterface defined in this package.
// The internal/platform.Platform type satisfies this interface.
func NewSystemMonitorWithPlatform(interval time.Duration, plat PlatformInterface) *SystemMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	sm := &SystemMonitor{
		data:              NewSystemData(),
		interval:          interval,
		platformAdapter:   NewPlatformAdapter(plat),
		uptimeReader:      newUptimeReader(),
		networkAddrReader: newNetworkAddressReader(),
		wirelessReader:    newWirelessReader(),
		processReader:     newProcessReader(),
		audioReader:       newAudioReader(),
		sysInfoReader:     newSysInfoReader(),
		tcpReader:         newTCPReader(),
		gpuReader:         newGPUReader(),
		mailReader:        newMailReader(),
		weatherReader:     newWeatherReader(),
		mpdReader:         newMPDReader(),
		ctx:               ctx,
		cancel:            cancel,
	}

	// Keep Linux fallback readers for cases where platform adapter fails or is nil
	sm.cpuReader = newCPUReader()
	sm.memReader = newMemoryReader()
	sm.networkReader = newNetworkReader()
	sm.filesystemReader = newFilesystemReader()
	sm.diskIOReader = newDiskIOReader()
	sm.hwmonReader = newHwmonReader()
	sm.batteryReader = newBatteryReader()

	return sm
}

// Start begins the monitoring loop in a background goroutine.
// It returns an error if the monitor is already running.
func (sm *SystemMonitor) Start() error {
	sm.mu.Lock()
	if sm.running {
		sm.mu.Unlock()
		return fmt.Errorf("monitor already running")
	}
	sm.running = true
	sm.mu.Unlock()

	// Perform initial update immediately
	if err := sm.Update(); err != nil {
		// Reset running state on failure
		sm.mu.Lock()
		sm.running = false
		sm.mu.Unlock()
		return fmt.Errorf("initial update failed: %w", err)
	}

	sm.wg.Add(1)
	go sm.monitorLoop()

	return nil
}

// Stop halts the monitoring loop and waits for it to complete.
func (sm *SystemMonitor) Stop() {
	sm.mu.Lock()
	if !sm.running {
		sm.mu.Unlock()
		return
	}
	sm.mu.Unlock()

	sm.cancel()
	sm.wg.Wait()

	sm.mu.Lock()
	sm.running = false
	sm.mu.Unlock()
}

// monitorLoop runs the periodic update cycle.
func (sm *SystemMonitor) monitorLoop() {
	defer sm.wg.Done()

	ticker := time.NewTicker(sm.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_ = sm.Update() // Ignore errors in background loop
		case <-sm.ctx.Done():
			return
		}
	}
}

// Update performs a single update of all system statistics.
// When a platform adapter is configured, it uses cross-platform providers.
// Falls back to Linux-specific readers when the platform adapter is nil or fails.
//
// Returns an *UpdateError if any component fails, preserving all individual errors.
// Use AsUpdateError() or errors.As() to inspect component-specific errors.
func (sm *SystemMonitor) Update() error {
	var errs []*ComponentError

	// Update CPU stats (prefer platform adapter)
	if sm.platformAdapter != nil {
		cpuStats, err := sm.platformAdapter.ReadCPUStats()
		if err != nil {
			errs = append(errs, NewComponentError(ErrorSourceCPU, true, err))
			// Fallback to Linux reader
			if sm.cpuReader != nil {
				if fallbackStats, fallbackErr := sm.cpuReader.ReadStats(); fallbackErr == nil {
					sm.data.setCPU(fallbackStats)
				}
			}
		} else {
			sm.data.setCPU(cpuStats)
		}
	} else if sm.cpuReader != nil {
		cpuStats, err := sm.cpuReader.ReadStats()
		if err != nil {
			errs = append(errs, NewComponentError(ErrorSourceCPU, false, err))
		} else {
			sm.data.setCPU(cpuStats)
		}
	}

	// Update memory stats (prefer platform adapter)
	if sm.platformAdapter != nil {
		memStats, err := sm.platformAdapter.ReadMemoryStats()
		if err != nil {
			errs = append(errs, NewComponentError(ErrorSourceMemory, true, err))
			// Fallback to Linux reader
			if sm.memReader != nil {
				if fallbackStats, fallbackErr := sm.memReader.ReadStats(); fallbackErr == nil {
					sm.data.setMemory(fallbackStats)
				}
			}
		} else {
			sm.data.setMemory(memStats)
		}
	} else if sm.memReader != nil {
		memStats, err := sm.memReader.ReadStats()
		if err != nil {
			errs = append(errs, NewComponentError(ErrorSourceMemory, false, err))
		} else {
			sm.data.setMemory(memStats)
		}
	}

	// Update uptime stats (Linux-specific only for now)
	if sm.uptimeReader != nil {
		uptimeStats, err := sm.uptimeReader.ReadStats()
		if err != nil {
			errs = append(errs, NewComponentError(ErrorSourceUptime, false, err))
		} else {
			sm.data.setUptime(uptimeStats)
		}
	}

	// Update network stats (prefer platform adapter)
	if sm.platformAdapter != nil {
		networkStats, err := sm.platformAdapter.ReadNetworkStats()
		if err != nil {
			errs = append(errs, NewComponentError(ErrorSourceNetwork, true, err))
			// Fallback to Linux reader
			if sm.networkReader != nil {
				if fallbackStats, fallbackErr := sm.networkReader.ReadStats(); fallbackErr == nil {
					sm.augmentNetworkStats(&fallbackStats)
					sm.data.setNetwork(fallbackStats)
				}
			}
		} else {
			// Augment with address information from Linux reader
			sm.augmentNetworkStats(&networkStats)
			sm.data.setNetwork(networkStats)
		}
	} else if sm.networkReader != nil {
		networkStats, err := sm.networkReader.ReadStats()
		if err != nil {
			errs = append(errs, NewComponentError(ErrorSourceNetwork, false, err))
		} else {
			sm.augmentNetworkStats(&networkStats)
			sm.data.setNetwork(networkStats)
		}
	}

	// Update filesystem stats (prefer platform adapter)
	if sm.platformAdapter != nil {
		filesystemStats, err := sm.platformAdapter.ReadFilesystemStats()
		if err != nil {
			errs = append(errs, NewComponentError(ErrorSourceFilesystem, true, err))
			// Fallback to Linux reader
			if sm.filesystemReader != nil {
				if fallbackStats, fallbackErr := sm.filesystemReader.ReadStats(); fallbackErr == nil {
					sm.data.setFilesystem(fallbackStats)
				}
			}
		} else {
			sm.data.setFilesystem(filesystemStats)
		}
	} else if sm.filesystemReader != nil {
		filesystemStats, err := sm.filesystemReader.ReadStats()
		if err != nil {
			errs = append(errs, NewComponentError(ErrorSourceFilesystem, false, err))
		} else {
			sm.data.setFilesystem(filesystemStats)
		}
	}

	// Update disk I/O stats (Linux-specific only for now)
	if sm.diskIOReader != nil {
		diskIOStats, err := sm.diskIOReader.ReadStats()
		if err != nil {
			errs = append(errs, NewComponentError(ErrorSourceDiskIO, false, err))
		} else {
			sm.data.setDiskIO(diskIOStats)
		}
	}

	// Update hardware monitoring stats (prefer platform adapter for sensors)
	if sm.platformAdapter != nil {
		hwmonStats, err := sm.platformAdapter.ReadSensorStats()
		if err != nil {
			errs = append(errs, NewComponentError(ErrorSourceHwmon, true, err))
			// Fallback to Linux reader
			if sm.hwmonReader != nil {
				if fallbackStats, fallbackErr := sm.hwmonReader.ReadStats(); fallbackErr == nil {
					sm.data.setHwmon(fallbackStats)
				}
			}
		} else {
			sm.data.setHwmon(hwmonStats)
		}
	} else if sm.hwmonReader != nil {
		hwmonStats, err := sm.hwmonReader.ReadStats()
		if err != nil {
			errs = append(errs, NewComponentError(ErrorSourceHwmon, false, err))
		} else {
			sm.data.setHwmon(hwmonStats)
		}
	}

	// Update process stats (Linux-specific only)
	if sm.processReader != nil {
		processStats, err := sm.processReader.ReadStats()
		if err != nil {
			errs = append(errs, NewComponentError(ErrorSourceProcess, false, err))
		} else {
			sm.data.setProcess(processStats)
		}
	}

	// Update battery stats (prefer platform adapter)
	if sm.platformAdapter != nil {
		batteryStats, err := sm.platformAdapter.ReadBatteryStats()
		if err != nil {
			errs = append(errs, NewComponentError(ErrorSourceBattery, true, err))
			// Fallback to Linux reader
			if sm.batteryReader != nil {
				if fallbackStats, fallbackErr := sm.batteryReader.ReadStats(); fallbackErr == nil {
					sm.data.setBattery(fallbackStats)
				}
			}
		} else {
			sm.data.setBattery(batteryStats)
		}
	} else if sm.batteryReader != nil {
		batteryStats, err := sm.batteryReader.ReadStats()
		if err != nil {
			errs = append(errs, NewComponentError(ErrorSourceBattery, false, err))
		} else {
			sm.data.setBattery(batteryStats)
		}
	}

	// Update audio stats (Linux-specific only)
	if sm.audioReader != nil {
		audioStats, err := sm.audioReader.ReadStats()
		if err != nil {
			errs = append(errs, NewComponentError(ErrorSourceAudio, false, err))
		} else {
			sm.data.setAudio(audioStats)
		}
	}

	// Update system info (Linux-specific only)
	if sm.sysInfoReader != nil {
		sysInfoStats, err := sm.sysInfoReader.ReadSystemInfo()
		if err != nil {
			errs = append(errs, NewComponentError(ErrorSourceSysInfo, false, err))
		} else {
			sm.data.setSysInfo(sysInfoStats)
		}
	}

	if len(errs) > 0 {
		return &UpdateError{Errors: errs}
	}
	return nil
}

// Data returns a snapshot of the current system data.
func (sm *SystemMonitor) Data() SystemData {
	sm.data.mu.RLock()
	defer sm.data.mu.RUnlock()
	return SystemData{
		CPU:        sm.data.CPU,
		Memory:     sm.data.Memory,
		Uptime:     sm.data.Uptime,
		Network:    sm.data.copyNetwork(),
		Filesystem: sm.data.copyFilesystem(),
		DiskIO:     sm.data.copyDiskIO(),
		Hwmon:      sm.data.copyHwmon(),
		Process:    sm.data.copyProcess(),
		Battery:    sm.data.copyBattery(),
		Audio:      sm.data.copyAudio(),
		SysInfo:    sm.data.copySysInfo(),
	}
}

// CPU returns the current CPU statistics.
func (sm *SystemMonitor) CPU() CPUStats {
	return sm.data.GetCPU()
}

// Memory returns the current memory statistics.
func (sm *SystemMonitor) Memory() MemoryStats {
	return sm.data.GetMemory()
}

// Uptime returns the current uptime statistics.
func (sm *SystemMonitor) Uptime() UptimeStats {
	return sm.data.GetUptime()
}

// Network returns the current network statistics.
func (sm *SystemMonitor) Network() NetworkStats {
	return sm.data.GetNetwork()
}

// Filesystem returns the current filesystem statistics.
func (sm *SystemMonitor) Filesystem() FilesystemStats {
	return sm.data.GetFilesystem()
}

// DiskIO returns the current disk I/O statistics.
func (sm *SystemMonitor) DiskIO() DiskIOStats {
	return sm.data.GetDiskIO()
}

// Hwmon returns the current hardware monitoring statistics.
func (sm *SystemMonitor) Hwmon() HwmonStats {
	return sm.data.GetHwmon()
}

// Process returns the current process statistics.
func (sm *SystemMonitor) Process() ProcessStats {
	return sm.data.GetProcess()
}

// Battery returns the current battery statistics.
func (sm *SystemMonitor) Battery() BatteryStats {
	return sm.data.GetBattery()
}

// Audio returns the current audio statistics.
func (sm *SystemMonitor) Audio() AudioStats {
	return sm.data.GetAudio()
}

// SysInfo returns the current system information.
func (sm *SystemMonitor) SysInfo() SystemInfo {
	return sm.data.GetSysInfo()
}

// TCP returns the current TCP connection statistics.
func (sm *SystemMonitor) TCP() TCPStats {
	stats, _ := sm.tcpReader.ReadStats()
	return stats
}

// TCPCountInRange counts TCP connections in the given port range.
func (sm *SystemMonitor) TCPCountInRange(minPort, maxPort int) int {
	return sm.tcpReader.CountInRange(minPort, maxPort)
}

// TCPConnectionByIndex returns a specific connection in the port range.
func (sm *SystemMonitor) TCPConnectionByIndex(minPort, maxPort, index int) *TCPConnection {
	return sm.tcpReader.GetConnectionByIndex(minPort, maxPort, index)
}

// GPU returns the current NVIDIA GPU statistics.
func (sm *SystemMonitor) GPU() GPUStats {
	stats, _ := sm.gpuReader.ReadStats()
	return stats
}

// Mail returns the current mail statistics.
func (sm *SystemMonitor) Mail() MailStats {
	stats, _ := sm.mailReader.ReadStats()
	return stats
}

// AddMailAccount adds a mail account for monitoring.
func (sm *SystemMonitor) AddMailAccount(config MailConfig) error {
	return sm.mailReader.AddAccount(config)
}

// RemoveMailAccount removes a mail account from monitoring.
func (sm *SystemMonitor) RemoveMailAccount(name string) {
	sm.mailReader.RemoveAccount(name)
}

// MailUnseenCount returns the unseen message count for an account.
func (sm *SystemMonitor) MailUnseenCount(name string) int {
	return sm.mailReader.GetUnseenCount(name)
}

// MailTotalCount returns the total message count for an account.
func (sm *SystemMonitor) MailTotalCount(name string) int {
	return sm.mailReader.GetTotalCount(name)
}

// MailTotalUnseen returns the sum of unseen messages across all accounts.
func (sm *SystemMonitor) MailTotalUnseen() int {
	return sm.mailReader.GetTotalUnseen()
}

// MailTotalMessages returns the sum of all messages across all accounts.
func (sm *SystemMonitor) MailTotalMessages() int {
	return sm.mailReader.GetTotalMessages()
}

// Weather returns weather data for the given station ID (ICAO code).
func (sm *SystemMonitor) Weather(stationID string) WeatherStats {
	stats, _ := sm.weatherReader.ReadWeather(stationID)
	return stats
}

// MPD returns the current MPD playback status.
func (sm *SystemMonitor) MPD() MPDStats {
	stats, _ := sm.mpdReader.ReadStats()
	return stats
}

// SetMPDHost sets the MPD server host.
func (sm *SystemMonitor) SetMPDHost(host string) {
	sm.mpdReader.SetHost(host)
}

// SetMPDPort sets the MPD server port.
func (sm *SystemMonitor) SetMPDPort(port int) {
	sm.mpdReader.SetPort(port)
}

// SetMPDPassword sets the MPD server password.
func (sm *SystemMonitor) SetMPDPassword(password string) {
	sm.mpdReader.SetPassword(password)
}

// augmentNetworkStats adds IP address, gateway, nameserver, and wireless information to network stats.
func (sm *SystemMonitor) augmentNetworkStats(stats *NetworkStats) {
	// Read interface addresses
	ifAddrs, err := sm.networkAddrReader.ReadInterfaceAddresses()
	if err == nil {
		for name, addrs := range ifAddrs {
			if ifStats, ok := stats.Interfaces[name]; ok {
				ifStats.IPv4Addrs = addrs.IPv4
				ifStats.IPv6Addrs = addrs.IPv6
				stats.Interfaces[name] = ifStats
			}
		}
	}

	// Read default gateway
	gateway, gwIface, err := sm.networkAddrReader.ReadDefaultGateway()
	if err == nil {
		stats.GatewayIP = gateway
		stats.GatewayInterface = gwIface
	}

	// Read nameservers
	nameservers, err := sm.networkAddrReader.ReadNameservers()
	if err == nil {
		stats.Nameservers = nameservers
	}

	// Read wireless info
	wirelessStats, err := sm.wirelessReader.ReadWirelessStats()
	if err == nil {
		for name, wireless := range wirelessStats {
			if ifStats, ok := stats.Interfaces[name]; ok {
				w := wireless // Create a copy to take pointer
				ifStats.Wireless = &w
				stats.Interfaces[name] = ifStats
			}
		}
	}
}

// IsRunning returns whether the monitor is currently running.
func (sm *SystemMonitor) IsRunning() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.running
}

// UsesPlatform returns true if the monitor is using the platform abstraction layer.
func (sm *SystemMonitor) UsesPlatform() bool {
	return sm.platformAdapter != nil
}

// Platform returns the underlying platform if one is configured.
// Returns nil if the monitor is using Linux-specific readers.
func (sm *SystemMonitor) Platform() *PlatformAdapter {
	return sm.platformAdapter
}
