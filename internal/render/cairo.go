// Package render provides Ebiten-based rendering capabilities for conky-go.
// This file implements the Cairo compatibility layer that translates
// Cairo drawing commands to Ebiten vector graphics.
package render

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// PatternType represents the type of Cairo pattern.
type PatternType int

const (
	// PatternTypeSolid is a solid color pattern.
	PatternTypeSolid PatternType = iota
	// PatternTypeLinear is a linear gradient pattern.
	PatternTypeLinear
	// PatternTypeRadial is a radial gradient pattern.
	PatternTypeRadial
	// PatternTypeSurface is a surface (image) pattern.
	PatternTypeSurface
)

// CairoFillRule represents the fill rule for fill operations.
type CairoFillRule int

const (
	// CairoFillRuleWinding is the non-zero winding rule.
	// A point is inside the shape if a ray from the point to infinity
	// crosses a non-zero sum of signed edge crossings.
	CairoFillRuleWinding CairoFillRule = 0
	// CairoFillRuleEvenOdd is the even-odd fill rule.
	// A point is inside the shape if a ray from the point to infinity
	// crosses an odd number of edges.
	CairoFillRuleEvenOdd CairoFillRule = 1
)

// CairoOperator represents the compositing operator.
type CairoOperator int

const (
	// CairoOperatorClear clears destination where source is drawn.
	CairoOperatorClear CairoOperator = 0
	// CairoOperatorSource replaces destination with source.
	CairoOperatorSource CairoOperator = 1
	// CairoOperatorOver draws source over destination (default).
	CairoOperatorOver CairoOperator = 2
	// CairoOperatorIn draws source where destination is opaque.
	CairoOperatorIn CairoOperator = 3
	// CairoOperatorOut draws source where destination is transparent.
	CairoOperatorOut CairoOperator = 4
	// CairoOperatorAtop draws source atop destination.
	CairoOperatorAtop CairoOperator = 5
	// CairoOperatorDest ignores source.
	CairoOperatorDest CairoOperator = 6
	// CairoOperatorDestOver draws destination over source.
	CairoOperatorDestOver CairoOperator = 7
	// CairoOperatorDestIn draws destination where source is opaque.
	CairoOperatorDestIn CairoOperator = 8
	// CairoOperatorDestOut draws destination where source is transparent.
	CairoOperatorDestOut CairoOperator = 9
	// CairoOperatorDestAtop draws destination atop source.
	CairoOperatorDestAtop CairoOperator = 10
	// CairoOperatorXor XORs source and destination.
	CairoOperatorXor CairoOperator = 11
	// CairoOperatorAdd adds source and destination.
	CairoOperatorAdd CairoOperator = 12
)

// ColorStop represents a color stop in a gradient.
type ColorStop struct {
	Offset float64
	Color  color.RGBA
}

// CairoPattern represents a Cairo pattern (solid color or gradient).
type CairoPattern struct {
	patternType PatternType
	// For solid patterns
	solidColor color.RGBA
	// For linear gradients: (x0, y0) to (x1, y1)
	x0, y0, x1, y1 float64
	// For radial gradients: (cx0, cy0, r0) to (cx1, cy1, r1)
	cx0, cy0, r0, cx1, cy1, r1 float64
	// Color stops for gradients
	colorStops []ColorStop
	// Extend mode for patterns
	extend PatternExtend
	// For surface patterns: the source surface/image
	surface *ebiten.Image
	mu      sync.Mutex
}

// NewSolidPattern creates a solid color pattern.
func NewSolidPattern(r, g, b, a float64) *CairoPattern {
	return &CairoPattern{
		patternType: PatternTypeSolid,
		solidColor: color.RGBA{
			R: clampToByte(r),
			G: clampToByte(g),
			B: clampToByte(b),
			A: clampToByte(a),
		},
	}
}

// NewLinearPattern creates a linear gradient pattern.
func NewLinearPattern(x0, y0, x1, y1 float64) *CairoPattern {
	return &CairoPattern{
		patternType: PatternTypeLinear,
		x0:          x0,
		y0:          y0,
		x1:          x1,
		y1:          y1,
		colorStops:  make([]ColorStop, 0),
	}
}

// NewRadialPattern creates a radial gradient pattern.
func NewRadialPattern(cx0, cy0, r0, cx1, cy1, r1 float64) *CairoPattern {
	return &CairoPattern{
		patternType: PatternTypeRadial,
		cx0:         cx0,
		cy0:         cy0,
		r0:          r0,
		cx1:         cx1,
		cy1:         cy1,
		r1:          r1,
		colorStops:  make([]ColorStop, 0),
	}
}

// NewSurfacePattern creates a surface (image) pattern.
// This is equivalent to cairo_pattern_create_for_surface.
func NewSurfacePattern(surface *CairoSurface) *CairoPattern {
	if surface == nil || surface.IsDestroyed() {
		return nil
	}
	return &CairoPattern{
		patternType: PatternTypeSurface,
		surface:     surface.Image(),
	}
}

// NewSurfacePatternFromImage creates a surface pattern from an Ebiten image.
func NewSurfacePatternFromImage(image *ebiten.Image) *CairoPattern {
	if image == nil {
		return nil
	}
	return &CairoPattern{
		patternType: PatternTypeSurface,
		surface:     image,
	}
}

// AddColorStopRGB adds an RGB color stop to a gradient pattern.
func (p *CairoPattern) AddColorStopRGB(offset, r, g, b float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.colorStops = append(p.colorStops, ColorStop{
		Offset: offset,
		Color: color.RGBA{
			R: clampToByte(r),
			G: clampToByte(g),
			B: clampToByte(b),
			A: 255,
		},
	})
}

// AddColorStopRGBA adds an RGBA color stop to a gradient pattern.
func (p *CairoPattern) AddColorStopRGBA(offset, r, g, b, a float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.colorStops = append(p.colorStops, ColorStop{
		Offset: offset,
		Color: color.RGBA{
			R: clampToByte(r),
			G: clampToByte(g),
			B: clampToByte(b),
			A: clampToByte(a),
		},
	})
}

// Type returns the pattern type.
func (p *CairoPattern) Type() PatternType {
	return p.patternType
}

// ColorAt returns the color at a given position (0.0 to 1.0) along the gradient.
// For solid patterns, returns the solid color.
func (p *CairoPattern) ColorAt(t float64) color.RGBA {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.patternType == PatternTypeSolid {
		return p.solidColor
	}

	if len(p.colorStops) == 0 {
		return color.RGBA{A: 255}
	}
	if len(p.colorStops) == 1 {
		return p.colorStops[0].Color
	}

	// Clamp t to [0, 1]
	if t <= 0 {
		return p.colorStops[0].Color
	}
	if t >= 1 {
		return p.colorStops[len(p.colorStops)-1].Color
	}

	// Find the two color stops that bracket t
	var before, after ColorStop
	before = p.colorStops[0]
	after = p.colorStops[len(p.colorStops)-1]

	for i := 0; i < len(p.colorStops)-1; i++ {
		if t >= p.colorStops[i].Offset && t <= p.colorStops[i+1].Offset {
			before = p.colorStops[i]
			after = p.colorStops[i+1]
			break
		}
	}

	// Interpolate between the two colors
	if after.Offset == before.Offset {
		return before.Color
	}
	ratio := (t - before.Offset) / (after.Offset - before.Offset)
	return color.RGBA{
		R: uint8(float64(before.Color.R) + ratio*(float64(after.Color.R)-float64(before.Color.R))),
		G: uint8(float64(before.Color.G) + ratio*(float64(after.Color.G)-float64(before.Color.G))),
		B: uint8(float64(before.Color.B) + ratio*(float64(after.Color.B)-float64(before.Color.B))),
		A: uint8(float64(before.Color.A) + ratio*(float64(after.Color.A)-float64(before.Color.A))),
	}
}

// ColorAtPoint returns the color at a given (x, y) point for gradient patterns.
func (p *CairoPattern) ColorAtPoint(x, y float64) color.RGBA {
	p.mu.Lock()
	patternType := p.patternType
	p.mu.Unlock()

	switch patternType {
	case PatternTypeSolid:
		return p.solidColor
	case PatternTypeLinear:
		return p.linearColorAt(x, y)
	case PatternTypeRadial:
		return p.radialColorAt(x, y)
	case PatternTypeSurface:
		return p.surfaceColorAt(x, y)
	default:
		return color.RGBA{A: 255}
	}
}

// linearColorAt calculates the color at a point for a linear gradient.
func (p *CairoPattern) linearColorAt(x, y float64) color.RGBA {
	// Calculate the projection of (x,y) onto the gradient line
	dx := p.x1 - p.x0
	dy := p.y1 - p.y0
	lengthSq := dx*dx + dy*dy
	if lengthSq == 0 {
		return p.ColorAt(0)
	}
	t := ((x-p.x0)*dx + (y-p.y0)*dy) / lengthSq
	return p.ColorAt(t)
}

// radialColorAt calculates the color at a point for a radial gradient.
func (p *CairoPattern) radialColorAt(x, y float64) color.RGBA {
	// Distance from center of outer circle
	dx := x - p.cx1
	dy := y - p.cy1
	dist := math.Sqrt(dx*dx + dy*dy)

	// Map distance to gradient position
	if p.r1 == p.r0 {
		if dist <= p.r1 {
			return p.ColorAt(1)
		}
		return p.ColorAt(0)
	}
	t := (dist - p.r0) / (p.r1 - p.r0)
	return p.ColorAt(t)
}

// surfaceColorAt returns the color at a point for a surface pattern.
// Note: Due to Ebiten limitations, this function may not work correctly
// outside of a running game loop. In such cases, it returns an opaque
// black pixel. For production use, this is called during the draw phase.
func (p *CairoPattern) surfaceColorAt(x, y float64) color.RGBA {
	p.mu.Lock()
	surface := p.surface
	x0, y0 := p.x0, p.y0
	p.mu.Unlock()

	if surface == nil {
		return color.RGBA{A: 255}
	}

	// Adjust for pattern offset
	px := int(x - x0)
	py := int(y - y0)

	// Check bounds
	bounds := surface.Bounds()
	if px < bounds.Min.X || px >= bounds.Max.X || py < bounds.Min.Y || py >= bounds.Max.Y {
		return color.RGBA{} // Transparent outside bounds
	}

	// Due to Ebiten limitations, reading pixels requires the game loop to be running.
	// The surfaceColorAt function is primarily used during the Mask operation,
	// which happens within the game loop's Draw phase. For use cases that require
	// reading pixel colors outside the game loop, use ReadPixels with proper timing.
	// Here we use a defer/recover to handle the case when this is called
	// outside the game loop (e.g., in tests).
	var result color.RGBA
	func() {
		defer func() {
			if r := recover(); r != nil {
				// Return opaque pixel if we can't read - caller should handle this
				result = color.RGBA{A: 255}
			}
		}()
		r, g, b, a := surface.At(px, py).RGBA()
		result = color.RGBA{
			R: uint8(r >> 8),
			G: uint8(g >> 8),
			B: uint8(b >> 8),
			A: uint8(a >> 8),
		}
	}()
	return result
}

// PatternExtend represents the extend mode for patterns.
// This controls how a pattern is rendered outside its defined area.
type PatternExtend int

const (
	// PatternExtendNone pads with transparent pixels.
	PatternExtendNone PatternExtend = iota
	// PatternExtendRepeat tiles the pattern.
	PatternExtendRepeat
	// PatternExtendReflect tiles the pattern, reflecting at boundaries.
	PatternExtendReflect
	// PatternExtendPad extends the edge color.
	PatternExtendPad
)

// SetExtend sets the extend mode for the pattern.
// This is equivalent to cairo_pattern_set_extend.
func (p *CairoPattern) SetExtend(extend PatternExtend) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.extend = extend
}

// GetExtend returns the current extend mode for the pattern.
// This is equivalent to cairo_pattern_get_extend.
func (p *CairoPattern) GetExtend() PatternExtend {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.extend
}

// CairoMatrix represents a 2D affine transformation matrix.
// The matrix is represented as:
//
//	| xx  xy |   | x |   | x0 |
//	| yx  yy | * | y | + | y0 |
//
// This matches Cairo's cairo_matrix_t structure.
type CairoMatrix struct {
	XX, XY float64 // First row: transformation for x
	YX, YY float64 // Second row: transformation for y
	X0, Y0 float64 // Translation offset
}

// NewIdentityMatrix creates a new identity matrix.
// This is equivalent to cairo_matrix_init_identity.
func NewIdentityMatrix() *CairoMatrix {
	return &CairoMatrix{
		XX: 1, XY: 0,
		YX: 0, YY: 1,
		X0: 0, Y0: 0,
	}
}

// NewTranslateMatrix creates a matrix that translates by (tx, ty).
// This is equivalent to cairo_matrix_init_translate.
func NewTranslateMatrix(tx, ty float64) *CairoMatrix {
	return &CairoMatrix{
		XX: 1, XY: 0,
		YX: 0, YY: 1,
		X0: tx, Y0: ty,
	}
}

// NewScaleMatrix creates a matrix that scales by (sx, sy).
// This is equivalent to cairo_matrix_init_scale.
func NewScaleMatrix(sx, sy float64) *CairoMatrix {
	return &CairoMatrix{
		XX: sx, XY: 0,
		YX: 0, YY: sy,
		X0: 0, Y0: 0,
	}
}

