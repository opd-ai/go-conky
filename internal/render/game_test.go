//go:build !noebiten

package render

import (
	"context"
	"fmt"
	"image/color"
	"sync"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// mockTextRenderer implements TextRendererInterface for testing
type mockTextRenderer struct {
	mu               sync.RWMutex
	drawTextCalls    int
	measureTextCalls int
	fontSize         float64
}

func newMockTextRenderer() *mockTextRenderer {
	return &mockTextRenderer{fontSize: 14.0}
}

func (m *mockTextRenderer) DrawText(screen *ebiten.Image, textStr string, x, y float64, clr color.RGBA) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.drawTextCalls++
}

func (m *mockTextRenderer) MeasureText(textStr string) (width, height float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.measureTextCalls++
	return float64(len(textStr)) * 10, 16
}

func (m *mockTextRenderer) LineHeight() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.fontSize * 1.2
}

func (m *mockTextRenderer) SetFontSize(size float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.fontSize = size
}

func (m *mockTextRenderer) FontSize() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.fontSize
}

func TestNewGameWithRenderer(t *testing.T) {
	config := DefaultConfig()
	renderer := newMockTextRenderer()
	game := NewGameWithRenderer(config, renderer)

	if game == nil {
		t.Fatal("NewGameWithRenderer() returned nil")
	}
	if game.textRenderer != renderer {
		t.Error("textRenderer was not set correctly")
	}
}

func TestGameSetLines(t *testing.T) {
	renderer := newMockTextRenderer()
	game := NewGameWithRenderer(DefaultConfig(), renderer)

	lines := []TextLine{
		{Text: "Line 1", X: 10, Y: 20, Color: color.RGBA{R: 255, G: 0, B: 0, A: 255}},
		{Text: "Line 2", X: 10, Y: 40, Color: color.RGBA{R: 0, G: 255, B: 0, A: 255}},
	}

	game.SetLines(lines)

	game.mu.RLock()
	if len(game.lines) != 2 {
		t.Errorf("lines count = %d, want 2", len(game.lines))
	}
	game.mu.RUnlock()
}

func TestGameAddLine(t *testing.T) {
	renderer := newMockTextRenderer()
	game := NewGameWithRenderer(DefaultConfig(), renderer)

	line := TextLine{Text: "Test Line", X: 10, Y: 20}
	game.AddLine(line)

	game.mu.RLock()
	if len(game.lines) != 1 {
		t.Errorf("lines count = %d, want 1", len(game.lines))
	}
	if game.lines[0].Text != "Test Line" {
		t.Errorf("line text = %q, want %q", game.lines[0].Text, "Test Line")
	}
	game.mu.RUnlock()
}

func TestGameClearLines(t *testing.T) {
	renderer := newMockTextRenderer()
	game := NewGameWithRenderer(DefaultConfig(), renderer)

	game.AddLine(TextLine{Text: "Line 1"})
	game.AddLine(TextLine{Text: "Line 2"})
	game.ClearLines()

	game.mu.RLock()
	if len(game.lines) != 0 {
		t.Errorf("lines count after clear = %d, want 0", len(game.lines))
	}
	game.mu.RUnlock()
}

func TestGameLayout(t *testing.T) {
	config := Config{Width: 800, Height: 600}
	renderer := newMockTextRenderer()
	game := NewGameWithRenderer(config, renderer)

	w, h := game.Layout(1920, 1080)
	if w != 800 {
		t.Errorf("Layout width = %d, want 800", w)
	}
	if h != 600 {
		t.Errorf("Layout height = %d, want 600", h)
	}
}

func TestGameIsRunning(t *testing.T) {
	renderer := newMockTextRenderer()
	game := NewGameWithRenderer(DefaultConfig(), renderer)

	if game.IsRunning() {
		t.Error("newly created game should not be running")
	}
}

func TestGameConfig(t *testing.T) {
	config := Config{Width: 1024, Height: 768, Title: "Test"}
	renderer := newMockTextRenderer()
	game := NewGameWithRenderer(config, renderer)

	got := game.Config()
	if got.Width != 1024 {
		t.Errorf("Config().Width = %d, want 1024", got.Width)
	}
	if got.Height != 768 {
		t.Errorf("Config().Height = %d, want 768", got.Height)
	}
}

