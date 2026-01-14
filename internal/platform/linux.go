package platform

import (
	"context"
	"fmt"
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

	// Initialize providers (stub implementations for now)
	p.cpu = &stubCPUProvider{}
	p.memory = &stubMemoryProvider{}
	p.network = &stubNetworkProvider{}
	p.filesystem = &stubFilesystemProvider{}
	p.battery = &stubBatteryProvider{}
	p.sensors = &stubSensorProvider{}

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

// Stub implementations - these will be replaced with real implementations
// when we refactor the existing monitor code in the next task.

type stubCPUProvider struct{}

func (s *stubCPUProvider) Usage() ([]float64, error) {
	return nil, fmt.Errorf("not yet implemented")
}

func (s *stubCPUProvider) TotalUsage() (float64, error) {
	return 0, fmt.Errorf("not yet implemented")
}

func (s *stubCPUProvider) Frequency() ([]float64, error) {
	return nil, fmt.Errorf("not yet implemented")
}

func (s *stubCPUProvider) Info() (*CPUInfo, error) {
	return nil, fmt.Errorf("not yet implemented")
}

func (s *stubCPUProvider) LoadAverage() (float64, float64, float64, error) {
	return 0, 0, 0, fmt.Errorf("not yet implemented")
}

type stubMemoryProvider struct{}

func (s *stubMemoryProvider) Stats() (*MemoryStats, error) {
	return nil, fmt.Errorf("not yet implemented")
}

func (s *stubMemoryProvider) SwapStats() (*SwapStats, error) {
	return nil, fmt.Errorf("not yet implemented")
}

type stubNetworkProvider struct{}

func (s *stubNetworkProvider) Interfaces() ([]string, error) {
	return nil, fmt.Errorf("not yet implemented")
}

func (s *stubNetworkProvider) Stats(interfaceName string) (*NetworkStats, error) {
	return nil, fmt.Errorf("not yet implemented")
}

func (s *stubNetworkProvider) AllStats() (map[string]*NetworkStats, error) {
	return nil, fmt.Errorf("not yet implemented")
}

type stubFilesystemProvider struct{}

func (s *stubFilesystemProvider) Mounts() ([]MountInfo, error) {
	return nil, fmt.Errorf("not yet implemented")
}

func (s *stubFilesystemProvider) Stats(mountPoint string) (*FilesystemStats, error) {
	return nil, fmt.Errorf("not yet implemented")
}

func (s *stubFilesystemProvider) DiskIO(device string) (*DiskIOStats, error) {
	return nil, fmt.Errorf("not yet implemented")
}

type stubBatteryProvider struct{}

func (s *stubBatteryProvider) Count() int {
	return 0
}

func (s *stubBatteryProvider) Stats(index int) (*BatteryStats, error) {
	return nil, fmt.Errorf("not yet implemented")
}

type stubSensorProvider struct{}

func (s *stubSensorProvider) Temperatures() ([]SensorReading, error) {
	return nil, fmt.Errorf("not yet implemented")
}

func (s *stubSensorProvider) Fans() ([]SensorReading, error) {
	return nil, fmt.Errorf("not yet implemented")
}
