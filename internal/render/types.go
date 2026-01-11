// Package render provides Ebiten-based rendering capabilities for conky-go.
package render

import (
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

// TextLine represents a line of text to be rendered.
type TextLine struct {
	Text  string
	X     float64
	Y     float64
	Color color.RGBA
}

// DataProvider is an interface for providing system data to the renderer.
type DataProvider interface {
	// Update refreshes the system data.
	Update() error
}
