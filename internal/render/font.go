// Package render provides Ebiten-based rendering capabilities for conky-go.
package render

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"sync"

	etext "github.com/hajimehoshi/ebiten/v2/text/v2"

	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gobolditalic"
	"golang.org/x/image/font/gofont/goitalic"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/gofont/gomonobolditalic"
	"golang.org/x/image/font/gofont/gomonoitalic"
	"golang.org/x/image/font/gofont/goregular"
)

// FontStyle represents font style variations.
type FontStyle int

const (
	// FontStyleRegular is the regular/normal font style.
	FontStyleRegular FontStyle = iota
	// FontStyleBold is the bold font style.
	FontStyleBold
	// FontStyleItalic is the italic font style.
	FontStyleItalic
	// FontStyleBoldItalic is the bold and italic font style.
	FontStyleBoldItalic
)

// String returns the string representation of a FontStyle.
func (fs FontStyle) String() string {
	switch fs {
	case FontStyleRegular:
		return "regular"
	case FontStyleBold:
		return "bold"
	case FontStyleItalic:
		return "italic"
	case FontStyleBoldItalic:
		return "bold-italic"
	default:
		return "unknown"
	}
}

// ParseFontStyle parses a string into a FontStyle.
func ParseFontStyle(s string) (FontStyle, error) {
	switch s {
	case "regular", "normal", "":
		return FontStyleRegular, nil
	case "bold":
		return FontStyleBold, nil
	case "italic":
		return FontStyleItalic, nil
	case "bold-italic", "bolditalic", "bold_italic":
		return FontStyleBoldItalic, nil
	default:
		return FontStyleRegular, fmt.Errorf("unknown font style: %s", s)
	}
}

// FontFamily represents a font family with multiple style variations.
type FontFamily struct {
	name  string
	fonts map[FontStyle]*etext.GoTextFaceSource
	mu    sync.RWMutex
}

// NewFontFamily creates a new FontFamily with the given name.
func NewFontFamily(name string) *FontFamily {
	return &FontFamily{
		name:  name,
		fonts: make(map[FontStyle]*etext.GoTextFaceSource),
	}
}

// Name returns the family name.
func (ff *FontFamily) Name() string {
	return ff.name
}

// AddFont adds a font source to the family for a specific style.
func (ff *FontFamily) AddFont(style FontStyle, source *etext.GoTextFaceSource) {
	ff.mu.Lock()
	defer ff.mu.Unlock()
	ff.fonts[style] = source
}

// GetFont returns the font source for a specific style, or nil if not found.
func (ff *FontFamily) GetFont(style FontStyle) *etext.GoTextFaceSource {
	ff.mu.RLock()
	defer ff.mu.RUnlock()
	return ff.fonts[style]
}

// GetFontWithFallback returns the font source for a style with fallback logic.
// If the requested style is not available, it falls back to regular style.
func (ff *FontFamily) GetFontWithFallback(style FontStyle) *etext.GoTextFaceSource {
	ff.mu.RLock()
	defer ff.mu.RUnlock()

	if source, ok := ff.fonts[style]; ok {
		return source
	}

	// Fallback chain: try to find closest match
	switch style {
	case FontStyleBoldItalic:
		if source, ok := ff.fonts[FontStyleBold]; ok {
			return source
		}
		if source, ok := ff.fonts[FontStyleItalic]; ok {
			return source
		}
	case FontStyleBold, FontStyleItalic:
		// Fall through to regular
	}

	// Final fallback to regular
	if source, ok := ff.fonts[FontStyleRegular]; ok {
		return source
	}

	// Return font in deterministic order: Bold > Italic > BoldItalic
	// This ensures consistent behavior across runs when regular is unavailable.
	for _, fallbackStyle := range []FontStyle{FontStyleBold, FontStyleItalic, FontStyleBoldItalic} {
		if source, ok := ff.fonts[fallbackStyle]; ok {
			return source
		}
	}

	return nil
}

// HasStyle returns true if the family has a font for the given style.
func (ff *FontFamily) HasStyle(style FontStyle) bool {
	ff.mu.RLock()
	defer ff.mu.RUnlock()
	_, ok := ff.fonts[style]
	return ok
}

// AvailableStyles returns a sorted list of available font styles.
// The result is sorted for consistent ordering across calls.
func (ff *FontFamily) AvailableStyles() []FontStyle {
	ff.mu.RLock()
	defer ff.mu.RUnlock()

	styles := make([]FontStyle, 0, len(ff.fonts))
	// Iterate in defined order for deterministic results
	for _, style := range []FontStyle{FontStyleRegular, FontStyleBold, FontStyleItalic, FontStyleBoldItalic} {
		if _, ok := ff.fonts[style]; ok {
			styles = append(styles, style)
		}
	}
	return styles
}

// FontManager manages font loading, caching, and fallback chains.
type FontManager struct {
	families      map[string]*FontFamily
	fallbackChain []string
	defaultFamily string
	mu            sync.RWMutex
}

// NewFontManager creates a new FontManager with embedded Go fonts.
func NewFontManager() *FontManager {
	fm := &FontManager{
		families:      make(map[string]*FontFamily),
		fallbackChain: []string{},
		defaultFamily: "GoMono",
	}

	// Load embedded Go fonts
	fm.loadEmbeddedFonts()

	return fm
}