// NewRotateMatrix creates a matrix that rotates by angle radians.
// This is equivalent to cairo_matrix_init_rotate.
func NewRotateMatrix(angle float64) *CairoMatrix {
	c := math.Cos(angle)
	s := math.Sin(angle)
	return &CairoMatrix{
		XX: c, XY: -s,
		YX: s, YY: c,
		X0: 0, Y0: 0,
	}
}

// Translate applies a translation to the matrix.
// This is equivalent to cairo_matrix_translate.
func (m *CairoMatrix) Translate(tx, ty float64) {
	m.X0 += m.XX*tx + m.XY*ty
	m.Y0 += m.YX*tx + m.YY*ty
}

// Scale applies a scale to the matrix.
// This is equivalent to cairo_matrix_scale.
func (m *CairoMatrix) Scale(sx, sy float64) {
	m.XX *= sx
	m.XY *= sy
	m.YX *= sx
	m.YY *= sy
}

// Rotate applies a rotation to the matrix.
// This is equivalent to cairo_matrix_rotate.
func (m *CairoMatrix) Rotate(angle float64) {
	c := math.Cos(angle)
	s := math.Sin(angle)
	newXX := m.XX*c + m.XY*s
	newXY := m.XX*(-s) + m.XY*c
	newYX := m.YX*c + m.YY*s
	newYY := m.YX*(-s) + m.YY*c
	m.XX = newXX
	m.XY = newXY
	m.YX = newYX
	m.YY = newYY
}

// Multiply combines this matrix with another matrix.
// The result is stored in this matrix.
//
// Following Cairo's convention: the effect is to first apply the receiver
// matrix, then apply the argument matrix. Mathematically, this requires
// computing:
//
//	result = other * m
//
// where transformations are applied in right-to-left order.
//
// This ensures that transforming a point p with the result gives:
//
//	result(p) = other(m(p)).
func (m *CairoMatrix) Multiply(other *CairoMatrix) {
	xx := other.XX*m.XX + other.XY*m.YX
	xy := other.XX*m.XY + other.XY*m.YY
	yx := other.YX*m.XX + other.YY*m.YX
	yy := other.YX*m.XY + other.YY*m.YY
	x0 := other.XX*m.X0 + other.XY*m.Y0 + other.X0
	y0 := other.YX*m.X0 + other.YY*m.Y0 + other.Y0
	m.XX = xx
	m.XY = xy
	m.YX = yx
	m.YY = yy
	m.X0 = x0
	m.Y0 = y0
}

// TransformPoint transforms a point using the matrix.
// This is equivalent to cairo_matrix_transform_point.
func (m *CairoMatrix) TransformPoint(x, y float64) (tx, ty float64) {
	tx = m.XX*x + m.XY*y + m.X0
	ty = m.YX*x + m.YY*y + m.Y0
	return tx, ty
}

// TransformDistance transforms a distance vector (no translation).
// This is equivalent to cairo_matrix_transform_distance.
func (m *CairoMatrix) TransformDistance(dx, dy float64) (tdx, tdy float64) {
	tdx = m.XX*dx + m.XY*dy
	tdy = m.YX*dx + m.YY*dy
	return tdx, tdy
}

// Invert inverts the matrix if possible.
// Returns true if successful, false if the matrix is singular.
// This is equivalent to cairo_matrix_invert.
func (m *CairoMatrix) Invert() bool {
	det := m.XX*m.YY - m.XY*m.YX
	if det == 0 || math.IsInf(det, 0) || math.IsNaN(det) {
		return false
	}
	invDet := 1.0 / det
	newXX := m.YY * invDet
	newXY := -m.XY * invDet
	newYX := -m.YX * invDet
	newYY := m.XX * invDet
	newX0 := (m.XY*m.Y0 - m.YY*m.X0) * invDet
	newY0 := (m.YX*m.X0 - m.XX*m.Y0) * invDet
	m.XX = newXX
	m.XY = newXY
	m.YX = newYX
	m.YY = newYY
	m.X0 = newX0
	m.Y0 = newY0
	return true
}

// Copy creates a copy of the matrix.
func (m *CairoMatrix) Copy() *CairoMatrix {
	return &CairoMatrix{
		XX: m.XX, XY: m.XY,
		YX: m.YX, YY: m.YY,
		X0: m.X0, Y0: m.Y0,
	}
}

// CairoRenderer implements a Cairo-compatible drawing API using Ebiten.

// CairoRenderer implements a Cairo-compatible drawing API using Ebiten.
// It maintains drawing state similar to Cairo's context and translates
// Cairo drawing commands to Ebiten vector operations.
type CairoRenderer struct {
	screen       *ebiten.Image
	currentColor color.RGBA
	lineWidth    float32
	lineCap      LineCap
	lineJoin     LineJoin
	antialias    bool
	dashPattern  []float64
	dashOffset   float64
	miterLimit   float64
	path         *vector.Path
	pathStartX   float32
	pathStartY   float32
	pathCurrentX float32
	pathCurrentY float32
	hasPath      bool
	// pathSegments tracks all path segments for CopyPath functionality.
	// This mirrors the operations performed on the vector.Path.
	pathSegments []PathSegment
	// Tracks whether path bounds have been initialized.
	// Separate from hasPath to handle multi-point operations (e.g., Rectangle)
	// that call expandPathBounds multiple times before setting hasPath = true.
	pathBoundsInit bool
	// Path bounding box (tracked during path operations)
	pathMinX float32
	pathMinY float32
	pathMaxX float32
	pathMaxY float32
	// Pattern/gradient source
	sourcePattern *CairoPattern
	// Text rendering fields
	textRenderer *TextRenderer
	fontFamily   string
	fontSlant    FontSlant
	fontWeight   FontWeight
	fontSize     float64
	// Transformation state
	translateX float64
	translateY float64
	rotation   float64
	scaleX     float64
	scaleY     float64
	// Transformation matrix (full matrix representation)
	matrix *CairoMatrix
	// Clip state
	clipPath *vector.Path
	hasClip  bool
	// Clip bounding box (tracked when clip is set)
	clipMinX float32
	clipMinY float32
	clipMaxX float32
	clipMaxY float32
	// State stack for save/restore
	stateStack []cairoState
	// Fill rule for fill operations
	fillRule CairoFillRule
	// Compositing operator
	operator CairoOperator
	// Group rendering state - stack of group surfaces
	groupStack []*groupState
	// Source surface for SetSourceSurface
	sourceSurface    *ebiten.Image
	sourceSurfaceX   float64
	sourceSurfaceY   float64
	hasSourceSurface bool
	mu               sync.Mutex
}

// cairoState holds a snapshot of the drawing state for save/restore.
type cairoState struct {
	currentColor  color.RGBA
	sourcePattern *CairoPattern
	lineWidth     float32
	lineCap       LineCap
	lineJoin      LineJoin
	antialias     bool
	dashPattern   []float64
	dashOffset    float64
	miterLimit    float64
	fontFamily    string
	fontSlant     FontSlant
	fontWeight    FontWeight
	fontSize      float64
	translateX    float64
	translateY    float64
	rotation      float64
	scaleX        float64
	scaleY        float64
	matrix        *CairoMatrix
	clipPath      *vector.Path
	hasClip       bool
	// Clip bounding box (tracked when clip is set)
	clipMinX float32
	clipMinY float32
	clipMaxX float32
	clipMaxY float32
	// Fill rule for fill operations
	fillRule CairoFillRule
	// Compositing operator
	operator CairoOperator
}

// groupState holds the state for a push_group operation.
type groupState struct {
	surface        *ebiten.Image // The group's temporary surface
	previousScreen *ebiten.Image // The screen before push_group was called
	content        CairoContent  // Content type for the group
}

// CairoContent represents the content type for group surfaces.
type CairoContent int

const (
	// CairoContentColor creates a group with RGB content (no alpha).
	CairoContentColor CairoContent = iota
	// CairoContentAlpha creates a group with alpha-only content.
	CairoContentAlpha
	// CairoContentColorAlpha creates a group with RGBA content (default).
	CairoContentColorAlpha
)

// FontSlant represents Cairo font slant styles.
type FontSlant int

const (
	// FontSlantNormal is the normal (upright) font slant.
	FontSlantNormal FontSlant = iota
	// FontSlantItalic is the italic font slant.
	FontSlantItalic
	// FontSlantOblique is the oblique font slant.
	FontSlantOblique
)

// FontWeight represents Cairo font weight styles.
type FontWeight int

const (
	// FontWeightNormal is the normal font weight.
	FontWeightNormal FontWeight = iota
	// FontWeightBold is the bold font weight.
	FontWeightBold
)

// TextExtents contains text measurement information.
type TextExtents struct {
	XBearing float64
	YBearing float64
	Width    float64
	Height   float64
	XAdvance float64
	YAdvance float64
}

// LineCap represents the style of line end points.
type LineCap int

const (
	// LineCapButt ends the line at the exact endpoint.
	LineCapButt LineCap = iota
	// LineCapRound ends the line with a semicircular cap.
	LineCapRound
	// LineCapSquare ends the line with a square cap extending past the endpoint.
	LineCapSquare
)

// LineJoin represents the style of line corners.
type LineJoin int

const (
	// LineJoinMiter creates a sharp corner.
	LineJoinMiter LineJoin = iota
	// LineJoinRound creates a rounded corner.
	LineJoinRound
	// LineJoinBevel creates a beveled corner.
	LineJoinBevel
)

// NewCairoRenderer creates a new CairoRenderer instance.
// The renderer is initialized with default state: black color, line width 1.0.
func NewCairoRenderer() *CairoRenderer {
	return &CairoRenderer{
		currentColor: color.RGBA{R: 0, G: 0, B: 0, A: 255},
		lineWidth:    1.0,
		lineCap:      LineCapButt,
		lineJoin:     LineJoinMiter,
		antialias:    true,
		path:         &vector.Path{},
		pathSegments: make([]PathSegment, 0),
		hasPath:      false,
		textRenderer: NewTextRenderer(),
		fontFamily:   "GoMono",
		fontSlant:    FontSlantNormal,
		fontWeight:   FontWeightNormal,
		fontSize:     14.0,
		translateX:   0,
		translateY:   0,
		rotation:     0,
		scaleX:       1.0,
		scaleY:       1.0,
		matrix:       NewIdentityMatrix(),
		stateStack:   make([]cairoState, 0),
		fillRule:     CairoFillRuleWinding,
		operator:     CairoOperatorOver,
	}
}

// SetScreen sets the target image for drawing operations.
// This must be called before any drawing functions.
func (cr *CairoRenderer) SetScreen(screen *ebiten.Image) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	cr.screen = screen
}

// Screen returns the current target image.
func (cr *CairoRenderer) Screen() *ebiten.Image {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	return cr.screen
}

// SetSourceRGB sets the current drawing color using RGB values (0.0-1.0).
// This is equivalent to cairo_set_source_rgb.
func (cr *CairoRenderer) SetSourceRGB(r, g, b float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	cr.currentColor = color.RGBA{
		R: clampToByte(r),
		G: clampToByte(g),
		B: clampToByte(b),
		A: 255,
	}
}

// SetSourceRGBA sets the current drawing color using RGBA values (0.0-1.0).
// This is equivalent to cairo_set_source_rgba.
func (cr *CairoRenderer) SetSourceRGBA(r, g, b, a float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	cr.currentColor = color.RGBA{
		R: clampToByte(r),
		G: clampToByte(g),
		B: clampToByte(b),
		A: clampToByte(a),
	}
}

// GetCurrentColor returns the current drawing color.
func (cr *CairoRenderer) GetCurrentColor() color.RGBA {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	return cr.currentColor
}

// SetSource sets the current source pattern for drawing operations.
// This is equivalent to cairo_set_source.
func (cr *CairoRenderer) SetSource(pattern *CairoPattern) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	cr.sourcePattern = pattern
	// For solid patterns, also update the current color for compatibility
	if pattern != nil && pattern.patternType == PatternTypeSolid {
		cr.currentColor = pattern.solidColor
	}
}

// GetSource returns the current source pattern.
func (cr *CairoRenderer) GetSource() *CairoPattern {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	return cr.sourcePattern
}

// SetLineWidth sets the line width for stroke operations.
// This is equivalent to cairo_set_line_width.
func (cr *CairoRenderer) SetLineWidth(width float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	if width <= 0 {
		width = 1.0
	}
	cr.lineWidth = float32(width)
}

// GetLineWidth returns the current line width.
func (cr *CairoRenderer) GetLineWidth() float64 {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	return float64(cr.lineWidth)
}

// SetLineCap sets the line cap style.
// This is equivalent to cairo_set_line_cap.
func (cr *CairoRenderer) SetLineCap(capStyle LineCap) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	cr.lineCap = capStyle
}

// GetLineCap returns the current line cap style.
func (cr *CairoRenderer) GetLineCap() LineCap {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	return cr.lineCap
}

// SetLineJoin sets the line join style.
// This is equivalent to cairo_set_line_join.
func (cr *CairoRenderer) SetLineJoin(join LineJoin) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	cr.lineJoin = join
}

// GetLineJoin returns the current line join style.
func (cr *CairoRenderer) GetLineJoin() LineJoin {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	return cr.lineJoin
}

