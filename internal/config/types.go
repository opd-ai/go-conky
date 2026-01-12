// Package config provides configuration data structures for conky-go.
// It defines types for both legacy .conkyrc and modern Lua configuration formats,
// enabling parsing and validation of Conky configurations.
package config

import (
	"fmt"
	"image/color"
	"time"
)

// Config represents the complete conky-go configuration.
// It aggregates window, display, and text template settings.
type Config struct {
	// Window contains window-related configuration options.
	Window WindowConfig
	// Display contains display and rendering settings.
	Display DisplayConfig
	// Text contains the text template and formatting settings.
	Text TextConfig
	// Colors contains color definitions.
	Colors ColorConfig
}

// WindowConfig holds window-related configuration options.
type WindowConfig struct {
	// OwnWindow determines if conky creates its own window.
	OwnWindow bool
	// Type specifies the window type (normal, desktop, dock, panel, override).
	Type WindowType
	// Transparent enables window transparency.
	Transparent bool
	// Hints contains window manager hints (undecorated, below, sticky, etc.).
	Hints []WindowHint
	// Width is the minimum window width in pixels.
	Width int
	// Height is the minimum window height in pixels.
	Height int
	// X is the horizontal window position offset.
	X int
	// Y is the vertical window position offset.
	Y int
	// Alignment specifies window alignment on screen.
	Alignment Alignment
}

// DisplayConfig holds display and rendering settings.
type DisplayConfig struct {
	// Background determines if conky runs in background mode.
	Background bool
	// DoubleBuffer enables double buffering to reduce flicker.
	DoubleBuffer bool
	// UpdateInterval is the time between display updates.
	UpdateInterval time.Duration
	// Font is the default font specification.
	Font string
	// FontSize is the default font size in points.
	FontSize float64
}

// TextConfig holds text template and formatting settings.
type TextConfig struct {
	// Template contains the text template lines.
	Template []string
}

// ColorConfig holds color definitions.
type ColorConfig struct {
	// Default is the default text color.
	Default color.RGBA
	// Color0 through Color9 are user-defined colors.
	Color0 color.RGBA
	Color1 color.RGBA
	Color2 color.RGBA
	Color3 color.RGBA
	Color4 color.RGBA
	Color5 color.RGBA
	Color6 color.RGBA
	Color7 color.RGBA
	Color8 color.RGBA
	Color9 color.RGBA
}

// WindowType represents the type of window to create.
type WindowType int

const (
	// WindowTypeNormal is a standard window.
	WindowTypeNormal WindowType = iota
	// WindowTypeDesktop is a desktop-level window.
	WindowTypeDesktop
	// WindowTypeDock is a dock/panel window.
	WindowTypeDock
	// WindowTypePanel is a panel window.
	WindowTypePanel
	// WindowTypeOverride uses override-redirect.
	WindowTypeOverride
)

// String returns the string representation of a WindowType.
func (wt WindowType) String() string {
	switch wt {
	case WindowTypeNormal:
		return "normal"
	case WindowTypeDesktop:
		return "desktop"
	case WindowTypeDock:
		return "dock"
	case WindowTypePanel:
		return "panel"
	case WindowTypeOverride:
		return "override"
	default:
		return "unknown"
	}
}

// ParseWindowType parses a string into a WindowType.
func ParseWindowType(s string) (WindowType, error) {
	switch s {
	case "normal":
		return WindowTypeNormal, nil
	case "desktop":
		return WindowTypeDesktop, nil
	case "dock":
		return WindowTypeDock, nil
	case "panel":
		return WindowTypePanel, nil
	case "override":
		return WindowTypeOverride, nil
	default:
		return WindowTypeNormal, fmt.Errorf("unknown window type: %s", s)
	}
}

// WindowHint represents a window manager hint.
type WindowHint int

const (
	// WindowHintUndecorated removes window decorations.
	WindowHintUndecorated WindowHint = iota
	// WindowHintBelow keeps the window below others.
	WindowHintBelow
	// WindowHintAbove keeps the window above others.
	WindowHintAbove
	// WindowHintSticky makes the window visible on all desktops.
	WindowHintSticky
	// WindowHintSkipTaskbar hides the window from the taskbar.
	WindowHintSkipTaskbar
	// WindowHintSkipPager hides the window from the pager.
	WindowHintSkipPager
)

