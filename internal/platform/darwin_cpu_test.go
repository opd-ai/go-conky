// +build darwin

package platform

import (
	"testing"
)

func TestDarwinCPUProvider_TotalUsage(t *testing.T) {
	provider := newDarwinCPUProvider()

	// First call should return 0 (no previous stats)
	usage, err := provider.TotalUsage()
	if err != nil {
		t.Fatalf("TotalUsage() failed: %v", err)
	}
	if usage != 0 {
		t.Errorf("Expected 0 on first call, got %f", usage)
	}

	// Second call should return a valid percentage
	usage, err = provider.TotalUsage()
	if err != nil {
		t.Fatalf("TotalUsage() failed on second call: %v", err)
	}
	if usage < 0 || usage > 100 {
		t.Errorf("Expected usage between 0 and 100, got %f", usage)
	}
}

func TestDarwinCPUProvider_Usage(t *testing.T) {
	provider := newDarwinCPUProvider()

	usages, err := provider.Usage()
	if err != nil {
		t.Fatalf("Usage() failed: %v", err)
	}

	if len(usages) == 0 {
		t.Error("Expected at least one CPU usage value")
	}

	for i, usage := range usages {
		if usage < 0 || usage > 100 {
			t.Errorf("CPU %d usage out of range: %f", i, usage)
		}
	}
}

func TestDarwinCPUProvider_Frequency(t *testing.T) {
	provider := newDarwinCPUProvider()

	frequencies, err := provider.Frequency()
	if err != nil {
		t.Fatalf("Frequency() failed: %v", err)
	}

	if len(frequencies) == 0 {
		t.Error("Expected at least one frequency value")
	}

	for i, freq := range frequencies {
		if freq <= 0 {
			t.Errorf("CPU %d frequency should be positive, got %f", i, freq)
		}
	}
}

func TestDarwinCPUProvider_Info(t *testing.T) {
	provider := newDarwinCPUProvider()

	info, err := provider.Info()
	if err != nil {
		t.Fatalf("Info() failed: %v", err)
	}

	if info.Model == "" {
		t.Error("CPU model should not be empty")
	}

	if info.Cores <= 0 {
		t.Errorf("Expected positive number of cores, got %d", info.Cores)
	}

	if info.Threads <= 0 {
		t.Errorf("Expected positive number of threads, got %d", info.Threads)
	}

	if info.Threads < info.Cores {
		t.Errorf("Threads (%d) should be >= Cores (%d)", info.Threads, info.Cores)
	}
}

func TestDarwinCPUProvider_LoadAverage(t *testing.T) {
	provider := newDarwinCPUProvider()

	load1, load5, load15, err := provider.LoadAverage()
	if err != nil {
		t.Fatalf("LoadAverage() failed: %v", err)
	}

	if load1 < 0 {
		t.Errorf("1-minute load average should be non-negative, got %f", load1)
	}

	if load5 < 0 {
		t.Errorf("5-minute load average should be non-negative, got %f", load5)
	}

	if load15 < 0 {
		t.Errorf("15-minute load average should be non-negative, got %f", load15)
	}
}
