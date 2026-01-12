// Package render provides Ebiten-based rendering capabilities for conky-go.
// This file implements color management and transparency handling utilities
// for parsing, converting, and manipulating colors in various formats.
package render

import (
	"fmt"
	"image/color"
	"math"
	"strconv"
	"strings"
)

// NamedColors maps CSS color names to their RGBA values.
// This provides compatibility with Conky configurations that use named colors.
var NamedColors = map[string]color.RGBA{
	// Basic colors
	"black":   {R: 0, G: 0, B: 0, A: 255},
	"white":   {R: 255, G: 255, B: 255, A: 255},
	"red":     {R: 255, G: 0, B: 0, A: 255},
	"green":   {R: 0, G: 128, B: 0, A: 255},
	"blue":    {R: 0, G: 0, B: 255, A: 255},
	"yellow":  {R: 255, G: 255, B: 0, A: 255},
	"cyan":    {R: 0, G: 255, B: 255, A: 255},
	"magenta": {R: 255, G: 0, B: 255, A: 255},

	// Extended colors
	"gray":       {R: 128, G: 128, B: 128, A: 255},
	"grey":       {R: 128, G: 128, B: 128, A: 255},
	"silver":     {R: 192, G: 192, B: 192, A: 255},
	"maroon":     {R: 128, G: 0, B: 0, A: 255},
	"olive":      {R: 128, G: 128, B: 0, A: 255},
	"lime":       {R: 0, G: 255, B: 0, A: 255},
	"aqua":       {R: 0, G: 255, B: 255, A: 255},
	"teal":       {R: 0, G: 128, B: 128, A: 255},
	"navy":       {R: 0, G: 0, B: 128, A: 255},
	"fuchsia":    {R: 255, G: 0, B: 255, A: 255},
	"purple":     {R: 128, G: 0, B: 128, A: 255},
	"orange":     {R: 255, G: 165, B: 0, A: 255},
	"pink":       {R: 255, G: 192, B: 203, A: 255},
	"brown":      {R: 165, G: 42, B: 42, A: 255},
	"coral":      {R: 255, G: 127, B: 80, A: 255},
	"gold":       {R: 255, G: 215, B: 0, A: 255},
	"indigo":     {R: 75, G: 0, B: 130, A: 255},
	"violet":     {R: 238, G: 130, B: 238, A: 255},
	"turquoise":  {R: 64, G: 224, B: 208, A: 255},
	"salmon":     {R: 250, G: 128, B: 114, A: 255},
	"khaki":      {R: 240, G: 230, B: 140, A: 255},
	"lavender":   {R: 230, G: 230, B: 250, A: 255},
	"beige":      {R: 245, G: 245, B: 220, A: 255},
	"ivory":      {R: 255, G: 255, B: 240, A: 255},
	"chocolate":  {R: 210, G: 105, B: 30, A: 255},
	"crimson":    {R: 220, G: 20, B: 60, A: 255},
	"darkblue":   {R: 0, G: 0, B: 139, A: 255},
	"darkgreen":  {R: 0, G: 100, B: 0, A: 255},
	"darkred":    {R: 139, G: 0, B: 0, A: 255},
	"darkorange": {R: 255, G: 140, B: 0, A: 255},
	"lightblue":  {R: 173, G: 216, B: 230, A: 255},
	"lightgreen": {R: 144, G: 238, B: 144, A: 255},
	"lightgray":  {R: 211, G: 211, B: 211, A: 255},
	"lightgrey":  {R: 211, G: 211, B: 211, A: 255},
	"darkgray":   {R: 169, G: 169, B: 169, A: 255},
	"darkgrey":   {R: 169, G: 169, B: 169, A: 255},

	// Transparent
	"transparent": {R: 0, G: 0, B: 0, A: 0},
}

