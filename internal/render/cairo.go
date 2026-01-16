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
	mu     sync.Mutex
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
	mu         sync.Mutex
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
}

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
// This is a no-op placeholder for Cairo compatibility.
func (cr *CairoRenderer) SetFillRule(_ int) {
	// Ebiten's vector package uses even-odd fill rule by default
}

// GetFillRule returns the current fill rule.
func (cr *CairoRenderer) GetFillRule() int {
	return 0 // CAIRO_FILL_RULE_WINDING
}

// SetOperator sets the compositing operator.
// This is a no-op placeholder for Cairo compatibility.
func (cr *CairoRenderer) SetOperator(_ int) {
	// Ebiten uses its own blend modes
}

// GetOperator returns the current operator.
func (cr *CairoRenderer) GetOperator() int {
	return 2 // CAIRO_OPERATOR_OVER (default)
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

// NewPath clears the current path and starts a new one.
// This is equivalent to cairo_new_path.
func (cr *CairoRenderer) NewPath() {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	cr.path = &vector.Path{}
	cr.hasPath = false
	cr.pathBoundsInit = false
	cr.pathMinX = 0
	cr.pathMinY = 0
	cr.pathMaxX = 0
	cr.pathMaxY = 0
}

// MoveTo begins a new sub-path at the given point.
// This is equivalent to cairo_move_to.
func (cr *CairoRenderer) MoveTo(x, y float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	cr.expandPathBounds(float32(x), float32(y))
	cr.path.MoveTo(float32(x), float32(y))
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
		cr.pathStartX = float32(x)
		cr.pathStartY = float32(y)
		cr.hasPath = true
	} else {
		cr.path.LineTo(float32(x), float32(y))
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
		cr.pathStartX = float32(startX)
		cr.pathStartY = float32(startY)
		cr.hasPath = true
	} else {
		cr.path.LineTo(float32(startX), float32(startY))
	}

	// Add arc using Ebiten's Arc method
	cr.path.Arc(float32(xc), float32(yc), float32(radius), float32(angle1), float32(angle2), vector.Clockwise)

	// Update current position
	endX := xc + radius*math.Cos(angle2)
	endY := yc + radius*math.Sin(angle2)
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
		cr.pathStartX = float32(startX)
		cr.pathStartY = float32(startY)
		cr.hasPath = true
	} else {
		cr.path.LineTo(float32(startX), float32(startY))
	}

	// Add arc in counter-clockwise direction
	cr.path.Arc(float32(xc), float32(yc), float32(radius), float32(angle1), float32(angle2), vector.CounterClockwise)

	// Update current position
	endX := xc + radius*math.Cos(angle2)
	endY := yc + radius*math.Sin(angle2)
	cr.pathCurrentX = float32(endX)
	cr.pathCurrentY = float32(endY)
}

// CurveTo adds a cubic Bézier curve to the path.
// This is equivalent to cairo_curve_to.
// If there is no current point, it starts from (0,0) as per Cairo convention.
func (cr *CairoRenderer) CurveTo(x1, y1, x2, y2, x3, y3 float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	if !cr.hasPath {
		// Cairo starts from (0,0) if no current point exists
		cr.expandPathBounds(0, 0)
		cr.path.MoveTo(0, 0)
		cr.pathStartX = 0
		cr.pathStartY = 0
		cr.hasPath = true
	}
	// Expand bounds to include all control points and end point
	cr.expandPathBounds(float32(x1), float32(y1))
	cr.expandPathBounds(float32(x2), float32(y2))
	cr.expandPathBounds(float32(x3), float32(y3))
	cr.path.CubicTo(float32(x1), float32(y1), float32(x2), float32(y2), float32(x3), float32(y3))
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

	cr.pathStartX = float32(x)
	cr.pathStartY = float32(y)
	cr.pathCurrentX = float32(x)
	cr.pathCurrentY = float32(y)
	cr.hasPath = true
}

// --- Relative Path Functions ---

// RelMoveTo moves the current point by a relative offset.
// This is equivalent to cairo_rel_move_to.
// If there is no current point, this function does nothing.
func (cr *CairoRenderer) RelMoveTo(dx, dy float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	if !cr.hasPath {
		// Cairo requires a current point for relative moves
		return
	}
	newX := float64(cr.pathCurrentX) + dx
	newY := float64(cr.pathCurrentY) + dy
	cr.expandPathBounds(float32(newX), float32(newY))
	cr.path.MoveTo(float32(newX), float32(newY))
	cr.pathStartX = float32(newX)
	cr.pathStartY = float32(newY)
	cr.pathCurrentX = float32(newX)
	cr.pathCurrentY = float32(newY)
}

// RelLineTo draws a line from the current point by a relative offset.
// This is equivalent to cairo_rel_line_to.
// If there is no current point, this function does nothing.
func (cr *CairoRenderer) RelLineTo(dx, dy float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	if !cr.hasPath {
		// Cairo requires a current point for relative line
		return
	}
	newX := float64(cr.pathCurrentX) + dx
	newY := float64(cr.pathCurrentY) + dy
	cr.expandPathBounds(float32(newX), float32(newY))
	cr.path.LineTo(float32(newX), float32(newY))
	cr.pathCurrentX = float32(newX)
	cr.pathCurrentY = float32(newY)
}

