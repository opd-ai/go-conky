package conky

import (
	"fmt"
	"image/color"

	"github.com/opd-ai/go-conky/internal/render"
)

// Rendering defaults for the game runner.
const (
	defaultBackgroundAlpha = 200  // Semi-transparent background
	defaultTextStartY      = 20.0 // Initial Y position for text
	defaultLineHeight      = 18.0 // Vertical spacing between lines
	defaultTextStartX      = 10.0 // Initial X position for text
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
	// Background color uses a hardcoded semi-transparent black because
	// config.ColorConfig currently has no dedicated background color field.
	// Colors.Default is specifically "the default text color" per its documentation,
	// so using it for background would be semantically incorrect and confusing.
	// TODO: Consider adding a dedicated background color field to ColorConfig.
	bgColor := color.RGBA{R: 0, G: 0, B: 0, A: defaultBackgroundAlpha}
	// Default text color is white if not specified in config
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
		y := defaultTextStartY
		for _, text := range textLines {
			lines = append(lines, render.TextLine{
				Text:  text,
				X:     defaultTextStartX,
				Y:     y,
				Color: textColor,
			})
			y += defaultLineHeight
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
	c.mu.Lock()
	c.gameRunner = gr
	c.mu.Unlock()
	gr.run(c)
}
