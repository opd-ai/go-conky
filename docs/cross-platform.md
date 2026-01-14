# Cross-Platform Deployment Guide

This guide covers installation, configuration, and deployment of go-conky across different operating systems and platforms.

## Overview

go-conky supports multiple platforms through a unified Platform interface that abstracts OS-specific system monitoring implementations:

- **Linux** - Full native support (primary platform)
- **Windows** - Full native support via WMI/PDH APIs
- **macOS** - Full native support via sysctl/IOKit
- **Remote Systems** - Monitor remote Linux/macOS systems via SSH without agent installation

## Supported Platforms

### Platform Feature Matrix

| Feature | Linux | Windows | macOS | Remote (Linux) | Remote (macOS) |
|---------|-------|---------|-------|----------------|----------------|
| CPU Usage (per-core) | ✓ | ✓ | ✓ | ✓ | ✓ |
| CPU Frequency | ✓ | ✓ | ✓ | ✓ | ✓ |
| CPU Info | ✓ | ✓ | ✓ | ✓ | ✓ |
| Load Average | ✓ | ⚠️ Not supported | ✓ | ✓ | ✓ |
| Memory Stats | ✓ | ✓ | ✓ | ✓ | ✓ |
| Swap/Page File | ✓ | ✓ | ✓ | ✓ | ✓ |
| Network Interfaces | ✓ | ✓ | ✓ | ✓ | ✓ |
| Network Stats | ✓ | ✓ | ✓ | ✓ | ✓ |
| Filesystem Mounts | ✓ | ✓ | ✓ | ✓ | ✓ |
| Filesystem Usage | ✓ | ✓ | ✓ | ✓ | ✓ |
| Disk I/O | ✓ | ✓ | ✓ | ✓ | Limited |
| Battery Status | ✓ | ✓ | ✓ | N/A | N/A |
| Temperature Sensors | ✓ | ✓ | ✓* | ✓ | - |
| Fan Sensors | ✓ | ✓ | - | ✓ | - |
| Rendering (Ebiten) | ✓ | ✓ | ✓ | N/A | N/A |

**Legend:**
- ✓ = Fully supported
- ⚠️ = Not available on this platform
- Limited = Partial support or limited functionality
- N/A = Not applicable
- \* = Requires root/admin privileges on macOS

## Linux Installation

### Prerequisites

**Required packages:**
```bash
# Debian/Ubuntu
sudo apt-get update
sudo apt-get install libx11-dev libxext-dev libxrandr-dev libxcursor-dev \
                     libxinerama-dev libxi-dev libgl1-mesa-dev libxxf86vm-dev

# Fedora/RHEL
sudo dnf install libX11-devel libXrandr-devel libXcursor-devel \
                 libXinerama-devel libXi-devel mesa-libGL-devel libXxf86vm-devel

# Arch Linux
sudo pacman -S libx11 libxrandr libxcursor libxinerama libxi mesa libxxf86vm
```

**Go 1.24 or later:**
```bash
# Download from https://golang.org/dl/
wget https://go.dev/dl/go1.24.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.24.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
```

### Building from Source

```bash
# Clone repository
git clone https://github.com/opd-ai/go-conky.git
cd go-conky

# Install dependencies
make deps

# Build binary
make build

# Install to system (optional)
sudo make install
```

### Pre-built Binaries

Download the latest release for your architecture:

```bash
# For x86_64/amd64
wget https://github.com/opd-ai/go-conky/releases/latest/download/conky-go-linux-amd64.tar.gz
tar xzf conky-go-linux-amd64.tar.gz
sudo cp conky-go-linux-amd64 /usr/local/bin/conky-go
sudo chmod +x /usr/local/bin/conky-go

# For ARM64
wget https://github.com/opd-ai/go-conky/releases/latest/download/conky-go-linux-arm64.tar.gz
tar xzf conky-go-linux-arm64.tar.gz
sudo cp conky-go-linux-arm64 /usr/local/bin/conky-go
sudo chmod +x /usr/local/bin/conky-go
```

### Running on Linux

