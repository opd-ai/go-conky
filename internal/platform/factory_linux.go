//go:build linux
// +build linux

package platform

// NewPlatform creates the appropriate Platform implementation for Linux.
func NewPlatform() (Platform, error) {
	return NewLinuxPlatform(), nil
}
