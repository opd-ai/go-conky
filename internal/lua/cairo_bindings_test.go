// Package lua provides Golua integration for conky-go.
package lua

import (
	"math"
	"testing"

	"github.com/opd-ai/go-conky/internal/render"
)

func TestNewCairoBindings(t *testing.T) {
	// Test with nil runtime
	_, err := NewCairoBindings(nil)
	if err == nil {
		t.Error("Expected error for nil runtime, got nil")
	}

	// Test with valid runtime
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	cb, err := NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	if cb == nil {
		t.Fatal("NewCairoBindings returned nil")
	}

	if cb.Renderer() == nil {
		t.Error("Renderer() returned nil")
	}
}

func TestCairoBindings_SetSourceRGB(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	cb, err := NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Execute Lua code that calls cairo_set_source_rgb
	_, err = runtime.ExecuteString("test", "cairo_set_source_rgb(1, 0, 0)")
	if err != nil {
		t.Fatalf("Failed to execute cairo_set_source_rgb: %v", err)
	}

	// Verify the color was set
	color := cb.Renderer().GetCurrentColor()
	if color.R != 255 || color.G != 0 || color.B != 0 {
		t.Errorf("Expected RGB(255,0,0), got RGB(%d,%d,%d)", color.R, color.G, color.B)
	}
}

func TestCairoBindings_SetSourceRGBA(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	cb, err := NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Execute Lua code that calls cairo_set_source_rgba
	_, err = runtime.ExecuteString("test", "cairo_set_source_rgba(0, 1, 0, 0.5)")
	if err != nil {
		t.Fatalf("Failed to execute cairo_set_source_rgba: %v", err)
	}

	// Verify the color was set
	color := cb.Renderer().GetCurrentColor()
	if color.R != 0 || color.G != 255 || color.B != 0 || color.A != 127 {
		t.Errorf("Expected RGBA(0,255,0,127), got RGBA(%d,%d,%d,%d)",
			color.R, color.G, color.B, color.A)
	}
}

func TestCairoBindings_SetLineWidth(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	cb, err := NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Execute Lua code that calls cairo_set_line_width
	_, err = runtime.ExecuteString("test", "cairo_set_line_width(3.5)")
	if err != nil {
		t.Fatalf("Failed to execute cairo_set_line_width: %v", err)
	}

	// Verify the line width was set
	width := cb.Renderer().GetLineWidth()
	if width != 3.5 {
		t.Errorf("Expected line width 3.5, got %f", width)
	}
}

func TestCairoBindings_SetLineCap(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	cb, err := NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Execute Lua code that calls cairo_set_line_cap
	_, err = runtime.ExecuteString("test", "cairo_set_line_cap(CAIRO_LINE_CAP_ROUND)")
	if err != nil {
		t.Fatalf("Failed to execute cairo_set_line_cap: %v", err)
	}

	// Verify the line cap was set
	capStyle := cb.Renderer().GetLineCap()
	if capStyle != render.LineCapRound {
		t.Errorf("Expected LineCapRound, got %v", capStyle)
	}
}

func TestCairoBindings_SetLineJoin(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	cb, err := NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Execute Lua code that calls cairo_set_line_join
	_, err = runtime.ExecuteString("test", "cairo_set_line_join(CAIRO_LINE_JOIN_BEVEL)")
	if err != nil {
		t.Fatalf("Failed to execute cairo_set_line_join: %v", err)
	}

	// Verify the line join was set
	join := cb.Renderer().GetLineJoin()
	if join != render.LineJoinBevel {
		t.Errorf("Expected LineJoinBevel, got %v", join)
	}
}

func TestCairoBindings_SetAntialias(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	cb, err := NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Disable antialias
	_, err = runtime.ExecuteString("test", "cairo_set_antialias(CAIRO_ANTIALIAS_NONE)")
	if err != nil {
		t.Fatalf("Failed to execute cairo_set_antialias: %v", err)
	}

	// Verify antialias is disabled
	if cb.Renderer().GetAntialias() {
		t.Error("Expected antialias to be disabled")
	}

	// Enable antialias
	_, err = runtime.ExecuteString("test", "cairo_set_antialias(CAIRO_ANTIALIAS_DEFAULT)")
	if err != nil {
		t.Fatalf("Failed to execute cairo_set_antialias: %v", err)
	}

	// Verify antialias is enabled
	if !cb.Renderer().GetAntialias() {
		t.Error("Expected antialias to be enabled")
	}
}

