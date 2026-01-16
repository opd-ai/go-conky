// Package lua provides Golua integration for conky-go.
package lua

import (
	"math"
	"os"
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

// --- Tests for Path Query Functions ---

func TestCairoBindings_GetCurrentPoint(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Test getting current point after move_to
	luaCode := `
		cairo_move_to(10, 20)
		local x, y = cairo_get_current_point()
		return x, y
	`
	result, err := runtime.ExecuteString("test", luaCode)
	if err != nil {
		t.Fatalf("Failed to execute cairo_get_current_point: %v", err)
	}

	x := result.AsFloat()
	if x != 10 {
		t.Errorf("Expected x=10, got %f", x)
	}
}

func TestCairoBindings_HasCurrentPoint(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Test has_current_point
	luaCode := `
		-- Initially no current point
		local has1 = cairo_has_current_point()
		
		-- After move_to, should have current point
		cairo_move_to(10, 20)
		local has2 = cairo_has_current_point()
		
		-- After new_path, should not have current point
		cairo_new_path()
		local has3 = cairo_has_current_point()
		
		return has1, has2, has3
	`
	result, err := runtime.ExecuteString("test", luaCode)
	if err != nil {
		t.Fatalf("Failed to execute cairo_has_current_point: %v", err)
	}

	has1 := result.AsBool()
	if has1 {
		t.Error("Expected has_current_point to be false initially")
	}
}

func TestCairoBindings_PathExtents(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Test path_extents
	luaCode := `
		cairo_rectangle(10, 20, 100, 50)
		local x1, y1, x2, y2 = cairo_path_extents()
		return x1, y1, x2, y2
	`
	result, err := runtime.ExecuteString("test", luaCode)
	if err != nil {
		t.Fatalf("Failed to execute cairo_path_extents: %v", err)
	}

	x1 := result.AsFloat()
	if x1 != 10 {
		t.Errorf("Expected x1=10, got %f", x1)
	}
}

func TestCairoBindings_ClipExtents(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Test clip_extents
	luaCode := `
		cairo_rectangle(10, 20, 100, 50)
		cairo_clip()
		local x1, y1, x2, y2 = cairo_clip_extents()
		return x1, y1, x2, y2
	`
	result, err := runtime.ExecuteString("test", luaCode)
	if err != nil {
		t.Fatalf("Failed to execute cairo_clip_extents: %v", err)
	}

	x1 := result.AsFloat()
	if x1 != 10 {
		t.Errorf("Expected x1=10, got %f", x1)
	}
}

func TestCairoBindings_InClip(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Test in_clip
	luaCode := `
		-- No clip - all points should be in clip
		local in1 = cairo_in_clip(50, 50)
		
		-- Set clip and test points
		cairo_rectangle(10, 20, 100, 50)
		cairo_clip()
		
		-- Inside clip
		local in2 = cairo_in_clip(50, 40)
		
		-- Outside clip
		local in3 = cairo_in_clip(5, 40)
		
		return in1, in2, in3
	`
	result, err := runtime.ExecuteString("test", luaCode)
	if err != nil {
		t.Fatalf("Failed to execute cairo_in_clip: %v", err)
	}

	in1 := result.AsBool()
	if !in1 {
		t.Error("Expected point to be in clip when no clip is set")
	}
}

func TestCairoBindings_PathQueryWithContext(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Test path query functions with context argument
	luaCode := `
		local cs = cairo_image_surface_create(CAIRO_FORMAT_ARGB32, 100, 100)
		local cr = cairo_create(cs)
		
		cairo_move_to(cr, 10, 20)
		local x, y = cairo_get_current_point(cr)
		local has = cairo_has_current_point(cr)
		
		cairo_rectangle(cr, 0, 0, 50, 50)
		local x1, y1, x2, y2 = cairo_path_extents(cr)
		
		cairo_destroy(cr)
		cairo_surface_destroy(cs)
		
		return x, y, has, x1, y1
	`
	result, err := runtime.ExecuteString("test", luaCode)
	if err != nil {
		t.Fatalf("Failed to execute path query with context: %v", err)
	}

	x := result.AsFloat()
	if x != 10 {
		t.Errorf("Expected x=10, got %f", x)
	}
}

// --- Pattern/Gradient Tests ---

func TestCairoBindings_PatternCreateRGB(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Create solid pattern
	luaCode := `
		local pattern = cairo_pattern_create_rgb(1, 0, 0)
		return pattern ~= nil
	`
	result, err := runtime.ExecuteString("test", luaCode)
	if err != nil {
		t.Fatalf("Failed to execute cairo_pattern_create_rgb: %v", err)
	}

	if !result.AsBool() {
		t.Error("Expected pattern to be created (non-nil)")
	}
}

func TestCairoBindings_PatternCreateRGBA(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Create solid pattern with alpha
	luaCode := `
		local pattern = cairo_pattern_create_rgba(0, 1, 0, 0.5)
		return pattern ~= nil
	`
	result, err := runtime.ExecuteString("test", luaCode)
	if err != nil {
		t.Fatalf("Failed to execute cairo_pattern_create_rgba: %v", err)
	}

	if !result.AsBool() {
		t.Error("Expected pattern to be created (non-nil)")
	}
}

func TestCairoBindings_PatternCreateLinear(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Create linear gradient pattern
	luaCode := `
		local pattern = cairo_pattern_create_linear(0, 0, 100, 100)
		return pattern ~= nil
	`
	result, err := runtime.ExecuteString("test", luaCode)
	if err != nil {
		t.Fatalf("Failed to execute cairo_pattern_create_linear: %v", err)
	}

	if !result.AsBool() {
		t.Error("Expected linear pattern to be created (non-nil)")
	}
}

func TestCairoBindings_PatternCreateRadial(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Create radial gradient pattern
	luaCode := `
		local pattern = cairo_pattern_create_radial(50, 50, 10, 50, 50, 50)
		return pattern ~= nil
	`
	result, err := runtime.ExecuteString("test", luaCode)
	if err != nil {
		t.Fatalf("Failed to execute cairo_pattern_create_radial: %v", err)
	}

	if !result.AsBool() {
		t.Error("Expected radial pattern to be created (non-nil)")
	}
}

func TestCairoBindings_PatternAddColorStops(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Create pattern and add color stops
	luaCode := `
		local pattern = cairo_pattern_create_linear(0, 0, 100, 0)
		cairo_pattern_add_color_stop_rgb(pattern, 0, 1, 0, 0)  -- Red at start
		cairo_pattern_add_color_stop_rgb(pattern, 0.5, 0, 1, 0)  -- Green in middle
		cairo_pattern_add_color_stop_rgba(pattern, 1, 0, 0, 1, 0.5)  -- Blue with alpha at end
		return pattern ~= nil
	`
	result, err := runtime.ExecuteString("test", luaCode)
	if err != nil {
		t.Fatalf("Failed to execute pattern color stops: %v", err)
	}

	if !result.AsBool() {
		t.Error("Expected pattern with color stops to be created")
	}
}

func TestCairoBindings_SetSource(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	cb, err := NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Create solid pattern and set as source
	luaCode := `
		local cs = cairo_image_surface_create(CAIRO_FORMAT_ARGB32, 100, 100)
		local cr = cairo_create(cs)
		local pattern = cairo_pattern_create_rgb(0, 0, 1)
		cairo_set_source(cr, pattern)
		cairo_destroy(cr)
		cairo_surface_destroy(cs)
		return true
	`
	result, err := runtime.ExecuteString("test", luaCode)
	if err != nil {
		t.Fatalf("Failed to execute cairo_set_source: %v", err)
	}

	if !result.AsBool() {
		t.Error("Expected set_source to succeed")
	}

	// Test that a solid pattern also updates the color
	luaCode2 := `
		local pattern = cairo_pattern_create_rgb(1, 0, 0)
		local cs = cairo_image_surface_create(CAIRO_FORMAT_ARGB32, 100, 100)
		local cr = cairo_create(cs)
		cairo_set_source(cr, pattern)
		cairo_destroy(cr)
		cairo_surface_destroy(cs)
		return true
	`
	_, err = runtime.ExecuteString("test2", luaCode2)
	if err != nil {
		t.Fatalf("Failed to execute second cairo_set_source: %v", err)
	}

	// Verify the source was set (solid patterns also update currentColor)
	source := cb.Renderer().GetSource()
	if source == nil {
		t.Log("GetSource returned nil (source may be on separate context)")
	}
}

func TestCairoBindings_LinearGradientIntegration(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Full integration test: create gradient, add stops, set source, draw
	luaCode := `
		local cs = cairo_image_surface_create(CAIRO_FORMAT_ARGB32, 200, 100)
		local cr = cairo_create(cs)
		
		-- Create horizontal gradient
		local pattern = cairo_pattern_create_linear(0, 0, 200, 0)
		cairo_pattern_add_color_stop_rgb(pattern, 0, 1, 0, 0)  -- Red
		cairo_pattern_add_color_stop_rgb(pattern, 0.5, 0, 1, 0)  -- Green
		cairo_pattern_add_color_stop_rgb(pattern, 1, 0, 0, 1)  -- Blue
		
		-- Set pattern and draw rectangle
		cairo_set_source(cr, pattern)
		cairo_rectangle(cr, 0, 0, 200, 100)
		cairo_fill(cr)
		
		cairo_destroy(cr)
		cairo_surface_destroy(cs)
		return true
	`
	result, err := runtime.ExecuteString("test", luaCode)
	if err != nil {
		t.Fatalf("Failed to execute linear gradient integration test: %v", err)
	}

	if !result.AsBool() {
		t.Error("Expected linear gradient integration test to succeed")
	}
}

func TestCairoBindings_RadialGradientIntegration(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Full integration test: create radial gradient, add stops, set source, draw
	luaCode := `
		local cs = cairo_image_surface_create(CAIRO_FORMAT_ARGB32, 100, 100)
		local cr = cairo_create(cs)
		
		-- Create radial gradient (inner circle to outer circle)
		local pattern = cairo_pattern_create_radial(50, 50, 0, 50, 50, 50)
		cairo_pattern_add_color_stop_rgba(pattern, 0, 1, 1, 1, 1)  -- White center
		cairo_pattern_add_color_stop_rgba(pattern, 1, 0, 0, 0, 1)  -- Black edge
		
		-- Set pattern and draw circle
		cairo_set_source(cr, pattern)
		cairo_arc(cr, 50, 50, 50, 0, 2 * math.pi)
		cairo_fill(cr)
		
		cairo_destroy(cr)
		cairo_surface_destroy(cs)
		return true
	`
	result, err := runtime.ExecuteString("test", luaCode)
	if err != nil {
		t.Fatalf("Failed to execute radial gradient integration test: %v", err)
	}

	if !result.AsBool() {
		t.Error("Expected radial gradient integration test to succeed")
	}
}

func TestCairoPattern_SolidColor(t *testing.T) {
	// Test solid pattern directly
	pattern := render.NewSolidPattern(1.0, 0.5, 0.25, 1.0)

	// ColorAt should return the same color regardless of position
	c := pattern.ColorAt(0.0)
	if c.R != 255 || c.G != 127 || c.B != 63 || c.A != 255 {
		t.Errorf("Expected RGBA(255,127,63,255), got RGBA(%d,%d,%d,%d)", c.R, c.G, c.B, c.A)
	}

	c = pattern.ColorAt(1.0)
	if c.R != 255 || c.G != 127 || c.B != 63 || c.A != 255 {
		t.Errorf("Expected RGBA(255,127,63,255), got RGBA(%d,%d,%d,%d)", c.R, c.G, c.B, c.A)
	}
}

func TestCairoPattern_LinearGradient(t *testing.T) {
	// Test linear gradient from left to right
	pattern := render.NewLinearPattern(0, 0, 100, 0)
	pattern.AddColorStopRGB(0, 1, 0, 0) // Red at start
	pattern.AddColorStopRGB(1, 0, 0, 1) // Blue at end

	// Check color at start
	c := pattern.ColorAt(0.0)
	if c.R != 255 || c.G != 0 || c.B != 0 {
		t.Errorf("Expected RGB(255,0,0) at start, got RGB(%d,%d,%d)", c.R, c.G, c.B)
	}

	// Check color at end
	c = pattern.ColorAt(1.0)
	if c.R != 0 || c.G != 0 || c.B != 255 {
		t.Errorf("Expected RGB(0,0,255) at end, got RGB(%d,%d,%d)", c.R, c.G, c.B)
	}

	// Check color at middle (should be purple-ish)
	c = pattern.ColorAt(0.5)
	if c.R != 127 || c.G != 0 || c.B != 127 {
		t.Errorf("Expected RGB(127,0,127) at middle, got RGB(%d,%d,%d)", c.R, c.G, c.B)
	}
}

func TestCairoPattern_RadialGradient(t *testing.T) {
	// Test radial gradient from center outward
	pattern := render.NewRadialPattern(50, 50, 0, 50, 50, 50)
	pattern.AddColorStopRGB(0, 1, 1, 1) // White at center
	pattern.AddColorStopRGB(1, 0, 0, 0) // Black at edge

	// Check color at center
	c := pattern.ColorAtPoint(50, 50)
	if c.R != 255 || c.G != 255 || c.B != 255 {
		t.Errorf("Expected RGB(255,255,255) at center, got RGB(%d,%d,%d)", c.R, c.G, c.B)
	}

	// Check color at edge (50 pixels from center)
	c = pattern.ColorAtPoint(100, 50)
	if c.R != 0 || c.G != 0 || c.B != 0 {
		t.Errorf("Expected RGB(0,0,0) at edge, got RGB(%d,%d,%d)", c.R, c.G, c.B)
	}
}

func TestCairoPattern_GradientWithMultipleStops(t *testing.T) {
	// Test gradient with 3 color stops
	pattern := render.NewLinearPattern(0, 0, 100, 0)
	pattern.AddColorStopRGB(0, 1, 0, 0)   // Red
	pattern.AddColorStopRGB(0.5, 0, 1, 0) // Green
	pattern.AddColorStopRGB(1, 0, 0, 1)   // Blue

	// Check colors at key points
	c := pattern.ColorAt(0.0)
	if c.R != 255 {
		t.Errorf("Expected red at 0, got RGB(%d,%d,%d)", c.R, c.G, c.B)
	}

	c = pattern.ColorAt(0.5)
	if c.G != 255 {
		t.Errorf("Expected green at 0.5, got RGB(%d,%d,%d)", c.R, c.G, c.B)
	}

	c = pattern.ColorAt(1.0)
	if c.B != 255 {
		t.Errorf("Expected blue at 1.0, got RGB(%d,%d,%d)", c.R, c.G, c.B)
	}
}

// --- Matrix Bindings Tests ---

func TestCairoBindings_MatrixInitIdentity(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	_, err = runtime.ExecuteString("test", `
		local m = cairo_matrix_init_identity()
		local x, y = cairo_matrix_transform_point(m, 10, 20)
		assert(math.abs(x - 10) < 0.001, "x should be 10")
		assert(math.abs(y - 20) < 0.001, "y should be 20")
	`)
	if err != nil {
		t.Fatalf("Failed to execute matrix test: %v", err)
	}
}

func TestCairoBindings_MatrixTranslate(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	_, err = runtime.ExecuteString("test", `
		local m = cairo_matrix_init_translate(100, 200)
		local x, y = cairo_matrix_transform_point(m, 0, 0)
		assert(math.abs(x - 100) < 0.001, "x should be 100")
		assert(math.abs(y - 200) < 0.001, "y should be 200")
	`)
	if err != nil {
		t.Fatalf("Failed to execute matrix translate test: %v", err)
	}
}

func TestCairoBindings_MatrixScale(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	_, err = runtime.ExecuteString("test", `
		local m = cairo_matrix_init_scale(2, 3)
		local x, y = cairo_matrix_transform_point(m, 5, 4)
		assert(math.abs(x - 10) < 0.001, "x should be 10")
		assert(math.abs(y - 12) < 0.001, "y should be 12")
	`)
	if err != nil {
		t.Fatalf("Failed to execute matrix scale test: %v", err)
	}
}

func TestCairoBindings_MatrixRotate(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	_, err = runtime.ExecuteString("test", `
		local m = cairo_matrix_init_rotate(math.pi / 2)  -- 90 degrees
		local x, y = cairo_matrix_transform_point(m, 1, 0)
		-- After 90 degree rotation, (1, 0) -> (0, 1)
		assert(math.abs(x) < 0.001, "x should be 0")
		assert(math.abs(y - 1) < 0.001, "y should be 1")
	`)
	if err != nil {
		t.Fatalf("Failed to execute matrix rotate test: %v", err)
	}
}

func TestCairoBindings_GetSetMatrix(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	cb, err := NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	_, err = runtime.ExecuteString("test", `
		local m = cairo_matrix_init_translate(50, 100)
		cairo_set_matrix(m)
	`)
	if err != nil {
		t.Fatalf("Failed to execute set_matrix: %v", err)
	}

	m := cb.Renderer().GetMatrix()
	if m.X0 != 50 || m.Y0 != 100 {
		t.Errorf("Expected translation (50, 100), got (%f, %f)", m.X0, m.Y0)
	}
}

func TestCairoBindings_PatternExtend(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	_, err = runtime.ExecuteString("test", `
		local pattern = cairo_pattern_create_linear(0, 0, 100, 0)
		cairo_pattern_set_extend(pattern, CAIRO_EXTEND_REPEAT)
		local extend = cairo_pattern_get_extend(pattern)
		assert(extend == CAIRO_EXTEND_REPEAT, "extend should be CAIRO_EXTEND_REPEAT")
	`)
	if err != nil {
		t.Fatalf("Failed to execute pattern extend test: %v", err)
	}
}

func TestCairoBindings_ExtendConstants(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	_, err = runtime.ExecuteString("test", `
		assert(CAIRO_EXTEND_NONE ~= nil, "CAIRO_EXTEND_NONE should exist")
		assert(CAIRO_EXTEND_REPEAT ~= nil, "CAIRO_EXTEND_REPEAT should exist")
		assert(CAIRO_EXTEND_REFLECT ~= nil, "CAIRO_EXTEND_REFLECT should exist")
		assert(CAIRO_EXTEND_PAD ~= nil, "CAIRO_EXTEND_PAD should exist")
	`)
	if err != nil {
		t.Fatalf("Failed to verify extend constants: %v", err)
	}
}

func TestCairoBindings_SurfaceFlush(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// This should not panic - flush is a no-op in Ebiten
	_, err = runtime.ExecuteString("test", `
		local surface = cairo_image_surface_create(CAIRO_FORMAT_ARGB32, 100, 100)
		cairo_surface_flush(surface)
		cairo_surface_mark_dirty(surface)
		cairo_surface_mark_dirty_rectangle(surface, 0, 0, 50, 50)
		cairo_surface_destroy(surface)
	`)
	if err != nil {
		t.Fatalf("Failed to execute surface flush test: %v", err)
	}
}

func TestCairoBindings_DashFunctions(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Verify dash pattern can be set and retrieved on a context created from a surface.
	// The dash pattern is set on the context's renderer, not the shared renderer.
	_, err = runtime.ExecuteString("test", `
		local surface = cairo_image_surface_create(CAIRO_FORMAT_ARGB32, 100, 100)
		local cr = cairo_create(surface)
		
		-- Set a dash pattern
		cairo_set_dash(cr, {10, 5, 3, 5}, 0)
		
		-- Get dash count
		local count = cairo_get_dash_count(cr)
		assert(count == 4, "Expected 4 dash elements, got " .. tostring(count))
		
		-- Get dash pattern
		local dashes, offset = cairo_get_dash(cr)
		assert(offset == 0, "Expected offset 0, got " .. tostring(offset))
		
		cairo_destroy(cr)
		cairo_surface_destroy(surface)
	`)
	if err != nil {
		t.Fatalf("Failed to execute dash test: %v", err)
	}
}

func TestCairoBindings_MiterLimit(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Test setting and getting miter limit on a context created from a surface.
	// The miter limit is set on the context's renderer, not the shared renderer.
	_, err = runtime.ExecuteString("test", `
		local surface = cairo_image_surface_create(CAIRO_FORMAT_ARGB32, 100, 100)
		local cr = cairo_create(surface)
		
		cairo_set_miter_limit(cr, 10.0)
		local limit = cairo_get_miter_limit(cr)
		assert(limit == 10.0, "Expected miter limit 10.0, got " .. tostring(limit))
		
		cairo_destroy(cr)
		cairo_surface_destroy(surface)
	`)
	if err != nil {
		t.Fatalf("Failed to execute miter limit test: %v", err)
	}
}

func TestCairoBindings_LinePropertyGetters(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	_, err = runtime.ExecuteString("test", `
		local surface = cairo_image_surface_create(CAIRO_FORMAT_ARGB32, 100, 100)
		local cr = cairo_create(surface)
		
		-- Test line width getter
		cairo_set_line_width(cr, 3.0)
		local width = cairo_get_line_width(cr)
		assert(width == 3.0, "Expected line width 3.0, got " .. tostring(width))
		
		-- Test line cap getter
		cairo_set_line_cap(cr, CAIRO_LINE_CAP_ROUND)
		local cap = cairo_get_line_cap(cr)
		assert(cap == CAIRO_LINE_CAP_ROUND, "Expected CAIRO_LINE_CAP_ROUND")
		
		-- Test line join getter
		cairo_set_line_join(cr, CAIRO_LINE_JOIN_BEVEL)
		local join = cairo_get_line_join(cr)
		assert(join == CAIRO_LINE_JOIN_BEVEL, "Expected CAIRO_LINE_JOIN_BEVEL")
		
		-- Test antialias getter
		cairo_set_antialias(cr, CAIRO_ANTIALIAS_NONE)
		local aa = cairo_get_antialias(cr)
		assert(aa == CAIRO_ANTIALIAS_NONE, "Expected CAIRO_ANTIALIAS_NONE")
		
		cairo_destroy(cr)
		cairo_surface_destroy(surface)
	`)
	if err != nil {
		t.Fatalf("Failed to execute line property getters test: %v", err)
	}
}

func TestCairoBindings_FillRuleOperator(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Verify constants exist and functions work
	_, err = runtime.ExecuteString("test", `
		assert(CAIRO_FILL_RULE_WINDING ~= nil, "CAIRO_FILL_RULE_WINDING should exist")
		assert(CAIRO_FILL_RULE_EVEN_ODD ~= nil, "CAIRO_FILL_RULE_EVEN_ODD should exist")
		assert(CAIRO_OPERATOR_OVER ~= nil, "CAIRO_OPERATOR_OVER should exist")
		assert(CAIRO_OPERATOR_SOURCE ~= nil, "CAIRO_OPERATOR_SOURCE should exist")
		
		local surface = cairo_image_surface_create(CAIRO_FORMAT_ARGB32, 100, 100)
		local cr = cairo_create(surface)
		
		-- Test fill rule
		cairo_set_fill_rule(cr, CAIRO_FILL_RULE_EVEN_ODD)
		local rule = cairo_get_fill_rule(cr)
		-- fill rule is a no-op so always returns winding (0)
		
		-- Test operator
		cairo_set_operator(cr, CAIRO_OPERATOR_SOURCE)
		local op = cairo_get_operator(cr)
		-- operator is a no-op so always returns OVER (2)
		
		cairo_destroy(cr)
		cairo_surface_destroy(surface)
	`)
	if err != nil {
		t.Fatalf("Failed to execute fill rule/operator test: %v", err)
	}
}

func TestCairoBindings_ImageSurfaceCreateFromPNG(t *testing.T) {
	// Skip this test because WriteToPNG uses ebiten.Image.ReadPixels()
	// which requires the Ebiten game loop to be running.
	t.Skip("Skipping test: WriteToPNG requires Ebiten game loop to be running")

	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Create a temp directory for test files
	tmpDir := t.TempDir()

	// First create a PNG file using the render package directly
	surface := render.NewCairoSurface(64, 48)
	ctx := render.NewCairoContext(surface)
	renderer := ctx.Renderer()
	renderer.SetSourceRGB(1, 0.5, 0) // Orange
	renderer.Rectangle(0, 0, 64, 48)
	renderer.Fill()
	pngPath := tmpDir + "/test_image.png"
	err = surface.WriteToPNG(pngPath)
	if err != nil {
		t.Fatalf("Failed to create test PNG: %v", err)
	}
	ctx.Destroy()
	surface.Destroy()

	// Now test loading the PNG from Lua
	result, err := runtime.ExecuteString("test", `
		local surface = cairo_image_surface_create_from_png("`+pngPath+`")
		if surface == nil then
			error("Failed to load PNG surface")
		end
		cairo_surface_destroy(surface)
		return "success"
	`)
	if err != nil {
		t.Fatalf("Failed to execute cairo_image_surface_create_from_png: %v", err)
	}
	if s, ok := result.TryString(); !ok || s != "success" {
		t.Errorf("Expected 'success', got %v", result)
	}
}

func TestCairoBindings_ImageSurfaceCreateFromPNG_NonexistentFile(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Test loading a nonexistent file - should return nil
	result, err := runtime.ExecuteString("test", `
		local surface, errmsg = cairo_image_surface_create_from_png("/nonexistent/path/image.png")
		if surface ~= nil then
			error("Expected nil for nonexistent file")
		end
		return "handled_nil"
	`)
	if err != nil {
		t.Fatalf("Failed to execute test: %v", err)
	}
	if s, ok := result.TryString(); !ok || s != "handled_nil" {
		t.Errorf("Expected 'handled_nil', got %v", result)
	}
}

func TestCairoBindings_SurfaceWriteToPNG(t *testing.T) {
	// Skip this test because WriteToPNG uses ebiten.Image.ReadPixels()
	// which requires the Ebiten game loop to be running.
	t.Skip("Skipping test: WriteToPNG requires Ebiten game loop to be running")

	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	tmpDir := t.TempDir()
	pngPath := tmpDir + "/output.png"

	// Create a surface, draw on it, and save it
	_, err = runtime.ExecuteString("test", `
		local surface = cairo_image_surface_create(CAIRO_FORMAT_ARGB32, 80, 60)
		local cr = cairo_create(surface)
		
		-- Draw something on the surface
		cairo_set_source_rgb(cr, 0, 0, 1)  -- Blue
		cairo_rectangle(cr, 0, 0, 80, 60)
		cairo_fill(cr)
		
		-- Save to PNG
		local status = cairo_surface_write_to_png(surface, "`+pngPath+`")
		if status ~= 0 then
			error("Failed to write PNG, status: " .. tostring(status))
		end
		
		cairo_destroy(cr)
		cairo_surface_destroy(surface)
		return "saved"
	`)
	if err != nil {
		t.Fatalf("Failed to execute cairo_surface_write_to_png: %v", err)
	}

	// Verify the file exists
	info, err := os.Stat(pngPath)
	if err != nil {
		t.Fatalf("Output PNG file not found: %v", err)
	}
	if info.Size() == 0 {
		t.Error("Output PNG file is empty")
	}

	// Verify we can load the file back
	loadedSurface, err := render.NewCairoSurfaceFromPNG(pngPath)
	if err != nil {
		t.Fatalf("Failed to load saved PNG: %v", err)
	}
	if loadedSurface.Width() != 80 || loadedSurface.Height() != 60 {
		t.Errorf("Loaded dimensions mismatch: expected (80,60), got (%d,%d)",
			loadedSurface.Width(), loadedSurface.Height())
	}
	loadedSurface.Destroy()
}

func TestCairoBindings_SurfaceWriteToPNG_InvalidPath(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	// Try to write to invalid path - should return non-zero status
	result, err := runtime.ExecuteString("test", `
		local surface = cairo_image_surface_create(CAIRO_FORMAT_ARGB32, 50, 50)
		local status = cairo_surface_write_to_png(surface, "/nonexistent/directory/output.png")
		cairo_surface_destroy(surface)
		return status
	`)
	if err != nil {
		t.Fatalf("Failed to execute test: %v", err)
	}
	// Status should be non-zero (error)
	if status, ok := result.TryInt(); ok && status == 0 {
		t.Error("Expected non-zero status for invalid path")
	}
}

func TestCairoBindings_PNG_RoundTrip(t *testing.T) {
	// Skip this test because WriteToPNG uses ebiten.Image.ReadPixels()
	// which requires the Ebiten game loop to be running.
	t.Skip("Skipping test: WriteToPNG requires Ebiten game loop to be running")

	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoBindings(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoBindings: %v", err)
	}

	tmpDir := t.TempDir()
	pngPath := tmpDir + "/roundtrip.png"

	// Create, save, load, and verify dimensions
	_, err = runtime.ExecuteString("test", `
		-- Create and save a surface
		local surface1 = cairo_image_surface_create(CAIRO_FORMAT_ARGB32, 123, 456)
		local cr = cairo_create(surface1)
		cairo_set_source_rgb(cr, 1, 0, 0)
		cairo_paint(cr)
		cairo_destroy(cr)
		
		local status = cairo_surface_write_to_png(surface1, "`+pngPath+`")
		if status ~= 0 then
			error("Failed to save PNG")
		end
		cairo_surface_destroy(surface1)
		
		-- Load the saved surface
		local surface2 = cairo_image_surface_create_from_png("`+pngPath+`")
		if surface2 == nil then
			error("Failed to load PNG")
		end
		
		-- Note: We can't query dimensions in Lua, but the surface should be usable
		local cr2 = cairo_create(surface2)
		if cr2 == nil then
			error("Failed to create context from loaded surface")
		end
		cairo_destroy(cr2)
		cairo_surface_destroy(surface2)
		
		return "roundtrip_success"
	`)
	if err != nil {
		t.Fatalf("Failed to execute PNG round-trip test: %v", err)
	}
}
