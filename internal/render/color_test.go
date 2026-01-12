package render

import (
	"image/color"
	"math"
	"testing"
)

func TestParseColorNamed(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected color.RGBA
		wantErr  bool
	}{
		{"red", "red", color.RGBA{R: 255, G: 0, B: 0, A: 255}, false},
		{"Red uppercase", "Red", color.RGBA{R: 255, G: 0, B: 0, A: 255}, false},
		{"RED uppercase", "RED", color.RGBA{R: 255, G: 0, B: 0, A: 255}, false},
		{"blue", "blue", color.RGBA{R: 0, G: 0, B: 255, A: 255}, false},
		{"green", "green", color.RGBA{R: 0, G: 128, B: 0, A: 255}, false},
		{"lime", "lime", color.RGBA{R: 0, G: 255, B: 0, A: 255}, false},
		{"white", "white", color.RGBA{R: 255, G: 255, B: 255, A: 255}, false},
		{"black", "black", color.RGBA{R: 0, G: 0, B: 0, A: 255}, false},
		{"transparent", "transparent", color.RGBA{R: 0, G: 0, B: 0, A: 0}, false},
		{"gray", "gray", color.RGBA{R: 128, G: 128, B: 128, A: 255}, false},
		{"grey", "grey", color.RGBA{R: 128, G: 128, B: 128, A: 255}, false},
		{"with spaces", "  red  ", color.RGBA{R: 255, G: 0, B: 0, A: 255}, false},
		{"orange", "orange", color.RGBA{R: 255, G: 165, B: 0, A: 255}, false},
		{"purple", "purple", color.RGBA{R: 128, G: 0, B: 128, A: 255}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseColor(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseColor(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("ParseColor(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParseColorHex(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected color.RGBA
		wantErr  bool
	}{
		// 6-character hex
		{"hex #RRGGBB", "#FF0000", color.RGBA{R: 255, G: 0, B: 0, A: 255}, false},
		{"hex #rrggbb lowercase", "#ff0000", color.RGBA{R: 255, G: 0, B: 0, A: 255}, false},
		{"hex without #", "FF0000", color.RGBA{R: 255, G: 0, B: 0, A: 255}, false},
		{"hex green", "#00FF00", color.RGBA{R: 0, G: 255, B: 0, A: 255}, false},
		{"hex blue", "#0000FF", color.RGBA{R: 0, G: 0, B: 255, A: 255}, false},
		{"hex mixed", "#1A2B3C", color.RGBA{R: 26, G: 43, B: 60, A: 255}, false},

		// 8-character hex (with alpha)
		{"hex #RRGGBBAA", "#FF000080", color.RGBA{R: 255, G: 0, B: 0, A: 128}, false},
		{"hex full alpha", "#FF0000FF", color.RGBA{R: 255, G: 0, B: 0, A: 255}, false},
		{"hex zero alpha", "#FF000000", color.RGBA{R: 255, G: 0, B: 0, A: 0}, false},

		// 3-character hex shorthand
		{"hex #RGB", "#F00", color.RGBA{R: 255, G: 0, B: 0, A: 255}, false},
		{"hex #rgb lowercase", "#f0f", color.RGBA{R: 255, G: 0, B: 255, A: 255}, false},
		{"hex ABC", "#ABC", color.RGBA{R: 170, G: 187, B: 204, A: 255}, false},

		// 4-character hex shorthand (with alpha)
		{"hex #RGBA", "#F008", color.RGBA{R: 255, G: 0, B: 0, A: 136}, false},
		{"hex #RGBA full", "#F00F", color.RGBA{R: 255, G: 0, B: 0, A: 255}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseColor(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseColor(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("ParseColor(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParseColorRGBFunc(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected color.RGBA
		wantErr  bool
	}{
		{"rgb basic", "rgb(255, 0, 0)", color.RGBA{R: 255, G: 0, B: 0, A: 255}, false},
		{"rgb no spaces", "rgb(255,0,0)", color.RGBA{R: 255, G: 0, B: 0, A: 255}, false},
		{"rgb uppercase", "RGB(255, 0, 0)", color.RGBA{R: 255, G: 0, B: 0, A: 255}, false},
		{"rgb mixed case", "Rgb(100, 150, 200)", color.RGBA{R: 100, G: 150, B: 200, A: 255}, false},
		{"rgb zeros", "rgb(0, 0, 0)", color.RGBA{R: 0, G: 0, B: 0, A: 255}, false},
		{"rgb max", "rgb(255, 255, 255)", color.RGBA{R: 255, G: 255, B: 255, A: 255}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseColor(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseColor(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("ParseColor(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParseColorRGBAFunc(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected color.RGBA
		wantErr  bool
	}{
		{"rgba float alpha", "rgba(255, 0, 0, 0.5)", color.RGBA{R: 255, G: 0, B: 0, A: 127}, false},
		{"rgba float zero", "rgba(255, 0, 0, 0.0)", color.RGBA{R: 255, G: 0, B: 0, A: 0}, false},
		{"rgba float one", "rgba(255, 0, 0, 1.0)", color.RGBA{R: 255, G: 0, B: 0, A: 255}, false},
		{"rgba int alpha", "rgba(255, 0, 0, 128)", color.RGBA{R: 255, G: 0, B: 0, A: 128}, false},
		{"rgba uppercase", "RGBA(255, 0, 0, 0.5)", color.RGBA{R: 255, G: 0, B: 0, A: 127}, false},
		{"rgba mixed case", "Rgba(100, 150, 200, 0.8)", color.RGBA{R: 100, G: 150, B: 200, A: 204}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseColor(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseColor(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("ParseColor(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParseColorErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"invalid name", "notacolor"},
		{"invalid hex length", "#FFFFF"},
		{"invalid hex chars", "#GGGGGG"},
		{"rgb missing values", "rgb(255, 0)"},
		{"rgb too many values", "rgb(255, 0, 0, 0)"},
		{"rgba missing values", "rgba(255, 0, 0)"},
		{"rgb invalid format", "rgb(abc, 0, 0)"},
		{"rgb out of range", "rgb(256, 0, 0)"},
		{"incomplete rgb", "rgb("},
		{"incomplete rgba", "rgba("},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseColor(tt.input)
			if err == nil {
				t.Errorf("ParseColor(%q) expected error, got nil", tt.input)
			}
		})
	}
}

func TestMustParseColor(t *testing.T) {
	// Test successful parsing
	c := MustParseColor("red")
	if c != NamedColors["red"] {
		t.Errorf("MustParseColor(\"red\") = %v, want %v", c, NamedColors["red"])
	}

	// Test panic on invalid input
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("MustParseColor(\"invalid\") did not panic")
		}
	}()
	MustParseColor("notacolor")
}

func TestToHex(t *testing.T) {
	tests := []struct {
		name     string
		color    color.RGBA
		expected string
	}{
		{"red full alpha", color.RGBA{R: 255, G: 0, B: 0, A: 255}, "#FF0000"},
		{"green full alpha", color.RGBA{R: 0, G: 255, B: 0, A: 255}, "#00FF00"},
		{"blue full alpha", color.RGBA{R: 0, G: 0, B: 255, A: 255}, "#0000FF"},
		{"white", color.RGBA{R: 255, G: 255, B: 255, A: 255}, "#FFFFFF"},
		{"black", color.RGBA{R: 0, G: 0, B: 0, A: 255}, "#000000"},
		{"with alpha", color.RGBA{R: 255, G: 0, B: 0, A: 128}, "#FF000080"},
		{"transparent", color.RGBA{R: 0, G: 0, B: 0, A: 0}, "#00000000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToHex(tt.color)
			if got != tt.expected {
				t.Errorf("ToHex(%v) = %q, want %q", tt.color, got, tt.expected)
			}
		})
	}
}

func TestToRGBA(t *testing.T) {
	tests := []struct {
		name     string
		color    color.RGBA
		expected string
	}{
		{"red full alpha", color.RGBA{R: 255, G: 0, B: 0, A: 255}, "rgba(255, 0, 0, 1.00)"},
		{"half alpha", color.RGBA{R: 255, G: 0, B: 0, A: 127}, "rgba(255, 0, 0, 0.50)"},
		{"transparent", color.RGBA{R: 0, G: 0, B: 0, A: 0}, "rgba(0, 0, 0, 0.00)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToRGBA(tt.color)
			if got != tt.expected {
				t.Errorf("ToRGBA(%v) = %q, want %q", tt.color, got, tt.expected)
			}
		})
	}
}

func TestWithAlpha(t *testing.T) {
	c := color.RGBA{R: 255, G: 0, B: 0, A: 255}

	got := WithAlpha(c, 128)
	expected := color.RGBA{R: 255, G: 0, B: 0, A: 128}
	if got != expected {
		t.Errorf("WithAlpha(%v, 128) = %v, want %v", c, got, expected)
	}
}

func TestWithOpacity(t *testing.T) {
	c := color.RGBA{R: 255, G: 0, B: 0, A: 255}

	tests := []struct {
		name     string
		opacity  float64
		expected uint8
	}{
		{"full opacity", 1.0, 255},
		{"half opacity", 0.5, 127},
		{"zero opacity", 0.0, 0},
		{"negative clamped", -0.5, 0},
		{"over 1 clamped", 1.5, 255},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WithOpacity(c, tt.opacity)
			if got.A != tt.expected {
				t.Errorf("WithOpacity(%v, %f).A = %d, want %d", c, tt.opacity, got.A, tt.expected)
			}
			// RGB should be unchanged
			if got.R != c.R || got.G != c.G || got.B != c.B {
				t.Errorf("WithOpacity changed RGB values: got %v, original %v", got, c)
			}
		})
	}
}

func TestBlend(t *testing.T) {
	c1 := color.RGBA{R: 255, G: 0, B: 0, A: 255} // Red
	c2 := color.RGBA{R: 0, G: 0, B: 255, A: 255} // Blue
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	black := color.RGBA{R: 0, G: 0, B: 0, A: 255}

	tests := []struct {
		name     string
		c1, c2   color.RGBA
		ratio    float64
		expected color.RGBA
	}{
		{"ratio 0", c1, c2, 0.0, c1},
		{"ratio 1", c1, c2, 1.0, c2},
		{"ratio 0.5", c1, c2, 0.5, color.RGBA{R: 127, G: 0, B: 127, A: 255}},
		{"black to white 0.5", black, white, 0.5, color.RGBA{R: 127, G: 127, B: 127, A: 255}},
		{"negative ratio clamped", c1, c2, -0.5, c1},
		{"over 1 ratio clamped", c1, c2, 1.5, c2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Blend(tt.c1, tt.c2, tt.ratio)
			if got != tt.expected {
				t.Errorf("Blend(%v, %v, %f) = %v, want %v", tt.c1, tt.c2, tt.ratio, got, tt.expected)
			}
		})
	}
}

func TestLighten(t *testing.T) {
	red := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	midGray := color.RGBA{R: 128, G: 128, B: 128, A: 255}

	tests := []struct {
		name     string
		color    color.RGBA
		amount   float64
		expected color.RGBA
	}{
		{"no change", red, 0.0, red},
		{"full lighten", midGray, 1.0, color.RGBA{R: 255, G: 255, B: 255, A: 255}},
		{"half lighten gray", midGray, 0.5, color.RGBA{R: 191, G: 191, B: 191, A: 255}},
		{"negative clamped", red, -0.5, red},
		{"over 1 clamped", midGray, 1.5, color.RGBA{R: 255, G: 255, B: 255, A: 255}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Lighten(tt.color, tt.amount)
			if got != tt.expected {
				t.Errorf("Lighten(%v, %f) = %v, want %v", tt.color, tt.amount, got, tt.expected)
			}
		})
	}
}

func TestDarken(t *testing.T) {
	red := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	midGray := color.RGBA{R: 128, G: 128, B: 128, A: 255}

	tests := []struct {
		name     string
		color    color.RGBA
		amount   float64
		expected color.RGBA
	}{
		{"no change", red, 0.0, red},
		{"full darken", midGray, 1.0, color.RGBA{R: 0, G: 0, B: 0, A: 255}},
		{"half darken gray", midGray, 0.5, color.RGBA{R: 64, G: 64, B: 64, A: 255}},
		{"negative clamped", red, -0.5, red},
		{"over 1 clamped", midGray, 1.5, color.RGBA{R: 0, G: 0, B: 0, A: 255}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Darken(tt.color, tt.amount)
			if got != tt.expected {
				t.Errorf("Darken(%v, %f) = %v, want %v", tt.color, tt.amount, got, tt.expected)
			}
		})
	}
}

func TestInvert(t *testing.T) {
	tests := []struct {
		name     string
		color    color.RGBA
		expected color.RGBA
	}{
		{"black to white", color.RGBA{R: 0, G: 0, B: 0, A: 255}, color.RGBA{R: 255, G: 255, B: 255, A: 255}},
		{"white to black", color.RGBA{R: 255, G: 255, B: 255, A: 255}, color.RGBA{R: 0, G: 0, B: 0, A: 255}},
		{"red to cyan", color.RGBA{R: 255, G: 0, B: 0, A: 255}, color.RGBA{R: 0, G: 255, B: 255, A: 255}},
		{"preserves alpha", color.RGBA{R: 255, G: 0, B: 0, A: 128}, color.RGBA{R: 0, G: 255, B: 255, A: 128}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Invert(tt.color)
			if got != tt.expected {
				t.Errorf("Invert(%v) = %v, want %v", tt.color, got, tt.expected)
			}
		})
	}
}

func TestGrayscale(t *testing.T) {
	tests := []struct {
		name     string
		color    color.RGBA
		expected color.RGBA
	}{
		{"white", color.RGBA{R: 255, G: 255, B: 255, A: 255}, color.RGBA{R: 255, G: 255, B: 255, A: 255}},
		{"black", color.RGBA{R: 0, G: 0, B: 0, A: 255}, color.RGBA{R: 0, G: 0, B: 0, A: 255}},
		{"red", color.RGBA{R: 255, G: 0, B: 0, A: 255}, color.RGBA{R: 76, G: 76, B: 76, A: 255}},
		{"green", color.RGBA{R: 0, G: 255, B: 0, A: 255}, color.RGBA{R: 149, G: 149, B: 149, A: 255}},
		{"blue", color.RGBA{R: 0, G: 0, B: 255, A: 255}, color.RGBA{R: 29, G: 29, B: 29, A: 255}},
		{"preserves alpha", color.RGBA{R: 255, G: 0, B: 0, A: 128}, color.RGBA{R: 76, G: 76, B: 76, A: 128}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Grayscale(tt.color)
			if got != tt.expected {
				t.Errorf("Grayscale(%v) = %v, want %v", tt.color, got, tt.expected)
			}
		})
	}
}

func TestRGBAToHSLAndBack(t *testing.T) {
	tests := []color.RGBA{
		{R: 255, G: 0, B: 0, A: 255},     // Red
		{R: 0, G: 255, B: 0, A: 255},     // Green
		{R: 0, G: 0, B: 255, A: 255},     // Blue
		{R: 255, G: 255, B: 0, A: 255},   // Yellow
		{R: 255, G: 0, B: 255, A: 255},   // Magenta
		{R: 0, G: 255, B: 255, A: 255},   // Cyan
		{R: 255, G: 255, B: 255, A: 255}, // White
		{R: 0, G: 0, B: 0, A: 255},       // Black
		{R: 128, G: 128, B: 128, A: 255}, // Gray
		{R: 100, G: 150, B: 200, A: 255}, // Random color
	}

	for _, c := range tests {
		t.Run(ToHex(c), func(t *testing.T) {
			hsl := RGBAToHSL(c)
			back := hsl.ToRGBA()

			// Allow for small rounding errors
			if !colorApproxEqual(c, back, 1) {
				t.Errorf("RGBA->HSL->RGBA conversion failed: %v -> %v -> %v", c, hsl, back)
			}
		})
	}
}

func colorApproxEqual(c1, c2 color.RGBA, tolerance uint8) bool {
	return absDiff(c1.R, c2.R) <= tolerance &&
		absDiff(c1.G, c2.G) <= tolerance &&
		absDiff(c1.B, c2.B) <= tolerance
}

func absDiff(a, b uint8) uint8 {
	if a > b {
		return a - b
	}
	return b - a
}

func TestHSLToRGBAWithAlpha(t *testing.T) {
	hsl := HSL{H: 0, S: 1.0, L: 0.5} // Red in HSL
	alpha := uint8(128)

	got := HSLToRGBA(hsl, alpha)
	if got.A != alpha {
		t.Errorf("HSLToRGBA() alpha = %d, want %d", got.A, alpha)
	}
	// Should be red
	if got.R != 255 || got.G != 0 || got.B != 0 {
		t.Errorf("HSLToRGBA(red) = %v, want red", got)
	}
}

func TestAdjustHue(t *testing.T) {
	red := color.RGBA{R: 255, G: 0, B: 0, A: 255}

	tests := []struct {
		name    string
		degrees float64
		// Expected approximate hue after adjustment
		expectedHueRange [2]float64
	}{
		{"no change", 0, [2]float64{0, 1}},
		{"rotate 120", 120, [2]float64{119, 121}},
		{"rotate 240", 240, [2]float64{239, 241}},
		{"rotate 360", 360, [2]float64{0, 1}},
		{"rotate negative", -120, [2]float64{239, 241}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AdjustHue(red, tt.degrees)
			hsl := RGBAToHSL(got)

			// Normalize hue
			h := hsl.H
			if h < 0 {
				h += 360
			}

			// Allow for some tolerance in hue
			// Check if hue is outside the expected range
			if h < tt.expectedHueRange[0] || h > tt.expectedHueRange[1] {
				// Also check if it's close to 360/0 (special case for red)
				if tt.expectedHueRange[0] < 5 && (h > 355 || h < 5) {
					return // OK for red/0 degrees
				}
				t.Errorf("AdjustHue(%v, %f) hue = %f, want in range %v", red, tt.degrees, h, tt.expectedHueRange)
			}
		})
	}
}

func TestSaturate(t *testing.T) {
	gray := color.RGBA{R: 128, G: 128, B: 128, A: 255}
	partialRed := color.RGBA{R: 192, G: 64, B: 64, A: 255}

	tests := []struct {
		name   string
		color  color.RGBA
		amount float64
	}{
		{"saturate gray", gray, 0.5},
		{"saturate partial color", partialRed, 0.5},
		{"full saturate", partialRed, 1.0},
		{"no change", partialRed, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Saturate(tt.color, tt.amount)

			// Just verify it doesn't panic and returns valid color
			if got.A != tt.color.A {
				t.Errorf("Saturate() changed alpha: got %d, want %d", got.A, tt.color.A)
			}

			// If amount > 0, saturation should increase (or stay same if already max)
			if tt.amount > 0 {
				origHSL := RGBAToHSL(tt.color)
				gotHSL := RGBAToHSL(got)
				if gotHSL.S < origHSL.S-0.01 { // Allow small tolerance
					t.Errorf("Saturate() decreased saturation: %f -> %f", origHSL.S, gotHSL.S)
				}
			}
		})
	}
}

