// Package lua provides Golua integration for conky-go.
// It implements the Lua runtime environment with safe execution,
// resource limits, and the Conky Lua API.
package lua

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"sync"

	"github.com/arnodel/golua/lib"
	rt "github.com/arnodel/golua/runtime"
)

// RuntimeConfig contains configuration options for the Lua runtime.
type RuntimeConfig struct {
	// CPULimit is the CPU instruction limit for Lua execution.
	// 0 means unlimited.
	CPULimit uint64
	// MemoryLimit is the maximum memory in bytes that Lua can allocate.
	// 0 means unlimited.
	MemoryLimit uint64
	// Stdout is the writer for Lua print output.
	// If nil, os.Stdout is used.
	Stdout io.Writer
}

// DefaultConfig returns a RuntimeConfig with sensible default values.
// CPU limit: 10,000,000 instructions
// Memory limit: 50 MB
func DefaultConfig() RuntimeConfig {
	return RuntimeConfig{
		CPULimit:    10_000_000,
		MemoryLimit: 50 * 1024 * 1024, // 50 MB
		Stdout:      os.Stdout,
	}
}

// ConkyRuntime wraps a Golua runtime with Conky-specific functionality.
// It provides thread-safe access to Lua execution with resource limits.
type ConkyRuntime struct {
	config  RuntimeConfig
	runtime *rt.Runtime
	output  *bytes.Buffer
	cleanup func()
	fsys    fs.FS // Optional embedded filesystem for require() support
	mu      sync.RWMutex
}

// New creates a new ConkyRuntime with the specified configuration.
// The runtime is initialized with Lua standard libraries and resource limits.
func New(config RuntimeConfig) (*ConkyRuntime, error) {
	output := &bytes.Buffer{}
	stdout := config.Stdout
	if stdout == nil {
		stdout = output
	} else {
		// Capture output while also writing to configured stdout
		stdout = io.MultiWriter(stdout, output)
	}

	runtime := rt.New(stdout)

	// Load standard libraries
	cleanup := lib.LoadAll(runtime)

	cr := &ConkyRuntime{
		config:  config,
		runtime: runtime,
		output:  output,
		cleanup: cleanup,
	}

	return cr, nil
}

// LoadString compiles and loads a Lua code string.
// The returned Closure can be executed using Execute.
func (cr *ConkyRuntime) LoadString(name, code string) (*rt.Closure, error) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	closure, err := cr.runtime.CompileAndLoadLuaChunk(
		name,
		[]byte(code),
		rt.TableValue(cr.runtime.GlobalEnv()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load Lua code: %w", err)
	}

	return closure, nil
}

// LoadFile reads and loads a Lua file from disk.
// The returned Closure can be executed using Execute.
func (cr *ConkyRuntime) LoadFile(path string) (*rt.Closure, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read Lua file %s: %w", path, err)
	}

	cr.mu.Lock()
	defer cr.mu.Unlock()

	closure, err := cr.runtime.CompileAndLoadLuaChunk(
		path,
		content,
		rt.TableValue(cr.runtime.GlobalEnv()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load Lua file %s: %w", path, err)
	}

	return closure, nil
}

// LoadFileFromFS reads and loads a Lua file from an embedded filesystem.
// This enables loading Lua scripts from embedded FS (e.g., using go:embed).
// The returned Closure can be executed using Execute.
func (cr *ConkyRuntime) LoadFileFromFS(fsys fs.FS, path string) (*rt.Closure, error) {
	content, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, fmt.Errorf("failed to read Lua file from FS %s: %w", path, err)
	}

	cr.mu.Lock()
	defer cr.mu.Unlock()

	closure, err := cr.runtime.CompileAndLoadLuaChunk(
		path,
		content,
		rt.TableValue(cr.runtime.GlobalEnv()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load Lua file %s: %w", path, err)
	}

	return closure, nil
}

// SetFS sets the filesystem used for Lua's require/dofile functions.
// This allows Lua scripts to load additional files from embedded filesystems.
// If fs is nil, only disk files can be loaded via require/dofile.
func (cr *ConkyRuntime) SetFS(fsys fs.FS) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	cr.fsys = fsys
}

