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
	mu           sync.Mutex
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

// NewPath clears the current path and starts a new one.
// This is equivalent to cairo_new_path.
func (cr *CairoRenderer) NewPath() {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	cr.path = &vector.Path{}
	cr.hasPath = false
}

// MoveTo begins a new sub-path at the given point.
// This is equivalent to cairo_move_to.
func (cr *CairoRenderer) MoveTo(x, y float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
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
		cr.path.MoveTo(0, 0)
		cr.pathStartX = 0
		cr.pathStartY = 0
		cr.hasPath = true
	}
	cr.path.CubicTo(float32(x1), float32(y1), float32(x2), float32(y2), float32(x3), float32(y3))
	cr.pathCurrentX = float32(x3)
	cr.pathCurrentY = float32(y3)
}

// Rectangle adds a closed rectangular sub-path.
// This is equivalent to cairo_rectangle.
func (cr *CairoRenderer) Rectangle(x, y, width, height float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

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
func (cr *CairoRenderer) DrawLine(x1, y1, x2, y2 float64) {
	cr.NewPath()
	cr.MoveTo(x1, y1)
	cr.LineTo(x2, y2)
	cr.Stroke()
}

// DrawRectangle draws a stroked rectangle.
// This is a convenience function that combines Rectangle and Stroke.
func (cr *CairoRenderer) DrawRectangle(x, y, width, height float64) {
	cr.NewPath()
	cr.Rectangle(x, y, width, height)
	cr.Stroke()
}

// FillRectangle draws a filled rectangle.
// This is a convenience function that combines Rectangle and Fill.
func (cr *CairoRenderer) FillRectangle(x, y, width, height float64) {
	cr.NewPath()
	cr.Rectangle(x, y, width, height)
	cr.Fill()
}

// DrawCircle draws a stroked circle.
// This is a convenience function that combines Arc and Stroke.
func (cr *CairoRenderer) DrawCircle(xc, yc, radius float64) {
	cr.NewPath()
	cr.Arc(xc, yc, radius, 0, 2*math.Pi)
	cr.ClosePath()
	cr.Stroke()
}

// FillCircle draws a filled circle.
// This is a convenience function that combines Arc and Fill.
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
