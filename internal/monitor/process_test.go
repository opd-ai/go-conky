package monitor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewProcessReader(t *testing.T) {
	reader := newProcessReader()

	if reader == nil {
		t.Fatal("newProcessReader returned nil")
	}

	if reader.procPath != "/proc" {
		t.Errorf("expected procPath /proc, got %s", reader.procPath)
	}

	if reader.lastCPUTimes == nil {
		t.Error("lastCPUTimes map should be initialized")
	}

	if reader.clkTck != 100.0 {
		t.Errorf("expected clkTck 100.0, got %f", reader.clkTck)
	}
}

func TestProcessReaderReadStats(t *testing.T) {
	// Create a temporary proc directory structure
	tmpDir := t.TempDir()

	reader := &processReader{
		procPath:     tmpDir,
		lastCPUTimes: make(map[int]cpuTime),
		clkTck:       100.0,
	}

	// Create /proc/stat file
	statContent := `cpu  1000 200 300 4000 100 50 25 0 0 0
cpu0 500 100 150 2000 50 25 12 0 0 0
cpu1 500 100 150 2000 50 25 13 0 0 0
`
	if err := os.WriteFile(filepath.Join(tmpDir, "stat"), []byte(statContent), 0o644); err != nil {
		t.Fatalf("failed to create stat file: %v", err)
	}

	// Create /proc/meminfo file
	meminfoContent := `MemTotal:       16000000 kB
MemFree:         8000000 kB
MemAvailable:   10000000 kB
`
	if err := os.WriteFile(filepath.Join(tmpDir, "meminfo"), []byte(meminfoContent), 0o644); err != nil {
		t.Fatalf("failed to create meminfo file: %v", err)
	}

	// Create a process directory with stat file
	pid1Dir := filepath.Join(tmpDir, "1")
	if err := os.MkdirAll(pid1Dir, 0o755); err != nil {
		t.Fatalf("failed to create process directory: %v", err)
	}

	// Process stat format: pid (comm) state ppid pgrp session tty_nr tpgid flags
	// minflt cminflt majflt cmajflt utime stime cutime cstime priority nice
	// num_threads itrealvalue starttime vsize rss ...
	procStatContent := "1 (systemd) S 0 1 1 0 -1 4194560 10000 20000 5 10 100 50 0 0 20 0 1 0 1 200000000 5000 18446744073709551615 1 1 0 0 0 0 671173123 4096 1260 0 0 0 17 0 0 0 0 0 0 0 0 0 0 0 0 0 0"
	if err := os.WriteFile(filepath.Join(pid1Dir, "stat"), []byte(procStatContent), 0o644); err != nil {
		t.Fatalf("failed to create process stat file: %v", err)
	}

	// Create another process
	pid2Dir := filepath.Join(tmpDir, "2")
	if err := os.MkdirAll(pid2Dir, 0o755); err != nil {
		t.Fatalf("failed to create process directory: %v", err)
	}

	procStatContent2 := "2 (kthreadd) S 0 0 0 0 -1 2129984 0 0 0 0 10 5 0 0 20 0 2 0 1 0 0 18446744073709551615 0 0 0 0 0 0 0 2147483647 0 0 0 0 0 1 0 0 0 0 0 0 0 0 0 0 0 0 0"
	if err := os.WriteFile(filepath.Join(pid2Dir, "stat"), []byte(procStatContent2), 0o644); err != nil {
		t.Fatalf("failed to create process stat file: %v", err)
	}

	// Read stats
	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats failed: %v", err)
	}

	// Verify basic stats
	if stats.TotalProcesses != 2 {
		t.Errorf("expected 2 total processes, got %d", stats.TotalProcesses)
	}

	// Both processes are in state 'S' (sleeping)
	if stats.SleepingProcesses != 2 {
		t.Errorf("expected 2 sleeping processes, got %d", stats.SleepingProcesses)
	}

	// Verify top processes lists are populated
	if len(stats.TopCPU) != 2 {
		t.Errorf("expected 2 processes in TopCPU, got %d", len(stats.TopCPU))
	}

	if len(stats.TopMem) != 2 {
		t.Errorf("expected 2 processes in TopMem, got %d", len(stats.TopMem))
	}
}

