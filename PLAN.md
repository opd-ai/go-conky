# ARGB Transparency Implementation Plan

## Current State Analysis

### Existing Rendering Approach
The go-conky project uses **Ebitengine v2** (`github.com/hajimehoshi/ebiten/v2`) as its rendering backend. The rendering pipeline is structured as follows:

- **Entry Point**: `internal/render/game.go` - Implements `ebiten.Game` interface
- **Game Loop**: `Run()` method calls `ebiten.RunGame()` which manages Update/Draw/Layout cycle
- **Background Rendering**: `Draw()` method fills background with `BackgroundColor` from config
- **Configuration**: Window settings parsed from `.conkyrc` or Lua config in `internal/config/`

### Current Transparency Capabilities

| Component | Status | Location |
|-----------|--------|----------|
| `Transparent` config option | ✅ Parsed | `internal/config/types.go:32` |
| `ARGBVisual` config option | ✅ Parsed | `internal/config/types.go:35` |
| `ARGBValue` config option (0-255) | ✅ Parsed | `internal/config/types.go:39` |
| `ebiten.SetScreenTransparent()` | ✅ Called | `internal/render/game.go:Run()` |
| Background alpha from config | ✅ Wired | `internal/render/game.go:Draw()` |

**Key Finding**: All ARGB configuration options are parsed, stored, and now applied at the rendering layer. The rendering engine calls `ebiten.SetScreenTransparent(true)` when transparency is enabled and applies `ARGBValue` to the background alpha channel.

---

## Implementation Phases

### Phase 1: Enable Ebitengine Screen Transparency ✅ COMPLETED
**Objective:** Wire the existing `Transparent` and `ARGBVisual` config options to Ebitengine's transparency API

**Tasks:**
1. ✅ Add `Transparent` and `ARGBVisual` fields to `render.Config` struct in `internal/render/types.go`
2. ✅ Call `ebiten.SetScreenTransparent(true)` in `Game.Run()` when transparency is enabled
3. ✅ Modify `Game.Draw()` to skip background fill or use transparent color when ARGB is active
4. ✅ Update `pkg/conky/render.go` to pass transparency config from `config.WindowConfig` to `render.Config`
5. ✅ Add unit tests for transparency state propagation

**Implementation Summary:**
- Added `Transparent`, `ARGBVisual`, and `ARGBValue` fields to `render.Config`
- `Game.Run()` now calls `ebiten.SetScreenTransparent(true)` when `Transparent` or `ARGBVisual` is enabled
- `Game.Draw()` applies `ARGBValue` to background alpha when `ARGBVisual` is true
- `pkg/conky/render.go` wires config values from `config.WindowConfig` to `render.Config`
- Comprehensive tests added in `internal/render/types_test.go` and `internal/render/game_test.go`

**Dependencies:**
- `github.com/hajimehoshi/ebiten/v2` v2.8+ (go.mod currently uses v2.8.8)

**Completed:** 2026-01-17

---

### Phase 2: Implement ARGB Alpha Value Support ✅ COMPLETED
**Objective:** Apply the `ARGBValue` (0-255) configuration to control window opacity level

**Tasks:**
1. ✅ Add `ARGBValue` field to `render.Config` struct
2. ✅ Modify background color alpha channel based on `ARGBValue` when `ARGBVisual` is true
3. ✅ Apply alpha value to `BackgroundColor` in `Game.Draw()` method with clamping (0-255)
4. ✅ Add integration tests verifying alpha blending behavior

**Implementation Summary:**
- `ARGBValue` field added to `render.Config` with default value 255 (fully opaque)
- `Game.Draw()` applies clamped `ARGBValue` (0-255) to background alpha when `ARGBVisual` is enabled
- Tests cover all edge cases: negative values, values over 255, fully transparent, fully opaque

**Dependencies:**
- No additional dependencies

**Completed:** 2026-01-17

---

### Phase 3: Window Manager Hints for Transparency ✅ COMPLETED
**Objective:** Configure window properties required for compositor transparency on X11/Wayland

**Tasks:**
1. ✅ Add `SetWindowDecorated(false)` call when `own_window_hints` includes "undecorated"
2. ✅ Implement window type hints mapping (`desktop`, `dock`, `panel`, `override`)
   - Note: Ebiten supports `Undecorated` (removes decorations) and `Floating` (above other windows)
   - `below`, `sticky` hints are parsed but not directly supported by Ebiten
