// Package lua provides Golua integration for conky-go.
package lua

import (
	"testing"

	rt "github.com/arnodel/golua/runtime"

	"github.com/opd-ai/go-conky/internal/render"
)

func TestNewCairoModule(t *testing.T) {
	// Test with nil runtime
	_, err := NewCairoModule(nil)
	if err == nil {
		t.Error("Expected error for nil runtime, got nil")
	}
	if err != ErrNilRuntime {
		t.Errorf("Expected ErrNilRuntime, got %v", err)
	}

	// Test with valid runtime
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	cm, err := NewCairoModule(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoModule: %v", err)
	}

	if cm == nil {
		t.Fatal("NewCairoModule returned nil")
	}

	if cm.Renderer() == nil {
		t.Error("Renderer() returned nil")
	}
}

func TestCairoModule_WithCairoRenderer(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	// Create a shared renderer
	sharedRenderer := render.NewCairoRenderer()
	sharedRenderer.SetLineWidth(5.0) // Set a distinctive value

	// Create CairoModule with the shared renderer
	cm, err := NewCairoModule(runtime, WithCairoRenderer(sharedRenderer))
	if err != nil {
		t.Fatalf("Failed to create CairoModule: %v", err)
	}

	// Verify the shared renderer is used
	if cm.Renderer() != sharedRenderer {
		t.Error("Expected CairoModule to use the shared renderer")
	}

	// Verify the renderer state is shared
	if cm.Renderer().GetLineWidth() != 5.0 {
		t.Errorf("Expected line width 5.0, got %f", cm.Renderer().GetLineWidth())
	}
}

func TestCairoModule_WithNilRenderer(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	// Create CairoModule with nil renderer option (should create new renderer)
	cm, err := NewCairoModule(runtime, WithCairoRenderer(nil))
	if err != nil {
		t.Fatalf("Failed to create CairoModule: %v", err)
	}

	// Should still have a renderer (fallback to creating new one)
	if cm.Renderer() == nil {
		t.Error("Expected CairoModule to have a renderer even with nil option")
	}
}

func TestCairoModule_RequireCairo(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoModule(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoModule: %v", err)
	}

	// Test that cairo is available as a global
	result, err := runtime.ExecuteString("test", `
		return type(cairo)
	`)
	if err != nil {
		t.Fatalf("Failed to access cairo global: %v", err)
	}
	if result.AsString() != "table" {
		t.Errorf("Expected cairo global to be a table, got %s", result.AsString())
	}

	// NOTE: require('cairo') is registered in package.loaded but may fail
	// in resource-limited contexts because Golua's require function
	// is not marked as CPU/memory safe. Scripts should use the global
	// cairo table directly instead: cairo.set_source_rgb(...)
	// This matches the common Conky pattern where cairo is available globally.
}

func TestCairoModule_ModuleFunctions(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	cm, err := NewCairoModule(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoModule: %v", err)
	}

	// Test using cairo functions through the global table
	_, err = runtime.ExecuteString("test", `
		cairo.set_source_rgb(1, 0, 0)
		cairo.set_line_width(2.5)
	`)
	if err != nil {
		t.Fatalf("Failed to execute cairo functions: %v", err)
	}

	// Verify the color was set
	color := cm.Renderer().GetCurrentColor()
	if color.R != 255 || color.G != 0 || color.B != 0 {
		t.Errorf("Expected RGB(255,0,0), got RGB(%d,%d,%d)", color.R, color.G, color.B)
	}

	// Verify the line width was set
	width := cm.Renderer().GetLineWidth()
	if width != 2.5 {
		t.Errorf("Expected line width 2.5, got %f", width)
	}
}

func TestCairoModule_ModuleConstants(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	cm, err := NewCairoModule(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoModule: %v", err)
	}

	// Test accessing constants through the global table
	_, err = runtime.ExecuteString("test", `
		cairo.set_line_cap(cairo.LINE_CAP_ROUND)
	`)
	if err != nil {
		t.Fatalf("Failed to execute: %v", err)
	}

	// Verify the line cap was set
	capStyle := cm.Renderer().GetLineCap()
	if capStyle != render.LineCapRound {
		t.Errorf("Expected LineCapRound, got %v", capStyle)
	}
}

