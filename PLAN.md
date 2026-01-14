# Embedding API Design Plan

This document provides a detailed technical plan for designing a public Go API that enables embedding the go-conky system monitor as a library component within third-party applications.

## Implementation Status

| Section | Status | Notes |
|---------|--------|-------|
| 1. API Interface Design | ✅ Complete | `pkg/conky/` package created with core interface |
| 2.1 Package Structure | ✅ Complete | Created `pkg/conky/` with conky.go, options.go, status.go, impl.go, doc.go |
| 2.2 Internal Package Modifications | ⏳ Pending | fs.FS support for config and lua packages not yet added |
| 2.3 Implementation | ✅ Complete | `conkyImpl` struct with lifecycle management |
| 3. Configuration Loading | ✅ Complete | Disk file, embedded FS, and io.Reader support |
| 4. Lifecycle Management | ✅ Complete | Start/Stop/Restart with thread safety |
| 5. Integration Examples | 📖 Reference | Documentation examples (not executable) |
| 6. Migration Path | ⏳ Pending | CLI not yet refactored to use public API |

## Table of Contents

1. [API Interface Design](#1-api-interface-design)
2. [Architecture Changes](#2-architecture-changes)
3. [Configuration Loading](#3-configuration-loading)
4. [Lifecycle Management](#4-lifecycle-management)
5. [Integration Examples](#5-integration-examples)
6. [Migration Path](#6-migration-path)

---

## 1. API Interface Design

### 1.1 Core Interface: `Conky`

The primary public interface that third-party applications will use to embed go-conky.

```go
// Package conky provides the public API for embedding the go-conky system monitor.
// It allows third-party applications to run go-conky as a library component
// with full lifecycle management and configuration flexibility.
package conky

import (
    "context"
    "io/fs"
    "time"
)

// Conky represents an embedded go-conky instance with full lifecycle control.
// It is safe for concurrent use from multiple goroutines.
type Conky interface {
    // Start begins the go-conky rendering loop.
    // It returns immediately after starting; the rendering runs in background goroutines.
    // Returns an error if already running or if initialization fails.
    Start() error

    // Stop gracefully shuts down the go-conky instance.
    // It waits for all goroutines to complete before returning.
    // Safe to call multiple times; subsequent calls are no-ops.
    Stop() error

    // Restart performs a stop followed by a start.
    // Configuration is reloaded from the original source.
    // Returns an error if restart fails; the instance will be in a stopped state.
    Restart() error

    // IsRunning returns true if the go-conky instance is currently running.
    IsRunning() bool

    // Status returns detailed status information about the instance.
    Status() Status

    // SetErrorHandler registers a callback for runtime errors.
    // The handler is invoked asynchronously; do not block in the handler.
    // Implementations of Conky MUST recover from panics in the handler so that
    // a buggy handler cannot crash the embedding application.
    SetErrorHandler(handler ErrorHandler)

    // SetEventHandler registers a callback for lifecycle events.
    SetEventHandler(handler EventHandler)
}

// Status represents the current state of a Conky instance.
type Status struct {
    // Running indicates if the instance is currently active.
    Running bool
    // StartTime is when the instance was last started (zero if never started).
    StartTime time.Time
    // UpdateCount is the number of update cycles completed since last start.
    UpdateCount uint64
    // LastError is the most recent error encountered (nil if none).
    LastError error
    // ConfigSource describes the configuration source (file path or "embedded").
    ConfigSource string
}

// ErrorHandler is a callback for runtime errors.
// It is called asynchronously when errors occur during operation.
type ErrorHandler func(err error)

// EventHandler is a callback for lifecycle events.
type EventHandler func(event Event)

// Event represents a lifecycle event.
type Event struct {
    Type      EventType
    Timestamp time.Time
    Message   string
}

// EventType enumerates lifecycle event types.
// Note: The underlying integer values are implementation details and should not
// be relied upon for serialization. Use the constant names for comparison.
type EventType int

const (
    // EventStarted is emitted when the instance starts successfully.
    EventStarted EventType = iota
    // EventStopped is emitted when the instance stops.
    EventStopped
    // EventRestarted is emitted after a successful restart.
    EventRestarted
    // EventConfigReloaded is emitted when configuration is reloaded.
    EventConfigReloaded
    // EventError is emitted when a recoverable error occurs.
    EventError
)
```

### 1.2 Configuration Options

```go
// Options configures the Conky instance behavior.
type Options struct {
    // UpdateInterval overrides the configuration file's update_interval.
    // Zero means use the configuration file's value.
    UpdateInterval time.Duration

    // WindowTitle overrides the window title.
    // Empty string means use the configuration file's value.
    WindowTitle string

    // Headless runs without creating a visible window.
    // Useful for testing or when only system data is needed.
    Headless bool

    // LuaCPULimit overrides the Lua CPU instruction limit.
    // Zero means use the default (10 million instructions).
    LuaCPULimit uint64

    // LuaMemoryLimit overrides the Lua memory limit in bytes.
    // Zero means use the default (50 MB).
    LuaMemoryLimit uint64

    // ShutdownTimeout sets the maximum time to wait for graceful shutdown.
    // Zero means use DefaultShutdownTimeout (5 seconds).
    ShutdownTimeout time.Duration

    // Logger sets a custom logger for debug/info messages.
    // If nil, no logging is performed.
    Logger Logger
}

// DefaultOptions returns Options with sensible defaults.
func DefaultOptions() Options {
    return Options{
        UpdateInterval: 0, // Use config file value
        Headless:       false,
        LuaCPULimit:    0, // Use default
        LuaMemoryLimit: 0, // Use default
    }
}

// Logger interface for custom logging.
type Logger interface {
    Debug(msg string, args ...any)
    Info(msg string, args ...any)
    Warn(msg string, args ...any)
    Error(msg string, args ...any)
}
```

### 1.3 Factory Functions

```go
// New creates a new Conky instance from a configuration file on disk.
// The configuration file can be in either legacy .conkyrc or modern Lua format.
// The instance is created but not started; call Start() to begin operation.
//
// Example:
//
//     conky, err := conky.New("/home/user/.conkyrc", nil)
//     if err != nil {
//         log.Fatal(err)
//     }
//     defer conky.Stop()
//     if err := conky.Start(); err != nil {
//         log.Fatal(err)
//     }
func New(configPath string, opts *Options) (Conky, error)

// NewFromFS creates a new Conky instance using configuration from an embedded filesystem.
// This enables bundling configuration files within the application binary using Go's embed package.
//
// The fsys parameter should contain the configuration files, and configPath is the path
// within the filesystem to the main configuration file.
//
// Example:
//
//     //go:embed configs/*
//     var configFS embed.FS
//
//     conky, err := conky.NewFromFS(configFS, "configs/myconky.lua", nil)
//     if err != nil {
//         log.Fatal(err)
//     }
func NewFromFS(fsys fs.FS, configPath string, opts *Options) (Conky, error)

// NewFromReader creates a new Conky instance from configuration content provided as an io.Reader.
// The format parameter specifies whether the content is "legacy" or "lua" format.
// This is useful for dynamically generated configurations or network-loaded configs.
//
// Example:
//
//     config := strings.NewReader(`
//         conky.config = { update_interval = 1 }
//         conky.text = [[CPU: ${cpu}%]]
//     `)
//     conky, err := conky.NewFromReader(config, "lua", nil)
func NewFromReader(r io.Reader, format string, opts *Options) (Conky, error)
```

### 1.4 Data Access Interface (Optional Advanced Usage)

For applications that want to access system monitoring data without the full rendering stack:

```go
// Monitor provides read-only access to system monitoring data.
// This interface is useful for applications that want system data
// without the full rendering overhead.
//
// The stat types (SystemData, CPUStats, MemoryStats, etc.) are defined in
// the internal/monitor package. When implementing the public API, these
// types will be re-exported or wrapped in the pkg/conky package.
type Monitor interface {
    // Data returns a snapshot of all current system data.
    Data() SystemData

    // CPU returns current CPU statistics.
    CPU() CPUStats
    // Memory returns current memory statistics.
    Memory() MemoryStats
    // Network returns current network statistics.
    Network() NetworkStats
    // Filesystem returns current filesystem statistics.
    Filesystem() FilesystemStats
    // Battery returns current battery statistics.
    Battery() BatteryStats
    // Uptime returns current uptime statistics.
    Uptime() UptimeStats
}

// GetMonitor returns the system monitor from a running Conky instance.
// Returns nil if the instance is not running.
//
// Note: GetMonitor is intentionally not part of the public Conky interface
// as it exposes internal monitoring details. Applications that need direct
// access to the underlying Monitor should use type assertion:
//
//   if impl, ok := c.(*conkyImpl); ok {
//       monitor := impl.GetMonitor()
//   }
//
// Alternatively, consider adding GetMonitor() to the Conky interface in a
// future version if this becomes a common use case.
func (c *conkyImpl) GetMonitor() Monitor
```

---

## 2. Architecture Changes

### 2.1 Package Structure

Create a new public package `pkg/conky/` that serves as the entry point for embedding:

```
go-conky/
├── cmd/
│   └── conky-go/                  # CLI application (uses pkg/conky)
│       └── main.go
├── pkg/
│   └── conky/                     # NEW: Public embedding API
│       ├── conky.go               # Main interface and factory functions
│       ├── options.go             # Options and configuration
│       ├── status.go              # Status and event types
│       ├── impl.go                # Private implementation
│       ├── impl_test.go           # Implementation tests
│       ├── example_test.go        # Runnable examples
│       └── doc.go                 # Package documentation
├── internal/                      # Existing internal packages (unchanged)
│   ├── config/
│   ├── lua/
│   ├── monitor/
│   ├── profiling/
│   ├── render/
│   └── window/                    # Window management (may need headless mode support)
├── test/
└── docs/
```

### 2.2 Internal Package Modifications

#### 2.2.1 `internal/config/` Changes

Add support for loading configuration from `fs.FS`:

```go
// parser.go additions

// ParseFromFS reads and parses a configuration file from an embedded filesystem.
func (p *Parser) ParseFromFS(fsys fs.FS, path string) (*Config, error) {
    content, err := fs.ReadFile(fsys, path)
    if err != nil {
        return nil, fmt.Errorf("failed to read config from FS %s: %w", path, err)
    }
    return p.Parse(content)
}

// ParseReader parses configuration from an io.Reader.
// The format parameter must be "legacy" or "lua".
func (p *Parser) ParseReader(r io.Reader, format string) (*Config, error) {
    content, err := io.ReadAll(r)
    if err != nil {
        return nil, fmt.Errorf("failed to read config: %w", err)
    }
    
    switch format {
    case "lua":
        return p.luaParser.Parse(content)
    case "legacy":
        return p.legacyParser.Parse(content)
    default:
        return nil, fmt.Errorf("unknown format: %s (expected 'lua' or 'legacy')", format)
    }
}
```

#### 2.2.2 `internal/render/` Changes

Add ability to stop the Ebiten game loop gracefully:

```go
// game.go additions

// RequestStop signals the game loop to stop after the current frame.
// This is a non-blocking call; use Wait() to block until stopped.
func (g *Game) RequestStop() {
    g.mu.Lock()
    defer g.mu.Unlock()
    g.stopRequested = true
}

// Wait blocks until the game loop has stopped.
func (g *Game) Wait() {
    g.wg.Wait()
}

// Update modification - check for stop request
func (g *Game) Update() error {
    g.mu.RLock()
    stopRequested := g.stopRequested
    g.mu.RUnlock()
    
    if stopRequested {
        return ebiten.Termination // Ebiten v2.5+ termination signal
        // Note: For older Ebiten versions, use a custom termination error
    }
    
    // ... existing update logic ...
    return nil
}
```

#### 2.2.3 `internal/lua/` Changes

Add support for loading Lua files from `fs.FS`:

```go
// runtime.go additions

// LoadFileFromFS reads and loads a Lua file from an embedded filesystem.
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
func (cr *ConkyRuntime) SetFS(fsys fs.FS) {
    cr.mu.Lock()
    defer cr.mu.Unlock()
    cr.fsys = fsys
    // Register custom searcher for require()
    cr.registerFSSearcher()
}
```

#### 2.2.4 `internal/monitor/` Changes

Add context-based cancellation for cleaner shutdown:

```go
// monitor.go modifications

// NewSystemMonitorWithContext creates a monitor with explicit context control.
func NewSystemMonitorWithContext(ctx context.Context, interval time.Duration) *SystemMonitor {
    innerCtx, cancel := context.WithCancel(ctx)
    
    return &SystemMonitor{
        data:     NewSystemData(),
        interval: interval,
        // ... readers initialization ...
        ctx:      innerCtx,
        cancel:   cancel,
    }
}
```

### 2.3 Implementation: `pkg/conky/impl.go`

```go
package conky

import (
    "context"
    "fmt"
    "io"
    "io/fs"
    "sync"
    "sync/atomic"
    "time"

    "github.com/opd-ai/go-conky/internal/config"
    "github.com/opd-ai/go-conky/internal/lua"
    "github.com/opd-ai/go-conky/internal/monitor"
    "github.com/opd-ai/go-conky/internal/render"
)

// DefaultShutdownTimeout is the default timeout for graceful shutdown.
// This can be overridden via Options.ShutdownTimeout.
const DefaultShutdownTimeout = 5 * time.Second

// conkyImpl is the private implementation of the Conky interface.
type conkyImpl struct {
    // Configuration
    cfg          *config.Config
    opts         Options
    configSource string  // Path or "embedded" or "reader"
    configLoader func() (*config.Config, error)
    fsys         fs.FS   // Embedded filesystem for Lua require() (nil for disk files)
    
    // Components
    monitor   *monitor.SystemMonitor
    runtime   *lua.ConkyRuntime
    game      *render.Game
    
    // State
    running     atomic.Bool
    startTime   time.Time
    updateCount atomic.Uint64
    lastError   atomic.Value // stores error
    
    // Handlers
    errorHandler ErrorHandler
    eventHandler EventHandler
    
    // Synchronization
    mu       sync.RWMutex
    ctx      context.Context
    cancel   context.CancelFunc
    wg       sync.WaitGroup
}

// Verify interface implementation
var _ Conky = (*conkyImpl)(nil)

func (c *conkyImpl) Start() error {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    if c.running.Load() {
        return fmt.Errorf("conky instance already running")
    }
    
    // Create cancellable context
    c.ctx, c.cancel = context.WithCancel(context.Background())
    
    // Initialize components
    if err := c.initComponents(); err != nil {
        if c.cancel != nil {
            c.cancel()
        }
        return fmt.Errorf("failed to initialize: %w", err)
    }
    
    // Start monitor
    if err := c.monitor.Start(); err != nil {
        c.cleanup()
        return fmt.Errorf("failed to start monitor: %w", err)
    }
    
    // Set running state BEFORE starting goroutine to avoid race
    c.running.Store(true)
    c.startTime = time.Now()
    
    // Start render loop in goroutine (non-blocking)
    c.wg.Add(1)
    go func() {
        defer c.wg.Done()
        defer c.cleanup()
        defer c.running.Store(false)
        
        if !c.opts.Headless {
            if err := c.game.Run(); err != nil {
                c.setError(err)
            }
        } else {
            // Headless mode: just wait for context cancellation
            <-c.ctx.Done()
        }
        
        c.emitEvent(EventStopped, "Instance stopped")
    }()
    
    c.emitEvent(EventStarted, "Instance started")
    
    return nil
}

func (c *conkyImpl) Stop() error {
    if !c.running.Load() {
        return nil // Already stopped
    }
    
    // Signal stop
    if c.cancel != nil {
        c.cancel()
    }
    
    // Request game to stop
    if c.game != nil {
        c.game.RequestStop()
    }
    
    // Wait for goroutines with timeout
    done := make(chan struct{})
    go func() {
        c.wg.Wait()
        close(done)
    }()
    
    // Use configured timeout or default
    timeout := c.opts.ShutdownTimeout
    if timeout <= 0 {
        timeout = DefaultShutdownTimeout
    }
    
    select {
    case <-done:
        return nil
    case <-time.After(timeout):
        return fmt.Errorf("shutdown timeout after %v: some goroutines did not stop", timeout)
    }
}

func (c *conkyImpl) Restart() error {
    // Stop if running
    if err := c.Stop(); err != nil {
        return fmt.Errorf("stop failed: %w", err)
    }
    
    // Reload configuration
    if c.configLoader != nil {
        cfg, err := c.configLoader()
        if err != nil {
            return fmt.Errorf("config reload failed: %w", err)
        }
        c.mu.Lock()
        c.cfg = cfg
        c.mu.Unlock()
        c.emitEvent(EventConfigReloaded, "Configuration reloaded")
    }
    
    // Start again
    if err := c.Start(); err != nil {
        return fmt.Errorf("start failed: %w", err)
    }
    
    c.emitEvent(EventRestarted, "Instance restarted")
    return nil
}

func (c *conkyImpl) IsRunning() bool {
    return c.running.Load()
}

func (c *conkyImpl) Status() Status {
    c.mu.RLock()
    startTime := c.startTime
    configSource := c.configSource
    c.mu.RUnlock()
    
    return Status{
        Running:      c.running.Load(),
        StartTime:    startTime,
        UpdateCount:  c.updateCount.Load(),
        LastError:    c.getError(),
        ConfigSource: configSource,
    }
}

func (c *conkyImpl) SetErrorHandler(handler ErrorHandler) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.errorHandler = handler
}

func (c *conkyImpl) SetEventHandler(handler EventHandler) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.eventHandler = handler
}

// Private helper methods

func (c *conkyImpl) initComponents() error {
    // Initialize system monitor
    interval := c.cfg.Display.UpdateInterval
    if c.opts.UpdateInterval > 0 {
        interval = c.opts.UpdateInterval
    }
    c.monitor = monitor.NewSystemMonitorWithContext(c.ctx, interval)
    
    // Initialize Lua runtime
    luaCfg := lua.DefaultConfig()
    if c.opts.LuaCPULimit > 0 {
        luaCfg.CPULimit = c.opts.LuaCPULimit
    }
    if c.opts.LuaMemoryLimit > 0 {
        luaCfg.MemoryLimit = c.opts.LuaMemoryLimit
    }
    var err error
    c.runtime, err = lua.New(luaCfg)
    if err != nil {
        return fmt.Errorf("lua runtime: %w", err)
    }
    
    // Initialize renderer
    renderCfg := render.Config{
        Width:          c.cfg.Window.Width,
        Height:         c.cfg.Window.Height,
        Title:          c.opts.WindowTitle,
        UpdateInterval: interval,
    }
    if renderCfg.Title == "" {
        renderCfg.Title = "conky-go"
    }
    c.game = render.NewGame(renderCfg)
    c.game.SetDataProvider(c.monitor)
    
    return nil
}

func (c *conkyImpl) cleanup() {
    if c.monitor != nil {
        c.monitor.Stop()
    }
    if c.runtime != nil {
        _ = c.runtime.Close()
    }
}

func (c *conkyImpl) setError(err error) {
    c.lastError.Store(err)
    
    c.mu.RLock()
    handler := c.errorHandler
    logger := c.opts.Logger
    c.mu.RUnlock()
    
    if handler != nil {
        go func() {
            defer func() {
                // Recover from panics in error handler to prevent crashing
                if r := recover(); r != nil {
                    if logger != nil {
                        logger.Error("error handler panicked", "panic", r, "original_error", err)
                    }
                }
            }()
            handler(err)
        }()
    }
    c.emitEvent(EventError, err.Error())
}

func (c *conkyImpl) getError() error {
    if v := c.lastError.Load(); v != nil {
        return v.(error)
    }
    return nil
}

func (c *conkyImpl) emitEvent(eventType EventType, message string) {
    c.mu.RLock()
    handler := c.eventHandler
    c.mu.RUnlock()
    
    if handler != nil {
        go handler(Event{
            Type:      eventType,
            Timestamp: time.Now(),
            Message:   message,
        })
    }
}
```

---

## 3. Configuration Loading

### 3.1 Strategy Overview

The API supports three configuration sources with a unified internal representation:

| Source | Factory Function | Use Case |
|--------|-----------------|----------|
| Disk file | `New()` | Traditional standalone usage |
| Embedded FS | `NewFromFS()` | Bundled applications |
| io.Reader | `NewFromReader()` | Dynamic/network configs |

### 3.2 Disk File Loading

```go
// New creates a Conky instance from a disk file.
func New(configPath string, opts *Options) (Conky, error) {
    if opts == nil {
        opts = &Options{}
    }
    
    parser, err := config.NewParser()
    if err != nil {
        return nil, fmt.Errorf("parser init: %w", err)
    }
    defer parser.Close()
    
    cfg, err := parser.ParseFile(configPath)
    if err != nil {
        return nil, fmt.Errorf("parse config: %w", err)
    }
    
    return &conkyImpl{
        cfg:          cfg,
        opts:         *opts,
        configSource: configPath,
        configLoader: func() (*config.Config, error) {
            p, err := config.NewParser()
            if err != nil {
                return nil, err
            }
            defer p.Close()
            return p.ParseFile(configPath)
        },
    }, nil
}
```

### 3.3 Embedded Filesystem Loading

```go
// NewFromFS creates a Conky instance from an embedded filesystem.
func NewFromFS(fsys fs.FS, configPath string, opts *Options) (Conky, error) {
    if opts == nil {
        opts = &Options{}
    }
    
    parser, err := config.NewParser()
    if err != nil {
        return nil, fmt.Errorf("parser init: %w", err)
    }
    defer parser.Close()
    
    cfg, err := parser.ParseFromFS(fsys, configPath)
    if err != nil {
        return nil, fmt.Errorf("parse config from FS: %w", err)
    }
    
    // Store fsys for Lua require() support
    return &conkyImpl{
        cfg:          cfg,
        opts:         *opts,
        configSource: "embedded:" + configPath,
        fsys:         fsys, // Store for Lua file access
        configLoader: func() (*config.Config, error) {
            p, err := config.NewParser()
            if err != nil {
                return nil, err
            }
            defer p.Close()
            return p.ParseFromFS(fsys, configPath)
        },
    }, nil
}
```

### 3.4 Reader Loading

```go
// NewFromReader creates a Conky instance from an io.Reader.
// Note: Assumes "bytes", "io", and "fmt" are imported.
func NewFromReader(r io.Reader, format string, opts *Options) (Conky, error) {
    if opts == nil {
        opts = &Options{}
    }
    
    // Read content once (can't re-read a Reader)
    content, err := io.ReadAll(r)
    if err != nil {
        return nil, fmt.Errorf("read config: %w", err)
    }
    
    parser, err := config.NewParser()
    if err != nil {
        return nil, fmt.Errorf("parser init: %w", err)
    }
    defer parser.Close()
    
    cfg, err := parser.ParseReader(bytes.NewReader(content), format)
    if err != nil {
        return nil, fmt.Errorf("parse config: %w", err)
    }
    
    return &conkyImpl{
        cfg:          cfg,
        opts:         *opts,
        configSource: "reader",
        configLoader: func() (*config.Config, error) {
            p, err := config.NewParser()
            if err != nil {
                return nil, err
            }
            defer p.Close()
            return p.ParseReader(bytes.NewReader(content), format)
        },
    }, nil
}
```

### 3.5 Lua Script File Resolution

When Lua configurations use `require()` or `dofile()`, the embedded filesystem must be used:

```go
// In lua/runtime.go
// Note: Requires importing "strings" and "io/fs" packages.

func (cr *ConkyRuntime) registerFSSearcher() {
    if cr.fsys == nil {
        return
    }
    
    // Register a custom package searcher that uses the embedded FS
    searcher := rt.NewGoFunction(func(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
        name := c.Arg(0).AsString()
        path := strings.ReplaceAll(name, ".", "/") + ".lua"
        
        content, err := fs.ReadFile(cr.fsys, path)
        if err != nil {
            return c.PushingNext1(t.Runtime, rt.NilValue), nil
        }
        
        closure, err := t.Runtime.CompileAndLoadLuaChunk(
            path, content,
            rt.TableValue(t.Runtime.GlobalEnv()),
        )
        if err != nil {
            return nil, err
        }
        
        return c.PushingNext1(t.Runtime, rt.FunctionValue(closure)), nil
    }, "fs_searcher", 1, false)
    
    // Add to package.searchers
    // ... implementation details ...
}
```

---

## 4. Lifecycle Management

### 4.1 State Machine

```
                    ┌─────────┐
                    │ Created │
                    └────┬────┘
                         │ Start()
                         ▼
┌──────────┐       ┌─────────┐
│ Stopped  │◄──────│ Running │
└──────────┘       └────┬────┘
     │                  │
     │ Start()          │ Stop() or
     │                  │ error/window close
     ▼                  ▼
┌─────────┐       ┌──────────┐
│ Running │◄──────│ Stopping │
└─────────┘       └──────────┘
      Restart() = Stop() + Start()
```

### 4.2 Thread Safety

All public methods are thread-safe:

```go
// Thread-safe state tracking using atomic operations
type conkyImpl struct {
    running     atomic.Bool     // Lock-free read in IsRunning()
    updateCount atomic.Uint64   // Lock-free increment in update loop
    lastError   atomic.Value    // Lock-free error storage
    
    mu sync.RWMutex             // Protects configuration changes
}
```

### 4.3 Graceful Shutdown

The stop sequence ensures clean resource release:

```go
func (c *conkyImpl) Stop() error {
    if !c.running.Load() {
        return nil
    }
    
    // 1. Signal context cancellation
    c.cancel()
    
    // 2. Request Ebiten to stop (if not headless)
    if c.game != nil {
        c.game.RequestStop()
    }
    
    // 3. Wait for all goroutines with timeout
    done := make(chan struct{})
    go func() {
        c.wg.Wait()
        close(done)
    }()
    
    select {
    case <-done:
        return nil
    case <-time.After(5 * time.Second):
        return fmt.Errorf("shutdown timeout: some goroutines did not stop")
    }
}
```

### 4.4 Independent Lifecycle

The embedding application is decoupled from go-conky's lifecycle:

```go
// Example: Application continues after go-conky stops
func main() {
    c, err := conky.New("~/.conkyrc", nil)
    if err != nil {
        log.Fatalf("failed to create conky instance: %v", err)
    }
    
    // Handle go-conky errors without crashing the app
    c.SetErrorHandler(func(err error) {
        log.Printf("conky error (non-fatal): %v", err)
    })
    
    // Handle events
    c.SetEventHandler(func(e conky.Event) {
        if e.Type == conky.EventStopped {
            log.Println("conky stopped, app continues running...")
        }
    })
    
    if err := c.Start(); err != nil {
        log.Fatalf("failed to start conky: %v", err)
    }
    
    // Application's own event loop continues independently
    for {
        // ... application logic ...
        time.Sleep(time.Second)
    }
}
```

### 4.5 Resource Cleanup

Resources are released in reverse order of initialization:

```go
func (c *conkyImpl) cleanup() {
    // 1. Stop system monitoring
    if c.monitor != nil {
        c.monitor.Stop()
        c.monitor = nil
    }
    
    // 2. Close Lua runtime (releases memory)
    if c.runtime != nil {
        _ = c.runtime.Close()
        c.runtime = nil
    }
    
    // 3. Game cleanup is automatic after Run() returns
    c.game = nil
}
```

---

## 5. Integration Examples

### 5.1 Basic Embedding

```go
package main

import (
    "log"
    "os"
    "os/signal"
    "syscall"
    
    "github.com/opd-ai/go-conky/pkg/conky"
)

func main() {
    // Create instance from disk config
    c, err := conky.New("/home/user/.conkyrc", nil)
    if err != nil {
        log.Fatalf("Failed to create conky: %v", err)
    }
    
    // Start (non-blocking)
    if err := c.Start(); err != nil {
        log.Fatalf("Failed to start: %v", err)
    }
    
    // Wait for user interrupt
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    <-sigCh
    
    // Clean shutdown
    if err := c.Stop(); err != nil {
        log.Printf("Warning: stop error: %v", err)
    }
}
```

### 5.2 Embedded Configuration

```go
package main

import (
    "embed"
    "log"
    
    "github.com/opd-ai/go-conky/pkg/conky"
)

//go:embed configs/*
var configFS embed.FS

func main() {
    // Load from embedded filesystem
    c, err := conky.NewFromFS(configFS, "configs/system-monitor.lua", &conky.Options{
        WindowTitle: "My App - System Monitor",
    })
    if err != nil {
        log.Fatalf("Failed to create conky: %v", err)
    }
    
    // Set up event handling
    c.SetEventHandler(func(e conky.Event) {
        log.Printf("[CONKY] %s: %s", e.Type, e.Message)
    })
    
    if err := c.Start(); err != nil {
        log.Fatalf("Failed to start: %v", err)
    }
    
    // Application continues with other work...
    select {}
}
```

### 5.3 Dynamic Configuration

```go
package main

import (
    "strings"
    "time"
    
    "github.com/opd-ai/go-conky/pkg/conky"
)

func main() {
    // Generate configuration dynamically
    luaConfig := `
conky.config = {
    update_interval = 2,
    own_window = true,
    own_window_type = 'desktop',
}

conky.text = [[
${color grey}Dynamic Config
CPU: ${cpu}% | RAM: ${memperc}%
]]
`
    
    c, err := conky.NewFromReader(
        strings.NewReader(luaConfig),
        "lua",
        nil,
    )
    if err != nil {
        panic(err)
    }
    
    if err := c.Start(); err != nil {
        panic(err)
    }
    
    // Later, update configuration
    time.Sleep(10 * time.Second)
    if err := c.Restart(); err != nil {
        panic(err)
    }
    
    select {}
}
```

### 5.4 Headless Mode (System Data Only)

```go
package main

import (
    "fmt"
    "time"
    
    "github.com/opd-ai/go-conky/pkg/conky"
)

func main() {
    // Run headless for data collection only
    c, err := conky.New("/home/user/.conkyrc", &conky.Options{
        Headless: true,
    })
    if err != nil {
        panic(err)
    }
    
    c.Start()
    
    // Access system monitoring data
    ticker := time.NewTicker(time.Second)
    for range ticker.C {
        if mon := c.GetMonitor(); mon != nil {
            cpu := mon.CPU()
            mem := mon.Memory()
            fmt.Printf("CPU: %.1f%% | RAM: %.1f%%\n", 
                cpu.UsagePercent, mem.UsagePercent)
        }
    }
}
```

### 5.5 Multiple Instances

```go
package main

import (
    "os"
    "os/signal"
    "syscall"
    
    "github.com/opd-ai/go-conky/pkg/conky"
)

func main() {
    // Run multiple conky instances
    configs := []string{
        "/home/user/.conky/cpu-monitor.lua",
        "/home/user/.conky/network-monitor.lua",
        "/home/user/.conky/disk-monitor.lua",
    }
    
    instances := make([]conky.Conky, len(configs))
    
    for i, cfg := range configs {
        c, err := conky.New(cfg, nil)
        if err != nil {
            panic(err)
        }
        instances[i] = c
        
        if err := c.Start(); err != nil {
            panic(err)
        }
    }
    
    // Wait for signal...
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT)
    <-sigCh
    
    // Stop all instances
    for _, c := range instances {
        c.Stop()
    }
}
```

### 5.6 Integration with GUI Framework

```go
package main

import (
    "fmt"
    "log"
    "time"
    
    "github.com/opd-ai/go-conky/pkg/conky"
    "fyne.io/fyne/v2/app"
    "fyne.io/fyne/v2/widget"
)

func main() {
    a := app.New()
    w := a.NewWindow("System Monitor")
    
    // Create conky instance (error handling included for production code)
    c, err := conky.New("~/.conkyrc", &conky.Options{
        Headless: true, // We'll display data in Fyne
    })
    if err != nil {
        log.Fatalf("failed to create conky instance: %v", err)
    }
    
    // Status label
    statusLabel := widget.NewLabel("Status: Stopped")
    
    // Control buttons with error handling
    startBtn := widget.NewButton("Start", func() {
        if err := c.Start(); err != nil {
            log.Printf("failed to start: %v", err)
            return
        }
        statusLabel.SetText("Status: Running")
    })
    
    stopBtn := widget.NewButton("Stop", func() {
        if err := c.Stop(); err != nil {
            log.Printf("failed to stop: %v", err)
        }
        statusLabel.SetText("Status: Stopped")
    })
    
    // CPU display (updated from conky monitor) with proper lifecycle
    done := make(chan struct{})
    cpuLabel := widget.NewLabel("CPU: --")
    go func() {
        ticker := time.NewTicker(time.Second)
        defer ticker.Stop()
        for {
            select {
            case <-done:
                return
            case <-ticker.C:
                if c.IsRunning() {
                    if mon := c.GetMonitor(); mon != nil {
                        cpu := mon.CPU()
                        cpuLabel.SetText(fmt.Sprintf("CPU: %.1f%%", cpu.UsagePercent))
                    }
                }
            }
        }
    }()
    
    // Clean up goroutine when window closes
    w.SetOnClosed(func() {
        close(done)
        if err := c.Stop(); err != nil {
            log.Printf("warning: failed to stop conky: %v", err)
        }
    })
    
    w.SetContent(widget.NewVBox(
        statusLabel,
        widget.NewHBox(startBtn, stopBtn),
        cpuLabel,
    ))
    
    w.ShowAndRun()
}
```

---

## 6. Migration Path

### 6.1 Current `cmd/conky-go/main.go` Refactoring

Transform the CLI to use the new public API:

**Before (current implementation):**
```go
package main

import (
    "flag"
    "fmt"
    "os"
)

func main() {
    os.Exit(run())
}

func run() int {
    configPath := flag.String("c", "", "Path to configuration file")
    // ... parsing and direct component initialization
    return 0
}
```

**After (using public API):**
```go
package main

import (
    "flag"
    "fmt"
    "os"
    "os/signal"
    "syscall"

    "github.com/opd-ai/go-conky/pkg/conky"
)

var Version = "0.1.0-dev"

func main() {
    os.Exit(run())
}

func run() int {
    // Parse flags
    configPath := flag.String("c", "", "Path to configuration file")
    version := flag.Bool("v", false, "Print version and exit")
    flag.Parse()

    if *version {
        fmt.Printf("conky-go version %s\n", Version)
        return 0
    }

    if *configPath == "" {
        fmt.Fprintln(os.Stderr, "No configuration file specified. Use -c <config>")
        return 1
    }

    // Create and start using public API
    c, err := conky.New(*configPath, nil)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        return 1
    }

    // Set up error handling
    c.SetErrorHandler(func(err error) {
        fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
    })

    if err := c.Start(); err != nil {
        fmt.Fprintf(os.Stderr, "Failed to start: %v\n", err)
        return 1
    }

    // Wait for termination signal
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
    
    for sig := range sigCh {
        switch sig {
        case syscall.SIGHUP:
            fmt.Println("Reloading configuration...")
            if err := c.Restart(); err != nil {
                fmt.Fprintf(os.Stderr, "Restart failed: %v\n", err)
            }
        default:
            fmt.Println("Shutting down...")
            if err := c.Stop(); err != nil {
                fmt.Fprintf(os.Stderr, "Stop error: %v\n", err)
            }
            return 0
        }
    }

    return 0
}
```

### 6.2 Step-by-Step Migration

1. **Create `pkg/conky/` package**
   - Define interfaces and types
   - Implement factory functions
   - Add comprehensive tests

2. **Add `fs.FS` support to internal packages**
   - `internal/config`: Add `ParseFromFS()`, `ParseReader()`
   - `internal/lua`: Add `LoadFileFromFS()`, `SetFS()`
   - Maintain backward compatibility with existing APIs

3. **Add graceful shutdown to render package**
   - Implement `RequestStop()` and `Wait()` on Game
   - Handle `ebiten.Termination` error properly

4. **Update `cmd/conky-go/main.go`**
   - Replace direct component usage with public API
   - Add signal handling for SIGHUP (restart)
   - Simplify error handling

5. **Add integration tests**
   - Test all factory functions
   - Test lifecycle transitions
   - Test concurrent access

6. **Update documentation**
   - Add `docs/embedding.md` user guide
   - Update `docs/api.md` with new public API
   - Add examples to `README.md`

### 6.3 Backward Compatibility

The migration maintains full backward compatibility:

- **Configuration files**: No changes required
- **CLI usage**: Same command-line interface
- **Internal packages**: New methods added, none removed
- **Behavior**: Identical rendering and monitoring

### 6.4 Testing Strategy

```go
// pkg/conky/conky_test.go

import (
    "embed"
    "sync"
    "testing"
    
    "github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
    // Test with valid config
    c, err := New("testdata/valid.conkyrc", nil)
    require.NoError(t, err)
    require.NotNil(t, c)
    require.False(t, c.IsRunning())
}

func TestNewFromFS(t *testing.T) {
    //go:embed testdata/*
    var testFS embed.FS
    
    c, err := NewFromFS(testFS, "testdata/valid.lua", nil)
    require.NoError(t, err)
    require.NotNil(t, c)
}

func TestLifecycle(t *testing.T) {
    c, _ := New("testdata/valid.conkyrc", &Options{Headless: true})
    
    // Start
    err := c.Start()
    require.NoError(t, err)
    require.True(t, c.IsRunning())
    
    // Stop
    err = c.Stop()
    require.NoError(t, err)
    require.False(t, c.IsRunning())
    
    // Restart
    err = c.Start()
    require.NoError(t, err)
    err = c.Restart()
    require.NoError(t, err)
    require.True(t, c.IsRunning())
    
    c.Stop()
}

func TestConcurrentAccess(t *testing.T) {
    c, _ := New("testdata/valid.conkyrc", &Options{Headless: true})
    c.Start()
    
    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            _ = c.IsRunning()
            _ = c.Status()
        }()
    }
    wg.Wait()
    
    c.Stop()
}
```

---

## Summary

This plan outlines a comprehensive design for a public embedding API that:

1. **Provides clean interfaces** - `Conky` interface with Start/Stop/Restart lifecycle
2. **Supports multiple configuration sources** - disk files, embedded FS, and io.Reader
3. **Ensures thread safety** - atomic operations and proper mutex usage
4. **Maintains independence** - embedding apps continue running if go-conky stops
5. **Preserves compatibility** - zero breaking changes to existing configs
6. **Enables gradual migration** - CLI refactored to use public API

The implementation requires minimal changes to internal packages (adding `fs.FS` support and graceful shutdown) while providing a complete public API in the new `pkg/conky/` package.