```bash
# Run with existing .conkyrc
conky-go -c ~/.conkyrc

# Run with Lua config
conky-go -c ~/.config/conky/conky.conf

# Run as background daemon
conky-go -c ~/.conkyrc &
```

### Linux-Specific Configuration

```lua
conky.config = {
    -- Linux window manager integration
    own_window = true,
    own_window_type = 'desktop',  -- or 'normal', 'dock', 'override'
    own_window_transparent = true,
    own_window_hints = 'undecorated,below,sticky,skip_taskbar,skip_pager',
    
    -- Display settings
    alignment = 'top_right',
    gap_x = 10,
    gap_y = 40,
}
```

## Windows Installation

### Prerequisites

**Windows 10/11 or Windows Server 2019+**
- No additional system libraries required (pure Go implementation)
- Administrator privileges recommended for full hardware monitoring

### Building from Source

```powershell
# Install Go 1.24 or later from https://golang.org/dl/

# Clone repository
git clone https://github.com/opd-ai/go-conky.git
cd go-conky

# Build binary
go build -o build\conky-go.exe .\cmd\conky-go
```

Or use the Makefile with `make` (requires Make for Windows):
```powershell
make build-windows
```

### Pre-built Binaries

Download the Windows release:

```powershell
# Download from GitHub releases
Invoke-WebRequest -Uri "https://github.com/opd-ai/go-conky/releases/latest/download/conky-go-windows-amd64.zip" -OutFile "conky-go.zip"
Expand-Archive conky-go.zip -DestinationPath .
```

### Running on Windows

```powershell
# Run with config file
.\conky-go.exe -c "C:\Users\YourName\.conkyrc"

# Run minimized as background process
Start-Process .\conky-go.exe -ArgumentList "-c","$env:USERPROFILE\.conkyrc" -WindowStyle Hidden
```

### Windows-Specific Notes

**Path Separators:**
- Use Windows-style paths with backslashes or forward slashes
- Example: `C:\Users\YourName\.conkyrc` or `C:/Users/YourName/.conkyrc`

**Performance Monitoring:**
- CPU monitoring uses Performance Data Helper (PDH) APIs
- Memory stats via GlobalMemoryStatusEx
- Network stats via GetIfTable2
- Requires Windows Performance Counters to be enabled

**Load Average:**
- Windows does not have a native load average concept
- `LoadAverage()` calls will return an error on Windows
- Use CPU percentage instead

**Sensors:**
- Limited sensor support (basic thermal monitoring only)
- Advanced sensor monitoring requires vendor-specific tools

### Windows Configuration Example

```lua
conky.config = {
    -- Windows-specific settings
    alignment = 'top_right',
    gap_x = 10,
    gap_y = 40,
    
    -- Window style
    own_window = true,
    own_window_type = 'normal',
    own_window_transparent = false,
    own_window_argb_visual = true,
    own_window_argb_value = 200,
}

conky.text = [[
${color grey}System:$color Windows ${execi 3600 systeminfo | findstr /B /C:"OS Name"}
${color grey}Uptime:$color $uptime
${color grey}CPU:$color $cpu% ${cpubar 8}
${color grey}RAM:$color $mem/$memmax - $memperc% ${membar 8}
${color grey}Swap:$color $swap/$swapmax - $swapperc% ${swapbar 8}
${color grey}Processes:$color $processes  ${color grey}Running:$color $running_processes

${color grey}Networking:
Down:$color ${downspeed eth0}  ${color grey}Up:$color ${upspeed eth0}
${downspeedgraph eth0 32,150} ${upspeedgraph eth0 32,150}

${color grey}File systems:
C:\ $color${fs_used C:\}/${fs_size C:\} ${fs_bar 6 C:\}
]]
```

## macOS Installation

### Prerequisites

**macOS 11 (Big Sur) or later**

**Xcode Command Line Tools:**
```bash
xcode-select --install
```

**Go 1.24 or later:**
```bash
# Using Homebrew
brew install go

# Or download from https://golang.org/dl/
```

### Building from Source

```bash
# Clone repository
git clone https://github.com/opd-ai/go-conky.git
cd go-conky

# Install dependencies
make deps

# Build binary
make build

# Install to /usr/local/bin (optional)
sudo make install
```