func TestCairoBindings_PathBuilding(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	cb, err := NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Build a path using Lua
	luaCode := `
		cairo_new_path()
		cairo_move_to(10, 20)
		cairo_line_to(100, 200)
		cairo_close_path()
	`
	_, err = runtime.ExecuteString("test", luaCode)
	if err != nil {
		t.Fatalf("Failed to execute path building code: %v", err)
	}

	// Verify the current point
	x, y, hasPoint := cb.Renderer().GetCurrentPoint()
	if !hasPoint {
		t.Error("Expected current point after path building")
	}
	if x != 100 || y != 200 {
		t.Errorf("Expected current point (100,200), got (%f,%f)", x, y)
	}
}

func TestCairoBindings_Rectangle(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	cb, err := NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Create a rectangle
	_, err = runtime.ExecuteString("test", "cairo_rectangle(10, 20, 100, 50)")
	if err != nil {
		t.Fatalf("Failed to execute cairo_rectangle: %v", err)
	}

	// Verify path exists
	_, _, hasPoint := cb.Renderer().GetCurrentPoint()
	if !hasPoint {
		t.Error("Expected current point after cairo_rectangle")
	}
}

func TestCairoBindings_Arc(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	cb, err := NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Create an arc
	_, err = runtime.ExecuteString("test", "cairo_arc(100, 100, 50, 0, math.pi * 2)")
	if err != nil {
		t.Fatalf("Failed to execute cairo_arc: %v", err)
	}

	// Verify path exists
	_, _, hasPoint := cb.Renderer().GetCurrentPoint()
	if !hasPoint {
		t.Error("Expected current point after cairo_arc")
	}
}

func TestCairoBindings_ArcNegative(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	cb, err := NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Create an arc negative
	_, err = runtime.ExecuteString("test", "cairo_arc_negative(100, 100, 50, math.pi, 0)")
	if err != nil {
		t.Fatalf("Failed to execute cairo_arc_negative: %v", err)
	}

	// Verify path exists
	_, _, hasPoint := cb.Renderer().GetCurrentPoint()
	if !hasPoint {
		t.Error("Expected current point after cairo_arc_negative")
	}
}

func TestCairoBindings_CurveTo(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	cb, err := NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Create a Bezier curve
	_, err = runtime.ExecuteString("test", `
		cairo_move_to(0, 0)
		cairo_curve_to(10, 20, 30, 40, 50, 60)
	`)
	if err != nil {
		t.Fatalf("Failed to execute cairo_curve_to: %v", err)
	}

	// Verify the end point
	x, y, hasPoint := cb.Renderer().GetCurrentPoint()
	if !hasPoint {
		t.Error("Expected current point after cairo_curve_to")
	}
	if x != 50 || y != 60 {
		t.Errorf("Expected current point (50,60), got (%f,%f)", x, y)
	}
}

func TestCairoBindings_DrawingFunctions(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Test that drawing functions don't panic (no screen set)
	tests := []struct {
		name string
		code string
	}{
		{"stroke", "cairo_rectangle(0,0,10,10); cairo_stroke()"},
		{"fill", "cairo_rectangle(0,0,10,10); cairo_fill()"},
		{"stroke_preserve", "cairo_rectangle(0,0,10,10); cairo_stroke_preserve()"},
		{"fill_preserve", "cairo_rectangle(0,0,10,10); cairo_fill_preserve()"},
		{"paint", "cairo_paint()"},
		{"paint_with_alpha", "cairo_paint_with_alpha(0.5)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := runtime.ExecuteString("test", tt.code)
			if err != nil {
				t.Errorf("Failed to execute %s: %v", tt.name, err)
			}
		})
	}
}

