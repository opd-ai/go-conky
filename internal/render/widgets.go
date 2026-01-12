// Package render provides Ebiten-based rendering capabilities for conky-go.
// This file implements progress bar and gauge widgets for visualizing
// percentage-based data like CPU usage, memory, battery level, etc.
package render

import (
	"image/color"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// WidgetStyle defines the visual appearance of progress bar and gauge widgets.
type WidgetStyle struct {
	// FillColor is the color used to fill the progress portion.
	FillColor color.RGBA
	// BackgroundColor is the background color of the widget area.
	BackgroundColor color.RGBA
	// BorderColor is the color used for the widget border.
	BorderColor color.RGBA
	// BorderWidth is the width of the border in pixels.
	BorderWidth float32
	// ShowBorder indicates whether to draw the border.
	ShowBorder bool
	// ShowBackground indicates whether to draw the background.
	ShowBackground bool
}

// DefaultWidgetStyle returns a WidgetStyle with sensible defaults.
func DefaultWidgetStyle() WidgetStyle {
	return WidgetStyle{
		FillColor:       color.RGBA{R: 100, G: 200, B: 100, A: 255},
		BackgroundColor: color.RGBA{R: 50, G: 50, B: 50, A: 200},
		BorderColor:     color.RGBA{R: 150, G: 150, B: 150, A: 255},
		BorderWidth:     1.0,
		ShowBorder:      true,
		ShowBackground:  true,
	}
}

// ProgressBar displays a linear progress indicator.
// It can be horizontal or vertical, showing a filled portion
// representing a value between minimum and maximum.
type ProgressBar struct {
	x, y          float64
	width, height float64
	style         WidgetStyle
	value         float64
	minValue      float64
	maxValue      float64
	vertical      bool
	reversed      bool
	mu            sync.RWMutex
}

// NewProgressBar creates a new progress bar with the specified dimensions.
func NewProgressBar(x, y, width, height float64) *ProgressBar {
	return &ProgressBar{
		x:        x,
		y:        y,
		width:    width,
		height:   height,
		style:    DefaultWidgetStyle(),
		value:    0,
		minValue: 0,
		maxValue: 100,
		vertical: false,
		reversed: false,
	}
}

// SetStyle sets the visual style of the progress bar.
func (pb *ProgressBar) SetStyle(style WidgetStyle) {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	pb.style = style
}

// SetPosition sets the top-left position of the progress bar.
func (pb *ProgressBar) SetPosition(x, y float64) {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	pb.x = x
	pb.y = y
}

// SetSize sets the width and height of the progress bar.
func (pb *ProgressBar) SetSize(width, height float64) {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	pb.width = width
	pb.height = height
}

// SetValue sets the current value of the progress bar.
// The value is clamped to the min/max range.
func (pb *ProgressBar) SetValue(value float64) {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	pb.value = value
}

// SetRange sets the minimum and maximum values for the progress bar.
// If maxVal <= minVal, the values are swapped to ensure a valid range.
func (pb *ProgressBar) SetRange(minVal, maxVal float64) {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	if maxVal <= minVal {
		minVal, maxVal = maxVal, minVal
		if maxVal == minVal {
			maxVal = minVal + 1
		}
	}
	pb.minValue = minVal
	pb.maxValue = maxVal
}

// SetVertical sets whether the progress bar should be oriented vertically.
// Vertical bars fill from bottom to top by default.
func (pb *ProgressBar) SetVertical(vertical bool) {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	pb.vertical = vertical
}

// SetReversed sets whether the fill direction should be reversed.
// Horizontal: normally fills left-to-right, reversed fills right-to-left.
// Vertical: normally fills bottom-to-top, reversed fills top-to-bottom.
func (pb *ProgressBar) SetReversed(reversed bool) {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	pb.reversed = reversed
}

// Value returns the current value of the progress bar.
func (pb *ProgressBar) Value() float64 {
	pb.mu.RLock()
	defer pb.mu.RUnlock()
	return pb.value
}

// Percentage returns the current value as a percentage (0-100).
func (pb *ProgressBar) Percentage() float64 {
	pb.mu.RLock()
	defer pb.mu.RUnlock()
	return pb.calculatePercentage()
}

// calculatePercentage returns the normalized percentage (0-100) without locking.
func (pb *ProgressBar) calculatePercentage() float64 {
	valueRange := pb.maxValue - pb.minValue
	if valueRange == 0 {
		return 0
	}
	pct := ((pb.value - pb.minValue) / valueRange) * 100
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	return pct
}

// Draw renders the progress bar onto the given screen.
func (pb *ProgressBar) Draw(screen *ebiten.Image) {
	pb.mu.RLock()
	defer pb.mu.RUnlock()

	// Draw background if enabled
	if pb.style.ShowBackground {
		vector.DrawFilledRect(
			screen,
			float32(pb.x), float32(pb.y),
			float32(pb.width), float32(pb.height),
			pb.style.BackgroundColor,
			false,
		)
	}

	// Calculate the fill percentage (0.0 to 1.0)
	pct := pb.calculatePercentage() / 100.0

	// Draw the filled portion
	if pb.vertical {
		fillHeight := pct * pb.height
		var fillY float64
		if pb.reversed {
			// Fill from top to bottom
			fillY = pb.y
		} else {
			// Fill from bottom to top
			fillY = pb.y + pb.height - fillHeight
		}
		vector.DrawFilledRect(
			screen,
			float32(pb.x), float32(fillY),
			float32(pb.width), float32(fillHeight),
			pb.style.FillColor,
			false,
		)
	} else {
		fillWidth := pct * pb.width
		var fillX float64
		if pb.reversed {
			// Fill from right to left
			fillX = pb.x + pb.width - fillWidth
		} else {
			// Fill from left to right
			fillX = pb.x
		}
		vector.DrawFilledRect(
			screen,
			float32(fillX), float32(pb.y),
			float32(fillWidth), float32(pb.height),
			pb.style.FillColor,
			false,
		)
	}

	// Draw border if enabled
	if pb.style.ShowBorder && pb.style.BorderWidth > 0 {
		vector.StrokeRect(
			screen,
			float32(pb.x), float32(pb.y),
			float32(pb.width), float32(pb.height),
			pb.style.BorderWidth,
			pb.style.BorderColor,
			false,
		)
	}
}

// Gauge displays a circular or arc-shaped progress indicator.
// It shows a value as a filled arc, commonly used for speedometers,
// CPU meters, and similar radial displays.
type Gauge struct {
	x, y       float64 // Center position
	radius     float64
	style      WidgetStyle
	value      float64
	minValue   float64
	maxValue   float64
	startAngle float64 // Starting angle in radians (0 = right, π/2 = down)
	endAngle   float64 // Ending angle in radians
	thickness  float64 // Arc thickness in pixels
	clockwise  bool    // Direction of fill
	mu         sync.RWMutex
}

// NewGauge creates a new gauge with the specified center position and radius.
func NewGauge(x, y, radius float64) *Gauge {
	return &Gauge{
		x:          x,
		y:          y,
		radius:     radius,
		style:      DefaultWidgetStyle(),
		value:      0,
		minValue:   0,
		maxValue:   100,
		startAngle: math.Pi * 0.75, // 135 degrees (upper-left quadrant)
		endAngle:   math.Pi * 2.25, // 405 degrees, 270° arc ending at lower-left
		thickness:  10,
		clockwise:  true,
	}
}

// SetStyle sets the visual style of the gauge.
func (g *Gauge) SetStyle(style WidgetStyle) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.style = style
}