func TestProcessReaderReadStatsWithRunningProcess(t *testing.T) {
	tmpDir := t.TempDir()

	reader := &processReader{
		procPath:     tmpDir,
		lastCPUTimes: make(map[int]cpuTime),
		clkTck:       100.0,
	}

	// Create /proc/stat file
	if err := os.WriteFile(filepath.Join(tmpDir, "stat"), []byte("cpu  1000 200 300 4000 100 50 25 0 0 0\n"), 0o644); err != nil {
		t.Fatalf("failed to create stat file: %v", err)
	}

	// Create /proc/meminfo file
	if err := os.WriteFile(filepath.Join(tmpDir, "meminfo"), []byte("MemTotal: 16000000 kB\n"), 0o644); err != nil {
		t.Fatalf("failed to create meminfo file: %v", err)
	}

	// Create a running process
	pid1Dir := filepath.Join(tmpDir, "1")
	if err := os.MkdirAll(pid1Dir, 0o755); err != nil {
		t.Fatalf("failed to create process directory: %v", err)
	}

	// State 'R' = Running
	procStatContent := "1 (stress) R 0 1 1 0 -1 4194560 10000 20000 5 10 5000 2500 0 0 20 0 4 0 1 200000000 10000 18446744073709551615 1 1 0 0 0 0 671173123 4096 1260 0 0 0 17 0 0 0 0 0 0 0 0 0 0 0 0 0 0"
	if err := os.WriteFile(filepath.Join(pid1Dir, "stat"), []byte(procStatContent), 0o644); err != nil {
		t.Fatalf("failed to create process stat file: %v", err)
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats failed: %v", err)
	}

	if stats.RunningProcesses != 1 {
		t.Errorf("expected 1 running process, got %d", stats.RunningProcesses)
	}

	if stats.TotalThreads != 4 {
		t.Errorf("expected 4 total threads, got %d", stats.TotalThreads)
	}
}

func TestProcessReaderZombieProcess(t *testing.T) {
	tmpDir := t.TempDir()

	reader := &processReader{
		procPath:     tmpDir,
		lastCPUTimes: make(map[int]cpuTime),
		clkTck:       100.0,
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "stat"), []byte("cpu  1000 200 300 4000 100 50 25 0 0 0\n"), 0o644); err != nil {
		t.Fatalf("failed to create stat file: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "meminfo"), []byte("MemTotal: 16000000 kB\n"), 0o644); err != nil {
		t.Fatalf("failed to create meminfo file: %v", err)
	}

	pid1Dir := filepath.Join(tmpDir, "1")
	if err := os.MkdirAll(pid1Dir, 0o755); err != nil {
		t.Fatalf("failed to create process directory: %v", err)
	}

	// State 'Z' = Zombie
	procStatContent := "1 (defunct) Z 0 1 1 0 -1 4194560 0 0 0 0 0 0 0 0 20 0 1 0 1 0 0 18446744073709551615 0 0 0 0 0 0 0 0 0 0 0 0 17 0 0 0 0 0 0 0 0 0 0 0 0 0 0"
	if err := os.WriteFile(filepath.Join(pid1Dir, "stat"), []byte(procStatContent), 0o644); err != nil {
		t.Fatalf("failed to create process stat file: %v", err)
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats failed: %v", err)
	}

	if stats.ZombieProcesses != 1 {
		t.Errorf("expected 1 zombie process, got %d", stats.ZombieProcesses)
	}
}

