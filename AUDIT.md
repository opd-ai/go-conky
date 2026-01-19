# Functional Audit Report: go-conky

**Audit Date:** 2025-01-19
**Auditor:** Automated Code Audit System
**Repository:** opd-ai/go-conky
**Commit:** Current HEAD

---

## AUDIT SUMMARY

This audit compares the documented functionality in README.md against the actual implementation in the codebase. The audit follows a dependency-based analysis order and systematically verifies feature implementation.

| Category | Count |
|----------|-------|
| **CRITICAL BUG** | 0 |
| **FUNCTIONAL MISMATCH** | 3 |
| **MISSING FEATURE** | 5 |
| **EDGE CASE BUG** | 2 |
| **PERFORMANCE ISSUE** | 1 |

**Overall Assessment:** The codebase is well-implemented with solid architecture. Most documented features are functional. The identified issues are primarily related to incomplete feature parity with original Conky rather than implementation bugs.

---

## DEPENDENCY ANALYSIS

The codebase follows a clean layered architecture:

**Level 0 (No internal imports):**
- `internal/config/types.go` - Configuration type definitions
- `internal/config/defaults.go` - Default configuration values
- `internal/platform/platform.go` - Platform interface definitions
- `internal/render/types.go` - Render type definitions
- `pkg/conky/options.go` - Options type definitions
- `pkg/conky/status.go` - Status type definitions

**Level 1:**
- `internal/config/legacy.go` - Legacy parser (uses types)
- `internal/config/validation.go` - Validation (uses types)
- `internal/monitor/*.go` - All monitor readers (uses platform)
- `internal/render/background.go` - Background rendering
- `internal/render/widget_marker.go` - Widget markers

**Level 2:**
- `internal/config/lua.go` - Lua parser (uses types, legacy)
- `internal/config/parser.go` - Unified parser
- `internal/config/migration.go` - Config migration
- `internal/lua/runtime.go` - Lua runtime

**Level 3:**
- `internal/lua/api.go` - Conky Lua API (uses monitor, render)
- `internal/lua/conditionals.go` - Conditional parsing
- `internal/lua/hooks.go` - Hook management
- `internal/render/game.go` - Ebiten game loop

**Level 4:**
- `pkg/conky/impl.go` - Main implementation
- `pkg/conky/factory.go` - Factory functions
- `pkg/conky/render.go` - Render integration

---

## DETAILED FINDINGS

### FUNCTIONAL MISMATCH: Window Hint "below" Not Supported by Ebiten
**File:** pkg/conky/render.go:166-170
**Severity:** Low
**Description:** The README and documentation mention support for Conky window hints including "below" (keep window below other windows). However, the implementation explicitly notes that this hint is not supported by Ebiten and is silently ignored.
**Expected Behavior:** According to Conky documentation, `own_window_hints below` should keep the window below all other windows.
**Actual Behavior:** The hint is parsed but not applied. A comment in the code acknowledges this: "WindowHintBelow, WindowHintSticky are not supported by Ebiten but are parsed and documented for completeness"
**Impact:** Users migrating from original Conky who rely on the "below" window hint will not get the expected behavior. The window will not stay below other windows.
**Reproduction:** Configure a `.conkyrc` with `own_window_hints below` and observe that the window does not stay below other windows.
**Code Reference:**
```go
// parseWindowHints in pkg/conky/render.go
case config.WindowHintAbove:
    floating = true
case config.WindowHintSkipTaskbar:
    skipTaskbar = true
case config.WindowHintSkipPager:
    skipPager = true
    // WindowHintBelow, WindowHintSticky are not supported by Ebiten
    // but are parsed and documented for completeness
```

---

