//go:build darwin
// +build darwin

package platform

import (
	"context"
	"testing"
)

func TestDarwinPlatform_Name(t *testing.T) {
	platform := NewDarwinPlatform()

	if platform.Name() != "darwin" {
		t.Errorf("Expected platform name 'darwin', got '%s'", platform.Name())
	}
}

func TestDarwinPlatform_Initialize(t *testing.T) {
	platform := NewDarwinPlatform()

	err := platform.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	// Verify all providers are initialized
	if platform.CPU() == nil {
		t.Error("CPU provider should not be nil after initialization")
	}

	if platform.Memory() == nil {
		t.Error("Memory provider should not be nil after initialization")
	}

	if platform.Network() == nil {
		t.Error("Network provider should not be nil after initialization")
	}

	if platform.Filesystem() == nil {
		t.Error("Filesystem provider should not be nil after initialization")
	}

	if platform.Battery() == nil {
		t.Error("Battery provider should not be nil after initialization")
	}

	if platform.Sensors() == nil {
		t.Error("Sensors provider should not be nil after initialization")
	}

	// Clean up
	err = platform.Close()
	if err != nil {
		t.Errorf("Close() failed: %v", err)
	}
}

func TestDarwinPlatform_Close(t *testing.T) {
	platform := NewDarwinPlatform()

	err := platform.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	err = platform.Close()
	if err != nil {
		t.Errorf("Close() failed: %v", err)
	}

	// Should be safe to call Close() multiple times
	err = platform.Close()
	if err != nil {
		t.Errorf("Second Close() failed: %v", err)
	}
}

func TestDarwinPlatform_Providers(t *testing.T) {
	platform := NewDarwinPlatform()

	err := platform.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}
	defer platform.Close()

	// Test CPU provider
	cpuUsage, err := platform.CPU().TotalUsage()
	if err != nil {
		t.Errorf("CPU().TotalUsage() failed: %v", err)
	}
	t.Logf("CPU usage: %.2f%%", cpuUsage)

	// Test Memory provider
	memStats, err := platform.Memory().Stats()
	if err != nil {
		t.Errorf("Memory().Stats() failed: %v", err)
	} else {
		t.Logf("Memory: %d MB used / %d MB total",
			memStats.Used/1024/1024, memStats.Total/1024/1024)
	}

	// Test Network provider
	interfaces, err := platform.Network().Interfaces()
	if err != nil {
		t.Errorf("Network().Interfaces() failed: %v", err)
	} else {
		t.Logf("Found %d network interfaces", len(interfaces))
	}

	// Test Filesystem provider
	mounts, err := platform.Filesystem().Mounts()
	if err != nil {
		t.Errorf("Filesystem().Mounts() failed: %v", err)
	} else {
		t.Logf("Found %d mounted filesystems", len(mounts))
	}

	// Test Battery provider
	batteryCount := platform.Battery().Count()
	t.Logf("Found %d batteries", batteryCount)

	// Test Sensors provider
	temps, err := platform.Sensors().Temperatures()
	if err != nil {
		t.Errorf("Sensors().Temperatures() failed: %v", err)
	} else {
		t.Logf("Found %d temperature sensors", len(temps))
	}
}
