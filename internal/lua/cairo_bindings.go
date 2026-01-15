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
	// Color functions - all accept optional cr as first argument
	cb.runtime.SetGoFunction("cairo_set_source_rgb", cb.setSourceRGB, 3, true)
	cb.runtime.SetGoFunction("cairo_set_source_rgba", cb.setSourceRGBA, 4, true)

	// Line style functions
	cb.runtime.SetGoFunction("cairo_set_line_width", cb.setLineWidth, 1, true)
	cb.runtime.SetGoFunction("cairo_set_line_cap", cb.setLineCap, 1, true)
	cb.runtime.SetGoFunction("cairo_set_line_join", cb.setLineJoin, 1, true)
	cb.runtime.SetGoFunction("cairo_set_antialias", cb.setAntialias, 1, true)

	// Path building functions
	cb.runtime.SetGoFunction("cairo_new_path", cb.newPath, 0, true)
	cb.runtime.SetGoFunction("cairo_move_to", cb.moveTo, 2, true)
	cb.runtime.SetGoFunction("cairo_line_to", cb.lineTo, 2, true)
	cb.runtime.SetGoFunction("cairo_close_path", cb.closePath, 0, true)
	cb.runtime.SetGoFunction("cairo_arc", cb.arc, 5, true)
	cb.runtime.SetGoFunction("cairo_arc_negative", cb.arcNegative, 5, true)
	cb.runtime.SetGoFunction("cairo_curve_to", cb.curveTo, 6, true)
	cb.runtime.SetGoFunction("cairo_rectangle", cb.rectangle, 4, true)

	// Relative path building functions
	cb.runtime.SetGoFunction("cairo_rel_move_to", cb.relMoveTo, 2, true)
	cb.runtime.SetGoFunction("cairo_rel_line_to", cb.relLineTo, 2, true)
	cb.runtime.SetGoFunction("cairo_rel_curve_to", cb.relCurveTo, 6, true)

	// Drawing functions
	cb.runtime.SetGoFunction("cairo_stroke", cb.stroke, 0, true)
	cb.runtime.SetGoFunction("cairo_fill", cb.fill, 0, true)
	cb.runtime.SetGoFunction("cairo_stroke_preserve", cb.strokePreserve, 0, true)
	cb.runtime.SetGoFunction("cairo_fill_preserve", cb.fillPreserve, 0, true)
	cb.runtime.SetGoFunction("cairo_paint", cb.paint, 0, true)
	cb.runtime.SetGoFunction("cairo_paint_with_alpha", cb.paintWithAlpha, 1, true)

	// Text functions
	cb.runtime.SetGoFunction("cairo_select_font_face", cb.selectFontFace, 3, true)
	cb.runtime.SetGoFunction("cairo_set_font_size", cb.setFontSize, 1, true)
	cb.runtime.SetGoFunction("cairo_show_text", cb.showText, 1, true)
	cb.runtime.SetGoFunction("cairo_text_extents", cb.textExtents, 1, true)

	// Transformation functions
	cb.runtime.SetGoFunction("cairo_translate", cb.translate, 2, true)
	cb.runtime.SetGoFunction("cairo_rotate", cb.rotate, 1, true)
	cb.runtime.SetGoFunction("cairo_scale", cb.scale, 2, true)
	cb.runtime.SetGoFunction("cairo_save", cb.save, 0, true)
	cb.runtime.SetGoFunction("cairo_restore", cb.restore, 0, true)
	cb.runtime.SetGoFunction("cairo_identity_matrix", cb.identityMatrix, 0, true)

	// Clipping functions
	cb.runtime.SetGoFunction("cairo_clip", cb.clip, 0, true)
	cb.runtime.SetGoFunction("cairo_clip_preserve", cb.clipPreserve, 0, true)
	cb.runtime.SetGoFunction("cairo_reset_clip", cb.resetClip, 0, true)
	cb.runtime.SetGoFunction("cairo_clip_extents", cb.clipExtents, 0, true)
	cb.runtime.SetGoFunction("cairo_in_clip", cb.inClip, 2, true)

	// Path query functions
	cb.runtime.SetGoFunction("cairo_get_current_point", cb.getCurrentPoint, 0, true)
	cb.runtime.SetGoFunction("cairo_has_current_point", cb.hasCurrentPoint, 0, true)
	cb.runtime.SetGoFunction("cairo_path_extents", cb.pathExtents, 0, true)

	// Register Cairo constants
	cb.registerConstants()

	// Register surface management functions
	cb.registerSurfaceFunctions()
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

	// Surface format constants (for cairo_image_surface_create)
	cb.runtime.SetGlobal("CAIRO_FORMAT_ARGB32", rt.IntValue(0))
	cb.runtime.SetGlobal("CAIRO_FORMAT_RGB24", rt.IntValue(1))
	cb.runtime.SetGlobal("CAIRO_FORMAT_A8", rt.IntValue(2))
	cb.runtime.SetGlobal("CAIRO_FORMAT_A1", rt.IntValue(3))
	cb.runtime.SetGlobal("CAIRO_FORMAT_RGB16_565", rt.IntValue(4))
}

