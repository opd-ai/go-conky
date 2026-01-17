package monitor

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// SystemMonitor provides centralized system monitoring capabilities.
// It periodically updates system statistics from /proc filesystem.
type SystemMonitor struct {
	data              *SystemData
	interval          time.Duration
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
	ctx               context.Context
	cancel            context.CancelFunc
	wg                sync.WaitGroup
	mu                sync.RWMutex
	running           bool
}

// NewSystemMonitor creates a new SystemMonitor with the specified update interval.
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
		ctx:               ctx,
		cancel:            cancel,
	}
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
func (sm *SystemMonitor) Update() error {
	var errs []error

	// Update CPU stats
	cpuStats, err := sm.cpuReader.ReadStats()
	if err != nil {
		errs = append(errs, fmt.Errorf("cpu: %w", err))
	} else {
		sm.data.setCPU(cpuStats)
	}

	// Update memory stats
	memStats, err := sm.memReader.ReadStats()
	if err != nil {
		errs = append(errs, fmt.Errorf("memory: %w", err))
	} else {
		sm.data.setMemory(memStats)
	}

	// Update uptime stats
	uptimeStats, err := sm.uptimeReader.ReadStats()
	if err != nil {
		errs = append(errs, fmt.Errorf("uptime: %w", err))
	} else {
		sm.data.setUptime(uptimeStats)
	}

	// Update network stats
	networkStats, err := sm.networkReader.ReadStats()
	if err != nil {
		errs = append(errs, fmt.Errorf("network: %w", err))
	} else {
		// Augment network stats with address information
		sm.augmentNetworkStats(&networkStats)
		sm.data.setNetwork(networkStats)
	}

	// Update filesystem stats
	filesystemStats, err := sm.filesystemReader.ReadStats()
	if err != nil {
		errs = append(errs, fmt.Errorf("filesystem: %w", err))
	} else {
		sm.data.setFilesystem(filesystemStats)
	}

	// Update disk I/O stats
	diskIOStats, err := sm.diskIOReader.ReadStats()
	if err != nil {
		errs = append(errs, fmt.Errorf("diskio: %w", err))
	} else {
		sm.data.setDiskIO(diskIOStats)
	}

	// Update hardware monitoring stats
	hwmonStats, err := sm.hwmonReader.ReadStats()
	if err != nil {
		errs = append(errs, fmt.Errorf("hwmon: %w", err))
	} else {
		sm.data.setHwmon(hwmonStats)
	}

	// Update process stats
	processStats, err := sm.processReader.ReadStats()
	if err != nil {
		errs = append(errs, fmt.Errorf("process: %w", err))
	} else {
		sm.data.setProcess(processStats)
	}

	// Update battery stats
	batteryStats, err := sm.batteryReader.ReadStats()
	if err != nil {
		errs = append(errs, fmt.Errorf("battery: %w", err))
	} else {
		sm.data.setBattery(batteryStats)
	}

	// Update audio stats
	audioStats, err := sm.audioReader.ReadStats()
	if err != nil {
		errs = append(errs, fmt.Errorf("audio: %w", err))
	} else {
		sm.data.setAudio(audioStats)
	}

	// Update system info (includes load averages which change frequently)
	sysInfoStats, err := sm.sysInfoReader.ReadSystemInfo()
	if err != nil {
		errs = append(errs, fmt.Errorf("sysinfo: %w", err))
	} else {
		sm.data.setSysInfo(sysInfoStats)
	}

	if len(errs) > 0 {
		errMsgs := make([]string, len(errs))
		for i, e := range errs {
			errMsgs[i] = e.Error()
		}
		return fmt.Errorf("update errors: %s", strings.Join(errMsgs, "; "))
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
