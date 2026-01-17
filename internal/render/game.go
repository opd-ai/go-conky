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
	"github.com/hajimehoshi/ebiten/v2/vector"
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

	// Render all text lines with inline widget support
	for _, line := range g.lines {
		g.drawLineWithWidgets(screen, line)
	}
}

// drawLineWithWidgets renders a text line, handling inline widget markers.
func (g *Game) drawLineWithWidgets(screen *ebiten.Image, line TextLine) {
	// Fast path: if no widget markers, just draw text
	if !ContainsWidgetMarker(line.Text) {
		g.textRenderer.DrawText(screen, line.Text, line.X, line.Y, line.Color)
		return
	}

	// Parse segments and render each one
	segments := ParseWidgetSegments(line.Text)
	x := line.X

	for _, seg := range segments {
		if seg.IsWidget && seg.Widget != nil {
			// Render the widget
			g.drawInlineWidget(screen, seg.Widget, x, line.Y, line.Color)
			x += seg.Widget.Width
		} else {
			// Render text segment
			g.textRenderer.DrawText(screen, seg.Text, x, line.Y, line.Color)
			textWidth, _ := g.textRenderer.MeasureText(seg.Text)
			x += textWidth
		}
	}
}

// drawInlineWidget renders a widget at the specified position.
func (g *Game) drawInlineWidget(screen *ebiten.Image, marker *WidgetMarker, x, y float64, clr color.RGBA) {
	// Adjust y to center the widget vertically on the text baseline
	// Text baseline is at y, widget should be centered around the text
	lineHeight := g.textRenderer.LineHeight()
	widgetY := y - lineHeight + (lineHeight-marker.Height)/2

	switch marker.Type {
	case WidgetTypeBar:
		g.drawProgressBar(screen, x, widgetY, marker.Width, marker.Height, marker.Value, clr)
	case WidgetTypeGraph:
		g.drawGraphWidget(screen, x, widgetY, marker.Width, marker.Height, marker.Value, clr)
	case WidgetTypeGauge:
		// Gauge is not yet implemented, fall back to bar
		g.drawProgressBar(screen, x, widgetY, marker.Width, marker.Height, marker.Value, clr)
	}
}

// drawProgressBar renders a horizontal progress bar.
func (g *Game) drawProgressBar(screen *ebiten.Image, x, y, width, height, value float64, clr color.RGBA) {
	// Draw background
	bgColor := color.RGBA{R: clr.R / 3, G: clr.G / 3, B: clr.B / 3, A: clr.A}
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(width), float32(height), bgColor, false)

	// Draw filled portion
	fillWidth := width * value / 100
	if fillWidth > width {
		fillWidth = width
	}
	if fillWidth > 0 {
		vector.DrawFilledRect(screen, float32(x), float32(y), float32(fillWidth), float32(height), clr, false)
	}

	// Draw border
	borderColor := color.RGBA{R: clr.R / 2, G: clr.G / 2, B: clr.B / 2, A: clr.A}
	vector.StrokeRect(screen, float32(x), float32(y), float32(width), float32(height), 1, borderColor, false)
}

// drawGraphWidget renders a simple filled area representing a graph.
func (g *Game) drawGraphWidget(screen *ebiten.Image, x, y, width, height, value float64, clr color.RGBA) {
	// Draw background
	bgColor := color.RGBA{R: clr.R / 3, G: clr.G / 3, B: clr.B / 3, A: clr.A}
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(width), float32(height), bgColor, false)

	// Draw filled area from bottom
	fillHeight := height * value / 100
	if fillHeight > height {
		fillHeight = height
	}
	if fillHeight > 0 {
		fillY := y + height - fillHeight
		// Use a gradient-like effect with lighter fill
		fillColor := color.RGBA{R: clr.R, G: clr.G, B: clr.B, A: uint8(float64(clr.A) * 0.7)}
		vector.DrawFilledRect(screen, float32(x), float32(fillY), float32(width), float32(fillHeight), fillColor, false)
	}

	// Draw border
	borderColor := color.RGBA{R: clr.R / 2, G: clr.G / 2, B: clr.B / 2, A: clr.A}
	vector.StrokeRect(screen, float32(x), float32(y), float32(width), float32(height), 1, borderColor, false)
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
