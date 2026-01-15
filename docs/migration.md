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
# Convert a legacy config to Lua (future feature)
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

### Network

| Variable | Description |
|----------|-------------|
| `${downspeed eth0}` | Download speed |
| `${upspeed eth0}` | Upload speed |
| `${totaldown eth0}` | Total downloaded |
| `${totalup eth0}` | Total uploaded |

### Hardware

| Variable | Description |
|----------|-------------|
| `${hwmon 0 temp 1}` | Temperature sensor |
| `${battery}` | Battery level |
| `${battery_time}` | Battery time remaining |
| `${acpiacadapter}` | AC adapter status |

### Processes

| Variable | Description |
|----------|-------------|
| `${processes}` | Total processes |
| `${running_processes}` | Running process count |
| `${top name 1}` | Top CPU process name |
| `${top cpu 1}` | Top process CPU % |
| `${top mem 1}` | Top process memory % |

### Formatting

| Variable | Description |
|----------|-------------|
| `${color}` | Reset to default color |
| `${color red}` | Set text color |
| `${color0}` - `${color9}` | Use predefined color |
| `${font}` | Reset to default font |
| `${font name:size=N}` | Set font |

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
