// Package lua provides Golua integration for conky-go.
// This file implements Cairo drawing function bindings that allow
// Lua scripts to call Cairo-compatible drawing operations.
package lua

import (
	"fmt"

	rt "github.com/arnodel/golua/runtime"

	"github.com/opd-ai/go-conky/internal/render"
)

// CairoBindings provides Cairo drawing function bindings for Lua.
// It manages a CairoRenderer instance and exposes Cairo-compatible
// functions to the Lua environment.
// The renderer field is immutable after initialization.
type CairoBindings struct {
	runtime  *ConkyRuntime
	renderer *render.CairoRenderer
}

// NewCairoBindings creates a new CairoBindings instance and registers
// all Cairo functions in the provided Lua runtime.
func NewCairoBindings(runtime *ConkyRuntime) (*CairoBindings, error) {
	if runtime == nil {
		return nil, fmt.Errorf("runtime cannot be nil")
	}

	cb := &CairoBindings{
		runtime:  runtime,
		renderer: render.NewCairoRenderer(),
	}

	cb.registerFunctions()
	return cb, nil
}

// Renderer returns the underlying CairoRenderer.
// This allows the rendering loop to set the screen and access the renderer.
// The renderer is immutable after CairoBindings creation, so no locking is needed.
func (cb *CairoBindings) Renderer() *render.CairoRenderer {
	return cb.renderer
}

// registerFunctions registers all Cairo drawing functions in the Lua environment.
func (cb *CairoBindings) registerFunctions() {
	// Color functions
	cb.runtime.SetGoFunction("cairo_set_source_rgb", cb.setSourceRGB, 3, false)
	cb.runtime.SetGoFunction("cairo_set_source_rgba", cb.setSourceRGBA, 4, false)

	// Line style functions
	cb.runtime.SetGoFunction("cairo_set_line_width", cb.setLineWidth, 1, false)
	cb.runtime.SetGoFunction("cairo_set_line_cap", cb.setLineCap, 1, false)
	cb.runtime.SetGoFunction("cairo_set_line_join", cb.setLineJoin, 1, false)
	cb.runtime.SetGoFunction("cairo_set_antialias", cb.setAntialias, 1, false)

	// Path building functions
	cb.runtime.SetGoFunction("cairo_new_path", cb.newPath, 0, false)
	cb.runtime.SetGoFunction("cairo_move_to", cb.moveTo, 2, false)
	cb.runtime.SetGoFunction("cairo_line_to", cb.lineTo, 2, false)
	cb.runtime.SetGoFunction("cairo_close_path", cb.closePath, 0, false)
	cb.runtime.SetGoFunction("cairo_arc", cb.arc, 5, false)
	cb.runtime.SetGoFunction("cairo_arc_negative", cb.arcNegative, 5, false)
	cb.runtime.SetGoFunction("cairo_curve_to", cb.curveTo, 6, false)
	cb.runtime.SetGoFunction("cairo_rectangle", cb.rectangle, 4, false)

	// Drawing functions
	cb.runtime.SetGoFunction("cairo_stroke", cb.stroke, 0, false)
	cb.runtime.SetGoFunction("cairo_fill", cb.fill, 0, false)
	cb.runtime.SetGoFunction("cairo_stroke_preserve", cb.strokePreserve, 0, false)
	cb.runtime.SetGoFunction("cairo_fill_preserve", cb.fillPreserve, 0, false)
	cb.runtime.SetGoFunction("cairo_paint", cb.paint, 0, false)
	cb.runtime.SetGoFunction("cairo_paint_with_alpha", cb.paintWithAlpha, 1, false)

	// Text functions
	cb.runtime.SetGoFunction("cairo_select_font_face", cb.selectFontFace, 3, false)
	cb.runtime.SetGoFunction("cairo_set_font_size", cb.setFontSize, 1, false)
	cb.runtime.SetGoFunction("cairo_show_text", cb.showText, 1, false)
	cb.runtime.SetGoFunction("cairo_text_extents", cb.textExtents, 1, false)

	// Transformation functions
	cb.runtime.SetGoFunction("cairo_translate", cb.translate, 2, false)
	cb.runtime.SetGoFunction("cairo_rotate", cb.rotate, 1, false)
	cb.runtime.SetGoFunction("cairo_scale", cb.scale, 2, false)
	cb.runtime.SetGoFunction("cairo_save", cb.save, 0, false)
	cb.runtime.SetGoFunction("cairo_restore", cb.restore, 0, false)
	cb.runtime.SetGoFunction("cairo_identity_matrix", cb.identityMatrix, 0, false)

	// Register Cairo constants
	cb.registerConstants()
}