// registerSurfaceFunctions registers surface management functions.
func (cb *CairoBindings) registerSurfaceFunctions() {
	cb.runtime.SetGoFunction("cairo_xlib_surface_create", cb.xlibSurfaceCreate, 5, false)
	cb.runtime.SetGoFunction("cairo_image_surface_create", cb.imageSurfaceCreate, 3, false)
	cb.runtime.SetGoFunction("cairo_create", cb.cairoCreate, 1, false)
	cb.runtime.SetGoFunction("cairo_destroy", cb.cairoDestroy, 1, false)
	cb.runtime.SetGoFunction("cairo_surface_destroy", cb.surfaceDestroy, 1, false)
}

// --- Surface Management Functions (CairoBindings) ---

// xlibSurfaceCreate handles cairo_xlib_surface_create(display, drawable, visual, width, height)
func (cb *CairoBindings) xlibSurfaceCreate(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	// display, drawable, visual are for compatibility (not used in Ebiten)
	_, err := c.IntArg(0)
	if err != nil {
		return nil, fmt.Errorf("cairo_xlib_surface_create: display: %w", err)
	}
	_, err = c.IntArg(1)
	if err != nil {
		return nil, fmt.Errorf("cairo_xlib_surface_create: drawable: %w", err)
	}
	_, err = c.IntArg(2)
	if err != nil {
		return nil, fmt.Errorf("cairo_xlib_surface_create: visual: %w", err)
	}
	width, err := c.IntArg(3)
	if err != nil {
		return nil, fmt.Errorf("cairo_xlib_surface_create: width: %w", err)
	}
	height, err := c.IntArg(4)
	if err != nil {
		return nil, fmt.Errorf("cairo_xlib_surface_create: height: %w", err)
	}

	surface := render.NewCairoXlibSurface(0, 0, 0, int(width), int(height))
	ud := rt.NewUserData(surface, nil)
	return c.PushingNext1(t.Runtime, rt.UserDataValue(ud)), nil
}

// imageSurfaceCreate handles cairo_image_surface_create(format, width, height)
func (cb *CairoBindings) imageSurfaceCreate(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	_, err := c.IntArg(0) // format - we always use ARGB32
	if err != nil {
		return nil, fmt.Errorf("cairo_image_surface_create: format: %w", err)
	}
	width, err := c.IntArg(1)
	if err != nil {
		return nil, fmt.Errorf("cairo_image_surface_create: width: %w", err)
	}
	height, err := c.IntArg(2)
	if err != nil {
		return nil, fmt.Errorf("cairo_image_surface_create: height: %w", err)
	}

	surface := render.NewCairoSurface(int(width), int(height))
	ud := rt.NewUserData(surface, nil)
	return c.PushingNext1(t.Runtime, rt.UserDataValue(ud)), nil
}