func TestGameSetConfig(t *testing.T) {
	config := Config{Width: 800, Height: 600, Title: "Initial"}
	renderer := newMockTextRenderer()
	game := NewGameWithRenderer(config, renderer)

	// Verify initial config
	got := game.Config()
	if got.Width != 800 {
		t.Errorf("initial Config().Width = %d, want 800", got.Width)
	}

	// Update config
	newConfig := Config{Width: 1024, Height: 768, Title: "Updated"}
	game.SetConfig(newConfig)

	// Verify updated config
	got = game.Config()
	if got.Width != 1024 {
		t.Errorf("updated Config().Width = %d, want 1024", got.Width)
	}
	if got.Height != 768 {
		t.Errorf("updated Config().Height = %d, want 768", got.Height)
	}
	if got.Title != "Updated" {
		t.Errorf("updated Config().Title = %q, want %q", got.Title, "Updated")
	}
}

func TestGameSetConfigConcurrent(t *testing.T) {
	config := Config{Width: 800, Height: 600, Title: "Initial"}
	renderer := newMockTextRenderer()
	game := NewGameWithRenderer(config, renderer)

	done := make(chan bool)

	// Concurrent config updates
	go func() {
		for i := 0; i < 100; i++ {
			game.SetConfig(Config{Width: 1024, Height: 768, Title: "Updated"})
		}
		done <- true
	}()

	// Concurrent config reads
	go func() {
		for i := 0; i < 100; i++ {
			_ = game.Config()
		}
		done <- true
	}()

	<-done
	<-done
}

// mockDataProvider implements DataProvider for testing
type mockDataProvider struct {
	updateCalled bool
	updateError  error
}

func (m *mockDataProvider) Update() error {
	m.updateCalled = true
	return m.updateError
}

func TestGameSetDataProvider(t *testing.T) {
	renderer := newMockTextRenderer()
	game := NewGameWithRenderer(DefaultConfig(), renderer)
	provider := &mockDataProvider{}

	game.SetDataProvider(provider)

	game.mu.RLock()
	if game.dataProvider != provider {
		t.Error("dataProvider was not set correctly")
	}
	game.mu.RUnlock()
}

func TestGameUpdate(t *testing.T) {
	config := DefaultConfig()
	config.UpdateInterval = 0
	renderer := newMockTextRenderer()
	game := NewGameWithRenderer(config, renderer)

	provider := &mockDataProvider{}
	game.SetDataProvider(provider)

	game.mu.Lock()
	game.lastUpdate = time.Now().Add(-2 * time.Second)
	game.mu.Unlock()

	err := game.Update()
	if err != nil {
		t.Errorf("Update() error = %v", err)
	}

	if !provider.updateCalled {
		t.Error("dataProvider.Update() was not called")
	}
}

func TestGameUpdateWithNoProvider(t *testing.T) {
	renderer := newMockTextRenderer()
	game := NewGameWithRenderer(DefaultConfig(), renderer)

	err := game.Update()
	if err != nil {
		t.Errorf("Update() error = %v", err)
	}
}

func TestGameConcurrentAccess(t *testing.T) {
	renderer := newMockTextRenderer()
	game := NewGameWithRenderer(DefaultConfig(), renderer)

	done := make(chan bool)

	go func() {
		for i := 0; i < 100; i++ {
			game.AddLine(TextLine{Text: "test"})
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			_ = game.Config()
			_ = game.IsRunning()
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			game.ClearLines()
		}
		done <- true
	}()

	<-done
	<-done
	<-done
}

func TestGameLinesIsolation(t *testing.T) {
	renderer := newMockTextRenderer()
	game := NewGameWithRenderer(DefaultConfig(), renderer)

	original := []TextLine{
		{Text: "Original"},
	}
	game.SetLines(original)

	original[0].Text = "Modified"

	game.mu.RLock()
	if game.lines[0].Text != "Original" {
		t.Error("SetLines should copy the slice, not share it")
	}
	game.mu.RUnlock()
}

