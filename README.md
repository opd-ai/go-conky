# Conky-Go

A 100% compatible reimplementation of [Conky](https://github.com/brndnmtthws/conky) system monitor in Go, built with modern architecture and cross-platform support.

## Why Conky-Go?

- **Perfect Compatibility**: Run your existing `.conkyrc` and Lua configurations without modification
- **Modern Architecture**: Built with Go for better memory safety, concurrency, and maintainability  
- **Cross-Platform**: Native support for Linux with planned Windows/macOS compatibility
- **Performance**: Leverages Ebiten's optimized 2D rendering pipeline for smooth 60fps updates
- **Safe Lua Execution**: Sandboxed Lua scripts with resource limits prevent system abuse

## Technology Stack

- **Go 1.21+**: Core language and standard library
- **[Ebiten](https://github.com/hajimehoshi/ebiten)**: 2D game engine for rendering (Apache 2.0)
- **[Golua](https://github.com/arnodel/golua)**: Pure Go Lua 5.4 implementation with sandboxing
- **Standard Library**: Direct `/proc` filesystem access for system monitoring

## Current Status

✅ **Core Implementation Complete** - Integration in progress

- [x] Project architecture and implementation plan
- [x] Comprehensive system monitoring backend (CPU, Memory, Network, Disk, Battery, Audio, etc.)
- [x] Ebiten rendering engine with text, widgets, and graphs
- [x] Golua integration with Conky API and Cairo bindings
- [x] Configuration parser (legacy `.conkyrc` + Lua formats)
- [x] Cairo compatibility layer for Lua scripts
- [x] Performance profiling and memory leak detection
- [ ] Full end-to-end integration
- [ ] Packaging and distribution

## Quick Start

### Prerequisites

- Go 1.21 or later
- Linux with X11 (primary target)
- X11 development headers:
  ```bash
  sudo apt-get install libx11-dev libxext-dev libxrandr-dev libxcursor-dev libxinerama-dev libxi-dev libgl1-mesa-dev libxxf86vm-dev
  ```

### Build and Run

```bash
# Clone the repository
git clone https://github.com/opd-ai/go-conky.git
cd go-conky

# Install dependencies
make deps

# Build the binary
make build

# Run with your existing Conky config
./build/conky-go -c ~/.conkyrc
```

## Configuration Compatibility

Conky-Go supports both legacy and modern configuration formats:

```lua
-- Modern Lua configuration (recommended)
conky.config = {
    background = false,
    font = 'DejaVu Sans Mono:size=10',
    update_interval = 1.0,
}

conky.text = [[
${color grey}CPU Usage:$color $cpu%
${color grey}RAM Usage:$color $mem/$memmax
]]
```

```ini
# Legacy .conkyrc format (fully supported)
background no
font DejaVu Sans Mono:size=10
update_interval 1.0

TEXT
${color grey}CPU Usage:$color $cpu%
${color grey}RAM Usage:$color $mem/$memmax
```

See the [Migration Guide](docs/migration.md) for detailed compatibility information.

## Development

### Building

```bash
make deps      # Install dependencies
make build     # Build binary
make test      # Run tests with race detection
make lint      # Run linter
make coverage  # Generate test coverage report
```

### Running Tests

```bash
# Run all tests
make test

# Run benchmarks
make bench

# Run integration tests
make integration
```

### Project Structure

```
conky-go/
├── cmd/conky-go/           # Main executable entry point
├── internal/
│   ├── config/             # Configuration parsing and validation
│   ├── lua/                # Golua integration and Conky API
│   ├── monitor/            # System monitoring backend
│   ├── profiling/          # CPU/memory profiling tools
│   └── render/             # Ebiten rendering engine
├── test/
│   ├── configs/            # Test configuration files
│   └── integration/        # Integration tests
├── docs/                   # Documentation
│   ├── architecture.md     # System architecture
│   ├── migration.md        # Migration guide from Conky
│   └── api.md              # API reference
└── scripts/                # Build and development scripts
```

### Documentation

- [Architecture Guide](docs/architecture.md) - System design and component overview
- [Migration Guide](docs/migration.md) - Migrating from Conky to Conky-Go
- [API Reference](docs/api.md) - Go packages and Lua API documentation

## Contributing

We welcome contributions! This project follows the "lazy programmer" philosophy - prefer well-tested libraries over custom implementations.

### Guidelines

- **Use interfaces** for all network types (`net.Conn`, `net.PacketConn`, `net.Addr`)
- **Protect shared state** with proper mutex usage (`sync.RWMutex` for read-heavy data)
- **Handle errors explicitly** - never ignore them, wrap with context using `fmt.Errorf("%w", err)`
- **Leverage existing libraries** - only write glue code, not core functionality
- **Respect licenses** - document all dependencies and their licenses

### Prohibited Dependencies

- ❌ `libp2p` - Use standard library networking instead
- ❌ Web frameworks (`echo`, `chi`, `gin`) - Use `net/http` directly
- ❌ CGO bindings where pure Go alternatives exist

See the [Architecture Guide](docs/architecture.md) for detailed design principles.

## License Compliance

All dependencies use permissive licenses compatible with commercial use:

- **Go Standard Library**: BSD-3-Clause
- **Ebiten**: Apache License 2.0  
- **Golua**: MIT License

## Roadmap

| Phase | Status | Description |
|-------|--------|-------------|
| Phase 1 | ✅ Complete | Foundation - project structure, basic monitoring |
| Phase 2 | ✅ Complete | System monitoring backend (CPU, Memory, Network, etc.) |
| Phase 3 | ✅ Complete | Ebiten rendering engine with widgets |
| Phase 4 | ✅ Complete | Lua integration and Cairo compatibility |
| Phase 5 | ✅ Complete | Configuration parser and migration tools |
| Phase 6 | 🔄 In Progress | Testing, documentation, and packaging |

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Acknowledgments

- Original [Conky](https://github.com/brndnmtthws/conky) project and maintainers
- [Ebiten](https://github.com/hajimehoshi/ebiten) game engine by Hajime Hoshi
- [Golua](https://github.com/arnodel/golua) pure Go Lua implementation
