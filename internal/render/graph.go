// Package render provides Ebiten-based rendering capabilities for conky-go.
// This file implements graph widget types including line graphs, bar graphs,
// and histograms for visualizing time-series and categorical data.
package render

import (
	"image/color"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// GraphStyle defines the visual appearance of graph widgets.
type GraphStyle struct {
	// FillColor is the color used to fill graph elements.
	FillColor color.RGBA
	// StrokeColor is the color used for graph outlines and lines.
	StrokeColor color.RGBA
	// StrokeWidth is the width of lines in pixels.
	StrokeWidth float32
	// BackgroundColor is the background color of the graph area.
	BackgroundColor color.RGBA
	// ShowBackground indicates whether to draw the background.
	ShowBackground bool
}

// DefaultGraphStyle returns a GraphStyle with sensible defaults.
func DefaultGraphStyle() GraphStyle {
	return GraphStyle{
		FillColor:       color.RGBA{R: 100, G: 200, B: 100, A: 200},
		StrokeColor:     color.RGBA{R: 150, G: 255, B: 150, A: 255},
		StrokeWidth:     1.0,
		BackgroundColor: color.RGBA{R: 30, G: 30, B: 30, A: 150},
		ShowBackground:  true,
	}
}

// GraphWidget is the interface that all graph widgets must implement.
type GraphWidget interface {
	// Draw renders the graph onto the given screen.
	Draw(screen *ebiten.Image)
	// SetStyle sets the visual style of the graph.
	SetStyle(style GraphStyle)
	// SetPosition sets the top-left position of the graph.
	SetPosition(x, y float64)
	// SetSize sets the width and height of the graph.
	SetSize(width, height float64)
}

// LineGraph displays data points connected by lines.
// It is suitable for showing trends over time.
type LineGraph struct {
	x, y          float64
	width, height float64
	style         GraphStyle
	data          []float64
	maxPoints     int
	minValue      float64
	maxValue      float64
	autoScale     bool
	mu            sync.RWMutex
}

// NewLineGraph creates a new line graph with the specified dimensions.
func NewLineGraph(x, y, width, height float64) *LineGraph {
	return &LineGraph{
		x:         x,
		y:         y,
		width:     width,
		height:    height,
		style:     DefaultGraphStyle(),
		data:      make([]float64, 0),
		maxPoints: 100,
		minValue:  0,
		maxValue:  100,
		autoScale: true,
	}
}

// SetStyle sets the visual style of the graph.
func (lg *LineGraph) SetStyle(style GraphStyle) {
	lg.mu.Lock()
	defer lg.mu.Unlock()
	lg.style = style
}

// SetPosition sets the top-left position of the graph.
func (lg *LineGraph) SetPosition(x, y float64) {
	lg.mu.Lock()
	defer lg.mu.Unlock()
	lg.x = x
	lg.y = y
}

// SetSize sets the width and height of the graph.
func (lg *LineGraph) SetSize(width, height float64) {
	lg.mu.Lock()
	defer lg.mu.Unlock()
	lg.width = width
	lg.height = height
}

// SetMaxPoints sets the maximum number of data points to display.
func (lg *LineGraph) SetMaxPoints(n int) {
	lg.mu.Lock()
	defer lg.mu.Unlock()
	if n > 0 {
		lg.maxPoints = n
	}
}

// SetRange sets the minimum and maximum values for the Y axis.
// This disables auto-scaling.
func (lg *LineGraph) SetRange(minVal, maxVal float64) {
	lg.mu.Lock()
	defer lg.mu.Unlock()
	lg.minValue = minVal
	lg.maxValue = maxVal
	lg.autoScale = false
}

// SetAutoScale enables or disables automatic Y-axis scaling.
func (lg *LineGraph) SetAutoScale(enabled bool) {
	lg.mu.Lock()
	defer lg.mu.Unlock()
	lg.autoScale = enabled
}

// AddPoint adds a new data point to the graph.
// Old points are removed when maxPoints is exceeded.
func (lg *LineGraph) AddPoint(value float64) {
	lg.mu.Lock()
	defer lg.mu.Unlock()
	lg.data = append(lg.data, value)
	if len(lg.data) > lg.maxPoints {
		lg.data = lg.data[len(lg.data)-lg.maxPoints:]
	}
}

// SetData replaces all data points in the graph.
func (lg *LineGraph) SetData(data []float64) {
	lg.mu.Lock()
	defer lg.mu.Unlock()
	lg.data = make([]float64, len(data))
	copy(lg.data, data)
	if len(lg.data) > lg.maxPoints {
		lg.data = lg.data[len(lg.data)-lg.maxPoints:]
	}
}

// ClearData removes all data points from the graph.
func (lg *LineGraph) ClearData() {
	lg.mu.Lock()
	defer lg.mu.Unlock()
	lg.data = lg.data[:0]
}

// Draw renders the line graph onto the given screen.
func (lg *LineGraph) Draw(screen *ebiten.Image) {
	lg.mu.RLock()
	defer lg.mu.RUnlock()

	// Draw background if enabled
	if lg.style.ShowBackground {
		vector.DrawFilledRect(
			screen,
			float32(lg.x), float32(lg.y),
			float32(lg.width), float32(lg.height),
			lg.style.BackgroundColor,
			false,
		)
	}

	if len(lg.data) < 2 {
		return
	}

	// Calculate value range
	minVal, maxVal := lg.minValue, lg.maxValue
	if lg.autoScale {
		minVal, maxVal = lg.data[0], lg.data[0]
		for _, v := range lg.data {
			if v < minVal {
				minVal = v
			}
			if v > maxVal {
				maxVal = v
			}
		}
		// Add 10% padding
		padding := (maxVal - minVal) * 0.1
		if padding == 0 {
			padding = 1
		}
		minVal -= padding
		maxVal += padding
	}

	valueRange := maxVal - minVal
	if valueRange == 0 {
		valueRange = 1
	}

	// Calculate point spacing
	pointSpacing := lg.width / float64(len(lg.data)-1)

	// Draw lines connecting points
	for i := 0; i < len(lg.data)-1; i++ {
		x1 := lg.x + float64(i)*pointSpacing
		x2 := lg.x + float64(i+1)*pointSpacing

		// Normalize values to graph height (inverted because Y grows down)
		normalizedY1 := (lg.data[i] - minVal) / valueRange
		normalizedY2 := (lg.data[i+1] - minVal) / valueRange

		y1 := lg.y + lg.height - (normalizedY1 * lg.height)
		y2 := lg.y + lg.height - (normalizedY2 * lg.height)

		vector.StrokeLine(
			screen,
			float32(x1), float32(y1),
			float32(x2), float32(y2),
			lg.style.StrokeWidth,
			lg.style.StrokeColor,
			false,
		)
	}
}

// BarGraph displays data as vertical or horizontal bars.
// It is suitable for showing categorical data or comparisons.
type BarGraph struct {
	x, y          float64
	width, height float64
	style         GraphStyle
	data          []float64
	labels        []string
	barSpacing    float64
	horizontal    bool
	minValue      float64
	maxValue      float64
	autoScale     bool
	mu            sync.RWMutex
}

// NewBarGraph creates a new bar graph with the specified dimensions.
func NewBarGraph(x, y, width, height float64) *BarGraph {
	return &BarGraph{
		x:          x,
		y:          y,
		width:      width,
		height:     height,
		style:      DefaultGraphStyle(),
		data:       make([]float64, 0),
		labels:     make([]string, 0),
		barSpacing: 2,
		horizontal: false,
		minValue:   0,
		maxValue:   100,
		autoScale:  true,
	}
}

// SetStyle sets the visual style of the graph.
func (bg *BarGraph) SetStyle(style GraphStyle) {
	bg.mu.Lock()
	defer bg.mu.Unlock()
	bg.style = style
}

// SetPosition sets the top-left position of the graph.
func (bg *BarGraph) SetPosition(x, y float64) {
	bg.mu.Lock()
	defer bg.mu.Unlock()
	bg.x = x
	bg.y = y
}

// SetSize sets the width and height of the graph.
func (bg *BarGraph) SetSize(width, height float64) {
	bg.mu.Lock()
	defer bg.mu.Unlock()
	bg.width = width
	bg.height = height
}

// SetBarSpacing sets the spacing between bars in pixels.
func (bg *BarGraph) SetBarSpacing(spacing float64) {
	bg.mu.Lock()
	defer bg.mu.Unlock()
	if spacing >= 0 {
		bg.barSpacing = spacing
	}
}

// SetHorizontal sets whether bars should be drawn horizontally.
func (bg *BarGraph) SetHorizontal(horizontal bool) {
	bg.mu.Lock()
	defer bg.mu.Unlock()
	bg.horizontal = horizontal
}

// SetRange sets the minimum and maximum values for the value axis.
// This disables auto-scaling.
func (bg *BarGraph) SetRange(minVal, maxVal float64) {
	bg.mu.Lock()
	defer bg.mu.Unlock()
	bg.minValue = minVal
	bg.maxValue = maxVal
	bg.autoScale = false
}

// SetAutoScale enables or disables automatic value axis scaling.
func (bg *BarGraph) SetAutoScale(enabled bool) {
	bg.mu.Lock()
	defer bg.mu.Unlock()
	bg.autoScale = enabled
}

// SetData replaces all data values in the graph.
func (bg *BarGraph) SetData(data []float64) {
	bg.mu.Lock()
	defer bg.mu.Unlock()
	bg.data = make([]float64, len(data))
	copy(bg.data, data)
}

// SetLabels sets the labels for each bar.
func (bg *BarGraph) SetLabels(labels []string) {
	bg.mu.Lock()
	defer bg.mu.Unlock()
	bg.labels = make([]string, len(labels))
	copy(bg.labels, labels)
}

// ClearData removes all data from the graph.
func (bg *BarGraph) ClearData() {
	bg.mu.Lock()
	defer bg.mu.Unlock()
	bg.data = bg.data[:0]
	bg.labels = bg.labels[:0]
}

// Draw renders the bar graph onto the given screen.
func (bg *BarGraph) Draw(screen *ebiten.Image) {
	bg.mu.RLock()
	defer bg.mu.RUnlock()

	// Draw background if enabled
	if bg.style.ShowBackground {
		vector.DrawFilledRect(
			screen,
			float32(bg.x), float32(bg.y),
			float32(bg.width), float32(bg.height),
			bg.style.BackgroundColor,
			false,
		)
	}

	if len(bg.data) == 0 {
		return
	}

	// Calculate value range
	minVal, maxVal := bg.minValue, bg.maxValue
	if bg.autoScale {
		minVal = 0
		maxVal = bg.data[0]
		for _, v := range bg.data {
			if v > maxVal {
				maxVal = v
			}
		}
		if maxVal == 0 {
			maxVal = 1
		}
	}

	valueRange := maxVal - minVal
	if valueRange == 0 {
		valueRange = 1
	}

	n := len(bg.data)

	if bg.horizontal {
		// Horizontal bars
		totalSpacing := bg.barSpacing * float64(n-1)
		barHeight := (bg.height - totalSpacing) / float64(n)

		for i, value := range bg.data {
			normalized := (value - minVal) / valueRange
			if normalized < 0 {
				normalized = 0
			}
			if normalized > 1 {
				normalized = 1
			}

			barWidth := normalized * bg.width
			barY := bg.y + float64(i)*(barHeight+bg.barSpacing)

			vector.DrawFilledRect(
				screen,
				float32(bg.x), float32(barY),
				float32(barWidth), float32(barHeight),
				bg.style.FillColor,
				false,
			)

			// Draw outline
			if bg.style.StrokeWidth > 0 {
				vector.StrokeRect(
					screen,
					float32(bg.x), float32(barY),
					float32(barWidth), float32(barHeight),
					bg.style.StrokeWidth,
					bg.style.StrokeColor,
					false,
				)
			}
		}
	} else {
		// Vertical bars
		totalSpacing := bg.barSpacing * float64(n-1)
		barWidth := (bg.width - totalSpacing) / float64(n)

		for i, value := range bg.data {
			normalized := (value - minVal) / valueRange
			if normalized < 0 {
				normalized = 0
			}
			if normalized > 1 {
				normalized = 1
			}

			barHeight := normalized * bg.height
			barX := bg.x + float64(i)*(barWidth+bg.barSpacing)
			barY := bg.y + bg.height - barHeight

			vector.DrawFilledRect(
				screen,
				float32(barX), float32(barY),
				float32(barWidth), float32(barHeight),
				bg.style.FillColor,
				false,
			)

			// Draw outline
			if bg.style.StrokeWidth > 0 {
				vector.StrokeRect(
					screen,
					float32(barX), float32(barY),
					float32(barWidth), float32(barHeight),
					bg.style.StrokeWidth,
					bg.style.StrokeColor,
					false,
				)
			}
		}
	}
}

// Histogram displays the frequency distribution of data values.
// Values are grouped into bins and displayed as bars.
type Histogram struct {
	x, y          float64
	width, height float64
	style         GraphStyle
	data          []float64
	binCount      int
	minValue      float64
	maxValue      float64
	autoRange     bool
	mu            sync.RWMutex
}

// NewHistogram creates a new histogram with the specified dimensions.
func NewHistogram(x, y, width, height float64) *Histogram {
	return &Histogram{
		x:         x,
		y:         y,
		width:     width,
		height:    height,
		style:     DefaultGraphStyle(),
		data:      make([]float64, 0),
		binCount:  10,
		minValue:  0,
		maxValue:  100,
		autoRange: true,
	}
}

// SetStyle sets the visual style of the histogram.
func (h *Histogram) SetStyle(style GraphStyle) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.style = style
}

