//go:build android
// +build android

package platform

import (
	"context"
	"sync"
)

// androidPlatform implements Platform for Android systems.
// Android is based on Linux kernel and shares the /proc filesystem interface
// for CPU, memory, and network monitoring. Battery and sensor access uses
// sysfs paths similar to Linux but with Android-specific considerations.
type androidPlatform struct {
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

// NewAndroidPlatform creates a new Android platform implementation.
func NewAndroidPlatform() Platform {
	return &androidPlatform{}
}

func (p *androidPlatform) Name() string {
	return "android"
}

func (p *androidPlatform) Initialize(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var cancel context.CancelFunc
	p.ctx, cancel = context.WithCancel(ctx)
	p.cancel = cancel

	// Initialize providers
	// Android uses /proc filesystem similar to Linux
	p.cpu = newAndroidCPUProvider()
	p.memory = newAndroidMemoryProvider()
	p.network = newAndroidNetworkProvider()
	p.filesystem = newAndroidFilesystemProvider()
	p.battery = newAndroidBatteryProvider()
	p.sensors = newAndroidSensorProvider()

	return nil
}

func (p *androidPlatform) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cancel != nil {
		p.cancel()
	}
	return nil
}

func (p *androidPlatform) CPU() CPUProvider {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.cpu
}

func (p *androidPlatform) Memory() MemoryProvider {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.memory
}

func (p *androidPlatform) Network() NetworkProvider {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.network
}

func (p *androidPlatform) Filesystem() FilesystemProvider {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.filesystem
}

func (p *androidPlatform) Battery() BatteryProvider {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.battery
}

func (p *androidPlatform) Sensors() SensorProvider {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.sensors
}
