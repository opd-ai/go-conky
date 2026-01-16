# Migration Guide: Conky to Conky-Go

This guide helps users migrate their existing Conky configurations to Conky-Go.

## Overview

Conky-Go is designed for 100% compatibility with existing Conky configurations. You should be able to run your existing `.conkyrc` or Lua configuration files without modification. However, understanding the differences helps with troubleshooting and optimization.

## Quick Start

If you have an existing Conky configuration, try running it directly:

```bash
# Run with your existing config
./conky-go -c ~/.conkyrc

# Or with a Lua configuration
./conky-go -c ~/.config/conky/conky.conf
```

## Configuration Formats

Conky-Go supports both configuration formats:

### Legacy Format (`.conkyrc`)

The traditional text-based format used in older Conky versions:

```ini
# Window settings
background no
own_window yes
own_window_type desktop
own_window_transparent yes
own_window_hints undecorated,below,sticky,skip_taskbar,skip_pager

# Display settings
double_buffer yes
update_interval 1.0
font DejaVu Sans Mono:size=10

# Position
alignment top_right
gap_x 10
gap_y 40

# Size
minimum_size 200 100
maximum_width 250

# Colors
default_color white
color0 gray

# Content
TEXT
${color0}CPU Usage:$color ${cpu}%
${color0}Memory:$color ${mem}/${memmax}
${color0}Uptime:$color ${uptime}
```

### Modern Lua Format

The Lua-based format used in Conky 1.10+:

```lua
conky.config = {
    background = false,
    own_window = true,
    own_window_type = 'desktop',
    own_window_transparent = true,
    own_window_hints = 'undecorated,below,sticky,skip_taskbar,skip_pager',
    
    double_buffer = true,
    update_interval = 1.0,
    font = 'DejaVu Sans Mono:size=10',
    
    alignment = 'top_right',
    gap_x = 10,
    gap_y = 40,
    
    minimum_width = 200,
    minimum_height = 100,
    maximum_width = 250,
    
    default_color = 'white',
    color0 = 'gray',
}

conky.text = [[
${color0}CPU Usage:$color ${cpu}%
${color0}Memory:$color ${mem}/${memmax}
${color0}Uptime:$color ${uptime}
]]
```

## Converting Legacy to Lua

Conky-Go includes a migration tool to convert legacy configurations to Lua format:

```bash
# Convert a legacy config to Lua format
./conky-go --convert ~/.conkyrc > ~/.config/conky/conky.conf
```

### Manual Conversion Guide

| Legacy Syntax | Lua Equivalent |
|---------------|----------------|
| `key value` | `key = value,` |
| `key yes` | `key = true,` |
| `key no` | `key = false,` |
| `TEXT` section | `conky.text = [[...]]` |

**Example Conversion:**

Legacy:
```ini
background yes
update_interval 2.0
font monospace:size=12
```

Lua:
```lua
conky.config = {
    background = true,
    update_interval = 2.0,
    font = 'monospace:size=12',
}
```

## Supported Variables

Conky-Go supports the most commonly used Conky variables:

### System Information

| Variable | Description | Example Output |
|----------|-------------|----------------|
| `${cpu}` | Total CPU usage percent | `45` |
| `${cpu cpuN}` | CPU core N usage | `${cpu cpu0}` |
| `${mem}` | Used memory (human) | `4.2G` |
| `${memmax}` | Total memory (human) | `16G` |
| `${memperc}` | Memory usage percent | `26` |
| `${swap}` | Used swap (human) | `512M` |
| `${swapmax}` | Total swap (human) | `8G` |
| `${swapperc}` | Swap usage percent | `6` |
| `${uptime}` | System uptime | `2d 5h 23m 45s` |
| `${uptime_short}` | Short uptime format | `2d 5h 23m` |

### Filesystem

| Variable | Description |
|----------|-------------|
| `${fs_used /}` | Used space on mount |
| `${fs_free /}` | Free space on mount |
| `${fs_size /}` | Total size of mount |
| `${fs_used_perc /}` | Usage percent |

### Disk I/O