// SetPosition sets the center position of the gauge.
func (g *Gauge) SetPosition(x, y float64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.x = x
	g.y = y
}

// SetRadius sets the outer radius of the gauge.
func (g *Gauge) SetRadius(radius float64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if radius > 0 {
		g.radius = radius
	}
}

// SetValue sets the current value of the gauge.
func (g *Gauge) SetValue(value float64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.value = value
}

// SetRange sets the minimum and maximum values for the gauge.
// If maxVal <= minVal, the values are swapped to ensure a valid range.
func (g *Gauge) SetRange(minVal, maxVal float64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if maxVal <= minVal {
		minVal, maxVal = maxVal, minVal
		if maxVal == minVal {
			maxVal = minVal + 1
		}
	}
	g.minValue = minVal
	g.maxValue = maxVal
}

// SetAngles sets the start and end angles for the gauge arc in radians.
// Angles are measured from the positive X axis (right), with positive
// values going clockwise. Common configurations:
// - Half circle (bottom): startAngle=π, endAngle=2π (or 0)
// - 270° arc: startAngle=0.75π, endAngle=2.25π
func (g *Gauge) SetAngles(startAngle, endAngle float64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.startAngle = startAngle
	g.endAngle = endAngle
}

// SetThickness sets the thickness of the gauge arc in pixels.
func (g *Gauge) SetThickness(thickness float64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if thickness > 0 {
		g.thickness = thickness
	}
}

// SetClockwise sets whether the gauge fills in clockwise direction.
func (g *Gauge) SetClockwise(clockwise bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.clockwise = clockwise
}

