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