// ParseColor parses a color string and returns an RGBA color.
// Supported formats:
//   - Named colors: "red", "blue", "green", etc.
//   - Hex formats: "#RGB", "#RGBA", "#RRGGBB", "#RRGGBBAA"
//   - Hex without #: "RGB", "RGBA", "RRGGBB", "RRGGBBAA"
//   - RGB function: "rgb(255, 0, 0)"
//   - RGBA function: "rgba(255, 0, 0, 0.5)" or "rgba(255, 0, 0, 128)"
//
// Returns an error if the color string cannot be parsed.
func ParseColor(s string) (color.RGBA, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return color.RGBA{}, fmt.Errorf("empty color string")
	}

	// Try named color first (case-insensitive)
	if clr, ok := NamedColors[strings.ToLower(s)]; ok {
		return clr, nil
	}

	// Try hex format
	if strings.HasPrefix(s, "#") || isHexString(s) {
		return parseHexColor(s)
	}

	// Try rgb/rgba function format
	lower := strings.ToLower(s)
	if strings.HasPrefix(lower, "rgba(") {
		return parseRGBAFunc(s)
	}
	if strings.HasPrefix(lower, "rgb(") {
		return parseRGBFunc(s)
	}

	return color.RGBA{}, fmt.Errorf("unrecognized color format: %q", s)
}

// MustParseColor parses a color string and panics if parsing fails.
// Use this only for known-good color values in initialization code.
func MustParseColor(s string) color.RGBA {
	c, err := ParseColor(s)
	if err != nil {
		panic(err)
	}
	return c
}

// isHexString checks if the string looks like a hex color (without #).
func isHexString(s string) bool {
	if len(s) != 3 && len(s) != 4 && len(s) != 6 && len(s) != 8 {
		return false
	}
	for _, c := range s {
		if !isHexDigit(c) {
			return false
		}
	}
	return true
}

// isHexDigit checks if a rune is a valid hexadecimal digit.
func isHexDigit(c rune) bool {
	return (c >= '0' && c <= '9') ||
		(c >= 'a' && c <= 'f') ||
		(c >= 'A' && c <= 'F')
}

// parseHexColor parses a hex color string.
func parseHexColor(s string) (color.RGBA, error) {
	// Remove # prefix if present
	s = strings.TrimPrefix(s, "#")

	switch len(s) {
	case 3: // RGB shorthand
		r, err := parseHexByte(s[0:1] + s[0:1])
		if err != nil {
			return color.RGBA{}, fmt.Errorf("invalid red component: %w", err)
		}
		g, err := parseHexByte(s[1:2] + s[1:2])
		if err != nil {
			return color.RGBA{}, fmt.Errorf("invalid green component: %w", err)
		}
		b, err := parseHexByte(s[2:3] + s[2:3])
		if err != nil {
			return color.RGBA{}, fmt.Errorf("invalid blue component: %w", err)
		}
		return color.RGBA{R: r, G: g, B: b, A: 255}, nil

	case 4: // RGBA shorthand
		r, err := parseHexByte(s[0:1] + s[0:1])
		if err != nil {
			return color.RGBA{}, fmt.Errorf("invalid red component: %w", err)
		}
		g, err := parseHexByte(s[1:2] + s[1:2])
		if err != nil {
			return color.RGBA{}, fmt.Errorf("invalid green component: %w", err)
		}
		b, err := parseHexByte(s[2:3] + s[2:3])
		if err != nil {
			return color.RGBA{}, fmt.Errorf("invalid blue component: %w", err)
		}
		a, err := parseHexByte(s[3:4] + s[3:4])
		if err != nil {
			return color.RGBA{}, fmt.Errorf("invalid alpha component: %w", err)
		}
		return color.RGBA{R: r, G: g, B: b, A: a}, nil

	case 6: // RRGGBB
		r, err := parseHexByte(s[0:2])
		if err != nil {
			return color.RGBA{}, fmt.Errorf("invalid red component: %w", err)
		}
		g, err := parseHexByte(s[2:4])
		if err != nil {
			return color.RGBA{}, fmt.Errorf("invalid green component: %w", err)
		}
		b, err := parseHexByte(s[4:6])
		if err != nil {
			return color.RGBA{}, fmt.Errorf("invalid blue component: %w", err)
		}
		return color.RGBA{R: r, G: g, B: b, A: 255}, nil

	case 8: // RRGGBBAA
		r, err := parseHexByte(s[0:2])
		if err != nil {
			return color.RGBA{}, fmt.Errorf("invalid red component: %w", err)
		}
		g, err := parseHexByte(s[2:4])
		if err != nil {
			return color.RGBA{}, fmt.Errorf("invalid green component: %w", err)
		}
		b, err := parseHexByte(s[4:6])
		if err != nil {
			return color.RGBA{}, fmt.Errorf("invalid blue component: %w", err)
		}
		a, err := parseHexByte(s[6:8])
		if err != nil {
			return color.RGBA{}, fmt.Errorf("invalid alpha component: %w", err)
		}
		return color.RGBA{R: r, G: g, B: b, A: a}, nil

	default:
		return color.RGBA{}, fmt.Errorf("invalid hex color length: %d", len(s))
	}
}

