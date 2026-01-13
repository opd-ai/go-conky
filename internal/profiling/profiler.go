// Package profiling provides CPU and memory profiling support for conky-go.
// It wraps Go's runtime/pprof package to provide convenient profiling
// functionality for performance analysis and optimization.
package profiling

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"sync"
)

// Profiler manages CPU and memory profiling for the application.
// It provides a thread-safe way to start and stop profiling sessions.
type Profiler struct {
	cpuFilePath string
	cpuFile     *os.File
	memFilePath string
	running     bool
	mu          sync.Mutex
}

// Config holds configuration for the profiler.
type Config struct {
	// CPUProfilePath is the file path for CPU profile output.
	// If empty, CPU profiling is disabled.
	CPUProfilePath string

	// MemProfilePath is the file path for memory profile output.
	// If empty, memory profiling is disabled.
	MemProfilePath string
}

// New creates a new Profiler with the given configuration.
// The profiler is not started automatically; call Start() to begin profiling.
func New(config Config) *Profiler {
	return &Profiler{
		cpuFilePath: config.CPUProfilePath,
		memFilePath: config.MemProfilePath,
	}
}

// Start begins CPU profiling if a CPU profile path was configured.
// It returns an error if profiling is already running or if the file cannot be created.
func (p *Profiler) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return errors.New("profiler is already running")
	}

	if p.cpuFilePath == "" {
		p.running = true
		return nil
	}

	f, err := os.Create(p.cpuFilePath)
	if err != nil {
		return fmt.Errorf("failed to create CPU profile file: %w", err)
	}

	if err := pprof.StartCPUProfile(f); err != nil {
		f.Close()
		return fmt.Errorf("failed to start CPU profile: %w", err)
	}

	p.cpuFile = f
	p.running = true
	return nil
}

// Stop stops CPU profiling and writes the memory profile if configured.
// It returns an error if profiling is not running or if writing profiles fails.
func (p *Profiler) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return errors.New("profiler is not running")
	}

	var errs []error

	// Stop CPU profiling
	if p.cpuFile != nil {
		pprof.StopCPUProfile()
		if err := p.cpuFile.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close CPU profile file: %w", err))
		}
		p.cpuFile = nil
	}

	// Write memory profile
	if p.memFilePath != "" {
		if err := p.writeMemProfile(); err != nil {
			errs = append(errs, err)
		}
	}

	p.running = false

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// writeMemProfileToPath writes a memory profile to the specified path.
// It forces garbage collection before taking the profile for accurate results.
func writeMemProfileToPath(path string) error {
	// Force garbage collection before taking memory profile
	runtime.GC()

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create memory profile file: %w", err)
	}
	defer f.Close()

	if err := pprof.WriteHeapProfile(f); err != nil {
		return fmt.Errorf("failed to write memory profile: %w", err)
	}

	return nil
}

// writeMemProfile writes the memory profile to the configured file.
func (p *Profiler) writeMemProfile() error {
	return writeMemProfileToPath(p.memFilePath)
}

// IsRunning returns true if the profiler is currently running.
func (p *Profiler) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.running
}

// SetMemProfilePath sets the memory profile output path.
// This can only be set when the profiler is not running.
func (p *Profiler) SetMemProfilePath(path string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return errors.New("cannot set memory profile path while profiler is running")
	}

	p.memFilePath = path
	return nil
}

// WriteMemProfileNow writes a memory profile immediately without stopping.
// This is useful for taking snapshots during program execution.
func (p *Profiler) WriteMemProfileNow(path string) error {
	return writeMemProfileToPath(path)
}

// ProfilingEnabled returns true if any profiling is configured.
func (c Config) ProfilingEnabled() bool {
	return c.CPUProfilePath != "" || c.MemProfilePath != ""
}
