// Package render provides Ebiten-based rendering capabilities for conky-go.
package render

import (
	"fmt"
	"image/color"
	"time"
)

// Config holds the rendering configuration options.
type Config struct {
	// Width is the window width in pixels.
	Width int
	// Height is the window height in pixels.
	Height int
	// Title is the window title.
	Title string
	// UpdateInterval is the time between system data updates.
	UpdateInterval time.Duration
	// BackgroundColor is the window background color.
	BackgroundColor color.RGBA
	// Transparent enables window transparency mode.
	// When enabled, the window background can be transparent if the compositor supports it.
	Transparent bool
	// ARGBVisual enables 32-bit ARGB visual for true transparency.
	// Requires a compositor (e.g., picom, compton) on Linux.
	// When true, enables Ebiten's screen transparency feature.
	ARGBVisual bool
	// ARGBValue sets the alpha value for ARGB transparency (0-255).
	// 0 is fully transparent, 255 is fully opaque.
	// Only effective when ARGBVisual is true.
	ARGBValue int
	// DrawBorders enables drawing a border around the content area.
	DrawBorders bool
	// DrawOutline enables drawing an outline (stroke) around text.
	DrawOutline bool
	// DrawShades enables drawing a drop shadow behind text.
	DrawShades bool
	// BorderWidth is the width of borders in pixels.
	BorderWidth int
	// BorderInnerMargin is the inner margin between border and content in pixels.
	BorderInnerMargin int
	// BorderOuterMargin is the outer margin between window edge and border in pixels.
	BorderOuterMargin int
	// StippledBorders enables stippled (dashed) border effect.
	StippledBorders bool
	// BorderColor is the color for borders. If zero value, uses a contrasting color.
	BorderColor color.RGBA
	// OutlineColor is the color for text outlines. If zero value, uses black.
	OutlineColor color.RGBA
	// ShadeColor is the color for text shadows. If zero value, uses dark gray.
	ShadeColor color.RGBA
}

// DefaultConfig returns a Config with sensible default values.
func DefaultConfig() Config {
	return Config{
		Width:             400,
		Height:            300,
		Title:             "conky-go",
		UpdateInterval:    time.Second,
		BackgroundColor:   color.RGBA{R: 0, G: 0, B: 0, A: 200},
		Transparent:       false,
		ARGBVisual:        false,
		ARGBValue:         255, // Fully opaque by default
		DrawBorders:       false,
		DrawOutline:       false,
		DrawShades:        false,
		BorderWidth:       1,
		BorderInnerMargin: 5,
		BorderOuterMargin: 5,
		StippledBorders:   false,
		BorderColor:       color.RGBA{R: 255, G: 255, B: 255, A: 255},
		OutlineColor:      color.RGBA{R: 0, G: 0, B: 0, A: 255},
		ShadeColor:        color.RGBA{R: 0, G: 0, B: 0, A: 128},
	}
}

// Validate checks if the Config has valid values.
// Returns an error if Width or Height are not positive.
func (c Config) Validate() error {
	if c.Width <= 0 {
		return fmt.Errorf("width must be positive, got %d", c.Width)
	}
	if c.Height <= 0 {
		return fmt.Errorf("height must be positive, got %d", c.Height)
	}
	return nil
}

// TextLine represents a line of text to be rendered.
type TextLine struct {
	// Text is the string content that will be rendered.
	Text string
	// X is the horizontal position of the text's origin, in pixels from the
	// left edge of the window (or drawing surface).
	X float64
	// Y is the vertical position of the text's baseline, in pixels from the
	// top edge of the window (or drawing surface).
	Y float64
	// Color is the text color in RGBA format. The alpha channel controls
	// the text's opacity if the renderer supports transparency.
	Color color.RGBA
}

// DataProvider is an interface for providing system data to the renderer.
type DataProvider interface {
	// Update refreshes the system data.
	Update() error
}