// cairoCreate handles cairo_create(surface)
func (cb *CairoBindings) cairoCreate(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	surfaceVal, err := c.UserDataArg(0)
	if err != nil {
		// No surface argument - return context using the shared renderer
		// This is common in Conky scripts that use the global context
		ctx := &sharedContext{renderer: cb.renderer}
		ud := rt.NewUserData(ctx, nil)
		return c.PushingNext1(t.Runtime, rt.UserDataValue(ud)), nil
	}

	surface, ok := surfaceVal.Value().(*render.CairoSurface)
	if !ok {
		return nil, fmt.Errorf("cairo_create: expected surface userdata")
	}

	ctx := render.NewCairoContext(surface)
	if ctx == nil {
		return nil, fmt.Errorf("cairo_create: failed to create context")
	}

	ud := rt.NewUserData(ctx, nil)
	return c.PushingNext1(t.Runtime, rt.UserDataValue(ud)), nil
}

// cairoDestroy handles cairo_destroy(cr)
func (cb *CairoBindings) cairoDestroy(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if c.NArgs() == 0 {
		return c.Next(), nil
	}

	ctxVal, err := c.UserDataArg(0)
	if err != nil {
		return c.Next(), nil // Ignore for compatibility
	}

	switch ctx := ctxVal.Value().(type) {
	case *render.CairoContext:
		ctx.Destroy()
	case *sharedContext:
		// Don't destroy the shared context
	}
	return c.Next(), nil
}

// surfaceDestroy handles cairo_surface_destroy(surface)
func (cb *CairoBindings) surfaceDestroy(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if c.NArgs() == 0 {
		return c.Next(), nil
	}

	surfaceVal, err := c.UserDataArg(0)
	if err != nil {
		return c.Next(), nil // Ignore for compatibility
	}

	surface, ok := surfaceVal.Value().(*render.CairoSurface)
	if !ok {
		return c.Next(), nil
	}

	surface.Destroy()
	return c.Next(), nil
}

// sharedContext is a wrapper for the global shared renderer.
// This is used when cairo_create is called without a surface argument.
type sharedContext struct {
	renderer *render.CairoRenderer
}

// getRendererFromContext extracts the CairoRenderer from a context userdata.
// The context can be a *render.CairoContext or a *sharedContext.
// Returns the renderer and the argument offset (1 if context was provided, 0 otherwise).
// It combines c.Args() and c.Etc() to get all arguments including varargs.
func (cb *CairoBindings) getRendererFromContext(c *rt.GoCont) (*render.CairoRenderer, int) {
	// Combine regular args and varargs to get total argument count
	allArgs := append(c.Args(), c.Etc()...)

	if len(allArgs) == 0 {
		return cb.renderer, 0
	}

	// Try to get the first argument as userdata (context)
	firstArg := allArgs[0]
	if ud, ok := firstArg.TryUserData(); ok {
		// Check what type of context we have
		switch ctx := ud.Value().(type) {
		case *render.CairoContext:
			r := ctx.Renderer()
			if r != nil {
				return r, 1
			}
			return cb.renderer, 1
		case *sharedContext:
			if ctx.renderer != nil {
				return ctx.renderer, 1
			}
			return cb.renderer, 1
		default:
			// Unknown userdata type (e.g., surface) - don't treat as context
			return cb.renderer, 0
		}
	}

	// First argument is not userdata, use global renderer
	return cb.renderer, 0
}

// getAllArgs combines Args() and Etc() to get all arguments including varargs
func getAllArgs(c *rt.GoCont) []rt.Value {
	return append(c.Args(), c.Etc()...)
}

// getFloatArg gets a float argument from the combined args slice
func getFloatArg(args []rt.Value, idx int) (float64, error) {
	if idx >= len(args) {
		return 0, fmt.Errorf("argument %d out of range (have %d)", idx, len(args))
	}
	if f, ok := args[idx].TryFloat(); ok {
		return f, nil
	}
	if i, ok := args[idx].TryInt(); ok {
		return float64(i), nil
	}
	return 0, fmt.Errorf("argument %d is not a number", idx)
}

