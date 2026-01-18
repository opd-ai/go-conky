//go:build linux

// Package render provides compositor detection for X11 on Linux.
package render

import (
	"os"
	"os/exec"
	"strings"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

// CompositorStatus represents the detected compositor state.
type CompositorStatus int

const (
	// CompositorUnknown means we couldn't determine compositor status.
	CompositorUnknown CompositorStatus = iota
	// CompositorActive means a compositor is running (transparency will work).
	CompositorActive
	// CompositorInactive means no compositor detected (transparency may fail).
	CompositorInactive
)

// String returns a human-readable compositor status.
func (cs CompositorStatus) String() string {
	switch cs {
	case CompositorActive:
		return "active"
	case CompositorInactive:
		return "inactive"
	default:
		return "unknown"
	}
}

// DetectCompositor checks if an X11 compositor is currently running.
// It uses multiple detection methods for better compatibility:
// 1. Check for _NET_WM_CM_Sn atom (standard EWMH compositor hint)
// 2. Check for known compositor processes
//
// Returns CompositorActive if a compositor is detected, CompositorInactive if none,
// or CompositorUnknown if detection failed.
func DetectCompositor() CompositorStatus {
	// First, try the X11 atom-based detection (most reliable)
	if status := detectCompositorAtom(); status != CompositorUnknown {
		return status
	}

	// Fallback: check for known compositor processes
	return detectCompositorProcess()
}

// detectCompositorAtom checks for the _NET_WM_CM_Sn atom which is set by
// EWMH-compliant compositors. The "n" is the screen number (usually 0).
func detectCompositorAtom() CompositorStatus {
	// Connect to X server
	conn, err := xgb.NewConn()
	if err != nil {
		return CompositorUnknown
	}
	defer conn.Close()

	// Get setup info
	setup := xproto.Setup(conn)
	if len(setup.Roots) == 0 {
		return CompositorUnknown
	}

	// Build the atom name for screen 0 (most common)
	// The format is _NET_WM_CM_Sn where n is the screen number
	atomName := "_NET_WM_CM_S0"

	// Intern the atom (get or create)
	atomReply, err := xproto.InternAtom(conn, false, uint16(len(atomName)), atomName).Reply()
	if err != nil || atomReply == nil {
		return CompositorUnknown
	}

	// Check if any window owns this selection
	// If there's an owner, a compositor is running
	owner, err := xproto.GetSelectionOwner(conn, atomReply.Atom).Reply()
	if err != nil {
		return CompositorUnknown
	}

	if owner.Owner != xproto.WindowNone {
		return CompositorActive
	}

	return CompositorInactive
}

// detectCompositorProcess checks for known compositor process names.
// This is a fallback method when X11 atom detection fails.
func detectCompositorProcess() CompositorStatus {
	// Known compositor process names
	compositors := []string{
		"picom",
		"compton",
		"compiz",
		"mutter",
		"kwin",
		"kwin_x11",
		"kwin_wayland",
		"xfwm4",
		"marco",
		"muffin",
		"openbox", // Openbox can have compositing via --composite
	}

	for _, compositor := range compositors {
		// Use pgrep to check for process
		cmd := exec.Command("pgrep", "-x", compositor)
		if err := cmd.Run(); err == nil {
			return CompositorActive
		}
	}

	return CompositorInactive
}

// IsWayland checks if the current session is running on Wayland.
// Wayland compositors always provide compositing, so transparency works.
func IsWayland() bool {
	// Check XDG_SESSION_TYPE
	if sessionType := os.Getenv("XDG_SESSION_TYPE"); strings.ToLower(sessionType) == "wayland" {
		return true
	}

	// Check WAYLAND_DISPLAY
	if waylandDisplay := os.Getenv("WAYLAND_DISPLAY"); waylandDisplay != "" {
		return true
	}

	return false
}

// CheckTransparencySupport returns a warning message if transparency may not work,
// or an empty string if transparency should work fine.
// This is intended to be called at startup when ARGB transparency is enabled.
func CheckTransparencySupport(argbVisual bool, transparent bool) string {
	// If transparency is not requested, no warning needed
	if !argbVisual && !transparent {
		return ""
	}

	// Wayland always has compositing
	if IsWayland() {
		return ""
	}

	// Check for compositor on X11
	status := DetectCompositor()

	switch status {
	case CompositorActive:
		return ""
	case CompositorInactive:
		return "Warning: No compositor detected. ARGB transparency requires a compositor " +
			"(such as picom, compiz, or a desktop environment's built-in compositor). " +
			"Window may appear opaque or have visual artifacts. " +
			"See docs/transparency.md for setup instructions."
	default:
		return "Warning: Could not detect compositor status. ARGB transparency may not work " +
			"if no compositor is running. See docs/transparency.md for details."
	}
}
