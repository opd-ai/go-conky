# Conky-Go API Reference

This document provides a reference for the Conky-Go API, including Go packages, Lua functions, and configuration options.

## Go Packages

### Package `config`

Import: `github.com/opd-ai/go-conky/internal/config`

The config package provides configuration parsing and validation for Conky configurations.

#### Types

##### Config

```go
type Config struct {
    Window  WindowConfig   // Window settings
    Display DisplayConfig  // Display settings
    Text    TextConfig     // Text template
    Colors  ColorConfig    // Color definitions
}
```

##### WindowConfig

```go
type WindowConfig struct {
    OwnWindow    bool          // Create own window
    Type         WindowType    // Window type (normal, desktop, dock, etc.)
    Transparent  bool          // Enable transparency
    Hints        []WindowHint  // Window manager hints
    Width        int           // Window width
    Height       int           // Window height
    X, Y         int           // Window position offset
    Alignment    Alignment     // Screen alignment
}
```

##### DisplayConfig

```go
type DisplayConfig struct {
    Background     bool          // Run in background
    DoubleBuffer   bool          // Enable double buffering
    UpdateInterval time.Duration // Update interval
    Font           string        // Default font
    FontSize       float64       // Default font size
}
```

#### Functions

##### ParseFile

```go
func (p *Parser) ParseFile(path string) (*Config, error)
```

Parses a configuration file, automatically detecting the format (legacy or Lua).

##### ValidateConfig

```go
func ValidateConfig(cfg *Config) error
```

Validates a configuration and returns the first error found.

---

### Package `monitor`

Import: `github.com/opd-ai/go-conky/internal/monitor`

The monitor package provides system monitoring capabilities.

#### Types

##### SystemMonitor

```go
type SystemMonitor struct {
    // ... internal fields
}
```

Main monitoring controller that aggregates all system statistics.

##### CPUStats

```go
type CPUStats struct {
    UsagePercent float64   // Overall CPU usage (0-100)
    Cores        []float64 // Per-core usage
    CPUCount     int       // Number of cores
    ModelName    string    // CPU model
    Frequency    float64   // Current frequency (MHz)
}
```

##### MemoryStats

```go
type MemoryStats struct {
    Total        uint64  // Total memory (bytes)
    Used         uint64  // Used memory (bytes)
    Free         uint64  // Free memory (bytes)
    Available    uint64  // Available memory (bytes)
    Buffers      uint64  // Buffer memory (bytes)
    Cached       uint64  // Cache memory (bytes)
    SwapTotal    uint64  // Total swap (bytes)
    SwapUsed     uint64  // Used swap (bytes)
    SwapFree     uint64  // Free swap (bytes)
    UsagePercent float64 // Memory usage (0-100)
    SwapPercent  float64 // Swap usage (0-100)
}
```

#### Functions

##### NewSystemMonitor

```go
func NewSystemMonitor(interval time.Duration) *SystemMonitor
```

Creates a new system monitor with the specified update interval.

##### Start / Stop

```go
func (sm *SystemMonitor) Start() error
func (sm *SystemMonitor) Stop()
```

Controls the background monitoring loop.

##### Update

```go
func (sm *SystemMonitor) Update() error
```

Performs a single update of all system statistics.

##### Data Accessors

```go
func (sm *SystemMonitor) CPU() CPUStats
func (sm *SystemMonitor) Memory() MemoryStats
func (sm *SystemMonitor) Uptime() UptimeStats
func (sm *SystemMonitor) Network() NetworkStats
func (sm *SystemMonitor) Filesystem() FilesystemStats
func (sm *SystemMonitor) DiskIO() DiskIOStats
func (sm *SystemMonitor) Hwmon() HwmonStats
func (sm *SystemMonitor) Process() ProcessStats
func (sm *SystemMonitor) Battery() BatteryStats
func (sm *SystemMonitor) Audio() AudioStats
```

Thread-safe accessors for system data.

