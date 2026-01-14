// +build !windows

package platform

// NewWindowsPlatform is a stub for non-Windows platforms.
func NewWindowsPlatform() Platform {
	// This should never be called on non-Windows platforms
	// The factory should prevent this
	panic("Windows platform called on non-Windows system")
}
