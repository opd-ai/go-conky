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

	// Pattern/gradient functions
	cb.runtime.SetGoFunction("cairo_pattern_create_rgb", cb.patternCreateRGB, 3, false)
	cb.runtime.SetGoFunction("cairo_pattern_create_rgba", cb.patternCreateRGBA, 4, false)
	cb.runtime.SetGoFunction("cairo_pattern_create_linear", cb.patternCreateLinear, 4, false)
	cb.runtime.SetGoFunction("cairo_pattern_create_radial", cb.patternCreateRadial, 6, false)
	cb.runtime.SetGoFunction("cairo_pattern_add_color_stop_rgb", cb.patternAddColorStopRGB, 5, false)
	cb.runtime.SetGoFunction("cairo_pattern_add_color_stop_rgba", cb.patternAddColorStopRGBA, 6, false)
	cb.runtime.SetGoFunction("cairo_set_source", cb.setSource, 2, false)
	cb.runtime.SetGoFunction("cairo_pattern_set_extend", cb.patternSetExtend, 2, false)
	cb.runtime.SetGoFunction("cairo_pattern_get_extend", cb.patternGetExtend, 1, false)

	// Matrix functions
	cb.runtime.SetGoFunction("cairo_get_matrix", cb.getMatrix, 0, true)
	cb.runtime.SetGoFunction("cairo_set_matrix", cb.setMatrix, 1, true)
	cb.runtime.SetGoFunction("cairo_transform", cb.transform, 1, true)
	cb.runtime.SetGoFunction("cairo_matrix_init", cb.matrixInit, 6, false)
	cb.runtime.SetGoFunction("cairo_matrix_init_identity", cb.matrixInitIdentity, 0, false)
	cb.runtime.SetGoFunction("cairo_matrix_init_translate", cb.matrixInitTranslate, 2, false)
	cb.runtime.SetGoFunction("cairo_matrix_init_scale", cb.matrixInitScale, 2, false)
	cb.runtime.SetGoFunction("cairo_matrix_init_rotate", cb.matrixInitRotate, 1, false)
	cb.runtime.SetGoFunction("cairo_matrix_translate", cb.matrixTranslate, 3, false)
	cb.runtime.SetGoFunction("cairo_matrix_scale", cb.matrixScale, 3, false)
	cb.runtime.SetGoFunction("cairo_matrix_rotate", cb.matrixRotate, 2, false)
	cb.runtime.SetGoFunction("cairo_matrix_invert", cb.matrixInvert, 1, false)
	cb.runtime.SetGoFunction("cairo_matrix_multiply", cb.matrixMultiply, 3, false)
	cb.runtime.SetGoFunction("cairo_matrix_transform_point", cb.matrixTransformPoint, 3, false)
	cb.runtime.SetGoFunction("cairo_matrix_transform_distance", cb.matrixTransformDistance, 3, false)

	// Dash and miter functions
	cb.runtime.SetGoFunction("cairo_set_dash", cb.setDash, 2, true)
	cb.runtime.SetGoFunction("cairo_get_dash", cb.getDash, 0, true)
	cb.runtime.SetGoFunction("cairo_get_dash_count", cb.getDashCount, 0, true)
	cb.runtime.SetGoFunction("cairo_set_miter_limit", cb.setMiterLimit, 1, true)
	cb.runtime.SetGoFunction("cairo_get_miter_limit", cb.getMiterLimit, 0, true)

	// Fill rule and operator functions
	cb.runtime.SetGoFunction("cairo_set_fill_rule", cb.setFillRule, 1, true)
	cb.runtime.SetGoFunction("cairo_get_fill_rule", cb.getFillRule, 0, true)
	cb.runtime.SetGoFunction("cairo_set_operator", cb.setOperator, 1, true)
	cb.runtime.SetGoFunction("cairo_get_operator", cb.getOperator, 0, true)

	// Getter functions for line properties
	cb.runtime.SetGoFunction("cairo_get_line_width", cb.getLineWidth, 0, true)
	cb.runtime.SetGoFunction("cairo_get_line_cap", cb.getLineCap, 0, true)
	cb.runtime.SetGoFunction("cairo_get_line_join", cb.getLineJoin, 0, true)
	cb.runtime.SetGoFunction("cairo_get_antialias", cb.getAntialias, 0, true)

	// Hit testing functions
	cb.runtime.SetGoFunction("cairo_in_fill", cb.inFill, 2, true)
	cb.runtime.SetGoFunction("cairo_in_stroke", cb.inStroke, 2, true)

	// Path extent functions
	cb.runtime.SetGoFunction("cairo_stroke_extents", cb.strokeExtents, 0, true)
	cb.runtime.SetGoFunction("cairo_fill_extents", cb.fillExtents, 0, true)

	// Font functions
	cb.runtime.SetGoFunction("cairo_font_extents", cb.fontExtents, 0, true)
	cb.runtime.SetGoFunction("cairo_get_font_face", cb.getFontFace, 0, true)
	cb.runtime.SetGoFunction("cairo_get_font_size", cb.getFontSize, 0, true)

	// Sub-path function
	cb.runtime.SetGoFunction("cairo_new_sub_path", cb.newSubPath, 0, true)

	// Coordinate transformation functions
	cb.runtime.SetGoFunction("cairo_user_to_device", cb.userToDevice, 2, true)
	cb.runtime.SetGoFunction("cairo_user_to_device_distance", cb.userToDeviceDistance, 2, true)
	cb.runtime.SetGoFunction("cairo_device_to_user", cb.deviceToUser, 2, true)
	cb.runtime.SetGoFunction("cairo_device_to_user_distance", cb.deviceToUserDistance, 2, true)

	// Path copying functions
	cb.runtime.SetGoFunction("cairo_copy_path", cb.copyPath, 0, true)
	cb.runtime.SetGoFunction("cairo_append_path", cb.appendPath, 1, true)

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

	// Pattern extend constants
	cb.runtime.SetGlobal("CAIRO_EXTEND_NONE", rt.IntValue(int64(render.PatternExtendNone)))
	cb.runtime.SetGlobal("CAIRO_EXTEND_REPEAT", rt.IntValue(int64(render.PatternExtendRepeat)))
	cb.runtime.SetGlobal("CAIRO_EXTEND_REFLECT", rt.IntValue(int64(render.PatternExtendReflect)))
	cb.runtime.SetGlobal("CAIRO_EXTEND_PAD", rt.IntValue(int64(render.PatternExtendPad)))

	// Fill rule constants
	cb.runtime.SetGlobal("CAIRO_FILL_RULE_WINDING", rt.IntValue(0))
	cb.runtime.SetGlobal("CAIRO_FILL_RULE_EVEN_ODD", rt.IntValue(1))

	// Operator constants (common ones)
	cb.runtime.SetGlobal("CAIRO_OPERATOR_CLEAR", rt.IntValue(0))
	cb.runtime.SetGlobal("CAIRO_OPERATOR_SOURCE", rt.IntValue(1))
	cb.runtime.SetGlobal("CAIRO_OPERATOR_OVER", rt.IntValue(2))
	cb.runtime.SetGlobal("CAIRO_OPERATOR_IN", rt.IntValue(3))
	cb.runtime.SetGlobal("CAIRO_OPERATOR_OUT", rt.IntValue(4))
	cb.runtime.SetGlobal("CAIRO_OPERATOR_ATOP", rt.IntValue(5))
	cb.runtime.SetGlobal("CAIRO_OPERATOR_DEST", rt.IntValue(6))
	cb.runtime.SetGlobal("CAIRO_OPERATOR_DEST_OVER", rt.IntValue(7))
	cb.runtime.SetGlobal("CAIRO_OPERATOR_DEST_IN", rt.IntValue(8))
	cb.runtime.SetGlobal("CAIRO_OPERATOR_DEST_OUT", rt.IntValue(9))
	cb.runtime.SetGlobal("CAIRO_OPERATOR_DEST_ATOP", rt.IntValue(10))
	cb.runtime.SetGlobal("CAIRO_OPERATOR_XOR", rt.IntValue(11))
	cb.runtime.SetGlobal("CAIRO_OPERATOR_ADD", rt.IntValue(12))
	cb.runtime.SetGlobal("CAIRO_OPERATOR_SATURATE", rt.IntValue(13))
}

