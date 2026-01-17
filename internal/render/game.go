// Package render provides Ebiten-based rendering capabilities for conky-go.
// It implements the core rendering engine using Ebiten v2 for cross-platform
// 2D graphics with support for text rendering and widget display.
package render

import (
	"context"
	"errors"
	"fmt"
	"image/color"
	"os"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// ErrGameTerminated is returned when the game loop is terminated via context cancellation.
var ErrGameTerminated = errors.New("game terminated")

// ErrorHandler is a function type for handling errors during game updates.
type ErrorHandler func(err error)

// DefaultErrorHandler writes errors to stderr.
func DefaultErrorHandler(err error) {
	fmt.Fprintf(os.Stderr, "update error: %v\n", err)
}

// TextRendererInterface defines the interface for text rendering.
// This allows for mocking in tests.
type TextRendererInterface interface {
	DrawText(screen *ebiten.Image, textStr string, x, y float64, clr color.RGBA)
	MeasureText(textStr string) (width, height float64)
	LineHeight() float64
	SetFontSize(size float64)
	FontSize() float64
}

// Game implements ebiten.Game interface and handles rendering.
type Game struct {
	config       Config
	textRenderer TextRendererInterface
	dataProvider DataProvider
	errorHandler ErrorHandler
	lastUpdate   time.Time
	lines        []TextLine
	mu           sync.RWMutex
	running      bool
	ctx          context.Context
}

// NewGame creates a new Game instance with the provided configuration.
func NewGame(config Config) *Game {
	return &Game{
		config:       config,
		textRenderer: NewTextRenderer(),
		errorHandler: DefaultErrorHandler,
		lastUpdate:   time.Now(),
		lines:        make([]TextLine, 0),
	}
}

// NewGameWithRenderer creates a new Game instance with a custom text renderer.
// This is useful for testing.
func NewGameWithRenderer(config Config, renderer TextRendererInterface) *Game {
	return &Game{
		config:       config,
		textRenderer: renderer,
		errorHandler: DefaultErrorHandler,
		lastUpdate:   time.Now(),
		lines:        make([]TextLine, 0),
	}
}

// SetErrorHandler sets a custom error handler for update errors.
// If nil is passed, errors will be silently ignored.
func (g *Game) SetErrorHandler(handler ErrorHandler) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.errorHandler = handler
}

// SetDataProvider sets the data provider for system monitoring updates.
func (g *Game) SetDataProvider(dp DataProvider) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.dataProvider = dp
}

// SetContext sets a context for the game loop. When the context is cancelled,
// the game loop will terminate gracefully.
func (g *Game) SetContext(ctx context.Context) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.ctx = ctx
}

// SetLines sets the text lines to be rendered.
func (g *Game) SetLines(lines []TextLine) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.lines = make([]TextLine, len(lines))
	copy(g.lines, lines)
}

// AddLine adds a single text line to be rendered.
func (g *Game) AddLine(line TextLine) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.lines = append(g.lines, line)
}

// ClearLines removes all text lines.
func (g *Game) ClearLines() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.lines = g.lines[:0]
}

// Update implements ebiten.Game.Update.
// It is called every tick (typically 60 times per second).
func (g *Game) Update() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Check for context cancellation (used for programmatic shutdown)
	if g.ctx != nil {
		select {
		case <-g.ctx.Done():
			return ErrGameTerminated
		default:
		}
	}

	// Update system data at configured intervals
	if g.dataProvider != nil && time.Since(g.lastUpdate) >= g.config.UpdateInterval {
		if err := g.dataProvider.Update(); err != nil {
			// Use error handler if configured
			if g.errorHandler != nil {
				g.errorHandler(err)
			}
		}
		g.lastUpdate = time.Now()
	}

	return nil
}

// Draw implements ebiten.Game.Draw.
// It is called every frame to render the screen.
func (g *Game) Draw(screen *ebiten.Image) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// Clear screen with background color
	screen.Fill(g.config.BackgroundColor)

	// Render all text lines
	for _, line := range g.lines {
		g.textRenderer.DrawText(screen, line.Text, line.X, line.Y, line.Color)
	}
}

// Layout implements ebiten.Game.Layout.
// It returns the game's logical screen size.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return g.config.Width, g.config.Height
}

// Config returns the current configuration.
func (g *Game) Config() Config {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.config
}

// SetConfig updates the game configuration in-place.
// This allows hot-reloading of configuration without stopping the game loop.
// Note: Window size changes may not take effect until the next window resize.
func (g *Game) SetConfig(config Config) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.config = config
}

// Run starts the Ebiten game loop.
// This function blocks until the window is closed.
func (g *Game) Run() error {
	ebiten.SetWindowSize(g.config.Width, g.config.Height)
	ebiten.SetWindowTitle(g.config.Title)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	g.mu.Lock()
	g.running = true
	g.mu.Unlock()

	err := ebiten.RunGame(g)

	g.mu.Lock()
	g.running = false
	g.mu.Unlock()

	return err
}

// IsRunning returns whether the game loop is currently running.
func (g *Game) IsRunning() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.running
}
