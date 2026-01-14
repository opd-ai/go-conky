// +build windows

package platform

import (
	"context"
	"sync"
)

// windowsPlatform implements Platform for Windows systems.
// It uses Windows API calls via syscalls for system metrics collection.
type windowsPlatform struct {
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	cpu        CPUProvider
	memory     MemoryProvider
	network    NetworkProvider
	filesystem FilesystemProvider
	battery    BatteryProvider
	sensors    SensorProvider
}

// NewWindowsPlatform creates a new Windows platform implementation.
func NewWindowsPlatform() Platform {
	return &windowsPlatform{}
}

func (p *windowsPlatform) Name() string {
	return "windows"
}

func (p *windowsPlatform) Initialize(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var cancel context.CancelFunc
	p.ctx, cancel = context.WithCancel(ctx)
	p.cancel = cancel

	// Initialize providers with real implementations
	p.cpu = newWindowsCPUProvider()
	p.memory = newWindowsMemoryProvider()
	p.network = newWindowsNetworkProvider()
	p.filesystem = newWindowsFilesystemProvider()
	p.battery = newWindowsBatteryProvider()
	p.sensors = newWindowsSensorProvider()

	return nil
}

func (p *windowsPlatform) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Close CPU provider to release PDH query resources
	if cpuProvider, ok := p.cpu.(*windowsCPUProvider); ok {
		cpuProvider.Close()
	}

	if p.cancel != nil {
		p.cancel()
	}
	return nil
}

func (p *windowsPlatform) CPU() CPUProvider {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.cpu
}

func (p *windowsPlatform) Memory() MemoryProvider {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.memory
}

func (p *windowsPlatform) Network() NetworkProvider {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.network
}

func (p *windowsPlatform) Filesystem() FilesystemProvider {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.filesystem
}

func (p *windowsPlatform) Battery() BatteryProvider {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.battery
}

func (p *windowsPlatform) Sensors() SensorProvider {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.sensors
}