// SetAntialias enables or disables antialiasing.
// This is equivalent to cairo_set_antialias.
func (cr *CairoRenderer) SetAntialias(enabled bool) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	cr.antialias = enabled
}

// GetAntialias returns whether antialiasing is enabled.
func (cr *CairoRenderer) GetAntialias() bool {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	return cr.antialias
}

// SetDash sets the dash pattern for stroking.
// dashes is a slice of positive values specifying on/off lengths.
// offset specifies where in the pattern to start.
// This is equivalent to cairo_set_dash.
func (cr *CairoRenderer) SetDash(dashes []float64, offset float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	cr.dashPattern = make([]float64, len(dashes))
	copy(cr.dashPattern, dashes)
	cr.dashOffset = offset
}

// GetDash returns the current dash pattern and offset.
func (cr *CairoRenderer) GetDash() ([]float64, float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	dashes := make([]float64, len(cr.dashPattern))
	copy(dashes, cr.dashPattern)
	return dashes, cr.dashOffset
}

// GetDashCount returns the number of elements in the dash pattern.
func (cr *CairoRenderer) GetDashCount() int {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	return len(cr.dashPattern)
}

// SetMiterLimit sets the miter limit for stroking.
// This is equivalent to cairo_set_miter_limit.
func (cr *CairoRenderer) SetMiterLimit(limit float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	cr.miterLimit = limit
}

// GetMiterLimit returns the current miter limit.
func (cr *CairoRenderer) GetMiterLimit() float64 {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	return cr.miterLimit
}

// SetFillRule sets the fill rule for fill operations.
// rule: 0 = CAIRO_FILL_RULE_WINDING (maps to ebiten.FillRuleNonZero)
// rule: 1 = CAIRO_FILL_RULE_EVEN_ODD (maps to ebiten.FillRuleEvenOdd)
func (cr *CairoRenderer) SetFillRule(rule int) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	// Clamp to valid values
	if rule < int(CairoFillRuleWinding) {
		rule = int(CairoFillRuleWinding)
	} else if rule > int(CairoFillRuleEvenOdd) {
		rule = int(CairoFillRuleEvenOdd)
	}
	cr.fillRule = CairoFillRule(rule)
}

// GetFillRule returns the current fill rule.
// Returns CairoFillRuleWinding (0) or CairoFillRuleEvenOdd (1).
func (cr *CairoRenderer) GetFillRule() int {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	return int(cr.fillRule)
}

// SetOperator sets the compositing operator.
// See CairoOperator* constants for valid values (0-12).
// Common operators:
// - CairoOperatorClear: clears destination
// - CairoOperatorSource: replaces destination
// - CairoOperatorOver: draws source over destination (default)
func (cr *CairoRenderer) SetOperator(op int) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	// Clamp to valid range
	if op < int(CairoOperatorClear) {
		op = int(CairoOperatorClear)
	} else if op > int(CairoOperatorAdd) {
		op = int(CairoOperatorAdd)
	}
	cr.operator = CairoOperator(op)
}

// GetOperator returns the current operator.
// Returns a CairoOperator value (0-12).
func (cr *CairoRenderer) GetOperator() int {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	return int(cr.operator)
}

// --- Path Building Functions ---

// expandPathBounds updates the path bounding box to include the given point.
// Must be called while holding the mutex.
func (cr *CairoRenderer) expandPathBounds(x, y float32) {
	if !cr.pathBoundsInit {
		// First point - initialize bounds
		cr.pathMinX = x
		cr.pathMinY = y
		cr.pathMaxX = x
		cr.pathMaxY = y
		cr.pathBoundsInit = true
		return
	}
	if x < cr.pathMinX {
		cr.pathMinX = x
	}
	if x > cr.pathMaxX {
		cr.pathMaxX = x
	}
	if y < cr.pathMinY {
		cr.pathMinY = y
	}
	if y > cr.pathMaxY {
		cr.pathMaxY = y
	}
}

// initPathAtOrigin initializes the path with a starting point at (0,0).
// Must be called while holding the mutex.
// This is used when path operations require a current point but none exists.
func (cr *CairoRenderer) initPathAtOrigin() {
	cr.expandPathBounds(0, 0)
	cr.path.MoveTo(0, 0)
	cr.pathSegments = append(cr.pathSegments, PathSegment{Type: PathMoveTo, X: 0, Y: 0})
	cr.pathStartX = 0
	cr.pathStartY = 0
	cr.pathCurrentX = 0
	cr.pathCurrentY = 0
	cr.hasPath = true
}

// NewPath clears the current path and starts a new one.
// This is equivalent to cairo_new_path.
func (cr *CairoRenderer) NewPath() {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	cr.newPathUnlocked()
}

// MoveTo begins a new sub-path at the given point.
// This is equivalent to cairo_move_to.
func (cr *CairoRenderer) MoveTo(x, y float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	cr.expandPathBounds(float32(x), float32(y))
	cr.path.MoveTo(float32(x), float32(y))
	cr.pathSegments = append(cr.pathSegments, PathSegment{Type: PathMoveTo, X: x, Y: y})
	cr.pathStartX = float32(x)
	cr.pathStartY = float32(y)
	cr.pathCurrentX = float32(x)
	cr.pathCurrentY = float32(y)
	cr.hasPath = true
}

// LineTo adds a line from the current point to the given point.
// This is equivalent to cairo_line_to.
func (cr *CairoRenderer) LineTo(x, y float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	cr.expandPathBounds(float32(x), float32(y))
	if !cr.hasPath {
		cr.path.MoveTo(float32(x), float32(y))
		cr.pathSegments = append(cr.pathSegments, PathSegment{Type: PathMoveTo, X: x, Y: y})
		cr.pathStartX = float32(x)
		cr.pathStartY = float32(y)
		cr.hasPath = true
	} else {
		cr.path.LineTo(float32(x), float32(y))
		cr.pathSegments = append(cr.pathSegments, PathSegment{Type: PathLineTo, X: x, Y: y})
	}
	cr.pathCurrentX = float32(x)
	cr.pathCurrentY = float32(y)
}

// ClosePath closes the current sub-path by drawing a line back to the start.
// This is equivalent to cairo_close_path.
func (cr *CairoRenderer) ClosePath() {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	if cr.hasPath {
		cr.path.Close()
		cr.pathSegments = append(cr.pathSegments, PathSegment{Type: PathClose})
	}
}

// Arc adds a circular arc to the current path.
// xc, yc: center coordinates
// radius: arc radius
// angle1, angle2: start and end angles in radians
// This is equivalent to cairo_arc.
func (cr *CairoRenderer) Arc(xc, yc, radius, angle1, angle2 float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	// Calculate start point
	startX := xc + radius*math.Cos(angle1)
	startY := yc + radius*math.Sin(angle1)

	// Expand bounds for arc - use bounding box of full circle (conservative)
	cr.expandPathBounds(float32(xc-radius), float32(yc-radius))
	cr.expandPathBounds(float32(xc+radius), float32(yc+radius))

	// Move or line to start point
	if !cr.hasPath {
		cr.path.MoveTo(float32(startX), float32(startY))
		cr.pathSegments = append(cr.pathSegments, PathSegment{Type: PathMoveTo, X: startX, Y: startY})
		cr.pathStartX = float32(startX)
		cr.pathStartY = float32(startY)
		cr.hasPath = true
	} else {
		cr.path.LineTo(float32(startX), float32(startY))
		cr.pathSegments = append(cr.pathSegments, PathSegment{Type: PathLineTo, X: startX, Y: startY})
	}

	// Add arc using Ebiten's Arc method
	cr.path.Arc(float32(xc), float32(yc), float32(radius), float32(angle1), float32(angle2), vector.Clockwise)

	// Track arc segment
	endX := xc + radius*math.Cos(angle2)
	endY := yc + radius*math.Sin(angle2)
	cr.pathSegments = append(cr.pathSegments, PathSegment{
		Type:    PathArc,
		X:       endX,
		Y:       endY,
		CenterX: xc,
		CenterY: yc,
		Radius:  radius,
		Angle1:  angle1,
		Angle2:  angle2,
	})

	// Update current position
	cr.pathCurrentX = float32(endX)
	cr.pathCurrentY = float32(endY)
}

// ArcNegative adds a circular arc in the negative direction.
// This is equivalent to cairo_arc_negative.
func (cr *CairoRenderer) ArcNegative(xc, yc, radius, angle1, angle2 float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	// Calculate start point
	startX := xc + radius*math.Cos(angle1)
	startY := yc + radius*math.Sin(angle1)

	// Expand bounds for arc - use bounding box of full circle (conservative)
	cr.expandPathBounds(float32(xc-radius), float32(yc-radius))
	cr.expandPathBounds(float32(xc+radius), float32(yc+radius))

	// Move or line to start point
	if !cr.hasPath {
		cr.path.MoveTo(float32(startX), float32(startY))
		cr.pathSegments = append(cr.pathSegments, PathSegment{Type: PathMoveTo, X: startX, Y: startY})
		cr.pathStartX = float32(startX)
		cr.pathStartY = float32(startY)
		cr.hasPath = true
	} else {
		cr.path.LineTo(float32(startX), float32(startY))
		cr.pathSegments = append(cr.pathSegments, PathSegment{Type: PathLineTo, X: startX, Y: startY})
	}

	// Add arc in counter-clockwise direction
	cr.path.Arc(float32(xc), float32(yc), float32(radius), float32(angle1), float32(angle2), vector.CounterClockwise)

	// Track arc segment
	endX := xc + radius*math.Cos(angle2)
	endY := yc + radius*math.Sin(angle2)
	cr.pathSegments = append(cr.pathSegments, PathSegment{
		Type:    PathArcNegative,
		X:       endX,
		Y:       endY,
		CenterX: xc,
		CenterY: yc,
		Radius:  radius,
		Angle1:  angle1,
		Angle2:  angle2,
	})

	// Update current position
	cr.pathCurrentX = float32(endX)
	cr.pathCurrentY = float32(endY)
}

// CurveTo adds a cubic Bézier curve to the path.
// This is equivalent to cairo_curve_to.
// If there is no current point, it starts from (0,0). Note: this differs from
// the C Cairo API, which reports an error when there is no current point.
func (cr *CairoRenderer) CurveTo(x1, y1, x2, y2, x3, y3 float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	if !cr.hasPath {
		cr.initPathAtOrigin()
	}
	// Expand bounds to include all control points and end point
	cr.expandPathBounds(float32(x1), float32(y1))
	cr.expandPathBounds(float32(x2), float32(y2))
	cr.expandPathBounds(float32(x3), float32(y3))
	cr.path.CubicTo(float32(x1), float32(y1), float32(x2), float32(y2), float32(x3), float32(y3))
	cr.pathSegments = append(cr.pathSegments, PathSegment{
		Type: PathCurveTo,
		X:    x3,
		Y:    y3,
		X1:   x1,
		Y1:   y1,
		X2:   x2,
		Y2:   y2,
	})
	cr.pathCurrentX = float32(x3)
	cr.pathCurrentY = float32(y3)
}

// Rectangle adds a closed rectangular sub-path.
// This is equivalent to cairo_rectangle.
func (cr *CairoRenderer) Rectangle(x, y, width, height float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	// Expand bounds to include all corners
	cr.expandPathBounds(float32(x), float32(y))
	cr.expandPathBounds(float32(x+width), float32(y+height))

	cr.path.MoveTo(float32(x), float32(y))
	cr.path.LineTo(float32(x+width), float32(y))
	cr.path.LineTo(float32(x+width), float32(y+height))
	cr.path.LineTo(float32(x), float32(y+height))
	cr.path.Close()

	// Track all segments
	cr.pathSegments = append(cr.pathSegments,
		PathSegment{Type: PathMoveTo, X: x, Y: y},
		PathSegment{Type: PathLineTo, X: x + width, Y: y},
		PathSegment{Type: PathLineTo, X: x + width, Y: y + height},
		PathSegment{Type: PathLineTo, X: x, Y: y + height},
		PathSegment{Type: PathClose},
	)

	cr.pathStartX = float32(x)
	cr.pathStartY = float32(y)
	cr.pathCurrentX = float32(x)
	cr.pathCurrentY = float32(y)
	cr.hasPath = true
}

// --- Relative Path Functions ---

// RelMoveTo moves the current point by a relative offset.
// This is equivalent to cairo_rel_move_to.
// If there is no current point, it starts from (0,0). Note: this differs from
// the C Cairo API, which reports an error when there is no current point.
func (cr *CairoRenderer) RelMoveTo(dx, dy float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	if !cr.hasPath {
		cr.initPathAtOrigin()
	}
	newX := float64(cr.pathCurrentX) + dx
	newY := float64(cr.pathCurrentY) + dy
	cr.expandPathBounds(float32(newX), float32(newY))
	cr.path.MoveTo(float32(newX), float32(newY))
	cr.pathSegments = append(cr.pathSegments, PathSegment{Type: PathMoveTo, X: newX, Y: newY})
	cr.pathStartX = float32(newX)
	cr.pathStartY = float32(newY)
	cr.pathCurrentX = float32(newX)
	cr.pathCurrentY = float32(newY)
}