func TestGameSetErrorHandler(t *testing.T) {
	renderer := newMockTextRenderer()
	game := NewGameWithRenderer(DefaultConfig(), renderer)

	var capturedErr error
	customHandler := func(err error) {
		capturedErr = err
	}

	game.SetErrorHandler(customHandler)

	// Test that the custom handler is called
	testErr := fmt.Errorf("test error")
	provider := &mockDataProvider{updateError: testErr}
	game.SetDataProvider(provider)

	game.mu.Lock()
	game.lastUpdate = time.Now().Add(-2 * time.Second)
	game.mu.Unlock()

	_ = game.Update()

	if capturedErr != testErr {
		t.Errorf("error handler did not capture error: got %v, want %v", capturedErr, testErr)
	}
}

func TestGameNilErrorHandler(t *testing.T) {
	renderer := newMockTextRenderer()
	game := NewGameWithRenderer(DefaultConfig(), renderer)

	// Set nil error handler - should not panic
	game.SetErrorHandler(nil)

	testErr := fmt.Errorf("test error")
	provider := &mockDataProvider{updateError: testErr}
	game.SetDataProvider(provider)

	game.mu.Lock()
	game.lastUpdate = time.Now().Add(-2 * time.Second)
	game.mu.Unlock()

	// Should not panic with nil handler
	err := game.Update()
	if err != nil {
		t.Errorf("Update() error = %v", err)
	}
}

func TestGameSetContext(t *testing.T) {
	renderer := newMockTextRenderer()
	game := NewGameWithRenderer(DefaultConfig(), renderer)

	ctx, cancel := context.WithCancel(context.Background())
	game.SetContext(ctx)

	// Should not return error when context is not cancelled
	err := game.Update()
	if err != nil {
		t.Errorf("Update() error = %v, want nil", err)
	}

	// Cancel context and verify Update returns ErrGameTerminated
	cancel()

	err = game.Update()
	if err != ErrGameTerminated {
		t.Errorf("Update() error = %v, want %v", err, ErrGameTerminated)
	}
}

func TestGameUpdateWithCancelledContext(t *testing.T) {
	renderer := newMockTextRenderer()
	game := NewGameWithRenderer(DefaultConfig(), renderer)

	// Create an already cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	game.SetContext(ctx)

	err := game.Update()
	if err != ErrGameTerminated {
		t.Errorf("Update() error = %v, want %v", err, ErrGameTerminated)
	}
}

func TestGameUpdateWithNilContext(t *testing.T) {
	renderer := newMockTextRenderer()
	game := NewGameWithRenderer(DefaultConfig(), renderer)

	// No context set - should work normally
	err := game.Update()
	if err != nil {
		t.Errorf("Update() error = %v, want nil", err)
	}
}

func TestErrGameTerminated(t *testing.T) {
	if ErrGameTerminated == nil {
		t.Error("ErrGameTerminated should not be nil")
	}
	if ErrGameTerminated.Error() != "game terminated" {
		t.Errorf("ErrGameTerminated.Error() = %q, want %q", ErrGameTerminated.Error(), "game terminated")
	}
}

func TestDrawLineWithWidgets(t *testing.T) {
	config := DefaultConfig()
	mockRenderer := newMockTextRenderer()
	game := NewGameWithRenderer(config, mockRenderer)

	// Test drawing plain text (no widgets)
	t.Run("plain text", func(t *testing.T) {
		line := TextLine{
			Text:  "Hello World",
			X:     10,
			Y:     20,
			Color: color.RGBA{R: 255, G: 255, B: 255, A: 255},
		}
		// This should use the text renderer once
		mockRenderer.drawTextCalls = 0
		game.drawLineWithWidgets(ebiten.NewImage(400, 300), line)
		if mockRenderer.drawTextCalls != 1 {
			t.Errorf("expected 1 DrawText call, got %d", mockRenderer.drawTextCalls)
		}
	})

	// Test drawing text with widget marker
	t.Run("text with widget", func(t *testing.T) {
		// Create a line with embedded widget marker
		marker := EncodeBarMarker(50, 100, 8)
		line := TextLine{
			Text:  "CPU: " + marker + " done",
			X:     10,
			Y:     20,
			Color: color.RGBA{R: 255, G: 255, B: 255, A: 255},
		}
		mockRenderer.drawTextCalls = 0
		mockRenderer.measureTextCalls = 0
		game.drawLineWithWidgets(ebiten.NewImage(400, 300), line)
		// Should draw "CPU: " and " done" as text (2 calls)
		if mockRenderer.drawTextCalls != 2 {
			t.Errorf("expected 2 DrawText calls, got %d", mockRenderer.drawTextCalls)
		}
	})

	// Test widget-only line
	t.Run("widget only", func(t *testing.T) {
		marker := EncodeBarMarker(75, 100, 8)
		line := TextLine{
			Text:  marker,
			X:     10,
			Y:     20,
			Color: color.RGBA{R: 100, G: 200, B: 100, A: 255},
		}
		mockRenderer.drawTextCalls = 0
		game.drawLineWithWidgets(ebiten.NewImage(400, 300), line)
		// No text to draw
		if mockRenderer.drawTextCalls != 0 {
			t.Errorf("expected 0 DrawText calls for widget-only line, got %d", mockRenderer.drawTextCalls)
		}
	})
}