// getIntArg gets an int argument from the combined args slice
func getIntArg(args []rt.Value, idx int) (int64, error) {
	if idx >= len(args) {
		return 0, fmt.Errorf("argument %d out of range (have %d)", idx, len(args))
	}
	if i, ok := args[idx].TryInt(); ok {
		return i, nil
	}
	if f, ok := args[idx].TryFloat(); ok {
		return int64(f), nil
	}
	return 0, fmt.Errorf("argument %d is not an integer", idx)
}

// getStringArg gets a string argument from the combined args slice
func getStringArg(args []rt.Value, idx int) (string, error) {
	if idx >= len(args) {
		return "", fmt.Errorf("argument %d out of range (have %d)", idx, len(args))
	}
	if s, ok := args[idx].TryString(); ok {
		return s, nil
	}
	return "", fmt.Errorf("argument %d is not a string", idx)
}

// --- Color Functions ---

// setSourceRGB handles cairo_set_source_rgb(cr, r, g, b)
// The cr argument is optional for backward compatibility.
func (cb *CairoBindings) setSourceRGB(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, offset := cb.getRendererFromContext(c)
	args := getAllArgs(c)

	r, err := getFloatArg(args, 0+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_set_source_rgb: r: %w", err)
	}
	g, err := getFloatArg(args, 1+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_set_source_rgb: g: %w", err)
	}
	b, err := getFloatArg(args, 2+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_set_source_rgb: b: %w", err)
	}

	renderer.SetSourceRGB(r, g, b)
	return c.Next(), nil
}

// setSourceRGBA handles cairo_set_source_rgba(cr, r, g, b, a)
// The cr argument is optional for backward compatibility.
func (cb *CairoBindings) setSourceRGBA(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, offset := cb.getRendererFromContext(c)
	args := getAllArgs(c)

	r, err := getFloatArg(args, 0+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_set_source_rgba: r: %w", err)
	}
	g, err := getFloatArg(args, 1+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_set_source_rgba: g: %w", err)
	}
	b, err := getFloatArg(args, 2+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_set_source_rgba: b: %w", err)
	}
	a, err := getFloatArg(args, 3+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_set_source_rgba: a: %w", err)
	}

	renderer.SetSourceRGBA(r, g, b, a)
	return c.Next(), nil
}

// --- Line Style Functions ---

// setLineWidth handles cairo_set_line_width(cr, width)
// The cr argument is optional for backward compatibility.
func (cb *CairoBindings) setLineWidth(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, offset := cb.getRendererFromContext(c)
	args := getAllArgs(c)

	width, err := getFloatArg(args, 0+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_set_line_width: %w", err)
	}

	renderer.SetLineWidth(width)
	return c.Next(), nil
}

// setLineCap handles cairo_set_line_cap(cr, cap)
// The cr argument is optional for backward compatibility.
func (cb *CairoBindings) setLineCap(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, offset := cb.getRendererFromContext(c)
	args := getAllArgs(c)

	capStyle, err := getIntArg(args, 0+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_set_line_cap: %w", err)
	}

	// Validate line cap value (0-2)
	if capStyle < 0 || capStyle > 2 {
		return nil, fmt.Errorf("cairo_set_line_cap: invalid line cap value %d (must be 0-2)", capStyle)
	}

	renderer.SetLineCap(render.LineCap(capStyle))
	return c.Next(), nil
}

// setLineJoin handles cairo_set_line_join(cr, join)
// The cr argument is optional for backward compatibility.
func (cb *CairoBindings) setLineJoin(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, offset := cb.getRendererFromContext(c)
	args := getAllArgs(c)

	join, err := getIntArg(args, 0+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_set_line_join: %w", err)
	}

	// Validate line join value (0-2)
	if join < 0 || join > 2 {
		return nil, fmt.Errorf("cairo_set_line_join: invalid line join value %d (must be 0-2)", join)
	}

	renderer.SetLineJoin(render.LineJoin(join))
	return c.Next(), nil
}

