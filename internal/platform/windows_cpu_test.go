// +build windows

package platform

import (
	"testing"
	"time"
)

func TestWindowsCPUProvider_TotalUsage(t *testing.T) {
	provider := newWindowsCPUProvider()

	usage, err := provider.TotalUsage()
	if err != nil {
		t.Fatalf("TotalUsage() error = %v", err)
	}

	if usage < 0 || usage > 100 {
		t.Errorf("TotalUsage = %v, want 0-100", usage)
	}
}

func TestWindowsCPUProvider_Usage(t *testing.T) {
	provider := newWindowsCPUProvider()

	usage, err := provider.Usage()
	if err != nil {
		t.Fatalf("Usage() error = %v", err)
	}

	if len(usage) == 0 {
		t.Error("Usage() returned empty slice")
	}

	for i, u := range usage {
		if u < 0 || u > 100 {
			t.Errorf("Usage[%d] = %v, want 0-100", i, u)
		}
	}
}

func TestWindowsCPUProvider_Info(t *testing.T) {
	provider := newWindowsCPUProvider()

	info, err := provider.Info()
	if err != nil {
		t.Fatalf("Info() error = %v", err)
	}

	if info.Cores <= 0 {
		t.Errorf("Cores = %v, want > 0", info.Cores)
	}

	if info.Threads <= 0 {
		t.Errorf("Threads = %v, want > 0", info.Threads)
	}
}

func TestWindowsCPUProvider_LoadAverage(t *testing.T) {
	provider := newWindowsCPUProvider()

	_, _, _, err := provider.LoadAverage()
	if err == nil {
		t.Error("LoadAverage() should return error on Windows")
	}
}

func TestWindowsCPUProvider_MultipleCalls(t *testing.T) {
	provider := newWindowsCPUProvider()

	// First call
	usage1, err := provider.TotalUsage()
	if err != nil {
		t.Fatalf("First TotalUsage() error = %v", err)
	}

	// Wait a bit
	time.Sleep(200 * time.Millisecond)

	// Second call
	usage2, err := provider.TotalUsage()
	if err != nil {
		t.Fatalf("Second TotalUsage() error = %v", err)
	}

	// Both should be valid percentages
	if usage1 < 0 || usage1 > 100 {
		t.Errorf("First usage = %v, want 0-100", usage1)
	}
	if usage2 < 0 || usage2 > 100 {
		t.Errorf("Second usage = %v, want 0-100", usage2)
	}
}
