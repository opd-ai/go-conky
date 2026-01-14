//go:build !darwin
// +build !darwin

package platform

// NewDarwinPlatform is a stub for non-Darwin platforms.
// It will never be called due to the factory logic, but needs to exist for compilation.
func NewDarwinPlatform() Platform {
	panic("NewDarwinPlatform called on non-Darwin platform")
}