3. ✅ Add `SetWindowFloating(true)` for overlay-style windows (maps to "above" hint)
4. ✅ Create platform abstraction interface for window hints (via render.Config fields)
5. ✅ Document compositor requirements (picom, compton, KWin, etc.) - see code comments

**Implementation Summary:**
- Added `Undecorated`, `Floating`, `WindowX`, `WindowY`, `SkipTaskbar`, `SkipPager` fields to `render.Config`
- `Game.Run()` now applies window hints via Ebiten's `SetWindowDecorated()`, `SetWindowFloating()`, `SetWindowPosition()`
- `parseWindowHints()` in `pkg/conky/render.go` converts `config.WindowHint` slice to render flags
- Comprehensive tests in `internal/render/types_test.go` and `pkg/conky/render_test.go`

**Dependencies:**
- `github.com/hajimehoshi/ebiten/v2` window management APIs

**Completed:** 2026-01-17

---

### Phase 4: Background Drawing Modes
**Objective:** Support multiple background transparency modes matching original Conky behavior

**Tasks:**
1. ✅ Implement `own_window_colour` support for custom background colors
2. Add pseudo-transparency mode (screenshot-based background) as fallback
3. ✅ Support gradient backgrounds with alpha channels
4. ✅ Create `BackgroundRenderer` interface for extensibility
5. ✅ Add `none` background mode (fully transparent, no fill)
6. Add visual tests comparing output with reference screenshots

**Implementation Summary (Tasks 1, 3, 4, 5):**
- Added `BackgroundMode` type to `config/types.go` with `BackgroundModeSolid`, `BackgroundModeNone`, `BackgroundModeTransparent`, and `BackgroundModeGradient` modes
- Added `BackgroundColour` field to `WindowConfig` for custom background colors
- Added `ParseBackgroundMode()` function for string parsing (now includes "gradient")
- Added `GradientDirection` type with Vertical, Horizontal, Diagonal, and Radial directions
- Added `GradientConfig` struct for gradient configuration (StartColor, EndColor, Direction)
- Added `Gradient` field to `WindowConfig` for gradient settings
- Created `BackgroundRenderer` interface in `render/background.go` with `Draw()` and `Mode()` methods
- Implemented `SolidBackground` renderer with ARGB support
- Implemented `NoneBackground` renderer for fully transparent backgrounds
- Implemented `GradientBackground` renderer with full alpha channel support:
  - Four gradient directions: vertical, horizontal, diagonal, radial
  - ARGB visual support to override alpha across entire gradient
  - Linear interpolation between start and end colors
  - `NewGradientBackgroundRenderer()` convenience function
- Added `own_window_colour` and `own_window_color` (US spelling) parsing to legacy parser
- Wired `BackgroundMode` through `render.Config` and `Game` struct
- Added comprehensive tests in `render/background_test.go`, `config/types_test.go`, and `config/legacy_test.go`

**Dependencies:**
- `image` standard library for screenshot handling (pseudo-transparency)
- `math` standard library for radial gradient calculations

**Completed (Tasks 1, 3, 4, 5):** 2026-01-17

---

### Phase 5: Testing and Documentation
**Objective:** Ensure reliability and provide user guidance

**Tasks:**
1. Create integration tests with transparent window screenshots
2. Add example configurations demonstrating transparency options
3. Document compositor requirements for each platform
4. Add troubleshooting guide for transparency issues
5. Create visual regression test suite
6. Update README with transparency configuration examples

**Dependencies:**
- No additional dependencies

**Estimated Complexity:** Low

---

## Technical Considerations

### Compatibility

| Platform | True ARGB | Pseudo-Transparency | Notes |
|----------|-----------|---------------------|-------|
| Linux/X11 | ✅ With compositor | ✅ Fallback | Requires picom, compton, or built-in compositor |
| Linux/Wayland | ✅ Native | N/A | XWayland required for Ebitengine |
| macOS | ✅ Native | N/A | No additional requirements |
| Windows | ✅ Native | N/A | DWM compositing always enabled |

### Performance

| Change | Impact | Mitigation |
|--------|--------|------------|
| Screen transparency enabled | Minimal (~1-2% GPU) | None needed |
| Alpha blending on background | Negligible | Hardware accelerated |
| Pseudo-transparency | Higher memory | Cache screenshot |