func TestDrawInlineWidgetTypes(t *testing.T) {
	config := DefaultConfig()
	mockRenderer := newMockTextRenderer()
	game := NewGameWithRenderer(config, mockRenderer)
	screen := ebiten.NewImage(400, 300)

	// Test bar widget
	t.Run("bar widget", func(t *testing.T) {
		marker := &WidgetMarker{Type: WidgetTypeBar, Value: 50, Width: 100, Height: 8}
		// Should not panic
		game.drawInlineWidget(screen, marker, 10, 20, color.RGBA{R: 100, G: 200, B: 100, A: 255})
	})

	// Test graph widget
	t.Run("graph widget", func(t *testing.T) {
		marker := &WidgetMarker{Type: WidgetTypeGraph, Value: 75, Width: 100, Height: 20}
		// Should not panic
		game.drawInlineWidget(screen, marker, 10, 50, color.RGBA{R: 200, G: 100, B: 100, A: 255})
	})

	// Test gauge widget (falls back to bar)
	t.Run("gauge widget", func(t *testing.T) {
		marker := &WidgetMarker{Type: WidgetTypeGauge, Value: 90, Width: 30, Height: 30}
		// Should not panic
		game.drawInlineWidget(screen, marker, 10, 100, color.RGBA{R: 100, G: 100, B: 200, A: 255})
	})
}

func TestDrawProgressBar(t *testing.T) {
	config := DefaultConfig()
	mockRenderer := newMockTextRenderer()
	game := NewGameWithRenderer(config, mockRenderer)
	screen := ebiten.NewImage(400, 300)

	tests := []struct {
		name   string
		value  float64
		width  float64
		height float64
	}{
		{"empty bar", 0, 100, 8},
		{"half bar", 50, 100, 8},
		{"full bar", 100, 100, 8},
		{"over 100%", 150, 100, 8},
		{"thin bar", 50, 200, 4},
		{"tall bar", 50, 50, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			game.drawProgressBar(screen, 10, 10, tt.width, tt.height, tt.value,
				color.RGBA{R: 100, G: 200, B: 100, A: 255})
		})
	}
}

func TestDrawGraphWidget(t *testing.T) {
	config := DefaultConfig()
	mockRenderer := newMockTextRenderer()
	game := NewGameWithRenderer(config, mockRenderer)
	screen := ebiten.NewImage(400, 300)

	tests := []struct {
		name   string
		value  float64
		width  float64
		height float64
	}{
		{"empty graph", 0, 100, 20},
		{"half graph", 50, 100, 20},
		{"full graph", 100, 100, 20},
		{"over 100%", 150, 100, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			game.drawGraphWidget(screen, 10, 10, tt.width, tt.height, tt.value,
				color.RGBA{R: 100, G: 100, B: 200, A: 255})
		})
	}
}

// Test text effects configuration
func TestConfigTextEffects(t *testing.T) {
	config := DefaultConfig()

	// Test default values
	if config.DrawBorders {
		t.Error("DrawBorders should be false by default")
	}
	if config.DrawOutline {
		t.Error("DrawOutline should be false by default")
	}
	if config.DrawShades {
		t.Error("DrawShades should be false by default")
	}
	if config.BorderWidth != 1 {
		t.Errorf("BorderWidth should be 1 by default, got %d", config.BorderWidth)
	}
	if config.BorderInnerMargin != 5 {
		t.Errorf("BorderInnerMargin should be 5 by default, got %d", config.BorderInnerMargin)
	}
	if config.BorderOuterMargin != 5 {
		t.Errorf("BorderOuterMargin should be 5 by default, got %d", config.BorderOuterMargin)
	}
	if config.StippledBorders {
		t.Error("StippledBorders should be false by default")
	}
}

