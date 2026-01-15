//go:build android
// +build android

package platform

import (
	"context"
	"testing"
)

func TestAndroidPlatform_Initialize(t *testing.T) {
	platform := NewAndroidPlatform()

	if platform.Name() != "android" {
		t.Errorf("Name() = %v, want android", platform.Name())
	}

	ctx := context.Background()
	err := platform.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}
	defer platform.Close()

	// Check that all providers are initialized
	if platform.CPU() == nil {
		t.Error("CPU() returned nil")
	}
	if platform.Memory() == nil {
		t.Error("Memory() returned nil")
	}
	if platform.Network() == nil {
		t.Error("Network() returned nil")
	}
	if platform.Filesystem() == nil {
		t.Error("Filesystem() returned nil")
	}
	if platform.Battery() == nil {
		t.Error("Battery() returned nil")
	}
	if platform.Sensors() == nil {
		t.Error("Sensors() returned nil")
	}
}

func TestAndroidPlatform_Close(t *testing.T) {
	platform := NewAndroidPlatform()

	ctx := context.Background()
	err := platform.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}

	err = platform.Close()
	if err != nil {
		t.Errorf("Close() failed: %v", err)
	}

	// Close should be idempotent
	err = platform.Close()
	if err != nil {
		t.Errorf("Second Close() failed: %v", err)
	}
}

func TestAndroidPlatform_ConcurrentAccess(t *testing.T) {
	platform := NewAndroidPlatform()

	ctx := context.Background()
	err := platform.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}
	defer platform.Close()

	// Test concurrent access to providers
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			_ = platform.CPU()
			_ = platform.Memory()
			_ = platform.Network()
			_ = platform.Filesystem()
			_ = platform.Battery()
			_ = platform.Sensors()
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
