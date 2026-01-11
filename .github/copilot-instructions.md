# Project Overview

Conky-Go is a 100% compatible reimplementation of the [Conky](https://github.com/brndnmtthws/conky) system monitor, written in Go with modern architecture and cross-platform support. The project aims to run existing `.conkyrc` and Lua configurations without modification, while leveraging Go's memory safety, concurrency, and maintainability advantages.

The primary target audience includes Linux users who rely on Conky for system monitoring, developers seeking to contribute to a modern Conky alternative, and system administrators looking for a maintainable monitoring solution. The project emphasizes the "lazy programmer" philosophy—prefer well-tested libraries over custom implementations.

**Current Status**: Early development phase. The core architecture is designed but not yet functional. Key implementation phases include system monitoring backend, Ebiten rendering foundation, Golua integration, configuration parsing, and Cairo compatibility layer.

## Technical Stack

- **Primary Language**: Go 1.21+
- **Frameworks**:
  - [Ebiten](https://github.com/hajimehoshi/ebiten) v2 - 2D game engine for rendering (Apache 2.0)
  - [Golua](https://github.com/arnodel/golua) - Pure Go Lua 5.4 implementation with sandboxing
- **Testing**: Go's built-in testing package with table-driven tests; race detector enabled
- **Build/Deploy**: Make-based build system with `make build`, `make test`, `make install`
- **System Integration**: Linux `/proc` filesystem for system monitoring, X11 for window management

## Code Assistance Guidelines

1. **Use interfaces for all network types** — Always use `net.Conn`, `net.PacketConn`, `net.Addr`, and `net.Listener` rather than concrete types like `net.TCPConn`, `net.UDPConn`, or `net.TCPAddr`. This enhances testability and flexibility.

2. **Protect shared state with proper mutex usage** — Use `sync.RWMutex` for read-heavy data structures. Lock appropriately in all goroutines accessing shared state. Example pattern from PLAN.md shows `mu sync.RWMutex` on all shared structs.

3. **Handle errors explicitly with context** — Never ignore errors. Wrap errors with context using `fmt.Errorf("description: %w", err)`. This provides clear error chains for debugging.

4. **Leverage existing libraries over custom implementations** — Only write glue code, not core functionality. Prefer standard library solutions where adequate. Avoid CGO bindings when pure Go alternatives exist.

5. **Follow the modular architecture** — Place code in appropriate internal packages: `config/` for configuration parsing, `monitor/` for system monitoring, `render/` for Ebiten rendering, `lua/` for Golua integration, `window/` for window management.

6. **Avoid prohibited dependencies** — Do not use `libp2p`, web frameworks (`echo`, `chi`, `gin`), or CGO bindings where pure Go alternatives exist. Use `net/http` directly for any HTTP needs.

7. **Document license compliance** — All dependencies must use permissive licenses compatible with MIT/Apache 2.0. Document all dependencies and their licenses in code or documentation.

## Project Context

- **Domain**: System monitoring with desktop widget rendering. Key concepts include Conky configuration parsing (both legacy `.conkyrc` text format and modern Lua `conky.config` format), system data collection via `/proc` filesystem, and Cairo-compatible 2D drawing commands.

- **Architecture**: Multi-module design with five core components: Configuration Parser, System Monitoring Backend, Rendering Engine (Ebiten), Lua Integration (Golua), and Window Management. Data flows from system → monitor backend → Lua processing → Cairo drawing commands → Ebiten rendering → window display.

- **Key Directories**:
  - `cmd/conky-go/` — Main executable entry point
  - `internal/config/` — Configuration parsing (legacy and Lua formats)
  - `internal/monitor/` — System monitoring (CPU, memory, network, filesystem)
  - `internal/render/` — Ebiten rendering engine and Cairo compatibility layer
  - `internal/lua/` — Golua runtime and Conky Lua API implementation
  - `internal/window/` — X11/Wayland window management
  - `test/configs/` — Test configuration files

- **Configuration**: Supports both legacy `.conkyrc` text format and modern Lua configuration format. Configuration hot-reloading is planned. Parse format detection via `conky.config` presence in file content.

## Quality Standards

- **Testing Requirements**: Maintain comprehensive test coverage using Go's built-in testing package. Write table-driven tests for business logic. Use `go test -race` to detect race conditions. Integration tests should validate real-world Conky configurations.

- **Performance Targets**: Startup time < 100ms, update latency < 16ms (60 FPS capable), memory footprint < 50MB, CPU usage < 1% idle.

- **Code Review Criteria**: Follow Go best practices, ensure proper error handling, validate mutex usage for concurrency safety, verify no prohibited dependencies are introduced.

- **Documentation Standards**: Update architecture documentation when adding new components. Include code examples in API documentation. Document any Lua API differences from original Conky.

## Networking Best Practices

When declaring network variables, always use interface types:

- Never use `net.UDPAddr`, `net.IPAddr`, or `net.TCPAddr`. Use `net.Addr` only instead.
- Never use `net.UDPConn`, use `net.PacketConn` instead.
- Never use `net.TCPConn`, use `net.Conn` instead.
- Never use `net.UDPListener` or `net.TCPListener`, use `net.Listener` instead.
- Never use a type switch or type assertion to convert from an interface type to a concrete type. Use the interface methods instead.

This approach enhances testability and flexibility when working with different network implementations or mocks.

## Development Workflow

```bash
# Install dependencies
make deps

# Run tests with race detection
make test

# Build binary
make build

# Run with existing Conky config
./build/conky-go -c ~/.conkyrc

# Run linter
make lint

# Generate coverage report
make coverage
```

## Conky Compatibility Notes

- **Cairo Functions**: The project must implement 180+ Cairo drawing functions for full Lua script compatibility. Priority is given to most-used functions.
- **System Variables**: Support for 250+ built-in Conky variables is planned.
- **Lua API**: Implement `conky_parse()`, `conky.info.*` variables, and event hooks (`conky_main`, `conky_start`, etc.).
- **Sandboxing**: Golua provides built-in CPU and memory limits for safe script execution.