func TestDesaturate(t *testing.T) {
	red := color.RGBA{R: 255, G: 0, B: 0, A: 255}

	tests := []struct {
		name   string
		color  color.RGBA
		amount float64
	}{
		{"desaturate red", red, 0.5},
		{"full desaturate", red, 1.0},
		{"no change", red, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Desaturate(tt.color, tt.amount)

			// Verify alpha unchanged
			if got.A != tt.color.A {
				t.Errorf("Desaturate() changed alpha: got %d, want %d", got.A, tt.color.A)
			}

			// If amount > 0, saturation should decrease
			if tt.amount > 0 {
				origHSL := RGBAToHSL(tt.color)
				gotHSL := RGBAToHSL(got)
				if gotHSL.S > origHSL.S+0.01 { // Allow small tolerance
					t.Errorf("Desaturate() increased saturation: %f -> %f", origHSL.S, gotHSL.S)
				}
			}
		})
	}
}

func TestAlphaBlend(t *testing.T) {
	tests := []struct {
		name     string
		bg       color.RGBA
		fg       color.RGBA
		expected color.RGBA
	}{
		{
			"opaque over opaque",
			color.RGBA{R: 255, G: 0, B: 0, A: 255},
			color.RGBA{R: 0, G: 255, B: 0, A: 255},
			color.RGBA{R: 0, G: 255, B: 0, A: 255},
		},
		{
			"transparent over opaque",
			color.RGBA{R: 255, G: 0, B: 0, A: 255},
			color.RGBA{R: 0, G: 255, B: 0, A: 0},
			color.RGBA{R: 255, G: 0, B: 0, A: 255},
		},
		{
			"half transparent over opaque",
			color.RGBA{R: 255, G: 0, B: 0, A: 255},
			color.RGBA{R: 0, G: 0, B: 255, A: 127},
			color.RGBA{R: 127, G: 0, B: 127, A: 255},
		},
		{
			"both transparent",
			color.RGBA{R: 0, G: 0, B: 0, A: 0},
			color.RGBA{R: 0, G: 0, B: 0, A: 0},
			color.RGBA{R: 0, G: 0, B: 0, A: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AlphaBlend(tt.bg, tt.fg)
			// Allow for rounding differences
			if !colorApproxEqual(got, tt.expected, 2) || absDiff(got.A, tt.expected.A) > 2 {
				t.Errorf("AlphaBlend(%v, %v) = %v, want %v", tt.bg, tt.fg, got, tt.expected)
			}
		})
	}
}