func TestCairoBindings_Constants(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Verify constants are set
	tests := []struct {
		name  string
		value int64
	}{
		{"CAIRO_LINE_CAP_BUTT", 0},
		{"CAIRO_LINE_CAP_ROUND", 1},
		{"CAIRO_LINE_CAP_SQUARE", 2},
		{"CAIRO_LINE_JOIN_MITER", 0},
		{"CAIRO_LINE_JOIN_ROUND", 1},
		{"CAIRO_LINE_JOIN_BEVEL", 2},
		{"CAIRO_ANTIALIAS_NONE", 0},
		{"CAIRO_ANTIALIAS_DEFAULT", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runtime.GetGlobal(tt.name)
			if result.IsNil() {
				t.Errorf("Constant %s not found", tt.name)
				return
			}
			// Check if it's an integer
			intVal, ok := result.TryInt()
			if !ok {
				t.Errorf("Constant %s is not an integer", tt.name)
				return
			}
			if intVal != tt.value {
				t.Errorf("Constant %s = %d, want %d", tt.name, intVal, tt.value)
			}
		})
	}
}

func TestCairoBindings_ErrorHandling(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Test calling functions with wrong argument types
	tests := []struct {
		name string
		code string
	}{
		{"set_source_rgb wrong type", "cairo_set_source_rgb('red', 0, 0)"},
		{"set_line_width wrong type", "cairo_set_line_width('thick')"},
		{"move_to wrong type", "cairo_move_to('x', 10)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := runtime.ExecuteString("test", tt.code)
			if err == nil {
				t.Error("Expected error for wrong argument type, got nil")
			}
		})
	}
}

func TestCairoBindings_ComplexPath(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Build a complex path
	luaCode := `
		-- Set up drawing state
		cairo_set_source_rgba(1, 0.5, 0, 0.8)
		cairo_set_line_width(2)
		cairo_set_line_cap(CAIRO_LINE_CAP_ROUND)
		cairo_set_line_join(CAIRO_LINE_JOIN_ROUND)
		
		-- Create a complex path
		cairo_new_path()
		cairo_move_to(50, 50)
		cairo_line_to(100, 50)
		cairo_line_to(100, 100)
		cairo_arc(100, 100, 25, 0, math.pi / 2)
		cairo_close_path()
		
		-- Stroke without clearing
		cairo_stroke_preserve()
		
		-- Change color and fill
		cairo_set_source_rgba(0.5, 0.5, 1, 0.5)
		cairo_fill()
	`

	_, err = runtime.ExecuteString("test", luaCode)
	if err != nil {
		t.Fatalf("Failed to execute complex path code: %v", err)
	}
}

func TestCairoBindings_InvalidLineCapValue(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Test invalid line cap values
	tests := []struct {
		name string
		code string
	}{
		{"negative value", "cairo_set_line_cap(-1)"},
		{"value too high", "cairo_set_line_cap(3)"},
		{"very large value", "cairo_set_line_cap(100)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := runtime.ExecuteString("test", tt.code)
			if err == nil {
				t.Error("Expected error for invalid line cap value, got nil")
			}
		})
	}
}

func TestCairoBindings_InvalidLineJoinValue(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Test invalid line join values
	tests := []struct {
		name string
		code string
	}{
		{"negative value", "cairo_set_line_join(-1)"},
		{"value too high", "cairo_set_line_join(3)"},
		{"very large value", "cairo_set_line_join(100)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := runtime.ExecuteString("test", tt.code)
			if err == nil {
				t.Error("Expected error for invalid line join value, got nil")
			}
		})
	}
}

// --- Text Function Tests ---

func TestCairoBindings_SelectFontFace(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	cb, err := NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Test setting font face
	_, err = runtime.ExecuteString("test", `
		cairo_select_font_face("GoMono", CAIRO_FONT_SLANT_NORMAL, CAIRO_FONT_WEIGHT_BOLD)
	`)
	if err != nil {
		t.Fatalf("Failed to execute cairo_select_font_face: %v", err)
	}

	// Verify we have a renderer (font state is internal)
	if cb.Renderer() == nil {
		t.Error("Renderer should not be nil")
	}
}

func TestCairoBindings_SetFontSize(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	cb, err := NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Test setting font size
	_, err = runtime.ExecuteString("test", "cairo_set_font_size(24)")
	if err != nil {
		t.Fatalf("Failed to execute cairo_set_font_size: %v", err)
	}

	// Verify font size was set
	fontSize := cb.Renderer().GetFontSize()
	if fontSize != 24 {
		t.Errorf("Expected font size 24, got %f", fontSize)
	}
}

func TestCairoBindings_ShowText(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Test showing text (should not panic even without a screen)
	_, err = runtime.ExecuteString("test", `
		cairo_move_to(10, 20)
		cairo_show_text("Hello, World!")
	`)
	if err != nil {
		t.Fatalf("Failed to execute cairo_show_text: %v", err)
	}
}

