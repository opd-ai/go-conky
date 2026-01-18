# Transparency and Window Hints Guide

This document covers window transparency, compositor requirements, and window manager integration for go-conky.

## Overview

go-conky supports various transparency modes and window hints to integrate seamlessly with desktop environments. However, proper transparency requires a running compositor on X11 systems, and not all window hints work identically across different window managers.

## Transparency Modes

### True ARGB Transparency

True transparency uses a 32-bit ARGB visual to create genuinely transparent windows that blend with the desktop background and other windows.

**Configuration:**
```lua
conky.config = {
    own_window = true,
    own_window_transparent = true,
    own_window_argb_visual = true,
    own_window_argb_value = 200,  -- 0-255, where 0 = fully transparent
}
```

**Requirements:**
- A running compositor (see [Compositor Requirements](#compositor-requirements))
- X11 display server (not Wayland native)
- Graphics drivers with compositing support

### Pseudo-Transparency

Pseudo-transparency simulates transparency by capturing a screenshot of the desktop at the window's position and using it as the background. This works without a compositor but has limitations.

**Configuration:**
```lua
conky.config = {
    own_window = true,
    own_window_transparent = false,
    background_mode = 'pseudo',
}
```

**Limitations:**
- Background doesn't update when desktop changes
- Moving the window requires a refresh
- Doesn't show windows behind conky

### No Background (Fully Transparent)

When using `background_mode = 'none'`, the window has no background at all.

**Configuration:**
```lua
conky.config = {
    own_window = true,
    own_window_transparent = true,
    background_mode = 'none',
}
```

### Solid Background

A simple opaque or semi-transparent solid color background.

**Configuration:**
```lua
conky.config = {
    own_window = true,
    own_window_transparent = false,
    own_window_colour = '000000',  -- Black
    own_window_argb_visual = true,
    own_window_argb_value = 180,  -- Semi-transparent
}
```

## Compositor Requirements

### What is a Compositor?

A compositor is a window manager component that manages window rendering, enabling effects like transparency, shadows, and animations. On X11, compositing is an optional feature that must be enabled separately.

### Detecting Compositor

go-conky attempts to detect whether a compositor is running and will log a warning if ARGB transparency is enabled without an active compositor.

**Common Compositors:**
- **Picom** (formerly Compton) - Standalone X11 compositor
- **Compiz** - Standalone compositor with window management
- **Mutter** - GNOME's compositor (built into GNOME Shell)
- **KWin** - KDE's compositor (built into Plasma)
- **Xfwm4** - XFCE's compositor (built into Xfwm4)
- **Marco** - MATE's compositor (built into Marco)

### Installing and Running a Compositor

**For standalone window managers (i3, bspwm, Openbox, etc.):**

```bash
# Install picom
sudo apt install picom  # Debian/Ubuntu
sudo pacman -S picom    # Arch Linux
sudo dnf install picom  # Fedora

# Run picom
picom &

# Or with specific settings for conky
picom --backend glx --vsync &
```

**Enable compositor in desktop environments:**

| Desktop Environment | How to Enable Compositor |
|---------------------|-------------------------|
| GNOME | Always enabled (Mutter) |
| KDE Plasma | System Settings → Display → Compositor |
| XFCE | Window Manager Tweaks → Compositor |
| MATE | Control Center → Windows → Enable compositing |
| Cinnamon | Always enabled (Muffin) |

### Compositor Configuration for Conky

**Picom configuration (`~/.config/picom/picom.conf`):**

```ini
# Enable transparency support
backend = "glx"

# Exclude conky from shadows (optional)
shadow-exclude = [
    "class_g = 'Conky'",
    "class_g = 'conky-go'"
]

# Don't apply dim or fade to conky
focus-exclude = [
    "class_g = 'Conky'",
    "class_g = 'conky-go'"
]
```

## Window Hints

### Supported Window Hints

go-conky supports the following window hints:

| Hint | Description | Support |
|------|-------------|---------|
| `undecorated` | Remove window decorations (title bar, borders) | ✅ Full |
| `below` | Keep window below others | ⚠️ Partial |
| `above` | Keep window above others (floating) | ✅ Full |
| `sticky` | Show window on all desktops/workspaces | ⚠️ Partial |
| `skip_taskbar` | Don't show in taskbar | ⚠️ WM-dependent |
| `skip_pager` | Don't show in pager/workspace switcher | ⚠️ WM-dependent |

**Configuration:**
```lua
conky.config = {
    own_window_hints = 'undecorated,below,sticky,skip_taskbar,skip_pager',
}
```

### Known Limitations

#### BUG-001: Window Hints Not Fully Enforced

**Issue:** Some window hints may not work correctly on all window managers due to how Ebiten (go-conky's rendering engine) interacts with different window managers.

**Affected Hints:**
- `below` - May not work on all window managers
- `sticky` - May require additional window manager configuration
- `skip_taskbar` / `skip_pager` - Window manager dependent

**Affected Window Managers:**
- **XFCE4**: `below` hint may be ignored
- **i3**: `below` has no effect (tiling window manager)
- **Openbox**: May require manual compositor configuration

**Workarounds:**

1. **Use `own_window_type = 'desktop'`** - This window type inherently stays below other windows:
   ```lua
   conky.config = {
       own_window_type = 'desktop',
   }
   ```

2. **Use window manager rules** - Configure your window manager to apply rules to the conky-go window:

   **For Openbox (`~/.config/openbox/rc.xml`):**
   ```xml
   <application name="conky-go">
       <layer>below</layer>
       <decor>no</decor>
       <skip_taskbar>yes</skip_taskbar>
       <skip_pager>yes</skip_pager>
   </application>
   ```

   **For i3 (`~/.config/i3/config`):**
   ```
   for_window [class="conky-go"] floating enable
   for_window [class="conky-go"] sticky enable
   for_window [class="conky-go"] move position 10 40
   ```

   **For bspwm (`~/.config/bspwm/bspwmrc`):**
   ```bash
   bspc rule -a conky-go state=floating layer=below sticky=on
   ```

3. **Use wmctrl for dynamic control:**
   ```bash
   # Keep conky-go below after startup
   sleep 1 && wmctrl -r "conky-go" -b add,below &
   ```

### Window Types

The `own_window_type` setting affects how window managers treat the conky window:

| Type | Description | Best For |
|------|-------------|----------|
| `normal` | Regular application window | Debugging, testing |
| `desktop` | Desktop widget/icons layer | Most use cases |
| `dock` | Panel/dock behavior | Status bars |
| `panel` | Panel behavior (similar to dock) | Status panels |
| `override` | Override redirect (bypass WM) | Advanced use |

**Recommended settings for desktop widget:**
```lua
conky.config = {
    own_window = true,
    own_window_type = 'desktop',
    own_window_transparent = true,
    own_window_hints = 'undecorated,skip_taskbar,skip_pager',
}
```

## Troubleshooting

### Transparency Not Working

**Symptom:** Window appears solid instead of transparent.

**Solutions:**
1. **Check compositor is running:**
   ```bash
   # Check for common compositors
   pgrep -x picom || pgrep -x compton || pgrep -x compiz
   ```

2. **Start a compositor:**
   ```bash
   picom --backend glx &
   ```

3. **Verify ARGB settings:**
   ```lua
   conky.config = {
       own_window_argb_visual = true,
       own_window_argb_value = 200,
   }
   ```

4. **Check graphics drivers:**
   ```bash
   glxinfo | grep "OpenGL renderer"
   ```

### Visual Artifacts on Background

**Symptom:** Background appears glitchy or shows artifacts.

**Solutions:**
1. **Try different backends with picom:**
   ```bash
   picom --backend glx &   # OpenGL backend
   picom --backend xrender &  # X Render backend
   ```

2. **Use pseudo-transparency as fallback:**
   ```lua
   conky.config = {
       own_window_transparent = false,
       background_mode = 'pseudo',
   }
   ```

3. **Use solid semi-transparent background:**
   ```lua
   conky.config = {
       own_window_transparent = false,
       own_window_colour = '1a1a1a',
       own_window_argb_visual = true,
       own_window_argb_value = 200,
   }
   ```

### Window Not Staying Below Other Windows

**Symptom:** Conky window appears above other windows.

**Solutions:**
1. **Use desktop window type:**
   ```lua
   conky.config = {
       own_window_type = 'desktop',
   }
   ```

2. **Add window manager rules** (see [Workarounds](#workarounds) above)

3. **Use wmctrl:**
   ```bash
   wmctrl -r "conky-go" -b add,below
   ```

### Window Not Appearing on All Desktops

**Symptom:** Conky only visible on one workspace.

**Solutions:**
1. **Add sticky hint:**
   ```lua
   conky.config = {
       own_window_hints = 'undecorated,below,sticky',
   }
   ```

2. **Use window manager rules for sticky behavior**

3. **For some window managers, sticky works better with desktop type:**
   ```lua
   conky.config = {
       own_window_type = 'desktop',
   }
   ```

### Conky Appearing in Taskbar

**Symptom:** Conky window shows up in taskbar/dock.

**Solutions:**
1. **Add skip_taskbar hint:**
   ```lua
   conky.config = {
       own_window_hints = 'undecorated,below,sticky,skip_taskbar',
   }
   ```

2. **Use desktop window type (usually hides from taskbar):**
   ```lua
   conky.config = {
       own_window_type = 'desktop',
   }
   ```

## Platform-Specific Notes

### Linux/X11

- Full transparency support with compositor
- All window hints supported (with WM limitations)
- Pseudo-transparency fallback available

### Linux/Wayland

- Currently runs via XWayland
- Transparency support depends on Wayland compositor
- Some window hints may not work under XWayland

### macOS

- Native transparency support (compositor always enabled)
- Window hints may behave differently
- Some hints not applicable

### Windows

- DWM (Desktop Window Manager) always provides compositing
- Transparency works without additional setup
- Window hints may not apply

## Compositor Compatibility Matrix

| Desktop/WM | Compositor | Transparency | Below Hint | Sticky |
|------------|------------|--------------|------------|--------|
| GNOME | Mutter | ✅ | ✅ | ✅ |
| KDE Plasma | KWin | ✅ | ✅ | ✅ |
| XFCE | Xfwm4 | ✅ | ⚠️ | ⚠️ |
| MATE | Marco | ✅ | ✅ | ✅ |
| Cinnamon | Muffin | ✅ | ✅ | ✅ |
| i3 | Picom | ✅ | N/A | ✅ |
| bspwm | Picom | ✅ | ✅ | ✅ |
| Openbox | Picom | ✅ | ⚠️ | ⚠️ |
| Sway | Built-in | ✅ | ⚠️ | ⚠️ |

**Legend:**
- ✅ = Works correctly
- ⚠️ = May require workarounds
- N/A = Not applicable (e.g., tiling WMs)

## Best Practices

1. **Start with desktop window type** - Most compatible for system monitors
2. **Enable ARGB for transparency** - Better visual quality than pseudo-transparency  
3. **Install a compositor** - Required for true transparency on X11
4. **Add window manager rules** - More reliable than window hints for some WMs
5. **Use pseudo-transparency as fallback** - Works without compositor
6. **Test on target system** - Window manager behavior varies

## Related Documentation

- [Cross-Platform Guide](cross-platform.md) - Platform-specific setup
- [Migration Guide](migration.md) - Migrating from original Conky
- [API Reference](api.md) - Configuration options