// RelLineTo draws a line from the current point by a relative offset.
// This is equivalent to cairo_rel_line_to.
// If there is no current point, it starts from (0,0). Note: this differs from
// the C Cairo API, which reports an error when there is no current point.
func (cr *CairoRenderer) RelLineTo(dx, dy float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	if !cr.hasPath {
		cr.initPathAtOrigin()
	}
	newX := float64(cr.pathCurrentX) + dx
	newY := float64(cr.pathCurrentY) + dy
	cr.expandPathBounds(float32(newX), float32(newY))
	cr.path.LineTo(float32(newX), float32(newY))
	cr.pathSegments = append(cr.pathSegments, PathSegment{Type: PathLineTo, X: newX, Y: newY})
	cr.pathCurrentX = float32(newX)
	cr.pathCurrentY = float32(newY)
}

// RelCurveTo adds a cubic Bézier curve relative to the current point.
// This is equivalent to cairo_rel_curve_to.
// If there is no current point, it starts from (0,0). Note: this differs from
// the C Cairo API, which reports an error when there is no current point.
func (cr *CairoRenderer) RelCurveTo(dx1, dy1, dx2, dy2, dx3, dy3 float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	if !cr.hasPath {
		cr.initPathAtOrigin()
	}
	curX := float64(cr.pathCurrentX)
	curY := float64(cr.pathCurrentY)
	// Expand bounds to include all control points and end point
	cr.expandPathBounds(float32(curX+dx1), float32(curY+dy1))
	cr.expandPathBounds(float32(curX+dx2), float32(curY+dy2))
	cr.expandPathBounds(float32(curX+dx3), float32(curY+dy3))
	cr.path.CubicTo(
		float32(curX+dx1), float32(curY+dy1),
		float32(curX+dx2), float32(curY+dy2),
		float32(curX+dx3), float32(curY+dy3),
	)
	cr.pathSegments = append(cr.pathSegments, PathSegment{
		Type: PathCurveTo,
		X:    curX + dx3,
		Y:    curY + dy3,
		X1:   curX + dx1,
		Y1:   curY + dy1,
		X2:   curX + dx2,
		Y2:   curY + dy2,
	})
	cr.pathCurrentX = float32(curX + dx3)
	cr.pathCurrentY = float32(curY + dy3)
}

// --- Drawing Operations ---

// Stroke draws the current path as a stroked line.
// This is equivalent to cairo_stroke.
func (cr *CairoRenderer) Stroke() {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	if !cr.canDraw() {
		return
	}

	opts := cr.buildStrokeOptions()
	vertices, indices := cr.path.AppendVerticesAndIndicesForStroke(nil, nil, opts)
	cr.setVertexColors(vertices)

	// Get clipped screen and adjust vertex coordinates
	screen, dx, dy := cr.getClippedScreen()
	cr.adjustVerticesForClip(vertices, dx, dy)

	screen.DrawTriangles(vertices, indices, emptySubImage, &ebiten.DrawTrianglesOptions{
		AntiAlias: cr.antialias,
		Blend:     cr.getEbitenBlend(),
	})

	// Clear the path after stroking
	cr.clearPathUnlocked()
}

// Fill fills the current path with the current color.
// This is equivalent to cairo_fill.
func (cr *CairoRenderer) Fill() {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	if !cr.canDraw() {
		return
	}

	vertices, indices := cr.path.AppendVerticesAndIndicesForFilling(nil, nil)
	cr.setVertexColors(vertices)

	// Get clipped screen and adjust vertex coordinates
	screen, dx, dy := cr.getClippedScreen()
	cr.adjustVerticesForClip(vertices, dx, dy)

	screen.DrawTriangles(vertices, indices, emptySubImage, &ebiten.DrawTrianglesOptions{
		AntiAlias: cr.antialias,
		FillRule:  cr.getEbitenFillRule(),
		Blend:     cr.getEbitenBlend(),
	})

	// Clear the path after filling
	cr.clearPathUnlocked()
}

// FillPreserve fills the current path without clearing it.
// This is equivalent to cairo_fill_preserve.
func (cr *CairoRenderer) FillPreserve() {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	if !cr.canDraw() {
		return
	}

	vertices, indices := cr.path.AppendVerticesAndIndicesForFilling(nil, nil)
	cr.setVertexColors(vertices)

	// Get clipped screen and adjust vertex coordinates
	screen, dx, dy := cr.getClippedScreen()
	cr.adjustVerticesForClip(vertices, dx, dy)

	screen.DrawTriangles(vertices, indices, emptySubImage, &ebiten.DrawTrianglesOptions{
		AntiAlias: cr.antialias,
		FillRule:  cr.getEbitenFillRule(),
		Blend:     cr.getEbitenBlend(),
	})
}

// StrokePreserve strokes the current path without clearing it.
// This is equivalent to cairo_stroke_preserve.
func (cr *CairoRenderer) StrokePreserve() {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	if !cr.canDraw() {
		return
	}

	opts := cr.buildStrokeOptions()
	vertices, indices := cr.path.AppendVerticesAndIndicesForStroke(nil, nil, opts)
	cr.setVertexColors(vertices)

	// Get clipped screen and adjust vertex coordinates
	screen, dx, dy := cr.getClippedScreen()
	cr.adjustVerticesForClip(vertices, dx, dy)

	screen.DrawTriangles(vertices, indices, emptySubImage, &ebiten.DrawTrianglesOptions{
		AntiAlias: cr.antialias,
		Blend:     cr.getEbitenBlend(),
	})
}

// Paint fills the entire surface with the current color or source pattern.
// When a clip region is set, only the clipped area is filled.
// This is equivalent to cairo_paint.
func (cr *CairoRenderer) Paint() {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	if cr.screen == nil {
		return
	}

	// Get clipped screen for painting
	screen, clipX, clipY := cr.getClippedScreen()

	// Check if we have a surface pattern to paint
	if cr.sourcePattern != nil && cr.sourcePattern.patternType == PatternTypeSurface {
		if cr.sourcePattern.surface != nil {
			opts := &ebiten.DrawImageOptions{
				Blend: cr.getEbitenBlend(),
			}
			// Apply pattern offset, adjusted for clip region
			opts.GeoM.Translate(cr.sourcePattern.x0-float64(clipX), cr.sourcePattern.y0-float64(clipY))
			screen.DrawImage(cr.sourcePattern.surface, opts)
			return
		}
	}

	// Fall back to solid color
	screen.Fill(cr.currentColor)
}

// PaintWithAlpha fills the entire surface with the current color or source pattern
// at the given alpha. When a clip region is set, only the clipped area is filled.
// This is equivalent to cairo_paint_with_alpha.
func (cr *CairoRenderer) PaintWithAlpha(alpha float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	if cr.screen == nil {
		return
	}

	// Get clipped screen for painting
	screen, clipX, clipY := cr.getClippedScreen()

	// Check if we have a surface pattern to paint
	if cr.sourcePattern != nil && cr.sourcePattern.patternType == PatternTypeSurface {
		if cr.sourcePattern.surface != nil {
			opts := &ebiten.DrawImageOptions{
				Blend: cr.getEbitenBlend(),
			}
			// Apply pattern offset, adjusted for clip region
			opts.GeoM.Translate(cr.sourcePattern.x0-float64(clipX), cr.sourcePattern.y0-float64(clipY))
			// Apply alpha via color matrix
			opts.ColorScale.Scale(1, 1, 1, float32(alpha))
			screen.DrawImage(cr.sourcePattern.surface, opts)
			return
		}
	}

	// Fall back to solid color with alpha
	clr := cr.currentColor
	clr.A = clampToByte(alpha)
	screen.Fill(clr)
}

// --- Convenience Drawing Functions ---
//
// These functions provide atomic operations that acquire the lock once and perform
// all operations under that single lock. This ensures thread-safe atomic drawing
// operations in concurrent scenarios.

// DrawLine draws a line from (x1,y1) to (x2,y2) with the current color and line width.
// This is a convenience function that combines NewPath, MoveTo, LineTo, and Stroke.
// This function is atomic - the mutex is held for the entire operation.
func (cr *CairoRenderer) DrawLine(x1, y1, x2, y2 float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	cr.newPathUnlocked()
	cr.moveToUnlocked(x1, y1)
	cr.lineToUnlocked(x2, y2)
	cr.strokeUnlocked()
}

// DrawRectangle draws a stroked rectangle.
// This is a convenience function that combines NewPath, Rectangle and Stroke.
// This function is atomic - the mutex is held for the entire operation.
func (cr *CairoRenderer) DrawRectangle(x, y, width, height float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	cr.newPathUnlocked()
	cr.rectangleUnlocked(x, y, width, height)
	cr.strokeUnlocked()
}

// FillRectangle draws a filled rectangle.
// This is a convenience function that combines NewPath, Rectangle and Fill.
// This function is atomic - the mutex is held for the entire operation.
func (cr *CairoRenderer) FillRectangle(x, y, width, height float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	cr.newPathUnlocked()
	cr.rectangleUnlocked(x, y, width, height)
	cr.fillUnlocked()
}

// DrawCircle draws a stroked circle.
// This is a convenience function that combines NewPath, Arc, ClosePath and Stroke.
// This function is atomic - the mutex is held for the entire operation.
func (cr *CairoRenderer) DrawCircle(xc, yc, radius float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	cr.newPathUnlocked()
	cr.arcUnlocked(xc, yc, radius, 0, 2*math.Pi)
	cr.closePathUnlocked()
	cr.strokeUnlocked()
}

// FillCircle draws a filled circle.
// This is a convenience function that combines NewPath, Arc, ClosePath and Fill.
// This function is atomic - the mutex is held for the entire operation.
func (cr *CairoRenderer) FillCircle(xc, yc, radius float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	cr.newPathUnlocked()
	cr.arcUnlocked(xc, yc, radius, 0, 2*math.Pi)
	cr.closePathUnlocked()
	cr.fillUnlocked()
}

// --- Helper Functions ---

// canDraw checks if drawing is possible (has screen and path).
// This must be called while holding the mutex.
func (cr *CairoRenderer) canDraw() bool {
	return cr.screen != nil && cr.hasPath
}

// buildStrokeOptions creates stroke options from current state.
// This must be called while holding the mutex.
func (cr *CairoRenderer) buildStrokeOptions() *vector.StrokeOptions {
	opts := &vector.StrokeOptions{
		Width: cr.lineWidth,
	}
	switch cr.lineCap {
	case LineCapButt:
		opts.LineCap = vector.LineCapButt
	case LineCapRound:
		opts.LineCap = vector.LineCapRound
	case LineCapSquare:
		opts.LineCap = vector.LineCapSquare
	}
	switch cr.lineJoin {
	case LineJoinMiter:
		opts.LineJoin = vector.LineJoinMiter
	case LineJoinRound:
		opts.LineJoin = vector.LineJoinRound
	case LineJoinBevel:
		opts.LineJoin = vector.LineJoinBevel
	}
	return opts
}

// getEbitenFillRule converts Cairo fill rule to Ebiten FillRule.
// This must be called while holding the mutex.
func (cr *CairoRenderer) getEbitenFillRule() ebiten.FillRule {
	switch cr.fillRule {
	case CairoFillRuleWinding:
		return ebiten.FillRuleNonZero
	case CairoFillRuleEvenOdd:
		return ebiten.FillRuleEvenOdd
	default:
		return ebiten.FillRuleNonZero
	}
}

// getEbitenBlend converts Cairo operator to Ebiten Blend mode.
// This must be called while holding the mutex.
func (cr *CairoRenderer) getEbitenBlend() ebiten.Blend {
	switch cr.operator {
	case CairoOperatorClear:
		// CAIRO_OPERATOR_CLEAR sets all blend factors to zero and uses Add operation.
		// This effectively clears (sets to transparent black) wherever the source
		// would be drawn, because: result = 0*src + 0*dst = 0
		return ebiten.Blend{
			BlendFactorSourceRGB:        ebiten.BlendFactorZero,
			BlendFactorSourceAlpha:      ebiten.BlendFactorZero,
			BlendFactorDestinationRGB:   ebiten.BlendFactorZero,
			BlendFactorDestinationAlpha: ebiten.BlendFactorZero,
			BlendOperationRGB:           ebiten.BlendOperationAdd,
			BlendOperationAlpha:         ebiten.BlendOperationAdd,
		}
	case CairoOperatorSource:
		return ebiten.BlendCopy
	case CairoOperatorOver:
		return ebiten.BlendSourceOver
	case CairoOperatorIn:
		return ebiten.BlendSourceIn
	case CairoOperatorOut:
		return ebiten.BlendSourceOut
	case CairoOperatorAtop:
		return ebiten.BlendSourceAtop
	case CairoOperatorDest:
		return ebiten.BlendDestination
	case CairoOperatorDestOver:
		return ebiten.BlendDestinationOver
	case CairoOperatorDestIn:
		return ebiten.BlendDestinationIn
	case CairoOperatorDestOut:
		return ebiten.BlendDestinationOut
	case CairoOperatorDestAtop:
		return ebiten.BlendDestinationAtop
	case CairoOperatorXor:
		return ebiten.BlendXor
	case CairoOperatorAdd:
		return ebiten.BlendLighter
	default:
		return ebiten.BlendSourceOver
	}
}