// registerConstants registers Cairo constants in the Lua environment.
func (cb *CairoBindings) registerConstants() {
	// Line cap constants
	cb.runtime.SetGlobal("CAIRO_LINE_CAP_BUTT", rt.IntValue(int64(render.LineCapButt)))
	cb.runtime.SetGlobal("CAIRO_LINE_CAP_ROUND", rt.IntValue(int64(render.LineCapRound)))
	cb.runtime.SetGlobal("CAIRO_LINE_CAP_SQUARE", rt.IntValue(int64(render.LineCapSquare)))

	// Line join constants
	cb.runtime.SetGlobal("CAIRO_LINE_JOIN_MITER", rt.IntValue(int64(render.LineJoinMiter)))
	cb.runtime.SetGlobal("CAIRO_LINE_JOIN_ROUND", rt.IntValue(int64(render.LineJoinRound)))
	cb.runtime.SetGlobal("CAIRO_LINE_JOIN_BEVEL", rt.IntValue(int64(render.LineJoinBevel)))

	// Antialias constants
	cb.runtime.SetGlobal("CAIRO_ANTIALIAS_NONE", rt.IntValue(0))
	cb.runtime.SetGlobal("CAIRO_ANTIALIAS_DEFAULT", rt.IntValue(1))

	// Font slant constants
	cb.runtime.SetGlobal("CAIRO_FONT_SLANT_NORMAL", rt.IntValue(int64(render.FontSlantNormal)))
	cb.runtime.SetGlobal("CAIRO_FONT_SLANT_ITALIC", rt.IntValue(int64(render.FontSlantItalic)))
	cb.runtime.SetGlobal("CAIRO_FONT_SLANT_OBLIQUE", rt.IntValue(int64(render.FontSlantOblique)))

	// Font weight constants
	cb.runtime.SetGlobal("CAIRO_FONT_WEIGHT_NORMAL", rt.IntValue(int64(render.FontWeightNormal)))
	cb.runtime.SetGlobal("CAIRO_FONT_WEIGHT_BOLD", rt.IntValue(int64(render.FontWeightBold)))
}

// --- Color Functions ---

// setSourceRGB handles cairo_set_source_rgb(r, g, b)
func (cb *CairoBindings) setSourceRGB(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	r, err := c.FloatArg(0)
	if err != nil {
		return nil, fmt.Errorf("cairo_set_source_rgb: r: %w", err)
	}
	g, err := c.FloatArg(1)
	if err != nil {
		return nil, fmt.Errorf("cairo_set_source_rgb: g: %w", err)
	}
	b, err := c.FloatArg(2)
	if err != nil {
		return nil, fmt.Errorf("cairo_set_source_rgb: b: %w", err)
	}

	cb.renderer.SetSourceRGB(r, g, b)
	return c.Next(), nil
}

// setSourceRGBA handles cairo_set_source_rgba(r, g, b, a)
func (cb *CairoBindings) setSourceRGBA(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	r, err := c.FloatArg(0)
	if err != nil {
		return nil, fmt.Errorf("cairo_set_source_rgba: r: %w", err)
	}
	g, err := c.FloatArg(1)
	if err != nil {
		return nil, fmt.Errorf("cairo_set_source_rgba: g: %w", err)
	}
	b, err := c.FloatArg(2)
	if err != nil {
		return nil, fmt.Errorf("cairo_set_source_rgba: b: %w", err)
	}
	a, err := c.FloatArg(3)
	if err != nil {
		return nil, fmt.Errorf("cairo_set_source_rgba: a: %w", err)
	}

	cb.renderer.SetSourceRGBA(r, g, b, a)
	return c.Next(), nil
}

// --- Line Style Functions ---

