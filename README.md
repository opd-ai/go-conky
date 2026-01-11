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

🚧 **Early Development Phase** - Not yet functional

- [x] Project architecture and implementation plan
- [x] Basic system monitoring backend (CPU, Memory, Uptime)
- [ ] Ebiten rendering foundation
- [ ] Golua integration and Conky API
- [ ] Configuration parser (legacy + Lua)
- [ ] Cairo compatibility layer

## Quick Start

> **Note:** This project is in early development and is not yet functional. The commands below represent the planned workflow.

```bash
# Clone the repository
git clone https://github.com/yourusername/conky-go.git
cd conky-go

# Once implemented, build with:
# make build

# Once built, run with your existing Conky config:
# ./build/conky-go -c ~/.conkyrc
```

## Configuration Compatibility

Conky-Go aims for 100% compatibility with existing configurations:

```lua
-- Modern Lua configuration (supported)
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
# Legacy .conkyrc format (also supported)
background no
font DejaVu Sans Mono:size=10
update_interval 1.0

TEXT
${color grey}CPU Usage:$color $cpu%
${color grey}RAM Usage:$color $mem/$memmax
```

## Development

### Prerequisites

- Go 1.21 or later
- Linux development environment (primary target)
- X11 development headers: `sudo apt-get install libx11-dev libxext-dev`

### Building

> **Note:** Build infrastructure is not yet implemented. The commands below represent the planned build workflow.

```bash
# Planned commands (not yet available):
# make deps     # Install dependencies
# make test     # Run tests
# make build    # Build binary
# make install  # Install system-wide
```

See [PLAN.md](PLAN.md) section 5.2 for the planned Makefile implementation.

### Project Structure

> **Note:** The following directory structure is planned but not yet implemented. See [PLAN.md](PLAN.md) for the complete implementation roadmap.

```
conky-go/
├── cmd/conky-go/          # Main executable (planned)
├── internal/
│   ├── config/            # Configuration parsing (planned)
│   ├── monitor/           # System monitoring (planned)
│   ├── render/            # Ebiten rendering engine (planned)
│   ├── lua/               # Golua integration (planned)
│   └── window/            # Window management (planned)
├── test/configs/          # Test configurations (planned)
└── docs/                  # Documentation (planned)
```

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

See [PLAN.md](PLAN.md) for detailed implementation guidelines and project architecture.

## License Compliance

All dependencies use permissive licenses compatible with commercial use:

- **Go Standard Library**: BSD-3-Clause
- **Ebiten**: Apache License 2.0  
- **Golua**: License verification in progress

## Roadmap

- **Phase 1** (3 weeks): Foundation - basic monitoring and rendering
- **Phase 2** (3 weeks): Complete system monitoring backend
- **Phase 3** (3 weeks): Full rendering engine with Ebiten
- **Phase 4** (3 weeks): Lua integration and Cairo compatibility  
- **Phase 5** (3 weeks): Configuration parser and migration tools
- **Phase 6** (3 weeks): Testing, optimization, and packaging

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Acknowledgments

- Original [Conky](https://github.com/brndnmtthws/conky) project and maintainers
- [Ebiten](https://github.com/hajimehoshi/ebiten) game engine by Hajime Hoshi
- [Golua](https://github.com/arnodel/golua) pure Go Lua implementation

---

**Note**: This project is in early development. Current code may not be functional. Check the [Issues](../../issues) page for current development status and known limitations.