Or use cross-platform build:
```bash
make build-darwin
```

### Pre-built Binaries

Download the macOS release:

```bash
# For Intel Macs (x86_64)
wget https://github.com/opd-ai/go-conky/releases/latest/download/conky-go-darwin-amd64.tar.gz
tar xzf conky-go-darwin-amd64.tar.gz
sudo cp conky-go-darwin-amd64 /usr/local/bin/conky-go
sudo chmod +x /usr/local/bin/conky-go

# For Apple Silicon (ARM64/M1/M2/M3)
wget https://github.com/opd-ai/go-conky/releases/latest/download/conky-go-darwin-arm64.tar.gz
tar xzf conky-go-darwin-arm64.tar.gz
sudo cp conky-go-darwin-arm64 /usr/local/bin/conky-go
sudo chmod +x /usr/local/bin/conky-go
```

### Running on macOS

```bash
# Run with config
conky-go -c ~/.conkyrc

# Run as background process
conky-go -c ~/.conkyrc &

# Run at login (using launchd)
# Create ~/Library/LaunchAgents/com.user.conky-go.plist
```

### macOS-Specific Notes

**System Monitoring APIs:**
- CPU: Uses `sysctl` and `host_processor_info` Mach APIs
- Memory: Uses `vm_stat` and `sysctl`
- Network: Uses `getifaddrs` and `sysctl`
- Filesystem: Uses `statfs`

**Temperature Sensors:**
- Requires root privileges for `powermetrics`
- Limited sensor support compared to Linux
- Consider using `iStats` or similar tools for detailed sensor data

**Permissions:**
- No special permissions required for basic monitoring
- Full Disk Access may be needed for certain filesystem operations

**Window Management:**
- macOS window positioning may behave differently from Linux
- Transparency and window hints may not work identically

### macOS Configuration Example

```lua
conky.config = {
    -- macOS-specific settings
    alignment = 'top_right',
    gap_x = 10,
    gap_y = 40,
    
    -- Display
    own_window = true,
    own_window_type = 'normal',
    own_window_transparent = true,
    own_window_argb_visual = true,
    own_window_argb_value = 150,
}

conky.text = [[
${color grey}System:$color macOS ${execi 3600 sw_vers -productVersion}
${color grey}Kernel:$color $kernel
${color grey}Uptime:$color $uptime
${color grey}CPU:$color $cpu% @ ${freq_g}GHz
${cpugraph 32,150}

${color grey}RAM:$color $mem/$memmax - $memperc%
${membar 8}

${color grey}Swap:$color $swap/$swapmax - $swapperc%
${swapbar 8}

${color grey}Networking:
Down:$color ${downspeed en0}  ${color grey}Up:$color ${upspeed en0}
${downspeedgraph en0 32,150} ${upspeedgraph en0 32,150}

${color grey}Disk Usage:
/:$color ${fs_used /}/${fs_size /}
${fs_bar 6 /}
]]
```

## Remote Monitoring via SSH

go-conky can monitor remote Linux and macOS systems via SSH **without requiring installation** on the target system. Data is collected using standard shell commands and parsed locally.

### Remote Monitoring Features

- **No Agent Required**: Uses standard shell commands available on target systems
- **Secure**: All data transmitted over encrypted SSH connection
- **Multiple Auth Methods**: Password, private key, or SSH agent
- **Auto-Detection**: Automatically detects remote OS (Linux or macOS)
- **Connection Management**: Automatic reconnection on network failures

### Remote Configuration

See the comprehensive [SSH Remote Monitoring Guide](ssh-remote-monitoring.md) for detailed setup instructions.

**Quick Example:**

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/opd-ai/go-conky/internal/platform"
)

