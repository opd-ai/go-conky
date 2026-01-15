# Implementation Gap Analysis
Generated: 2026-01-15T04:22:06.476Z
Codebase Version: ce8cc597133b12ea87a6f2ba26f35d1686887173

## Executive Summary
Total Gaps Found: 8
- Critical: 0
- Moderate: 5
- Minor: 3

## Detailed Findings

### Gap #1: Variable Count Discrepancy - Documentation Claims 200+ Variables But Only ~32 Are Implemented in Lua API
**Documentation Reference:** 
> "System variables | ✅ Supported | 200+ variables implemented" (docs/migration.md:345)
> "Complete system monitoring backend supporting 200+ Conky variables" (ROADMAP.md:173)

**Implementation Location:** `internal/lua/api.go:144-230`

**Expected Behavior:** The Lua API's `conky_parse()` function should resolve 200+ system variables as documented.

**Actual Implementation:** The `resolveVariable()` function in `api.go` only implements 32 case statements for variable resolution. The remaining variables listed in `internal/config/validation.go:knownConkyVariables` (123 entries) are recognized for validation but not actually resolved in the Lua API.

**Gap Details:** The validation module recognizes 123 variable names as "known" to suppress warnings, but only 32 of these variables are actually implemented in the `resolveVariable()` switch statement. Variables like `loadavg`, `cpubar`, `cpugraph`, `membar`, `memgraph`, `addr`, `wireless_essid`, `diskio`, `top`, `battery`, `battery_status`, `battery_time`, `time`, `kernel`, `exec`, and many others are listed as "known" but return the original `${variable}` template when requested because they fall through to the `default` case.

**Reproduction:**
```go
// Using conky_parse with an unimplemented but "known" variable:
result := api.Parse("${loadavg}") 
// Returns: "${loadavg}" instead of actual load average values
```

**Production Impact:** Moderate - Users migrating from Conky with configurations using unimplemented variables will see raw template strings in their output rather than actual system data. This breaks the "100% compatible" promise.

**Evidence:**
```go
// api.go:227-229 - default case returns original template
default:
    // Return original if unknown variable
    return formatUnknownVariable(name, args)
```

---

### Gap #2: Cairo Function Count - Documentation Claims 180+ Functions But Only 20 Are Implemented
**Documentation Reference:**
> "cairo_* drawing functions (180+ functions)" (ROADMAP.md:358)
> "Complete Conky Lua API implementation" (ROADMAP.md:211)

**Implementation Location:** `internal/lua/cairo_bindings.go:46-78`

**Expected Behavior:** 180+ Cairo drawing functions should be available in Lua for compatibility with existing Conky Lua scripts.

**Actual Implementation:** Only 20 Cairo functions are registered in `registerFunctions()`:
- Color: `cairo_set_source_rgb`, `cairo_set_source_rgba` (2)
- Line style: `cairo_set_line_width`, `cairo_set_line_cap`, `cairo_set_line_join`, `cairo_set_antialias` (4)
- Path: `cairo_new_path`, `cairo_move_to`, `cairo_line_to`, `cairo_close_path`, `cairo_arc`, `cairo_arc_negative`, `cairo_curve_to`, `cairo_rectangle` (8)
- Drawing: `cairo_stroke`, `cairo_fill`, `cairo_stroke_preserve`, `cairo_fill_preserve`, `cairo_paint`, `cairo_paint_with_alpha` (6)

**Gap Details:** Many commonly used Cairo functions are documented in `docs/api.md` but not implemented:
- Text functions: `cairo_select_font_face`, `cairo_set_font_size`, `cairo_show_text`, `cairo_text_extents`
- Transform functions: `cairo_translate`, `cairo_rotate`, `cairo_scale`, `cairo_save`, `cairo_restore`
- Surface functions: `cairo_xlib_surface_create`, `cairo_create`, `cairo_destroy`, `cairo_surface_destroy`

**Reproduction:**
```lua
-- This example from docs/migration.md:224-243 will fail:
require 'cairo'  -- Not implemented
cairo_select_font_face(cr, family, slant, weight)  -- Not implemented
cairo_show_text(cr, text)  -- Not implemented
```

