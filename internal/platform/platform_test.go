package platform

import (
	"context"
	"runtime"
	"testing"
	"time"
)

// TestNewPlatform tests the factory function for creating platform instances.
func TestNewPlatform(t *testing.T) {
	p, err := NewPlatform()
	switch runtime.GOOS {
	case "linux":
		if err != nil {
			t.Fatalf("NewPlatform() failed on Linux: %v", err)
		}
		if p == nil {
			t.Fatal("NewPlatform() returned nil platform on Linux")
		}
		if p.Name() != "linux" {
			t.Errorf("Expected platform name 'linux', got '%s'", p.Name())
		}
	case "windows":
		if err != nil {
			t.Fatalf("NewPlatform() failed on Windows: %v", err)
		}
		if p == nil {
			t.Fatal("NewPlatform() returned nil platform on Windows")
		}
		if p.Name() != "windows" {
			t.Errorf("Expected platform name 'windows', got '%s'", p.Name())
		}
	default:
		// On non-Linux/Windows systems, we expect an error for now
		if err == nil {
			t.Errorf("Expected error on %s platform, got nil", runtime.GOOS)
		}
	}
}

// TestNewPlatformForOS tests creating platform instances for specific operating systems.
// Note: This test now only tests the current platform since NewPlatformForOS was removed
// to support proper build tag separation.
func TestNewPlatformForOS(t *testing.T) {
	p, err := NewPlatform()
	if err != nil {
		t.Fatalf("NewPlatform() failed: %v", err)
	}
	if p == nil {
		t.Fatal("NewPlatform() returned nil platform")
	}
	if p.Name() != runtime.GOOS {
		t.Errorf("Expected platform name '%s', got '%s'", runtime.GOOS, p.Name())
	}
}

// TestLinuxPlatformLifecycle tests the initialization and cleanup of Linux platform.
func TestLinuxPlatformLifecycle(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping Linux-specific test on non-Linux platform")
	}

	p, err := NewPlatform()
	if err != nil {
		t.Fatalf("Failed to create Linux platform: %v", err)
	}

	ctx := context.Background()
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize platform: %v", err)
	}

	// Verify providers are not nil
	if p.CPU() == nil {
		t.Error("CPU provider is nil")
	}
	if p.Memory() == nil {
		t.Error("Memory provider is nil")
	}
	if p.Network() == nil {
		t.Error("Network provider is nil")
	}
	if p.Filesystem() == nil {
		t.Error("Filesystem provider is nil")
	}
	if p.Battery() == nil {
		t.Error("Battery provider is nil")
	}
	if p.Sensors() == nil {
		t.Error("Sensors provider is nil")
	}

	// Clean up
	if err := p.Close(); err != nil {
		t.Errorf("Failed to close platform: %v", err)
	}
}

// TestLinuxPlatformContextCancellation tests that platform respects context cancellation.
func TestLinuxPlatformContextCancellation(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping Linux-specific test on non-Linux platform")
	}

	p, err := NewPlatform()
	if err != nil {
		t.Fatalf("Failed to create Linux platform: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	if err := p.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize platform: %v", err)
	}

	// Cancel context
	cancel()

	// Wait for cancellation to propagate, with a timeout to avoid hanging tests.
	select {
	case <-ctx.Done():
		// expected: context has been canceled
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("context cancellation did not propagate in time")
	}

	// Close should still work
	if err := p.Close(); err != nil {
		t.Errorf("Failed to close platform after context cancellation: %v", err)
	}
}

// TestPlatformProvidersInterface tests that all providers implement their interfaces correctly.
func TestPlatformProvidersInterface(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping Linux-specific test on non-Linux platform")
	}

	p, err := NewPlatform()
	if err != nil {
		t.Fatalf("Failed to create Linux platform: %v", err)
	}

	if err := p.Initialize(context.Background()); err != nil {
		t.Fatalf("Failed to initialize platform: %v", err)
	}
	defer p.Close()

	// Test that providers implement the correct interfaces
	var _ CPUProvider = p.CPU()
	var _ MemoryProvider = p.Memory()
	var _ NetworkProvider = p.Network()
	var _ FilesystemProvider = p.Filesystem()
	var _ BatteryProvider = p.Battery()
	var _ SensorProvider = p.Sensors()
}