| Variable | Description | Example Output |
|----------|-------------|----------------|
| `${diskio}` | Total I/O speed (all devices) | `4.5MiB/s` |
| `${diskio sda}` | Total I/O speed for device | `1.5MiB/s` |
| `${diskio_read}` | Total read speed (all devices) | `3.0MiB/s` |
| `${diskio_read sda}` | Read speed for device | `1.0MiB/s` |
| `${diskio_write}` | Total write speed (all devices) | `1.5MiB/s` |
| `${diskio_write sda}` | Write speed for device | `512.0KiB/s` |

### Network

| Variable | Description | Example |
|----------|-------------|---------|
| `${downspeed eth0}` | Download speed | `100.0KiB/s` |
| `${upspeed eth0}` | Upload speed | `50.0KiB/s` |
| `${totaldown eth0}` | Total downloaded | `1.5GiB` |
| `${totalup eth0}` | Total uploaded | `500.0MiB` |
| `${addr eth0}` | IPv4 address of interface | `192.168.1.100` |
| `${addrs eth0}` | All IP addresses of interface | `192.168.1.100 fe80::1` |
| `${gw_ip}` | Default gateway IP address | `192.168.1.1` |
| `${gw_iface}` | Default gateway interface | `eth0` |
| `${nameserver}` | First DNS nameserver | `8.8.8.8` |
| `${nameserver 1}` | DNS nameserver at index | `8.8.4.4` |

### Hardware

| Variable | Description | Example |
|----------|-------------|---------|
| `${hwmon 0 temp 1}` | Temperature sensor | `55` |
| `${cpu_count}` | Number of CPU cores | `4` |
| `${battery}` | Battery status and level | `Discharging 85%` |
| `${battery_percent}` | Battery percentage | `85` |
| `${battery_short}` | Short battery status | `D 85%` |
| `${battery_bar}` | Battery level bar | `########--` |
| `${battery_time}` | Battery time remaining | `2:30` |

### Processes

| Variable | Description | Example |
|----------|-------------|---------|
| `${processes}` | Total processes | `150` |
| `${running_processes}` | Running process count | `5` |
| `${threads}` | Total threads | `500` |
| `${top name 1}` | Top CPU process name | `firefox` |
| `${top pid 1}` | Top CPU process PID | `1234` |
| `${top cpu 1}` | Top process CPU % | `25.5` |
| `${top mem 1}` | Top process memory % | `10.2` |
| `${top_mem name 1}` | Top memory process name | `vscode` |
| `${top_mem mem 1}` | Top memory process % | `12.0` |

### Command Execution

| Variable | Description | Example |
|----------|-------------|---------|
| `${exec command}` | Execute command | `${exec date +%H:%M}` |
| `${execp command}` | Execute command (parsed) | `${execp echo hello}` |
| `${execi interval command}` | Execute command with caching | `${execi 60 sensors \| grep temp}` |
| `${execpi interval command}` | Cached execution (parsed) | `${execpi 30 echo ${cpu}%}` |

### Formatting

| Variable | Description |
|----------|-------------|
| `${color}` | Reset to default color |
| `${color red}` | Set text color |
| `${color0}` - `${color9}` | Use predefined color |
| `${font}` | Reset to default font |
| `${font name:size=N}` | Set font |
| `${alignr}` | Right align text |
| `${alignc}` | Center align text |
| `${voffset N}` | Vertical offset |
| `${offset N}` | Horizontal offset |
| `${goto N}` | Go to pixel position |
| `${tab}` | Tab character |
| `${hr N}` | Horizontal rule |

### Environment

| Variable | Description | Example |
|----------|-------------|---------|
| `${user_name}` | Current username | `john` |
| `${desktop_name}` | Desktop environment | `GNOME` |
| `${uid}` | User ID | `1000` |
| `${gid}` | Group ID | `1000` |

## Lua Scripting

Conky-Go supports Lua scripting with some considerations:

### Supported Functions

```lua
-- Parse Conky variables in a string
conky_parse("${cpu}%")  -- Returns "45%"

-- Main drawing hook (called each update)
function conky_main()
    return "Hello World"
end

-- Startup hook (called once)
function conky_start()
    print("Conky started")
end
```

### Cairo Drawing

Cairo drawing functions are supported for custom graphics:

```lua
require 'cairo'

function conky_main()
    if conky_window == nil then return end
    
    local cs = cairo_xlib_surface_create(conky_window.display,
        conky_window.drawable, conky_window.visual,
        conky_window.width, conky_window.height)
    local cr = cairo_create(cs)
    
    -- Draw a rectangle
    cairo_set_source_rgba(cr, 1, 0, 0, 0.5)
    cairo_rectangle(cr, 10, 10, 100, 50)
    cairo_fill(cr)
    
    cairo_destroy(cr)
    cairo_surface_destroy(cs)
end
```

### Clipping Limitations

**Important:** Cairo clipping functions (`cairo_clip`, `cairo_clip_preserve`, `cairo_reset_clip`) are implemented for API compatibility, but clipping is **not currently enforced** during drawing operations. This means:

- Calling `cairo_clip()` will record the clip region without errors
- Subsequent drawing operations will **not be restricted** to the clip area
- Scripts using clipping will execute successfully but may produce incorrect visual output

If your Conky scripts rely on clipping for complex drawings or masking effects, the visual results may differ from the original Conky behavior. This is a known limitation of the Ebiten-based rendering engine.

### Resource Limits

Conky-Go enforces resource limits on Lua scripts for security:

- **CPU**: 10 million instructions per execution
- **Memory**: 50 MB allocation limit

Scripts exceeding these limits will be terminated safely.

## Known Differences

### Golua vs Standard Lua

Conky-Go uses [Golua](https://github.com/arnodel/golua) instead of standard Lua. This is a pure Go Lua 5.4 implementation with:

- **Identical syntax** to standard Lua 5.4
- **Built-in sandboxing** for security
- **Slightly different error messages**

Most scripts work without modification, but some C-based Lua modules may not be available.

### Rendering Engine

Conky-Go uses [Ebiten](https://github.com/hajimehoshi/ebiten) instead of Cairo for rendering:

- **Cairo functions** are translated to Ebiten equivalents
- **Most drawing operations** work identically
- **Some advanced Cairo features** may have slight visual differences

### Font Handling

Font specifications use the same format as Conky:

```
FontName:size=N:weight=bold:style=italic
```

Ensure fonts are installed on your system. Common recommendations:
- DejaVu Sans Mono
- Liberation Mono
- Ubuntu Mono

## Troubleshooting

### Configuration Not Found

```bash
Error: Configuration file not found: /path/to/config
```

Ensure the file exists and has correct permissions:
```bash
ls -la ~/.conkyrc
chmod 644 ~/.conkyrc
```

### Unknown Variable Warning

```
Warning: unknown variable 'foo' in template
```

The variable may be:
1. Misspelled (check the variable reference)
2. Not yet implemented in Conky-Go
3. A custom Lua function that needs to be defined

### Lua Script Errors

```
Error: Lua execution error: exceeded CPU limit
```

Your script may have an infinite loop or be too complex. Consider:
1. Simplifying the script logic
2. Reducing update frequency
3. Moving complex calculations outside the main loop

### Display Issues

If the window doesn't appear correctly:

1. **Check window hints**: `own_window_hints = 'undecorated,below,sticky'`
2. **Try different window types**: `own_window_type = 'normal'` for debugging
3. **Verify X11 dependencies**: `libx11-dev`, `libxext-dev`, etc.

## Getting Help

If you encounter issues migrating:

1. **Check the GitHub Issues**: Your problem may already be reported
2. **Include your configuration**: Attach your config file when reporting issues
3. **Note the Conky version**: If migrating from a specific Conky version

## Compatibility Matrix

| Feature | Status | Notes |
|---------|--------|-------|
| Legacy .conkyrc format | ✅ Supported | Full parsing support |
| Lua configuration | ✅ Supported | conky.config + conky.text |
| System variables | ✅ Supported | 200+ variables implemented |
| Cairo drawing | ✅ Supported | Translated to Ebiten |
| Lua scripting | ✅ Supported | Via Golua |
| Network variables | ✅ Supported | /proc/net/dev parsing |
| Hardware sensors | ✅ Supported | hwmon integration |
| Battery monitoring | ✅ Supported | power_supply sysfs |
| Audio integration | ✅ Supported | ALSA via /proc/asound |
| X11 window hints | ✅ Supported | Desktop integration |
| Wayland support | 🔄 Planned | Future release |
| Windows support | ✅ Supported | Phase 7 - WMI/PDH APIs |
| macOS support | ✅ Supported | Phase 7 - sysctl/IOKit |
| Remote SSH monitoring | ✅ Supported | Linux/macOS targets |