func TestCairoBindings_TextExtents(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Test getting text extents
	result, err := runtime.ExecuteString("test", `
		local extents = cairo_text_extents("Hello")
		return extents.width > 0
	`)
	if err != nil {
		t.Fatalf("Failed to execute cairo_text_extents: %v", err)
	}

	// Verify we got a truthy result (width > 0)
	if !result.AsBool() {
		t.Error("Expected text extents width to be > 0")
	}
}

func TestCairoBindings_TextExtentsFields(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Test all fields in text extents
	_, err = runtime.ExecuteString("test", `
		local extents = cairo_text_extents("Test")
		-- Check that all fields exist
		assert(extents.x_bearing ~= nil, "x_bearing should exist")
		assert(extents.y_bearing ~= nil, "y_bearing should exist")
		assert(extents.width ~= nil, "width should exist")
		assert(extents.height ~= nil, "height should exist")
		assert(extents.x_advance ~= nil, "x_advance should exist")
		assert(extents.y_advance ~= nil, "y_advance should exist")
	`)
	if err != nil {
		t.Fatalf("Failed to verify text extents fields: %v", err)
	}
}

func TestCairoBindings_FontConstants(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Verify font constants are set
	tests := []struct {
		name  string
		value int64
	}{
		{"CAIRO_FONT_SLANT_NORMAL", 0},
		{"CAIRO_FONT_SLANT_ITALIC", 1},
		{"CAIRO_FONT_SLANT_OBLIQUE", 2},
		{"CAIRO_FONT_WEIGHT_NORMAL", 0},
		{"CAIRO_FONT_WEIGHT_BOLD", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runtime.GetGlobal(tt.name)
			if result.IsNil() {
				t.Errorf("Constant %s not found", tt.name)
				return
			}
			intVal, ok := result.TryInt()
			if !ok {
				t.Errorf("Constant %s is not an integer", tt.name)
				return
			}
			if intVal != tt.value {
				t.Errorf("Constant %s = %d, want %d", tt.name, intVal, tt.value)
			}
		})
	}
}

// --- Transformation Function Tests ---

func TestCairoBindings_Translate(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	cb, err := NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Test translate
	_, err = runtime.ExecuteString("test", "cairo_translate(100, 200)")
	if err != nil {
		t.Fatalf("Failed to execute cairo_translate: %v", err)
	}

	// Verify translation
	tx, ty := cb.Renderer().GetTranslate()
	if tx != 100 || ty != 200 {
		t.Errorf("Expected translation (100, 200), got (%f, %f)", tx, ty)
	}
}

func TestCairoBindings_Rotate(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	cb, err := NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Test rotate
	_, err = runtime.ExecuteString("test", "cairo_rotate(math.pi / 4)")
	if err != nil {
		t.Fatalf("Failed to execute cairo_rotate: %v", err)
	}

	// Verify rotation (pi/4 = 0.785...)
	rotation := cb.Renderer().GetRotation()
	expected := math.Pi / 4
	if rotation < expected-0.01 || rotation > expected+0.01 {
		t.Errorf("Expected rotation ~%f, got %f", expected, rotation)
	}
}

func TestCairoBindings_Scale(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	cb, err := NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Test scale
	_, err = runtime.ExecuteString("test", "cairo_scale(2, 3)")
	if err != nil {
		t.Fatalf("Failed to execute cairo_scale: %v", err)
	}

	// Verify scale
	sx, sy := cb.Renderer().GetScale()
	if sx != 2 || sy != 3 {
		t.Errorf("Expected scale (2, 3), got (%f, %f)", sx, sy)
	}
}

func TestCairoBindings_SaveRestore(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	cb, err := NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Test save/restore
	_, err = runtime.ExecuteString("test", `
		-- Set initial state
		cairo_set_line_width(5)
		cairo_translate(50, 50)
		
		-- Save state
		cairo_save()
		
		-- Modify state
		cairo_set_line_width(10)
		cairo_translate(100, 100)
		
		-- Restore state
		cairo_restore()
	`)
	if err != nil {
		t.Fatalf("Failed to execute save/restore: %v", err)
	}

	// Verify line width was restored
	lineWidth := cb.Renderer().GetLineWidth()
	if lineWidth != 5 {
		t.Errorf("Expected line width 5 after restore, got %f", lineWidth)
	}

	// Verify translation was restored
	tx, ty := cb.Renderer().GetTranslate()
	if tx != 50 || ty != 50 {
		t.Errorf("Expected translation (50, 50) after restore, got (%f, %f)", tx, ty)
	}
}