func TestCairoModule_ConkyWindowNil(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoModule(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoModule: %v", err)
	}

	// Initially conky_window should be nil
	result, err := runtime.ExecuteString("test", `
		return conky_window == nil
	`)
	if err != nil {
		t.Fatalf("Failed to execute: %v", err)
	}
	if result != rt.BoolValue(true) {
		t.Error("Expected conky_window to be nil initially")
	}
}

func TestCairoModule_UpdateWindowInfo(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	cm, err := NewCairoModule(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoModule: %v", err)
	}

	// Update window info
	cm.UpdateWindowInfo(800, 600, 1, 2, 3)

	// Check that conky_window is no longer nil
	result, err := runtime.ExecuteString("test", `
		return conky_window ~= nil
	`)
	if err != nil {
		t.Fatalf("Failed to execute: %v", err)
	}
	if result != rt.BoolValue(true) {
		t.Error("Expected conky_window to be set after UpdateWindowInfo")
	}

	// Check width
	result, err = runtime.ExecuteString("test", `
		return conky_window.width
	`)
	if err != nil {
		t.Fatalf("Failed to execute: %v", err)
	}
	if val, ok := result.TryInt(); !ok || val != 800 {
		t.Errorf("Expected width=800, got %v", result)
	}

	// Check height
	result, err = runtime.ExecuteString("test", `
		return conky_window.height
	`)
	if err != nil {
		t.Fatalf("Failed to execute: %v", err)
	}
	if val, ok := result.TryInt(); !ok || val != 600 {
		t.Errorf("Expected height=600, got %v", result)
	}
}

func TestCairoModule_ConkyWindowCheckPattern(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	cm, err := NewCairoModule(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoModule: %v", err)
	}

	// Test the common Conky pattern: if conky_window == nil then return end
	// Before UpdateWindowInfo, this should return early
	result, err := runtime.ExecuteString("test", `
		local function draw()
			if conky_window == nil then
				return "no_window"
			end
			return "has_window"
		end
		return draw()
	`)
	if err != nil {
		t.Fatalf("Failed to execute: %v", err)
	}
	if result.AsString() != "no_window" {
		t.Errorf("Expected 'no_window' before UpdateWindowInfo, got %s", result.AsString())
	}

	// After UpdateWindowInfo, the pattern should continue
	cm.UpdateWindowInfo(640, 480, 0, 0, 0)

	result, err = runtime.ExecuteString("test", `
		local function draw()
			if conky_window == nil then
				return "no_window"
			end
			return "has_window"
		end
		return draw()
	`)
	if err != nil {
		t.Fatalf("Failed to execute: %v", err)
	}
	if result.AsString() != "has_window" {
		t.Errorf("Expected 'has_window' after UpdateWindowInfo, got %s", result.AsString())
	}
}