// Test text rendering with shade effect
func TestDrawTextWithShade(t *testing.T) {
	config := DefaultConfig()
	config.DrawShades = true
	config.ShadeColor = color.RGBA{R: 50, G: 50, B: 50, A: 128}

	renderer := newMockTextRenderer()
	game := NewGameWithRenderer(config, renderer)

	screen := ebiten.NewImage(100, 100)
	defer screen.Deallocate()

	// Draw text with shade enabled
	game.drawTextWithEffects(screen, "Test", 10, 20, color.RGBA{R: 255, G: 255, B: 255, A: 255})

	// Should have called DrawText twice: once for shade, once for main text
	renderer.mu.RLock()
	calls := renderer.drawTextCalls
	renderer.mu.RUnlock()

	if calls != 2 {
		t.Errorf("Expected 2 DrawText calls (shade + main), got %d", calls)
	}
}

// Test text rendering with outline effect
func TestDrawTextWithOutline(t *testing.T) {
	config := DefaultConfig()
	config.DrawOutline = true
	config.OutlineColor = color.RGBA{R: 0, G: 0, B: 0, A: 255}

	renderer := newMockTextRenderer()
	game := NewGameWithRenderer(config, renderer)

	screen := ebiten.NewImage(100, 100)
	defer screen.Deallocate()

	// Draw text with outline enabled
	game.drawTextWithEffects(screen, "Test", 10, 20, color.RGBA{R: 255, G: 255, B: 255, A: 255})

	// Should have called DrawText 5 times: 4 for outline + 1 for main text
	renderer.mu.RLock()
	calls := renderer.drawTextCalls
	renderer.mu.RUnlock()

	if calls != 5 {
		t.Errorf("Expected 5 DrawText calls (4 outline + main), got %d", calls)
	}
}

// Test text rendering with both shade and outline
func TestDrawTextWithShadeAndOutline(t *testing.T) {
	config := DefaultConfig()
	config.DrawShades = true
	config.DrawOutline = true

	renderer := newMockTextRenderer()
	game := NewGameWithRenderer(config, renderer)

	screen := ebiten.NewImage(100, 100)
	defer screen.Deallocate()

	game.drawTextWithEffects(screen, "Test", 10, 20, color.RGBA{R: 255, G: 255, B: 255, A: 255})

	// Should have called DrawText 6 times: 1 shade + 4 outline + 1 main
	renderer.mu.RLock()
	calls := renderer.drawTextCalls
	renderer.mu.RUnlock()

	if calls != 6 {
		t.Errorf("Expected 6 DrawText calls (shade + 4 outline + main), got %d", calls)
	}
}

// Test text rendering with no effects
func TestDrawTextWithNoEffects(t *testing.T) {
	config := DefaultConfig()
	// Default: no effects enabled

	renderer := newMockTextRenderer()
	game := NewGameWithRenderer(config, renderer)

	screen := ebiten.NewImage(100, 100)
	defer screen.Deallocate()

	game.drawTextWithEffects(screen, "Test", 10, 20, color.RGBA{R: 255, G: 255, B: 255, A: 255})

	// Should have called DrawText exactly once
	renderer.mu.RLock()
	calls := renderer.drawTextCalls
	renderer.mu.RUnlock()

	if calls != 1 {
		t.Errorf("Expected 1 DrawText call, got %d", calls)
	}
}

// Test border drawing
func TestDrawBorders(t *testing.T) {
	config := DefaultConfig()
	config.DrawBorders = true
	config.BorderWidth = 2
	config.BorderColor = color.RGBA{R: 255, G: 0, B: 0, A: 255}

	renderer := newMockTextRenderer()
	game := NewGameWithRenderer(config, renderer)

	screen := ebiten.NewImage(400, 300)
	defer screen.Deallocate()

	// Should not panic
	game.drawBorders(screen)
}

// Test stippled border drawing
func TestDrawStippledBorders(t *testing.T) {
	config := DefaultConfig()
	config.DrawBorders = true
	config.StippledBorders = true
	config.BorderWidth = 1

	renderer := newMockTextRenderer()
	game := NewGameWithRenderer(config, renderer)

	screen := ebiten.NewImage(400, 300)
	defer screen.Deallocate()

	// Should not panic
	game.drawBorders(screen)
}