func TestProcessReaderStoppedProcess(t *testing.T) {
	tmpDir := t.TempDir()

	reader := &processReader{
		procPath:     tmpDir,
		lastCPUTimes: make(map[int]cpuTime),
		clkTck:       100.0,
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "stat"), []byte("cpu  1000 200 300 4000 100 50 25 0 0 0\n"), 0o644); err != nil {
		t.Fatalf("failed to create stat file: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "meminfo"), []byte("MemTotal: 16000000 kB\n"), 0o644); err != nil {
		t.Fatalf("failed to create meminfo file: %v", err)
	}

	pid1Dir := filepath.Join(tmpDir, "1")
	if err := os.MkdirAll(pid1Dir, 0o755); err != nil {
		t.Fatalf("failed to create process directory: %v", err)
	}

	// State 'T' = Stopped
	procStatContent := "1 (vim) T 0 1 1 0 -1 4194560 10000 20000 5 10 100 50 0 0 20 0 1 0 1 200000000 5000 18446744073709551615 1 1 0 0 0 0 671173123 4096 1260 0 0 0 17 0 0 0 0 0 0 0 0 0 0 0 0 0 0"
	if err := os.WriteFile(filepath.Join(pid1Dir, "stat"), []byte(procStatContent), 0o644); err != nil {
		t.Fatalf("failed to create process stat file: %v", err)
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats failed: %v", err)
	}

	if stats.StoppedProcesses != 1 {
		t.Errorf("expected 1 stopped process, got %d", stats.StoppedProcesses)
	}
}

func TestProcessReaderParseProcessStat(t *testing.T) {
	reader := &processReader{
		procPath:         "/proc",
		lastCPUTimes:     make(map[int]cpuTime),
		clkTck:           100.0,
		totalMemoryBytes: 16384 * 1024 * 1024, // 16GB
	}

	tests := []struct {
		name          string
		content       string
		wantErr       bool
		expectedName  string
		expectedState string
	}{
		{
			name:          "normal process",
			content:       "1 (systemd) S 0 1 1 0 -1 4194560 10000 20000 5 10 100 50 0 0 20 0 1 0 1 200000000 5000 18446744073709551615 1 1 0 0 0 0 671173123 4096 1260 0 0 0 17 0 0 0 0 0 0 0 0 0 0 0 0 0 0",
			wantErr:       false,
			expectedName:  "systemd",
			expectedState: "S",
		},
		{
			name:          "process with spaces in name",
			content:       "1234 (Web Content) S 0 1 1 0 -1 4194560 10000 20000 5 10 100 50 0 0 20 0 1 0 1 200000000 5000 18446744073709551615 1 1 0 0 0 0 671173123 4096 1260 0 0 0 17 0 0 0 0 0 0 0 0 0 0 0 0 0 0",
			wantErr:       false,
			expectedName:  "Web Content",
			expectedState: "S",
		},
		{
			name:          "process with parentheses in name",
			content:       "1234 (test(1)) R 0 1 1 0 -1 4194560 10000 20000 5 10 100 50 0 0 20 0 1 0 1 200000000 5000 18446744073709551615 1 1 0 0 0 0 671173123 4096 1260 0 0 0 17 0 0 0 0 0 0 0 0 0 0 0 0 0 0",
			wantErr:       false,
			expectedName:  "test(1)",
			expectedState: "R",
		},
		{
			name:    "missing parentheses",
			content: "1234 systemd S 0 1 1 0 -1 4194560",
			wantErr: true,
		},
		{
			name:    "not enough fields",
			content: "1234 (systemd) S 0 1 1 0 -1 4194560",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proc := ProcessInfo{PID: 1}
			var ct cpuTime

			err := reader.parseProcessStat(&proc, &ct, tt.content, 0)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if proc.Name != tt.expectedName {
				t.Errorf("expected name %q, got %q", tt.expectedName, proc.Name)
			}

			if proc.State != tt.expectedState {
				t.Errorf("expected state %q, got %q", tt.expectedState, proc.State)
			}
		})
	}
}

