//go:build !linux

// Package render provides stub window hint support for non-Linux platforms.
package render

// ApplyWindowHints is a no-op on non-Linux platforms.
// X11 EWMH hints for skip_taskbar and skip_pager are Linux-specific.
func ApplyWindowHints(skipTaskbar, skipPager bool) error {
	return nil
}

// CloseWindowHints is a no-op on non-Linux platforms.
func CloseWindowHints() {
}