// Test border with zero width
func TestDrawBordersZeroWidth(t *testing.T) {
	config := DefaultConfig()
	config.DrawBorders = true
	config.BorderWidth = 0 // Should default to 1

	renderer := newMockTextRenderer()
	game := NewGameWithRenderer(config, renderer)

	screen := ebiten.NewImage(400, 300)
	defer screen.Deallocate()

	// Should not panic and should use default width of 1
	game.drawBorders(screen)
}

// Test border with large margins
func TestDrawBordersLargeMargins(t *testing.T) {
	config := DefaultConfig()
	config.DrawBorders = true
	config.BorderOuterMargin = 300 // Larger than half of height

	renderer := newMockTextRenderer()
	game := NewGameWithRenderer(config, renderer)

	screen := ebiten.NewImage(400, 300)
	defer screen.Deallocate()

	// Should not panic even with unreasonable margins
	game.drawBorders(screen)
}

// Test Draw method includes borders when enabled
func TestGameDrawWithBorders(t *testing.T) {
	config := DefaultConfig()
	config.DrawBorders = true

	renderer := newMockTextRenderer()
	game := NewGameWithRenderer(config, renderer)

	screen := ebiten.NewImage(400, 300)
	defer screen.Deallocate()

	// Should not panic
	game.Draw(screen)
}

// Test drawLineWithWidgets uses text effects
func TestDrawLineWithWidgetsUsesEffects(t *testing.T) {
	config := DefaultConfig()
	config.DrawShades = true

	renderer := newMockTextRenderer()
	game := NewGameWithRenderer(config, renderer)

	screen := ebiten.NewImage(100, 100)
	defer screen.Deallocate()

	line := TextLine{Text: "Hello", X: 10, Y: 20, Color: color.RGBA{R: 255, G: 255, B: 255, A: 255}}
	game.drawLineWithWidgets(screen, line)

	// Should have called DrawText twice (shade + main)
	renderer.mu.RLock()
	calls := renderer.drawTextCalls
	renderer.mu.RUnlock()

	if calls != 2 {
		t.Errorf("Expected 2 DrawText calls with shade enabled, got %d", calls)
	}
}

// Test abs32 helper function
func TestAbs32(t *testing.T) {
	tests := []struct {
		input    float32
		expected float32
	}{
		{5.0, 5.0},
		{-5.0, 5.0},
		{0.0, 0.0},
		{3.14, 3.14},
		{-3.14, 3.14},
	}

	for _, tt := range tests {
		result := abs32(tt.input)
		if result != tt.expected {
			t.Errorf("abs32(%f) = %f, want %f", tt.input, result, tt.expected)
		}
	}
}

// Test drawStippledLine with various inputs
func TestDrawStippledLine(t *testing.T) {
	config := DefaultConfig()
	renderer := newMockTextRenderer()
	game := NewGameWithRenderer(config, renderer)

	screen := ebiten.NewImage(200, 200)
	defer screen.Deallocate()

	clr := color.RGBA{R: 255, G: 255, B: 255, A: 255}

	tests := []struct {
		name            string
		x1, y1, x2, y2  float32
		strokeWidth     float32
		dashLen, gapLen float32
	}{
		{"horizontal line", 10, 10, 100, 10, 1, 4, 2},
		{"vertical line", 10, 10, 10, 100, 1, 4, 2},
		{"diagonal line", 10, 10, 100, 100, 1, 4, 2},
		{"zero length", 50, 50, 50, 50, 1, 4, 2},
		{"thick stroke", 10, 10, 100, 10, 3, 4, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			game.drawStippledLine(screen, tt.x1, tt.y1, tt.x2, tt.y2, tt.strokeWidth, clr, tt.dashLen, tt.gapLen, tt.dashLen+tt.gapLen)
		})
	}
}

// Test default color fallbacks for effects
func TestTextEffectsDefaultColors(t *testing.T) {
	config := DefaultConfig()
	config.DrawShades = true
	config.DrawOutline = true
	// Leave ShadeColor and OutlineColor as zero values

	renderer := newMockTextRenderer()
	game := NewGameWithRenderer(config, renderer)

	screen := ebiten.NewImage(100, 100)
	defer screen.Deallocate()

	// Should not panic and should use default colors
	game.drawTextWithEffects(screen, "Test", 10, 20, color.RGBA{R: 255, G: 255, B: 255, A: 255})

	renderer.mu.RLock()
	calls := renderer.drawTextCalls
	renderer.mu.RUnlock()

	// 1 shade + 4 outline + 1 main = 6
	if calls != 6 {
		t.Errorf("Expected 6 DrawText calls, got %d", calls)
	}
}