func TestLuminance(t *testing.T) {
	tests := []struct {
		name     string
		color    color.RGBA
		expected float64
		delta    float64
	}{
		{"white", color.RGBA{R: 255, G: 255, B: 255, A: 255}, 1.0, 0.01},
		{"black", color.RGBA{R: 0, G: 0, B: 0, A: 255}, 0.0, 0.01},
		{"red", color.RGBA{R: 255, G: 0, B: 0, A: 255}, 0.2126, 0.01},
		{"green", color.RGBA{R: 0, G: 255, B: 0, A: 255}, 0.7152, 0.01},
		{"blue", color.RGBA{R: 0, G: 0, B: 255, A: 255}, 0.0722, 0.01},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Luminance(tt.color)
			if math.Abs(got-tt.expected) > tt.delta {
				t.Errorf("Luminance(%v) = %f, want %f (±%f)", tt.color, got, tt.expected, tt.delta)
			}
		})
	}
}

func TestContrastRatio(t *testing.T) {
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	black := color.RGBA{R: 0, G: 0, B: 0, A: 255}

	// Maximum contrast is white on black
	ratio := ContrastRatio(white, black)
	if ratio < 20 || ratio > 22 {
		t.Errorf("ContrastRatio(white, black) = %f, expected ~21", ratio)
	}

	// Same color should give ratio 1
	ratio = ContrastRatio(white, white)
	if math.Abs(ratio-1.0) > 0.01 {
		t.Errorf("ContrastRatio(white, white) = %f, expected 1.0", ratio)
	}
}

