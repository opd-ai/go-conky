// Package lua provides Golua integration for conky-go.
// This file implements the 'cairo' module for require('cairo') support
// and the conky_window global, matching the standard Conky Lua patterns.
package lua

import (
	rt "github.com/arnodel/golua/runtime"

	"github.com/opd-ai/go-conky/internal/render"
)

// CairoModule provides Cairo module support for Lua scripts.
// It allows Lua scripts to use `require 'cairo'` to load Cairo functions,
// matching the standard Conky pattern used in existing Lua scripts.
type CairoModule struct {
	runtime  *ConkyRuntime
	renderer *render.CairoRenderer
}

// CairoModuleOption configures a CairoModule instance at construction time.
// This allows callers to inject shared dependencies such as the CairoRenderer.
type CairoModuleOption func(*CairoModule)

// WithCairoRenderer configures a CairoModule to use the provided CairoRenderer
// instance instead of creating its own. This is useful when CairoModule is
// used alongside other Cairo integrations (e.g., CairoBindings) so that all
// components share a single renderer instance.
func WithCairoRenderer(renderer *render.CairoRenderer) CairoModuleOption {
	return func(cm *CairoModule) {
		if renderer != nil {
			cm.renderer = renderer
		}
	}
}

// NewCairoModule creates a new CairoModule instance and registers
// the 'cairo' module in Lua's package.preload table.
// It also sets up the conky_window global table and registers global
// cairo_* functions for backward compatibility with existing scripts.
//
// By default, NewCairoModule creates its own CairoRenderer instance. When
// integrating with other components that also use Cairo (such as
// CairoBindings), callers should provide a shared renderer using
// the WithCairoRenderer option to avoid multiple independent renderer
// instances in the same application.
func NewCairoModule(runtime *ConkyRuntime, opts ...CairoModuleOption) (*CairoModule, error) {
	if runtime == nil {
		return nil, ErrNilRuntime
	}

	cm := &CairoModule{
		runtime: runtime,
	}

	// Apply any provided options (e.g., shared renderer injection).
	for _, opt := range opts {
		if opt != nil {
			opt(cm)
		}
	}

	// Preserve existing behavior: if no renderer was injected, create one.
	if cm.renderer == nil {
		cm.renderer = render.NewCairoRenderer()
	}

	cm.registerModule()
	cm.setupConkyWindow()
	// Always register global cairo_* functions for backward compatibility
	cm.registerFunctionsAsGlobals()

	return cm, nil
}

// Renderer returns the underlying CairoRenderer.
// This allows the rendering loop to set the screen and access the renderer.
func (cm *CairoModule) Renderer() *render.CairoRenderer {
	return cm.renderer
}

// UpdateWindowInfo updates the conky_window global with current window dimensions.
// This should be called each frame before Lua drawing hooks are executed.
func (cm *CairoModule) UpdateWindowInfo(width, height int, display, drawable, visual uintptr) {
	// Create window table outside the lock
	windowTable := rt.NewTable()
	windowTable.Set(rt.StringValue("width"), rt.IntValue(int64(width)))
	windowTable.Set(rt.StringValue("height"), rt.IntValue(int64(height)))

	// These are placeholders for X11 compatibility.
	// In Ebiten, we don't have direct X11 access, but we provide the fields
	// so that scripts checking for nil conky_window work correctly.
	// The actual values are implementation-specific handles.
	windowTable.Set(rt.StringValue("display"), rt.IntValue(int64(display)))
	windowTable.Set(rt.StringValue("drawable"), rt.IntValue(int64(drawable)))
	windowTable.Set(rt.StringValue("visual"), rt.IntValue(int64(visual)))

	// Use the runtime's public API which handles locking
	cm.runtime.SetGlobal("conky_window", rt.TableValue(windowTable))
}