// setVertexColors sets the current color on all vertices.
// This must be called while holding the mutex.
func (cr *CairoRenderer) setVertexColors(vertices []ebiten.Vertex) {
	r := float32(cr.currentColor.R) / 255
	g := float32(cr.currentColor.G) / 255
	b := float32(cr.currentColor.B) / 255
	a := float32(cr.currentColor.A) / 255
	for i := range vertices {
		vertices[i].ColorR = r
		vertices[i].ColorG = g
		vertices[i].ColorB = b
		vertices[i].ColorA = a
	}
}

// getClippedScreen returns the screen to draw on, applying clipping if set.
// If there is a clip region, it returns a SubImage representing the clipped area.
// Otherwise, it returns the original screen.
// Also returns the offset (dx, dy) that needs to be applied to vertex coordinates.
// This must be called while holding the mutex.
func (cr *CairoRenderer) getClippedScreen() (screen *ebiten.Image, dx, dy float32) {
	if !cr.hasClip || cr.screen == nil {
		return cr.screen, 0, 0
	}

	// Create a rectangle from the clip bounds
	clipRect := image.Rect(
		int(cr.clipMinX),
		int(cr.clipMinY),
		int(cr.clipMaxX),
		int(cr.clipMaxY),
	)

	// SubImage returns an image.Image, but for Ebiten images it's always *ebiten.Image
	subImg := cr.screen.SubImage(clipRect)
	if subImg == nil {
		// Fallback if SubImage fails (shouldn't happen for valid Ebiten images)
		return cr.screen, 0, 0
	}

	// Type assert to *ebiten.Image (safe per Ebiten documentation)
	return subImg.(*ebiten.Image), cr.clipMinX, cr.clipMinY
}

// adjustVerticesForClip offsets vertex positions by the clip origin.
// When drawing to a SubImage, we need to adjust vertex positions since the
// SubImage coordinate system starts at (0,0) for the clip region.
// This must be called while holding the mutex.
func (cr *CairoRenderer) adjustVerticesForClip(vertices []ebiten.Vertex, dx, dy float32) {
	if dx == 0 && dy == 0 {
		return
	}
	for i := range vertices {
		vertices[i].DstX -= dx
		vertices[i].DstY -= dy
	}
}

// --- Unlocked Internal Methods ---
//
// These internal methods perform path and drawing operations without acquiring the mutex.
// They are used by the atomic convenience functions to perform multiple operations
// under a single lock. All of these methods MUST be called while holding the mutex.

// newPathUnlocked clears the current path without acquiring the mutex.
// This must be called while holding the mutex.
func (cr *CairoRenderer) newPathUnlocked() {
	cr.clearPathUnlocked()
}

// moveToUnlocked begins a new sub-path at the given point without acquiring the mutex.
// This must be called while holding the mutex.
func (cr *CairoRenderer) moveToUnlocked(x, y float64) {
	cr.expandPathBounds(float32(x), float32(y))
	cr.path.MoveTo(float32(x), float32(y))
	cr.pathSegments = append(cr.pathSegments, PathSegment{Type: PathMoveTo, X: x, Y: y})
	cr.pathStartX = float32(x)
	cr.pathStartY = float32(y)
	cr.pathCurrentX = float32(x)
	cr.pathCurrentY = float32(y)
	cr.hasPath = true
}

// lineToUnlocked adds a line to the path without acquiring the mutex.
// This must be called while holding the mutex.
func (cr *CairoRenderer) lineToUnlocked(x, y float64) {
	cr.expandPathBounds(float32(x), float32(y))
	if !cr.hasPath {
		cr.path.MoveTo(float32(x), float32(y))
		cr.pathSegments = append(cr.pathSegments, PathSegment{Type: PathMoveTo, X: x, Y: y})
		cr.pathStartX = float32(x)
		cr.pathStartY = float32(y)
		cr.hasPath = true
	} else {
		cr.path.LineTo(float32(x), float32(y))
		cr.pathSegments = append(cr.pathSegments, PathSegment{Type: PathLineTo, X: x, Y: y})
	}
	cr.pathCurrentX = float32(x)
	cr.pathCurrentY = float32(y)
}

// rectangleUnlocked adds a rectangle to the path without acquiring the mutex.
// This must be called while holding the mutex.
func (cr *CairoRenderer) rectangleUnlocked(x, y, width, height float64) {
	// Expand bounds to include all corners
	cr.expandPathBounds(float32(x), float32(y))
	cr.expandPathBounds(float32(x+width), float32(y+height))

	cr.path.MoveTo(float32(x), float32(y))
	cr.path.LineTo(float32(x+width), float32(y))
	cr.path.LineTo(float32(x+width), float32(y+height))
	cr.path.LineTo(float32(x), float32(y+height))
	cr.path.Close()

	// Track all segments
	cr.pathSegments = append(cr.pathSegments,
		PathSegment{Type: PathMoveTo, X: x, Y: y},
		PathSegment{Type: PathLineTo, X: x + width, Y: y},
		PathSegment{Type: PathLineTo, X: x + width, Y: y + height},
		PathSegment{Type: PathLineTo, X: x, Y: y + height},
		PathSegment{Type: PathClose},
	)

	cr.pathStartX = float32(x)
	cr.pathStartY = float32(y)
	cr.pathCurrentX = float32(x)
	cr.pathCurrentY = float32(y)
	cr.hasPath = true
}

// arcUnlocked adds a circular arc to the path without acquiring the mutex.
// This must be called while holding the mutex.
func (cr *CairoRenderer) arcUnlocked(xc, yc, radius, angle1, angle2 float64) {
	// Calculate start point
	startX := xc + radius*math.Cos(angle1)
	startY := yc + radius*math.Sin(angle1)

	// Expand bounds for arc - use bounding box of full circle (conservative)
	cr.expandPathBounds(float32(xc-radius), float32(yc-radius))
	cr.expandPathBounds(float32(xc+radius), float32(yc+radius))

	// Move or line to start point
	if !cr.hasPath {
		cr.path.MoveTo(float32(startX), float32(startY))
		cr.pathSegments = append(cr.pathSegments, PathSegment{Type: PathMoveTo, X: startX, Y: startY})
		cr.pathStartX = float32(startX)
		cr.pathStartY = float32(startY)
		cr.hasPath = true
	} else {
		cr.path.LineTo(float32(startX), float32(startY))
		cr.pathSegments = append(cr.pathSegments, PathSegment{Type: PathLineTo, X: startX, Y: startY})
	}

	// Add arc using Ebiten's Arc method
	cr.path.Arc(float32(xc), float32(yc), float32(radius), float32(angle1), float32(angle2), vector.Clockwise)

	// Track arc segment
	endX := xc + radius*math.Cos(angle2)
	endY := yc + radius*math.Sin(angle2)
	cr.pathSegments = append(cr.pathSegments, PathSegment{
		Type:    PathArc,
		X:       endX,
		Y:       endY,
		CenterX: xc,
		CenterY: yc,
		Radius:  radius,
		Angle1:  angle1,
		Angle2:  angle2,
	})

	// Update current position
	cr.pathCurrentX = float32(endX)
	cr.pathCurrentY = float32(endY)
}

// closePathUnlocked closes the current sub-path without acquiring the mutex.
// This must be called while holding the mutex.
func (cr *CairoRenderer) closePathUnlocked() {
	if cr.hasPath {
		cr.path.Close()
		cr.pathSegments = append(cr.pathSegments, PathSegment{Type: PathClose})
	}
}

// clearPathUnlocked resets the path state after drawing without acquiring the mutex.
// This must be called while holding the mutex.
func (cr *CairoRenderer) clearPathUnlocked() {
	cr.path = &vector.Path{}
	cr.pathSegments = make([]PathSegment, 0)
	cr.hasPath = false
	cr.pathBoundsInit = false
	cr.pathMinX = 0
	cr.pathMinY = 0
	cr.pathMaxX = 0
	cr.pathMaxY = 0
}

// strokeUnlocked strokes the current path without acquiring the mutex.
// This must be called while holding the mutex.
func (cr *CairoRenderer) strokeUnlocked() {
	if !cr.canDraw() {
		return
	}

	opts := cr.buildStrokeOptions()
	vertices, indices := cr.path.AppendVerticesAndIndicesForStroke(nil, nil, opts)
	cr.setVertexColors(vertices)

	// Get clipped screen and adjust vertex coordinates
	screen, dx, dy := cr.getClippedScreen()
	cr.adjustVerticesForClip(vertices, dx, dy)

	screen.DrawTriangles(vertices, indices, emptySubImage, &ebiten.DrawTrianglesOptions{
		AntiAlias: cr.antialias,
		Blend:     cr.getEbitenBlend(),
	})

	// Clear the path after stroking
	cr.clearPathUnlocked()
}

// fillUnlocked fills the current path without acquiring the mutex.
// This must be called while holding the mutex.
func (cr *CairoRenderer) fillUnlocked() {
	if !cr.canDraw() {
		return
	}

	vertices, indices := cr.path.AppendVerticesAndIndicesForFilling(nil, nil)
	cr.setVertexColors(vertices)

	// Get clipped screen and adjust vertex coordinates
	screen, dx, dy := cr.getClippedScreen()
	cr.adjustVerticesForClip(vertices, dx, dy)

	screen.DrawTriangles(vertices, indices, emptySubImage, &ebiten.DrawTrianglesOptions{
		AntiAlias: cr.antialias,
		FillRule:  cr.getEbitenFillRule(),
		Blend:     cr.getEbitenBlend(),
	})

	// Clear the path after filling
	cr.clearPathUnlocked()
}

// GetCurrentPoint returns the current point in the path.
func (cr *CairoRenderer) GetCurrentPoint() (x, y float64, hasPoint bool) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	return float64(cr.pathCurrentX), float64(cr.pathCurrentY), cr.hasPath
}

// HasCurrentPoint returns whether there is a current point defined.
// This is equivalent to cairo_has_current_point.
func (cr *CairoRenderer) HasCurrentPoint() bool {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	return cr.hasPath
}

// PathExtents returns the bounding box of the current path in user-space coordinates.
// This is equivalent to cairo_path_extents.
// Returns (x1, y1, x2, y2) where (x1, y1) is the top-left and (x2, y2) is the bottom-right.
// If there is no current path, returns (0, 0, 0, 0).
func (cr *CairoRenderer) PathExtents() (x1, y1, x2, y2 float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	if !cr.hasPath {
		return 0, 0, 0, 0
	}

	// Return tracked path bounding box
	return float64(cr.pathMinX), float64(cr.pathMinY),
		float64(cr.pathMaxX), float64(cr.pathMaxY)
}

// ClipExtents returns the bounding box of the current clip region.
// This is equivalent to cairo_clip_extents.
// Returns (x1, y1, x2, y2) where (x1, y1) is the top-left and (x2, y2) is the bottom-right.
// If there is no clip region, returns (0, 0, screenWidth, screenHeight) or (0, 0, 0, 0) if no screen.
func (cr *CairoRenderer) ClipExtents() (x1, y1, x2, y2 float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	if !cr.hasClip {
		// No clip - return the entire surface bounds
		if cr.screen != nil {
			bounds := cr.screen.Bounds()
			return float64(bounds.Min.X), float64(bounds.Min.Y),
				float64(bounds.Max.X), float64(bounds.Max.Y)
		}
		return 0, 0, 0, 0
	}

	// Return tracked clip bounding box
	return float64(cr.clipMinX), float64(cr.clipMinY),
		float64(cr.clipMaxX), float64(cr.clipMaxY)
}

// InClip returns whether the given point is inside the current clip region.
// This is equivalent to cairo_in_clip.
// If there is no clip region, returns true (entire surface is available).
func (cr *CairoRenderer) InClip(x, y float64) bool {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	if !cr.hasClip {
		// No clip region means the entire surface is available
		return true
	}

	// Check if point is within tracked clip bounds
	return float32(x) >= cr.clipMinX && float32(x) <= cr.clipMaxX &&
		float32(y) >= cr.clipMinY && float32(y) <= cr.clipMaxY
}

// clampToByte converts a float64 value (0.0-1.0) to a byte (0-255).
func clampToByte(v float64) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 1 {
		return 255
	}
	return uint8(v * 255)
}

// --- Text Functions ---

// SelectFontFace sets the font family, slant, and weight for text rendering.
// This is equivalent to cairo_select_font_face.
func (cr *CairoRenderer) SelectFontFace(family string, slant FontSlant, weight FontWeight) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	cr.fontFamily = family
	cr.fontSlant = slant
	cr.fontWeight = weight

	// Map Cairo slant/weight to internal FontStyle
	style := cr.mapToFontStyle(slant, weight)
	cr.textRenderer.SetFont(family, style)
}

// mapToFontStyle converts Cairo slant/weight to internal FontStyle.
// Must be called while holding the mutex.
func (cr *CairoRenderer) mapToFontStyle(slant FontSlant, weight FontWeight) FontStyle {
	if weight == FontWeightBold {
		if slant == FontSlantItalic || slant == FontSlantOblique {
			return FontStyleBoldItalic
		}
		return FontStyleBold
	}
	if slant == FontSlantItalic || slant == FontSlantOblique {
		return FontStyleItalic
	}
	return FontStyleRegular
}

// SetFontSize sets the font size for text rendering.
// This is equivalent to cairo_set_font_size.
func (cr *CairoRenderer) SetFontSize(size float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	if size <= 0 {
		size = 14.0
	}
	cr.fontSize = size
	cr.textRenderer.SetFontSize(size)
}