// setLineWidth handles cairo_set_line_width(width)
func (cb *CairoBindings) setLineWidth(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	width, err := c.FloatArg(0)
	if err != nil {
		return nil, fmt.Errorf("cairo_set_line_width: %w", err)
	}

	cb.renderer.SetLineWidth(width)
	return c.Next(), nil
}

// setLineCap handles cairo_set_line_cap(cap)
func (cb *CairoBindings) setLineCap(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	capStyle, err := c.IntArg(0)
	if err != nil {
		return nil, fmt.Errorf("cairo_set_line_cap: %w", err)
	}

	// Validate line cap value (0-2)
	if capStyle < 0 || capStyle > 2 {
		return nil, fmt.Errorf("cairo_set_line_cap: invalid line cap value %d (must be 0-2)", capStyle)
	}

	cb.renderer.SetLineCap(render.LineCap(capStyle))
	return c.Next(), nil
}

// setLineJoin handles cairo_set_line_join(join)
func (cb *CairoBindings) setLineJoin(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	join, err := c.IntArg(0)
	if err != nil {
		return nil, fmt.Errorf("cairo_set_line_join: %w", err)
	}

	// Validate line join value (0-2)
	if join < 0 || join > 2 {
		return nil, fmt.Errorf("cairo_set_line_join: invalid line join value %d (must be 0-2)", join)
	}

	cb.renderer.SetLineJoin(render.LineJoin(join))
	return c.Next(), nil
}

// setAntialias handles cairo_set_antialias(mode)
func (cb *CairoBindings) setAntialias(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	mode, err := c.IntArg(0)
	if err != nil {
		return nil, fmt.Errorf("cairo_set_antialias: %w", err)
	}

	// 0 = none, anything else = enabled
	cb.renderer.SetAntialias(mode != 0)
	return c.Next(), nil
}

// --- Path Building Functions ---

// newPath handles cairo_new_path()
func (cb *CairoBindings) newPath(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	cb.renderer.NewPath()
	return c.Next(), nil
}

// moveTo handles cairo_move_to(x, y)
func (cb *CairoBindings) moveTo(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	x, err := c.FloatArg(0)
	if err != nil {
		return nil, fmt.Errorf("cairo_move_to: x: %w", err)
	}
	y, err := c.FloatArg(1)
	if err != nil {
		return nil, fmt.Errorf("cairo_move_to: y: %w", err)
	}

	cb.renderer.MoveTo(x, y)
	return c.Next(), nil
}

// lineTo handles cairo_line_to(x, y)
func (cb *CairoBindings) lineTo(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	x, err := c.FloatArg(0)
	if err != nil {
		return nil, fmt.Errorf("cairo_line_to: x: %w", err)
	}
	y, err := c.FloatArg(1)
	if err != nil {
		return nil, fmt.Errorf("cairo_line_to: y: %w", err)
	}

	cb.renderer.LineTo(x, y)
	return c.Next(), nil
}

// closePath handles cairo_close_path()
func (cb *CairoBindings) closePath(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	cb.renderer.ClosePath()
	return c.Next(), nil
}

// arc handles cairo_arc(xc, yc, radius, angle1, angle2)
func (cb *CairoBindings) arc(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	xc, err := c.FloatArg(0)
	if err != nil {
		return nil, fmt.Errorf("cairo_arc: xc: %w", err)
	}
	yc, err := c.FloatArg(1)
	if err != nil {
		return nil, fmt.Errorf("cairo_arc: yc: %w", err)
	}
	radius, err := c.FloatArg(2)
	if err != nil {
		return nil, fmt.Errorf("cairo_arc: radius: %w", err)
	}
	angle1, err := c.FloatArg(3)
	if err != nil {
		return nil, fmt.Errorf("cairo_arc: angle1: %w", err)
	}
	angle2, err := c.FloatArg(4)
	if err != nil {
		return nil, fmt.Errorf("cairo_arc: angle2: %w", err)
	}

	cb.renderer.Arc(xc, yc, radius, angle1, angle2)
	return c.Next(), nil
}