func TestCairoModule_DrawingFunctions(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	cm, err := NewCairoModule(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoModule: %v", err)
	}

	// Test all drawing functions through the global cairo table
	tests := []struct {
		name string
		code string
	}{
		{"set_source_rgba", "cairo.set_source_rgba(1, 0, 0, 0.5)"},
		{"set_line_cap", "cairo.set_line_cap(cairo.LINE_CAP_BUTT)"},
		{"set_line_join", "cairo.set_line_join(cairo.LINE_JOIN_BEVEL)"},
		{"set_antialias", "cairo.set_antialias(cairo.ANTIALIAS_DEFAULT)"},
		{"new_path", "cairo.new_path()"},
		{"move_to", "cairo.move_to(10, 20)"},
		{"line_to", "cairo.line_to(100, 200)"},
		{"close_path", "cairo.close_path()"},
		{"rectangle", "cairo.rectangle(0, 0, 100, 100)"},
		{"arc", "cairo.arc(50, 50, 25, 0, math.pi)"},
		{"arc_negative", "cairo.arc_negative(50, 50, 25, math.pi, 0)"},
		{"curve_to", "cairo.curve_to(10, 20, 30, 40, 50, 60)"},
		{"stroke", "cairo.new_path(); cairo.move_to(0,0); cairo.line_to(10,10); cairo.stroke()"},
		{"fill", "cairo.new_path(); cairo.rectangle(0,0,10,10); cairo.fill()"},
		{"stroke_preserve", "cairo.new_path(); cairo.rectangle(0,0,10,10); cairo.stroke_preserve()"},
		{"fill_preserve", "cairo.new_path(); cairo.rectangle(0,0,10,10); cairo.fill_preserve()"},
		{"paint", "cairo.paint()"},
		{"paint_with_alpha", "cairo.paint_with_alpha(0.5)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := runtime.ExecuteString("test", tt.code)
			if err != nil {
				t.Errorf("Failed to execute %s: %v", tt.name, err)
			}
		})
	}

	// Verify drawing state after the tests
	capStyle := cm.Renderer().GetLineCap()
	if capStyle != render.LineCapButt {
		t.Errorf("Expected line cap to be LineCapButt, got %v", capStyle)
	}
}

func TestCairoModule_ComplexScript(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	cm, err := NewCairoModule(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoModule: %v", err)
	}

	// Update window info to enable drawing
	cm.UpdateWindowInfo(800, 600, 0, 0, 0)

	// Test a complex script similar to what users would write
	// Using both 'require' pattern and direct access patterns
	_, err = runtime.ExecuteString("test", `
		-- This is the standard Conky pattern
		if conky_window == nil then
			return
		end
		
		-- Set up drawing using global cairo table
		cairo.set_source_rgba(0.5, 0.7, 0.9, 1.0)
		cairo.set_line_width(2)
		cairo.set_line_cap(cairo.LINE_CAP_ROUND)
		cairo.set_line_join(cairo.LINE_JOIN_ROUND)
		
		-- Draw a rectangle
		cairo.new_path()
		cairo.rectangle(10, 10, conky_window.width - 20, conky_window.height - 20)
		cairo.stroke_preserve()
		
		-- Fill with different color
		cairo.set_source_rgba(0.1, 0.2, 0.3, 0.5)
		cairo.fill()
		
		-- Draw an arc
		cairo.new_path()
		cairo.set_source_rgb(1, 1, 1)
		local cx = conky_window.width / 2
		local cy = conky_window.height / 2
		cairo.arc(cx, cy, 50, 0, math.pi * 2)
		cairo.stroke()
	`)
	if err != nil {
		t.Fatalf("Failed to execute complex script: %v", err)
	}
}

