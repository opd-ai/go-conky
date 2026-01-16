# Conky-Go Functional Audit Report

**Audit Date:** 2026-01-16  
**Auditor:** Automated Code Audit (GitHub Copilot)  
**Repository:** opd-ai/go-conky  
**Commit:** HEAD (copilot/perform-functional-audit-go-codebase)

---

## AUDIT SUMMARY

This audit compares the documented functionality in README.md and supporting documentation against the actual implementation. The codebase demonstrates a well-structured Go project with comprehensive system monitoring capabilities and Cairo-compatible rendering.

| Category | Count |
|----------|-------|
| **CRITICAL BUG** | 0 |
| **FUNCTIONAL MISMATCH** | 4 |
| **MISSING FEATURE** | 5 |
| **EDGE CASE BUG** | 3 |
| **PERFORMANCE ISSUE** | 1 |

**Overall Assessment:** The codebase is well-implemented with proper concurrency safety, error handling, and modular architecture. Most discrepancies are partial implementations or documentation clarifications rather than critical bugs.

---

## DETAILED FINDINGS

---

### FUNCTIONAL MISMATCH: Clipping Not Enforced During Drawing Operations

**File:** internal/render/cairo.go:1679-1788  
**Severity:** Medium  
**Description:** The Cairo clipping functions (`Clip`, `ClipPreserve`, `ResetClip`) are implemented but only track the clip region without actually enforcing it during drawing operations. The code explicitly documents this as a limitation but the documentation (docs/api.md, docs/migration.md) does not mention this behavior.

**Expected Behavior:** Per Cairo API and documentation in docs/api.md, `cairo_clip()` should restrict subsequent drawing operations to within the clip region.

**Actual Behavior:** The clip region is recorded for API compatibility but drawing operations are NOT restricted to the clip area. Scripts using clipping will execute without errors but produce incorrect visual output.

**Impact:** Conky Lua scripts that rely on clipping for complex drawings will not render correctly. This affects scripts that use clip paths for masking or complex UI layouts.

**Reproduction:** Execute a Lua script that uses `cairo_clip()` followed by a fill operation that extends beyond the clip bounds.

**Code Reference:**
```go
// --- Clipping Functions ---
//
// IMPORTANT: Clipping is currently a partial implementation.
// The clip region is tracked (stored in clipPath/hasClip) but NOT enforced
// during drawing operations. This means calling Clip() will record the clip
// region for API compatibility, but subsequent drawing will NOT be restricted
// to the clip area.
func (cr *CairoRenderer) Clip() {
    // ... implementation stores clip but doesn't enforce ...
}
```

---

### FUNCTIONAL MISMATCH: CopyPath Returns Empty Slice

**File:** internal/render/cairo.go:2183-2189  
**Severity:** Low  
**Description:** The `CopyPath` function is documented to return a representation of the current path, but the implementation always returns an empty slice with a comment stating "full path iteration not implemented."

**Expected Behavior:** `cairo_copy_path()` in the Lua API should return a table representation of the current path's segments.

**Actual Behavior:** Returns an empty slice/table regardless of path content.

**Impact:** Lua scripts that need to introspect or duplicate path data will receive empty results. This is a compatibility gap with original Cairo behavior.

**Reproduction:** Call `cairo_copy_path()` after building a path with `cairo_move_to`, `cairo_line_to`, etc.

**Code Reference:**
```go
// CopyPath returns a simplified representation of the current path.
// Note: This returns a basic representation - full path iteration is not supported.
func (cr *CairoRenderer) CopyPath() []PathSegment {
    cr.mu.Lock()
    defer cr.mu.Unlock()
    // Return empty slice - full path iteration not implemented
    return []PathSegment{}
}
```

---

### FUNCTIONAL MISMATCH: SetFillRule and SetOperator Are No-Ops

**File:** internal/render/cairo.go:751-771  
**Severity:** Low  
**Description:** The `SetFillRule` and `SetOperator` functions accept parameters but are no-op implementations, always returning default values. The comments document this as intentional for Cairo compatibility, but users may expect these to affect rendering.

**Expected Behavior:** `cairo_set_fill_rule()` should affect how paths are filled (winding vs even-odd). `cairo_set_operator()` should control compositing operations.

