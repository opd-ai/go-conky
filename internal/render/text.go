package render

import (
	"image/color"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	etext "github.com/hajimehoshi/ebiten/v2/text/v2"
)

// defaultFontSize is the default font size in points.
const defaultFontSize = 14.0

// defaultFontFamily is the default font family name.
const defaultFontFamily = "GoMono"

// TextRenderer handles text rendering using Ebiten's text package.
// It supports multiple fonts through a FontManager and font style variations.
type TextRenderer struct {
	fontManager *FontManager
	fontFamily  string
	fontStyle   FontStyle
	fontSize    float64
	mu          sync.RWMutex
}

// NewTextRenderer creates a new TextRenderer with the default monospace font.
func NewTextRenderer() *TextRenderer {
	return &TextRenderer{
		fontManager: NewFontManager(),
		fontFamily:  defaultFontFamily,
		fontStyle:   FontStyleRegular,
		fontSize:    defaultFontSize,
	}
}

// NewTextRendererWithManager creates a TextRenderer using a shared FontManager.
// This allows multiple renderers to share font resources.
func NewTextRendererWithManager(fm *FontManager) *TextRenderer {
	return &TextRenderer{
		fontManager: fm,
		fontFamily:  fm.DefaultFamily(),
		fontStyle:   FontStyleRegular,
		fontSize:    defaultFontSize,
	}
}

// SetFontSize sets the font size for text rendering.
// If a non-positive size is provided, the font size is reset to defaultFontSize.
func (tr *TextRenderer) SetFontSize(size float64) {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	if size <= 0 {
		tr.fontSize = defaultFontSize
		return
	}
	tr.fontSize = size
}

// FontSize returns the current font size.
func (tr *TextRenderer) FontSize() float64 {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	return tr.fontSize
}

// SetFont sets the font family and style for text rendering.
// If the family doesn't exist, the default family is used.
func (tr *TextRenderer) SetFont(family string, style FontStyle) {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	tr.fontFamily = family
	tr.fontStyle = style
}

// SetFontFamily sets the font family for text rendering.
func (tr *TextRenderer) SetFontFamily(family string) {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	tr.fontFamily = family
}

// FontFamily returns the current font family name.
func (tr *TextRenderer) FontFamily() string {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	return tr.fontFamily
}

// SetFontStyle sets the font style (regular, bold, italic, bold-italic).
func (tr *TextRenderer) SetFontStyle(style FontStyle) {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	tr.fontStyle = style
}

// FontStyle returns the current font style.
func (tr *TextRenderer) FontStyle() FontStyle {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	return tr.fontStyle
}

// FontManager returns the underlying FontManager for advanced font operations.
func (tr *TextRenderer) FontManager() *FontManager {
	return tr.fontManager
}

// LoadFontFromFile loads a font from a file and registers it with the manager.
func (tr *TextRenderer) LoadFontFromFile(familyName string, style FontStyle, filePath string) error {
	return tr.fontManager.LoadFontFromFile(familyName, style, filePath)
}

// currentFontSource returns the current font source based on family and style.
// Must be called with mu held (at least for read).
func (tr *TextRenderer) currentFontSource() *etext.GoTextFaceSource {
	return tr.fontManager.GetFontWithFallback(tr.fontFamily, tr.fontStyle)
}

// DrawText renders text at the specified position with the given color.
func (tr *TextRenderer) DrawText(screen *ebiten.Image, textStr string, x, y float64, clr color.RGBA) {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	source := tr.currentFontSource()
	if source == nil {
		return // No font available
	}

	face := &etext.GoTextFace{
		Source: source,
		Size:   tr.fontSize,
	}

	op := &etext.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(clr)

	etext.Draw(screen, textStr, face, op)
}

// DrawTextWithStyle renders text with a specific style override.
func (tr *TextRenderer) DrawTextWithStyle(screen *ebiten.Image, textStr string, x, y float64, clr color.RGBA, style FontStyle) {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	source := tr.fontManager.GetFontWithFallback(tr.fontFamily, style)
	if source == nil {
		return // No font available
	}

	face := &etext.GoTextFace{
		Source: source,
		Size:   tr.fontSize,
	}

	op := &etext.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(clr)

	etext.Draw(screen, textStr, face, op)
}

// MeasureText returns the width and height of the given text string.
func (tr *TextRenderer) MeasureText(textStr string) (width, height float64) {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	source := tr.currentFontSource()
	if source == nil {
		return 0, 0
	}

	face := &etext.GoTextFace{
		Source: source,
		Size:   tr.fontSize,
	}

	// lineSpacingInPixels should be the line height for proper text measurement
	lineSpacing := tr.fontSize * 1.2
	w, h := etext.Measure(textStr, face, lineSpacing)
	return w, h
}

// LineHeight returns the height of a single line of text.
func (tr *TextRenderer) LineHeight() float64 {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	return tr.fontSize * 1.2 // Standard line height multiplier
}
