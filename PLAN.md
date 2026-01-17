# Ebitengine to Fyne Migration Plan

## 1. Current State Analysis

### 1.1 Ebitengine Components Currently in Use

| Component | Location | Purpose | Lines |
|-----------|----------|---------|-------|
| `ebiten.Game` interface | `internal/render/game.go` | Core game loop (Update/Draw/Layout) | 520 |
| `ebiten.Image` | Throughout `internal/render/` | Drawing surface and texture management | ~3000 |
| `ebiten.DrawImageOptions` | `game.go`, `cairo.go`, `image.go` | Image transformation (translate, scale, rotate) | ~150 |
| `ebiten.DrawTrianglesOptions` | `cairo.go`, `widgets.go` | Antialiasing, fill rules, blending | ~100 |
| `vector.Path` | `cairo.go` | Path building (MoveTo, LineTo, CurveTo, Arc) | ~800 |
| `vector.DrawFilledRect` | `game.go`, `graph.go`, `widgets.go` | Filled rectangle rendering | ~50 |
| `vector.StrokeRect` | `game.go`, `graph.go`, `widgets.go` | Rectangle stroke/outline | ~30 |
| `vector.StrokeLine` | `game.go`, `graph.go` | Line drawing (solid, dashed) | ~20 |
| `etext.Draw` | `text.go` | Text rendering with fonts | ~30 |
| `etext.Measure` | `text.go` | Text measurement | ~15 |
| `etext.GoTextFace` | `text.go`, `font.go` | Font face management | ~50 |
| `ebiten.NewImage` | Multiple files | Image/surface creation | ~40 |
| `ebiten.NewImageFromImage` | `image.go` | Load images from Go image.Image | ~10 |
| Window management | `game.go` | `SetWindowSize`, `SetWindowTitle`, `RunGame` | ~20 |
| `ColorScale` | `text.go`, `image.go` | Color/alpha modulation | ~15 |
| `Blend` modes | `cairo.go` | Alpha compositing (SourceIn, SourceOver, etc.) | ~50 |

### 1.2 ARGB Transparency Blockers

**Critical Finding: Migration Direction Is Incorrect**

After thorough analysis, the premise of this migration needs reconsideration:

| Feature | Ebitengine (Current) | Fyne (Proposed) |
|---------|---------------------|-----------------|
| True ARGB Window Transparency | ✅ **Supported** via `ebiten.SetScreenTransparent(true)` | ❌ **NOT Supported** - Only widget-level alpha |
| Window-Level Alpha Blending | ✅ Compositing with X11/XWayland | ❌ Windows always opaque at OS level |
| Per-Pixel Transparency | ✅ Full support with compositor | ❌ Not available |
| Status | Available since v1.11 | Feature request open since 2019 (Issue #181) |

**Implications:**
- Migrating to Fyne would **LOSE** ARGB transparency support, not gain it
- Fyne's GitHub issues #181, #1297, #3004 confirm no current support for transparent windows
- Ebitengine already provides the transparency capabilities needed for conky-style overlays

### 1.3 Files Requiring Migration

```
internal/render/
├── cairo.go        (3296 lines) - Cairo compatibility layer, heaviest Ebiten usage
├── game.go         (520 lines)  - Core game loop and window management
├── text.go         (192 lines)  - Text rendering
├── font.go         (266 lines)  - Font management
├── graph.go        (698 lines)  - Graph widgets (LineGraph, BarGraph, Histogram)
├── widgets.go      (512 lines)  - Progress bars and gauges
├── image.go        (444 lines)  - Image loading and caching
├── widget_marker.go(188 lines)  - Widget marker parsing
├── types.go        (97 lines)   - Core type definitions
├── color.go        (122 lines)  - Color utilities
└── perf.go         (52 lines)   - Performance monitoring

pkg/conky/
└── render.go       (61 lines)   - Public rendering interface

internal/lua/
├── cairo_bindings.go  (950+ lines) - Lua-to-Cairo API bindings
└── cairo_module.go    (200+ lines) - Cairo module registration
```

**Total Lines Affected:** ~6,500+ lines

---

## 2. Migration Strategy

### 2.1 Recommended Approach: Do Not Migrate

Given the critical finding that Fyne lacks ARGB transparency support (a core requirement for conky-style desktop widgets), the recommended approach is:

**Option A: Stay with Ebitengine (Recommended)**
- Ebitengine already supports ARGB transparency
- Current implementation is feature-complete for Cairo compatibility
- Use `ebiten.SetScreenTransparent(true)` to enable transparency

**Option B: Conditional Migration (If Fyne adds transparency)**
- Wait for Fyne to implement window-level transparency (Issue #181)
- Implement abstraction layer now to enable future migration
- Estimated Fyne support: Unknown (feature requested since 2019)

### 2.2 If Migration Is Still Required (Alternative Approach)

If migration is mandated despite the transparency limitation, use an incremental approach with a compatibility layer:

```
Phase 1: Abstraction Layer    (2 weeks)
Phase 2: Core Rendering       (3 weeks)
Phase 3: Cairo Compatibility  (4 weeks)
Phase 4: Widget Migration     (2 weeks)
Phase 5: Testing & Validation (2 weeks)
```

### 2.3 Compatibility Layer Architecture

```go
// internal/render/renderer.go (new abstraction)
type Renderer interface {
    // Surface Management
    CreateSurface(width, height int) Surface
    SetScreen(surface Surface)
    
    // Path Operations
    NewPath()
    MoveTo(x, y float64)
    LineTo(x, y float64)
    CurveTo(x1, y1, x2, y2, x3, y3 float64)
    Arc(xc, yc, radius, angle1, angle2 float64)
    ClosePath()
    
    // Drawing Operations
    Fill()
    Stroke()
    FillPreserve()
    StrokePreserve()
    
    // Color and Style
    SetSourceRGBA(r, g, b, a float64)
    SetLineWidth(width float64)
    
    // Text
    DrawText(text string, x, y float64, color RGBA)
    MeasureText(text string) (width, height float64)
    
    // Transformation
    Save()
    Restore()
    Translate(tx, ty float64)
    Rotate(angle float64)
    Scale(sx, sy float64)
}
```

---

## 3. Component Mapping

### 3.1 Core Graphics Mapping

| Ebitengine Component | Fyne Equivalent | Complexity | Notes |
|---------------------|-----------------|------------|-------|
| `ebiten.Image` | `canvas.Raster` / `image.Image` | HIGH | Fyne uses CanvasObjects, not direct image buffers |
| `ebiten.DrawImageOptions` | `canvas.Image` transforms | MEDIUM | Limited transformation support |
| `ebiten.Game` | `fyne.App` + `fyne.Window` | MEDIUM | Different lifecycle model |
| `RunGame()` | `app.Run()` | LOW | One-time initialization |
| `SetWindowSize` | `window.Resize()` | LOW | Direct mapping |
| `SetWindowTitle` | `window.SetTitle()` | LOW | Direct mapping |
| `SetScreenTransparent` | **NOT AVAILABLE** | BLOCKER | No Fyne equivalent |

### 3.2 Vector Graphics Mapping

| Ebitengine Component | Fyne Equivalent | Complexity | Notes |
|---------------------|-----------------|------------|-------|
| `vector.Path` | `canvas.Raster` with custom draw | HIGH | No path API in Fyne |
| `vector.DrawFilledRect` | `canvas.Rectangle` | LOW | Direct mapping |
| `vector.StrokeRect` | `canvas.Rectangle` + border | MEDIUM | Need custom border handling |
| `vector.StrokeLine` | `canvas.Line` | LOW | Direct mapping |
| `path.Arc()` | `canvas.Arc` / custom Raster | MEDIUM | Limited arc support |
| `path.CubicTo()` | Custom Raster implementation | HIGH | No Bezier curves in Fyne |
| Fill rules (EvenOdd/NonZero) | Custom implementation | HIGH | Not supported in Fyne |

### 3.3 Text Rendering Mapping

| Ebitengine Component | Fyne Equivalent | Complexity | Notes |
|---------------------|-----------------|------------|-------|
| `etext.Draw` | `canvas.Text` | LOW | Different API but similar capability |
| `etext.Measure` | `fyne.MeasureText` | LOW | Direct mapping |
| `etext.GoTextFace` | `fyne.TextStyle` + `theme.TextFont()` | MEDIUM | Different font model |
| Custom font loading | `fyne.StaticResource` + custom theme | HIGH | Requires Theme interface implementation |

### 3.4 Compositing and Blending

| Ebitengine Component | Fyne Equivalent | Complexity | Notes |
|---------------------|-----------------|------------|-------|
| `ebiten.BlendSourceOver` | Default behavior | LOW | Standard alpha blending |
| `ebiten.BlendSourceIn` | **NOT AVAILABLE** | HIGH | Must implement in software |
| `ebiten.BlendClear` | Clear canvas | MEDIUM | Different approach needed |
| Custom blend modes | `canvas.Raster` pixel ops | HIGH | Software implementation required |

---

## 4. Implementation Steps

### Phase 1: Abstraction Layer (Week 1-2)
1. Define `Renderer` interface in `internal/render/renderer.go`
2. Wrap existing Ebitengine code to implement interface
3. Refactor `CairoRenderer` and `Game` to use abstraction

### Phase 2: Core Fyne Rendering (Week 3-5)
4. Create `internal/render/fyne/` with `FyneRenderer` structure
5. Implement surface management using `canvas.Raster`
6. Implement basic rectangle/line drawing
7. Create software path rasterizer (MoveTo, LineTo, Fill, Stroke)

### Phase 3: Cairo Compatibility (Week 6-9)
8. Implement arc and Bezier curve tessellation
9. Implement matrix transformations (translate, rotate, scale)
10. Implement gradient patterns (linear, radial)
11. Implement clipping (rectangular and path-based)

### Phase 4: Widget Migration (Week 10-11)
12. Migrate graphs (LineGraph, BarGraph, Histogram)
13. Migrate progress widgets (ProgressBar, Gauge)
14. Migrate image handling (ImageWidget, ImageCache)

### Phase 5: Testing & Validation (Week 12-13)
15. Unit testing with output comparison
16. Integration testing with real configurations
17. Document transparency limitations

---

## 5. ARGB Transparency Implementation

### 5.1 Current Ebitengine Solution (Working)

```go
// In main() or initialization:
func main() {
    ebiten.SetScreenTransparent(true)
    ebiten.SetWindowDecorated(false) // Optional: frameless window
    
    game := render.NewGame(config)
    ebiten.RunGame(game)
}

// In Game.Draw():
func (g *Game) Draw(screen *ebiten.Image) {
    // Do NOT call screen.Fill() for full transparency
    // Or use transparent background:
    screen.Fill(color.RGBA{R: 0, G: 0, B: 0, A: 128}) // Semi-transparent
    
    // Draw content on top
    g.drawContent(screen)
}
```

**Requirements for X11:**
- Compositing window manager (KWin, GNOME Shell, etc.)
- ARGB visual support

### 5.2 Fyne Transparency Limitations

**Current State (NOT supported):**
```go
// This does NOT work in Fyne:
window.Canvas().SetBackgroundColor(color.RGBA{0, 0, 0, 0}) // Window still opaque
```

**Fyne Issues Tracking This:**
- [#181](https://github.com/fyne-io/fyne/issues/181) - Transparent window background (since 2019)
- [#1297](https://github.com/fyne-io/fyne/issues/1297) - Transparent Background
- [#3004](https://github.com/fyne-io/fyne/issues/3004) - Transparent and frameless window

### 5.3 Workaround Options (If Migration Required)

1. **Widget-Level Transparency Only**
   ```go
   // Widgets can have transparent backgrounds
   rect := canvas.NewRectangle(color.RGBA{0, 0, 0, 128})
   // But the window itself remains opaque
   ```

2. **Custom Fyne Fork** (Not Recommended)
   - Modify `fyne/internal/driver/glfw/window.go`
   - Add `glfw.WindowHint(glfw.TransparentFramebuffer, glfw.True)`
   - Requires maintaining a fork; may break with Fyne updates

3. **Screenshot-Based Background**
   - Capture desktop screenshot
   - Use as window background
   - Fake transparency (not real-time)

4. **Native Window Handle Manipulation**
   - Use unsafe Go to access underlying GLFW window
   - Platform-specific X11/Windows code
   - Breaks cross-platform compatibility

### 5.4 Recommendation

**Maintain Ebitengine for transparency-dependent features:**

```go
// Hybrid approach (if needed):
// Use Ebitengine for main rendering window
// Use Fyne for configuration dialogs or settings UI
```

---

## 6. Testing & Validation

### 6.1 Migration Verification Checklist

**Core Rendering:**
- [ ] Rectangle drawing (filled and stroked)
- [ ] Line drawing (solid and dashed)
- [ ] Arc drawing (circles, arcs, ellipses)
- [ ] Bezier curve drawing
- [ ] Path operations (move, line, curve, close)
- [ ] Fill operations (solid color, patterns)
- [ ] Stroke operations (width, caps, joins)

**Cairo Compatibility:**
- [ ] `cairo_set_source_rgba` equivalent
- [ ] `cairo_rectangle` equivalent
- [ ] `cairo_arc` equivalent
- [ ] `cairo_curve_to` equivalent
- [ ] `cairo_fill` and `cairo_stroke` equivalent
- [ ] `cairo_save` and `cairo_restore` equivalent
- [ ] `cairo_clip` equivalent
- [ ] Gradient patterns (linear, radial)
- [ ] Surface patterns

**Text Rendering:**
- [ ] Basic text drawing
- [ ] Text measurement
- [ ] Font family selection
- [ ] Font size changes
- [ ] Bold/italic/regular styles
- [ ] Text alignment

**Transformations:**
- [ ] Translation
- [ ] Rotation
- [ ] Scaling
- [ ] Matrix transformations
- [ ] Coordinate conversion

**Widgets:**
- [ ] ProgressBar (horizontal and vertical)
- [ ] Gauge (arc-based)
- [ ] LineGraph
- [ ] BarGraph
- [ ] Histogram
- [ ] ImageWidget

**Transparency (BLOCKER for Fyne):**
- [ ] Window-level transparency
- [ ] Semi-transparent backgrounds
- [ ] Per-pixel alpha blending

### 6.2 Performance Benchmarks

| Metric | Ebitengine Target | Fyne Target |
|--------|-------------------|-------------|
| Startup time | < 100ms | < 200ms |
| Frame time (60 FPS) | < 16ms | < 16ms |
| Memory usage | < 50MB | < 75MB |
| CPU idle | < 1% | < 2% |
| Path rendering (1000 pts) | < 5ms | < 10ms |
| Text rendering (100 lines) | < 2ms | < 3ms |

### 6.3 Visual Regression Testing

```bash
# Generate reference screenshots with Ebitengine
go test -run TestVisualRegression ./internal/render/ -update-golden

# Compare Fyne output against reference
go test -run TestVisualRegression ./internal/render/fyne/
```

---

## 7. Risks & Mitigations

### 7.1 Critical Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| **No ARGB transparency in Fyne** | CERTAIN | CRITICAL | Do not migrate; wait for Fyne feature |
| Fyne API changes | Medium | High | Pin to specific version; use abstraction |
| Performance degradation | Medium | High | Early benchmarking; software fallbacks |
| Cairo API gaps | High | High | Software implementation for missing features |

### 7.2 Technical Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Path rasterization quality | Medium | Medium | Use established algorithms; anti-aliasing |
| Font rendering differences | Medium | Medium | Document differences; provide fallbacks |
| Blend mode limitations | High | Medium | Software blending for unsupported modes |
| Memory usage increase | Medium | Low | Profile early; optimize hot paths |
| Cross-platform inconsistencies | Medium | Medium | Platform-specific testing matrix |

### 7.3 Project Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Extended timeline | High | Medium | Phased approach; prioritize core features |
| Breaking changes | Medium | High | Maintain abstraction layer; feature flags |
| User migration issues | Low | Medium | Provide migration guide; compatibility mode |

### 7.4 Decision Matrix

| If This Happens... | Then Do This... |
|--------------------|-----------------|
| Fyne adds transparency support | Re-evaluate migration with updated timeline |
| Fyne 3.0 released with breaking changes | Update migration plan; assess new APIs |
| Performance unacceptable | Optimize critical paths; consider hybrid approach |
| Cairo compatibility gaps too large | Document limitations; provide workarounds |
| User feedback requests transparency | Confirm staying with Ebitengine |

---

## 8. Conclusion

### 8.1 Summary

This analysis reveals that **migrating from Ebitengine to Fyne would be counterproductive** for the go-conky project's stated goal of achieving ARGB transparency. The key findings are:

1. **Ebitengine already supports ARGB transparency** via `ebiten.SetScreenTransparent(true)`
2. **Fyne does NOT support true window transparency** (feature requested since 2019)
3. Migration would require 6,500+ lines of code changes across 15+ files
4. Many Cairo-compatible features would need software reimplementation

### 8.2 Recommendations

**Immediate Actions:**
1. Enable transparency in Ebitengine by adding `ebiten.SetScreenTransparent(true)`
2. Document the transparency setup in user documentation
3. Close or reassess any migration tickets

**If Migration Still Desired:**
1. Wait for Fyne to implement transparency (monitor Issue #181)
2. Implement the abstraction layer now to ease future migration
3. Consider a hybrid approach: Ebitengine for rendering, Fyne for configuration UI

### 8.3 Timeline Summary

| Phase | Duration | Status |
|-------|----------|--------|
| Current state analysis | 1 day | ✅ Complete |
| Abstraction layer | 2 weeks | 🔜 Optional |
| Core Fyne rendering | 3 weeks | ⏸️ Blocked on transparency |
| Cairo compatibility | 4 weeks | ⏸️ Blocked |
| Widget migration | 2 weeks | ⏸️ Blocked |
| Testing & validation | 2 weeks | ⏸️ Blocked |
| **Total** | **13 weeks** | **⏸️ Not recommended** |