// Value returns the current value of the gauge.
func (g *Gauge) Value() float64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.value
}

// Percentage returns the current value as a percentage (0-100).
func (g *Gauge) Percentage() float64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.calculatePercentage()
}

// calculatePercentage returns the normalized percentage (0-100) without locking.
func (g *Gauge) calculatePercentage() float64 {
	valueRange := g.maxValue - g.minValue
	if valueRange == 0 {
		return 0
	}
	pct := ((g.value - g.minValue) / valueRange) * 100
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	return pct
}

// Draw renders the gauge onto the given screen.
func (g *Gauge) Draw(screen *ebiten.Image) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.radius <= 0 || g.thickness <= 0 {
		return
	}

	// Draw background arc if enabled
	if g.style.ShowBackground {
		g.drawArc(screen, g.startAngle, g.endAngle, g.style.BackgroundColor)
	}

	// Calculate the fill percentage (0.0 to 1.0)
	pct := g.calculatePercentage() / 100.0

	if pct > 0 {
		// Calculate the fill angle
		totalAngle := g.endAngle - g.startAngle
		fillAngle := totalAngle * pct

		var fillStart, fillEnd float64
		if g.clockwise {
			fillStart = g.startAngle
			fillEnd = g.startAngle + fillAngle
		} else {
			fillEnd = g.endAngle
			fillStart = g.endAngle - fillAngle
		}

		// Draw the filled arc
		g.drawArc(screen, fillStart, fillEnd, g.style.FillColor)
	}
}

// drawArc draws an arc segment using line segments to approximate the curve.
func (g *Gauge) drawArc(screen *ebiten.Image, startAngle, endAngle float64, clr color.RGBA) {
	// Calculate the number of segments based on arc length for smooth curves
	arcLength := math.Abs(endAngle-startAngle) * g.radius
	segments := int(arcLength / 2) // Roughly 2 pixels per segment
	if segments < 8 {
		segments = 8
	}
	if segments > 360 {
		segments = 360
	}

	angleStep := (endAngle - startAngle) / float64(segments)
	innerRadius := g.radius - g.thickness
	if innerRadius < 0 {
		innerRadius = 0
	}

	// Draw the arc as filled trapezoids between segments
	for i := 0; i < segments; i++ {
		angle1 := startAngle + float64(i)*angleStep
		angle2 := startAngle + float64(i+1)*angleStep

		// Outer edge points
		outerX1 := g.x + g.radius*math.Cos(angle1)
		outerY1 := g.y + g.radius*math.Sin(angle1)
		outerX2 := g.x + g.radius*math.Cos(angle2)
		outerY2 := g.y + g.radius*math.Sin(angle2)

		// Inner edge points
		innerX1 := g.x + innerRadius*math.Cos(angle1)
		innerY1 := g.y + innerRadius*math.Sin(angle1)
		innerX2 := g.x + innerRadius*math.Cos(angle2)
		innerY2 := g.y + innerRadius*math.Sin(angle2)

		// Draw as two triangles to form a quad
		// Triangle 1: outer1, outer2, inner1
		drawTriangle(screen,
			float32(outerX1), float32(outerY1),
			float32(outerX2), float32(outerY2),
			float32(innerX1), float32(innerY1),
			clr)

		// Triangle 2: inner1, outer2, inner2
		drawTriangle(screen,
			float32(innerX1), float32(innerY1),
			float32(outerX2), float32(outerY2),
			float32(innerX2), float32(innerY2),
			clr)
	}
}

// drawTriangle draws a filled triangle using Ebiten's vector package.
func drawTriangle(screen *ebiten.Image, x1, y1, x2, y2, x3, y3 float32, clr color.RGBA) {
	var path vector.Path
	path.MoveTo(x1, y1)
	path.LineTo(x2, y2)
	path.LineTo(x3, y3)
	path.Close()

	vertices, indices := path.AppendVerticesAndIndicesForFilling(nil, nil)

	// Set color for all vertices
	r := float32(clr.R) / 255
	g := float32(clr.G) / 255
	b := float32(clr.B) / 255
	a := float32(clr.A) / 255
	for i := range vertices {
		vertices[i].ColorR = r
		vertices[i].ColorG = g
		vertices[i].ColorB = b
		vertices[i].ColorA = a
	}

	screen.DrawTriangles(vertices, indices, emptySubImage, nil)
}

// emptySubImage is a 1x1 white image used for filling shapes.
var emptySubImage = func() *ebiten.Image {
	img := ebiten.NewImage(1, 1)
	img.Fill(color.White)
	return img
}()
