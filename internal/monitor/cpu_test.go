package monitor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCPUTimesTotal(t *testing.T) {
	ct := cpuTimes{
		user:    100,
		nice:    10,
		system:  50,
		idle:    500,
		iowait:  20,
		irq:     5,
		softirq: 3,
		steal:   2,
	}

	expected := uint64(690)
	if ct.total() != expected {
		t.Errorf("total() = %d, want %d", ct.total(), expected)
	}
}

func TestCPUTimesIdleTime(t *testing.T) {
	ct := cpuTimes{
		idle:   500,
		iowait: 20,
	}

	expected := uint64(520)
	if ct.idleTime() != expected {
		t.Errorf("idleTime() = %d, want %d", ct.idleTime(), expected)
	}
}

func TestParseCPULine(t *testing.T) {
	tests := []struct {
		name    string
		fields  []string
		want    cpuTimes
		wantErr bool
	}{
		{
			name:   "valid line with all fields",
			fields: []string{"100", "10", "50", "500", "20", "5", "3", "2"},
			want: cpuTimes{
				user:    100,
				nice:    10,
				system:  50,
				idle:    500,
				iowait:  20,
				irq:     5,
				softirq: 3,
				steal:   2,
			},
			wantErr: false,
		},
		{
			name:   "valid line with 7 fields",
			fields: []string{"100", "10", "50", "500", "20", "5", "3"},
			want: cpuTimes{
				user:    100,
				nice:    10,
				system:  50,
				idle:    500,
				iowait:  20,
				irq:     5,
				softirq: 3,
				steal:   0,
			},
			wantErr: false,
		},
		{
			name:    "insufficient fields",
			fields:  []string{"100", "10", "50"},
			wantErr: true,
		},
		{
			name:    "invalid number",
			fields:  []string{"100", "abc", "50", "500", "20", "5", "3"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCPULine(tt.fields)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseCPULine() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseCPULine() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCPUReaderCalculateUsage(t *testing.T) {
	reader := newCPUReader()

	tests := []struct {
		name     string
		prev     cpuTimes
		curr     cpuTimes
		expected float64
	}{
		{
			name: "50% usage",
			prev: cpuTimes{user: 100, nice: 0, system: 0, idle: 100, iowait: 0, irq: 0, softirq: 0, steal: 0},
			curr: cpuTimes{user: 150, nice: 0, system: 0, idle: 150, iowait: 0, irq: 0, softirq: 0, steal: 0},
			// total delta = 100, idle delta = 50, usage = (100-50)/100 * 100 = 50%
			expected: 50.0,
		},
		{
			name: "0% usage (all idle)",
			prev: cpuTimes{user: 100, nice: 0, system: 0, idle: 100, iowait: 0, irq: 0, softirq: 0, steal: 0},
			curr: cpuTimes{user: 100, nice: 0, system: 0, idle: 200, iowait: 0, irq: 0, softirq: 0, steal: 0},
			// total delta = 100, idle delta = 100, usage = (100-100)/100 * 100 = 0%
			expected: 0.0,
		},
		{
			name: "100% usage (no idle)",
			prev: cpuTimes{user: 100, nice: 0, system: 0, idle: 100, iowait: 0, irq: 0, softirq: 0, steal: 0},
			curr: cpuTimes{user: 200, nice: 0, system: 0, idle: 100, iowait: 0, irq: 0, softirq: 0, steal: 0},
			// total delta = 100, idle delta = 0, usage = (100-0)/100 * 100 = 100%
			expected: 100.0,
		},
		{
			name:     "no delta",
			prev:     cpuTimes{user: 100, nice: 0, system: 0, idle: 100, iowait: 0, irq: 0, softirq: 0, steal: 0},
			curr:     cpuTimes{user: 100, nice: 0, system: 0, idle: 100, iowait: 0, irq: 0, softirq: 0, steal: 0},
			expected: 0.0,
		},
		{
			name: "counter wrap-around (curr total < prev total)",
			prev: cpuTimes{user: 1000, nice: 0, system: 0, idle: 1000, iowait: 0, irq: 0, softirq: 0, steal: 0},
			curr: cpuTimes{user: 100, nice: 0, system: 0, idle: 100, iowait: 0, irq: 0, softirq: 0, steal: 0},
			// Should return 0.0 for counter wrap-around
			expected: 0.0,
		},
		{
			name: "idle counter wrap-around",
			prev: cpuTimes{user: 100, nice: 0, system: 0, idle: 1000, iowait: 0, irq: 0, softirq: 0, steal: 0},
			curr: cpuTimes{user: 200, nice: 0, system: 0, idle: 100, iowait: 0, irq: 0, softirq: 0, steal: 0},
			// Should return 0.0 for idle counter wrap-around
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reader.calculateUsage(tt.prev, tt.curr)
			if got != tt.expected {
				t.Errorf("calculateUsage() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCPUReaderCalculateCoreUsage(t *testing.T) {
	reader := newCPUReader()

	prev := []cpuTimes{
		{user: 100, idle: 100},
		{user: 50, idle: 50},
	}
	curr := []cpuTimes{
		{user: 150, idle: 150},
		{user: 100, idle: 100},
	}

	got := reader.calculateCoreUsage(prev, curr)
	if len(got) != 2 {
		t.Fatalf("calculateCoreUsage() returned %d cores, want 2", len(got))
	}

	// Both cores should show 50% usage
	for i, usage := range got {
		if usage != 50.0 {
			t.Errorf("core %d usage = %v, want 50.0", i, usage)
		}
	}
}

func TestCPUReaderWithMockFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create mock /proc/stat
	statContent := `cpu  100 10 50 500 20 5 3 2
cpu0 50 5 25 250 10 2 1 1
cpu1 50 5 25 250 10 3 2 1
intr 12345
`
	if err := os.WriteFile(filepath.Join(tmpDir, "stat"), []byte(statContent), 0644); err != nil {
		t.Fatalf("failed to write mock stat: %v", err)
	}

	// Create mock /proc/cpuinfo
	cpuinfoContent := `processor	: 0
model name	: Test CPU Model
cpu MHz		: 2400.123

processor	: 1
model name	: Test CPU Model
cpu MHz		: 2400.456
`
	if err := os.WriteFile(filepath.Join(tmpDir, "cpuinfo"), []byte(cpuinfoContent), 0644); err != nil {
		t.Fatalf("failed to write mock cpuinfo: %v", err)
	}

	reader := &cpuReader{
		procStatPath: filepath.Join(tmpDir, "stat"),
		procInfoPath: filepath.Join(tmpDir, "cpuinfo"),
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	if stats.CPUCount != 2 {
		t.Errorf("CPUCount = %d, want 2", stats.CPUCount)
	}
	if stats.ModelName != "Test CPU Model" {
		t.Errorf("ModelName = %q, want %q", stats.ModelName, "Test CPU Model")
	}
	if stats.Frequency != 2400.123 {
		t.Errorf("Frequency = %v, want 2400.123", stats.Frequency)
	}
}

func TestCPUReaderMissingFile(t *testing.T) {
	reader := &cpuReader{
		procStatPath: "/nonexistent/stat",
		procInfoPath: "/nonexistent/cpuinfo",
	}

	_, err := reader.ReadStats()
	if err == nil {
		t.Error("ReadStats() should return error for missing file")
	}
}