// GetFontSize returns the current font size.
func (cr *CairoRenderer) GetFontSize() float64 {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	return cr.fontSize
}

// ShowText renders text at the current point.
// This is equivalent to cairo_show_text.
// If no path exists, text is rendered at (0, 0).
func (cr *CairoRenderer) ShowText(text string) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	// Get current position (or 0,0 if no path)
	x := float64(cr.pathCurrentX)
	y := float64(cr.pathCurrentY)

	// Only draw if we have a screen
	if cr.screen != nil {
		// Apply transformation
		tx, ty := cr.transformPoint(x, y)

		// Draw text using the text renderer
		cr.textRenderer.DrawText(cr.screen, text, tx, ty, cr.currentColor)
	}

	// Update current point by advancing by text width
	// (this happens regardless of whether we have a screen)
	w, _ := cr.textRenderer.MeasureText(text)
	cr.pathCurrentX = float32(x + w)
	cr.hasPath = true
}

// TextExtentsResult returns the measurements of the given text.
// This is equivalent to cairo_text_extents.
func (cr *CairoRenderer) TextExtentsResult(text string) TextExtents {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	w, h := cr.textRenderer.MeasureText(text)

	return TextExtents{
		XBearing: 0,
		YBearing: -h, // Negative because text extends upward from baseline
		Width:    w,
		Height:   h,
		XAdvance: w,
		YAdvance: 0,
	}
}

// TextPath adds the outline of the given text to the current path.
// This is equivalent to cairo_text_path.
//
// Implementation Note: Since Ebiten does not expose the actual glyph outlines
// from the underlying font library, this implementation creates a rectangular
// approximation using the text bounding box. The rectangle path can then be
// stroked or filled. For true glyph outline support, a CGO-based font library
// would be required.
//
// The text outline is added at the current path position (or 0,0 if no path exists).
// After calling TextPath, the current point is advanced by the text width.
func (cr *CairoRenderer) TextPath(text string) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	// Get current position (or 0,0 if no path)
	x := float64(cr.pathCurrentX)
	y := float64(cr.pathCurrentY)

	// Measure the text
	w, h := cr.textRenderer.MeasureText(text)
	if w == 0 || h == 0 {
		return
	}

	// Add a rectangle path representing the text bounds.
	// In Cairo, text is drawn with the baseline at the specified y position,
	// so the rectangle should extend upward from the baseline by the height.
	// The y-bearing is typically negative (extends above baseline).
	rectX := x
	rectY := y - h // Text extends upward from baseline

	// Add the rectangle as path segments (using internal method to avoid deadlock)
	cr.path.MoveTo(float32(rectX), float32(rectY))
	cr.path.LineTo(float32(rectX+w), float32(rectY))
	cr.path.LineTo(float32(rectX+w), float32(rectY+h))
	cr.path.LineTo(float32(rectX), float32(rectY+h))
	cr.path.Close()

	// Track path segments for CopyPath
	cr.pathSegments = append(cr.pathSegments,
		PathSegment{Type: PathMoveTo, X: rectX, Y: rectY},
		PathSegment{Type: PathLineTo, X: rectX + w, Y: rectY},
		PathSegment{Type: PathLineTo, X: rectX + w, Y: rectY + h},
		PathSegment{Type: PathLineTo, X: rectX, Y: rectY + h},
		PathSegment{Type: PathClose},
	)

	// Update path bounds
	cr.expandPathBounds(float32(rectX), float32(rectY))
	cr.expandPathBounds(float32(rectX+w), float32(rectY+h))

	// Update current point by advancing by text width
	cr.pathCurrentX = float32(x + w)
	cr.hasPath = true
}

// --- Transformation Functions ---

// Translate moves the coordinate system origin.
// This is equivalent to cairo_translate.
func (cr *CairoRenderer) Translate(tx, ty float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	cr.translateX += tx
	cr.translateY += ty
	// Also update the matrix
	if cr.matrix != nil {
		cr.matrix.Translate(tx, ty)
	}
}

// GetTranslate returns the current translation values.
func (cr *CairoRenderer) GetTranslate() (x, y float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	return cr.translateX, cr.translateY
}

// Rotate rotates the coordinate system by an angle in radians.
// This is equivalent to cairo_rotate.
func (cr *CairoRenderer) Rotate(angle float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	cr.rotation += angle
	// Also update the matrix
	if cr.matrix != nil {
		cr.matrix.Rotate(angle)
	}
}

// GetRotation returns the current rotation in radians.
func (cr *CairoRenderer) GetRotation() float64 {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	return cr.rotation
}

// Scale scales the coordinate system.
// This is equivalent to cairo_scale.
func (cr *CairoRenderer) Scale(sx, sy float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	cr.scaleX *= sx
	cr.scaleY *= sy
	// Also update the matrix
	if cr.matrix != nil {
		cr.matrix.Scale(sx, sy)
	}
}

// GetScale returns the current scale factors.
func (cr *CairoRenderer) GetScale() (sx, sy float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	return cr.scaleX, cr.scaleY
}

// Save saves the current drawing state to a stack.
// This is equivalent to cairo_save.
func (cr *CairoRenderer) Save() {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	// Copy the matrix for the saved state
	var matrixCopy *CairoMatrix
	if cr.matrix != nil {
		matrixCopy = cr.matrix.Copy()
	}

	// Copy the dash pattern
	dashCopy := make([]float64, len(cr.dashPattern))
	copy(dashCopy, cr.dashPattern)

	state := cairoState{
		currentColor:  cr.currentColor,
		sourcePattern: cr.sourcePattern,
		lineWidth:     cr.lineWidth,
		lineCap:       cr.lineCap,
		lineJoin:      cr.lineJoin,
		antialias:     cr.antialias,
		dashPattern:   dashCopy,
		dashOffset:    cr.dashOffset,
		miterLimit:    cr.miterLimit,
		fontFamily:    cr.fontFamily,
		fontSlant:     cr.fontSlant,
		fontWeight:    cr.fontWeight,
		fontSize:      cr.fontSize,
		translateX:    cr.translateX,
		translateY:    cr.translateY,
		rotation:      cr.rotation,
		scaleX:        cr.scaleX,
		scaleY:        cr.scaleY,
		matrix:        matrixCopy,
		clipPath:      cr.clipPath,
		hasClip:       cr.hasClip,
		clipMinX:      cr.clipMinX,
		clipMinY:      cr.clipMinY,
		clipMaxX:      cr.clipMaxX,
		clipMaxY:      cr.clipMaxY,
		fillRule:      cr.fillRule,
		operator:      cr.operator,
	}
	cr.stateStack = append(cr.stateStack, state)
}

// Restore restores the drawing state from the stack.
// This is equivalent to cairo_restore.
// If the stack is empty, this function does nothing.
func (cr *CairoRenderer) Restore() {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	if len(cr.stateStack) == 0 {
		return
	}

	// Pop the last state
	lastIdx := len(cr.stateStack) - 1
	state := cr.stateStack[lastIdx]
	cr.stateStack = cr.stateStack[:lastIdx]

	// Restore all state
	cr.currentColor = state.currentColor
	cr.sourcePattern = state.sourcePattern
	cr.lineWidth = state.lineWidth
	cr.lineCap = state.lineCap
	cr.lineJoin = state.lineJoin
	cr.antialias = state.antialias
	cr.dashPattern = state.dashPattern
	cr.dashOffset = state.dashOffset
	cr.miterLimit = state.miterLimit
	cr.fontFamily = state.fontFamily
	cr.fontSlant = state.fontSlant
	cr.fontWeight = state.fontWeight
	cr.fontSize = state.fontSize
	cr.translateX = state.translateX
	cr.translateY = state.translateY
	cr.rotation = state.rotation
	cr.scaleX = state.scaleX
	cr.scaleY = state.scaleY
	cr.matrix = state.matrix
	cr.clipPath = state.clipPath
	cr.hasClip = state.hasClip
	cr.clipMinX = state.clipMinX
	cr.clipMinY = state.clipMinY
	cr.clipMaxX = state.clipMaxX
	cr.clipMaxY = state.clipMaxY
	cr.fillRule = state.fillRule
	cr.operator = state.operator

	// Update text renderer to match restored state
	cr.textRenderer.SetFontSize(state.fontSize)
	style := cr.mapToFontStyle(state.fontSlant, state.fontWeight)
	cr.textRenderer.SetFont(state.fontFamily, style)
}

// transformPoint applies the current transformation to a point.
// Must be called while holding the mutex.
func (cr *CairoRenderer) transformPoint(x, y float64) (tx, ty float64) {
	// Use the matrix if available
	if cr.matrix != nil {
		return cr.matrix.TransformPoint(x, y)
	}

	// Fallback to simple transformations
	// Apply scale
	x *= cr.scaleX
	y *= cr.scaleY

	// Apply rotation
	if cr.rotation != 0 {
		cos := math.Cos(cr.rotation)
		sin := math.Sin(cr.rotation)
		x, y = x*cos-y*sin, x*sin+y*cos
	}

	// Apply translation
	tx = x + cr.translateX
	ty = y + cr.translateY

	return tx, ty
}

// IdentityMatrix resets the transformation matrix to identity.
func (cr *CairoRenderer) IdentityMatrix() {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	cr.translateX = 0
	cr.translateY = 0
	cr.rotation = 0
	cr.scaleX = 1.0
	cr.scaleY = 1.0
	cr.matrix = NewIdentityMatrix()
}

// GetMatrix returns a copy of the current transformation matrix.
// This is equivalent to cairo_get_matrix.
func (cr *CairoRenderer) GetMatrix() *CairoMatrix {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	if cr.matrix == nil {
		return NewIdentityMatrix()
	}
	return cr.matrix.Copy()
}

// SetMatrix sets the current transformation matrix.
// This is equivalent to cairo_set_matrix.
func (cr *CairoRenderer) SetMatrix(m *CairoMatrix) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	if m == nil {
		cr.matrix = NewIdentityMatrix()
	} else {
		cr.matrix = m.Copy()
	}
	// Sync the legacy transformation fields for backward compatibility
	cr.translateX = cr.matrix.X0
	cr.translateY = cr.matrix.Y0
	// Note: scaleX/Y and rotation cannot be easily extracted from a general matrix
	// We keep them at 1.0/0.0 when using SetMatrix directly
	cr.scaleX = 1.0
	cr.scaleY = 1.0
	cr.rotation = 0
}

// Transform multiplies the current transformation matrix by the given matrix.
// This is equivalent to cairo_transform.
func (cr *CairoRenderer) Transform(m *CairoMatrix) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	if m == nil {
		return
	}
	if cr.matrix == nil {
		cr.matrix = NewIdentityMatrix()
	}
	cr.matrix.Multiply(m)
	// Sync translation for backward compatibility
	cr.translateX = cr.matrix.X0
	cr.translateY = cr.matrix.Y0
}

// --- Clipping Functions ---
//
// Clipping is enforced for rectangular clip regions using Ebiten's SubImage
// functionality. When a clip is set, drawing operations use a SubImage
// representing the clipped rectangular area, which naturally restricts
// drawing to that region.
//
// Limitations:
// - Only rectangular clipping is supported (based on the path's bounding box)
// - Non-rectangular paths (arcs, curves) are clipped by their bounding rectangle
// - Clip intersection is not implemented; each Clip() call replaces the previous clip

// Clip establishes a new clip region by intersecting the current clip region
// with the current path and clears the path.
// This is equivalent to cairo_clip.
//
// Note: Clipping is enforced for the bounding rectangle of the path. Non-rectangular
// paths are clipped by their bounding box, not their exact shape.
func (cr *CairoRenderer) Clip() {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	if !cr.hasPath {
		return
	}

	// Store the current path as the clip path.
	// We're swapping out the path pointer and creating a new one,
	// so the clip path is safe from future modifications.
	cr.clipPath = cr.path
	cr.hasClip = true
	// Save the path bounds as clip bounds
	cr.clipMinX = cr.pathMinX
	cr.clipMinY = cr.pathMinY
	cr.clipMaxX = cr.pathMaxX
	cr.clipMaxY = cr.pathMaxY

	// Clear the current path (as per Cairo behavior)
	cr.path = &vector.Path{}
	cr.hasPath = false
	cr.pathBoundsInit = false
	cr.pathMinX = 0
	cr.pathMinY = 0
	cr.pathMaxX = 0
	cr.pathMaxY = 0
}

// ClipPreserve establishes a new clip region without clearing the current path.
// This is equivalent to cairo_clip_preserve.
//
// Note: Clipping is enforced for the bounding rectangle of the path. Non-rectangular
// paths are clipped by their bounding box, not their exact shape.
//
// Since Ebiten's vector.Path cannot be copied, we store the current path
// as the clip path and create a fresh path for continued drawing. The path state
// (current point, etc.) is preserved but the path data is now isolated.
func (cr *CairoRenderer) ClipPreserve() {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	if !cr.hasPath {
		return
	}

	// Store the current path as the clip path.
	// To avoid aliasing issues, we swap to a new path for drawing and
	// restore the current point so subsequent path operations can continue.
	cr.clipPath = cr.path
	cr.hasClip = true
	// Save the path bounds as clip bounds
	cr.clipMinX = cr.pathMinX
	cr.clipMinY = cr.pathMinY
	cr.clipMaxX = cr.pathMaxX
	cr.clipMaxY = cr.pathMaxY

	// Create a new path but preserve the current point for continued drawing.
	// This avoids the aliasing issue where modifying the current path
	// would also modify the clip path.
	currentX := cr.pathCurrentX
	currentY := cr.pathCurrentY
	cr.path = &vector.Path{}
	// Re-establish the current point on the new path
	cr.path.MoveTo(currentX, currentY)
	cr.pathStartX = currentX
	cr.pathStartY = currentY
	// Reset path bounds and add current point
	cr.pathMinX = currentX
	cr.pathMinY = currentY
	cr.pathMaxX = currentX
	cr.pathMaxY = currentY
	cr.hasPath = true // Explicitly set for consistency with MoveTo
}

