//go:build darwin
// +build darwin

package platform

// NewPlatform creates the appropriate Platform implementation for macOS.
func NewPlatform() (Platform, error) {
	return NewDarwinPlatform(), nil
}