// parseHexByte parses a two-character hex string to a byte.
func parseHexByte(s string) (uint8, error) {
	val, err := strconv.ParseUint(s, 16, 8)
	if err != nil {
		return 0, err
	}
	return uint8(val), nil
}

// parseRGBFunc parses an "rgb(r, g, b)" format string.
func parseRGBFunc(s string) (color.RGBA, error) {
	s = strings.TrimSpace(s)
	lower := strings.ToLower(s)

	if !strings.HasPrefix(lower, "rgb(") || !strings.HasSuffix(s, ")") {
		return color.RGBA{}, fmt.Errorf("invalid rgb() format: %q", s)
	}

	// Extract content between parentheses
	content := s[4 : len(s)-1]
	parts := strings.Split(content, ",")
	if len(parts) != 3 {
		return color.RGBA{}, fmt.Errorf("rgb() requires exactly 3 values, got %d", len(parts))
	}

	r, err := parseColorComponent(strings.TrimSpace(parts[0]))
	if err != nil {
		return color.RGBA{}, fmt.Errorf("invalid red value: %w", err)
	}
	g, err := parseColorComponent(strings.TrimSpace(parts[1]))
	if err != nil {
		return color.RGBA{}, fmt.Errorf("invalid green value: %w", err)
	}
	b, err := parseColorComponent(strings.TrimSpace(parts[2]))
	if err != nil {
		return color.RGBA{}, fmt.Errorf("invalid blue value: %w", err)
	}

	return color.RGBA{R: r, G: g, B: b, A: 255}, nil
}

// parseRGBAFunc parses an "rgba(r, g, b, a)" format string.
func parseRGBAFunc(s string) (color.RGBA, error) {
	s = strings.TrimSpace(s)
	lower := strings.ToLower(s)

	if !strings.HasPrefix(lower, "rgba(") || !strings.HasSuffix(s, ")") {
		return color.RGBA{}, fmt.Errorf("invalid rgba() format: %q", s)
	}

	// Extract content between parentheses
	content := s[5 : len(s)-1]
	parts := strings.Split(content, ",")
	if len(parts) != 4 {
		return color.RGBA{}, fmt.Errorf("rgba() requires exactly 4 values, got %d", len(parts))
	}

	r, err := parseColorComponent(strings.TrimSpace(parts[0]))
	if err != nil {
		return color.RGBA{}, fmt.Errorf("invalid red value: %w", err)
	}
	g, err := parseColorComponent(strings.TrimSpace(parts[1]))
	if err != nil {
		return color.RGBA{}, fmt.Errorf("invalid green value: %w", err)
	}
	b, err := parseColorComponent(strings.TrimSpace(parts[2]))
	if err != nil {
		return color.RGBA{}, fmt.Errorf("invalid blue value: %w", err)
	}
	a, err := parseAlphaComponent(strings.TrimSpace(parts[3]))
	if err != nil {
		return color.RGBA{}, fmt.Errorf("invalid alpha value: %w", err)
	}

	return color.RGBA{R: r, G: g, B: b, A: a}, nil
}

// parseColorComponent parses a color component value (0-255).
func parseColorComponent(s string) (uint8, error) {
	val, err := strconv.ParseUint(s, 10, 8)
	if err != nil {
		return 0, err
	}
	return uint8(val), nil
}

// parseAlphaComponent parses an alpha value.
// Accepts both 0-255 integer and 0.0-1.0 float formats.
func parseAlphaComponent(s string) (uint8, error) {
	// Try float first (0.0 - 1.0)
	if strings.Contains(s, ".") {
		val, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, err
		}
		if val < 0 {
			val = 0
		}
		if val > 1 {
			val = 1
		}
		return uint8(val * 255), nil
	}

	// Try integer (0-255)
	val, err := strconv.ParseUint(s, 10, 8)
	if err != nil {
		return 0, err
	}
	return uint8(val), nil
}