// setAntialias handles cairo_set_antialias(cr, mode)
// The cr argument is optional for backward compatibility.
func (cb *CairoBindings) setAntialias(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, offset := cb.getRendererFromContext(c)
	args := getAllArgs(c)

	mode, err := getIntArg(args, 0+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_set_antialias: %w", err)
	}

	// 0 = none, anything else = enabled
	renderer.SetAntialias(mode != 0)
	return c.Next(), nil
}

// --- Path Building Functions ---

// newPath handles cairo_new_path(cr)
// The cr argument is optional for backward compatibility.
func (cb *CairoBindings) newPath(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, _ := cb.getRendererFromContext(c)
	renderer.NewPath()
	return c.Next(), nil
}

// moveTo handles cairo_move_to(cr, x, y)
// The cr argument is optional for backward compatibility.
func (cb *CairoBindings) moveTo(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, offset := cb.getRendererFromContext(c)
	args := getAllArgs(c)

	x, err := getFloatArg(args, 0+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_move_to: x: %w", err)
	}
	y, err := getFloatArg(args, 1+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_move_to: y: %w", err)
	}

	renderer.MoveTo(x, y)
	return c.Next(), nil
}

// lineTo handles cairo_line_to(cr, x, y)
// The cr argument is optional for backward compatibility.
func (cb *CairoBindings) lineTo(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, offset := cb.getRendererFromContext(c)
	args := getAllArgs(c)

	x, err := getFloatArg(args, 0+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_line_to: x: %w", err)
	}
	y, err := getFloatArg(args, 1+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_line_to: y: %w", err)
	}

	renderer.LineTo(x, y)
	return c.Next(), nil
}

// closePath handles cairo_close_path(cr)
// The cr argument is optional for backward compatibility.
func (cb *CairoBindings) closePath(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, _ := cb.getRendererFromContext(c)
	renderer.ClosePath()
	return c.Next(), nil
}

// arc handles cairo_arc(cr, xc, yc, radius, angle1, angle2)
// The cr argument is optional for backward compatibility.
func (cb *CairoBindings) arc(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, offset := cb.getRendererFromContext(c)
	args := getAllArgs(c)

	xc, err := getFloatArg(args, 0+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_arc: xc: %w", err)
	}
	yc, err := getFloatArg(args, 1+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_arc: yc: %w", err)
	}
	radius, err := getFloatArg(args, 2+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_arc: radius: %w", err)
	}
	angle1, err := getFloatArg(args, 3+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_arc: angle1: %w", err)
	}
	angle2, err := getFloatArg(args, 4+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_arc: angle2: %w", err)
	}

	renderer.Arc(xc, yc, radius, angle1, angle2)
	return c.Next(), nil
}

// arcNegative handles cairo_arc_negative(cr, xc, yc, radius, angle1, angle2)
// The cr argument is optional for backward compatibility.
func (cb *CairoBindings) arcNegative(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, offset := cb.getRendererFromContext(c)
	args := getAllArgs(c)

	xc, err := getFloatArg(args, 0+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_arc_negative: xc: %w", err)
	}
	yc, err := getFloatArg(args, 1+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_arc_negative: yc: %w", err)
	}
	radius, err := getFloatArg(args, 2+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_arc_negative: radius: %w", err)
	}
	angle1, err := getFloatArg(args, 3+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_arc_negative: angle1: %w", err)
	}
	angle2, err := getFloatArg(args, 4+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_arc_negative: angle2: %w", err)
	}

	renderer.ArcNegative(xc, yc, radius, angle1, angle2)
	return c.Next(), nil
}

// curveTo handles cairo_curve_to(cr, x1, y1, x2, y2, x3, y3)
// The cr argument is optional for backward compatibility.
func (cb *CairoBindings) curveTo(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, offset := cb.getRendererFromContext(c)
	args := getAllArgs(c)

	x1, err := getFloatArg(args, 0+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_curve_to: x1: %w", err)
	}
	y1, err := getFloatArg(args, 1+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_curve_to: y1: %w", err)
	}
	x2, err := getFloatArg(args, 2+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_curve_to: x2: %w", err)
	}
	y2, err := getFloatArg(args, 3+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_curve_to: y2: %w", err)
	}
	x3, err := getFloatArg(args, 4+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_curve_to: x3: %w", err)
	}
	y3, err := getFloatArg(args, 5+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_curve_to: y3: %w", err)
	}

	renderer.CurveTo(x1, y1, x2, y2, x3, y3)
	return c.Next(), nil
}