func main() {
    // Configure remote connection
    config := platform.RemoteConfig{
        Host: "server.example.com",
        Port: 22,
        User: "monitor",
        AuthMethod: platform.KeyAuth{
            PrivateKeyPath: "/home/user/.ssh/id_rsa",
        },
        CommandTimeout: 5 * time.Second,
    }
    
    // Create remote platform
    remote, err := platform.NewRemotePlatform(config)
    if err != nil {
        panic(err)
    }
    defer remote.Close()
    
    // Initialize connection
    if err := remote.Initialize(context.Background()); err != nil {
        panic(err)
    }
    
    // Collect metrics
    cpuUsage, _ := remote.CPU().TotalUsage()
    memStats, _ := remote.Memory().Stats()
    
    fmt.Printf("Remote CPU: %.1f%%\n", cpuUsage)
    fmt.Printf("Remote Memory: %.1f%%\n", memStats.UsedPercent)
}
```

### Supported Remote Platforms

| Target OS | Support Level | Available Metrics |
|-----------|---------------|-------------------|
| Linux | Full | CPU, Memory, Network, Disk, Sensors |
| macOS | Full | CPU, Memory, Network, Disk |
| Windows | Planned | Not yet implemented |

## Cross-Platform Build System

### Building for All Platforms

```bash
# Build for all platforms
make build-all

# Build specific platform
make build-linux
make build-windows
make build-darwin

# Create distribution packages
make dist-all
```

### Cross-Compilation Examples

**From Linux to Windows:**
```bash
GOOS=windows GOARCH=amd64 go build -o conky-go.exe ./cmd/conky-go
```

**From Linux to macOS:**
```bash
# Intel Mac
GOOS=darwin GOARCH=amd64 go build -o conky-go-mac ./cmd/conky-go

# Apple Silicon
GOOS=darwin GOARCH=arm64 go build -o conky-go-mac-arm ./cmd/conky-go
```

**From macOS to Linux:**
```bash
GOOS=linux GOARCH=amd64 go build -o conky-go-linux ./cmd/conky-go
```

### Architecture Support

| Platform | Architectures | Notes |
|----------|--------------|-------|
| Linux | amd64, arm64 | Full support |
| Windows | amd64 | Full support |
| macOS | amd64, arm64 | Full support (Intel & Apple Silicon) |

## Troubleshooting

### Linux Issues

**X11 Display Issues:**
```bash
# Ensure DISPLAY is set
echo $DISPLAY

# Grant X11 access
xhost +local:

# Check X11 libraries
dpkg -l | grep libx11
```

**Permission Errors:**
```bash
# For /proc filesystem access
sudo chmod +r /proc/stat /proc/meminfo /proc/cpuinfo

# For hwmon sensors
sudo chmod +r /sys/class/hwmon/hwmon*/temp*_input
```

**Window Not Appearing:**
- Check window manager compatibility
- Try different `own_window_type` settings
- Verify compositor is running

### Windows Issues

**Performance Counters Not Working:**
```powershell
# Rebuild performance counter library
lodctr /R
```

**High CPU Usage:**
- Increase `update_interval` in config
- Disable unused monitoring features
- Check for WMI service issues

**Network Interface Not Found:**
- Use `ipconfig /all` to verify interface names
- Windows uses different naming convention (e.g., "Ethernet", "Wi-Fi")

### macOS Issues

**Sensor Data Not Available:**
```bash
# Temperature monitoring requires root
sudo conky-go -c ~/.conkyrc

# Or use iStats for sensor data
sudo gem install iStats
istats
```

**Window Transparency Not Working:**
- Ensure compositor is enabled
- Try adjusting `own_window_argb_value`
- Check macOS version compatibility

**CPU Frequency Shows Zero:**
- macOS restricts frequency information on newer systems
- Use Activity Monitor for comparison

### Remote Monitoring Issues

**Connection Timeout:**
```bash
# Test SSH connection manually
ssh -v user@host

# Increase timeout in config
config.CommandTimeout = 10 * time.Second
```

**Authentication Failures:**
```bash
# Verify SSH key permissions
chmod 600 ~/.ssh/id_rsa

# Test SSH agent
ssh-add -l