// ToHex converts a color to a hex string with # prefix.
// Format: #RRGGBB or #RRGGBBAA if alpha is not 255.
func ToHex(c color.RGBA) string {
	if c.A == 255 {
		return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B)
	}
	return fmt.Sprintf("#%02X%02X%02X%02X", c.R, c.G, c.B, c.A)
}

// ToRGBA converts a color to an "rgba(r, g, b, a)" string.
func ToRGBA(c color.RGBA) string {
	alpha := float64(c.A) / 255.0
	return fmt.Sprintf("rgba(%d, %d, %d, %.2f)", c.R, c.G, c.B, alpha)
}

// WithAlpha returns a new color with the specified alpha value (0-255).
func WithAlpha(c color.RGBA, alpha uint8) color.RGBA {
	return color.RGBA{R: c.R, G: c.G, B: c.B, A: alpha}
}

// WithOpacity returns a new color with the specified opacity (0.0-1.0).
func WithOpacity(c color.RGBA, opacity float64) color.RGBA {
	if opacity < 0 {
		opacity = 0
	}
	if opacity > 1 {
		opacity = 1
	}
	return color.RGBA{R: c.R, G: c.G, B: c.B, A: uint8(opacity * 255)}
}

// Blend blends two colors together with the specified ratio (0.0-1.0).
// A ratio of 0.0 returns c1, 1.0 returns c2, 0.5 returns an even mix.
func Blend(c1, c2 color.RGBA, ratio float64) color.RGBA {
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}

	return color.RGBA{
		R: blendChannel(c1.R, c2.R, ratio),
		G: blendChannel(c1.G, c2.G, ratio),
		B: blendChannel(c1.B, c2.B, ratio),
		A: blendChannel(c1.A, c2.A, ratio),
	}
}

// blendChannel blends two channel values with the given ratio.
func blendChannel(a, b uint8, ratio float64) uint8 {
	return uint8(float64(a)*(1-ratio) + float64(b)*ratio)
}

// Lighten returns a lighter version of the color.
// Amount is a value from 0.0-1.0, where 0.0 returns the original color
// and 1.0 returns white.
func Lighten(c color.RGBA, amount float64) color.RGBA {
	if amount < 0 {
		amount = 0
	}
	if amount > 1 {
		amount = 1
	}

	return color.RGBA{
		R: uint8(float64(c.R) + (255-float64(c.R))*amount),
		G: uint8(float64(c.G) + (255-float64(c.G))*amount),
		B: uint8(float64(c.B) + (255-float64(c.B))*amount),
		A: c.A,
	}
}

// Darken returns a darker version of the color.
// Amount is a value from 0.0-1.0, where 0.0 returns the original color
// and 1.0 returns black.
func Darken(c color.RGBA, amount float64) color.RGBA {
	if amount < 0 {
		amount = 0
	}
	if amount > 1 {
		amount = 1
	}

	return color.RGBA{
		R: uint8(float64(c.R) * (1 - amount)),
		G: uint8(float64(c.G) * (1 - amount)),
		B: uint8(float64(c.B) * (1 - amount)),
		A: c.A,
	}
}

// Invert returns the inverted (complementary) color.
func Invert(c color.RGBA) color.RGBA {
	return color.RGBA{
		R: 255 - c.R,
		G: 255 - c.G,
		B: 255 - c.B,
		A: c.A,
	}
}

// Grayscale converts a color to grayscale using luminance weights.
func Grayscale(c color.RGBA) color.RGBA {
	// Use standard luminance formula
	lum := uint8(0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B))
	return color.RGBA{R: lum, G: lum, B: lum, A: c.A}
}

// HSL represents a color in Hue-Saturation-Lightness space.
// H is in range [0, 360), S and L are in range [0, 1].
type HSL struct {
	H, S, L float64
}