---

### Package `lua`

Import: `github.com/opd-ai/go-conky/internal/lua`

The lua package provides Lua scripting integration using Golua.

#### Types

##### ConkyRuntime

```go
type ConkyRuntime struct {
    // ... internal fields
}
```

Wraps a Golua runtime with Conky-specific functionality.

##### RuntimeConfig

```go
type RuntimeConfig struct {
    CPULimit    uint64    // CPU instruction limit (0 = unlimited)
    MemoryLimit uint64    // Memory limit in bytes (0 = unlimited)
    Stdout      io.Writer // Output writer for print()
}
```

#### Functions

##### New

```go
func New(config RuntimeConfig) (*ConkyRuntime, error)
```

Creates a new Conky Lua runtime with resource limits.

##### DefaultConfig

```go
func DefaultConfig() RuntimeConfig
```

Returns sensible defaults (10M CPU, 50MB memory).

##### ExecuteString

```go
func (cr *ConkyRuntime) ExecuteString(name, code string) (rt.Value, error)
```

Compiles and executes Lua code from a string.

##### ExecuteFile

```go
func (cr *ConkyRuntime) ExecuteFile(path string) (rt.Value, error)
```

Loads and executes a Lua file.

##### SetGoFunction

```go
func (cr *ConkyRuntime) SetGoFunction(name string, fn rt.GoFunctionFunc, nArgs int, hasVarArgs bool)
```

Registers a Go function callable from Lua.

##### CallFunction

```go
func (cr *ConkyRuntime) CallFunction(name string, args ...rt.Value) (rt.Value, error)
```

Calls a Lua function by name with arguments.

---

### Package `render`

Import: `github.com/opd-ai/go-conky/internal/render`

The render package provides Ebiten-based rendering.

#### Types

##### Game

```go
type Game struct {
    // ... internal fields
}
```

Implements `ebiten.Game` interface for rendering.

##### Config

```go
type Config struct {
    Width           int           // Window width
    Height          int           // Window height
    Title           string        // Window title
    UpdateInterval  time.Duration // Data update interval
    BackgroundColor color.RGBA    // Background color
}
```

##### TextLine

```go
type TextLine struct {
    Text  string     // Text content
    X, Y  float64    // Position
    Color color.RGBA // Text color
}
```

#### Functions

##### NewGame

```go
func NewGame(config Config) *Game
```

Creates a new Game with default text renderer.

##### Run

```go
func (g *Game) Run() error
```

Starts the Ebiten game loop (blocks until window closes).

##### SetLines / AddLine / ClearLines

```go
func (g *Game) SetLines(lines []TextLine)
func (g *Game) AddLine(line TextLine)
func (g *Game) ClearLines()
```

Manages text lines to render.

##### SetDataProvider

```go
func (g *Game) SetDataProvider(dp DataProvider)
```

Sets the data provider for updates (e.g., SystemMonitor).

---

## Lua API

### Global Functions

#### conky_parse

```lua
result = conky_parse(template)
```

Parses a template string containing Conky variables and returns the result with variables substituted.

**Example:**
```lua
cpu_usage = conky_parse("${cpu}%")  -- Returns "45%"
```

### Event Hooks

#### conky_main

```lua
function conky_main()
    -- Called every update cycle
    return "Text to display"
end
```

Main hook called on each update. Return value is appended to display.

#### conky_start

```lua
function conky_start()
    -- Called once at startup
end
```

Initialization hook called when Conky starts.

#### conky_shutdown

```lua
function conky_shutdown()
    -- Called before Conky exits
end
```

Cleanup hook called on shutdown.

### Cairo Drawing Functions

Cairo functions are available for custom graphics:

#### Color and Style

```lua
cairo_set_source_rgba(cr, r, g, b, a)  -- Set color (0-1 range)
cairo_set_line_width(cr, width)         -- Set line width
cairo_set_line_cap(cr, cap)             -- Set line cap style
cairo_set_line_join(cr, join)           -- Set line join style
```