### FUNCTIONAL MISMATCH: Skip Taskbar/Pager Hints Documented But Not Effective
**File:** internal/render/types.go:50-55
**Severity:** Low
**Description:** The configuration supports `skip_taskbar` and `skip_pager` window hints, and these are parsed and stored in the render Config. However, the comments explicitly state these are "not directly supported by Ebiten."
**Expected Behavior:** Windows with `skip_taskbar` should not appear in the taskbar.
**Actual Behavior:** The values are stored but have no effect on actual window behavior.
**Impact:** Users expecting these hints to work will not see the expected behavior.
**Reproduction:** Configure `own_window_hints skip_taskbar,skip_pager` and observe the window still appears in taskbar/pager.
**Code Reference:**
```go
// SkipTaskbar hides the window from the taskbar when true.
// Note: This is not directly supported by Ebiten but documented for completeness.
SkipTaskbar bool
// SkipPager hides the window from the pager when true.
// Note: This is not directly supported by Ebiten but documented for completeness.
SkipPager bool
```

---

### FUNCTIONAL MISMATCH: Gauge Widget Falls Back to Bar
**File:** internal/render/game.go:359-362
**Severity:** Low
**Description:** The widget marker system defines `WidgetTypeGauge` for circular gauge widgets, but the rendering code falls back to a progress bar when a gauge is requested.
**Expected Behavior:** `${gauge}` variables should render as circular gauges.
**Actual Behavior:** Gauges render as horizontal progress bars.
**Impact:** Visual appearance differs from expected. Users expecting circular gauges will see bars instead.
**Reproduction:** Use any gauge variable like `${cpugauge}` (if supported) and observe it renders as a bar.
**Code Reference:**
```go
case WidgetTypeGauge:
    // Gauge is not yet implemented, fall back to bar
    g.drawProgressBar(screen, x, widgetY, marker.Width, marker.Height, marker.Value, clr)
```

---

### MISSING FEATURE: Platform Abstraction Not Integrated with Monitor Package
**File:** internal/platform/platform.go, internal/monitor/monitor.go
**Severity:** Medium
**Description:** The codebase has a well-designed cross-platform abstraction layer in `internal/platform/` with implementations for Linux, Windows, Darwin, and Android. However, the `internal/monitor/` package has its own Linux-specific implementations that read directly from `/proc` filesystem, rather than using the platform abstraction.
**Expected Behavior:** The monitor package should use the platform abstraction for cross-platform compatibility.
**Actual Behavior:** The monitor package uses hardcoded Linux `/proc` filesystem access, making it Linux-only despite the existence of cross-platform abstractions.
**Impact:** The system monitor functionality will not work correctly on Windows, macOS, or Android, even though platform implementations exist.
**Reproduction:** Attempt to run on macOS or Windows and observe that system monitoring variables return zero or default values instead of actual system data. Specifically, check `${cpu}`, `${mem}`, `${fs_used /}` which will display "0" or empty values.
**Code Reference:**
```go
// monitor/monitor.go uses Linux-specific readers:
cpuReader:         newCPUReader(),       // reads from /proc/stat
memReader:         newMemoryReader(),     // reads from /proc/meminfo
// etc.

// While platform/linux_cpu.go has cross-platform interface:
type CPUProvider interface {
    Usage() ([]float64, error)
    TotalUsage() (float64, error)
    // etc.
}
```

---

### MISSING FEATURE: MPD Integration Not Implemented
**File:** internal/lua/conditionals.go:385-389
**Severity:** Low
**Description:** The conditional parser supports `if_mpd_playing` syntax for checking MPD (Music Player Daemon) playback status, but the implementation is a stub that always returns false.
**Expected Behavior:** `${if_mpd_playing}` should check if MPD is currently playing music.
**Actual Behavior:** Always returns false regardless of MPD state.
**Impact:** Conky configurations that display MPD status conditionally will not work correctly.
**Reproduction:** Use `${if_mpd_playing}content${endif}` in a config; the content will never be displayed.
**Code Reference:**
```go
// evalIfMPDPlaying checks if MPD is playing.
// This is a stub - MPD integration would need a separate implementation.
func (api *ConkyAPI) evalIfMPDPlaying() bool {
    // MPD integration not implemented
    return false
}
```

---