// RelCurveTo adds a cubic Bézier curve relative to the current point.
// This is equivalent to cairo_rel_curve_to.
// If there is no current point, this function does nothing.
func (cr *CairoRenderer) RelCurveTo(dx1, dy1, dx2, dy2, dx3, dy3 float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	if !cr.hasPath {
		// Cairo requires a current point for relative curves
		return
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
	cr.screen.DrawTriangles(vertices, indices, emptySubImage, &ebiten.DrawTrianglesOptions{
		AntiAlias: cr.antialias,
	})

	// Clear the path after stroking
	cr.path = &vector.Path{}
	cr.hasPath = false
	cr.pathBoundsInit = false
	cr.pathMinX = 0
	cr.pathMinY = 0
	cr.pathMaxX = 0
	cr.pathMaxY = 0
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
	cr.screen.DrawTriangles(vertices, indices, emptySubImage, &ebiten.DrawTrianglesOptions{
		AntiAlias: cr.antialias,
	})

	// Clear the path after filling
	cr.path = &vector.Path{}
	cr.hasPath = false
	cr.pathBoundsInit = false
	cr.pathMinX = 0
	cr.pathMinY = 0
	cr.pathMaxX = 0
	cr.pathMaxY = 0
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
	cr.screen.DrawTriangles(vertices, indices, emptySubImage, &ebiten.DrawTrianglesOptions{
		AntiAlias: cr.antialias,
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
	cr.screen.DrawTriangles(vertices, indices, emptySubImage, &ebiten.DrawTrianglesOptions{
		AntiAlias: cr.antialias,
	})
}

// Paint fills the entire surface with the current color.
// This is equivalent to cairo_paint.
func (cr *CairoRenderer) Paint() {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	if cr.screen == nil {
		return
	}

	cr.screen.Fill(cr.currentColor)
}

// PaintWithAlpha fills the entire surface with the current color at the given alpha.
// This is equivalent to cairo_paint_with_alpha.
func (cr *CairoRenderer) PaintWithAlpha(alpha float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	if cr.screen == nil {
		return
	}

	clr := cr.currentColor
	clr.A = clampToByte(alpha)
	cr.screen.Fill(clr)
}

// --- Convenience Drawing Functions ---

// DrawLine draws a line from (x1,y1) to (x2,y2) with the current color and line width.
// This is a convenience function that combines MoveTo, LineTo, and Stroke.
// DrawLine draws a line from (x1,y1) to (x2,y2) with the current color and line width.
// This is a convenience function that combines MoveTo, LineTo, and Stroke.
// Note: This function is NOT atomic - each internal method call acquires and releases
// the mutex independently. For atomic operations, use explicit locking at the caller level.
func (cr *CairoRenderer) DrawLine(x1, y1, x2, y2 float64) {
	cr.NewPath()
	cr.MoveTo(x1, y1)
	cr.LineTo(x2, y2)
	cr.Stroke()
}

// DrawRectangle draws a stroked rectangle.
// This is a convenience function that combines Rectangle and Stroke.
// Note: This function is NOT atomic - each internal method call acquires and releases
// the mutex independently. For atomic operations, use explicit locking at the caller level.
func (cr *CairoRenderer) DrawRectangle(x, y, width, height float64) {
	cr.NewPath()
	cr.Rectangle(x, y, width, height)
	cr.Stroke()
}

// FillRectangle draws a filled rectangle.
// This is a convenience function that combines Rectangle and Fill.
// Note: This function is NOT atomic - each internal method call acquires and releases
// the mutex independently. For atomic operations, use explicit locking at the caller level.
func (cr *CairoRenderer) FillRectangle(x, y, width, height float64) {
	cr.NewPath()
	cr.Rectangle(x, y, width, height)
	cr.Fill()
}

// DrawCircle draws a stroked circle.
// This is a convenience function that combines Arc and Stroke.
// Note: This function is NOT atomic - each internal method call acquires and releases
// the mutex independently. For atomic operations, use explicit locking at the caller level.
func (cr *CairoRenderer) DrawCircle(xc, yc, radius float64) {
	cr.NewPath()
	cr.Arc(xc, yc, radius, 0, 2*math.Pi)
	cr.ClosePath()
	cr.Stroke()
}

// FillCircle draws a filled circle.
// This is a convenience function that combines Arc and Fill.
// Note: This function is NOT atomic - each internal method call acquires and releases
// the mutex independently. For atomic operations, use explicit locking at the caller level.
func (cr *CairoRenderer) FillCircle(xc, yc, radius float64) {
	cr.NewPath()
	cr.Arc(xc, yc, radius, 0, 2*math.Pi)
	cr.ClosePath()
	cr.Fill()
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
// IMPORTANT: Clipping is currently a partial implementation.
// The clip region is tracked (stored in clipPath/hasClip) but NOT enforced
// during drawing operations. This means calling Clip() will record the clip
// region for API compatibility, but subsequent drawing will NOT be restricted
// to the clip area.
//
// Full clipping support would require either:
// - Using Ebiten's SubImage for rectangular clips only
// - Implementing stencil buffer or alpha mask clipping for arbitrary paths
//
// For now, scripts that use clipping will execute without errors, but the
// visual clipping effect will not be applied.

// Clip establishes a new clip region by intersecting the current clip region
// with the current path and clears the path.
// This is equivalent to cairo_clip.
//
// WARNING: Clipping is NOT currently enforced during drawing operations.
// The clip region is recorded but drawing will not be restricted to the clip area.
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
// WARNING: Clipping is NOT currently enforced during drawing operations.
// The clip region is recorded but drawing will not be restricted to the clip area.
//
// Note: Since Ebiten's vector.Path cannot be copied, we store the current path
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
)

// PathSegment represents a segment of a path.
type PathSegment struct {
	Type PathSegmentType
	X, Y float64
	// For curves: control points
	X1, Y1, X2, Y2 float64
}

// CopyPath returns a simplified representation of the current path.
// Note: This returns a basic representation - full path iteration is not supported.
func (cr *CairoRenderer) CopyPath() []PathSegment {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	// Return empty slice - full path iteration not implemented
	return []PathSegment{}
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
