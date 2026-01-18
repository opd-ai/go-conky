//go:build !linux

// Package render provides screen capture stubs for non-Linux platforms.
package render

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
)

// X11ScreenshotProvider is not available on non-Linux platforms.
// It always returns an error indicating the feature is unavailable.
func X11ScreenshotProvider(x, y, width, height int) (*ebiten.Image, error) {
	return nil, fmt.Errorf("X11 screenshot provider is only available on Linux")
}
