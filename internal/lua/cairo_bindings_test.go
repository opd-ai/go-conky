// Package lua provides Golua integration for conky-go.
package lua

import (
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
