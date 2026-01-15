//go:build !noebiten

package conky

import (
	"fmt"
	"image/color"

	"github.com/opd-ai/go-conky/internal/render"
)

// gameRunner provides the Ebiten game integration for rendering.
type gameRunner struct {
	game *render.Game
}

// newGameRunner creates a new game runner for the Ebiten rendering loop.
func newGameRunner() *gameRunner {
	return &gameRunner{}
}

// run creates and runs the Ebiten rendering loop.
// This method blocks until the window is closed or context is cancelled.
func (gr *gameRunner) run(c *conkyImpl) {
	// Get configuration values with fallbacks
	c.mu.RLock()
	width := c.cfg.Window.Width
	height := c.cfg.Window.Height
	title := c.opts.WindowTitle
	interval := c.cfg.Display.UpdateInterval
	bgColor := c.cfg.Colors.Default
	textLines := c.cfg.Text.Template
	textColor := c.cfg.Colors.Default
	ctx := c.ctx
	c.mu.RUnlock()

	// Apply defaults
	if width <= 0 {
		width = 400
	}
	if height <= 0 {
		height = 300
	}
	if title == "" {
		title = "conky-go"
	}
	if interval <= 0 {
		interval = defaultUpdateInterval
	}
	// Default background is semi-transparent black
	if bgColor == (color.RGBA{}) {
		bgColor = color.RGBA{R: 0, G: 0, B: 0, A: 200}
	}
	// Default text color is white
	if textColor == (color.RGBA{}) {
		textColor = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	}

	// Create render configuration
	renderConfig := render.Config{
		Width:           width,
		Height:          height,
		Title:           title,
		UpdateInterval:  interval,
		BackgroundColor: bgColor,
	}

	// Create the game instance
	gr.game = render.NewGame(renderConfig)
	gr.game.SetDataProvider(c.monitor)
	gr.game.SetContext(ctx)

	// Set up initial text lines from configuration template
	if len(textLines) > 0 {
		lines := make([]render.TextLine, 0, len(textLines))
		y := 20.0 // Start position
		for _, text := range textLines {
			lines = append(lines, render.TextLine{
				Text:  text,
				X:     10,
				Y:     y,
				Color: textColor,
			})
			y += 18 // Line height
		}
		gr.game.SetLines(lines)
	}

	// Run the Ebiten game loop (blocks until window close or context cancel)
	if err := gr.game.Run(); err != nil {
		// ErrGameTerminated is expected when context is cancelled
		if err != render.ErrGameTerminated {
			c.notifyError(fmt.Errorf("render loop error: %w", err))
		}
	}
}

// runRenderLoop creates and runs the Ebiten rendering loop.
func (c *conkyImpl) runRenderLoop() {
	gr := newGameRunner()
	gr.run(c)
}
