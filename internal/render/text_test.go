//go:build !noebiten

package render

import (
	"testing"
)

func TestNewTextRenderer(t *testing.T) {
	tr := NewTextRenderer()

	if tr == nil {
		t.Fatal("NewTextRenderer() returned nil")
	}
	if tr.fontSource == nil {
		t.Error("fontSource should not be nil")
	}
	if tr.fontSize != defaultFontSize {
		t.Errorf("fontSize = %v, want %v", tr.fontSize, defaultFontSize)
	}
}

func TestTextRendererSetFontSize(t *testing.T) {
	tr := NewTextRenderer()

	tr.SetFontSize(24.0)

	if tr.FontSize() != 24.0 {
		t.Errorf("FontSize() = %v, want 24.0", tr.FontSize())
	}
}

func TestTextRendererFontSize(t *testing.T) {
	tr := NewTextRenderer()

	size := tr.FontSize()
	if size != defaultFontSize {
		t.Errorf("FontSize() = %v, want %v", size, defaultFontSize)
	}
}

func TestTextRendererMeasureText(t *testing.T) {
	tr := NewTextRenderer()

	width, height := tr.MeasureText("Hello")

	if width <= 0 {
		t.Errorf("MeasureText width = %v, want > 0", width)
	}
	if height <= 0 {
		t.Errorf("MeasureText height = %v, want > 0", height)
	}
}

func TestTextRendererMeasureTextEmpty(t *testing.T) {
	tr := NewTextRenderer()

	width, _ := tr.MeasureText("")

	// Empty string should have width 0 or very small
	if width > 1 {
		t.Errorf("MeasureText empty string width = %v, want <= 1", width)
	}
}

func TestTextRendererLineHeight(t *testing.T) {
	tr := NewTextRenderer()

	lineHeight := tr.LineHeight()

	expectedMin := defaultFontSize
	expectedMax := defaultFontSize * 1.5

	if lineHeight < expectedMin || lineHeight > expectedMax {
		t.Errorf("LineHeight() = %v, want between %v and %v", lineHeight, expectedMin, expectedMax)
	}
}

func TestTextRendererConcurrentAccess(t *testing.T) {
	tr := NewTextRenderer()

	done := make(chan bool)

	go func() {
		for i := 0; i < 100; i++ {
			tr.SetFontSize(float64(12 + i%10))
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			_ = tr.FontSize()
			_ = tr.LineHeight()
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			_, _ = tr.MeasureText("Hello World")
		}
		done <- true
	}()

	<-done
	<-done
	<-done
}

func TestTextRendererMeasureTextDifferentStrings(t *testing.T) {
	tr := NewTextRenderer()

	tests := []struct {
		name string
		text string
	}{
		{"single char", "A"},
		{"word", "Hello"},
		{"sentence", "Hello, World!"},
		{"numbers", "12345"},
		{"special chars", "!@#$%"},
		{"mixed", "Hello 123!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			width, height := tr.MeasureText(tt.text)

			if len(tt.text) > 0 {
				if width <= 0 {
					t.Errorf("MeasureText(%q) width = %v, want > 0", tt.text, width)
				}
				if height <= 0 {
					t.Errorf("MeasureText(%q) height = %v, want > 0", tt.text, height)
				}
			}
		})
	}
}

func TestTextRendererFontSizeAffectsMeasurement(t *testing.T) {
	tr := NewTextRenderer()

	tr.SetFontSize(14.0)
	w1, _ := tr.MeasureText("Hello")

	tr.SetFontSize(28.0)
	w2, _ := tr.MeasureText("Hello")

	if w2 <= w1 {
		t.Errorf("Larger font size should produce larger width: %v <= %v", w2, w1)
	}
}