**Actual Behavior:** Both functions silently ignore their input parameters. `GetFillRule()` always returns 0 (WINDING), `GetOperator()` always returns 2 (OVER).

**Impact:** Scripts that rely on even-odd fill rule or alternative compositing operators will produce incorrect visual output.

**Reproduction:** Set fill rule to EVEN_ODD and fill a self-intersecting path.

**Code Reference:**
```go
// SetFillRule sets the fill rule for fill operations.
// This is a no-op placeholder for Cairo compatibility.
func (cr *CairoRenderer) SetFillRule(_ int) {
    // Ebiten's vector package uses even-odd fill rule by default
}

// SetOperator sets the compositing operator.
// This is a no-op placeholder for Cairo compatibility.
func (cr *CairoRenderer) SetOperator(_ int) {
    // Ebiten uses its own blend modes
}
```

---

### FUNCTIONAL MISMATCH: Sysname Always Returns "Linux"

**File:** internal/monitor/sysinfo.go:51-57  
**Severity:** Low  
**Description:** The `ReadSystemInfo` function hardcodes `Sysname` to "Linux" regardless of the actual platform. While the platform package supports Windows, macOS, and Android, this monitor code assumes Linux.

**Expected Behavior:** The `${sysname}` variable should return the actual OS name based on the platform.

**Actual Behavior:** Always returns "Linux" even when running on other platforms.

**Impact:** On cross-platform builds (Windows, macOS, Android), the `${sysname}` variable returns incorrect data.

**Reproduction:** Run the application on Windows or macOS and check the `${sysname}` variable output.

**Code Reference:**
```go
func (r *sysInfoReader) ReadSystemInfo() (SystemInfo, error) {
    info := SystemInfo{
        Sysname: "Linux", // Always Linux for this implementation
        Machine: r.getMachine(),
    }
    // ...
}
```

---

### MISSING FEATURE: Configuration Hot-Reloading Not Implemented

**File:** pkg/conky/conky.go (entire file)  
**Severity:** Medium  
**Description:** README.md mentions "Configuration hot-reloading is planned" and the Event system includes `EventConfigReloaded`, but there is no implementation for reloading configuration at runtime. The `Restart()` method provides a workaround but requires full restart.

**Expected Behavior:** Ability to reload configuration changes without restarting the application.

**Actual Behavior:** Configuration can only be changed by stopping and restarting the Conky instance.

**Impact:** Users must restart the application to apply configuration changes, reducing usability.

**Reproduction:** Modify configuration file while application is running; changes will not take effect until restart.

**Code Reference:**
```go
// status.go defines EventConfigReloaded but it's never emitted in the codebase
const (
    // ...
    // EventConfigReloaded is emitted when configuration is reloaded.
    EventConfigReloaded
    // ...
)
```

---

### DOCUMENTATION ISSUE: --convert CLI Flag for Legacy Config Conversion

**File:** cmd/conky-go/main.go, internal/config/migration.go, docs/migration.md:99-104  
**Severity:** Low  
**Description:** The `--convert` flag for converting legacy configurations to Lua format is implemented and wired to `config.MigrateLegacyFile`, but the migration documentation still describes it as a "future feature."

**Expected Behavior:** Documentation should accurately describe `--convert` as an implemented flag and provide correct usage examples for converting legacy configs to Lua.

**Actual Behavior:** The binary exposes a working `--convert` flag, while the docs present it as a future feature, creating a mismatch between implementation and documentation.

**Impact:** Users may assume the feature is unavailable and avoid using a working conversion flag, leading to unnecessary manual conversion or external tooling.

**Reproduction:** Run `./conky-go --help` (or inspect `cmd/conky-go/main.go` and `internal/config/migration.go`) to confirm `--convert` is implemented, then compare with the snippet in `docs/migration.md` that labels it as a "future feature."

**Code Reference (docs/migration.md):**
```
# Convert a legacy config to Lua (future feature)
./conky-go --convert ~/.conkyrc > ~/.config/conky/conky.conf
```

---

### MISSING FEATURE: Nvidia GPU Variables Return Empty

