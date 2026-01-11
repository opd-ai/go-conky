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
}

// DefaultConfig returns a Config with sensible default values.
func DefaultConfig() Config {
	return Config{
		Width:           400,
		Height:          300,
		Title:           "conky-go",
		UpdateInterval:  time.Second,
		BackgroundColor: color.RGBA{R: 0, G: 0, B: 0, A: 200},
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