func TestIsLightAndIsDark(t *testing.T) {
	tests := []struct {
		name    string
		color   color.RGBA
		isLight bool
	}{
		{"white", color.RGBA{R: 255, G: 255, B: 255, A: 255}, true},
		{"black", color.RGBA{R: 0, G: 0, B: 0, A: 255}, false},
		{"yellow", color.RGBA{R: 255, G: 255, B: 0, A: 255}, true},
		{"dark blue", color.RGBA{R: 0, G: 0, B: 128, A: 255}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if IsLight(tt.color) != tt.isLight {
				t.Errorf("IsLight(%v) = %v, want %v", tt.color, IsLight(tt.color), tt.isLight)
			}
			if IsDark(tt.color) != !tt.isLight {
				t.Errorf("IsDark(%v) = %v, want %v", tt.color, IsDark(tt.color), !tt.isLight)
			}
		})
	}
}

func TestGradient(t *testing.T) {
	red := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	blue := color.RGBA{R: 0, G: 0, B: 255, A: 255}
	green := color.RGBA{R: 0, G: 255, B: 0, A: 255}

	t.Run("two color gradient", func(t *testing.T) {
		g := NewGradient(
			GradientStop{Position: 0, Color: red},
			GradientStop{Position: 1, Color: blue},
		)

		// At position 0
		got := g.At(0)
		if got != red {
			t.Errorf("Gradient.At(0) = %v, want %v", got, red)
		}

		// At position 1
		got = g.At(1)
		if got != blue {
			t.Errorf("Gradient.At(1) = %v, want %v", got, blue)
		}

		// At position 0.5
		got = g.At(0.5)
		expected := Blend(red, blue, 0.5)
		if got != expected {
			t.Errorf("Gradient.At(0.5) = %v, want %v", got, expected)
		}
	})

	t.Run("three color gradient", func(t *testing.T) {
		g := NewGradient(
			GradientStop{Position: 0, Color: red},
			GradientStop{Position: 0.5, Color: green},
			GradientStop{Position: 1, Color: blue},
		)

		// At position 0.25 (between red and green)
		got := g.At(0.25)
		expected := Blend(red, green, 0.5)
		if got != expected {
			t.Errorf("Gradient.At(0.25) = %v, want %v", got, expected)
		}

		// At position 0.75 (between green and blue)
		got = g.At(0.75)
		expected = Blend(green, blue, 0.5)
		if got != expected {
			t.Errorf("Gradient.At(0.75) = %v, want %v", got, expected)
		}
	})

	t.Run("unsorted stops are sorted", func(t *testing.T) {
		g := NewGradient(
			GradientStop{Position: 1, Color: blue},
			GradientStop{Position: 0, Color: red},
		)

		got := g.At(0)
		if got != red {
			t.Errorf("Gradient.At(0) with unsorted stops = %v, want %v", got, red)
		}
	})

	t.Run("out of range positions", func(t *testing.T) {
		g := NewGradient(
			GradientStop{Position: 0, Color: red},
			GradientStop{Position: 1, Color: blue},
		)

		// Below 0
		got := g.At(-0.5)
		if got != red {
			t.Errorf("Gradient.At(-0.5) = %v, want %v", got, red)
		}

		// Above 1
		got = g.At(1.5)
		if got != blue {
			t.Errorf("Gradient.At(1.5) = %v, want %v", got, blue)
		}
	})

	t.Run("empty gradient", func(t *testing.T) {
		g := NewGradient()
		got := g.At(0.5)
		if got != (color.RGBA{}) {
			t.Errorf("Empty Gradient.At(0.5) = %v, want zero", got)
		}
	})

	t.Run("single stop gradient", func(t *testing.T) {
		g := NewGradient(GradientStop{Position: 0.5, Color: red})
		got := g.At(0)
		if got != red {
			t.Errorf("Single stop Gradient.At(0) = %v, want %v", got, red)
		}
	})
}

