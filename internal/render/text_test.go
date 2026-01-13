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
	if tr.fontManager == nil {
		t.Error("fontManager should not be nil")
	}
	if tr.fontSize != defaultFontSize {
		t.Errorf("fontSize = %v, want %v", tr.fontSize, defaultFontSize)
	}
	if tr.fontFamily != defaultFontFamily {
		t.Errorf("fontFamily = %v, want %v", tr.fontFamily, defaultFontFamily)
	}
}

func TestTextRendererSetFontSize(t *testing.T) {
	tr := NewTextRenderer()

	tr.SetFontSize(24.0)

	if tr.FontSize() != 24.0 {
		t.Errorf("FontSize() = %v, want 24.0", tr.FontSize())
	}
}

func TestTextRendererSetFontSizeNegative(t *testing.T) {
	tr := NewTextRenderer()

	// Set to negative value should reset to default
	tr.SetFontSize(-5.0)

	if tr.FontSize() != defaultFontSize {
		t.Errorf("FontSize() after negative = %v, want %v", tr.FontSize(), defaultFontSize)
	}
}

func TestTextRendererSetFontSizeZero(t *testing.T) {
	tr := NewTextRenderer()

	// Set to zero should reset to default
	tr.SetFontSize(0)

	if tr.FontSize() != defaultFontSize {
		t.Errorf("FontSize() after zero = %v, want %v", tr.FontSize(), defaultFontSize)
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

			if tt.text != "" {
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

func TestTextRendererWithManager(t *testing.T) {
	fm := NewFontManager()
	tr := NewTextRendererWithManager(fm)

	if tr == nil {
		t.Fatal("NewTextRendererWithManager() returned nil")
	}
	if tr.fontManager != fm {
		t.Error("fontManager should be the one passed to constructor")
	}
}

func TestTextRendererSetFont(t *testing.T) {
	tr := NewTextRenderer()

	tr.SetFont("GoSans", FontStyleBold)

	if tr.FontFamily() != "GoSans" {
		t.Errorf("FontFamily() = %v, want GoSans", tr.FontFamily())
	}
	if tr.FontStyle() != FontStyleBold {
		t.Errorf("FontStyle() = %v, want bold", tr.FontStyle())
	}
}

func TestTextRendererSetFontFamily(t *testing.T) {
	tr := NewTextRenderer()

	tr.SetFontFamily("GoSans")

	if tr.FontFamily() != "GoSans" {
		t.Errorf("FontFamily() = %v, want GoSans", tr.FontFamily())
	}
}

func TestTextRendererSetFontStyle(t *testing.T) {
	tr := NewTextRenderer()

	tr.SetFontStyle(FontStyleItalic)

	if tr.FontStyle() != FontStyleItalic {
		t.Errorf("FontStyle() = %v, want italic", tr.FontStyle())
	}
}

func TestTextRendererFontManager(t *testing.T) {
	tr := NewTextRenderer()

	fm := tr.FontManager()
	if fm == nil {
		t.Error("FontManager() should not return nil")
	}
}

func TestTextRendererLoadFontFromFile(t *testing.T) {
	tr := NewTextRenderer()

	// Non-existent file should fail
	err := tr.LoadFontFromFile("TestFamily", FontStyleRegular, "/nonexistent/font.ttf")
	if err == nil {
		t.Error("LoadFontFromFile should fail for non-existent file")
	}
}

func TestTextRendererDifferentFonts(t *testing.T) {
	tr := NewTextRenderer()

	// Measure with GoMono
	tr.SetFontFamily("GoMono")
	tr.SetFontStyle(FontStyleRegular)
	w1, _ := tr.MeasureText("Hello")

	// Measure with GoSans - should be different
	tr.SetFontFamily("GoSans")
	w2, _ := tr.MeasureText("Hello")

	// Different fonts typically have different metrics
	// This may not always be true, but for Go fonts they should differ
	if w1 == w2 {
		t.Log("Warning: GoMono and GoSans have same width for 'Hello' (may be acceptable)")
	}
}

func TestTextRendererStyleVariations(t *testing.T) {
	tr := NewTextRenderer()
	tr.SetFontFamily("GoMono")

	styles := []FontStyle{
		FontStyleRegular,
		FontStyleBold,
		FontStyleItalic,
		FontStyleBoldItalic,
	}

	for _, style := range styles {
		t.Run(style.String(), func(t *testing.T) {
			tr.SetFontStyle(style)
			w, h := tr.MeasureText("Test")
			if w <= 0 || h <= 0 {
				t.Errorf("Measurement failed for style %v", style)
			}
		})
	}
}

func TestTextRendererConcurrentFontAccess(t *testing.T) {
	tr := NewTextRenderer()
	done := make(chan bool)

	go func() {
		for i := 0; i < 100; i++ {
			tr.SetFontFamily("GoMono")
			tr.SetFontStyle(FontStyleBold)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			tr.SetFontFamily("GoSans")
			tr.SetFontStyle(FontStyleItalic)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			_ = tr.FontFamily()
			_ = tr.FontStyle()
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
	<-done
}
