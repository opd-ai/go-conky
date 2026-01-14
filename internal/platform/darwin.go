//go:build darwin
// +build darwin

package platform

import (
	"context"
	"sync"
)

// darwinPlatform implements Platform for macOS/Darwin systems.
// It uses sysctl, mach APIs, and other macOS-specific system calls for metrics collection.
type darwinPlatform struct {
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

// NewDarwinPlatform creates a new macOS/Darwin platform implementation.
func NewDarwinPlatform() Platform {
	return &darwinPlatform{}
}

func (p *darwinPlatform) Name() string {
	return "darwin"
}

func (p *darwinPlatform) Initialize(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var cancel context.CancelFunc
	p.ctx, cancel = context.WithCancel(ctx)
	p.cancel = cancel

	// Initialize providers with real implementations
	p.cpu = newDarwinCPUProvider()
	p.memory = newDarwinMemoryProvider()
	p.network = newDarwinNetworkProvider()
	p.filesystem = newDarwinFilesystemProvider()
	p.battery = newDarwinBatteryProvider()
	p.sensors = newDarwinSensorProvider()

	return nil
}

func (p *darwinPlatform) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cancel != nil {
		p.cancel()
	}
	return nil
}

func (p *darwinPlatform) CPU() CPUProvider {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.cpu
}

func (p *darwinPlatform) Memory() MemoryProvider {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.memory
}

func (p *darwinPlatform) Network() NetworkProvider {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.network
}

func (p *darwinPlatform) Filesystem() FilesystemProvider {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.filesystem
}

func (p *darwinPlatform) Battery() BatteryProvider {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.battery
}

func (p *darwinPlatform) Sensors() SensorProvider {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.sensors
}