func TestProcessReaderCPUPercentCalculation(t *testing.T) {
	tmpDir := t.TempDir()

	reader := &processReader{
		procPath:     tmpDir,
		lastCPUTimes: make(map[int]cpuTime),
		clkTck:       100.0,
	}

	// Create base files
	if err := os.WriteFile(filepath.Join(tmpDir, "meminfo"), []byte("MemTotal: 16000000 kB\n"), 0o644); err != nil {
		t.Fatalf("failed to create meminfo file: %v", err)
	}

	pid1Dir := filepath.Join(tmpDir, "1")
	if err := os.MkdirAll(pid1Dir, 0o755); err != nil {
		t.Fatalf("failed to create process directory: %v", err)
	}

	// First sample - total CPU time = 10000, process CPU time = 1000 (utime=500, stime=500)
	if err := os.WriteFile(filepath.Join(tmpDir, "stat"), []byte("cpu  2000 1000 1000 5000 500 250 250 0 0 0\n"), 0o644); err != nil {
		t.Fatalf("failed to create stat file: %v", err)
	}
	procStatContent := "1 (test) R 0 1 1 0 -1 4194560 0 0 0 0 500 500 0 0 20 0 1 0 1 200000000 5000 18446744073709551615 1 1 0 0 0 0 0 0 0 0 0 0 17 0 0 0 0 0 0 0 0 0 0 0 0 0 0"
	if err := os.WriteFile(filepath.Join(pid1Dir, "stat"), []byte(procStatContent), 0o644); err != nil {
		t.Fatalf("failed to create process stat file: %v", err)
	}

	// First read (establishes baseline)
	_, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("first ReadStats failed: %v", err)
	}

	// Second sample - CPU time increased
	// Total CPU delta = 1000, process CPU delta = 100 (10% CPU)
	if err := os.WriteFile(filepath.Join(tmpDir, "stat"), []byte("cpu  2200 1100 1100 5400 550 275 275 0 0 0\n"), 0o644); err != nil {
		t.Fatalf("failed to update stat file: %v", err)
	}
	procStatContent2 := "1 (test) R 0 1 1 0 -1 4194560 0 0 0 0 550 550 0 0 20 0 1 0 1 200000000 5000 18446744073709551615 1 1 0 0 0 0 0 0 0 0 0 0 17 0 0 0 0 0 0 0 0 0 0 0 0 0 0"
	if err := os.WriteFile(filepath.Join(pid1Dir, "stat"), []byte(procStatContent2), 0o644); err != nil {
		t.Fatalf("failed to update process stat file: %v", err)
	}

	// Second read (should calculate CPU percentage)
	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("second ReadStats failed: %v", err)
	}

	if len(stats.TopCPU) == 0 {
		t.Fatal("no processes in TopCPU")
	}

	// CPU percentage should be approximately (100/1000) * 100 = 10%
	// But due to implementation details it may be slightly higher
	// Allow tolerance for floating point and calculation differences
	cpuPercent := stats.TopCPU[0].CPUPercent
	if cpuPercent < 9.0 || cpuPercent > 12.0 {
		t.Errorf("expected CPU percent around 10%%, got %f%%", cpuPercent)
	}
}

