//go:build android
// +build android

package platform

// NewPlatform creates the appropriate Platform implementation for Android.
func NewPlatform() (Platform, error) {
	return NewAndroidPlatform(), nil
}