// String returns the string representation of a WindowHint.
func (wh WindowHint) String() string {
	switch wh {
	case WindowHintUndecorated:
		return "undecorated"
	case WindowHintBelow:
		return "below"
	case WindowHintAbove:
		return "above"
	case WindowHintSticky:
		return "sticky"
	case WindowHintSkipTaskbar:
		return "skip_taskbar"
	case WindowHintSkipPager:
		return "skip_pager"
	default:
		return "unknown"
	}
}

// ParseWindowHint parses a string into a WindowHint.
func ParseWindowHint(s string) (WindowHint, error) {
	switch s {
	case "undecorated":
		return WindowHintUndecorated, nil
	case "below":
		return WindowHintBelow, nil
	case "above":
		return WindowHintAbove, nil
	case "sticky":
		return WindowHintSticky, nil
	case "skip_taskbar":
		return WindowHintSkipTaskbar, nil
	case "skip_pager":
		return WindowHintSkipPager, nil
	default:
		return WindowHintUndecorated, fmt.Errorf("unknown window hint: %s", s)
	}
}

// Alignment specifies window alignment on screen.
type Alignment int

const (
	// AlignmentTopLeft aligns to top-left corner.
	AlignmentTopLeft Alignment = iota
	// AlignmentTopMiddle aligns to top-center.
	AlignmentTopMiddle
	// AlignmentTopRight aligns to top-right corner.
	AlignmentTopRight
	// AlignmentMiddleLeft aligns to middle-left.
	AlignmentMiddleLeft
	// AlignmentMiddleMiddle aligns to center.
	AlignmentMiddleMiddle
	// AlignmentMiddleRight aligns to middle-right.
	AlignmentMiddleRight
	// AlignmentBottomLeft aligns to bottom-left corner.
	AlignmentBottomLeft
	// AlignmentBottomMiddle aligns to bottom-center.
	AlignmentBottomMiddle
	// AlignmentBottomRight aligns to bottom-right corner.
	AlignmentBottomRight
)

// String returns the string representation of an Alignment.
func (a Alignment) String() string {
	switch a {
	case AlignmentTopLeft:
		return "top_left"
	case AlignmentTopMiddle:
		return "top_middle"
	case AlignmentTopRight:
		return "top_right"
	case AlignmentMiddleLeft:
		return "middle_left"
	case AlignmentMiddleMiddle:
		return "middle_middle"
	case AlignmentMiddleRight:
		return "middle_right"
	case AlignmentBottomLeft:
		return "bottom_left"
	case AlignmentBottomMiddle:
		return "bottom_middle"
	case AlignmentBottomRight:
		return "bottom_right"
	default:
		return "unknown"
	}
}

// ParseAlignment parses a string into an Alignment.
func ParseAlignment(s string) (Alignment, error) {
	switch s {
	case "top_left", "tl":
		return AlignmentTopLeft, nil
	case "top_middle", "tm", "top_center", "tc":
		return AlignmentTopMiddle, nil
	case "top_right", "tr":
		return AlignmentTopRight, nil
	case "middle_left", "ml":
		return AlignmentMiddleLeft, nil
	case "middle_middle", "mm", "middle_center", "mc", "center", "c":
		return AlignmentMiddleMiddle, nil
	case "middle_right", "mr":
		return AlignmentMiddleRight, nil
	case "bottom_left", "bl":
		return AlignmentBottomLeft, nil
	case "bottom_middle", "bm", "bottom_center", "bc":
		return AlignmentBottomMiddle, nil
	case "bottom_right", "br":
		return AlignmentBottomRight, nil
	default:
		return AlignmentTopLeft, fmt.Errorf("unknown alignment: %s", s)
	}
}

// Validate checks if the Config has valid values.
func (c *Config) Validate() error {
	if c.Window.Width < 0 {
		return fmt.Errorf("window width must be non-negative, got %d", c.Window.Width)
	}
	if c.Window.Height < 0 {
		return fmt.Errorf("window height must be non-negative, got %d", c.Window.Height)
	}
	if c.Display.UpdateInterval < 0 {
		return fmt.Errorf("update interval must be non-negative, got %v", c.Display.UpdateInterval)
	}
	return nil
}