**File:** internal/lua/api.go:471-476  
**Severity:** Low  
**Description:** The `${nvidia}` and `${nvidiagraph}` variables are handled in the API but always return empty strings. Comments indicate this "Requires nvidia-smi integration."

**Expected Behavior:** GPU monitoring variables for Nvidia cards should return actual GPU data.

**Actual Behavior:** Always returns empty string.

**Impact:** Users with Nvidia GPUs cannot monitor GPU stats through Conky variables.

**Reproduction:** Use `${nvidia}` variable in configuration.

**Code Reference:**
```go
// Nvidia GPU (stubs for compatibility)
case "nvidia":
    return "" // Requires nvidia-smi integration
case "nvidiagraph":
    return ""
```

---

### MISSING FEATURE: Mail/IMAP/POP3 Variables Always Return "0"

**File:** internal/lua/api.go:486-492  
**Severity:** Low  
**Description:** Mail-related variables (`${imap_unseen}`, `${pop3_unseen}`, `${new_mails}`, etc.) are documented as stubs and always return "0".

**Expected Behavior:** Mail monitoring should work when properly configured.

**Actual Behavior:** All mail variables return "0" regardless of configuration.

**Impact:** Mail monitoring functionality is not available.

**Reproduction:** Configure IMAP/POP3 settings and use mail variables.

**Code Reference:**
```go
// IMAP/POP3/mail stubs
case "imap_unseen", "imap_messages":
    return "0"
case "pop3_unseen", "pop3_used":
    return "0"
case "new_mails", "mails":
    return "0"
```

---

### MISSING FEATURE: Weather and Stock Variables Return Empty

**File:** internal/lua/api.go:494-500  
**Severity:** Low  
**Description:** Weather and stock quote variables are stubs that return empty strings.

**Expected Behavior:** Weather and stock information when configured.

**Actual Behavior:** Always returns empty string.

**Impact:** Weather and stock monitoring not available.

**Code Reference:**
```go
// Weather stubs
case "weather":
    return ""

// Stock ticker stub
case "stockquote":
    return ""
```

---

### EDGE CASE BUG: InFill Check Uses Zero-Initialized Bounds

**File:** internal/render/cairo.go:2080-2090  
**Severity:** Low  
**Description:** The `InFill` and `InStroke` functions check if path bounds are all zero to detect an empty path, but this fails for valid paths that happen to include the origin point (0,0).

**Expected Behavior:** `InFill` should correctly identify points inside paths that pass through or near the origin.

**Actual Behavior:** A path containing (0,0) will have `pathMinX=0, pathMinY=0` which triggers the empty-path check, causing `InFill` to incorrectly return `false`.

**Impact:** Hit testing fails for paths near the origin.

**Reproduction:** Create a rectangle path from (-10, -10) to (10, 10), then call `InFill(0, 0)`.

**Code Reference:**
```go
func (cr *CairoRenderer) InFill(x, y float64) bool {
    cr.mu.Lock()
    defer cr.mu.Unlock()

    // Check if point is within path bounds
    if cr.pathMinX == 0 && cr.pathMinY == 0 && cr.pathMaxX == 0 && cr.pathMaxY == 0 {
        return false  // BUG: This falsely triggers for paths at origin
    }
    // ...
}
```

---

### EDGE CASE BUG: expandPathBoundsUnlocked Has Same Zero-Check Issue

**File:** internal/render/cairo.go:2208-2228  
**Severity:** Low  
**Description:** The `expandPathBoundsUnlocked` function uses zero-checks for all bounds to detect initialization state, but this fails for paths that start at or cross the origin.

**Expected Behavior:** Path bounds should be correctly tracked regardless of coordinate values.

**Actual Behavior:** A path starting at (0, 0) will have its bounds incorrectly re-initialized when subsequent points are added.

**Impact:** Path bounding boxes may be incorrect for paths near the origin, affecting `PathExtents`, `ClipExtents`, and hit testing.

**Code Reference:**
```go
func (cr *CairoRenderer) expandPathBoundsUnlocked(x, y float32) {
    if cr.pathMinX == 0 && cr.pathMaxX == 0 {  // BUG: Fails for origin paths
        cr.pathMinX = x
        cr.pathMaxX = x
        cr.pathMinY = y
        cr.pathMaxY = y
    } else {
        // ... update bounds ...
    }
}
```

