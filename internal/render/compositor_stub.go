//go:build !linux

// Package render provides compositor detection stubs for non-Linux platforms.
package render

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

// DetectCompositor returns CompositorActive on non-Linux platforms.
// Windows (DWM) and macOS always have compositing enabled.
func DetectCompositor() CompositorStatus {
	return CompositorActive
}

// IsWayland returns false on non-Linux platforms.
func IsWayland() bool {
	return false
}

// CheckTransparencySupport returns an empty string on non-Linux platforms
// because Windows (DWM) and macOS always have compositing enabled.
func CheckTransparencySupport(argbVisual, transparent bool) string {
	return ""
}