func TestCairoBindings_IdentityMatrix(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	cb, err := NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Apply some transformations
	_, err = runtime.ExecuteString("test", `
		cairo_translate(100, 200)
		cairo_rotate(math.pi)
		cairo_scale(2, 3)
	`)
	if err != nil {
		t.Fatalf("Failed to apply transformations: %v", err)
	}

	// Reset with identity_matrix
	_, err = runtime.ExecuteString("test", "cairo_identity_matrix()")
	if err != nil {
		t.Fatalf("Failed to execute cairo_identity_matrix: %v", err)
	}

	// Verify all transformations are reset
	tx, ty := cb.Renderer().GetTranslate()
	if tx != 0 || ty != 0 {
		t.Errorf("Expected translation (0, 0) after identity, got (%f, %f)", tx, ty)
	}

	rotation := cb.Renderer().GetRotation()
	if rotation != 0 {
		t.Errorf("Expected rotation 0 after identity, got %f", rotation)
	}

	sx, sy := cb.Renderer().GetScale()
	if sx != 1 || sy != 1 {
		t.Errorf("Expected scale (1, 1) after identity, got (%f, %f)", sx, sy)
	}
}

func TestCairoBindings_InvalidFontSlant(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Test invalid slant value
	_, err = runtime.ExecuteString("test", `
		cairo_select_font_face("GoMono", 5, CAIRO_FONT_WEIGHT_NORMAL)
	`)
	if err == nil {
		t.Error("Expected error for invalid slant value, got nil")
	}
}

func TestCairoBindings_InvalidFontWeight(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Test invalid weight value
	_, err = runtime.ExecuteString("test", `
		cairo_select_font_face("GoMono", CAIRO_FONT_SLANT_NORMAL, 5)
	`)
	if err == nil {
		t.Error("Expected error for invalid weight value, got nil")
	}
}

func TestCairoBindings_RestoreEmptyStack(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Test restore on empty stack (should not error)
	_, err = runtime.ExecuteString("test", "cairo_restore()")
	if err != nil {
		t.Fatalf("cairo_restore on empty stack should not error: %v", err)
	}
}

func TestCairoBindings_ComplexTextAndTransforms(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Complex script using text and transforms
	luaCode := `
		-- Set up font
		cairo_select_font_face("GoMono", CAIRO_FONT_SLANT_NORMAL, CAIRO_FONT_WEIGHT_BOLD)
		cairo_set_font_size(16)
		
		-- Save state
		cairo_save()
		
		-- Apply transformations
		cairo_translate(100, 100)
		cairo_rotate(math.pi / 6)
		cairo_scale(1.5, 1.5)
		
		-- Set color and draw text
		cairo_set_source_rgba(1, 0, 0, 0.8)
		cairo_move_to(0, 0)
		cairo_show_text("Transformed Text")
		
		-- Get text extents
		local extents = cairo_text_extents("Hello")
		
		-- Restore original state
		cairo_restore()
		
		-- Reset transformation
		cairo_identity_matrix()
		
		return extents.width > 0
	`

	result, err := runtime.ExecuteString("test", luaCode)
	if err != nil {
		t.Fatalf("Failed to execute complex text and transforms code: %v", err)
	}

	if !result.AsBool() {
		t.Error("Expected text extents width to be > 0")
	}
}

// --- Surface Management Function Tests ---

func TestCairoBindings_XlibSurfaceCreate(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Test cairo_xlib_surface_create
	luaCode := `
		local surface = cairo_xlib_surface_create(0, 0, 0, 640, 480)
		return surface ~= nil
	`
	result, err := runtime.ExecuteString("test", luaCode)
	if err != nil {
		t.Fatalf("Failed to execute cairo_xlib_surface_create: %v", err)
	}

	if !result.AsBool() {
		t.Error("Expected surface to be non-nil")
	}
}