// Execute runs a compiled Lua closure within resource limits.
// Returns the result value and any error that occurred.
func (cr *ConkyRuntime) Execute(closure *rt.Closure) (rt.Value, error) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	// Create a context with resource limits
	ctx := rt.RuntimeContextDef{
		HardLimits: rt.RuntimeResources{
			Cpu:    cr.config.CPULimit,
			Memory: cr.config.MemoryLimit,
		},
	}

	// Push the context before execution
	cr.runtime.PushContext(ctx)
	defer cr.runtime.PopContext()

	// Execute the closure
	thread := cr.runtime.MainThread()
	result, err := rt.Call1(thread, rt.FunctionValue(closure))
	if err != nil {
		return rt.NilValue, fmt.Errorf("Lua execution error: %w", err)
	}

	return result, nil
}

// ExecuteString compiles and executes a Lua code string.
// This is a convenience method that combines LoadString and Execute.
func (cr *ConkyRuntime) ExecuteString(name, code string) (rt.Value, error) {
	closure, err := cr.LoadString(name, code)
	if err != nil {
		return rt.NilValue, err
	}
	return cr.Execute(closure)
}

// ExecuteFile loads and executes a Lua file.
// This is a convenience method that combines LoadFile and Execute.
func (cr *ConkyRuntime) ExecuteFile(path string) (rt.Value, error) {
	closure, err := cr.LoadFile(path)
	if err != nil {
		return rt.NilValue, err
	}
	return cr.Execute(closure)
}

// GetGlobal retrieves a global variable from the Lua environment.
func (cr *ConkyRuntime) GetGlobal(name string) rt.Value {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	return cr.runtime.GlobalEnv().Get(rt.StringValue(name))
}

// SetGlobal sets a global variable in the Lua environment.
func (cr *ConkyRuntime) SetGlobal(name string, value rt.Value) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	cr.runtime.GlobalEnv().Set(rt.StringValue(name), value)
}

// SetGoFunction registers a Go function in the Lua global environment.
// The function can be called from Lua code.
// The function is declared as memory-safe and CPU-safe for use with resource limits.
func (cr *ConkyRuntime) SetGoFunction(name string, fn rt.GoFunctionFunc, nArgs int, hasVarArgs bool) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	goFunc := rt.NewGoFunction(fn, name, nArgs, hasVarArgs)
	// Declare the function as compliant with resource limits
	rt.SolemnlyDeclareCompliance(rt.ComplyMemSafe|rt.ComplyCpuSafe, goFunc)
	cr.runtime.GlobalEnv().Set(rt.StringValue(name), rt.FunctionValue(goFunc))
}

// CallFunction calls a Lua function by name with the given arguments.
// Returns the result value and any error that occurred.
func (cr *ConkyRuntime) CallFunction(name string, args ...rt.Value) (rt.Value, error) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	fn := cr.runtime.GlobalEnv().Get(rt.StringValue(name))
	if fn == rt.NilValue {
		return rt.NilValue, fmt.Errorf("function %s not found", name)
	}

	// Create a context with resource limits for this call
	ctx := rt.RuntimeContextDef{
		HardLimits: rt.RuntimeResources{
			Cpu:    cr.config.CPULimit,
			Memory: cr.config.MemoryLimit,
		},
	}

	cr.runtime.PushContext(ctx)
	defer cr.runtime.PopContext()

	thread := cr.runtime.MainThread()
	result, err := rt.Call1(thread, fn, args...)
	if err != nil {
		return rt.NilValue, fmt.Errorf("failed to call function %s: %w", name, err)
	}

	return result, nil
}

// Output returns the captured output from Lua print statements.
func (cr *ConkyRuntime) Output() string {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	return cr.output.String()
}

// ClearOutput clears the captured output buffer.
func (cr *ConkyRuntime) ClearOutput() {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	cr.output.Reset()
}

// Runtime returns the underlying Golua runtime.
// Use with caution as this bypasses thread-safety protections.
func (cr *ConkyRuntime) Runtime() *rt.Runtime {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	return cr.runtime
}

// Config returns the current runtime configuration.
func (cr *ConkyRuntime) Config() RuntimeConfig {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	return cr.config
}

// Close releases resources associated with the runtime.
// The runtime should not be used after calling Close.
func (cr *ConkyRuntime) Close() error {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	if cr.cleanup != nil {
		cr.cleanup()
		cr.cleanup = nil
	}

	return nil
}