func TestGradientAddStop(t *testing.T) {
	g := NewGradient(
		GradientStop{Position: 0, Color: color.RGBA{R: 255, G: 0, B: 0, A: 255}},
		GradientStop{Position: 1, Color: color.RGBA{R: 0, G: 0, B: 255, A: 255}},
	)

	green := color.RGBA{R: 0, G: 255, B: 0, A: 255}
	g.AddStop(0.5, green)

	stops := g.Stops()
	if len(stops) != 3 {
		t.Fatalf("Expected 3 stops, got %d", len(stops))
	}

	// Verify middle stop
	if stops[1].Position != 0.5 || stops[1].Color != green {
		t.Errorf("Middle stop = %v, want {0.5, green}", stops[1])
	}
}

func TestGradientStops(t *testing.T) {
	red := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	blue := color.RGBA{R: 0, G: 0, B: 255, A: 255}

	g := NewGradient(
		GradientStop{Position: 0, Color: red},
		GradientStop{Position: 1, Color: blue},
	)

	stops := g.Stops()

	// Verify it's a copy
	stops[0].Color = color.RGBA{R: 0, G: 255, B: 0, A: 255}
	originalStops := g.Stops()
	if originalStops[0].Color.G != 0 {
		t.Error("Stops() should return a copy, not original slice")
	}
}

