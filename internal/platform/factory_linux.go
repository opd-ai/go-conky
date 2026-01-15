//go:build linux && !android
// +build linux,!android

package platform

// NewPlatform creates the appropriate Platform implementation for Linux.
func NewPlatform() (Platform, error) {
	return NewLinuxPlatform(), nil
}