// registerModule registers the cairo module as a global and (when possible) in package.loaded.
//
// Preferred usage:
//
//	Scripts should use the global `cairo` table directly, e.g.:
//	  cairo.set_source_rgb(...)
//	  local cairo = cairo -- if a local alias is desired
//
// Compatibility usage:
//
//	For existing Conky Lua scripts that call `require "cairo"`, this method also
//	registers the module in package.loaded so that `require("cairo")` returns
//	the same table. However, this depends on the Lua `package` table and Golua's
//	`require` implementation being available and enabled.
//
//	In sandboxed or resource-limited contexts, `require` may be disabled or
//	restricted for CPU/memory-safety reasons, so scripts should not rely on
//	`require("cairo")` being available. The global `cairo` table is always
//	created by conky-go and is the recommended access pattern.
func (cm *CairoModule) registerModule() {
	// Create the cairo module table
	cairoTable := rt.NewTable()
	cm.registerModuleFunctions(cairoTable)
	cm.registerModuleConstants(cairoTable)
	cairoTableVal := rt.TableValue(cairoTable)

	// Register cairo as a global table for direct access
	// This allows scripts to use: local cairo = cairo (or just use cairo.*)
	cm.runtime.SetGlobal("cairo", cairoTableVal)

	// Also try to register in package.loaded if available
	pkgVal := cm.runtime.runtime.Registry(rt.StringValue("package"))
	if pkgVal.IsNil() {
		return
	}

	pkgTable, ok := pkgVal.TryTable()
	if !ok {
		return
	}

	// Register in package.loaded so require('cairo') returns the cached module
	loadedVal := pkgTable.Get(rt.StringValue("loaded"))
	loadedTable, ok := loadedVal.TryTable()
	if ok {
		loadedTable.Set(rt.StringValue("cairo"), cairoTableVal)
	}
}

// registerModuleFunctions registers all Cairo functions in the given table.
func (cm *CairoModule) registerModuleFunctions(table *rt.Table) {
	// Color functions
	cm.setTableGoFunction(table, "set_source_rgb", cm.setSourceRGB, 3)
	cm.setTableGoFunction(table, "set_source_rgba", cm.setSourceRGBA, 4)

	// Line style functions
	cm.setTableGoFunction(table, "set_line_width", cm.setLineWidth, 1)
	cm.setTableGoFunction(table, "set_line_cap", cm.setLineCap, 1)
	cm.setTableGoFunction(table, "set_line_join", cm.setLineJoin, 1)
	cm.setTableGoFunction(table, "set_antialias", cm.setAntialias, 1)

	// Path building functions
	cm.setTableGoFunction(table, "new_path", cm.newPath, 0)
	cm.setTableGoFunction(table, "move_to", cm.moveTo, 2)
	cm.setTableGoFunction(table, "line_to", cm.lineTo, 2)
	cm.setTableGoFunction(table, "close_path", cm.closePath, 0)
	cm.setTableGoFunction(table, "arc", cm.arc, 5)
	cm.setTableGoFunction(table, "arc_negative", cm.arcNegative, 5)
	cm.setTableGoFunction(table, "curve_to", cm.curveTo, 6)
	cm.setTableGoFunction(table, "rectangle", cm.rectangle, 4)

	// Drawing functions
	cm.setTableGoFunction(table, "stroke", cm.stroke, 0)
	cm.setTableGoFunction(table, "fill", cm.fill, 0)
	cm.setTableGoFunction(table, "stroke_preserve", cm.strokePreserve, 0)
	cm.setTableGoFunction(table, "fill_preserve", cm.fillPreserve, 0)
	cm.setTableGoFunction(table, "paint", cm.paint, 0)
	cm.setTableGoFunction(table, "paint_with_alpha", cm.paintWithAlpha, 1)

	// Surface management functions
	cm.setTableGoFunction(table, "xlib_surface_create", cm.xlibSurfaceCreate, 5)
	cm.setTableGoFunction(table, "image_surface_create", cm.imageSurfaceCreate, 3)
	cm.setTableGoFunction(table, "create", cm.cairoCreate, 1)
	cm.setTableGoFunction(table, "destroy", cm.cairoDestroy, 1)
	cm.setTableGoFunction(table, "surface_destroy", cm.surfaceDestroy, 1)
}