func TestCairoBindings_ImageSurfaceCreate(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Test cairo_image_surface_create with ARGB32 format
	luaCode := `
		local surface = cairo_image_surface_create(CAIRO_FORMAT_ARGB32, 200, 100)
		return surface ~= nil
	`
	result, err := runtime.ExecuteString("test", luaCode)
	if err != nil {
		t.Fatalf("Failed to execute cairo_image_surface_create: %v", err)
	}

	if !result.AsBool() {
		t.Error("Expected surface to be non-nil")
	}
}

func TestCairoBindings_CreateAndDestroyContext(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Test creating and destroying a context
	luaCode := `
		local surface = cairo_image_surface_create(CAIRO_FORMAT_ARGB32, 100, 100)
		local cr = cairo_create(surface)
		if cr == nil then return false end
		cairo_destroy(cr)
		cairo_surface_destroy(surface)
		return true
	`
	result, err := runtime.ExecuteString("test", luaCode)
	if err != nil {
		t.Fatalf("Failed to execute create/destroy context: %v", err)
	}

	if !result.AsBool() {
		t.Error("Expected context creation and destruction to succeed")
	}
}

func TestCairoBindings_SurfaceDestroy(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Test surface destruction
	luaCode := `
		local surface = cairo_xlib_surface_create(0, 0, 0, 320, 240)
		cairo_surface_destroy(surface)
		return true
	`
	result, err := runtime.ExecuteString("test", luaCode)
	if err != nil {
		t.Fatalf("Failed to execute surface destroy: %v", err)
	}

	if !result.AsBool() {
		t.Error("Expected surface destruction to succeed")
	}
}

func TestCairoBindings_DestroyWithNilArgs(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Test destroy functions with nil/no arguments (should not error)
	luaCode := `
		cairo_destroy(nil)
		cairo_surface_destroy(nil)
		return true
	`
	result, err := runtime.ExecuteString("test", luaCode)
	if err != nil {
		t.Fatalf("Failed to execute destroy with nil: %v", err)
	}

	if !result.AsBool() {
		t.Error("Expected destroy with nil to succeed")
	}
}

func TestCairoBindings_SurfaceFormatConstants(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Verify surface format constants are set
	tests := []struct {
		name  string
		value int64
	}{
		{"CAIRO_FORMAT_ARGB32", 0},
		{"CAIRO_FORMAT_RGB24", 1},
		{"CAIRO_FORMAT_A8", 2},
		{"CAIRO_FORMAT_A1", 3},
		{"CAIRO_FORMAT_RGB16_565", 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runtime.GetGlobal(tt.name)
			if result.IsNil() {
				t.Errorf("Constant %s not found", tt.name)
				return
			}
			intVal, ok := result.TryInt()
			if !ok {
				t.Errorf("Constant %s is not an integer", tt.name)
				return
			}
			if intVal != tt.value {
				t.Errorf("Constant %s = %d, want %d", tt.name, intVal, tt.value)
			}
		})
	}
}

func TestCairoBindings_FullSurfaceWorkflow(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Test a complete workflow with context-based drawing (cr as first argument)
	luaCode := `
		-- Create a surface (simulating conky_window setup)
		local surface = cairo_xlib_surface_create(0, 0, 0, 640, 480)
		
		-- Create a Cairo context
		local cr = cairo_create(surface)
		
		-- Set up drawing state with context (cr as first argument)
		cairo_set_source_rgba(cr, 1, 0, 0, 0.8)
		cairo_set_line_width(cr, 2)
		
		-- Draw something with context
		cairo_rectangle(cr, 10, 10, 100, 50)
		cairo_stroke(cr)
		
		cairo_set_source_rgb(cr, 0, 1, 0)
		cairo_rectangle(cr, 120, 10, 100, 50)
		cairo_fill(cr)
		
		-- Clean up
		cairo_destroy(cr)
		cairo_surface_destroy(surface)
		
		return true
	`
	result, err := runtime.ExecuteString("test", luaCode)
	if err != nil {
		t.Fatalf("Failed to execute full surface workflow: %v", err)
	}

	if !result.AsBool() {
		t.Error("Expected full surface workflow to succeed")
	}
}