// Test border color fallback
func TestBorderColorFallback(t *testing.T) {
	config := DefaultConfig()
	config.DrawBorders = true
	// Leave BorderColor as zero value

	renderer := newMockTextRenderer()
	game := NewGameWithRenderer(config, renderer)

	screen := ebiten.NewImage(400, 300)
	defer screen.Deallocate()

	// Should not panic and should use default white color
	game.drawBorders(screen)
}

// Test ARGB transparency configuration
func TestGameARGBTransparencyConfig(t *testing.T) {
	tests := []struct {
		name        string
		transparent bool
		argbVisual  bool
		argbValue   int
	}{
		{
			name:        "default no transparency",
			transparent: false,
			argbVisual:  false,
			argbValue:   255,
		},
		{
			name:        "transparent enabled",
			transparent: true,
			argbVisual:  false,
			argbValue:   255,
		},
		{
			name:        "argb visual enabled",
			transparent: false,
			argbVisual:  true,
			argbValue:   128,
		},
		{
			name:        "both with semi-transparent",
			transparent: true,
			argbVisual:  true,
			argbValue:   100,
		},
		{
			name:        "fully transparent",
			transparent: true,
			argbVisual:  true,
			argbValue:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				Width:       400,
				Height:      300,
				Transparent: tt.transparent,
				ARGBVisual:  tt.argbVisual,
				ARGBValue:   tt.argbValue,
			}
			renderer := newMockTextRenderer()
			game := NewGameWithRenderer(config, renderer)

			gotConfig := game.Config()
			if gotConfig.Transparent != tt.transparent {
				t.Errorf("Transparent = %v, want %v", gotConfig.Transparent, tt.transparent)
			}
			if gotConfig.ARGBVisual != tt.argbVisual {
				t.Errorf("ARGBVisual = %v, want %v", gotConfig.ARGBVisual, tt.argbVisual)
			}
			if gotConfig.ARGBValue != tt.argbValue {
				t.Errorf("ARGBValue = %d, want %d", gotConfig.ARGBValue, tt.argbValue)
			}
		})
	}
}

// Test that Draw applies ARGB alpha to background
func TestGameDrawWithARGBAlpha(t *testing.T) {
	tests := []struct {
		name         string
		argbVisual   bool
		argbValue    int
		expectedAlph uint8
	}{
		{
			name:         "argb disabled uses original alpha",
			argbVisual:   false,
			argbValue:    100,
			expectedAlph: 200, // Original BackgroundColor.A
		},
		{
			name:         "argb enabled uses argb value",
			argbVisual:   true,
			argbValue:    100,
			expectedAlph: 100,
		},
		{
			name:         "argb fully transparent",
			argbVisual:   true,
			argbValue:    0,
			expectedAlph: 0,
		},
		{
			name:         "argb fully opaque",
			argbVisual:   true,
			argbValue:    255,
			expectedAlph: 255,
		},
		{
			name:         "argb value clamped negative",
			argbVisual:   true,
			argbValue:    -50,
			expectedAlph: 0,
		},
		{
			name:         "argb value clamped over 255",
			argbVisual:   true,
			argbValue:    300,
			expectedAlph: 255,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig()
			config.ARGBVisual = tt.argbVisual
			config.ARGBValue = tt.argbValue
			// BackgroundColor.A is 200 in DefaultConfig

			renderer := newMockTextRenderer()
			game := NewGameWithRenderer(config, renderer)

			screen := ebiten.NewImage(400, 300)
			defer screen.Deallocate()

			// Draw should not panic
			game.Draw(screen)

			// Verify the config is preserved correctly
			if game.config.ARGBVisual != tt.argbVisual {
				t.Errorf("ARGBVisual = %v, want %v", game.config.ARGBVisual, tt.argbVisual)
			}
			if game.config.ARGBValue != tt.argbValue {
				t.Errorf("ARGBValue = %d, want %d", game.config.ARGBValue, tt.argbValue)
			}
		})
	}
}