// registerModuleConstants registers Cairo constants in the module table.
func (cm *CairoModule) registerModuleConstants(table *rt.Table) {
	// Line cap constants
	table.Set(rt.StringValue("LINE_CAP_BUTT"), rt.IntValue(int64(render.LineCapButt)))
	table.Set(rt.StringValue("LINE_CAP_ROUND"), rt.IntValue(int64(render.LineCapRound)))
	table.Set(rt.StringValue("LINE_CAP_SQUARE"), rt.IntValue(int64(render.LineCapSquare)))

	// Line join constants
	table.Set(rt.StringValue("LINE_JOIN_MITER"), rt.IntValue(int64(render.LineJoinMiter)))
	table.Set(rt.StringValue("LINE_JOIN_ROUND"), rt.IntValue(int64(render.LineJoinRound)))
	table.Set(rt.StringValue("LINE_JOIN_BEVEL"), rt.IntValue(int64(render.LineJoinBevel)))

	// Antialias constants
	table.Set(rt.StringValue("ANTIALIAS_NONE"), rt.IntValue(0))
	table.Set(rt.StringValue("ANTIALIAS_DEFAULT"), rt.IntValue(1))

	// Surface format constants
	table.Set(rt.StringValue("FORMAT_ARGB32"), rt.IntValue(0))
	table.Set(rt.StringValue("FORMAT_RGB24"), rt.IntValue(1))
	table.Set(rt.StringValue("FORMAT_A8"), rt.IntValue(2))
	table.Set(rt.StringValue("FORMAT_A1"), rt.IntValue(3))
	table.Set(rt.StringValue("FORMAT_RGB16_565"), rt.IntValue(4))
}

// setTableGoFunction registers a Go function in a Lua table.
func (cm *CairoModule) setTableGoFunction(table *rt.Table, name string, fn rt.GoFunctionFunc, nArgs int) {
	goFunc := rt.NewGoFunction(fn, name, nArgs, false)
	rt.SolemnlyDeclareCompliance(rt.ComplyMemSafe|rt.ComplyCpuSafe, goFunc)
	table.Set(rt.StringValue(name), rt.FunctionValue(goFunc))
}

