package render

import (
	"bytes"
	"image/color"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/gomonobold"
)

// defaultFontSize is the default font size in points.
const defaultFontSize = 14.0

// TextRenderer handles text rendering using Ebiten's text package.
type TextRenderer struct {
	fontSource *text.GoTextFaceSource
	fontSize   float64
	mu         sync.RWMutex
}

// NewTextRenderer creates a new TextRenderer with the default monospace font.
func NewTextRenderer() *TextRenderer {
	fontSource, err := text.NewGoTextFaceSource(bytes.NewReader(gomonobold.TTF))
	if err != nil {
		// This should never fail with the embedded font
		panic("failed to load embedded font: " + err.Error())
	}

	return &TextRenderer{
		fontSource: fontSource,
		fontSize:   defaultFontSize,
	}
}

// SetFontSize sets the font size for text rendering.
func (tr *TextRenderer) SetFontSize(size float64) {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	tr.fontSize = size
}

// FontSize returns the current font size.
func (tr *TextRenderer) FontSize() float64 {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	return tr.fontSize
}

// DrawText renders text at the specified position with the given color.
func (tr *TextRenderer) DrawText(screen *ebiten.Image, textStr string, x, y float64, clr color.RGBA) {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	face := &text.GoTextFace{
		Source: tr.fontSource,
		Size:   tr.fontSize,
	}

	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(clr)

	text.Draw(screen, textStr, face, op)
}

// MeasureText returns the width and height of the given text string.
func (tr *TextRenderer) MeasureText(textStr string) (width, height float64) {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	face := &text.GoTextFace{
		Source: tr.fontSource,
		Size:   tr.fontSize,
	}

	// lineSpacingInPixels should be the line height for proper text measurement
	lineSpacing := tr.fontSize * 1.2
	w, h := text.Measure(textStr, face, lineSpacing)
	return w, h
}

// LineHeight returns the height of a single line of text.
func (tr *TextRenderer) LineHeight() float64 {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	return tr.fontSize * 1.2 // Standard line height multiplier
}