// RGBAToHSL converts an RGBA color to HSL.
func RGBAToHSL(c color.RGBA) HSL {
	r := float64(c.R) / 255.0
	g := float64(c.G) / 255.0
	b := float64(c.B) / 255.0

	maxVal := math.Max(r, math.Max(g, b))
	minVal := math.Min(r, math.Min(g, b))
	l := (maxVal + minVal) / 2

	var h, s float64

	if maxVal == minVal {
		// Achromatic
		h, s = 0, 0
	} else {
		d := maxVal - minVal
		if l > 0.5 {
			s = d / (2 - maxVal - minVal)
		} else {
			s = d / (maxVal + minVal)
		}

		switch maxVal {
		case r:
			h = (g - b) / d
			if g < b {
				h += 6
			}
		case g:
			h = (b-r)/d + 2
		case b:
			h = (r-g)/d + 4
		}
		h *= 60
	}

	return HSL{H: h, S: s, L: l}
}

// ToRGBA converts an HSL color to RGBA with full opacity.
func (hsl HSL) ToRGBA() color.RGBA {
	return HSLToRGBA(hsl, 255)
}

// HSLToRGBA converts an HSL color to RGBA with the specified alpha.
func HSLToRGBA(hsl HSL, alpha uint8) color.RGBA {
	if hsl.S == 0 {
		// Achromatic
		l := uint8(hsl.L * 255)
		return color.RGBA{R: l, G: l, B: l, A: alpha}
	}

	var q float64
	if hsl.L < 0.5 {
		q = hsl.L * (1 + hsl.S)
	} else {
		q = hsl.L + hsl.S - hsl.L*hsl.S
	}
	p := 2*hsl.L - q

	hNorm := hsl.H / 360.0

	r := hueToRGB(p, q, hNorm+1.0/3.0)
	g := hueToRGB(p, q, hNorm)
	b := hueToRGB(p, q, hNorm-1.0/3.0)

	return color.RGBA{
		R: uint8(r * 255),
		G: uint8(g * 255),
		B: uint8(b * 255),
		A: alpha,
	}
}

// hueToRGB is a helper for HSL to RGB conversion.
func hueToRGB(p, q, t float64) float64 {
	if t < 0 {
		t++
	}
	if t > 1 {
		t--
	}
	if t < 1.0/6.0 {
		return p + (q-p)*6*t
	}
	if t < 0.5 {
		return q
	}
	if t < 2.0/3.0 {
		return p + (q-p)*(2.0/3.0-t)*6
	}
	return p
}

// AdjustHue rotates the hue by the specified degrees.
func AdjustHue(c color.RGBA, degrees float64) color.RGBA {
	hsl := RGBAToHSL(c)
	hsl.H = math.Mod(hsl.H+degrees, 360)
	if hsl.H < 0 {
		hsl.H += 360
	}
	result := hsl.ToRGBA()
	result.A = c.A
	return result
}

// Saturate increases the saturation of a color.
// Amount is a value from 0.0-1.0 representing how much to increase.
func Saturate(c color.RGBA, amount float64) color.RGBA {
	if amount < 0 {
		amount = 0
	}
	if amount > 1 {
		amount = 1
	}

	hsl := RGBAToHSL(c)
	hsl.S += (1 - hsl.S) * amount
	if hsl.S > 1 {
		hsl.S = 1
	}
	result := hsl.ToRGBA()
	result.A = c.A
	return result
}

// Desaturate decreases the saturation of a color.
// Amount is a value from 0.0-1.0 representing how much to decrease.
func Desaturate(c color.RGBA, amount float64) color.RGBA {
	if amount < 0 {
		amount = 0
	}
	if amount > 1 {
		amount = 1
	}

	hsl := RGBAToHSL(c)
	hsl.S *= (1 - amount)
	if hsl.S < 0 {
		hsl.S = 0
	}
	result := hsl.ToRGBA()
	result.A = c.A
	return result
}

// AlphaBlend composites a foreground color over a background color
// using standard alpha blending (Porter-Duff "over" operator).
func AlphaBlend(bg, fg color.RGBA) color.RGBA {
	// Convert alpha to 0-1 range
	fgA := float64(fg.A) / 255.0
	bgA := float64(bg.A) / 255.0

	// Calculate output alpha
	outA := fgA + bgA*(1-fgA)
	if outA == 0 {
		return color.RGBA{}
	}

	// Calculate output colors
	outR := (float64(fg.R)*fgA + float64(bg.R)*bgA*(1-fgA)) / outA
	outG := (float64(fg.G)*fgA + float64(bg.G)*bgA*(1-fgA)) / outA
	outB := (float64(fg.B)*fgA + float64(bg.B)*bgA*(1-fgA)) / outA

	return color.RGBA{
		R: uint8(outR),
		G: uint8(outG),
		B: uint8(outB),
		A: uint8(outA * 255),
	}
}