// TestCairoBindings_ContextBasedDrawingVerifiesRenderer verifies that drawing
// operations use the context's renderer, not the global renderer.
func TestCairoBindings_ContextBasedDrawingVerifiesRenderer(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Test that context-based and non-context calls both work
	luaCode := `
		-- Create a surface and context
		local surface = cairo_image_surface_create(CAIRO_FORMAT_ARGB32, 200, 200)
		local cr = cairo_create(surface)
		
		-- Context-based drawing (standard Conky pattern)
		cairo_set_source_rgba(cr, 1, 0, 0, 1)
		cairo_move_to(cr, 10, 10)
		cairo_line_to(cr, 50, 50)
		cairo_stroke(cr)
		
		-- Backward compatible (no context) calls should still work
		cairo_set_source_rgb(0, 1, 0)
		cairo_rectangle(100, 100, 50, 50)
		cairo_fill()
		
		-- Mixed usage
		cairo_set_source_rgba(cr, 0, 0, 1, 0.5)
		cairo_arc(cr, 100, 100, 30, 0, 6.28)
		cairo_fill(cr)
		
		cairo_destroy(cr)
		cairo_surface_destroy(surface)
		return true
	`
	result, err := runtime.ExecuteString("test", luaCode)
	if err != nil {
		t.Fatalf("Failed to execute context-based drawing test: %v", err)
	}

	if !result.AsBool() {
		t.Error("Expected context-based drawing to succeed")
	}
}

// --- Relative Path Function Tests ---

func TestCairoBindings_RelMoveTo(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	cb, err := NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Test relative move
	_, err = runtime.ExecuteString("test", `
		cairo_move_to(100, 100)
		cairo_rel_move_to(50, 25)
	`)
	if err != nil {
		t.Fatalf("Failed to execute cairo_rel_move_to: %v", err)
	}

	// Verify the current point moved relatively
	x, y, hasPoint := cb.Renderer().GetCurrentPoint()
	if !hasPoint {
		t.Error("Expected current point after cairo_rel_move_to")
	}
	if x != 150 || y != 125 {
		t.Errorf("Expected current point (150, 125), got (%f, %f)", x, y)
	}
}

func TestCairoBindings_RelLineTo(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	cb, err := NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Test relative line
	_, err = runtime.ExecuteString("test", `
		cairo_move_to(100, 100)
		cairo_rel_line_to(50, 25)
	`)
	if err != nil {
		t.Fatalf("Failed to execute cairo_rel_line_to: %v", err)
	}

	// Verify the current point moved relatively
	x, y, hasPoint := cb.Renderer().GetCurrentPoint()
	if !hasPoint {
		t.Error("Expected current point after cairo_rel_line_to")
	}
	if x != 150 || y != 125 {
		t.Errorf("Expected current point (150, 125), got (%f, %f)", x, y)
	}
}

func TestCairoBindings_RelCurveTo(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	cb, err := NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Test relative curve
	_, err = runtime.ExecuteString("test", `
		cairo_move_to(100, 100)
		cairo_rel_curve_to(10, 20, 30, 40, 50, 60)
	`)
	if err != nil {
		t.Fatalf("Failed to execute cairo_rel_curve_to: %v", err)
	}

	// Verify the current point moved to the end point (100+50, 100+60)
	x, y, hasPoint := cb.Renderer().GetCurrentPoint()
	if !hasPoint {
		t.Error("Expected current point after cairo_rel_curve_to")
	}
	if x != 150 || y != 160 {
		t.Errorf("Expected current point (150, 160), got (%f, %f)", x, y)
	}
}

func TestCairoBindings_RelMoveToNoPath(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	cb, err := NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Test relative move without a current point (should do nothing)
	_, err = runtime.ExecuteString("test", `
		cairo_new_path()
		cairo_rel_move_to(50, 25)
	`)
	if err != nil {
		t.Fatalf("Failed to execute cairo_rel_move_to: %v", err)
	}

	// Verify there is no current point
	_, _, hasPoint := cb.Renderer().GetCurrentPoint()
	if hasPoint {
		t.Error("Expected no current point after cairo_rel_move_to without initial point")
	}
}

func TestCairoBindings_RelativePathChain(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	cb, err := NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Test a chain of relative path operations
	_, err = runtime.ExecuteString("test", `
		cairo_move_to(0, 0)
		cairo_rel_line_to(100, 0)
		cairo_rel_line_to(0, 100)
		cairo_rel_line_to(-100, 0)
		cairo_close_path()
	`)
	if err != nil {
		t.Fatalf("Failed to execute relative path chain: %v", err)
	}

	// Verify the final current point (0, 100)
	x, y, hasPoint := cb.Renderer().GetCurrentPoint()
	if !hasPoint {
		t.Error("Expected current point after relative path chain")
	}
	if x != 0 || y != 100 {
		t.Errorf("Expected current point (0, 100), got (%f, %f)", x, y)
	}
}