// registerFunctionsAsGlobals registers Cairo functions as globals.
// This is the fallback when package library is not available.
func (cm *CairoModule) registerFunctionsAsGlobals() {
	// Color functions
	cm.runtime.SetGoFunction("cairo_set_source_rgb", cm.setSourceRGB, 3, false)
	cm.runtime.SetGoFunction("cairo_set_source_rgba", cm.setSourceRGBA, 4, false)

	// Line style functions
	cm.runtime.SetGoFunction("cairo_set_line_width", cm.setLineWidth, 1, false)
	cm.runtime.SetGoFunction("cairo_set_line_cap", cm.setLineCap, 1, false)
	cm.runtime.SetGoFunction("cairo_set_line_join", cm.setLineJoin, 1, false)
	cm.runtime.SetGoFunction("cairo_set_antialias", cm.setAntialias, 1, false)

	// Path building functions
	cm.runtime.SetGoFunction("cairo_new_path", cm.newPath, 0, false)
	cm.runtime.SetGoFunction("cairo_move_to", cm.moveTo, 2, false)
	cm.runtime.SetGoFunction("cairo_line_to", cm.lineTo, 2, false)
	cm.runtime.SetGoFunction("cairo_close_path", cm.closePath, 0, false)
	cm.runtime.SetGoFunction("cairo_arc", cm.arc, 5, false)
	cm.runtime.SetGoFunction("cairo_arc_negative", cm.arcNegative, 5, false)
	cm.runtime.SetGoFunction("cairo_curve_to", cm.curveTo, 6, false)
	cm.runtime.SetGoFunction("cairo_rectangle", cm.rectangle, 4, false)

	// Drawing functions
	cm.runtime.SetGoFunction("cairo_stroke", cm.stroke, 0, false)
	cm.runtime.SetGoFunction("cairo_fill", cm.fill, 0, false)
	cm.runtime.SetGoFunction("cairo_stroke_preserve", cm.strokePreserve, 0, false)
	cm.runtime.SetGoFunction("cairo_fill_preserve", cm.fillPreserve, 0, false)
	cm.runtime.SetGoFunction("cairo_paint", cm.paint, 0, false)
	cm.runtime.SetGoFunction("cairo_paint_with_alpha", cm.paintWithAlpha, 1, false)

	// Surface management functions
	cm.runtime.SetGoFunction("cairo_xlib_surface_create", cm.xlibSurfaceCreate, 5, false)
	cm.runtime.SetGoFunction("cairo_image_surface_create", cm.imageSurfaceCreate, 3, false)
	cm.runtime.SetGoFunction("cairo_create", cm.cairoCreate, 1, false)
	cm.runtime.SetGoFunction("cairo_destroy", cm.cairoDestroy, 1, false)
	cm.runtime.SetGoFunction("cairo_surface_destroy", cm.surfaceDestroy, 1, false)

	// Register Cairo constants as globals
	cm.runtime.SetGlobal("CAIRO_LINE_CAP_BUTT", rt.IntValue(int64(render.LineCapButt)))
	cm.runtime.SetGlobal("CAIRO_LINE_CAP_ROUND", rt.IntValue(int64(render.LineCapRound)))
	cm.runtime.SetGlobal("CAIRO_LINE_CAP_SQUARE", rt.IntValue(int64(render.LineCapSquare)))
	cm.runtime.SetGlobal("CAIRO_LINE_JOIN_MITER", rt.IntValue(int64(render.LineJoinMiter)))
	cm.runtime.SetGlobal("CAIRO_LINE_JOIN_ROUND", rt.IntValue(int64(render.LineJoinRound)))
	cm.runtime.SetGlobal("CAIRO_LINE_JOIN_BEVEL", rt.IntValue(int64(render.LineJoinBevel)))
	cm.runtime.SetGlobal("CAIRO_ANTIALIAS_NONE", rt.IntValue(0))
	cm.runtime.SetGlobal("CAIRO_ANTIALIAS_DEFAULT", rt.IntValue(1))

	// Surface format constants
	cm.runtime.SetGlobal("CAIRO_FORMAT_ARGB32", rt.IntValue(0))
	cm.runtime.SetGlobal("CAIRO_FORMAT_RGB24", rt.IntValue(1))
	cm.runtime.SetGlobal("CAIRO_FORMAT_A8", rt.IntValue(2))
	cm.runtime.SetGlobal("CAIRO_FORMAT_A1", rt.IntValue(3))
	cm.runtime.SetGlobal("CAIRO_FORMAT_RGB16_565", rt.IntValue(4))
}

// setupConkyWindow initializes the conky_window global to nil.
// Scripts check `if conky_window == nil` to determine if the window is available.
// When the rendering loop starts, UpdateWindowInfo() sets conky_window to a table
// with width, height, display, drawable, and visual properties.
func (cm *CairoModule) setupConkyWindow() {
	// Set conky_window to nil until the rendering loop starts
	// and calls UpdateWindowInfo with actual dimensions.
	cm.runtime.SetGlobal("conky_window", rt.NilValue)
}

// --- Cairo Drawing Functions (shared with CairoBindings) ---
// These functions wrap the CairoRenderer methods for Lua access.

func (cm *CairoModule) setSourceRGB(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	r, err := c.FloatArg(0)
	if err != nil {
		return nil, err
	}
	g, err := c.FloatArg(1)
	if err != nil {
		return nil, err
	}
	b, err := c.FloatArg(2)
	if err != nil {
		return nil, err
	}
	cm.renderer.SetSourceRGB(r, g, b)
	return c.Next(), nil
}

func (cm *CairoModule) setSourceRGBA(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	r, err := c.FloatArg(0)
	if err != nil {
		return nil, err
	}
	g, err := c.FloatArg(1)
	if err != nil {
		return nil, err
	}
	b, err := c.FloatArg(2)
	if err != nil {
		return nil, err
	}
	a, err := c.FloatArg(3)
	if err != nil {
		return nil, err
	}
	cm.renderer.SetSourceRGBA(r, g, b, a)
	return c.Next(), nil
}

