# Conky-Go Architecture

This document describes the high-level architecture of Conky-Go, a Go reimplementation of the [Conky](https://github.com/brndnmtthws/conky) system monitor.

## Overview

Conky-Go is designed as a modular system with clear separation of concerns. The architecture consists of five core components that work together to provide system monitoring with customizable display output.

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│  Configuration  │────│  Lua Integration │────│ System Monitor  │
│     Parser      │    │    (golua)       │    │    Backend      │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                        │                       │
         │                        │                       │
         └────────────────────────┼───────────────────────┘
                                  │
                          ┌──────────────────┐
                          │  Rendering Engine│
                          │    (Ebiten)      │
                          └──────────────────┘
                                  │
                          ┌──────────────────┐
                          │ Window Manager   │
                          │  & Compositing   │
                          └──────────────────┘
```

## Core Components

### 1. Configuration Parser (`internal/config/`)

The configuration parser handles both legacy `.conkyrc` text format and modern Lua configuration format. It provides:

- **Legacy Parser** (`legacy.go`): Parses traditional `.conkyrc` files with key-value configuration followed by a `TEXT` section
- **Lua Parser** (`lua.go`): Parses modern Lua-based configurations using `conky.config` and `conky.text` tables
- **Validation** (`validation.go`): Comprehensive configuration validation with warnings for unknown variables
- **Migration** (`migration.go`): Tools to convert legacy configurations to modern Lua format
- **Type Definitions** (`types.go`): Core configuration data structures

**Key Types:**
```go
type Config struct {
    Window  WindowConfig   // Window positioning and properties
    Display DisplayConfig  // Display settings (update interval, fonts)
    Text    TextConfig     // Text template content
    Colors  ColorConfig    // Color definitions
}
```

### 2. System Monitoring Backend (`internal/monitor/`)

The system monitoring backend collects data from the Linux `/proc` filesystem and other system sources. It provides thread-safe access to system statistics with configurable update intervals.

**Supported Monitoring:**

| Component | Source | Description |
|-----------|--------|-------------|
| CPU | `/proc/stat`, `/proc/cpuinfo` | Usage percentage, per-core stats, frequency |
| Memory | `/proc/meminfo` | Total, used, free, cached, swap statistics |
| Uptime | `/proc/uptime` | System uptime and idle time |
| Network | `/proc/net/dev` | Interface statistics, bytes/packets per second |
| Filesystem | `/proc/mounts`, `statvfs()` | Mounted filesystems, usage, inodes |
| Disk I/O | `/proc/diskstats` | Read/write operations, bytes per second |
| Hardware | `/sys/class/hwmon/` | Temperature sensors, fan speeds |
| Processes | `/proc/[pid]/stat` | Process count, top CPU/memory consumers |
| Battery | `/sys/class/power_supply/` | Battery level, charging status |
| Audio | `/proc/asound/cards` | Audio cards, volume levels |

**Architecture:**
```go
type SystemMonitor struct {
    data     *SystemData     // Aggregated system data
    interval time.Duration   // Update interval
    ctx      context.Context // Cancellation context
    // Individual readers for each subsystem
    cpuReader, memReader, uptimeReader, ...
}
```

### 3. Rendering Engine (`internal/render/`)

The rendering engine uses [Ebiten](https://github.com/hajimehoshi/ebiten) v2 for cross-platform 2D graphics. It provides:

- **Game Loop** (`game.go`): Implements `ebiten.Game` interface for Update/Draw/Layout
- **Text Rendering** (`text.go`): Font loading and text drawing with color support
- **Widgets** (`widgets.go`): Progress bars, gauges, and other visual elements
- **Graphs** (`graph.go`): Line graphs, bar graphs, and histograms for data visualization
- **Cairo Compatibility** (`cairo.go`): Translation layer for Cairo drawing commands
- **Color Management** (`color.go`): Color parsing and manipulation
- **Performance** (`perf.go`): Frame rate and performance monitoring

**Key Interfaces:**
```go
type TextRendererInterface interface {
    DrawText(screen *ebiten.Image, text string, x, y float64, color color.RGBA)
    MeasureText(text string) (width, height float64)
    LineHeight() float64
    SetFontSize(size float64)
    FontSize() float64
}
```

### 4. Lua Integration (`internal/lua/`)

Lua integration uses [Golua](https://github.com/arnodel/golua) for pure Go Lua 5.4 execution. It provides:

- **Runtime** (`runtime.go`): Safe Lua execution environment with resource limits
- **Conky API** (`api.go`): Implementation of Conky Lua functions (`conky_parse`, etc.)
- **Cairo Bindings** (`cairo_bindings.go`): Lua bindings for Cairo drawing functions
- **Event Hooks** (`hooks.go`): Support for `conky_main`, `conky_start`, etc.

**Sandboxing:**
```go
type RuntimeConfig struct {
    CPULimit    uint64 // Instruction limit (default: 10M)
    MemoryLimit uint64 // Memory limit (default: 50MB)
    Stdout      io.Writer
}
```

### 5. Profiling (`internal/profiling/`)

Development and debugging tools:

- **CPU/Memory Profiling** (`profiler.go`): Integration with Go's pprof
- **Leak Detection** (`leak_detector.go`): Memory leak detection and monitoring

## Data Flow

```
System Data Sources (/proc, /sys)
           │
           ▼
┌─────────────────────┐
│   System Monitor    │ ← Updates at configured interval
│ (cpuReader, memReader, etc.) │
└─────────────────────┘
           │
           ▼
┌─────────────────────┐
│    SystemData       │ ← Thread-safe data aggregation
└─────────────────────┘
           │
           ▼
┌─────────────────────┐
│  Lua Runtime        │ ← Variable substitution via conky_parse()
│  (conky_parse, hooks) │
└─────────────────────┘
           │
           ▼
┌─────────────────────┐
│  Cairo Commands     │ ← Drawing instructions from Lua scripts
└─────────────────────┘
           │
           ▼
┌─────────────────────┐
│  Ebiten Renderer    │ ← Translates Cairo → Ebiten graphics
│  (60 FPS game loop) │
└─────────────────────┘
           │
           ▼
┌─────────────────────┐
│  Window Display     │ ← X11/Wayland output
└─────────────────────┘
```

## Thread Safety

All shared state is protected with appropriate mutex types:

- **`sync.RWMutex`**: Used for read-heavy data structures (SystemData, Game state)
- **`sync.Mutex`**: Used for write-heavy state (Cairo renderer context)

Pattern example:
```go
func (sd *SystemData) GetCPU() CPUStats {
    sd.mu.RLock()
    defer sd.mu.RUnlock()
    return sd.CPU  // Return copy for isolation
}
```

## Dependencies

| Library | Purpose | License |
|---------|---------|---------|
| [Ebiten v2](https://github.com/hajimehoshi/ebiten) | 2D rendering engine | Apache 2.0 |
| [Golua](https://github.com/arnodel/golua) | Lua 5.4 runtime | MIT |
| Go Standard Library | System calls, parsing | BSD-3-Clause |

## Directory Structure

```
conky-go/
├── cmd/conky-go/           # Main executable entry point
├── internal/
│   ├── config/             # Configuration parsing and validation
│   ├── lua/                # Golua integration and Conky API
│   ├── monitor/            # System monitoring backend
│   ├── profiling/          # CPU/memory profiling tools
│   └── render/             # Ebiten rendering engine
├── test/
│   ├── configs/            # Test configuration files
│   └── integration/        # Integration tests
├── docs/                   # Documentation
│   ├── architecture.md     # This file
│   ├── migration.md        # Migration guide from Conky
│   └── api.md              # API reference
└── scripts/                # Build and development scripts
```

## Performance Targets

| Metric | Target | Rationale |
|--------|--------|-----------|
| Startup time | < 100ms | Faster than original Conky (~200ms) |
| Update latency | < 16ms | 60 FPS capable rendering |
| Memory footprint | < 50MB | Comparable to Conky (20-40MB) |
| CPU usage | < 1% idle | Minimal background overhead |

## Extension Points

1. **Custom Monitors**: Add new readers in `internal/monitor/` following the `*Reader` pattern
2. **Custom Widgets**: Implement widget interfaces in `internal/render/widgets.go`
3. **Lua Functions**: Register Go functions via `ConkyRuntime.SetGoFunction()`
4. **Configuration Options**: Add fields to `config.Config` and update parsers