func TestProcessReaderMemoryPercentCalculation(t *testing.T) {
	tmpDir := t.TempDir()

	reader := &processReader{
		procPath:     tmpDir,
		lastCPUTimes: make(map[int]cpuTime),
		clkTck:       100.0,
	}

	// Total memory: 16GB
	if err := os.WriteFile(filepath.Join(tmpDir, "meminfo"), []byte("MemTotal: 16777216 kB\n"), 0o644); err != nil {
		t.Fatalf("failed to create meminfo file: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "stat"), []byte("cpu  1000 200 300 4000 100 50 25 0 0 0\n"), 0o644); err != nil {
		t.Fatalf("failed to create stat file: %v", err)
	}

	pid1Dir := filepath.Join(tmpDir, "1")
	if err := os.MkdirAll(pid1Dir, 0o755); err != nil {
		t.Fatalf("failed to create process directory: %v", err)
	}

	// RSS = 4194304 pages * 4096 bytes = 16GB (but that's not right in this context)
	// Let's use RSS = 409600 pages = 1.6GB which is 10% of 16GB
	procStatContent := "1 (bigproc) S 0 1 1 0 -1 4194560 0 0 0 0 100 50 0 0 20 0 1 0 1 200000000 409600 18446744073709551615 1 1 0 0 0 0 0 0 0 0 0 0 17 0 0 0 0 0 0 0 0 0 0 0 0 0 0"
	if err := os.WriteFile(filepath.Join(pid1Dir, "stat"), []byte(procStatContent), 0o644); err != nil {
		t.Fatalf("failed to create process stat file: %v", err)
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats failed: %v", err)
	}

	if len(stats.TopMem) == 0 {
		t.Fatal("no processes in TopMem")
	}

	// RSS = 409600 pages * 4096 = 1,677,721,600 bytes
	// Total = 16777216 kB * 1024 = 17,179,869,184 bytes
	// Percent = (1677721600 / 17179869184) * 100 ≈ 9.77%
	memPercent := stats.TopMem[0].MemPercent
	if memPercent < 9.0 || memPercent > 10.5 {
		t.Errorf("expected memory percent around 9.8%%, got %f%%", memPercent)
	}
}

func TestProcessReaderNonPIDDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	reader := &processReader{
		procPath:     tmpDir,
		lastCPUTimes: make(map[int]cpuTime),
		clkTck:       100.0,
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "stat"), []byte("cpu  1000 200 300 4000 100 50 25 0 0 0\n"), 0o644); err != nil {
		t.Fatalf("failed to create stat file: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "meminfo"), []byte("MemTotal: 16000000 kB\n"), 0o644); err != nil {
		t.Fatalf("failed to create meminfo file: %v", err)
	}

	// Create non-PID directories that should be ignored
	for _, name := range []string{"self", "net", "sys", "bus"} {
		if err := os.MkdirAll(filepath.Join(tmpDir, name), 0o755); err != nil {
			t.Fatalf("failed to create %s directory: %v", name, err)
		}
	}

	// Create one actual PID directory
	pid1Dir := filepath.Join(tmpDir, "1")
	if err := os.MkdirAll(pid1Dir, 0o755); err != nil {
		t.Fatalf("failed to create process directory: %v", err)
	}

	procStatContent := "1 (init) S 0 1 1 0 -1 4194560 0 0 0 0 100 50 0 0 20 0 1 0 1 200000000 5000 18446744073709551615 1 1 0 0 0 0 0 0 0 0 0 0 17 0 0 0 0 0 0 0 0 0 0 0 0 0 0"
	if err := os.WriteFile(filepath.Join(pid1Dir, "stat"), []byte(procStatContent), 0o644); err != nil {
		t.Fatalf("failed to create process stat file: %v", err)
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats failed: %v", err)
	}

	if stats.TotalProcesses != 1 {
		t.Errorf("expected 1 process, got %d", stats.TotalProcesses)
	}
}

func TestProcessReaderTopProcessLimit(t *testing.T) {
	tmpDir := t.TempDir()

	reader := &processReader{
		procPath:     tmpDir,
		lastCPUTimes: make(map[int]cpuTime),
		clkTck:       100.0,
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "stat"), []byte("cpu  1000 200 300 4000 100 50 25 0 0 0\n"), 0o644); err != nil {
		t.Fatalf("failed to create stat file: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "meminfo"), []byte("MemTotal: 16000000 kB\n"), 0o644); err != nil {
		t.Fatalf("failed to create meminfo file: %v", err)
	}

	// Create more than topProcessCount processes
	for i := 1; i <= 20; i++ {
		pidDir := filepath.Join(tmpDir, fmt.Sprintf("%d", i))
		if err := os.MkdirAll(pidDir, 0o755); err != nil {
			t.Fatalf("failed to create process directory: %v", err)
		}

		// RSS increases with PID for testing sort order
		rss := 1000 + i*100
		procStatContent := fmt.Sprintf("%d (proc%d) S 0 1 1 0 -1 4194560 0 0 0 0 %d 0 0 0 20 0 1 0 1 200000000 %d 18446744073709551615 1 1 0 0 0 0 0 0 0 0 0 0 17 0 0 0 0 0 0 0 0 0 0 0 0 0 0", i, i, i*10, rss)
		if err := os.WriteFile(filepath.Join(pidDir, "stat"), []byte(procStatContent), 0o644); err != nil {
			t.Fatalf("failed to create process stat file: %v", err)
		}
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats failed: %v", err)
	}

	if stats.TotalProcesses != 20 {
		t.Errorf("expected 20 total processes, got %d", stats.TotalProcesses)
	}

	if len(stats.TopCPU) != TopProcessCount {
		t.Errorf("expected %d TopCPU processes, got %d", TopProcessCount, len(stats.TopCPU))
	}

	if len(stats.TopMem) != TopProcessCount {
		t.Errorf("expected %d TopMem processes, got %d", TopProcessCount, len(stats.TopMem))
	}

	// TopMem should be sorted by memory (highest first)
	// PID 20 should have highest memory (RSS = 1000 + 20*100 = 3000)
	if len(stats.TopMem) > 0 && stats.TopMem[0].PID != 20 {
		t.Errorf("expected PID 20 to be first in TopMem, got PID %d", stats.TopMem[0].PID)
	}
}

