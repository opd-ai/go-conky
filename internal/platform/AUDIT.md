# Audit: internal/platform
**Date**: 2026-02-23
**Status**: Needs Work

## Summary
The platform package provides cross-platform system monitoring abstractions with 90 Go files implementing native support for Linux, Windows, macOS, and Android, plus remote SSH monitoring. Overall architecture is excellent with strong interface design and comprehensive provider implementations. However, test execution is blocked by Ebiten initialization issues, and several medium-priority issues exist around incomplete features and error handling patterns.

## Issues Found
- [ ] high concurrency — Swallowed errors during SSH session cleanup could mask connection failures (`remote.go:357-358`, `remote.go:361-362`, `remote.go:400-401`)
- [ ] high error-handling — Multiple format errors don't use %w wrapping, breaking error chains (`android_battery.go:41`, `android_cpu.go:73,79,243,248,256`, `android_filesystem.go:160`, `android_network.go:67`, `android_sensors.go:186`, `darwin_battery.go:44`)
- [ ] med incomplete — macOS DiskIO() returns stub zero values with TODO comment (`darwin_filesystem.go:118`)
- [ ] med testing — Test suite blocked by Ebiten initialization (glfw: DISPLAY environment variable missing); requires headless test environment
- [ ] low documentation — No package-level doc.go for remote monitoring features despite 500+ lines implementing SSH connection management
- [ ] low api-design — RemoteConfig struct has 17 fields, could benefit from builder pattern or option functions

## Test Coverage
Unable to determine percentage due to test initialization failure. Package has 217 test functions across 7,319 lines of test code (approximately 45% test-to-code ratio), suggesting comprehensive test coverage if executable.

## Dependencies
**External:**
- `golang.org/x/crypto/ssh` — SSH client implementation for remote monitoring
- `golang.org/x/crypto/ssh/agent` — SSH agent authentication support
- `golang.org/x/crypto/ssh/knownhosts` — Host key verification

**Internal:**
- `github.com/opd-ai/go-conky/pkg/conky` — CircuitBreaker for SSH failure protection

**Platform-Specific:**
- Windows: `syscall` package for PDH (Performance Data Helper) API access
- Linux/Android: `/proc` filesystem parsing for metrics
- macOS: `sysctl` and IOKit (noted as requiring CGO for full DiskIO implementation)

## Recommendations
1. **HIGH PRIORITY**: Wrap SSH session cleanup errors in remote.go:357-362,400-401 with proper error context, or log them instead of silent `_ =` assignment
2. **HIGH PRIORITY**: Update all fmt.Errorf calls to use %w for error wrapping (10+ locations identified across android/darwin files)
3. **MEDIUM PRIORITY**: Implement darwin_filesystem.go DiskIO() or document as intentional limitation if iostat/CGO dependency rejected
4. **MEDIUM PRIORITY**: Configure headless test environment (Xvfb or mock Ebiten) to enable test execution in CI/CD
5. **LOW PRIORITY**: Consider builder pattern for RemoteConfig to reduce API surface and improve usability