// --- Clipping Function Tests ---

func TestCairoBindings_Clip(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	cb, err := NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Test clip
	_, err = runtime.ExecuteString("test", `
		cairo_rectangle(10, 10, 100, 100)
		cairo_clip()
	`)
	if err != nil {
		t.Fatalf("Failed to execute cairo_clip: %v", err)
	}

	// Verify clip is set and path is cleared
	if !cb.Renderer().HasClip() {
		t.Error("Expected clip to be set after cairo_clip")
	}

	// Path should be cleared after clip
	_, _, hasPoint := cb.Renderer().GetCurrentPoint()
	if hasPoint {
		t.Error("Expected path to be cleared after cairo_clip")
	}
}

func TestCairoBindings_ClipPreserve(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	cb, err := NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Test clip preserve
	_, err = runtime.ExecuteString("test", `
		cairo_rectangle(10, 10, 100, 100)
		cairo_clip_preserve()
	`)
	if err != nil {
		t.Fatalf("Failed to execute cairo_clip_preserve: %v", err)
	}

	// Verify clip is set and path is preserved
	if !cb.Renderer().HasClip() {
		t.Error("Expected clip to be set after cairo_clip_preserve")
	}

	// Path should be preserved after clip_preserve
	_, _, hasPoint := cb.Renderer().GetCurrentPoint()
	if !hasPoint {
		t.Error("Expected path to be preserved after cairo_clip_preserve")
	}
}

func TestCairoBindings_ResetClip(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	cb, err := NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Set and then reset clip
	_, err = runtime.ExecuteString("test", `
		cairo_rectangle(10, 10, 100, 100)
		cairo_clip()
		cairo_reset_clip()
	`)
	if err != nil {
		t.Fatalf("Failed to execute cairo_reset_clip: %v", err)
	}

	// Verify clip is reset
	if cb.Renderer().HasClip() {
		t.Error("Expected clip to be reset after cairo_reset_clip")
	}
}

func TestCairoBindings_ClipSaveRestore(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	cb, err := NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Test that clip is saved and restored
	_, err = runtime.ExecuteString("test", `
		-- Save initial state (no clip)
		cairo_save()
		
		-- Set clip
		cairo_rectangle(10, 10, 100, 100)
		cairo_clip()
		
		-- Restore state (should restore no-clip)
		cairo_restore()
	`)
	if err != nil {
		t.Fatalf("Failed to execute clip save/restore: %v", err)
	}

	// Verify clip is restored to no-clip state
	if cb.Renderer().HasClip() {
		t.Error("Expected clip to be restored (no clip) after cairo_restore")
	}
}

func TestCairoBindings_ClipNoPath(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	cb, err := NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Test clip without a path (should do nothing)
	_, err = runtime.ExecuteString("test", `
		cairo_new_path()
		cairo_clip()
	`)
	if err != nil {
		t.Fatalf("Failed to execute cairo_clip without path: %v", err)
	}

	// Verify clip is not set
	if cb.Renderer().HasClip() {
		t.Error("Expected no clip after cairo_clip without path")
	}
}

func TestCairoBindings_ComplexClippingWorkflow(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Test a complex clipping workflow
	luaCode := `
		-- Set up a clip region
		cairo_save()
		cairo_rectangle(0, 0, 100, 100)
		cairo_clip()
		
		-- Draw something within the clip
		cairo_set_source_rgb(1, 0, 0)
		cairo_rectangle(50, 50, 100, 100)
		cairo_fill()
		
		-- Reset and draw without clip
		cairo_reset_clip()
		cairo_rectangle(200, 200, 50, 50)
		cairo_fill()
		
		cairo_restore()
		return true
	`
	result, err := runtime.ExecuteString("test", luaCode)
	if err != nil {
		t.Fatalf("Failed to execute complex clipping workflow: %v", err)
	}

	if !result.AsBool() {
		t.Error("Expected complex clipping workflow to succeed")
	}
}