// loadEmbeddedFonts loads the embedded Go fonts into the manager.
func (fm *FontManager) loadEmbeddedFonts() {
	// Load Go Mono family
	goMonoFamily := NewFontFamily("GoMono")
	fm.loadEmbeddedFont(goMonoFamily, FontStyleRegular, gomono.TTF)
	fm.loadEmbeddedFont(goMonoFamily, FontStyleBold, gomonobold.TTF)
	fm.loadEmbeddedFont(goMonoFamily, FontStyleItalic, gomonoitalic.TTF)
	fm.loadEmbeddedFont(goMonoFamily, FontStyleBoldItalic, gomonobolditalic.TTF)
	fm.families["GoMono"] = goMonoFamily
	fm.families["gomono"] = goMonoFamily // lowercase alias

	// Load Go Sans (regular) family
	goSansFamily := NewFontFamily("GoSans")
	fm.loadEmbeddedFont(goSansFamily, FontStyleRegular, goregular.TTF)
	fm.loadEmbeddedFont(goSansFamily, FontStyleBold, gobold.TTF)
	fm.loadEmbeddedFont(goSansFamily, FontStyleItalic, goitalic.TTF)
	fm.loadEmbeddedFont(goSansFamily, FontStyleBoldItalic, gobolditalic.TTF)
	fm.families["GoSans"] = goSansFamily
	fm.families["gosans"] = goSansFamily // lowercase alias
	fm.families["Go"] = goSansFamily     // alias

	// Set default fallback chain
	fm.fallbackChain = []string{"GoMono", "GoSans"}
}

// loadEmbeddedFont loads an embedded font from byte data.
// Failures are silently ignored since embedded fonts should always be valid.
func (fm *FontManager) loadEmbeddedFont(family *FontFamily, style FontStyle, data []byte) {
	source, err := etext.NewGoTextFaceSource(bytes.NewReader(data))
	if err != nil {
		// Silently ignore errors for embedded fonts - they should always work
		return
	}
	family.AddFont(style, source)
}

// LoadFontFromFile loads a font from a file path and registers it with the given family and style.
func (fm *FontManager) LoadFontFromFile(familyName string, style FontStyle, filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read font file %s: %w", filePath, err)
	}

	return fm.LoadFontFromData(familyName, style, data)
}

// LoadFontFromData loads a font from byte data and registers it.
func (fm *FontManager) LoadFontFromData(familyName string, style FontStyle, data []byte) error {
	source, err := etext.NewGoTextFaceSource(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to parse font data: %w", err)
	}

	fm.mu.Lock()
	defer fm.mu.Unlock()

	family, ok := fm.families[familyName]
	if !ok {
		family = NewFontFamily(familyName)
		fm.families[familyName] = family
	}

	family.AddFont(style, source)
	return nil
}

// GetFamily returns a font family by name, or nil if not found.
func (fm *FontManager) GetFamily(name string) *FontFamily {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	return fm.families[name]
}

// GetFont returns the font source for a family and style.
// Returns nil if the family doesn't exist or if the family has no fonts
// available (even after style fallback within the family).
func (fm *FontManager) GetFont(familyName string, style FontStyle) *etext.GoTextFaceSource {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	family, ok := fm.families[familyName]
	if !ok {
		return nil
	}

	return family.GetFontWithFallback(style)
}

// GetFontWithFallback returns a font source, falling back through the chain if needed.
func (fm *FontManager) GetFontWithFallback(familyName string, style FontStyle) *etext.GoTextFaceSource {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	// Try the requested family first
	if family, ok := fm.families[familyName]; ok {
		if source := family.GetFontWithFallback(style); source != nil {
			return source
		}
	}

	// Try fallback chain
	for _, fallbackName := range fm.fallbackChain {
		if family, ok := fm.families[fallbackName]; ok {
			if source := family.GetFontWithFallback(style); source != nil {
				return source
			}
		}
	}

	// Last resort: try default family
	if family, ok := fm.families[fm.defaultFamily]; ok {
		return family.GetFontWithFallback(style)
	}

	return nil
}

// SetFallbackChain sets the font family fallback chain.
func (fm *FontManager) SetFallbackChain(families []string) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fm.fallbackChain = make([]string, len(families))
	copy(fm.fallbackChain, families)
}

// SetDefaultFamily sets the default font family name.
func (fm *FontManager) SetDefaultFamily(familyName string) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fm.defaultFamily = familyName
}

// DefaultFamily returns the default font family name.
func (fm *FontManager) DefaultFamily() string {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	return fm.defaultFamily
}

// FallbackChain returns a copy of the current fallback chain.
func (fm *FontManager) FallbackChain() []string {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	chain := make([]string, len(fm.fallbackChain))
	copy(chain, fm.fallbackChain)
	return chain
}

// ListFamilies returns a list of canonical font family names.
// Aliases are excluded; only the primary name of each family is returned.
// The result is sorted for consistent ordering.
func (fm *FontManager) ListFamilies() []string {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	seen := make(map[string]bool)
	var families []string
	for _, family := range fm.families {
		// Only add the canonical name, not aliases
		canonicalName := family.Name()
		if !seen[canonicalName] {
			seen[canonicalName] = true
			families = append(families, canonicalName)
		}
	}
	// Sort for deterministic ordering
	sort.Strings(families)
	return families
}

// RegisterAlias registers an alias name for an existing family.
func (fm *FontManager) RegisterAlias(alias, familyName string) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	family, ok := fm.families[familyName]
	if !ok {
		return fmt.Errorf("font family %s not found", familyName)
	}

	fm.families[alias] = family
	return nil
}