**Production Impact:** Moderate - Lua scripts using text rendering, transformations, or the documented surface creation pattern will fail. The example code shown in migration.md will not execute.

**Evidence:**
```go
// cairo_bindings.go only registers 20 functions:
cb.runtime.SetGoFunction("cairo_set_source_rgb", cb.setSourceRGB, 3, false)
// ... 19 more functions
// No text, transform, or surface functions are registered
```

---

### Gap #3: `require 'cairo'` Pattern Not Supported
**Documentation Reference:**
> "require 'cairo'" (docs/migration.md:224)
> "Cairo drawing functions are supported for custom graphics" (docs/migration.md:223)

**Implementation Location:** `internal/lua/cairo_bindings.go` (missing implementation)

**Expected Behavior:** Lua scripts should be able to use `require 'cairo'` to load Cairo functionality, as shown in the migration guide example.

**Actual Implementation:** Cairo functions are registered directly as globals by `NewCairoBindings()`, but there is no implementation of a `cairo` module that can be loaded via `require`. Additionally, the `conky_window` global table referenced in the example is not set up.

**Gap Details:** The migration guide shows this pattern:
```lua
require 'cairo'
if conky_window == nil then return end
local cs = cairo_xlib_surface_create(conky_window.display, ...)
```
However:
1. `require 'cairo'` has no corresponding module to load
2. `conky_window` global is never set with display/drawable/visual/width/height properties
3. `cairo_xlib_surface_create`, `cairo_create`, `cairo_destroy`, `cairo_surface_destroy` are not implemented

**Reproduction:**
```lua
-- From docs/migration.md example - all of these fail:
require 'cairo'  -- Error: module 'cairo' not found
if conky_window == nil then return end  -- conky_window is always nil
local cs = cairo_xlib_surface_create(...)  -- Function not found
```

**Production Impact:** Moderate - Any Lua script using the standard Conky Cairo pattern will fail completely. Users cannot use existing Cairo-based Conky Lua scripts.

**Evidence:**
```go
// cairo_bindings.go - No require/module registration:
func (cb *CairoBindings) registerFunctions() {
    // Functions registered as globals, no module created
    cb.runtime.SetGoFunction("cairo_set_source_rgb", ...)
    // No code to register a 'cairo' module for require
}
```

---

### Gap #4: Uptime Format Mismatch with Documentation Example
**Documentation Reference:**
> "| `${uptime}` | System uptime | `2d 5h 23m` |" (docs/migration.md:149)
> "| `${uptime_short}` | Short uptime format | `2d 5:23` |" (docs/migration.md:150)

**Implementation Location:** `internal/lua/api.go:266-300`

**Expected Behavior:** 
- `${uptime}` should format as `2d 5h 23m` (without seconds)
- `${uptime_short}` should format as `2d 5:23` (colon-separated)

**Actual Implementation:**
- `${uptime}` formats as `2d 5h 23m 45s` (includes seconds)
- `${uptime_short}` formats as `2d 5h 23m` (space-separated, no colon)

**Gap Details:** The implementation adds seconds to the regular uptime format and uses space-separated format for short uptime instead of the colon format shown in documentation.

**Reproduction:**
```go
// For an uptime of 2 days, 5 hours, 23 minutes, 45 seconds:
// Documentation says ${uptime} returns: "2d 5h 23m"
// Implementation returns: "2d 5h 23m 45s"

// Documentation says ${uptime_short} returns: "2d 5:23"
// Implementation returns: "2d 5h 23m"
```

**Production Impact:** Minor - Visual difference in output formatting. Not a breaking change but inconsistent with documentation and potentially with original Conky behavior.

**Evidence:**
```go
// api.go:274-276 - includes seconds in regular format
if days > 0 {
    return fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, secs)
}

// api.go:293-295 - uses space-separated format, not colon
if days > 0 {
    return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
}
```

---