func (cm *CairoModule) setLineWidth(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	width, err := c.FloatArg(0)
	if err != nil {
		return nil, err
	}
	cm.renderer.SetLineWidth(width)
	return c.Next(), nil
}

func (cm *CairoModule) setLineCap(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	capStyle, err := c.IntArg(0)
	if err != nil {
		return nil, err
	}
	// Validate line cap value against defined constants
	if capStyle < int64(render.LineCapButt) || capStyle > int64(render.LineCapSquare) {
		return nil, ErrInvalidLineCap
	}
	cm.renderer.SetLineCap(render.LineCap(capStyle))
	return c.Next(), nil
}

func (cm *CairoModule) setLineJoin(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	join, err := c.IntArg(0)
	if err != nil {
		return nil, err
	}
	// Validate line join value against defined constants
	if join < int64(render.LineJoinMiter) || join > int64(render.LineJoinBevel) {
		return nil, ErrInvalidLineJoin
	}
	cm.renderer.SetLineJoin(render.LineJoin(join))
	return c.Next(), nil
}

func (cm *CairoModule) setAntialias(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	mode, err := c.IntArg(0)
	if err != nil {
		return nil, err
	}
	cm.renderer.SetAntialias(mode != 0)
	return c.Next(), nil
}

func (cm *CairoModule) newPath(_ *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	cm.renderer.NewPath()
	return c.Next(), nil
}

func (cm *CairoModule) moveTo(_ *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	x, err := c.FloatArg(0)
	if err != nil {
		return nil, err
	}
	y, err := c.FloatArg(1)
	if err != nil {
		return nil, err
	}
	cm.renderer.MoveTo(x, y)
	return c.Next(), nil
}

func (cm *CairoModule) lineTo(_ *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	x, err := c.FloatArg(0)
	if err != nil {
		return nil, err
	}
	y, err := c.FloatArg(1)
	if err != nil {
		return nil, err
	}
	cm.renderer.LineTo(x, y)
	return c.Next(), nil
}

func (cm *CairoModule) closePath(_ *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	cm.renderer.ClosePath()
	return c.Next(), nil
}

func (cm *CairoModule) arc(_ *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	xc, err := c.FloatArg(0)
	if err != nil {
		return nil, err
	}
	yc, err := c.FloatArg(1)
	if err != nil {
		return nil, err
	}
	radius, err := c.FloatArg(2)
	if err != nil {
		return nil, err
	}
	angle1, err := c.FloatArg(3)
	if err != nil {
		return nil, err
	}
	angle2, err := c.FloatArg(4)
	if err != nil {
		return nil, err
	}
	cm.renderer.Arc(xc, yc, radius, angle1, angle2)
	return c.Next(), nil
}

func (cm *CairoModule) arcNegative(_ *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	xc, err := c.FloatArg(0)
	if err != nil {
		return nil, err
	}
	yc, err := c.FloatArg(1)
	if err != nil {
		return nil, err
	}
	radius, err := c.FloatArg(2)
	if err != nil {
		return nil, err
	}
	angle1, err := c.FloatArg(3)
	if err != nil {
		return nil, err
	}
	angle2, err := c.FloatArg(4)
	if err != nil {
		return nil, err
	}
	cm.renderer.ArcNegative(xc, yc, radius, angle1, angle2)
	return c.Next(), nil
}

func (cm *CairoModule) curveTo(_ *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	x1, err := c.FloatArg(0)
	if err != nil {
		return nil, err
	}
	y1, err := c.FloatArg(1)
	if err != nil {
		return nil, err
	}
	x2, err := c.FloatArg(2)
	if err != nil {
		return nil, err
	}
	y2, err := c.FloatArg(3)
	if err != nil {
		return nil, err
	}
	x3, err := c.FloatArg(4)
	if err != nil {
		return nil, err
	}
	y3, err := c.FloatArg(5)
	if err != nil {
		return nil, err
	}
	cm.renderer.CurveTo(x1, y1, x2, y2, x3, y3)
	return c.Next(), nil
}