// SetPosition sets the top-left position of the histogram.
func (h *Histogram) SetPosition(x, y float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.x = x
	h.y = y
}

// SetSize sets the width and height of the histogram.
func (h *Histogram) SetSize(width, height float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.width = width
	h.height = height
}

// SetBinCount sets the number of bins for the histogram.
func (h *Histogram) SetBinCount(n int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if n > 0 {
		h.binCount = n
	}
}

// SetRange sets the value range for binning.
// This disables auto-ranging.
func (h *Histogram) SetRange(minVal, maxVal float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.minValue = minVal
	h.maxValue = maxVal
	h.autoRange = false
}

// SetAutoRange enables or disables automatic range detection.
func (h *Histogram) SetAutoRange(enabled bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.autoRange = enabled
}

// SetData replaces all data values in the histogram.
func (h *Histogram) SetData(data []float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.data = make([]float64, len(data))
	copy(h.data, data)
}

// AddValue adds a single value to the histogram data.
func (h *Histogram) AddValue(value float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.data = append(h.data, value)
}

// ClearData removes all data from the histogram.
func (h *Histogram) ClearData() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.data = h.data[:0]
}

// Draw renders the histogram onto the given screen.
func (h *Histogram) Draw(screen *ebiten.Image) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Draw background if enabled
	if h.style.ShowBackground {
		vector.DrawFilledRect(
			screen,
			float32(h.x), float32(h.y),
			float32(h.width), float32(h.height),
			h.style.BackgroundColor,
			false,
		)
	}

	if len(h.data) == 0 || h.binCount <= 0 {
		return
	}

	// Calculate value range
	minVal, maxVal := h.minValue, h.maxValue
	if h.autoRange {
		minVal, maxVal = h.data[0], h.data[0]
		for _, v := range h.data {
			if v < minVal {
				minVal = v
			}
			if v > maxVal {
				maxVal = v
			}
		}
		// Add small padding to include edge values
		padding := (maxVal - minVal) * 0.01
		if padding == 0 {
			padding = 0.5
		}
		minVal -= padding
		maxVal += padding
	}

	valueRange := maxVal - minVal
	if valueRange == 0 {
		valueRange = 1
	}

	// Calculate bin counts
	bins := make([]int, h.binCount)
	binWidth := valueRange / float64(h.binCount)

	for _, v := range h.data {
		binIndex := int((v - minVal) / binWidth)
		if binIndex < 0 {
			binIndex = 0
		}
		if binIndex >= h.binCount {
			binIndex = h.binCount - 1
		}
		bins[binIndex]++
	}

	// Find max bin count for scaling
	maxCount := 0
	for _, count := range bins {
		if count > maxCount {
			maxCount = count
		}
	}

	if maxCount == 0 {
		return
	}

	// Draw bins as bars
	barWidth := h.width / float64(h.binCount)

	for i, count := range bins {
		normalized := float64(count) / float64(maxCount)
		barHeight := normalized * h.height
		barX := h.x + float64(i)*barWidth
		barY := h.y + h.height - barHeight

		vector.DrawFilledRect(
			screen,
			float32(barX), float32(barY),
			float32(barWidth), float32(barHeight),
			h.style.FillColor,
			false,
		)

		// Draw outline
		if h.style.StrokeWidth > 0 {
			vector.StrokeRect(
				screen,
				float32(barX), float32(barY),
				float32(barWidth), float32(barHeight),
				h.style.StrokeWidth,
				h.style.StrokeColor,
				false,
			)
		}
	}
}