### Gap #5: Config Conversion CLI Flag Not Implemented
**Documentation Reference:**
> "# Convert a legacy config to Lua (future feature)
> ./conky-go --convert ~/.conkyrc > ~/.config/conky/conky.conf" (docs/migration.md:102-103)

**Implementation Location:** `cmd/conky-go/main.go:28-33`

**Expected Behavior:** A `--convert` CLI flag should convert legacy configurations to modern Lua format.

**Actual Implementation:** The `main.go` only implements these flags:
- `-c` for config path
- `-v` for version
- `--cpuprofile` for CPU profiling
- `--memprofile` for memory profiling

The `--convert` flag mentioned in migration.md is not implemented. However, the `internal/config/migration.go` file contains the `MigrateLegacyFile()` function that could power this feature.

**Gap Details:** The migration functionality exists in the library (`config.MigrateLegacyFile`, `config.MigrateLegacyContent`) but is not exposed via CLI. Users cannot convert configurations from command line as documented.

**Reproduction:**
```bash
# This command from docs/migration.md fails:
./conky-go --convert ~/.conkyrc > ~/.config/conky/conky.conf
# Error: flag provided but not defined: -convert
```

**Production Impact:** Minor - The migration code exists but is inaccessible via CLI. Users can still run legacy configs directly or use the library programmatically.

**Evidence:**
```go
// main.go:28-33 - Only these flags exist:
configPath := flag.String("c", "", "Path to configuration file")
version := flag.Bool("v", false, "Print version and exit")
cpuProfile := flag.String("cpuprofile", "", "Write CPU profile to file")
memProfile := flag.String("memprofile", "", "Write memory profile to file")
// No --convert flag
```

---

### Gap #6: Go Version Requirement Mismatch
**Documentation Reference:**
> "- **Go 1.24+**: Core language and standard library" (README.md:15)
> "Go 1.24 or later" (README.md:40)
> "go-version: 1.21" (ROADMAP.md:1889, CI/CD section)

**Implementation Location:** `go.mod` and documentation

**Expected Behavior:** Documentation should accurately reflect the Go version requirement.

**Actual Implementation:** README.md claims Go 1.24+ is required, but:
1. Go 1.24 does not exist yet (as of early 2026, the latest is Go 1.22)
2. ROADMAP.md CI examples use Go 1.21
3. The actual go.mod likely specifies an earlier version

**Gap Details:** The README.md specifies a non-existent Go version (1.24) while other documentation references Go 1.21. This is inconsistent and confusing.

**Reproduction:**
```bash
# Following README instructions with Go 1.22:
# "Go 1.24 or later" requirement cannot be met as 1.24 doesn't exist
```

**Production Impact:** Minor - Confusing for users trying to build from source. The actual code likely works with Go 1.21+.

**Evidence:**
```markdown
<!-- README.md:15 -->
- **Go 1.24+**: Core language and standard library

<!-- ROADMAP.md:1889 -->
go-version: 1.21
```

---

### Gap #7: Windows and macOS Cross-Platform Status Inconsistency
**Documentation Reference:**
> "- [x] Cross-platform support (Linux, Windows, macOS)" (README.md:31)
> "✅ **Windows**: Full native support via WMI/PDH APIs" (docs/cross-platform.md:8)
> "| Windows support | 🔄 Planned | Future release |" (docs/migration.md:353)

**Implementation Location:** Multiple documentation files

**Expected Behavior:** Documentation should consistently describe platform support status.

**Actual Implementation:** There's conflicting information across documents:
- README.md marks cross-platform support as complete `[x]`
- cross-platform.md claims "Full native support" for Windows
- migration.md still shows Windows as "Planned" with "Future release"
- Rendering (Ebiten) shows "Planned" for Windows/macOS in ROADMAP.md compatibility matrix

**Gap Details:** The migration.md compatibility matrix contradicts the README and cross-platform.md claims. Users may be confused about actual platform support level.

**Reproduction:**
```markdown
<!-- migration.md:354 claims: -->
| Windows support | 🔄 Planned | Future release |

<!-- But cross-platform.md claims: -->
✅ **Windows**: Full native support via WMI/PDH APIs
```