func TestCairoModule_ErrorHandling(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = NewCairoModule(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoModule: %v", err)
	}

	// Test error handling for invalid arguments
	tests := []struct {
		name string
		code string
	}{
		{"set_source_rgb wrong type", "cairo.set_source_rgb('red', 0, 0)"},
		{"set_line_width wrong type", "cairo.set_line_width('thick')"},
		{"set_line_cap invalid value", "cairo.set_line_cap(99)"},
		{"set_line_join invalid value", "cairo.set_line_join(-1)"},
		{"move_to wrong type", "cairo.move_to('x', 10)"},
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

func TestCairoModule_FallbackToGlobals(t *testing.T) {
	// Test that functions are also registered as globals for backward compatibility
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	cm, err := NewCairoModule(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoModule: %v", err)
	}

	// The module should also register global functions for backward compatibility
	// This test uses the global cairo_* functions directly
	_, err = runtime.ExecuteString("test", `
		cairo_set_source_rgb(0, 1, 0)
		cairo_set_line_width(3.0)
	`)
	if err != nil {
		t.Fatalf("Failed to use global cairo functions: %v", err)
	}

	// Verify the color was set
	color := cm.Renderer().GetCurrentColor()
	if color.G != 255 {
		t.Errorf("Expected G=255, got %d", color.G)
	}

	// Verify the line width was set
	width := cm.Renderer().GetLineWidth()
	if width != 3.0 {
		t.Errorf("Expected line width 3.0, got %f", width)
	}
}

func TestCairoModule_UpdateWindowInfoAllFields(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	cm, err := NewCairoModule(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoModule: %v", err)
	}

	// Use UpdateWindowInfo which sets defaults for text region
	cm.UpdateWindowInfo(800, 600, 1, 2, 3)

	// Test all fields with a table-driven approach
	tests := []struct {
		field    string
		expected int64
	}{
		{"width", 800},
		{"height", 600},
		{"display", 1},
		{"drawable", 2},
		{"visual", 3},
		{"text_start_x", 10},       // default margin
		{"text_start_y", 20},       // default margin
		{"text_width", 780},        // 800 - 20
		{"text_height", 570},       // 600 - 30
		{"border_inner_margin", 0}, // default
		{"border_outer_margin", 0}, // default
		{"border_width", 0},        // default
	}

	for _, tc := range tests {
		t.Run(tc.field, func(t *testing.T) {
			result, err := runtime.ExecuteString("test", `return conky_window.`+tc.field)
			if err != nil {
				t.Fatalf("Failed to execute: %v", err)
			}
			if val, ok := result.TryInt(); !ok || val != tc.expected {
				t.Errorf("Expected %s=%d, got %v", tc.field, tc.expected, result)
			}
		})
	}
}

func TestCairoModule_UpdateWindowInfoFull(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	cm, err := NewCairoModule(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoModule: %v", err)
	}

	// Use UpdateWindowInfoFull with custom values
	info := WindowInfo{
		Width:             1024,
		Height:            768,
		Display:           100,
		Drawable:          200,
		Visual:            300,
		BorderInnerMargin: 5,
		BorderOuterMargin: 10,
		BorderWidth:       2,
		TextStartX:        15,
		TextStartY:        25,
		TextWidth:         994,
		TextHeight:        718,
	}
	cm.UpdateWindowInfoFull(info)

	// Test all custom fields
	tests := []struct {
		field    string
		expected int64
	}{
		{"width", 1024},
		{"height", 768},
		{"display", 100},
		{"drawable", 200},
		{"visual", 300},
		{"border_inner_margin", 5},
		{"border_outer_margin", 10},
		{"border_width", 2},
		{"text_start_x", 15},
		{"text_start_y", 25},
		{"text_width", 994},
		{"text_height", 718},
	}

	for _, tc := range tests {
		t.Run(tc.field, func(t *testing.T) {
			result, err := runtime.ExecuteString("test", `return conky_window.`+tc.field)
			if err != nil {
				t.Fatalf("Failed to execute: %v", err)
			}
			if val, ok := result.TryInt(); !ok || val != tc.expected {
				t.Errorf("Expected %s=%d, got %v", tc.field, tc.expected, result)
			}
		})
	}
}

func TestCairoModule_WindowInfoInLuaScript(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	cm, err := NewCairoModule(runtime)
	if err != nil {
		t.Fatalf("Failed to create CairoModule: %v", err)
	}

	cm.UpdateWindowInfo(640, 480, 0, 0, 0)

	// Test a realistic Lua script that uses the text region fields
	result, err := runtime.ExecuteString("test", `
		if conky_window == nil then
			return "no_window"
		end
		
		-- Calculate center of text region (common pattern in Conky scripts)
		local center_x = conky_window.text_start_x + (conky_window.text_width / 2)
		local center_y = conky_window.text_start_y + (conky_window.text_height / 2)
		
		-- Return the calculated center
		return center_x .. "," .. center_y
	`)
	if err != nil {
		t.Fatalf("Failed to execute: %v", err)
	}

	// Expected: text_start_x=10, text_width=620, so center_x = 10 + 310 = 320
	// text_start_y=20, text_height=450, so center_y = 20 + 225 = 245
	expected := "320,245"
	if result.AsString() != expected {
		t.Errorf("Expected center=%s, got %s", expected, result.AsString())
	}
}