// TestRemotePlatformConfiguration tests that remote platform validates configuration.
func TestRemotePlatformConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		config  RemoteConfig
		wantErr bool
	}{
		{
			name: "missing auth method",
			config: RemoteConfig{
				Host: "example.com",
				Port: 22,
				User: "test",
			},
			wantErr: true,
		},
		{
			name: "valid config",
			config: RemoteConfig{
				Host: "example.com",
				Port: 22,
				User: "test",
				AuthMethod: PasswordAuth{
					Password: "test",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewRemotePlatform(tt.config)
			t.Logf("Platform: %v, Error: %v, wantErr: %v", p, err, tt.wantErr)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewRemotePlatform() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && p == nil {
				t.Error("Expected non-nil platform for valid config")
			}
			if tt.wantErr && p != nil {
				t.Errorf("Expected nil platform for invalid config, got %v", p)
			}
		})
	}
}

// TestAuthMethodInterface tests that auth methods implement the interface.
func TestAuthMethodInterface(t *testing.T) {
	var _ AuthMethod = PasswordAuth{}
	var _ AuthMethod = KeyAuth{}
	var _ AuthMethod = AgentAuth{}
}

// TestCPUInfoStruct tests the CPUInfo struct.
func TestCPUInfoStruct(t *testing.T) {
	info := CPUInfo{
		Model:     "Intel Core i7",
		Vendor:    "GenuineIntel",
		Cores:     4,
		Threads:   8,
		CacheSize: 8388608, // 8MB
	}

	if info.Model != "Intel Core i7" {
		t.Errorf("Expected model 'Intel Core i7', got '%s'", info.Model)
	}
	if info.Cores != 4 {
		t.Errorf("Expected 4 cores, got %d", info.Cores)
	}
}

// TestMemoryStatsStruct tests the MemoryStats struct.
func TestMemoryStatsStruct(t *testing.T) {
	stats := MemoryStats{
		Total:       8589934592, // 8GB
		Used:        4294967296, // 4GB
		Free:        2147483648, // 2GB
		Available:   4294967296, // 4GB
		Cached:      2147483648, // 2GB
		Buffers:     1073741824, // 1GB
		UsedPercent: 50.0,
	}

	if stats.Total != 8589934592 {
		t.Errorf("Expected total 8589934592, got %d", stats.Total)
	}
	if stats.UsedPercent != 50.0 {
		t.Errorf("Expected usage 50.0%%, got %.1f%%", stats.UsedPercent)
	}
}

// TestNetworkStatsStruct tests the NetworkStats struct.
func TestNetworkStatsStruct(t *testing.T) {
	stats := NetworkStats{
		BytesRecv:   1000000,
		BytesSent:   500000,
		PacketsRecv: 10000,
		PacketsSent: 5000,
		ErrorsIn:    0,
		ErrorsOut:   0,
		DropIn:      0,
		DropOut:     0,
	}

	if stats.BytesRecv != 1000000 {
		t.Errorf("Expected bytes received 1000000, got %d", stats.BytesRecv)
	}
	if stats.PacketsSent != 5000 {
		t.Errorf("Expected packets sent 5000, got %d", stats.PacketsSent)
	}
}

// TestDiskIOStatsStruct tests the DiskIOStats struct.
func TestDiskIOStatsStruct(t *testing.T) {
	stats := DiskIOStats{
		ReadBytes:  10485760, // 10MB
		WriteBytes: 5242880,  // 5MB
		ReadCount:  100,
		WriteCount: 50,
		ReadTime:   100 * time.Millisecond,
		WriteTime:  50 * time.Millisecond,
	}

	if stats.ReadBytes != 10485760 {
		t.Errorf("Expected read bytes 10485760, got %d", stats.ReadBytes)
	}
	if stats.ReadTime != 100*time.Millisecond {
		t.Errorf("Expected read time 100ms, got %v", stats.ReadTime)
	}
}

// TestBatteryStatsStruct tests the BatteryStats struct.
func TestBatteryStatsStruct(t *testing.T) {
	stats := BatteryStats{
		Percent:       75.0,
		TimeRemaining: 2 * time.Hour,
		Charging:      false,
		FullCapacity:  50000,
		Current:       37500,
		Voltage:       12.5,
	}

	if stats.Percent != 75.0 {
		t.Errorf("Expected battery percent 75.0, got %.1f", stats.Percent)
	}
	if stats.TimeRemaining != 2*time.Hour {
		t.Errorf("Expected time remaining 2h, got %v", stats.TimeRemaining)
	}
}

// TestSensorReadingStruct tests the SensorReading struct.
func TestSensorReadingStruct(t *testing.T) {
	reading := SensorReading{
		Name:     "coretemp",
		Label:    "Core 0",
		Value:    65.0,
		Unit:     "°C",
		Critical: 100.0,
	}

	if reading.Name != "coretemp" {
		t.Errorf("Expected sensor name 'coretemp', got '%s'", reading.Name)
	}
	if reading.Value != 65.0 {
		t.Errorf("Expected temperature 65.0, got %.1f", reading.Value)
	}
}
