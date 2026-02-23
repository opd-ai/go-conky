# Audit: github.com/opd-ai/go-conky/internal/lua
**Date**: 2026-02-23  
**Status**: Complete

## Summary
The `internal/lua` package provides Golua integration for conky-go, implementing Lua runtime management, Conky API functions, Cairo drawing bindings, hook system, and conditional parsing. The package is well-architected with strong test coverage (1.43:1 test:source ratio, 9366 test lines vs 6549 source lines). Mutex usage is consistent across shared state (45 lock operations). However, several issues were identified including a potential race condition in cache cleanup, missing package-level documentation file, and unimplemented MPD/music player features marked as stubs.

## Issues Found
- [ ] **medium** Concurrency Safety — Potential race condition in `api.go:2459-2484` - `StartCacheCleanup()` reads `cleanupConfig.CleanupInterval` after unlocking mutex, but `SetCacheCleanupConfig()` can modify it concurrently
- [ ] **low** Documentation — Missing `doc.go` file for package-level documentation (package godoc exists but scattered across multiple files)
- [ ] **low** Stub Implementation — `conditionals.go:387-390` - `evalIfMPDPlaying()` always returns false, stub comment acknowledges MPD integration not implemented
- [ ] **low** Stub Implementation — `api.go:2191-2203` - `resolveNvidiaGraph()` returns GPU utilization as text instead of rendering graph, comment indicates graph rendering not yet implemented
- [ ] **low** Error Handling — `api.go:2402-2409` - Silent error swallowing in `luaValueToString()` using blank identifier for `TryString()`, `TryInt()`, `TryFloat()` errors (acceptable given type checking, but undocumented)
- [ ] **low** Test Infrastructure — Tests panic with "GLFW library is not initialized" due to Ebiten dependency, requiring headless/mock testing approach or test build tags

## Test Coverage
**Estimated**: 80-90% (cannot run `go test -cover` due to GLFW/Ebiten initialization requirement in headless environment)

**Evidence**:
- Test:Source line ratio: 1.43 (9366 test lines / 6549 source lines)
- All source files have corresponding `*_test.go` files
- Comprehensive table-driven tests observed in `api_test.go`, `hooks_test.go`, `runtime_test.go`
- Tests cover concurrent access patterns, cache cleanup, hook management, variable parsing

**Target**: 65% ✓ (likely exceeds target based on test volume)

## Dependencies

**External**:
- `github.com/arnodel/golua/lib` - Lua standard library implementation (pure Go)
- `github.com/arnodel/golua/runtime` - Golua runtime core (pure Go)

**Internal**:
- `internal/monitor` - System monitoring data providers (CPU, memory, network, etc.)
- `internal/render` - Cairo-compatible rendering engine (Ebiten-based)

**Standard Library**: `bytes`, `errors`, `fmt`, `io`, `io/fs`, `os`, `os/exec`, `regexp`, `strconv`, `strings`, `sync`, `time`

**Assessment**: Clean dependency tree, no circular dependencies detected, appropriate use of interfaces for testability (SystemDataProvider)

## Recommendations
1. **Fix race condition in StartCacheCleanup** - Store `interval` in local variable before unlocking mutex at `api.go:2469`, or use atomic operations for config access
2. **Add doc.go file** - Consolidate package-level documentation into canonical `doc.go` file with usage examples
3. **Document stub implementations** - Add explicit godoc comments to stub functions indicating they are placeholders with tracking issue/roadmap references
4. **Add build tags for tests** - Use `//go:build !headless` or mock Ebiten dependencies to enable CI testing without X11/GLFW
5. **Document error swallowing rationale** - Add inline comment explaining why `TryString/TryInt/TryFloat` errors are ignored in type conversion helpers