func TestProcessReaderMissingStatFile(t *testing.T) {
	tmpDir := t.TempDir()

	reader := &processReader{
		procPath:     tmpDir,
		lastCPUTimes: make(map[int]cpuTime),
		clkTck:       100.0,
	}

	// Missing /proc/stat should cause an error
	if err := os.WriteFile(filepath.Join(tmpDir, "meminfo"), []byte("MemTotal: 16000000 kB\n"), 0o644); err != nil {
		t.Fatalf("failed to create meminfo file: %v", err)
	}

	_, err := reader.ReadStats()
	if err == nil {
		t.Error("expected error for missing /proc/stat")
	}
}

func TestProcessReaderMissingMeminfo(t *testing.T) {
	tmpDir := t.TempDir()

	reader := &processReader{
		procPath:     tmpDir,
		lastCPUTimes: make(map[int]cpuTime),
		clkTck:       100.0,
	}

	// Create stat but not meminfo
	if err := os.WriteFile(filepath.Join(tmpDir, "stat"), []byte("cpu  1000 200 300 4000 100 50 25 0 0 0\n"), 0o644); err != nil {
		t.Fatalf("failed to create stat file: %v", err)
	}

	_, err := reader.ReadStats()
	if err == nil {
		t.Error("expected error for missing /proc/meminfo")
	}

	if !strings.Contains(err.Error(), "total memory") {
		t.Errorf("expected 'total memory' in error message, got: %v", err)
	}
}

func TestProcessStatsGetterSetter(t *testing.T) {
	sd := NewSystemData()

	initialStats := sd.GetProcess()
	if initialStats.TotalProcesses != 0 {
		t.Errorf("expected initial TotalProcesses 0, got %d", initialStats.TotalProcesses)
	}

	newStats := ProcessStats{
		TotalProcesses:    100,
		RunningProcesses:  5,
		SleepingProcesses: 90,
		ZombieProcesses:   1,
		StoppedProcesses:  4,
		TotalThreads:      500,
		TopCPU: []ProcessInfo{
			{PID: 1, Name: "test", CPUPercent: 50.0},
		},
		TopMem: []ProcessInfo{
			{PID: 2, Name: "big", MemBytes: 1024 * 1024 * 1024},
		},
	}

	sd.setProcess(newStats)

	retrieved := sd.GetProcess()

	if retrieved.TotalProcesses != 100 {
		t.Errorf("expected TotalProcesses 100, got %d", retrieved.TotalProcesses)
	}

	if retrieved.RunningProcesses != 5 {
		t.Errorf("expected RunningProcesses 5, got %d", retrieved.RunningProcesses)
	}

	if len(retrieved.TopCPU) != 1 {
		t.Errorf("expected 1 TopCPU process, got %d", len(retrieved.TopCPU))
	}

	if len(retrieved.TopMem) != 1 {
		t.Errorf("expected 1 TopMem process, got %d", len(retrieved.TopMem))
	}

	// Verify deep copy - modifying retrieved should not affect original
	retrieved.TopCPU[0].CPUPercent = 75.0
	original := sd.GetProcess()
	if original.TopCPU[0].CPUPercent == 75.0 {
		t.Error("deep copy failed - modification affected original")
	}
}