// ResetClip resets the clip region to an infinitely large shape.
// This is equivalent to cairo_reset_clip.
//
// Note: See the Clipping Functions section comment for limitations.
func (cr *CairoRenderer) ResetClip() {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	cr.clipPath = nil
	cr.hasClip = false
	cr.clipMinX = 0
	cr.clipMinY = 0
	cr.clipMaxX = 0
	cr.clipMaxY = 0
}

// HasClip returns whether a clip region is currently set.
func (cr *CairoRenderer) HasClip() bool {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	return cr.hasClip
}

// --- Surface Management Functions ---

// CairoSurface represents a Cairo-compatible drawing surface.
// In our Ebiten-based implementation, this wraps an Ebiten image.
// This provides compatibility with Conky Lua scripts that use
// cairo_xlib_surface_create and related functions.
type CairoSurface struct {
	image     *ebiten.Image
	width     int
	height    int
	destroyed bool
	mu        sync.Mutex
}

// NewCairoSurface creates a new Cairo surface with the specified dimensions.
// This is the Ebiten equivalent of cairo_image_surface_create.
func NewCairoSurface(width, height int) *CairoSurface {
	if width <= 0 {
		width = 1
	}
	if height <= 0 {
		height = 1
	}
	return &CairoSurface{
		image:  ebiten.NewImage(width, height),
		width:  width,
		height: height,
	}
}

// NewCairoXlibSurface creates a surface compatible with X11 drawables.
// In our Ebiten implementation, we don't have direct X11 access, so this
// creates an Ebiten-backed surface with the specified dimensions.
// The display, drawable, and visual parameters are accepted for API
// compatibility but are not used in the Ebiten implementation.
//
// This is equivalent to cairo_xlib_surface_create(display, drawable, visual, width, height).
func NewCairoXlibSurface(display, drawable, visual uintptr, width, height int) *CairoSurface {
	// In Ebiten, we don't have direct X11 surface access.
	// We create an Ebiten image that will be composited onto the main screen.
	return NewCairoSurface(width, height)
}

// Image returns the underlying Ebiten image.
// This allows integration with the rendering loop.
func (s *CairoSurface) Image() *ebiten.Image {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.destroyed {
		return nil
	}
	return s.image
}

// Width returns the surface width.
func (s *CairoSurface) Width() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.width
}

// Height returns the surface height.
func (s *CairoSurface) Height() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.height
}

// IsDestroyed returns whether the surface has been destroyed.
func (s *CairoSurface) IsDestroyed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.destroyed
}

// Destroy releases the surface resources.
// This is equivalent to cairo_surface_destroy.
// After calling Destroy, the surface should not be used.
func (s *CairoSurface) Destroy() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.destroyed {
		return
	}
	s.destroyed = true
	// Dispose of the Ebiten image to free GPU resources
	if s.image != nil {
		s.image.Deallocate()
		s.image = nil
	}
}

// Flush completes any pending drawing operations.
// This is equivalent to cairo_surface_flush.
// In Ebiten, drawing is immediate, so this is a no-op for API compatibility.
func (s *CairoSurface) Flush() {
	// Ebiten handles drawing synchronization automatically.
	// This method exists for API compatibility with Cairo scripts.
}

// MarkDirty marks the entire surface as dirty.
// This is equivalent to cairo_surface_mark_dirty.
// In Ebiten, this is not needed as the GPU texture is managed automatically.
func (s *CairoSurface) MarkDirty() {
	// Ebiten manages GPU texture updates automatically.
	// This method exists for API compatibility with Cairo scripts.
}

// MarkDirtyRectangle marks a rectangular region as dirty.
// This is equivalent to cairo_surface_mark_dirty_rectangle.
// In Ebiten, this is not needed as the GPU texture is managed automatically.
func (s *CairoSurface) MarkDirtyRectangle(x, y, width, height int) {
	// Ebiten manages GPU texture updates automatically.
	// This method exists for API compatibility with Cairo scripts.
}

// NewCairoSurfaceFromPNG loads a PNG image file and creates a surface from it.
// This is equivalent to cairo_image_surface_create_from_png.
// Returns the surface and any error encountered during loading.
func NewCairoSurfaceFromPNG(filename string) (*CairoSurface, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("cairo_image_surface_create_from_png: %w", err)
	}
	defer file.Close()

	img, err := png.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("cairo_image_surface_create_from_png: failed to decode PNG: %w", err)
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	if width <= 0 {
		width = 1
	}
	if height <= 0 {
		height = 1
	}

	ebitenImg := ebiten.NewImage(width, height)

	// Convert the image to RGBA and write to Ebiten image
	rgbaImg := imageToRGBA(img)
	ebitenImg.WritePixels(rgbaImg.Pix)

	return &CairoSurface{
		image:  ebitenImg,
		width:  width,
		height: height,
	}, nil
}

// imageToRGBA converts any image.Image to *image.RGBA.
// This ensures consistent pixel format for Ebiten.
func imageToRGBA(img image.Image) *image.RGBA {
	if rgba, ok := img.(*image.RGBA); ok {
		return rgba
	}

	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			rgba.Set(x, y, img.At(x, y))
		}
	}
	return rgba
}

// WriteToPNG saves the surface to a PNG file.
// This is equivalent to cairo_surface_write_to_png.
// Returns any error encountered during saving.
func (s *CairoSurface) WriteToPNG(filename string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.destroyed {
		return fmt.Errorf("cairo_surface_write_to_png: surface has been destroyed")
	}
	if s.image == nil {
		return fmt.Errorf("cairo_surface_write_to_png: surface image is nil")
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("cairo_surface_write_to_png: %w", err)
	}
	defer file.Close()

	// Get pixels from Ebiten image
	// Create an RGBA image with the surface dimensions
	rgbaImg := image.NewRGBA(image.Rect(0, 0, s.width, s.height))
	s.image.ReadPixels(rgbaImg.Pix)

	if err := png.Encode(file, rgbaImg); err != nil {
		return fmt.Errorf("cairo_surface_write_to_png: failed to encode PNG: %w", err)
	}

	return nil
}

// CairoContext wraps a CairoRenderer with its associated surface.
// This provides the cairo_create/cairo_destroy pattern expected by Lua scripts.
type CairoContext struct {
	renderer  *CairoRenderer
	surface   *CairoSurface
	destroyed bool
	mu        sync.Mutex
}

// NewCairoContext creates a Cairo context for drawing on the given surface.
// This is equivalent to cairo_create(surface).
func NewCairoContext(surface *CairoSurface) *CairoContext {
	if surface == nil {
		return nil
	}

	// Get the image atomically - this will return nil if surface is destroyed
	image := surface.Image()
	if image == nil {
		return nil
	}

	renderer := NewCairoRenderer()
	renderer.SetScreen(image)

	return &CairoContext{
		renderer: renderer,
		surface:  surface,
	}
}

// Renderer returns the underlying CairoRenderer for drawing operations.
// Returns nil if the context has been destroyed.
func (ctx *CairoContext) Renderer() *CairoRenderer {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	if ctx.destroyed {
		return nil
	}
	return ctx.renderer
}

// Surface returns the associated surface.
// Returns nil if the context has been destroyed.
func (ctx *CairoContext) Surface() *CairoSurface {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	if ctx.destroyed {
		return nil
	}
	return ctx.surface
}

// IsDestroyed returns whether the context has been destroyed.
func (ctx *CairoContext) IsDestroyed() bool {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	return ctx.destroyed
}

// Destroy releases the context resources.
// This is equivalent to cairo_destroy.
// Note: This does NOT destroy the associated surface.
// The surface must be destroyed separately with surface.Destroy().
func (ctx *CairoContext) Destroy() {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	if ctx.destroyed {
		return
	}
	ctx.destroyed = true
	ctx.renderer = nil
	// Note: We don't destroy the surface here - that's the caller's responsibility
}

// --- Additional Cairo Functions ---

// InFill returns true if the given point is inside the current path's fill area.
// This is useful for hit testing in interactive applications.
func (cr *CairoRenderer) InFill(x, y float64) bool {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	// Check if point is within path bounds
	if cr.pathMinX == 0 && cr.pathMinY == 0 && cr.pathMaxX == 0 && cr.pathMaxY == 0 {
		return false
	}
	fx, fy := float32(x), float32(y)
	return fx >= cr.pathMinX && fx <= cr.pathMaxX && fy >= cr.pathMinY && fy <= cr.pathMaxY
}

// InStroke returns true if the given point is on the current path's stroke.
// This uses a simplified bounding box check with line width consideration.
func (cr *CairoRenderer) InStroke(x, y float64) bool {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	if cr.pathMinX == 0 && cr.pathMinY == 0 && cr.pathMaxX == 0 && cr.pathMaxY == 0 {
		return false
	}
	halfWidth := float32(cr.lineWidth) / 2
	fx, fy := float32(x), float32(y)
	return fx >= cr.pathMinX-halfWidth && fx <= cr.pathMaxX+halfWidth &&
		fy >= cr.pathMinY-halfWidth && fy <= cr.pathMaxY+halfWidth
}

// StrokeExtents returns the bounding box of what the current path would cover if stroked.
func (cr *CairoRenderer) StrokeExtents() (x1, y1, x2, y2 float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	halfWidth := float32(cr.lineWidth / 2)
	return float64(cr.pathMinX - halfWidth), float64(cr.pathMinY - halfWidth),
		float64(cr.pathMaxX + halfWidth), float64(cr.pathMaxY + halfWidth)
}

// FillExtents returns the bounding box of what the current path would cover if filled.
func (cr *CairoRenderer) FillExtents() (x1, y1, x2, y2 float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	return float64(cr.pathMinX), float64(cr.pathMinY), float64(cr.pathMaxX), float64(cr.pathMaxY)
}

// FontExtentsResult contains font metrics.
type FontExtentsResult struct {
	Ascent      float64
	Descent     float64
	Height      float64
	MaxXAdvance float64
	MaxYAdvance float64
}

// FontExtents returns the metrics for the current font.
func (cr *CairoRenderer) FontExtents() FontExtentsResult {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	// Simplified font metrics based on font size
	return FontExtentsResult{
		Ascent:      cr.fontSize * 0.8,
		Descent:     cr.fontSize * 0.2,
		Height:      cr.fontSize * 1.2,
		MaxXAdvance: cr.fontSize * 0.6,
		MaxYAdvance: 0,
	}
}

// NewSubPath starts a new sub-path without moving the current point.
// This is useful for creating disconnected path segments.
func (cr *CairoRenderer) NewSubPath() {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	// Reset current point tracking by setting to zero position
	cr.pathCurrentX = 0
	cr.pathCurrentY = 0
}

// PathSegmentType represents the type of a path segment.
type PathSegmentType int

const (
	// PathMoveTo is a move-to segment.
	PathMoveTo PathSegmentType = iota
	// PathLineTo is a line-to segment.
	PathLineTo
	// PathCurveTo is a curve-to segment.
	PathCurveTo
	// PathClose is a close-path segment.
	PathClose
	// PathArc is an arc segment (clockwise direction).
	PathArc
	// PathArcNegative is an arc segment (counter-clockwise direction).
	PathArcNegative
)

// PathSegment represents a segment of a path.
type PathSegment struct {
	Type PathSegmentType
	X, Y float64
	// For curves: control points
	X1, Y1, X2, Y2 float64
	// For arcs: center coordinates, radius, and angles
	CenterX, CenterY float64
	Radius           float64
	Angle1, Angle2   float64
}

// CopyPath returns a representation of the current path.
// The returned slice contains all path segments that have been added since the
// last NewPath call. This includes MoveTo, LineTo, CurveTo, ClosePath, Arc,
// and ArcNegative operations.
func (cr *CairoRenderer) CopyPath() []PathSegment {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	// Return a copy of the segments to avoid mutation
	if len(cr.pathSegments) == 0 {
		return []PathSegment{}
	}
	result := make([]PathSegment, len(cr.pathSegments))
	copy(result, cr.pathSegments)
	return result
}

// AppendPath appends the given path segments to the current path.
func (cr *CairoRenderer) AppendPath(segments []PathSegment) {
	for _, seg := range segments {
		switch seg.Type {
		case PathMoveTo:
			cr.MoveTo(seg.X, seg.Y)
		case PathLineTo:
			cr.LineTo(seg.X, seg.Y)
		case PathCurveTo:
			cr.CurveTo(seg.X1, seg.Y1, seg.X2, seg.Y2, seg.X, seg.Y)
		case PathClose:
			cr.ClosePath()
		case PathArc:
			cr.Arc(seg.CenterX, seg.CenterY, seg.Radius, seg.Angle1, seg.Angle2)
		case PathArcNegative:
			cr.ArcNegative(seg.CenterX, seg.CenterY, seg.Radius, seg.Angle1, seg.Angle2)
		}
	}
}