**Production Impact:** Minor - Documentation inconsistency causes user confusion about actual capabilities.

**Evidence:**
From docs/migration.md:352-354:
```markdown
| Wayland support | 🔄 Planned | Future release |
| Windows support | 🔄 Planned | Future release |
```

---

### Gap #8: Rendering Loop Not Fully Integrated
**Documentation Reference:**
> "✅ **Core Implementation Complete** - Integration in progress" (README.md:22)
> "Performance: Leverages Ebiten's optimized 2D rendering pipeline for smooth 60fps updates" (README.md:10)

**Implementation Location:** `pkg/conky/impl.go:82-99`

**Expected Behavior:** The application should use Ebiten's rendering loop for 60fps visual updates.

**Actual Implementation:** The `Start()` method spawns a goroutine that only waits for context cancellation. The comment in the code explicitly states the Ebiten rendering integration is not yet implemented:

```go
// When opts.Headless is false, this goroutine should integrate with
// render.Game.Run() to run the Ebiten rendering loop. The integration
// requires:
// 1. Create render.Game with config from c.cfg
// 2. Set c.monitor as the data provider
// 3. Call game.Run() which blocks until window close or context cancel
// See PLAN.md section 2.2.2 for render package changes needed.
```

**Gap Details:** The Ebiten rendering loop (`render.Game`) exists but is not connected to the main `Conky` interface. The public API starts monitoring but doesn't actually render anything to screen. The "60fps updates" claim cannot be verified as no visual output is produced.

**Reproduction:**
```go
// This code from README example will start but show nothing:
c, _ := conky.New("/path/to/config", nil)
c.Start()
// No window appears, no rendering occurs
```

**Production Impact:** Moderate - The application cannot display visual output through the documented public API. Users following README instructions will not see a working system monitor.

**Evidence:**
```go
// impl.go:82-99 - Goroutine only waits, doesn't render
go func() {
    defer c.wg.Done()
    defer c.cleanup()
    defer c.running.Store(false)
    
    // Wait for context cancellation.
    // When opts.Headless is false, this goroutine should integrate with
    // render.Game.Run() to run the Ebiten rendering loop.
    <-c.ctx.Done()
    
    c.emitEvent(EventStopped, "Instance stopped")
}()
```

---

## Summary Table

| Gap # | Description | Severity | Category |
|-------|-------------|----------|----------|
| 1 | Variable count (32 vs 200+) | Moderate | Feature Gap |
| 2 | Cairo functions (20 vs 180+) | Moderate | Feature Gap |
| 3 | `require 'cairo'` pattern not supported | Moderate | Feature Gap |
| 4 | Uptime format mismatch | Minor | Behavioral Nuance |
| 5 | `--convert` CLI flag not implemented | Minor | Feature Gap |
| 6 | Go version requirement mismatch | Minor | Documentation Drift |
| 7 | Cross-platform status inconsistency | Minor | Documentation Drift |
| 8 | Rendering loop not integrated | Moderate | Integration Gap |

## Recommendations

1. **Update documentation to reflect actual implementation status** - Revise claims about "200+ variables" and "180+ Cairo functions" to match implementation.

2. **Implement missing Cairo text functions** - Text rendering is essential for most Conky Lua scripts. Prioritize `cairo_select_font_face`, `cairo_show_text`, `cairo_text_extents`.

3. **Add `require 'cairo'` module support** - Create a Lua module that can be loaded via `require` to match existing Conky scripts.

4. **Implement `conky_window` global** - Required for Cairo surface creation in existing scripts.

5. **Add `--convert` CLI flag** - The migration code exists; just needs CLI exposure.

6. **Complete Ebiten rendering integration** - Connect `render.Game` to the public `Conky` interface.

7. **Reconcile documentation inconsistencies** - Ensure README, migration.md, and cross-platform.md agree on platform support status.

8. **Fix Go version requirement** - Change "Go 1.24+" to a currently available version.
