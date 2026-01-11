//go:build !noebiten

package render

import (
	"image/color"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// mockTextRenderer implements TextRendererInterface for testing
type mockTextRenderer struct {
	drawTextCalls    int
	measureTextCalls int
	fontSize         float64
}

func newMockTextRenderer() *mockTextRenderer {
	return &mockTextRenderer{fontSize: 14.0}
}

func (m *mockTextRenderer) DrawText(screen *ebiten.Image, textStr string, x, y float64, clr color.RGBA) {
	m.drawTextCalls++
}

func (m *mockTextRenderer) MeasureText(textStr string) (width, height float64) {
	m.measureTextCalls++
	return float64(len(textStr)) * 10, 16
}

func (m *mockTextRenderer) LineHeight() float64 {
	return m.fontSize * 1.2
}

func (m *mockTextRenderer) SetFontSize(size float64) {
	m.fontSize = size
}

func (m *mockTextRenderer) FontSize() float64 {
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