// Luminance returns the relative luminance of a color (0.0-1.0).
// This is useful for determining if a color is "light" or "dark".
func Luminance(c color.RGBA) float64 {
	// sRGB to linear RGB conversion
	r := sRGBToLinear(float64(c.R) / 255.0)
	g := sRGBToLinear(float64(c.G) / 255.0)
	b := sRGBToLinear(float64(c.B) / 255.0)

	// ITU-R BT.709 coefficients
	return 0.2126*r + 0.7152*g + 0.0722*b
}

// sRGBToLinear converts an sRGB component to linear RGB.
func sRGBToLinear(c float64) float64 {
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

// ContrastRatio calculates the contrast ratio between two colors.
// Returns a value between 1.0 (no contrast) and 21.0 (maximum contrast).
// WCAG recommends at least 4.5:1 for normal text and 3:1 for large text.
func ContrastRatio(c1, c2 color.RGBA) float64 {
	l1 := Luminance(c1)
	l2 := Luminance(c2)

	lighter := math.Max(l1, l2)
	darker := math.Min(l1, l2)

	return (lighter + 0.05) / (darker + 0.05)
}

// IsLight returns true if the color is considered "light" (luminance > 0.5).
func IsLight(c color.RGBA) bool {
	return Luminance(c) > 0.5
}

// IsDark returns true if the color is considered "dark" (luminance <= 0.5).
func IsDark(c color.RGBA) bool {
	return Luminance(c) <= 0.5
}

// Gradient represents a color gradient for smooth color transitions.
type Gradient struct {
	stops []GradientStop
}

// GradientStop represents a color at a specific position in a gradient.
type GradientStop struct {
	Position float64    // Position in range [0, 1]
	Color    color.RGBA // Color at this position
}

// NewGradient creates a new gradient with the specified color stops.
// Stops should be sorted by position, but will be auto-sorted if not.
func NewGradient(stops ...GradientStop) *Gradient {
	g := &Gradient{
		stops: make([]GradientStop, len(stops)),
	}
	copy(g.stops, stops)

	// Sort stops by position
	for i := 1; i < len(g.stops); i++ {
		for j := i; j > 0 && g.stops[j].Position < g.stops[j-1].Position; j-- {
			g.stops[j], g.stops[j-1] = g.stops[j-1], g.stops[j]
		}
	}

	return g
}

// At returns the interpolated color at the specified position (0.0-1.0).
func (g *Gradient) At(position float64) color.RGBA {
	if len(g.stops) == 0 {
		return color.RGBA{}
	}
	if len(g.stops) == 1 {
		return g.stops[0].Color
	}

	if position <= 0 {
		return g.stops[0].Color
	}
	if position >= 1 {
		return g.stops[len(g.stops)-1].Color
	}

	// Find the two stops to interpolate between
	var i int
	for i = 1; i < len(g.stops); i++ {
		if g.stops[i].Position >= position {
			break
		}
	}

	// Interpolate between stops[i-1] and stops[i]
	stop1 := g.stops[i-1]
	stop2 := g.stops[i]

	// Calculate the ratio between the two stops
	ratio := (position - stop1.Position) / (stop2.Position - stop1.Position)

	return Blend(stop1.Color, stop2.Color, ratio)
}

// AddStop adds a color stop to the gradient.
func (g *Gradient) AddStop(position float64, clr color.RGBA) {
	stop := GradientStop{Position: position, Color: clr}
	g.stops = append(g.stops, stop)

	// Re-sort
	for i := len(g.stops) - 1; i > 0 && g.stops[i].Position < g.stops[i-1].Position; i-- {
		g.stops[i], g.stops[i-1] = g.stops[i-1], g.stops[i]
	}
}

// Stops returns the gradient stops.
func (g *Gradient) Stops() []GradientStop {
	result := make([]GradientStop, len(g.stops))
	copy(result, g.stops)
	return result
}
