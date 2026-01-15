//go:build windows
// +build windows

package platform

// NewPlatform creates the appropriate Platform implementation for Windows.
func NewPlatform() (Platform, error) {
	return NewWindowsPlatform(), nil
}
