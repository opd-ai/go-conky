//go:build windows
// +build windows

package platform

import (
	"context"
	"testing"
)

func TestWindowsPlatform_Name(t *testing.T) {
	platform := NewWindowsPlatform()

	if platform.Name() != "windows" {
		t.Errorf("Name() = %v, want windows", platform.Name())
	}
}

func TestWindowsPlatform_Initialize(t *testing.T) {
	platform := NewWindowsPlatform()
	ctx := context.Background()

	err := platform.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	// Clean up
	defer platform.Close()

	// Verify all providers are initialized
	if platform.CPU() == nil {
		t.Error("CPU provider is nil")
	}
	if platform.Memory() == nil {
		t.Error("Memory provider is nil")
	}
	if platform.Network() == nil {
		t.Error("Network provider is nil")
	}
	if platform.Filesystem() == nil {
		t.Error("Filesystem provider is nil")
	}
	if platform.Battery() == nil {
		t.Error("Battery provider is nil")
	}
	if platform.Sensors() == nil {
		t.Error("Sensors provider is nil")
	}
}

func TestWindowsPlatform_Close(t *testing.T) {
	platform := NewWindowsPlatform()
	ctx := context.Background()

	err := platform.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	err = platform.Close()
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestWindowsPlatform_Providers(t *testing.T) {
	platform := NewWindowsPlatform()
	ctx := context.Background()

	err := platform.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	defer platform.Close()

	// Test CPU provider
	cpuUsage, err := platform.CPU().TotalUsage()
	if err != nil {
		t.Errorf("CPU().TotalUsage() error = %v", err)
	}
	if cpuUsage < 0 || cpuUsage > 100 {
		t.Errorf("CPU usage = %v, want 0-100", cpuUsage)
	}

	// Test Memory provider
	memStats, err := platform.Memory().Stats()
	if err != nil {
		t.Errorf("Memory().Stats() error = %v", err)
	}
	if memStats != nil && memStats.Total == 0 {
		t.Error("Memory total is 0")
	}

	// Test Network provider
	interfaces, err := platform.Network().Interfaces()
	if err != nil {
		t.Errorf("Network().Interfaces() error = %v", err)
	}
	if len(interfaces) == 0 {
		t.Error("No network interfaces found")
	}

	// Test Filesystem provider
	mounts, err := platform.Filesystem().Mounts()
	if err != nil {
		t.Errorf("Filesystem().Mounts() error = %v", err)
	}
	if len(mounts) == 0 {
		t.Error("No filesystem mounts found")
	}

	// Test Battery provider
	batteryCount := platform.Battery().Count()
	if batteryCount < 0 || batteryCount > 1 {
		t.Errorf("Battery count = %v, want 0 or 1", batteryCount)
	}

	// Test Sensors provider
	temps, err := platform.Sensors().Temperatures()
	if err != nil {
		t.Errorf("Sensors().Temperatures() error = %v", err)
	}
	if temps == nil {
		t.Error("Temperatures is nil")
	}
}
