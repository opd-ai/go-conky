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
| `ebiten.SetScreenTransparent()` | ❌ Not called | `internal/render/game.go` |
| Background alpha from config | ❌ Not wired | `internal/render/game.go:Draw()` |

**Key Finding**: All ARGB configuration options are already parsed and stored in `config.WindowConfig`, but the rendering layer (`internal/render/`) does not use these values to enable Ebitengine's transparency features.

---

## Implementation Phases

### Phase 1: Enable Ebitengine Screen Transparency
**Objective:** Wire the existing `Transparent` and `ARGBVisual` config options to Ebitengine's transparency API

**Tasks:**
1. Add `Transparent` and `ARGBVisual` fields to `render.Config` struct in `internal/render/types.go`
2. Call `ebiten.SetScreenTransparent(true)` in `Game.Run()` when transparency is enabled
3. Modify `Game.Draw()` to skip background fill or use transparent color when ARGB is active
4. Update `pkg/conky/impl.go` to pass transparency config from `config.WindowConfig` to `render.Config`
5. Add unit tests for transparency state propagation

**Dependencies:**
- `github.com/hajimehoshi/ebiten/v2` v2.8+ (go.mod currently uses v2.8.8)

**Estimated Complexity:** Low

---

### Phase 2: Implement ARGB Alpha Value Support
**Objective:** Apply the `ARGBValue` (0-255) configuration to control window opacity level

**Tasks:**
1. Add `ARGBValue` field to `render.Config` struct
2. Modify background color alpha channel based on `ARGBValue` when `ARGBVisual` is true
3. Create helper function `applyARGBAlpha(baseColor color.RGBA, argbValue int) color.RGBA`
4. Apply alpha value to `BackgroundColor` in `Game.Draw()` method
5. Add validation: clamp `ARGBValue` to 0-255 range in config parsing
6. Add integration test verifying alpha blending behavior

**Dependencies:**
- No additional dependencies

**Estimated Complexity:** Low

---

### Phase 3: Window Manager Hints for Transparency
**Objective:** Configure window properties required for compositor transparency on X11/Wayland

**Tasks:**
1. Add `SetWindowDecorated(false)` call when `own_window_hints` includes "undecorated"
2. Implement window type hints mapping (`desktop`, `dock`, `panel`, `override`)
3. Add `SetWindowFloating(true)` for overlay-style windows
4. Create platform abstraction interface for window hints (future-proofing)
5. Document compositor requirements (picom, compton, KWin, etc.)

**Dependencies:**
- `github.com/hajimehoshi/ebiten/v2` window management APIs

**Estimated Complexity:** Medium

---

### Phase 4: Background Drawing Modes
**Objective:** Support multiple background transparency modes matching original Conky behavior

**Tasks:**
1. Implement `own_window_colour` support for custom background colors
2. Add pseudo-transparency mode (screenshot-based background) as fallback
3. Support gradient backgrounds with alpha channels
4. Create `BackgroundRenderer` interface for extensibility
5. Add `none` background mode (fully transparent, no fill)
6. Add visual tests comparing output with reference screenshots

**Dependencies:**
- `image` standard library for screenshot handling (pseudo-transparency)

**Estimated Complexity:** Medium

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

- [ ] ARGB windows render with proper alpha blending on X11 with compositor
- [ ] Background transparency configurable via `own_window_argb_value` (0-255)
- [ ] `own_window_transparent yes` enables transparent mode
- [ ] `own_window_argb_visual yes` enables 32-bit ARGB visual
- [ ] No performance regression on existing functionality (< 5% impact)
- [ ] Fallback to solid background when compositor unavailable
- [ ] Documentation covers setup for major compositors
- [ ] Unit tests cover all transparency configuration combinations

---

## Implementation Priority

| Phase | Timeline | Priority | Description |
|-------|----------|----------|-------------|
| Phase 1 | Week 1 | Critical | Core transparency |
| Phase 2 | Week 1-2 | Critical | Alpha value support |
| Phase 3 | Week 2-3 | Important | Window hints |
| Phase 4 | Week 3-4 | Nice-to-have | Background modes |
| Phase 5 | Week 4 | Important | Testing/docs |

**Total Estimated Timeline:** 4 weeks

---

## Appendix: Key Files to Modify

| File | Changes |
|------|---------|
| `internal/render/types.go` | Add `Transparent`, `ARGBVisual`, `ARGBValue` fields |
| `internal/render/game.go` | Call `SetScreenTransparent()`, modify `Draw()` |
| `pkg/conky/impl.go` | Wire config values to render.Config |
| `internal/config/validation.go` | Add ARGBValue range validation |
| `docs/transparency.md` | New documentation file |