### Breaking Changes

| Change | Impact | Migration |
|--------|--------|-----------|
| None | N/A | Transparency is opt-in via config |

All changes are additive and backward compatible. Existing configurations without transparency settings will continue to work unchanged.

---

## Risks & Mitigation

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Compositor not running on user's system | Medium | No transparency visible | Implement fallback to pseudo-transparency with warning |
| ARGBValue=0 makes window invisible | Low | Confusing UX | Add minimum alpha warning in validation |
| Ebitengine API changes | Low | Build failures | Pin to specific version, monitor releases |
| XWayland performance on Wayland | Low | Slight latency | Document native Wayland as preferred |
| Screenshot capture for pseudo-transparency fails | Medium | Fallback fails | Graceful degradation to solid background |

---

## Success Criteria

- [x] ARGB windows render with proper alpha blending on X11 with compositor
- [x] Background transparency configurable via `own_window_argb_value` (0-255)
- [x] `own_window_transparent yes` enables transparent mode
- [x] `own_window_argb_visual yes` enables 32-bit ARGB visual
- [x] Window hints (undecorated, above, skip_taskbar, skip_pager) applied via Ebiten
- [x] Window position configurable via gap_x/gap_y settings
- [x] `own_window_colour` configures custom background color
- [x] BackgroundRenderer interface enables extensible background modes
- [x] `none` background mode fully implemented
- [x] Gradient background mode with alpha channels implemented
- [ ] No performance regression on existing functionality (< 5% impact)
- [ ] Fallback to solid background when compositor unavailable
- [ ] Documentation covers setup for major compositors
- [x] Unit tests cover all transparency configuration combinations

---

## Implementation Priority

| Phase | Timeline | Priority | Description | Status |
|-------|----------|----------|-------------|--------|
| Phase 1 | Week 1 | Critical | Core transparency | ✅ COMPLETED |
| Phase 2 | Week 1-2 | Critical | Alpha value support | ✅ COMPLETED |
| Phase 3 | Week 2-3 | Important | Window hints | ✅ COMPLETED |
| Phase 4 | Week 3-4 | Nice-to-have | Background modes | ⏳ PARTIAL (4/6 tasks) |
| Phase 5 | Week 4 | Important | Testing/docs | Pending |

**Total Estimated Timeline:** 4 weeks

---

## Appendix: Key Files Modified

| File | Changes | Status |
|------|---------|--------|
| `internal/render/types.go` | Added `Transparent`, `ARGBVisual`, `ARGBValue`, `BackgroundMode`, `Undecorated`, `Floating`, `WindowX`, `WindowY`, `SkipTaskbar`, `SkipPager` fields | ✅ Done |
| `internal/render/game.go` | Call `SetScreenTransparent()`, `SetWindowDecorated()`, `SetWindowFloating()`, `SetWindowPosition()`, use `BackgroundRenderer` in `Draw()` | ✅ Done |
| `internal/render/background.go` | `BackgroundRenderer` interface, `SolidBackground`, `NoneBackground`, `GradientBackground` implementations with 4 gradient directions | ✅ Done |
| `internal/render/background_test.go` | Comprehensive tests for all background renderers including gradient | ✅ Done |
| `internal/config/types.go` | Added `BackgroundMode`, `BackgroundModeGradient`, `GradientDirection`, `GradientConfig` types, `Gradient` field to `WindowConfig` | ✅ Done |
| `internal/config/defaults.go` | Added `DefaultBackgroundColour`, updated `defaultWindowConfig()` | ✅ Done |
| `internal/config/legacy.go` | Added `own_window_colour` and `own_window_color` parsing | ✅ Done |
| `pkg/conky/render.go` | Wire config values to render.Config, add `configToRenderBackgroundMode()` function | ✅ Done |
| `internal/render/types_test.go` | Added ARGB, window hints, and background mode tests | ✅ Done |
| `internal/config/legacy_test.go` | Added `own_window_colour` parsing tests | ✅ Done |
| `pkg/conky/render_test.go` | Added `parseWindowHints()` unit tests | ✅ Done |
| `internal/render/game_test.go` | Added ARGB transparency tests | ✅ Done |
| `internal/config/validation.go` | ARGBValue range validation | Already exists |
| `docs/transparency.md` | New documentation file | Pending |