// arcNegative handles cairo_arc_negative(xc, yc, radius, angle1, angle2)
func (cb *CairoBindings) arcNegative(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	xc, err := c.FloatArg(0)
	if err != nil {
		return nil, fmt.Errorf("cairo_arc_negative: xc: %w", err)
	}
	yc, err := c.FloatArg(1)
	if err != nil {
		return nil, fmt.Errorf("cairo_arc_negative: yc: %w", err)
	}
	radius, err := c.FloatArg(2)
	if err != nil {
		return nil, fmt.Errorf("cairo_arc_negative: radius: %w", err)
	}
	angle1, err := c.FloatArg(3)
	if err != nil {
		return nil, fmt.Errorf("cairo_arc_negative: angle1: %w", err)
	}
	angle2, err := c.FloatArg(4)
	if err != nil {
		return nil, fmt.Errorf("cairo_arc_negative: angle2: %w", err)
	}

	cb.renderer.ArcNegative(xc, yc, radius, angle1, angle2)
	return c.Next(), nil
}

// curveTo handles cairo_curve_to(x1, y1, x2, y2, x3, y3)
func (cb *CairoBindings) curveTo(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	x1, err := c.FloatArg(0)
	if err != nil {
		return nil, fmt.Errorf("cairo_curve_to: x1: %w", err)
	}
	y1, err := c.FloatArg(1)
	if err != nil {
		return nil, fmt.Errorf("cairo_curve_to: y1: %w", err)
	}
	x2, err := c.FloatArg(2)
	if err != nil {
		return nil, fmt.Errorf("cairo_curve_to: x2: %w", err)
	}
	y2, err := c.FloatArg(3)
	if err != nil {
		return nil, fmt.Errorf("cairo_curve_to: y2: %w", err)
	}
	x3, err := c.FloatArg(4)
	if err != nil {
		return nil, fmt.Errorf("cairo_curve_to: x3: %w", err)
	}
	y3, err := c.FloatArg(5)
	if err != nil {
		return nil, fmt.Errorf("cairo_curve_to: y3: %w", err)
	}

	cb.renderer.CurveTo(x1, y1, x2, y2, x3, y3)
	return c.Next(), nil
}

// rectangle handles cairo_rectangle(x, y, width, height)
func (cb *CairoBindings) rectangle(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	x, err := c.FloatArg(0)
	if err != nil {
		return nil, fmt.Errorf("cairo_rectangle: x: %w", err)
	}
	y, err := c.FloatArg(1)
	if err != nil {
		return nil, fmt.Errorf("cairo_rectangle: y: %w", err)
	}
	width, err := c.FloatArg(2)
	if err != nil {
		return nil, fmt.Errorf("cairo_rectangle: width: %w", err)
	}
	height, err := c.FloatArg(3)
	if err != nil {
		return nil, fmt.Errorf("cairo_rectangle: height: %w", err)
	}

	cb.renderer.Rectangle(x, y, width, height)
	return c.Next(), nil
}

// --- Drawing Functions ---

// stroke handles cairo_stroke()
func (cb *CairoBindings) stroke(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	cb.renderer.Stroke()
	return c.Next(), nil
}

// fill handles cairo_fill()
func (cb *CairoBindings) fill(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	cb.renderer.Fill()
	return c.Next(), nil
}

// strokePreserve handles cairo_stroke_preserve()
func (cb *CairoBindings) strokePreserve(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	cb.renderer.StrokePreserve()
	return c.Next(), nil
}

// fillPreserve handles cairo_fill_preserve()
func (cb *CairoBindings) fillPreserve(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	cb.renderer.FillPreserve()
	return c.Next(), nil
}

// paint handles cairo_paint()
func (cb *CairoBindings) paint(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	cb.renderer.Paint()
	return c.Next(), nil
}

// paintWithAlpha handles cairo_paint_with_alpha(alpha)
func (cb *CairoBindings) paintWithAlpha(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	alpha, err := c.FloatArg(0)
	if err != nil {
		return nil, fmt.Errorf("cairo_paint_with_alpha: %w", err)
	}

	cb.renderer.PaintWithAlpha(alpha)
	return c.Next(), nil
}

// --- Text Functions ---