func TestNamedColorsExist(t *testing.T) {
	// Verify some expected named colors exist
	expectedColors := []string{
		"red", "green", "blue", "white", "black",
		"yellow", "cyan", "magenta", "orange", "purple",
		"gray", "grey", "transparent",
	}

	for _, name := range expectedColors {
		if _, ok := NamedColors[name]; !ok {
			t.Errorf("Expected named color %q not found", name)
		}
	}
}

func BenchmarkParseColorHex(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = ParseColor("#FF5733")
	}
}

func BenchmarkParseColorNamed(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = ParseColor("red")
	}
}

func BenchmarkParseColorRGBA(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = ParseColor("rgba(255, 87, 51, 0.5)")
	}
}

func BenchmarkBlend(b *testing.B) {
	c1 := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	c2 := color.RGBA{R: 0, G: 0, B: 255, A: 255}
	for i := 0; i < b.N; i++ {
		_ = Blend(c1, c2, 0.5)
	}
}

func BenchmarkRGBAToHSL(b *testing.B) {
	c := color.RGBA{R: 100, G: 150, B: 200, A: 255}
	for i := 0; i < b.N; i++ {
		_ = RGBAToHSL(c)
	}
}

func BenchmarkHSLToRGBA(b *testing.B) {
	hsl := HSL{H: 210, S: 0.5, L: 0.6}
	for i := 0; i < b.N; i++ {
		_ = hsl.ToRGBA()
	}
}
