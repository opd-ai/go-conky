// Package render provides Ebiten-based rendering capabilities for conky-go.
// This file implements the Cairo compatibility layer that translates
// Cairo drawing commands to Ebiten vector graphics.
package render

import (
	"image/color"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

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
	path         *vector.Path
	pathStartX   float32
	pathStartY   float32
	pathCurrentX float32
	pathCurrentY float32
	hasPath      bool
	// Path bounding box (tracked during path operations)
	pathMinX float32
	pathMinY float32
	pathMaxX float32
	pathMaxY float32
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
	currentColor color.RGBA
	lineWidth    float32
	lineCap      LineCap
	lineJoin     LineJoin
	antialias    bool
	fontFamily   string
	fontSlant    FontSlant
	fontWeight   FontWeight
	fontSize     float64
	translateX   float64
	translateY   float64
	rotation     float64
	scaleX       float64
	scaleY       float64
	clipPath     *vector.Path
	hasClip      bool
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

// --- Path Building Functions ---

// expandPathBounds updates the path bounding box to include the given point.
// Must be called while holding the mutex.
func (cr *CairoRenderer) expandPathBounds(x, y float32) {
	if !cr.hasPath {
		// First point - initialize bounds
		cr.pathMinX = x
		cr.pathMinY = y
		cr.pathMaxX = x
		cr.pathMaxY = y
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

	state := cairoState{
		currentColor: cr.currentColor,
		lineWidth:    cr.lineWidth,
		lineCap:      cr.lineCap,
		lineJoin:     cr.lineJoin,
		antialias:    cr.antialias,
		fontFamily:   cr.fontFamily,
		fontSlant:    cr.fontSlant,
		fontWeight:   cr.fontWeight,
		fontSize:     cr.fontSize,
		translateX:   cr.translateX,
		translateY:   cr.translateY,
		rotation:     cr.rotation,
		scaleX:       cr.scaleX,
		scaleY:       cr.scaleY,
		clipPath:     cr.clipPath,
		hasClip:      cr.hasClip,
		clipMinX:     cr.clipMinX,
		clipMinY:     cr.clipMinY,
		clipMaxX:     cr.clipMaxX,
		clipMaxY:     cr.clipMaxY,
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
	cr.lineWidth = state.lineWidth
	cr.lineCap = state.lineCap
	cr.lineJoin = state.lineJoin
	cr.antialias = state.antialias
	cr.fontFamily = state.fontFamily
	cr.fontSlant = state.fontSlant
	cr.fontWeight = state.fontWeight
	cr.fontSize = state.fontSize
	cr.translateX = state.translateX
	cr.translateY = state.translateY
	cr.rotation = state.rotation
	cr.scaleX = state.scaleX
	cr.scaleY = state.scaleY
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
	// Early return if transformation is identity
	if cr.scaleX == 1 && cr.scaleY == 1 && cr.rotation == 0 && cr.translateX == 0 && cr.translateY == 0 {
		return x, y
	}

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