// selectFontFace handles cairo_select_font_face(family, slant, weight)
func (cb *CairoBindings) selectFontFace(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	family, err := c.StringArg(0)
	if err != nil {
		return nil, fmt.Errorf("cairo_select_font_face: family: %w", err)
	}
	slant, err := c.IntArg(1)
	if err != nil {
		return nil, fmt.Errorf("cairo_select_font_face: slant: %w", err)
	}
	weight, err := c.IntArg(2)
	if err != nil {
		return nil, fmt.Errorf("cairo_select_font_face: weight: %w", err)
	}

	// Validate slant value (0-2)
	if slant < 0 || slant > 2 {
		return nil, fmt.Errorf("cairo_select_font_face: invalid slant value %d (must be 0-2)", slant)
	}

	// Validate weight value (0-1)
	if weight < 0 || weight > 1 {
		return nil, fmt.Errorf("cairo_select_font_face: invalid weight value %d (must be 0-1)", weight)
	}

	cb.renderer.SelectFontFace(family, render.FontSlant(slant), render.FontWeight(weight))
	return c.Next(), nil
}

// setFontSize handles cairo_set_font_size(size)
func (cb *CairoBindings) setFontSize(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	size, err := c.FloatArg(0)
	if err != nil {
		return nil, fmt.Errorf("cairo_set_font_size: %w", err)
	}

	cb.renderer.SetFontSize(size)
	return c.Next(), nil
}

// showText handles cairo_show_text(text)
func (cb *CairoBindings) showText(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	text, err := c.StringArg(0)
	if err != nil {
		return nil, fmt.Errorf("cairo_show_text: %w", err)
	}

	cb.renderer.ShowText(text)
	return c.Next(), nil
}

// textExtents handles cairo_text_extents(text) and returns a table with extents
func (cb *CairoBindings) textExtents(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	text, err := c.StringArg(0)
	if err != nil {
		return nil, fmt.Errorf("cairo_text_extents: %w", err)
	}

	extents := cb.renderer.TextExtentsResult(text)

	// Create a Lua table with the text extents
	extentsTable := rt.NewTable()
	extentsTable.Set(rt.StringValue("x_bearing"), rt.FloatValue(extents.XBearing))
	extentsTable.Set(rt.StringValue("y_bearing"), rt.FloatValue(extents.YBearing))
	extentsTable.Set(rt.StringValue("width"), rt.FloatValue(extents.Width))
	extentsTable.Set(rt.StringValue("height"), rt.FloatValue(extents.Height))
	extentsTable.Set(rt.StringValue("x_advance"), rt.FloatValue(extents.XAdvance))
	extentsTable.Set(rt.StringValue("y_advance"), rt.FloatValue(extents.YAdvance))

	return c.PushingNext1(t.Runtime, rt.TableValue(extentsTable)), nil
}

// --- Transformation Functions ---

// translate handles cairo_translate(tx, ty)
func (cb *CairoBindings) translate(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	tx, err := c.FloatArg(0)
	if err != nil {
		return nil, fmt.Errorf("cairo_translate: tx: %w", err)
	}
	ty, err := c.FloatArg(1)
	if err != nil {
		return nil, fmt.Errorf("cairo_translate: ty: %w", err)
	}

	cb.renderer.Translate(tx, ty)
	return c.Next(), nil
}

// rotate handles cairo_rotate(angle)
func (cb *CairoBindings) rotate(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	angle, err := c.FloatArg(0)
	if err != nil {
		return nil, fmt.Errorf("cairo_rotate: %w", err)
	}

	cb.renderer.Rotate(angle)
	return c.Next(), nil
}

// scale handles cairo_scale(sx, sy)
func (cb *CairoBindings) scale(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	sx, err := c.FloatArg(0)
	if err != nil {
		return nil, fmt.Errorf("cairo_scale: sx: %w", err)
	}
	sy, err := c.FloatArg(1)
	if err != nil {
		return nil, fmt.Errorf("cairo_scale: sy: %w", err)
	}

	cb.renderer.Scale(sx, sy)
	return c.Next(), nil
}

// save handles cairo_save()
func (cb *CairoBindings) save(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	cb.renderer.Save()
	return c.Next(), nil
}

// restore handles cairo_restore()
func (cb *CairoBindings) restore(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	cb.renderer.Restore()
	return c.Next(), nil
}

// identityMatrix handles cairo_identity_matrix()
func (cb *CairoBindings) identityMatrix(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	cb.renderer.IdentityMatrix()
	return c.Next(), nil
}
