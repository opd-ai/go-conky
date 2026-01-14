//go:build linux
// +build linux

package platform

import (
	"context"
	"sync"
)

// linuxPlatform implements Platform for Linux systems.
// This is a stub implementation that will be fully implemented in the next phase
// by refactoring existing monitor code.
type linuxPlatform struct {
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

// NewLinuxPlatform creates a new Linux platform implementation.
func NewLinuxPlatform() Platform {
	return &linuxPlatform{}
}

func (p *linuxPlatform) Name() string {
	return "linux"
}

func (p *linuxPlatform) Initialize(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var cancel context.CancelFunc
	p.ctx, cancel = context.WithCancel(ctx)
	p.cancel = cancel

	// Initialize providers with real implementations
	p.cpu = newLinuxCPUProvider()
	p.memory = newLinuxMemoryProvider()
	p.network = newLinuxNetworkProvider()
	p.filesystem = newLinuxFilesystemProvider()
	p.battery = newLinuxBatteryProvider()
	p.sensors = newLinuxSensorProvider()

	return nil
}

func (p *linuxPlatform) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cancel != nil {
		p.cancel()
	}
	return nil
}

func (p *linuxPlatform) CPU() CPUProvider {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.cpu
}

func (p *linuxPlatform) Memory() MemoryProvider {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.memory
}

func (p *linuxPlatform) Network() NetworkProvider {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.network
}

func (p *linuxPlatform) Filesystem() FilesystemProvider {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.filesystem
}

func (p *linuxPlatform) Battery() BatteryProvider {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.battery
}

func (p *linuxPlatform) Sensors() SensorProvider {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.sensors
}