#### Drawing Primitives

```lua
cairo_move_to(cr, x, y)                 -- Move to position
cairo_line_to(cr, x, y)                 -- Line to position
cairo_rectangle(cr, x, y, w, h)         -- Draw rectangle
cairo_arc(cr, xc, yc, r, a1, a2)        -- Draw arc
cairo_curve_to(cr, x1, y1, x2, y2, x3, y3)  -- Bezier curve
```

#### Path Operations

```lua
cairo_new_path(cr)      -- Start new path
cairo_close_path(cr)    -- Close current path
cairo_stroke(cr)        -- Stroke current path
cairo_fill(cr)          -- Fill current path
cairo_paint(cr)         -- Paint entire surface
```

#### Text

```lua
cairo_select_font_face(cr, family, slant, weight)
cairo_set_font_size(cr, size)
cairo_show_text(cr, text)
cairo_text_extents(cr, text)  -- Returns text metrics
```

#### Transformations

```lua
cairo_translate(cr, x, y)     -- Translate origin
cairo_rotate(cr, angle)       -- Rotate (radians)
cairo_scale(cr, sx, sy)       -- Scale
cairo_save(cr)                -- Save state
cairo_restore(cr)             -- Restore state
```

---

## Configuration Options

### Window Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `own_window` | bool | false | Create own window |
| `own_window_type` | string | "normal" | Window type: normal, desktop, dock, panel, override |
| `own_window_transparent` | bool | false | Enable transparency |
| `own_window_hints` | string | "" | Comma-separated hints: undecorated, below, above, sticky, skip_taskbar, skip_pager |
| `alignment` | string | "top_left" | Window alignment on screen |
| `gap_x` | int | 0 | Horizontal gap from edge |
| `gap_y` | int | 0 | Vertical gap from edge |
| `minimum_width` | int | 10 | Minimum window width |
| `minimum_height` | int | 10 | Minimum window height |
| `maximum_width` | int | 0 | Maximum width (0 = unlimited) |

### Display Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `background` | bool | false | Fork to background |
| `double_buffer` | bool | true | Enable double buffering |
| `update_interval` | float | 1.0 | Update interval in seconds |
| `font` | string | "DejaVu Sans Mono:size=10" | Default font |
| `default_color` | string | "white" | Default text color |
| `color0` - `color9` | string | - | Custom color definitions |

### Alignment Values

| Value | Aliases | Position |
|-------|---------|----------|
| `top_left` | tl | Top-left corner |
| `top_middle` | tm, top_center, tc | Top center |
| `top_right` | tr | Top-right corner |
| `middle_left` | ml | Left center |
| `middle_middle` | mm, center, c | Screen center |
| `middle_right` | mr | Right center |
| `bottom_left` | bl | Bottom-left corner |
| `bottom_middle` | bm, bottom_center, bc | Bottom center |
| `bottom_right` | br | Bottom-right corner |

---

## Error Handling

All functions return errors following Go conventions:

```go
monitor := monitor.NewSystemMonitor(time.Second)
if err := monitor.Start(); err != nil {
    log.Fatalf("failed to start monitor: %v", err)
}
defer monitor.Stop()
```

Lua errors are wrapped with context:

```go
result, err := runtime.ExecuteFile("config.lua")
if err != nil {
    // Error includes file name and line number
    log.Printf("Lua error: %v", err)
}
```

---

## Thread Safety

All public APIs are thread-safe:

- **SystemMonitor**: Uses `sync.RWMutex` for all data access
- **ConkyRuntime**: Uses `sync.RWMutex` for Lua state access
- **Game**: Uses `sync.RWMutex` for line management

Pattern for concurrent access:
```go
// Safe to call from multiple goroutines
go func() {
    cpu := monitor.CPU()
    fmt.Printf("CPU: %.1f%%\n", cpu.UsagePercent)
}()
```
