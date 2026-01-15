//go:build android
// +build android

package platform

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAndroidCPUProvider_TotalUsage(t *testing.T) {
	// Create a temporary /proc/stat file
	tmpDir := t.TempDir()
	statPath := filepath.Join(tmpDir, "stat")

	// Write initial /proc/stat content
	initialContent := `cpu  100 0 50 850 0 0 0 0 0 0
cpu0 25 0 12 212 0 0 0 0 0 0
cpu1 25 0 13 213 0 0 0 0 0 0
cpu2 25 0 12 212 0 0 0 0 0 0
cpu3 25 0 13 213 0 0 0 0 0 0
`
	if err := os.WriteFile(statPath, []byte(initialContent), 0o644); err != nil {
		t.Fatalf("Failed to write initial stat file: %v", err)
	}

	provider := &androidCPUProvider{
		prevStats:    make(map[int]cpuTimes),
		procStatPath: statPath,
	}

	// First read (should return 0 as there's no previous data)
	usage, err := provider.TotalUsage()
	if err != nil {
		t.Fatalf("First TotalUsage() failed: %v", err)
	}
	if usage != 0 {
		t.Errorf("First TotalUsage() = %v, want 0", usage)
	}

	// Write updated /proc/stat content (simulate CPU usage)
	updatedContent := `cpu  200 0 100 900 0 0 0 0 0 0
cpu0 50 0 25 225 0 0 0 0 0 0
cpu1 50 0 25 225 0 0 0 0 0 0
cpu2 50 0 25 225 0 0 0 0 0 0
cpu3 50 0 25 225 0 0 0 0 0 0
`
	if err := os.WriteFile(statPath, []byte(updatedContent), 0o644); err != nil {
		t.Fatalf("Failed to write updated stat file: %v", err)
	}

	// Second read (should show CPU usage)
	usage, err = provider.TotalUsage()
	if err != nil {
		t.Fatalf("Second TotalUsage() failed: %v", err)
	}

	// Expected calculation: 75% usage
	expectedUsage := 75.0
	if usage < expectedUsage-0.1 || usage > expectedUsage+0.1 {
		t.Errorf("Second TotalUsage() = %v, want ~%v", usage, expectedUsage)
	}
}

func TestAndroidCPUProvider_Usage(t *testing.T) {
	tmpDir := t.TempDir()
	statPath := filepath.Join(tmpDir, "stat")

	initialContent := `cpu  100 0 50 850 0 0 0 0 0 0
cpu0 25 0 12 212 0 0 0 0 0 0
cpu1 25 0 13 213 0 0 0 0 0 0
`
	if err := os.WriteFile(statPath, []byte(initialContent), 0o644); err != nil {
		t.Fatalf("Failed to write initial stat file: %v", err)
	}

	provider := &androidCPUProvider{
		prevStats:    make(map[int]cpuTimes),
		procStatPath: statPath,
	}

	usages, err := provider.Usage()
	if err != nil {
		t.Fatalf("First Usage() failed: %v", err)
	}
	if len(usages) != 2 {
		t.Errorf("First Usage() returned %d cores, want 2", len(usages))
	}

	for i, usage := range usages {
		if usage != 0 {
			t.Errorf("First Usage()[%d] = %v, want 0", i, usage)
		}
	}
}

func TestAndroidCPUProvider_LoadAverage(t *testing.T) {
	tmpDir := t.TempDir()
	loadavgPath := filepath.Join(tmpDir, "loadavg")

	content := "0.52 0.58 0.59 3/815 12345\n"
	if err := os.WriteFile(loadavgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write loadavg file: %v", err)
	}

	provider := &androidCPUProvider{
		procLoadavgPath: loadavgPath,
	}

	load1, load5, load15, err := provider.LoadAverage()
	if err != nil {
		t.Fatalf("LoadAverage() failed: %v", err)
	}

	if load1 != 0.52 {
		t.Errorf("load1 = %v, want 0.52", load1)
	}
	if load5 != 0.58 {
		t.Errorf("load5 = %v, want 0.58", load5)
	}
	if load15 != 0.59 {
		t.Errorf("load15 = %v, want 0.59", load15)
	}
}

func TestAndroidCPUProvider_Info(t *testing.T) {
	tmpDir := t.TempDir()
	cpuinfoPath := filepath.Join(tmpDir, "cpuinfo")

	// Android-style cpuinfo
	content := `processor	: 0
Processor	: ARMv7 Processor rev 4 (v7l)
Hardware	: Qualcomm Snapdragon
cpu cores	: 4
siblings	: 8
cache size	: 2048 KB

processor	: 1
Processor	: ARMv7 Processor rev 4 (v7l)
Hardware	: Qualcomm Snapdragon
cpu cores	: 4
siblings	: 8
cache size	: 2048 KB
`
	if err := os.WriteFile(cpuinfoPath, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write cpuinfo file: %v", err)
	}

	provider := &androidCPUProvider{
		procInfoPath: cpuinfoPath,
	}

	info, err := provider.Info()
	if err != nil {
		t.Fatalf("Info() failed: %v", err)
	}

	if info.Model != "ARMv7 Processor rev 4 (v7l)" {
		t.Errorf("Model = %v, want ARMv7 Processor rev 4 (v7l)", info.Model)
	}
	if info.Cores != 4 {
		t.Errorf("Cores = %v, want 4", info.Cores)
	}
	if info.Threads != 8 {
		t.Errorf("Threads = %v, want 8", info.Threads)
	}
}

func TestAndroidCPUProvider_Frequency(t *testing.T) {
	tmpDir := t.TempDir()
	cpuinfoPath := filepath.Join(tmpDir, "cpuinfo")

	content := `processor	: 0
cpu MHz		: 1800.123
cache size	: 2048 KB

processor	: 1
cpu MHz		: 1850.456
cache size	: 2048 KB
`
	if err := os.WriteFile(cpuinfoPath, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write cpuinfo file: %v", err)
	}

	provider := &androidCPUProvider{
		procInfoPath: cpuinfoPath,
	}

	frequencies, err := provider.Frequency()
	if err != nil {
		t.Fatalf("Frequency() failed: %v", err)
	}

	if len(frequencies) != 2 {
		t.Fatalf("Frequency() returned %d values, want 2", len(frequencies))
	}

	if frequencies[0] != 1800.123 {
		t.Errorf("Frequency()[0] = %v, want 1800.123", frequencies[0])
	}
	if frequencies[1] != 1850.456 {
		t.Errorf("Frequency()[1] = %v, want 1850.456", frequencies[1])
	}
}
