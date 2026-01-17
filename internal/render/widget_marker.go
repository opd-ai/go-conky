// Package render provides Ebiten-based rendering capabilities for conky-go.
// This file implements the widget marker system for embedding graphical widgets
// (progress bars, graphs) inline with text content.
package render

import (
	"fmt"
	"strconv"
	"strings"
)

// WidgetType represents the type of inline widget.
type WidgetType int

const (
	// WidgetTypeBar represents a horizontal progress bar.
	WidgetTypeBar WidgetType = iota
	// WidgetTypeGraph represents a line/area graph.
	WidgetTypeGraph
	// WidgetTypeGauge represents a circular gauge.
	WidgetTypeGauge
)

// String returns the string representation of the widget type.
func (wt WidgetType) String() string {
	switch wt {
	case WidgetTypeBar:
		return "bar"
	case WidgetTypeGraph:
		return "graph"
	case WidgetTypeGauge:
		return "gauge"
	default:
		return "unknown"
	}
}

// WidgetMarker encodes parameters for rendering an inline graphical widget.
// It is embedded in text content and parsed by the rendering layer.
type WidgetMarker struct {
	// Type is the kind of widget to render.
	Type WidgetType
	// Value is the current value (0-100 for percentage-based widgets).
	Value float64
	// Width is the widget width in pixels.
	Width float64
	// Height is the widget height in pixels.
	Height float64
}

// markerPrefix and markerSuffix delimit widget markers in text.
// Using null bytes ensures they won't appear in normal text.
const (
	markerPrefix = "\x00WGT:"
	markerSuffix = "\x00"
)

// Encode returns the string representation of the widget marker.
// Format: \x00WGT:type:value:width:height\x00
func (wm WidgetMarker) Encode() string {
	return fmt.Sprintf("%s%s:%.2f:%.0f:%.0f%s",
		markerPrefix,
		wm.Type.String(),
		wm.Value,
		wm.Width,
		wm.Height,
		markerSuffix,
	)
}

// DecodeWidgetMarker parses a widget marker string.
// Returns nil if the string is not a valid marker.
func DecodeWidgetMarker(s string) *WidgetMarker {
	if !strings.HasPrefix(s, markerPrefix) || !strings.HasSuffix(s, markerSuffix) {
		return nil
	}

	// Extract content between prefix and suffix
	content := s[len(markerPrefix) : len(s)-len(markerSuffix)]
	parts := strings.Split(content, ":")
	if len(parts) != 4 {
		return nil
	}

	// Parse widget type
	var wType WidgetType
	switch parts[0] {
	case "bar":
		wType = WidgetTypeBar
	case "graph":
		wType = WidgetTypeGraph
	case "gauge":
		wType = WidgetTypeGauge
	default:
		return nil
	}

	// Parse value
	value, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return nil
	}

	// Parse width
	width, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return nil
	}

	// Parse height
	height, err := strconv.ParseFloat(parts[3], 64)
	if err != nil {
		return nil
	}

	return &WidgetMarker{
		Type:   wType,
		Value:  value,
		Width:  width,
		Height: height,
	}
}

// ContainsWidgetMarker checks if a string contains any widget markers.
func ContainsWidgetMarker(s string) bool {
	return strings.Contains(s, markerPrefix)
}

// WidgetSegment represents either a text segment or a widget marker.
type WidgetSegment struct {
	// IsWidget is true if this segment is a widget marker.
	IsWidget bool
	// Text contains the text content (if IsWidget is false).
	Text string
	// Widget contains the widget marker (if IsWidget is true).
	Widget *WidgetMarker
}

// ParseWidgetSegments splits a string into text segments and widget markers.
func ParseWidgetSegments(s string) []WidgetSegment {
	if !ContainsWidgetMarker(s) {
		return []WidgetSegment{{IsWidget: false, Text: s}}
	}

	var segments []WidgetSegment
	remaining := s

	for len(remaining) > 0 {
		// Find the next marker
		startIdx := strings.Index(remaining, markerPrefix)
		if startIdx == -1 {
			// No more markers, rest is text
			if len(remaining) > 0 {
				segments = append(segments, WidgetSegment{IsWidget: false, Text: remaining})
			}
			break
		}

		// Add text before the marker
		if startIdx > 0 {
			segments = append(segments, WidgetSegment{IsWidget: false, Text: remaining[:startIdx]})
		}

		// Find the end of the marker
		remaining = remaining[startIdx:]
		endIdx := strings.Index(remaining[1:], markerSuffix) // Skip first char to avoid matching prefix
		if endIdx == -1 {
			// Malformed marker, treat rest as text
			segments = append(segments, WidgetSegment{IsWidget: false, Text: remaining})
			break
		}
		endIdx += 2 // Adjust for the skipped char and include the suffix

		// Parse the marker
		markerStr := remaining[:endIdx]
		marker := DecodeWidgetMarker(markerStr)
		if marker != nil {
			segments = append(segments, WidgetSegment{IsWidget: true, Widget: marker})
		} else {
			// Malformed marker, treat as text
			segments = append(segments, WidgetSegment{IsWidget: false, Text: markerStr})
		}

		remaining = remaining[endIdx:]
	}

	return segments
}

// EncodeBarMarker creates a widget marker for a horizontal bar.
func EncodeBarMarker(value, width, height float64) string {
	return WidgetMarker{
		Type:   WidgetTypeBar,
		Value:  value,
		Width:  width,
		Height: height,
	}.Encode()
}

// EncodeGraphMarker creates a widget marker for a graph.
func EncodeGraphMarker(value, width, height float64) string {
	return WidgetMarker{
		Type:   WidgetTypeGraph,
		Value:  value,
		Width:  width,
		Height: height,
	}.Encode()
}

// EncodeGaugeMarker creates a widget marker for a gauge.
func EncodeGaugeMarker(value, width, height float64) string {
	return WidgetMarker{
		Type:   WidgetTypeGauge,
		Value:  value,
		Width:  width,
		Height: height,
	}.Encode()
}