// UserToDevice transforms user-space coordinates to device-space.
func (cr *CairoRenderer) UserToDevice(x, y float64) (dx, dy float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	return cr.transformPointUnlocked(x, y)
}

// transformPointUnlocked transforms a point without acquiring the lock.
func (cr *CairoRenderer) transformPointUnlocked(x, y float64) (tx, ty float64) {
	// Apply scale
	x *= cr.scaleX
	y *= cr.scaleY

	// Apply rotation
	if cr.rotation != 0 {
		cos := math.Cos(cr.rotation)
		sin := math.Sin(cr.rotation)
		x, y = x*cos-y*sin, x*sin+y*cos
	}

	// Apply translation
	return x + cr.translateX, y + cr.translateY
}

// UserToDeviceDistance transforms a distance vector from user to device space.
func (cr *CairoRenderer) UserToDeviceDistance(dx, dy float64) (ddx, ddy float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	// Apply scale
	dx *= cr.scaleX
	dy *= cr.scaleY

	// Apply rotation
	if cr.rotation != 0 {
		cos := math.Cos(cr.rotation)
		sin := math.Sin(cr.rotation)
		dx, dy = dx*cos-dy*sin, dx*sin+dy*cos
	}

	return dx, dy
}

// DeviceToUser transforms device-space coordinates to user-space.
func (cr *CairoRenderer) DeviceToUser(dx, dy float64) (x, y float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	// Reverse translation
	dx -= cr.translateX
	dy -= cr.translateY

	// Reverse rotation
	if cr.rotation != 0 {
		cos := math.Cos(-cr.rotation)
		sin := math.Sin(-cr.rotation)
		dx, dy = dx*cos-dy*sin, dx*sin+dy*cos
	}

	// Reverse scale
	if cr.scaleX != 0 {
		dx /= cr.scaleX
	}
	if cr.scaleY != 0 {
		dy /= cr.scaleY
	}

	return dx, dy
}

// DeviceToUserDistance transforms a distance vector from device to user space.
func (cr *CairoRenderer) DeviceToUserDistance(ddx, ddy float64) (dx, dy float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	// Reverse rotation
	if cr.rotation != 0 {
		cos := math.Cos(-cr.rotation)
		sin := math.Sin(-cr.rotation)
		ddx, ddy = ddx*cos-ddy*sin, ddx*sin+ddy*cos
	}

	// Reverse scale
	if cr.scaleX != 0 {
		ddx /= cr.scaleX
	}
	if cr.scaleY != 0 {
		ddy /= cr.scaleY
	}

	return ddx, ddy
}

// GetFontFace returns the current font family name.
func (cr *CairoRenderer) GetFontFace() string {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	return cr.fontFamily
}

// GetFontSlant returns the current font slant.
func (cr *CairoRenderer) GetFontSlant() FontSlant {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	return cr.fontSlant
}

// GetFontWeight returns the current font weight.
func (cr *CairoRenderer) GetFontWeight() FontWeight {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	return cr.fontWeight
}

// --- Mask Functions ---
//
// Mask operations composite the current source using a mask pattern's alpha
// channel. Where the mask is opaque, the source is fully applied; where
// transparent, no source is applied.

// Mask paints the current source using the alpha channel of the given pattern
// as a mask. This is equivalent to cairo_mask.
//
// The mask pattern's alpha channel modulates the current source: where the
// mask is opaque (alpha = 1), the source is fully applied; where transparent
// (alpha = 0), no source is applied. Intermediate alpha values produce
// partial transparency.
//
// The current path is not affected by this operation.
func (cr *CairoRenderer) Mask(pattern *CairoPattern) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	if cr.screen == nil || pattern == nil {
		return
	}

	// Get screen bounds
	bounds := cr.screen.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	if w <= 0 || h <= 0 {
		return
	}

	// Create a temporary image to hold the masked result
	tempImg := ebiten.NewImage(w, h)
	defer tempImg.Deallocate()

	// For each pixel, compute the mask alpha and blend accordingly
	// We use a pixel-by-pixel approach for accuracy
	pixels := make([]byte, w*h*4)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			// Get mask alpha at this point
			maskColor := pattern.ColorAtPoint(float64(x), float64(y))
			maskAlpha := float64(maskColor.A) / 255.0

			// Compute final color: source color with alpha modulated by mask
			finalR := uint8(float64(cr.currentColor.R) * maskAlpha)
			finalG := uint8(float64(cr.currentColor.G) * maskAlpha)
			finalB := uint8(float64(cr.currentColor.B) * maskAlpha)
			finalA := uint8(float64(cr.currentColor.A) * maskAlpha)

			// Write to pixel buffer (RGBA format)
			offset := (y*w + x) * 4
			pixels[offset] = finalR
			pixels[offset+1] = finalG
			pixels[offset+2] = finalB
			pixels[offset+3] = finalA
		}
	}

	tempImg.WritePixels(pixels)

	// Get clipped screen for compositing
	screen, _, _ := cr.getClippedScreen()

	// Composite the masked image onto the screen
	screen.DrawImage(tempImg, &ebiten.DrawImageOptions{
		Blend: cr.getEbitenBlend(),
	})
}

// MaskSurface paints the current source using the alpha channel of the given
// surface as a mask. This is equivalent to cairo_mask_surface.
//
// The surface is placed at (surfaceX, surfaceY) in user-space coordinates.
// The alpha channel of the surface modulates the current source color.
//
// Implementation note: This uses Ebiten's DrawImage with color matrix to
// achieve the masking effect without requiring ReadPixels, which has
// limitations in Ebiten's execution model.
func (cr *CairoRenderer) MaskSurface(surface *CairoSurface, surfaceX, surfaceY float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	if cr.screen == nil || surface == nil || surface.IsDestroyed() {
		return
	}

	maskImg := surface.Image()
	if maskImg == nil {
		return
	}

	// Get mask bounds
	maskBounds := maskImg.Bounds()
	maskW := maskBounds.Dx()
	maskH := maskBounds.Dy()

	if maskW <= 0 || maskH <= 0 {
		return
	}

	// Apply transformation to surface position
	tx, ty := cr.transformPointUnlocked(surfaceX, surfaceY)

	// Create a temporary image the size of the mask to hold the colored result
	tempImg := ebiten.NewImage(maskW, maskH)
	defer tempImg.Deallocate()

	// Fill temp image with the source color
	tempImg.Fill(cr.currentColor)

	// Get clipped screen for compositing
	screen, _, _ := cr.getClippedScreen()

	// Use Ebiten's blend modes to apply the mask.
	// We draw the colored image using the mask's alpha channel.
	// The BlendSourceIn blend mode uses: result = source * dest_alpha
	// So we first draw the mask to establish alpha, then draw color with SourceIn.

	// Alternative approach: Use a ColorMatrix to modulate the colored image
	// by the mask's alpha. We draw the mask first, then draw the color on top.

	// Create a destination image for compositing
	compositeImg := ebiten.NewImage(maskW, maskH)
	defer compositeImg.Deallocate()

	// Draw the mask to establish the alpha channel
	compositeImg.DrawImage(maskImg, nil)

	// Now draw the colored image with SourceIn blending
	// SourceIn: result = source * dest_alpha (uses destination alpha as mask)
	opts := &ebiten.DrawImageOptions{
		Blend: ebiten.BlendSourceIn,
	}
	compositeImg.DrawImage(tempImg, opts)

	// Finally, draw the composite result to the screen at the specified position
	screenOpts := &ebiten.DrawImageOptions{
		Blend: cr.getEbitenBlend(),
	}
	screenOpts.GeoM.Translate(tx, ty)
	screen.DrawImage(compositeImg, screenOpts)
}

// --- Group Rendering Functions ---
//
// Group rendering allows drawing to a temporary surface that can later be
// composited back to the main surface. This is useful for complex effects
// like transparency, blur, or when you need to capture drawing operations.

// PushGroup temporarily redirects drawing to an internal group surface.
// This is equivalent to cairo_push_group.
//
// All subsequent drawing operations will be directed to the group surface
// until PopGroup or PopGroupToSource is called. Groups can be nested.
func (cr *CairoRenderer) PushGroup() {
	cr.PushGroupWithContent(CairoContentColorAlpha)
}

// PushGroupWithContent temporarily redirects drawing to a group surface with
// the specified content type. This is equivalent to cairo_push_group_with_content.
//
// The content type determines the properties of the group surface:
// - CairoContentColor: RGB only (no alpha channel)
// - CairoContentAlpha: Alpha only (no color)
// - CairoContentColorAlpha: Full RGBA (default)
func (cr *CairoRenderer) PushGroupWithContent(content CairoContent) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	if cr.screen == nil {
		return
	}

	// Get screen dimensions
	bounds := cr.screen.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	if w <= 0 || h <= 0 {
		return
	}

	// Create a new group surface
	groupSurface := ebiten.NewImage(w, h)

	// For alpha-only content, we still use RGBA but will only use alpha channel
	// For color-only content, we start with opaque background
	if content == CairoContentColor {
		groupSurface.Fill(color.RGBA{A: 255})
	}

	// Save current screen and push group state
	state := &groupState{
		surface:        groupSurface,
		previousScreen: cr.screen,
		content:        content,
	}
	cr.groupStack = append(cr.groupStack, state)

	// Redirect drawing to the group surface
	cr.screen = groupSurface
}

// PopGroup terminates the current group and returns its contents as a pattern.
// This is equivalent to cairo_pop_group.
//
// The returned pattern can be used with SetSource to composite the group
// contents onto the original surface. Returns nil if no group was pushed.
func (cr *CairoRenderer) PopGroup() *CairoPattern {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	return cr.popGroupUnlocked()
}

// popGroupUnlocked pops the group without acquiring the mutex.
// Returns a surface pattern containing the group contents.
// Must be called while holding the mutex.
func (cr *CairoRenderer) popGroupUnlocked() *CairoPattern {
	if len(cr.groupStack) == 0 {
		return nil
	}

	// Pop the group state
	lastIdx := len(cr.groupStack) - 1
	state := cr.groupStack[lastIdx]
	cr.groupStack = cr.groupStack[:lastIdx]

	// Restore the previous screen
	cr.screen = state.previousScreen

	// Create a surface pattern from the group surface
	// The pattern will contain the group's image data
	pattern := &CairoPattern{
		patternType: PatternTypeSurface,
		surface:     state.surface,
	}

	return pattern
}

// PopGroupToSource terminates the current group and sets it as the source.
// This is equivalent to cairo_pop_group_to_source.
//
// This is a convenience function equivalent to:
//
//	pattern := cr.PopGroup()
//	cr.SetSource(pattern)
func (cr *CairoRenderer) PopGroupToSource() {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	pattern := cr.popGroupUnlocked()
	if pattern != nil {
		cr.sourcePattern = pattern
	}
}

// GetGroupTarget returns the current target surface.
// If a group is active, this returns the group surface; otherwise
// it returns the original surface.
// This is equivalent to cairo_get_group_target.
func (cr *CairoRenderer) GetGroupTarget() *ebiten.Image {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	return cr.screen
}

// HasGroup returns whether a group is currently active.
func (cr *CairoRenderer) HasGroup() bool {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	return len(cr.groupStack) > 0
}

// --- Source Surface Functions ---
//
// These functions allow using a surface (image) as the source for drawing
// operations. This enables painting images or compositing surfaces.

// SetSourceSurface sets a surface as the source for subsequent drawing operations.
// The surface is placed at (x, y) in user-space coordinates.
// This is equivalent to cairo_set_source_surface.
//
// After calling SetSourceSurface, operations like Paint() will draw the surface
// content instead of a solid color.
func (cr *CairoRenderer) SetSourceSurface(surface *CairoSurface, x, y float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	if surface == nil || surface.IsDestroyed() {
		cr.sourceSurface = nil
		cr.hasSourceSurface = false
		return
	}

	cr.sourceSurface = surface.Image()
	cr.sourceSurfaceX = x
	cr.sourceSurfaceY = y
	cr.hasSourceSurface = cr.sourceSurface != nil

	// Create a surface pattern and set it as source
	if cr.hasSourceSurface {
		cr.sourcePattern = &CairoPattern{
			patternType: PatternTypeSurface,
			surface:     cr.sourceSurface,
			x0:          x,
			y0:          y,
		}
	}
}

// SetSourceSurfaceImage sets an Ebiten image as the source for drawing operations.
// This is a convenience function for cases where you have a raw Ebiten image
// instead of a CairoSurface.
func (cr *CairoRenderer) SetSourceSurfaceImage(image *ebiten.Image, x, y float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	cr.sourceSurface = image
	cr.sourceSurfaceX = x
	cr.sourceSurfaceY = y
	cr.hasSourceSurface = image != nil

	// Create a surface pattern and set it as source
	if cr.hasSourceSurface {
		cr.sourcePattern = &CairoPattern{
			patternType: PatternTypeSurface,
			surface:     image,
			x0:          x,
			y0:          y,
		}
	}
}
