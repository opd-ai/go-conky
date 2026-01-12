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
	data             *SystemData
	interval         time.Duration
	cpuReader        *cpuReader
	memReader        *memoryReader
	uptimeReader     *uptimeReader
	networkReader    *networkReader
	filesystemReader *filesystemReader
	diskIOReader     *diskIOReader
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	mu               sync.RWMutex
	running          bool
}

// NewSystemMonitor creates a new SystemMonitor with the specified update interval.
func NewSystemMonitor(interval time.Duration) *SystemMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	return &SystemMonitor{
		data:             NewSystemData(),
		interval:         interval,
		cpuReader:        newCPUReader(),
		memReader:        newMemoryReader(),
		uptimeReader:     newUptimeReader(),
		networkReader:    newNetworkReader(),
		filesystemReader: newFilesystemReader(),
		diskIOReader:     newDiskIOReader(),
		ctx:              ctx,
		cancel:           cancel,
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

// IsRunning returns whether the monitor is currently running.
func (sm *SystemMonitor) IsRunning() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.running
}
