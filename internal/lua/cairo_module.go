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

// NewCairoModule creates a new CairoModule instance and registers
// the 'cairo' module in Lua's package.preload table.
// It also sets up the conky_window global table and registers global
// cairo_* functions for backward compatibility with existing scripts.
func NewCairoModule(runtime *ConkyRuntime) (*CairoModule, error) {
	if runtime == nil {
		return nil, ErrNilRuntime
	}

	cm := &CairoModule{
		runtime:  runtime,
		renderer: render.NewCairoRenderer(),
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
	cm.runtime.mu.Lock()
	defer cm.runtime.mu.Unlock()

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

	cm.runtime.runtime.GlobalEnv().Set(rt.StringValue("conky_window"), rt.TableValue(windowTable))
}

// registerModule registers the cairo module as a global and in package.loaded.
// The recommended pattern is to use the global cairo table directly (e.g., cairo.set_source_rgb()).
// NOTE: require('cairo') is registered in package.loaded but may fail in resource-limited
// contexts because Golua's require function is not marked as CPU/memory-safe.
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

	// Register Cairo constants as globals
	cm.runtime.SetGlobal("CAIRO_LINE_CAP_BUTT", rt.IntValue(int64(render.LineCapButt)))
	cm.runtime.SetGlobal("CAIRO_LINE_CAP_ROUND", rt.IntValue(int64(render.LineCapRound)))
	cm.runtime.SetGlobal("CAIRO_LINE_CAP_SQUARE", rt.IntValue(int64(render.LineCapSquare)))
	cm.runtime.SetGlobal("CAIRO_LINE_JOIN_MITER", rt.IntValue(int64(render.LineJoinMiter)))
	cm.runtime.SetGlobal("CAIRO_LINE_JOIN_ROUND", rt.IntValue(int64(render.LineJoinRound)))
	cm.runtime.SetGlobal("CAIRO_LINE_JOIN_BEVEL", rt.IntValue(int64(render.LineJoinBevel)))
	cm.runtime.SetGlobal("CAIRO_ANTIALIAS_NONE", rt.IntValue(0))
	cm.runtime.SetGlobal("CAIRO_ANTIALIAS_DEFAULT", rt.IntValue(1))
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
	if capStyle < 0 || capStyle > 2 {
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
	if join < 0 || join > 2 {
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