// registerSurfaceFunctions registers surface management functions.
func (cb *CairoBindings) registerSurfaceFunctions() {
	cb.runtime.SetGoFunction("cairo_xlib_surface_create", cb.xlibSurfaceCreate, 5, false)
	cb.runtime.SetGoFunction("cairo_image_surface_create", cb.imageSurfaceCreate, 3, false)
	cb.runtime.SetGoFunction("cairo_image_surface_create_from_png", cb.imageSurfaceCreateFromPNG, 1, false)
	cb.runtime.SetGoFunction("cairo_surface_write_to_png", cb.surfaceWriteToPNG, 2, false)
	cb.runtime.SetGoFunction("cairo_create", cb.cairoCreate, 1, false)
	cb.runtime.SetGoFunction("cairo_destroy", cb.cairoDestroy, 1, false)
	cb.runtime.SetGoFunction("cairo_surface_destroy", cb.surfaceDestroy, 1, false)
	cb.runtime.SetGoFunction("cairo_surface_flush", cb.surfaceFlush, 1, false)
	cb.runtime.SetGoFunction("cairo_surface_mark_dirty", cb.surfaceMarkDirty, 1, false)
	cb.runtime.SetGoFunction("cairo_surface_mark_dirty_rectangle", cb.surfaceMarkDirtyRectangle, 5, false)
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

// --- Pattern/Gradient Functions ---

// patternCreateRGB handles cairo_pattern_create_rgb(r, g, b)
// Creates a solid pattern with the given RGB color values (0.0-1.0).
func (cb *CairoBindings) patternCreateRGB(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	args := getAllArgs(c)

	r, err := getFloatArg(args, 0)
	if err != nil {
		return nil, fmt.Errorf("cairo_pattern_create_rgb: r: %w", err)
	}
	g, err := getFloatArg(args, 1)
	if err != nil {
		return nil, fmt.Errorf("cairo_pattern_create_rgb: g: %w", err)
	}
	b, err := getFloatArg(args, 2)
	if err != nil {
		return nil, fmt.Errorf("cairo_pattern_create_rgb: b: %w", err)
	}

	pattern := render.NewSolidPattern(r, g, b, 1.0)
	ud := rt.NewUserData(pattern, nil)
	return c.PushingNext1(t.Runtime, rt.UserDataValue(ud)), nil
}

// patternCreateRGBA handles cairo_pattern_create_rgba(r, g, b, a)
// Creates a solid pattern with the given RGBA color values (0.0-1.0).
func (cb *CairoBindings) patternCreateRGBA(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	args := getAllArgs(c)

	r, err := getFloatArg(args, 0)
	if err != nil {
		return nil, fmt.Errorf("cairo_pattern_create_rgba: r: %w", err)
	}
	g, err := getFloatArg(args, 1)
	if err != nil {
		return nil, fmt.Errorf("cairo_pattern_create_rgba: g: %w", err)
	}
	b, err := getFloatArg(args, 2)
	if err != nil {
		return nil, fmt.Errorf("cairo_pattern_create_rgba: b: %w", err)
	}
	a, err := getFloatArg(args, 3)
	if err != nil {
		return nil, fmt.Errorf("cairo_pattern_create_rgba: a: %w", err)
	}

	pattern := render.NewSolidPattern(r, g, b, a)
	ud := rt.NewUserData(pattern, nil)
	return c.PushingNext1(t.Runtime, rt.UserDataValue(ud)), nil
}

// patternCreateLinear handles cairo_pattern_create_linear(x0, y0, x1, y1)
// Creates a linear gradient pattern from (x0, y0) to (x1, y1).
func (cb *CairoBindings) patternCreateLinear(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	args := getAllArgs(c)

	x0, err := getFloatArg(args, 0)
	if err != nil {
		return nil, fmt.Errorf("cairo_pattern_create_linear: x0: %w", err)
	}
	y0, err := getFloatArg(args, 1)
	if err != nil {
		return nil, fmt.Errorf("cairo_pattern_create_linear: y0: %w", err)
	}
	x1, err := getFloatArg(args, 2)
	if err != nil {
		return nil, fmt.Errorf("cairo_pattern_create_linear: x1: %w", err)
	}
	y1, err := getFloatArg(args, 3)
	if err != nil {
		return nil, fmt.Errorf("cairo_pattern_create_linear: y1: %w", err)
	}

	pattern := render.NewLinearPattern(x0, y0, x1, y1)
	ud := rt.NewUserData(pattern, nil)
	return c.PushingNext1(t.Runtime, rt.UserDataValue(ud)), nil
}

// patternCreateRadial handles cairo_pattern_create_radial(cx0, cy0, r0, cx1, cy1, r1)
// Creates a radial gradient pattern between two circles.
func (cb *CairoBindings) patternCreateRadial(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	args := getAllArgs(c)

	cx0, err := getFloatArg(args, 0)
	if err != nil {
		return nil, fmt.Errorf("cairo_pattern_create_radial: cx0: %w", err)
	}
	cy0, err := getFloatArg(args, 1)
	if err != nil {
		return nil, fmt.Errorf("cairo_pattern_create_radial: cy0: %w", err)
	}
	r0, err := getFloatArg(args, 2)
	if err != nil {
		return nil, fmt.Errorf("cairo_pattern_create_radial: r0: %w", err)
	}
	cx1, err := getFloatArg(args, 3)
	if err != nil {
		return nil, fmt.Errorf("cairo_pattern_create_radial: cx1: %w", err)
	}
	cy1, err := getFloatArg(args, 4)
	if err != nil {
		return nil, fmt.Errorf("cairo_pattern_create_radial: cy1: %w", err)
	}
	r1, err := getFloatArg(args, 5)
	if err != nil {
		return nil, fmt.Errorf("cairo_pattern_create_radial: r1: %w", err)
	}

	pattern := render.NewRadialPattern(cx0, cy0, r0, cx1, cy1, r1)
	ud := rt.NewUserData(pattern, nil)
	return c.PushingNext1(t.Runtime, rt.UserDataValue(ud)), nil
}

// getPatternArg extracts a CairoPattern from a userdata argument.
func getPatternArg(args []rt.Value, idx int) (*render.CairoPattern, error) {
	if idx >= len(args) {
		return nil, fmt.Errorf("missing argument at index %d", idx)
	}
	ud, ok := args[idx].TryUserData()
	if !ok {
		return nil, fmt.Errorf("argument at index %d is not a pattern", idx)
	}
	pattern, ok := ud.Value().(*render.CairoPattern)
	if !ok {
		return nil, fmt.Errorf("argument at index %d is not a pattern", idx)
	}
	return pattern, nil
}

// patternAddColorStopRGB handles cairo_pattern_add_color_stop_rgb(pattern, offset, r, g, b)
// Adds a color stop to a gradient pattern.
func (cb *CairoBindings) patternAddColorStopRGB(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	args := getAllArgs(c)

	pattern, err := getPatternArg(args, 0)
	if err != nil {
		return nil, fmt.Errorf("cairo_pattern_add_color_stop_rgb: pattern: %w", err)
	}
	offset, err := getFloatArg(args, 1)
	if err != nil {
		return nil, fmt.Errorf("cairo_pattern_add_color_stop_rgb: offset: %w", err)
	}
	r, err := getFloatArg(args, 2)
	if err != nil {
		return nil, fmt.Errorf("cairo_pattern_add_color_stop_rgb: r: %w", err)
	}
	g, err := getFloatArg(args, 3)
	if err != nil {
		return nil, fmt.Errorf("cairo_pattern_add_color_stop_rgb: g: %w", err)
	}
	b, err := getFloatArg(args, 4)
	if err != nil {
		return nil, fmt.Errorf("cairo_pattern_add_color_stop_rgb: b: %w", err)
	}

	pattern.AddColorStopRGB(offset, r, g, b)
	return c.Next(), nil
}

// patternAddColorStopRGBA handles cairo_pattern_add_color_stop_rgba(pattern, offset, r, g, b, a)
// Adds a color stop with alpha to a gradient pattern.
func (cb *CairoBindings) patternAddColorStopRGBA(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	args := getAllArgs(c)

	pattern, err := getPatternArg(args, 0)
	if err != nil {
		return nil, fmt.Errorf("cairo_pattern_add_color_stop_rgba: pattern: %w", err)
	}
	offset, err := getFloatArg(args, 1)
	if err != nil {
		return nil, fmt.Errorf("cairo_pattern_add_color_stop_rgba: offset: %w", err)
	}
	r, err := getFloatArg(args, 2)
	if err != nil {
		return nil, fmt.Errorf("cairo_pattern_add_color_stop_rgba: r: %w", err)
	}
	g, err := getFloatArg(args, 3)
	if err != nil {
		return nil, fmt.Errorf("cairo_pattern_add_color_stop_rgba: g: %w", err)
	}
	b, err := getFloatArg(args, 4)
	if err != nil {
		return nil, fmt.Errorf("cairo_pattern_add_color_stop_rgba: b: %w", err)
	}
	a, err := getFloatArg(args, 5)
	if err != nil {
		return nil, fmt.Errorf("cairo_pattern_add_color_stop_rgba: a: %w", err)
	}

	pattern.AddColorStopRGBA(offset, r, g, b, a)
	return c.Next(), nil
}

// setSource handles cairo_set_source(cr, pattern)
// Sets the source pattern for subsequent drawing operations.
func (cb *CairoBindings) setSource(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	args := getAllArgs(c)

	// Get renderer from first arg (cr)
	renderer, offset := cb.getRendererFromArgs(args)

	pattern, err := getPatternArg(args, offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_set_source: pattern: %w", err)
	}

	renderer.SetSource(pattern)
	return c.Next(), nil
}

// getRendererFromArgs extracts a renderer from the args if present, otherwise returns default.
// Returns the renderer and the offset to use for remaining arguments.
// The first argument's userdata value can be *render.CairoRenderer, *render.CairoContext,
// or *sharedContext. If none match, returns the default shared renderer with offset 0.
func (cb *CairoBindings) getRendererFromArgs(args []rt.Value) (*render.CairoRenderer, int) {
	if len(args) > 0 {
		if ud, ok := args[0].TryUserData(); ok {
			switch v := ud.Value().(type) {
			case *render.CairoRenderer:
				return v, 1
			case *render.CairoContext:
				if r := v.Renderer(); r != nil {
					return r, 1
				}
			case *sharedContext:
				return v.renderer, 1
			}
		}
	}
	return cb.renderer, 0
}

// --- Pattern Extend Functions ---

// patternSetExtend handles cairo_pattern_set_extend(pattern, extend)
func (cb *CairoBindings) patternSetExtend(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	args := getAllArgs(c)
	pattern, err := getPatternArg(args, 0)
	if err != nil {
		return nil, fmt.Errorf("cairo_pattern_set_extend: pattern: %w", err)
	}
	extend, err := getIntArg(args, 1)
	if err != nil {
		return nil, fmt.Errorf("cairo_pattern_set_extend: extend: %w", err)
	}
	pattern.SetExtend(render.PatternExtend(extend))
	return c.Next(), nil
}

// patternGetExtend handles cairo_pattern_get_extend(pattern)
func (cb *CairoBindings) patternGetExtend(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	args := getAllArgs(c)
	pattern, err := getPatternArg(args, 0)
	if err != nil {
		return nil, fmt.Errorf("cairo_pattern_get_extend: pattern: %w", err)
	}
	extend := pattern.GetExtend()
	return c.PushingNext1(t.Runtime, rt.IntValue(int64(extend))), nil
}

// --- Matrix Functions ---

// getMatrix handles cairo_get_matrix(cr)
func (cb *CairoBindings) getMatrix(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, _ := cb.getRendererFromContext(c)
	m := renderer.GetMatrix()
	ud := rt.NewUserData(m, nil)
	return c.PushingNext1(t.Runtime, rt.UserDataValue(ud)), nil
}

// setMatrix handles cairo_set_matrix(cr, matrix)
func (cb *CairoBindings) setMatrix(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, offset := cb.getRendererFromContext(c)
	args := getAllArgs(c)
	m, err := getMatrixArg(args, offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_set_matrix: matrix: %w", err)
	}
	renderer.SetMatrix(m)
	return c.Next(), nil
}

// transform handles cairo_transform(cr, matrix)
func (cb *CairoBindings) transform(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, offset := cb.getRendererFromContext(c)
	args := getAllArgs(c)
	m, err := getMatrixArg(args, offset)
	if err != nil {
		return nil, fmt.Errorf("cairo_transform: matrix: %w", err)
	}
	renderer.Transform(m)
	return c.Next(), nil
}

// matrixInit handles cairo_matrix_init(xx, yx, xy, yy, x0, y0)
func (cb *CairoBindings) matrixInit(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	args := getAllArgs(c)
	xx, err := getFloatArg(args, 0)
	if err != nil {
		return nil, fmt.Errorf("cairo_matrix_init: xx: %w", err)
	}
	yx, err := getFloatArg(args, 1)
	if err != nil {
		return nil, fmt.Errorf("cairo_matrix_init: yx: %w", err)
	}
	xy, err := getFloatArg(args, 2)
	if err != nil {
		return nil, fmt.Errorf("cairo_matrix_init: xy: %w", err)
	}
	yy, err := getFloatArg(args, 3)
	if err != nil {
		return nil, fmt.Errorf("cairo_matrix_init: yy: %w", err)
	}
	x0, err := getFloatArg(args, 4)
	if err != nil {
		return nil, fmt.Errorf("cairo_matrix_init: x0: %w", err)
	}
	y0, err := getFloatArg(args, 5)
	if err != nil {
		return nil, fmt.Errorf("cairo_matrix_init: y0: %w", err)
	}
	m := &render.CairoMatrix{XX: xx, YX: yx, XY: xy, YY: yy, X0: x0, Y0: y0}
	ud := rt.NewUserData(m, nil)
	return c.PushingNext1(t.Runtime, rt.UserDataValue(ud)), nil
}

// matrixInitIdentity handles cairo_matrix_init_identity()
func (cb *CairoBindings) matrixInitIdentity(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	m := render.NewIdentityMatrix()
	ud := rt.NewUserData(m, nil)
	return c.PushingNext1(t.Runtime, rt.UserDataValue(ud)), nil
}

// matrixInitTranslate handles cairo_matrix_init_translate(tx, ty)
func (cb *CairoBindings) matrixInitTranslate(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	args := getAllArgs(c)
	tx, err := getFloatArg(args, 0)
	if err != nil {
		return nil, fmt.Errorf("cairo_matrix_init_translate: tx: %w", err)
	}
	ty, err := getFloatArg(args, 1)
	if err != nil {
		return nil, fmt.Errorf("cairo_matrix_init_translate: ty: %w", err)
	}
	m := render.NewTranslateMatrix(tx, ty)
	ud := rt.NewUserData(m, nil)
	return c.PushingNext1(t.Runtime, rt.UserDataValue(ud)), nil
}

// matrixInitScale handles cairo_matrix_init_scale(sx, sy)
func (cb *CairoBindings) matrixInitScale(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	args := getAllArgs(c)
	sx, err := getFloatArg(args, 0)
	if err != nil {
		return nil, fmt.Errorf("cairo_matrix_init_scale: sx: %w", err)
	}
	sy, err := getFloatArg(args, 1)
	if err != nil {
		return nil, fmt.Errorf("cairo_matrix_init_scale: sy: %w", err)
	}
	m := render.NewScaleMatrix(sx, sy)
	ud := rt.NewUserData(m, nil)
	return c.PushingNext1(t.Runtime, rt.UserDataValue(ud)), nil
}

// matrixInitRotate handles cairo_matrix_init_rotate(angle)
func (cb *CairoBindings) matrixInitRotate(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	args := getAllArgs(c)
	angle, err := getFloatArg(args, 0)
	if err != nil {
		return nil, fmt.Errorf("cairo_matrix_init_rotate: angle: %w", err)
	}
	m := render.NewRotateMatrix(angle)
	ud := rt.NewUserData(m, nil)
	return c.PushingNext1(t.Runtime, rt.UserDataValue(ud)), nil
}

// matrixTranslate handles cairo_matrix_translate(matrix, tx, ty)
func (cb *CairoBindings) matrixTranslate(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	args := getAllArgs(c)
	m, err := getMatrixArg(args, 0)
	if err != nil {
		return nil, fmt.Errorf("cairo_matrix_translate: matrix: %w", err)
	}
	tx, err := getFloatArg(args, 1)
	if err != nil {
		return nil, fmt.Errorf("cairo_matrix_translate: tx: %w", err)
	}
	ty, err := getFloatArg(args, 2)
	if err != nil {
		return nil, fmt.Errorf("cairo_matrix_translate: ty: %w", err)
	}
	m.Translate(tx, ty)
	return c.Next(), nil
}

// matrixScale handles cairo_matrix_scale(matrix, sx, sy)
func (cb *CairoBindings) matrixScale(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	args := getAllArgs(c)
	m, err := getMatrixArg(args, 0)
	if err != nil {
		return nil, fmt.Errorf("cairo_matrix_scale: matrix: %w", err)
	}
	sx, err := getFloatArg(args, 1)
	if err != nil {
		return nil, fmt.Errorf("cairo_matrix_scale: sx: %w", err)
	}
	sy, err := getFloatArg(args, 2)
	if err != nil {
		return nil, fmt.Errorf("cairo_matrix_scale: sy: %w", err)
	}
	m.Scale(sx, sy)
	return c.Next(), nil
}

// matrixRotate handles cairo_matrix_rotate(matrix, angle)
func (cb *CairoBindings) matrixRotate(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	args := getAllArgs(c)
	m, err := getMatrixArg(args, 0)
	if err != nil {
		return nil, fmt.Errorf("cairo_matrix_rotate: matrix: %w", err)
	}
	angle, err := getFloatArg(args, 1)
	if err != nil {
		return nil, fmt.Errorf("cairo_matrix_rotate: angle: %w", err)
	}
	m.Rotate(angle)
	return c.Next(), nil
}

// matrixInvert handles cairo_matrix_invert(matrix)
func (cb *CairoBindings) matrixInvert(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	args := getAllArgs(c)
	m, err := getMatrixArg(args, 0)
	if err != nil {
		return nil, fmt.Errorf("cairo_matrix_invert: matrix: %w", err)
	}
	success := m.Invert()
	if !success {
		return nil, fmt.Errorf("cairo_matrix_invert: matrix is singular")
	}
	return c.Next(), nil
}

// matrixMultiply handles cairo_matrix_multiply(result, a, b)
func (cb *CairoBindings) matrixMultiply(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	args := getAllArgs(c)
	result, err := getMatrixArg(args, 0)
	if err != nil {
		return nil, fmt.Errorf("cairo_matrix_multiply: result: %w", err)
	}
	a, err := getMatrixArg(args, 1)
	if err != nil {
		return nil, fmt.Errorf("cairo_matrix_multiply: a: %w", err)
	}
	b, err := getMatrixArg(args, 2)
	if err != nil {
		return nil, fmt.Errorf("cairo_matrix_multiply: b: %w", err)
	}
	// result = a * b
	result.XX = a.XX*b.XX + a.XY*b.YX
	result.XY = a.XX*b.XY + a.XY*b.YY
	result.YX = a.YX*b.XX + a.YY*b.YX
	result.YY = a.YX*b.XY + a.YY*b.YY
	result.X0 = a.XX*b.X0 + a.XY*b.Y0 + a.X0
	result.Y0 = a.YX*b.X0 + a.YY*b.Y0 + a.Y0
	return c.Next(), nil
}

// matrixTransformPoint handles cairo_matrix_transform_point(matrix, x, y)
func (cb *CairoBindings) matrixTransformPoint(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	args := getAllArgs(c)
	m, err := getMatrixArg(args, 0)
	if err != nil {
		return nil, fmt.Errorf("cairo_matrix_transform_point: matrix: %w", err)
	}
	x, err := getFloatArg(args, 1)
	if err != nil {
		return nil, fmt.Errorf("cairo_matrix_transform_point: x: %w", err)
	}
	y, err := getFloatArg(args, 2)
	if err != nil {
		return nil, fmt.Errorf("cairo_matrix_transform_point: y: %w", err)
	}
	tx, ty := m.TransformPoint(x, y)
	return c.PushingNext(t.Runtime, rt.FloatValue(tx), rt.FloatValue(ty)), nil
}

// matrixTransformDistance handles cairo_matrix_transform_distance(matrix, dx, dy)
func (cb *CairoBindings) matrixTransformDistance(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	args := getAllArgs(c)
	m, err := getMatrixArg(args, 0)
	if err != nil {
		return nil, fmt.Errorf("cairo_matrix_transform_distance: matrix: %w", err)
	}
	dx, err := getFloatArg(args, 1)
	if err != nil {
		return nil, fmt.Errorf("cairo_matrix_transform_distance: dx: %w", err)
	}
	dy, err := getFloatArg(args, 2)
	if err != nil {
		return nil, fmt.Errorf("cairo_matrix_transform_distance: dy: %w", err)
	}
	tdx, tdy := m.TransformDistance(dx, dy)
	return c.PushingNext(t.Runtime, rt.FloatValue(tdx), rt.FloatValue(tdy)), nil
}

// --- Surface Functions ---

// surfaceFlush handles cairo_surface_flush(surface)
func (cb *CairoBindings) surfaceFlush(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
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
	surface.Flush()
	return c.Next(), nil
}

// surfaceMarkDirty handles cairo_surface_mark_dirty(surface)
func (cb *CairoBindings) surfaceMarkDirty(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
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
	surface.MarkDirty()
	return c.Next(), nil
}

// surfaceMarkDirtyRectangle handles cairo_surface_mark_dirty_rectangle(surface, x, y, width, height)
func (cb *CairoBindings) surfaceMarkDirtyRectangle(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	args := getAllArgs(c)
	if len(args) < 5 {
		return c.Next(), nil
	}
	surfaceVal, ok := args[0].TryUserData()
	if !ok {
		return c.Next(), nil
	}
	surface, ok := surfaceVal.Value().(*render.CairoSurface)
	if !ok {
		return c.Next(), nil
	}
	x, _ := getIntArg(args, 1)
	y, _ := getIntArg(args, 2)
	width, _ := getIntArg(args, 3)
	height, _ := getIntArg(args, 4)
	surface.MarkDirtyRectangle(int(x), int(y), int(width), int(height))
	return c.Next(), nil
}

// imageSurfaceCreateFromPNG handles cairo_image_surface_create_from_png(filename)
// Loads a PNG image file and creates a Cairo surface from it.
func (cb *CairoBindings) imageSurfaceCreateFromPNG(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	filename, err := c.StringArg(0)
	if err != nil {
		return nil, fmt.Errorf("cairo_image_surface_create_from_png: filename: %w", err)
	}

	surface, err := render.NewCairoSurfaceFromPNG(filename)
	if err != nil {
		// Return nil and error message (or just nil for Cairo compatibility)
		return c.PushingNext(t.Runtime, rt.NilValue, rt.StringValue(err.Error())), nil
	}

	ud := rt.NewUserData(surface, nil)
	return c.PushingNext1(t.Runtime, rt.UserDataValue(ud)), nil
}

// surfaceWriteToPNG handles cairo_surface_write_to_png(surface, filename)
// Saves the surface to a PNG image file.
func (cb *CairoBindings) surfaceWriteToPNG(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	args := getAllArgs(c)
	if len(args) < 2 {
		return nil, fmt.Errorf("cairo_surface_write_to_png: requires surface and filename arguments")
	}

	surfaceVal, ok := args[0].TryUserData()
	if !ok {
		return nil, fmt.Errorf("cairo_surface_write_to_png: first argument must be a surface")
	}
	surface, ok := surfaceVal.Value().(*render.CairoSurface)
	if !ok {
		return nil, fmt.Errorf("cairo_surface_write_to_png: first argument must be a surface")
	}

	filenameVal := args[1]
	filename, ok := filenameVal.TryString()
	if !ok {
		return nil, fmt.Errorf("cairo_surface_write_to_png: filename must be a string")
	}

	err := surface.WriteToPNG(filename)
	if err != nil {
		// Return error status (non-zero means error in Cairo)
		return c.PushingNext1(t.Runtime, rt.IntValue(1)), nil
	}

	// Return success status (0 in Cairo)
	return c.PushingNext1(t.Runtime, rt.IntValue(0)), nil
}

// getMatrixArg extracts a CairoMatrix from the args slice
func getMatrixArg(args []rt.Value, idx int) (*render.CairoMatrix, error) {
	if idx >= len(args) {
		return nil, fmt.Errorf("argument %d out of range (have %d)", idx, len(args))
	}
	if ud, ok := args[idx].TryUserData(); ok {
		if m, ok := ud.Value().(*render.CairoMatrix); ok {
			return m, nil
		}
	}
	return nil, fmt.Errorf("argument %d is not a matrix", idx)
}

// --- Dash and Miter Functions ---

// setDash handles cairo_set_dash(dashes, offset)
func (cb *CairoBindings) setDash(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, offset := cb.getRendererFromContext(c)
	args := getAllArgs(c)
	// Get dashes table
	if offset >= len(args) {
		return c.Next(), nil
	}
	dashesVal := args[offset]
	dashOffset := 0.0
	if offset+1 < len(args) {
		if f, ok := args[offset+1].TryFloat(); ok {
			dashOffset = f
		} else if i, ok := args[offset+1].TryInt(); ok {
			dashOffset = float64(i)
		}
	}
	// Parse dashes table - iterate through consecutive indices
	var dashes []float64
	if tbl, ok := dashesVal.TryTable(); ok {
		for i := int64(1); ; i++ {
			v := tbl.Get(rt.IntValue(i))
			if v == rt.NilValue {
				break
			}
			if f, ok := v.TryFloat(); ok {
				dashes = append(dashes, f)
			} else if iv, ok := v.TryInt(); ok {
				dashes = append(dashes, float64(iv))
			}
		}
	}
	renderer.SetDash(dashes, dashOffset)
	return c.Next(), nil
}

// getDash handles cairo_get_dash() -> returns dashes table, offset
func (cb *CairoBindings) getDash(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, _ := cb.getRendererFromContext(c)
	dashes, offset := renderer.GetDash()
	// Create Lua table for dashes
	dashTable := rt.NewTable()
	for i, d := range dashes {
		dashTable.Set(rt.IntValue(int64(i+1)), rt.FloatValue(d))
	}
	return c.PushingNext(t.Runtime, rt.TableValue(dashTable), rt.FloatValue(offset)), nil
}

// getDashCount handles cairo_get_dash_count()
func (cb *CairoBindings) getDashCount(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, _ := cb.getRendererFromContext(c)
	count := renderer.GetDashCount()
	return c.PushingNext1(t.Runtime, rt.IntValue(int64(count))), nil
}

// setMiterLimit handles cairo_set_miter_limit(limit)
func (cb *CairoBindings) setMiterLimit(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, offset := cb.getRendererFromContext(c)
	args := getAllArgs(c)
	limit, _ := getFloatArg(args, offset)
	renderer.SetMiterLimit(limit)
	return c.Next(), nil
}

// getMiterLimit handles cairo_get_miter_limit()
func (cb *CairoBindings) getMiterLimit(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, _ := cb.getRendererFromContext(c)
	limit := renderer.GetMiterLimit()
	return c.PushingNext1(t.Runtime, rt.FloatValue(limit)), nil
}

// --- Fill Rule and Operator Functions ---

// setFillRule handles cairo_set_fill_rule(rule)
func (cb *CairoBindings) setFillRule(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	args := getAllArgs(c)
	offset := 0
	renderer := cb.renderer
	if len(args) > 0 {
		if ud, ok := args[0].TryUserData(); ok {
			if ctx, ok := ud.Value().(*sharedContext); ok {
				renderer = ctx.renderer
				offset = 1
			}
		}
	}
	rule, _ := getIntArg(args, offset)
	renderer.SetFillRule(int(rule))
	return c.Next(), nil
}

// getFillRule handles cairo_get_fill_rule()
func (cb *CairoBindings) getFillRule(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	args := getAllArgs(c)
	renderer := cb.renderer
	if len(args) > 0 {
		if ud, ok := args[0].TryUserData(); ok {
			if ctx, ok := ud.Value().(*sharedContext); ok {
				renderer = ctx.renderer
			}
		}
	}
	rule := renderer.GetFillRule()
	return c.PushingNext1(t.Runtime, rt.IntValue(int64(rule))), nil
}

// setOperator handles cairo_set_operator(op)
func (cb *CairoBindings) setOperator(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	args := getAllArgs(c)
	offset := 0
	renderer := cb.renderer
	if len(args) > 0 {
		if ud, ok := args[0].TryUserData(); ok {
			if ctx, ok := ud.Value().(*sharedContext); ok {
				renderer = ctx.renderer
				offset = 1
			}
		}
	}
	op, _ := getIntArg(args, offset)
	renderer.SetOperator(int(op))
	return c.Next(), nil
}

// getOperator handles cairo_get_operator()
func (cb *CairoBindings) getOperator(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	args := getAllArgs(c)
	renderer := cb.renderer
	if len(args) > 0 {
		if ud, ok := args[0].TryUserData(); ok {
			if ctx, ok := ud.Value().(*sharedContext); ok {
				renderer = ctx.renderer
			}
		}
	}
	op := renderer.GetOperator()
	return c.PushingNext1(t.Runtime, rt.IntValue(int64(op))), nil
}

// --- Line Property Getters ---

// getLineWidth handles cairo_get_line_width()
func (cb *CairoBindings) getLineWidth(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, _ := cb.getRendererFromContext(c)
	width := renderer.GetLineWidth()
	return c.PushingNext1(t.Runtime, rt.FloatValue(width)), nil
}

// getLineCap handles cairo_get_line_cap()
func (cb *CairoBindings) getLineCap(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, _ := cb.getRendererFromContext(c)
	cap := renderer.GetLineCap()
	return c.PushingNext1(t.Runtime, rt.IntValue(int64(cap))), nil
}

// getLineJoin handles cairo_get_line_join()
func (cb *CairoBindings) getLineJoin(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, _ := cb.getRendererFromContext(c)
	join := renderer.GetLineJoin()
	return c.PushingNext1(t.Runtime, rt.IntValue(int64(join))), nil
}

// getAntialias handles cairo_get_antialias()
func (cb *CairoBindings) getAntialias(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	renderer, _ := cb.getRendererFromContext(c)
	aa := renderer.GetAntialias()
	if aa {
		return c.PushingNext1(t.Runtime, rt.IntValue(1)), nil // CAIRO_ANTIALIAS_DEFAULT
	}
	return c.PushingNext1(t.Runtime, rt.IntValue(0)), nil // CAIRO_ANTIALIAS_NONE
}

// --- Hit Testing Functions ---

// inFill handles cairo_in_fill(x, y)
func (cb *CairoBindings) inFill(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	args := getAllArgs(c)
	renderer := cb.renderer
	startIdx := 0
	if len(args) > 2 {
		if ud, ok := args[0].TryUserData(); ok {
			if ctx, ok := ud.Value().(*sharedContext); ok {
				renderer = ctx.renderer
				startIdx = 1
			}
		}
	}
	x, _ := args[startIdx].TryFloat()
	y, _ := args[startIdx+1].TryFloat()
	result := renderer.InFill(x, y)
	return c.PushingNext1(t.Runtime, rt.BoolValue(result)), nil
}

// inStroke handles cairo_in_stroke(x, y)
func (cb *CairoBindings) inStroke(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	args := getAllArgs(c)
	renderer := cb.renderer
	startIdx := 0
	if len(args) > 2 {
		if ud, ok := args[0].TryUserData(); ok {
			if ctx, ok := ud.Value().(*sharedContext); ok {
				renderer = ctx.renderer
				startIdx = 1
			}
		}
	}
	x, _ := args[startIdx].TryFloat()
	y, _ := args[startIdx+1].TryFloat()
	result := renderer.InStroke(x, y)
	return c.PushingNext1(t.Runtime, rt.BoolValue(result)), nil
}

// --- Path Extent Functions ---

// strokeExtents handles cairo_stroke_extents()
func (cb *CairoBindings) strokeExtents(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	args := getAllArgs(c)
	renderer := cb.renderer
	if len(args) > 0 {
		if ud, ok := args[0].TryUserData(); ok {
			if ctx, ok := ud.Value().(*sharedContext); ok {
				renderer = ctx.renderer
			}
		}
	}
	x1, y1, x2, y2 := renderer.StrokeExtents()
	return c.PushingNext(t.Runtime, rt.FloatValue(x1), rt.FloatValue(y1),
		rt.FloatValue(x2), rt.FloatValue(y2)), nil
}

// fillExtents handles cairo_fill_extents()
func (cb *CairoBindings) fillExtents(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	args := getAllArgs(c)
	renderer := cb.renderer
	if len(args) > 0 {
		if ud, ok := args[0].TryUserData(); ok {
			if ctx, ok := ud.Value().(*sharedContext); ok {
				renderer = ctx.renderer
			}
		}
	}
	x1, y1, x2, y2 := renderer.FillExtents()
	return c.PushingNext(t.Runtime, rt.FloatValue(x1), rt.FloatValue(y1),
		rt.FloatValue(x2), rt.FloatValue(y2)), nil
}

// --- Font Functions ---

// fontExtents handles cairo_font_extents()
func (cb *CairoBindings) fontExtents(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	args := getAllArgs(c)
	renderer := cb.renderer
	if len(args) > 0 {
		if ud, ok := args[0].TryUserData(); ok {
			if ctx, ok := ud.Value().(*sharedContext); ok {
				renderer = ctx.renderer
			}
		}
	}
	extents := renderer.FontExtents()
	table := rt.NewTable()
	table.Set(rt.StringValue("ascent"), rt.FloatValue(extents.Ascent))
	table.Set(rt.StringValue("descent"), rt.FloatValue(extents.Descent))
	table.Set(rt.StringValue("height"), rt.FloatValue(extents.Height))
	table.Set(rt.StringValue("max_x_advance"), rt.FloatValue(extents.MaxXAdvance))
	table.Set(rt.StringValue("max_y_advance"), rt.FloatValue(extents.MaxYAdvance))
	return c.PushingNext1(t.Runtime, rt.TableValue(table)), nil
}

// getFontFace handles cairo_get_font_face()
func (cb *CairoBindings) getFontFace(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	args := getAllArgs(c)
	renderer := cb.renderer
	if len(args) > 0 {
		if ud, ok := args[0].TryUserData(); ok {
			if ctx, ok := ud.Value().(*sharedContext); ok {
				renderer = ctx.renderer
			}
		}
	}
	face := renderer.GetFontFace()
	return c.PushingNext1(t.Runtime, rt.StringValue(face)), nil
}

// getFontSize handles cairo_get_font_size()
func (cb *CairoBindings) getFontSize(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	args := getAllArgs(c)
	renderer := cb.renderer
	if len(args) > 0 {
		if ud, ok := args[0].TryUserData(); ok {
			if ctx, ok := ud.Value().(*sharedContext); ok {
				renderer = ctx.renderer
			}
		}
	}
	size := renderer.GetFontSize()
	return c.PushingNext1(t.Runtime, rt.FloatValue(size)), nil
}

// --- Sub-path Function ---

// newSubPath handles cairo_new_sub_path()
func (cb *CairoBindings) newSubPath(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	args := getAllArgs(c)
	renderer := cb.renderer
	if len(args) > 0 {
		if ud, ok := args[0].TryUserData(); ok {
			if ctx, ok := ud.Value().(*sharedContext); ok {
				renderer = ctx.renderer
			}
		}
	}
	renderer.NewSubPath()
	return c.Next(), nil
}

// --- Coordinate Transformation Functions ---

// userToDevice handles cairo_user_to_device(x, y)
func (cb *CairoBindings) userToDevice(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	args := getAllArgs(c)
	renderer := cb.renderer
	startIdx := 0
	if len(args) > 2 {
		if ud, ok := args[0].TryUserData(); ok {
			if ctx, ok := ud.Value().(*sharedContext); ok {
				renderer = ctx.renderer
				startIdx = 1
			}
		}
	}
	x, _ := args[startIdx].TryFloat()
	y, _ := args[startIdx+1].TryFloat()
	dx, dy := renderer.UserToDevice(x, y)
	return c.PushingNext(t.Runtime, rt.FloatValue(dx), rt.FloatValue(dy)), nil
}

// userToDeviceDistance handles cairo_user_to_device_distance(dx, dy)
func (cb *CairoBindings) userToDeviceDistance(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	args := getAllArgs(c)
	renderer := cb.renderer
	startIdx := 0
	if len(args) > 2 {
		if ud, ok := args[0].TryUserData(); ok {
			if ctx, ok := ud.Value().(*sharedContext); ok {
				renderer = ctx.renderer
				startIdx = 1
			}
		}
	}
	dx, _ := args[startIdx].TryFloat()
	dy, _ := args[startIdx+1].TryFloat()
	ddx, ddy := renderer.UserToDeviceDistance(dx, dy)
	return c.PushingNext(t.Runtime, rt.FloatValue(ddx), rt.FloatValue(ddy)), nil
}

// deviceToUser handles cairo_device_to_user(x, y)
func (cb *CairoBindings) deviceToUser(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	args := getAllArgs(c)
	renderer := cb.renderer
	startIdx := 0
	if len(args) > 2 {
		if ud, ok := args[0].TryUserData(); ok {
			if ctx, ok := ud.Value().(*sharedContext); ok {
				renderer = ctx.renderer
				startIdx = 1
			}
		}
	}
	dx, _ := args[startIdx].TryFloat()
	dy, _ := args[startIdx+1].TryFloat()
	x, y := renderer.DeviceToUser(dx, dy)
	return c.PushingNext(t.Runtime, rt.FloatValue(x), rt.FloatValue(y)), nil
}

// deviceToUserDistance handles cairo_device_to_user_distance(dx, dy)
func (cb *CairoBindings) deviceToUserDistance(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	args := getAllArgs(c)
	renderer := cb.renderer
	startIdx := 0
	if len(args) > 2 {
		if ud, ok := args[0].TryUserData(); ok {
			if ctx, ok := ud.Value().(*sharedContext); ok {
				renderer = ctx.renderer
				startIdx = 1
			}
		}
	}
	ddx, _ := args[startIdx].TryFloat()
	ddy, _ := args[startIdx+1].TryFloat()
	dx, dy := renderer.DeviceToUserDistance(ddx, ddy)
	return c.PushingNext(t.Runtime, rt.FloatValue(dx), rt.FloatValue(dy)), nil
}

// --- Path Copying Functions ---

// copyPath handles cairo_copy_path()
func (cb *CairoBindings) copyPath(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	args := getAllArgs(c)
	renderer := cb.renderer
	if len(args) > 0 {
		if ud, ok := args[0].TryUserData(); ok {
			if ctx, ok := ud.Value().(*sharedContext); ok {
				renderer = ctx.renderer
			}
		}
	}
	segments := renderer.CopyPath()
	// Return as a table of segments
	table := rt.NewTable()
	for i, seg := range segments {
		segTable := rt.NewTable()
		segTable.Set(rt.StringValue("type"), rt.IntValue(int64(seg.Type)))
		segTable.Set(rt.StringValue("x"), rt.FloatValue(seg.X))
		segTable.Set(rt.StringValue("y"), rt.FloatValue(seg.Y))
		if seg.Type == render.PathCurveTo {
			segTable.Set(rt.StringValue("x1"), rt.FloatValue(seg.X1))
			segTable.Set(rt.StringValue("y1"), rt.FloatValue(seg.Y1))
			segTable.Set(rt.StringValue("x2"), rt.FloatValue(seg.X2))
			segTable.Set(rt.StringValue("y2"), rt.FloatValue(seg.Y2))
		}
		table.Set(rt.IntValue(int64(i+1)), rt.TableValue(segTable))
	}
	return c.PushingNext1(t.Runtime, rt.TableValue(table)), nil
}

// appendPath handles cairo_append_path(path)
func (cb *CairoBindings) appendPath(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	args := getAllArgs(c)
	renderer := cb.renderer
	startIdx := 0
	if len(args) > 1 {
		if ud, ok := args[0].TryUserData(); ok {
			if ctx, ok := ud.Value().(*sharedContext); ok {
				renderer = ctx.renderer
				startIdx = 1
			}
		}
	}
	pathTable, ok := args[startIdx].TryTable()
	if !ok {
		return c.Next(), nil
	}
	var segments []render.PathSegment
	// Iterate through consecutive indices (Lua tables are 1-indexed)
	for i := int64(1); ; i++ {
		v := pathTable.Get(rt.IntValue(i))
		if v == rt.NilValue {
			break
		}
		if segTable, ok := v.TryTable(); ok {
			seg := render.PathSegment{}
			if typeVal := segTable.Get(rt.StringValue("type")); typeVal != rt.NilValue {
				if ti, ok := typeVal.TryInt(); ok {
					seg.Type = render.PathSegmentType(ti)
				}
			}
			if xVal := segTable.Get(rt.StringValue("x")); xVal != rt.NilValue {
				if f, ok := xVal.TryFloat(); ok {
					seg.X = f
				}
			}
			if yVal := segTable.Get(rt.StringValue("y")); yVal != rt.NilValue {
				if f, ok := yVal.TryFloat(); ok {
					seg.Y = f
				}
			}
			if seg.Type == render.PathCurveTo {
				if x1Val := segTable.Get(rt.StringValue("x1")); x1Val != rt.NilValue {
					if f, ok := x1Val.TryFloat(); ok {
						seg.X1 = f
					}
				}
				if y1Val := segTable.Get(rt.StringValue("y1")); y1Val != rt.NilValue {
					if f, ok := y1Val.TryFloat(); ok {
						seg.Y1 = f
					}
				}
				if x2Val := segTable.Get(rt.StringValue("x2")); x2Val != rt.NilValue {
					if f, ok := x2Val.TryFloat(); ok {
						seg.X2 = f
					}
				}
				if y2Val := segTable.Get(rt.StringValue("y2")); y2Val != rt.NilValue {
					if f, ok := y2Val.TryFloat(); ok {
						seg.Y2 = f
					}
				}
			}
			segments = append(segments, seg)
		}
	}
	renderer.AppendPath(segments)
	return c.Next(), nil
}