// rectangle handles cairo_rectangle(cr, x, y, width, height)
// The cr argument is optional for backward compatibility.
func (cb *CairoBindings) rectangle(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, offset := cb.getRendererFromContext(c)
	args := getAllArgs(c)

	x, err := getFloatArg(args, 0+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_rectangle: x: %w", err)
	}
	y, err := getFloatArg(args, 1+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_rectangle: y: %w", err)
	}
	width, err := getFloatArg(args, 2+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_rectangle: width: %w", err)
	}
	height, err := getFloatArg(args, 3+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_rectangle: height: %w", err)
	}

	renderer.Rectangle(x, y, width, height)
	return c.Next(), nil
}

// --- Drawing Functions ---

// stroke handles cairo_stroke(cr)
// The cr argument is optional for backward compatibility.
func (cb *CairoBindings) stroke(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, _ := cb.getRendererFromContext(c)
	renderer.Stroke()
	return c.Next(), nil
}

// fill handles cairo_fill(cr)
// The cr argument is optional for backward compatibility.
func (cb *CairoBindings) fill(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, _ := cb.getRendererFromContext(c)
	renderer.Fill()
	return c.Next(), nil
}

// strokePreserve handles cairo_stroke_preserve(cr)
// The cr argument is optional for backward compatibility.
func (cb *CairoBindings) strokePreserve(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, _ := cb.getRendererFromContext(c)
	renderer.StrokePreserve()
	return c.Next(), nil
}

// fillPreserve handles cairo_fill_preserve(cr)
// The cr argument is optional for backward compatibility.
func (cb *CairoBindings) fillPreserve(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, _ := cb.getRendererFromContext(c)
	renderer.FillPreserve()
	return c.Next(), nil
}

// paint handles cairo_paint(cr)
// The cr argument is optional for backward compatibility.
func (cb *CairoBindings) paint(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, _ := cb.getRendererFromContext(c)
	renderer.Paint()
	return c.Next(), nil
}

// paintWithAlpha handles cairo_paint_with_alpha(cr, alpha)
// The cr argument is optional for backward compatibility.
func (cb *CairoBindings) paintWithAlpha(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, offset := cb.getRendererFromContext(c)
	args := getAllArgs(c)

	alpha, err := getFloatArg(args, 0+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_paint_with_alpha: %w", err)
	}

	renderer.PaintWithAlpha(alpha)
	return c.Next(), nil
}

// --- Text Functions ---

// selectFontFace handles cairo_select_font_face(cr, family, slant, weight)
// The cr argument is optional for backward compatibility.
func (cb *CairoBindings) selectFontFace(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, offset := cb.getRendererFromContext(c)
	args := getAllArgs(c)

	family, err := getStringArg(args, 0+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_select_font_face: family: %w", err)
	}
	slant, err := getIntArg(args, 1+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_select_font_face: slant: %w", err)
	}
	weight, err := getIntArg(args, 2+offset)
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

	renderer.SelectFontFace(family, render.FontSlant(slant), render.FontWeight(weight))
	return c.Next(), nil
}

// setFontSize handles cairo_set_font_size(cr, size)
// The cr argument is optional for backward compatibility.
func (cb *CairoBindings) setFontSize(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, offset := cb.getRendererFromContext(c)
	args := getAllArgs(c)

	size, err := getFloatArg(args, 0+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_set_font_size: %w", err)
	}

	renderer.SetFontSize(size)
	return c.Next(), nil
}

// showText handles cairo_show_text(cr, text)
// The cr argument is optional for backward compatibility.
func (cb *CairoBindings) showText(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, offset := cb.getRendererFromContext(c)
	args := getAllArgs(c)

	text, err := getStringArg(args, 0+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_show_text: %w", err)
	}

	renderer.ShowText(text)
	return c.Next(), nil
}