### MISSING FEATURE: APCUPSD/UPS Monitoring Not Implemented
**File:** internal/lua/api.go:576-581
**Severity:** Low
**Description:** The Lua API has stubs for APCUPSD (UPS monitoring) variables that return "N/A" instead of actual values.
**Expected Behavior:** Variables like `${apcupsd_status}`, `${apcupsd_charge}`, etc. should return UPS status when APCUPSD daemon is running.
**Actual Behavior:** All APCUPSD variables return "N/A".
**Impact:** Users with UPS systems using APCUPSD cannot monitor battery backup status.
**Reproduction:** Use any `${apcupsd_*}` variable; it will always display "N/A".
**Code Reference:**
```go
// Apcupsd (UPS) stubs - not implemented; requires APCUPSD daemon and NIS protocol.
// Users should use ${execi} with apcaccess command. See docs/migration.md.
case "apcupsd", "apcupsd_model", "apcupsd_status", "apcupsd_linev",
    "apcupsd_load", "apcupsd_charge", "apcupsd_timeleft", "apcupsd_temp",
    "apcupsd_battv", "apcupsd_cable", "apcupsd_driver", "apcupsd_upsmode",
    "apcupsd_name", "apcupsd_hostname":
    return "N/A"
```

---

### MISSING FEATURE: Stock Quote Not Implemented
**File:** internal/lua/api.go:603-606
**Severity:** Low
**Description:** The `${stockquote}` variable is defined but returns "N/A" as a stub.
**Expected Behavior:** Should retrieve stock price data.
**Actual Behavior:** Returns "N/A".
**Impact:** Users cannot display stock information.
**Reproduction:** Use `${stockquote AAPL}` in config; displays "N/A".
**Code Reference:**
```go
// Stock ticker stub - not implemented; requires external API keys.
// Users should use ${execi} with custom scripts. See docs/migration.md.
case "stockquote":
    return "N/A"
```

---

### MISSING FEATURE: Graphs Not Fully Functional
**File:** internal/render/game.go:386-406
**Severity:** Medium
**Description:** Graph widgets (`${cpugraph}`, `${memgraph}`, etc.) are rendered as simple filled rectangles showing a single value rather than historical line/area graphs showing data over time.
**Expected Behavior:** Graph variables should display historical data as a line or area chart showing values over time.
**Actual Behavior:** Graphs display a single filled rectangle representing the current value only, without historical context.
**Impact:** Users expecting time-series visualizations will only see instantaneous snapshots.
**Reproduction:** Use `${cpugraph}` and observe it shows a single bar rather than a graph of CPU usage over time.
**Code Reference:**
```go
// drawGraphWidget renders a simple filled area representing a graph.
func (g *Game) drawGraphWidget(screen *ebiten.Image, x, y, width, height, value float64, clr color.RGBA) {
    // Draw background
    bgColor := color.RGBA{R: clr.R / 3, G: clr.G / 3, B: clr.B / 3, A: clr.A}
    vector.DrawFilledRect(screen, float32(x), float32(y), float32(width), float32(height), bgColor, false)

    // Draw filled area from bottom
    fillHeight := height * value / 100
    // ... renders single value, no historical data tracking
}
```

---

### EDGE CASE BUG: Scroll State Memory Leak Over Time
**File:** internal/lua/api.go:1899-1936
**Severity:** Low
**Description:** The `resolveScroll` function maintains scroll state in a map (`api.scrollStates`) that is never cleaned up. If the template text changes or configurations are reloaded, old scroll state entries accumulate in memory.
**Expected Behavior:** Scroll states should be cleaned up when no longer needed.
**Actual Behavior:** Scroll states persist indefinitely in the `scrollStates` map.
**Impact:** Long-running instances with dynamic configurations may experience gradual memory growth. Impact is minimal for typical use cases.
**Reproduction:** Repeatedly reload configurations with different scroll text content; observe scrollStates map growth.
**Code Reference:**
```go
// resolveScroll maintains persistent state for each unique scroll instance
scrollKey := strings.Join(args, "|")

api.mu.Lock()
defer api.mu.Unlock()

// Get or create scroll state
state, exists := api.scrollStates[scrollKey]
if !exists {
    state = &scrollState{
        position:   0,
        lastUpdate: time.Now(),
    }
    api.scrollStates[scrollKey] = state  // Never removed
}
```

---

