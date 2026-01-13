package profiling

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestNew(t *testing.T) {
	config := Config{
		CPUProfilePath: "/tmp/cpu.prof",
		MemProfilePath: "/tmp/mem.prof",
	}

	p := New(config)

	if p == nil {
		t.Fatal("New() returned nil")
	}
	if p.running {
		t.Error("new profiler should not be running")
	}
	if p.memFilePath != "/tmp/mem.prof" {
		t.Errorf("memFilePath = %q, want %q", p.memFilePath, "/tmp/mem.prof")
	}
}

func TestProfilerStartStop(t *testing.T) {
	tmpDir := t.TempDir()
	cpuPath := filepath.Join(tmpDir, "cpu.prof")
	memPath := filepath.Join(tmpDir, "mem.prof")

	config := Config{
		CPUProfilePath: cpuPath,
		MemProfilePath: memPath,
	}

	p := New(config)

	// Start profiling
	if err := p.Start(cpuPath); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	if !p.IsRunning() {
		t.Error("IsRunning() should return true after Start()")
	}

	// Attempt to start again should fail
	if err := p.Start(cpuPath); err == nil {
		t.Error("Start() should fail when already running")
	}

	// Stop profiling
	if err := p.Stop(); err != nil {
		t.Fatalf("Stop() failed: %v", err)
	}

	if p.IsRunning() {
		t.Error("IsRunning() should return false after Stop()")
	}

	// Verify CPU profile was written
	if _, err := os.Stat(cpuPath); os.IsNotExist(err) {
		t.Error("CPU profile file was not created")
	}

	// Verify memory profile was written
	if _, err := os.Stat(memPath); os.IsNotExist(err) {
		t.Error("memory profile file was not created")
	}
}

func TestProfilerStopWithoutStart(t *testing.T) {
	p := New(Config{})

	if err := p.Stop(); err == nil {
		t.Error("Stop() should fail when profiler is not running")
	}
}

func TestProfilerCPUOnly(t *testing.T) {
	tmpDir := t.TempDir()
	cpuPath := filepath.Join(tmpDir, "cpu.prof")

	p := New(Config{})

	if err := p.Start(cpuPath); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	if err := p.Stop(); err != nil {
		t.Fatalf("Stop() failed: %v", err)
	}

	// Verify CPU profile was written
	if _, err := os.Stat(cpuPath); os.IsNotExist(err) {
		t.Error("CPU profile file was not created")
	}
}

func TestProfilerMemoryOnly(t *testing.T) {
	tmpDir := t.TempDir()
	memPath := filepath.Join(tmpDir, "mem.prof")

	config := Config{
		MemProfilePath: memPath,
	}
	p := New(config)

	// Start with empty CPU path
	if err := p.Start(""); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	if err := p.Stop(); err != nil {
		t.Fatalf("Stop() failed: %v", err)
	}

	// Verify memory profile was written
	if _, err := os.Stat(memPath); os.IsNotExist(err) {
		t.Error("memory profile file was not created")
	}
}

func TestProfilerNoProfiling(t *testing.T) {
	p := New(Config{})

	if err := p.Start(""); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	if !p.IsRunning() {
		t.Error("IsRunning() should return true even with no profiling configured")
	}

	if err := p.Stop(); err != nil {
		t.Fatalf("Stop() failed: %v", err)
	}
}

func TestProfilerSetMemProfilePath(t *testing.T) {
	p := New(Config{})

	// Should work when not running
	if err := p.SetMemProfilePath("/tmp/mem.prof"); err != nil {
		t.Fatalf("SetMemProfilePath() failed: %v", err)
	}

	if p.memFilePath != "/tmp/mem.prof" {
		t.Errorf("memFilePath = %q, want %q", p.memFilePath, "/tmp/mem.prof")
	}

	// Start profiling
	if err := p.Start(""); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	// Should fail when running
	if err := p.SetMemProfilePath("/tmp/other.prof"); err == nil {
		t.Error("SetMemProfilePath() should fail when running")
	}

	if err := p.Stop(); err != nil {
		t.Fatalf("Stop() failed: %v", err)
	}
}

func TestProfilerWriteMemProfileNow(t *testing.T) {
	tmpDir := t.TempDir()
	memPath := filepath.Join(tmpDir, "snapshot.prof")

	p := New(Config{})

	if err := p.WriteMemProfileNow(memPath); err != nil {
		t.Fatalf("WriteMemProfileNow() failed: %v", err)
	}

	// Verify memory profile was written
	info, err := os.Stat(memPath)
	if os.IsNotExist(err) {
		t.Error("memory profile file was not created")
	}
	if info.Size() == 0 {
		t.Error("memory profile file should not be empty")
	}
}

func TestProfilerConcurrency(t *testing.T) {
	tmpDir := t.TempDir()
	cpuPath := filepath.Join(tmpDir, "cpu.prof")
	memPath := filepath.Join(tmpDir, "mem.prof")

	config := Config{
		MemProfilePath: memPath,
	}
	p := New(config)

	if err := p.Start(cpuPath); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	var wg sync.WaitGroup

	// Concurrent reads of IsRunning
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = p.IsRunning()
			}
		}()
	}

	wg.Wait()

	if err := p.Stop(); err != nil {
		t.Fatalf("Stop() failed: %v", err)
	}
}

func TestProfilerInvalidCPUPath(t *testing.T) {
	p := New(Config{})

	// Try to create a file in a non-existent directory
	err := p.Start("/nonexistent/directory/cpu.prof")
	if err == nil {
		t.Error("Start() should fail with invalid path")
		p.Stop()
	}
}

func TestProfilerInvalidMemPath(t *testing.T) {
	config := Config{
		MemProfilePath: "/nonexistent/directory/mem.prof",
	}
	p := New(config)

	// Start should succeed (CPU path is empty)
	if err := p.Start(""); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	// Stop should return error because memory profile path is invalid
	if err := p.Stop(); err == nil {
		t.Error("Stop() should fail with invalid memory profile path")
	}
}

func TestConfigProfilingEnabled(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		want   bool
	}{
		{
			name:   "no profiling",
			config: Config{},
			want:   false,
		},
		{
			name: "CPU only",
			config: Config{
				CPUProfilePath: "/tmp/cpu.prof",
			},
			want: true,
		},
		{
			name: "memory only",
			config: Config{
				MemProfilePath: "/tmp/mem.prof",
			},
			want: true,
		},
		{
			name: "both enabled",
			config: Config{
				CPUProfilePath: "/tmp/cpu.prof",
				MemProfilePath: "/tmp/mem.prof",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.ProfilingEnabled(); got != tt.want {
				t.Errorf("ProfilingEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}