// textExtents handles cairo_text_extents(cr, text) and returns a table with extents
// The cr argument is optional for backward compatibility.
func (cb *CairoBindings) textExtents(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, offset := cb.getRendererFromContext(c)
	args := getAllArgs(c)

	text, err := getStringArg(args, 0+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_text_extents: %w", err)
	}

	extents := renderer.TextExtentsResult(text)

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

// translate handles cairo_translate(cr, tx, ty)
// The cr argument is optional for backward compatibility.
func (cb *CairoBindings) translate(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, offset := cb.getRendererFromContext(c)
	args := getAllArgs(c)

	tx, err := getFloatArg(args, 0+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_translate: tx: %w", err)
	}
	ty, err := getFloatArg(args, 1+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_translate: ty: %w", err)
	}

	renderer.Translate(tx, ty)
	return c.Next(), nil
}

// rotate handles cairo_rotate(cr, angle)
// The cr argument is optional for backward compatibility.
func (cb *CairoBindings) rotate(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, offset := cb.getRendererFromContext(c)
	args := getAllArgs(c)

	angle, err := getFloatArg(args, 0+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_rotate: %w", err)
	}

	renderer.Rotate(angle)
	return c.Next(), nil
}

// scale handles cairo_scale(cr, sx, sy)
// The cr argument is optional for backward compatibility.
func (cb *CairoBindings) scale(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, offset := cb.getRendererFromContext(c)
	args := getAllArgs(c)

	sx, err := getFloatArg(args, 0+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_scale: sx: %w", err)
	}
	sy, err := getFloatArg(args, 1+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_scale: sy: %w", err)
	}

	renderer.Scale(sx, sy)
	return c.Next(), nil
}

// save handles cairo_save(cr)
// The cr argument is optional for backward compatibility.
func (cb *CairoBindings) save(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, _ := cb.getRendererFromContext(c)
	renderer.Save()
	return c.Next(), nil
}

// restore handles cairo_restore(cr)
// The cr argument is optional for backward compatibility.
func (cb *CairoBindings) restore(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, _ := cb.getRendererFromContext(c)
	renderer.Restore()
	return c.Next(), nil
}

// identityMatrix handles cairo_identity_matrix(cr)
// The cr argument is optional for backward compatibility.
func (cb *CairoBindings) identityMatrix(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, _ := cb.getRendererFromContext(c)
	renderer.IdentityMatrix()
	return c.Next(), nil
}

// --- Relative Path Functions ---

// relMoveTo handles cairo_rel_move_to(cr, dx, dy)
// The cr argument is optional for backward compatibility.
func (cb *CairoBindings) relMoveTo(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, offset := cb.getRendererFromContext(c)
	args := getAllArgs(c)

	dx, err := getFloatArg(args, 0+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_rel_move_to: dx: %w", err)
	}
	dy, err := getFloatArg(args, 1+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_rel_move_to: dy: %w", err)
	}

	renderer.RelMoveTo(dx, dy)
	return c.Next(), nil
}

// relLineTo handles cairo_rel_line_to(cr, dx, dy)
// The cr argument is optional for backward compatibility.
func (cb *CairoBindings) relLineTo(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, offset := cb.getRendererFromContext(c)
	args := getAllArgs(c)

	dx, err := getFloatArg(args, 0+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_rel_line_to: dx: %w", err)
	}
	dy, err := getFloatArg(args, 1+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_rel_line_to: dy: %w", err)
	}

	renderer.RelLineTo(dx, dy)
	return c.Next(), nil
}

// relCurveTo handles cairo_rel_curve_to(cr, dx1, dy1, dx2, dy2, dx3, dy3)
// The cr argument is optional for backward compatibility.
func (cb *CairoBindings) relCurveTo(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, offset := cb.getRendererFromContext(c)
	args := getAllArgs(c)

	dx1, err := getFloatArg(args, 0+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_rel_curve_to: dx1: %w", err)
	}
	dy1, err := getFloatArg(args, 1+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_rel_curve_to: dy1: %w", err)
	}
	dx2, err := getFloatArg(args, 2+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_rel_curve_to: dx2: %w", err)
	}
	dy2, err := getFloatArg(args, 3+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_rel_curve_to: dy2: %w", err)
	}
	dx3, err := getFloatArg(args, 4+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_rel_curve_to: dx3: %w", err)
	}
	dy3, err := getFloatArg(args, 5+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_rel_curve_to: dy3: %w", err)
	}

	renderer.RelCurveTo(dx1, dy1, dx2, dy2, dx3, dy3)
	return c.Next(), nil
}

// --- Clipping Functions ---

// clip handles cairo_clip(cr)
// The cr argument is optional for backward compatibility.
func (cb *CairoBindings) clip(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, _ := cb.getRendererFromContext(c)
	renderer.Clip()
	return c.Next(), nil
}

// clipPreserve handles cairo_clip_preserve(cr)
// The cr argument is optional for backward compatibility.
func (cb *CairoBindings) clipPreserve(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, _ := cb.getRendererFromContext(c)
	renderer.ClipPreserve()
	return c.Next(), nil
}

// resetClip handles cairo_reset_clip(cr)
// The cr argument is optional for backward compatibility.
func (cb *CairoBindings) resetClip(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, _ := cb.getRendererFromContext(c)
	renderer.ResetClip()
	return c.Next(), nil
}

// clipExtents handles cairo_clip_extents(cr)
// Returns x1, y1, x2, y2 - the bounding box of the current clip region.
// The cr argument is optional for backward compatibility.
func (cb *CairoBindings) clipExtents(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, _ := cb.getRendererFromContext(c)
	x1, y1, x2, y2 := renderer.ClipExtents()
	return c.PushingNext(t.Runtime,
		rt.FloatValue(x1),
		rt.FloatValue(y1),
		rt.FloatValue(x2),
		rt.FloatValue(y2),
	), nil
}

// inClip handles cairo_in_clip(cr, x, y)
// Returns true if the given point is inside the current clip region.
// The cr argument is optional for backward compatibility.
func (cb *CairoBindings) inClip(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, offset := cb.getRendererFromContext(c)
	args := getAllArgs(c)

	x, err := getFloatArg(args, 0+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_in_clip: x: %w", err)
	}
	y, err := getFloatArg(args, 1+offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_in_clip: y: %w", err)
	}

	result := renderer.InClip(x, y)
	return c.PushingNext1(t.Runtime, rt.BoolValue(result)), nil
}

// --- Path Query Functions ---

// getCurrentPoint handles cairo_get_current_point(cr)
// Returns x, y - the current point in the path.
// The cr argument is optional for backward compatibility.
func (cb *CairoBindings) getCurrentPoint(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, _ := cb.getRendererFromContext(c)
	x, y, _ := renderer.GetCurrentPoint()
	return c.PushingNext(t.Runtime,
		rt.FloatValue(x),
		rt.FloatValue(y),
	), nil
}

// hasCurrentPoint handles cairo_has_current_point(cr)
// Returns true if there is a current point defined.
// The cr argument is optional for backward compatibility.
func (cb *CairoBindings) hasCurrentPoint(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, _ := cb.getRendererFromContext(c)
	hasPoint := renderer.HasCurrentPoint()
	return c.PushingNext1(t.Runtime, rt.BoolValue(hasPoint)), nil
}

// pathExtents handles cairo_path_extents(cr)
// Returns x1, y1, x2, y2 - the bounding box of the current path.
// The cr argument is optional for backward compatibility.
func (cb *CairoBindings) pathExtents(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, _ := cb.getRendererFromContext(c)
	x1, y1, x2, y2 := renderer.PathExtents()
	return c.PushingNext(t.Runtime,
		rt.FloatValue(x1),
		rt.FloatValue(y1),
		rt.FloatValue(x2),
		rt.FloatValue(y2),
	), nil
}