### EDGE CASE BUG: Exec Cache Never Cleaned
**File:** internal/lua/api.go:1283-1329
**Severity:** Low
**Description:** The `resolveExeci` function caches command outputs with expiration times, but expired entries are never removed from the cache. While expired entries are refreshed on access, entries for commands that are no longer used remain in memory indefinitely.
**Expected Behavior:** Expired or unused cache entries should be periodically cleaned up.
**Actual Behavior:** Cache entries persist indefinitely even after expiration.
**Impact:** Long-running instances with changing `${execi}` commands may accumulate stale cache entries.
**Reproduction:** Use multiple different `${execi}` commands over time; observe execCache map growth.
**Code Reference:**
```go
// Check cache with read lock
api.mu.RLock()
entry, exists := api.execCache[cmdStr]
api.mu.RUnlock()

now := time.Now()
if exists && now.Before(entry.expiresAt) {
    return entry.output
}

// Cache miss or expired - execute command
// ... updates cache but never removes old entries
```

---

### PERFORMANCE ISSUE: Gradient Background Recalculated Every Frame
**File:** internal/render/background.go:149-176
**Severity:** Medium
**Description:** The `GradientBackground.Draw()` method creates a new pixel buffer and recalculates the entire gradient for every frame, even though the gradient is static.
**Expected Behavior:** The gradient should be calculated once and cached as an image.
**Actual Behavior:** Full gradient calculation occurs every frame (60 times per second).
**Impact:** Unnecessary CPU usage when using gradient backgrounds. At 400×300 resolution, this is 120,000 pixels recalculated 60 times per second, totaling 7.2 million pixel calculations per second.
**Reproduction:** Configure a gradient background and monitor CPU usage; compare against solid background.
**Code Reference:**
```go
// Draw renders the gradient background to the screen.
func (gb *GradientBackground) Draw(screen *ebiten.Image) {
    bounds := screen.Bounds()
    w := bounds.Dx()
    h := bounds.Dy()

    // Create a pixel buffer for the gradient - EVERY FRAME
    pixels := make([]byte, w*h*4)

    for y := 0; y < h; y++ {
        for x := 0; x < w; x++ {
            t := gb.interpolationFactor(x, y, w, h)
            c := gb.lerpColor(t)
            // ... per-pixel calculation
        }
    }

    screen.WritePixels(pixels)
}
```

---

## QUALITY OBSERVATIONS

### Positive Findings

1. **Thread Safety:** Excellent use of `sync.RWMutex` for protecting shared state throughout the codebase. Atomic operations used appropriately for counters.

2. **Error Handling:** Consistent error wrapping with context using `fmt.Errorf("description: %w", err)` pattern.

3. **Interface Usage:** Clean interface definitions for testability (e.g., `TextRendererInterface`, `DataProvider`, `SystemDataProvider`).

4. **Configuration Parsing:** Robust parsing for both legacy and Lua configuration formats with proper format detection.

5. **Resource Management:** Proper cleanup patterns with deferred cleanup functions and context cancellation.

6. **Test Coverage:** Comprehensive table-driven tests in config and monitor packages.

### Documentation Quality

The README.md is well-structured and accurate regarding the project's current state (early development). It appropriately notes that the project is in development and some features are not yet implemented.

---

## RECOMMENDATIONS

1. **Integrate Platform Abstraction:** Refactor the monitor package to use the platform abstraction layer to achieve true cross-platform support.

2. **Cache Gradient Images:** Pre-render gradient backgrounds once and reuse the cached image.

3. **Add Cleanup for Scroll/Exec Cache:** Implement periodic cleanup of expired entries or use a time-based eviction cache.

4. **Document Unsupported Features:** Create a compatibility matrix showing which Conky features are supported, partially supported, or not implemented.

5. **Implement Historical Graphs:** Add a ring buffer for storing historical values to enable proper graph rendering.

---

## VERIFICATION CHECKLIST (Audit Methodology)

The following methodology was applied during this audit:

- Dependency analysis completed before code examination
- Audit progression followed dependency levels strictly
- All findings include specific file references and line numbers
- Each bug explanation includes reproduction steps
- Severity ratings align with actual impact on functionality
- No code modifications were made (analysis only)

---

*End of Audit Report*