---

### EDGE CASE BUG: RelMoveTo/RelLineTo/RelCurveTo Silently Fail Without Current Point

**File:** internal/render/cairo.go:977-1035  
**Severity:** Low  
**Description:** The relative path functions (`RelMoveTo`, `RelLineTo`, `RelCurveTo`) silently return without doing anything if there's no current point, without reporting an error. This matches Cairo's behavior but may be surprising.

**Expected Behavior:** Either set an error state or default to (0,0) like `CurveTo` does.

**Actual Behavior:** Operations are silently skipped, which could lead to unexpected path shapes.

**Impact:** User scripts may have silently incomplete paths if they start with relative operations.

**Reproduction:** Call `cairo_rel_line_to(cr, 10, 10)` without first calling `cairo_move_to`.

**Code Reference:**
```go
func (cr *CairoRenderer) RelMoveTo(dx, dy float64) {
    cr.mu.Lock()
    defer cr.mu.Unlock()
    if !cr.hasPath {
        // Cairo requires a current point for relative moves
        return  // Silent failure
    }
    // ...
}
```

---

### PERFORMANCE ISSUE: Convenience Drawing Functions Not Atomic

**File:** internal/render/cairo.go:1161-1208  
**Severity:** Low  
**Description:** The convenience functions (`DrawLine`, `DrawRectangle`, `FillRectangle`, `DrawCircle`, `FillCircle`) are documented as NOT atomic, with each internal method call acquiring and releasing the mutex independently.

**Expected Behavior:** Thread-safe atomic operations for convenience functions.

**Actual Behavior:** Multiple lock/unlock cycles per operation could lead to visual artifacts in concurrent scenarios where another goroutine modifies state between operations.

**Impact:** In highly concurrent applications, these functions may produce inconsistent results.

**Reproduction:** Call convenience functions from multiple goroutines simultaneously while also calling state-modifying functions.

**Code Reference:**
```go
// DrawLine draws a line from (x1,y1) to (x2,y2) with the current color and line width.
// Note: This function is NOT atomic - each internal method call acquires and releases
// the mutex independently. For atomic operations, use explicit locking at the caller level.
func (cr *CairoRenderer) DrawLine(x1, y1, x2, y2 float64) {
    cr.NewPath()   // Lock/Unlock
    cr.MoveTo(x1, y1)  // Lock/Unlock
    cr.LineTo(x2, y2)  // Lock/Unlock
    cr.Stroke()        // Lock/Unlock
}
```

---

## QUALITY VERIFICATION

### Dependency Analysis
- **Level 0 Files:** internal/config/types.go, internal/monitor/types.go, internal/render/types.go, pkg/conky/status.go, pkg/conky/options.go
- **Level 1+ Files:** All other files import from Level 0 packages

### Audit Progression
- ✅ README.md reviewed for functional requirements
- ✅ Documentation (docs/) reviewed for API specifications  
- ✅ Level 0 files audited first (types, constants)
- ✅ Level 1+ files audited in dependency order
- ✅ All findings include specific file references and line numbers
- ✅ Severity ratings aligned with actual impact
- ✅ No code modifications made (analysis only)

### Concurrency Safety
- ✅ All public APIs use `sync.RWMutex` appropriately
- ✅ Lock ordering is consistent within packages
- ✅ Deep copies made for returned data structures
- ✅ No obvious deadlock patterns detected

### Error Handling
- ✅ Errors wrapped with context using `fmt.Errorf`
- ✅ Errors propagated appropriately
- ✅ Resource cleanup uses defer patterns

---

## RECOMMENDATIONS

1. **High Priority:** Document clipping limitations in docs/migration.md to set user expectations.

2. **Medium Priority:** Consider using a `pathBoundsInitialized` boolean flag instead of zero-checks for path bounds tracking.

3. **Low Priority:** Implement the `--convert` flag or remove reference from documentation.

4. **Low Priority:** Make `Sysname` platform-aware by using `runtime.GOOS`.

5. **Future Consideration:** Implement atomic versions of convenience drawing functions for thread-critical applications.

---

*End of Audit Report*
