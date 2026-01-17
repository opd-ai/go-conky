// Package render provides background rendering capabilities for conky-go.
package render

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// BackgroundMode specifies how the window background is rendered.
type BackgroundMode int

const (
	// BackgroundModeSolid draws a solid background color.
	BackgroundModeSolid BackgroundMode = iota
	// BackgroundModeNone draws no background (fully transparent).
	BackgroundModeNone
)

// BackgroundRenderer is an interface for rendering window backgrounds.
// It allows for extensible background rendering strategies.
type BackgroundRenderer interface {
	// Draw renders the background to the screen.
	Draw(screen *ebiten.Image)
	// Mode returns the background mode.
	Mode() BackgroundMode
}

// SolidBackground renders a solid color background.
type SolidBackground struct {
	color     color.RGBA
	argbValue int  // Alpha value override (0-255) when ARGB is enabled
	argbOn    bool // Whether ARGB visual is enabled
}

// NewSolidBackground creates a new solid background renderer.
func NewSolidBackground(c color.RGBA) *SolidBackground {
	return &SolidBackground{
		color:     c,
		argbValue: 255,
		argbOn:    false,
	}
}

// WithARGB configures ARGB visual settings for the background.
// When enabled, the argbValue overrides the color's alpha channel.
func (sb *SolidBackground) WithARGB(enabled bool, value int) *SolidBackground {
	sb.argbOn = enabled
	// Clamp ARGB value to valid range
	if value < 0 {
		value = 0
	} else if value > 255 {
		value = 255
	}
	sb.argbValue = value
	return sb
}

// Draw renders the solid background to the screen.
func (sb *SolidBackground) Draw(screen *ebiten.Image) {
	c := sb.color
	if sb.argbOn {
		c.A = uint8(sb.argbValue)
	}
	screen.Fill(c)
}

// Mode returns BackgroundModeSolid.
func (sb *SolidBackground) Mode() BackgroundMode {
	return BackgroundModeSolid
}

// Color returns the background color.
func (sb *SolidBackground) Color() color.RGBA {
	return sb.color
}

// NoneBackground renders no background (fully transparent).
type NoneBackground struct{}

// NewNoneBackground creates a new none/transparent background renderer.
func NewNoneBackground() *NoneBackground {
	return &NoneBackground{}
}

// Draw clears the screen with fully transparent color.
func (nb *NoneBackground) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 0, G: 0, B: 0, A: 0})
}

// Mode returns BackgroundModeNone.
func (nb *NoneBackground) Mode() BackgroundMode {
	return BackgroundModeNone
}

// NewBackgroundRenderer creates a BackgroundRenderer based on the mode and color.
// For BackgroundModeNone, the color is ignored.
// For BackgroundModeSolid, a SolidBackground is created with the given color.
func NewBackgroundRenderer(mode BackgroundMode, bgColor color.RGBA, argbVisual bool, argbValue int) BackgroundRenderer {
	switch mode {
	case BackgroundModeNone:
		return NewNoneBackground()
	default:
		return NewSolidBackground(bgColor).WithARGB(argbVisual, argbValue)
	}
}