func (cm *CairoModule) rectangle(_ *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	x, err := c.FloatArg(0)
	if err != nil {
		return nil, err
	}
	y, err := c.FloatArg(1)
	if err != nil {
		return nil, err
	}
	width, err := c.FloatArg(2)
	if err != nil {
		return nil, err
	}
	height, err := c.FloatArg(3)
	if err != nil {
		return nil, err
	}
	cm.renderer.Rectangle(x, y, width, height)
	return c.Next(), nil
}

func (cm *CairoModule) stroke(_ *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	cm.renderer.Stroke()
	return c.Next(), nil
}

func (cm *CairoModule) fill(_ *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	cm.renderer.Fill()
	return c.Next(), nil
}

func (cm *CairoModule) strokePreserve(_ *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	cm.renderer.StrokePreserve()
	return c.Next(), nil
}

func (cm *CairoModule) fillPreserve(_ *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	cm.renderer.FillPreserve()
	return c.Next(), nil
}

func (cm *CairoModule) paint(_ *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	cm.renderer.Paint()
	return c.Next(), nil
}

func (cm *CairoModule) paintWithAlpha(_ *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	alpha, err := c.FloatArg(0)
	if err != nil {
		return nil, err
	}
	cm.renderer.PaintWithAlpha(alpha)
	return c.Next(), nil
}

// --- Surface Management Functions ---

// xlibSurfaceCreate handles cairo_xlib_surface_create(display, drawable, visual, width, height)
func (cm *CairoModule) xlibSurfaceCreate(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	// display, drawable, visual are for compatibility (not used in Ebiten)
	_, err := c.IntArg(0)
	if err != nil {
		return nil, err
	}
	_, err = c.IntArg(1)
	if err != nil {
		return nil, err
	}
	_, err = c.IntArg(2)
	if err != nil {
		return nil, err
	}
	width, err := c.IntArg(3)
	if err != nil {
		return nil, err
	}
	height, err := c.IntArg(4)
	if err != nil {
		return nil, err
	}

	surface := render.NewCairoXlibSurface(0, 0, 0, int(width), int(height))
	ud := rt.NewUserData(surface, nil)
	return c.PushingNext1(t.Runtime, rt.UserDataValue(ud)), nil
}

// imageSurfaceCreate handles cairo_image_surface_create(format, width, height)
func (cm *CairoModule) imageSurfaceCreate(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	_, err := c.IntArg(0) // format - we always use ARGB32
	if err != nil {
		return nil, err
	}
	width, err := c.IntArg(1)
	if err != nil {
		return nil, err
	}
	height, err := c.IntArg(2)
	if err != nil {
		return nil, err
	}

	surface := render.NewCairoSurface(int(width), int(height))
	ud := rt.NewUserData(surface, nil)
	return c.PushingNext1(t.Runtime, rt.UserDataValue(ud)), nil
}

// cairoCreate handles cairo_create(surface)
func (cm *CairoModule) cairoCreate(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	surfaceVal, err := c.UserDataArg(0)
	if err != nil {
		// No surface argument - return context using the shared renderer
		ctx := &moduleContext{renderer: cm.renderer}
		ud := rt.NewUserData(ctx, nil)
		return c.PushingNext1(t.Runtime, rt.UserDataValue(ud)), nil
	}

	surface, ok := surfaceVal.Value().(*render.CairoSurface)
	if !ok {
		return nil, ErrInvalidSurface
	}

	ctx := render.NewCairoContext(surface)
	if ctx == nil {
		return nil, ErrContextCreation
	}

	ud := rt.NewUserData(ctx, nil)
	return c.PushingNext1(t.Runtime, rt.UserDataValue(ud)), nil
}

// cairoDestroy handles cairo_destroy(cr)
func (cm *CairoModule) cairoDestroy(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
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
	case *moduleContext:
		// Don't destroy the shared context
	}
	return c.Next(), nil
}

// surfaceDestroy handles cairo_surface_destroy(surface)
func (cm *CairoModule) surfaceDestroy(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
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

// moduleContext is a wrapper for the global shared renderer.
// This is used when cairo_create is called without a surface argument.
type moduleContext struct {
	renderer *render.CairoRenderer
}