# Use verbose mode for debugging
SSH_DEBUG=1 conky-go -c config.conf
```

**Command Not Found on Remote:**
- Ensure standard utilities are installed (`sysctl`, `cat`, `df`, etc.)
- Check PATH environment variable on remote system
- Some commands may require full paths (e.g., `/usr/bin/uptime`)

## Performance Tuning

### Update Interval

Adjust `update_interval` to balance responsiveness and resource usage:

```lua
conky.config = {
    -- Update every second (responsive but higher CPU)
    update_interval = 1.0,
    
    -- Update every 2 seconds (balanced)
    update_interval = 2.0,
    
    -- Update every 5 seconds (lower CPU usage)
    update_interval = 5.0,
}
```

### Platform-Specific Optimizations

**Linux:**
- Use `/proc` filesystem directly for lowest overhead
- Disable unused sensors to reduce I/O

**Windows:**
- Minimize WMI queries by caching results
- Use PDH for better performance than WMI

**macOS:**
- Cache `sysctl` results where possible
- Avoid frequent `powermetrics` calls (requires root)

**Remote:**
- Use longer `update_interval` for remote systems
- Enable connection pooling for multiple remote hosts
- Cache static data (CPU model, etc.) to reduce SSH overhead

## Configuration Examples

### Universal Configuration

This configuration works across all platforms with automatic adaptation:

```lua
conky.config = {
    alignment = 'top_right',
    gap_x = 10,
    gap_y = 40,
    update_interval = 2.0,
    
    own_window = true,
    own_window_type = 'normal',
    own_window_transparent = true,
    
    font = 'monospace:size=10',
    use_xft = true,
}

conky.text = [[
${color grey}System:$color $nodename
${color grey}Uptime:$color $uptime
${color grey}Kernel:$color $kernel

${color grey}CPU:$color $cpu%
${cpugraph 32,150}

${color grey}RAM:$color $mem/$memmax - $memperc%
${membar 8}

${color grey}Disk:$color ${fs_used /}/${fs_size /}
${fs_bar 6 /}

${color grey}Network:
Down:$color ${downspeed eth0}
Up:$color ${upspeed eth0}
]]
```

### Platform Detection in Lua

```lua
-- Detect platform and adjust config
local function get_platform()
    local handle = io.popen("uname -s")
    local result = handle:read("*a")
    handle:close()
    return result:lower():match("^%s*(.-)%s*$")
end

local platform = get_platform()

conky.config = {
    alignment = 'top_right',
    gap_x = platform == "darwin" and 20 or 10,
    gap_y = 40,
}
```

## Additional Resources

- **Architecture Guide**: [architecture.md](architecture.md) - System design and components
- **Migration Guide**: [migration.md](migration.md) - Migrating from original Conky
- **API Reference**: [api.md](api.md) - Go packages and Lua API documentation
- **SSH Monitoring**: [ssh-remote-monitoring.md](ssh-remote-monitoring.md) - Detailed remote setup
- **Conky Documentation**: [Conky Variables](http://conky.sourceforge.net/variables.html) - Original Conky reference

## Platform Support Status

### Current Status (Phase 7)

- ✅ **Linux**: Production ready (primary platform)
- ✅ **Windows**: Full support implemented and tested
- ✅ **macOS**: Full support implemented and tested  
- ✅ **Remote SSH**: Linux and macOS remote monitoring implemented
- ⏳ **Android**: Planned (not yet implemented)

### Future Enhancements

- **Android Platform**: Native monitoring for Android devices
- **Remote Windows**: SSH monitoring for Windows systems (PowerShell-based)
- **Cross-Platform Rendering**: Enhanced window management for Windows/macOS
- **Platform Auto-Detection**: Automatic configuration adaptation based on detected platform

## Getting Help

- **GitHub Issues**: [opd-ai/go-conky/issues](https://github.com/opd-ai/go-conky/issues)
- **Discussions**: [opd-ai/go-conky/discussions](https://github.com/opd-ai/go-conky/discussions)
- **Original Conky**: [Conky Documentation](http://conky.sourceforge.net/)

## License

go-conky is licensed under the MIT License. See [LICENSE](../LICENSE) for details.

All platform-specific implementations use only permissive-licensed dependencies:
- **Go Standard Library**: BSD-3-Clause
- **golang.org/x/sys**: BSD-3-Clause
- **golang.org/x/crypto**: BSD-3-Clause
