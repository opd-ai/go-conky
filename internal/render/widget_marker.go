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
	// WidgetTypeImage represents an embedded image.
	WidgetTypeImage
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
	case WidgetTypeImage:
		return "image"
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
	// ID identifies the data source for historical tracking (e.g., "cpu", "mem", "net_eth0_down").
	// Used by graph widgets to maintain separate time-series histories.
	ID string
}

// markerPrefix and markerSuffix delimit widget markers in text.
// Using null bytes ensures they won't appear in normal text.
const (
	markerPrefix = "\x00WGT:"
	markerSuffix = "\x00"
)

// Encode returns the string representation of the widget marker.
// Format: \x00WGT:type:value:width:height\x00 or \x00WGT:type:value:width:height:id\x00
func (wm WidgetMarker) Encode() string {
	if wm.ID == "" {
		return fmt.Sprintf("%s%s:%.2f:%.0f:%.0f%s",
			markerPrefix,
			wm.Type.String(),
			wm.Value,
			wm.Width,
			wm.Height,
			markerSuffix,
		)
	}
	return fmt.Sprintf("%s%s:%.2f:%.0f:%.0f:%s%s",
		markerPrefix,
		wm.Type.String(),
		wm.Value,
		wm.Width,
		wm.Height,
		wm.ID,
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
	// Support both 4-part (legacy) and 5-part (with ID) formats
	if len(parts) < 4 || len(parts) > 5 {
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

	// Parse optional ID
	var id string
	if len(parts) == 5 {
		id = parts[4]
	}

	return &WidgetMarker{
		Type:   wType,
		Value:  value,
		Width:  width,
		Height: height,
		ID:     id,
	}
}

// ContainsWidgetMarker checks if a string contains any widget markers.
func ContainsWidgetMarker(s string) bool {
	return strings.Contains(s, markerPrefix)
}

// WidgetSegment represents either a text segment, a widget marker, or an image marker.
type WidgetSegment struct {
	// IsWidget is true if this segment is a widget marker.
	IsWidget bool
	// IsImage is true if this segment is an image marker.
	IsImage bool
	// Text contains the text content (if IsWidget and IsImage are false).
	Text string
	// Widget contains the widget marker (if IsWidget is true).
	Widget *WidgetMarker
	// Image contains the image marker (if IsImage is true).
	Image *ImageMarker
}

// ParseWidgetSegments splits a string into text segments, widget markers, and image markers.
func ParseWidgetSegments(s string) []WidgetSegment {
	if !ContainsWidgetMarker(s) && !ContainsImageMarker(s) {
		return []WidgetSegment{{IsWidget: false, IsImage: false, Text: s}}
	}

	var segments []WidgetSegment
	remaining := s

	for remaining != "" {
		// Find the next widget marker
		widgetIdx := strings.Index(remaining, markerPrefix)
		// Find the next image marker
		imageIdx := strings.Index(remaining, imageMarkerPrefix)

		// If no more markers, rest is text
		if widgetIdx == -1 && imageIdx == -1 {
			if remaining != "" {
				segments = append(segments, WidgetSegment{IsWidget: false, IsImage: false, Text: remaining})
			}
			break
		}

		// Determine which marker comes first
		var startIdx int
		var isImageMarker bool
		switch {
		case widgetIdx == -1:
			startIdx = imageIdx
			isImageMarker = true
		case imageIdx == -1:
			startIdx = widgetIdx
			isImageMarker = false
		case imageIdx < widgetIdx:
			startIdx = imageIdx
			isImageMarker = true
		default:
			startIdx = widgetIdx
			isImageMarker = false
		}

		// Add text before the marker
		if startIdx > 0 {
			segments = append(segments, WidgetSegment{IsWidget: false, IsImage: false, Text: remaining[:startIdx]})
		}

		// Find the end of the marker
		remaining = remaining[startIdx:]
		endIdx := strings.Index(remaining[1:], markerSuffix) // Skip first char to avoid matching prefix
		if endIdx == -1 {
			// Malformed marker, treat rest as text
			segments = append(segments, WidgetSegment{IsWidget: false, IsImage: false, Text: remaining})
			break
		}
		endIdx += 2 // Adjust for the skipped char and include the suffix

		// Parse the marker
		markerStr := remaining[:endIdx]
		if isImageMarker {
			imgMarker := DecodeImageMarker(markerStr)
			if imgMarker != nil {
				segments = append(segments, WidgetSegment{IsImage: true, Image: imgMarker})
			} else {
				// Malformed marker, treat as text
				segments = append(segments, WidgetSegment{IsWidget: false, IsImage: false, Text: markerStr})
			}
		} else {
			marker := DecodeWidgetMarker(markerStr)
			if marker != nil {
				segments = append(segments, WidgetSegment{IsWidget: true, Widget: marker})
			} else {
				// Malformed marker, treat as text
				segments = append(segments, WidgetSegment{IsWidget: false, IsImage: false, Text: markerStr})
			}
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

// EncodeGraphMarkerWithID creates a widget marker for a graph with historical tracking.
// The id parameter identifies the data source for time-series accumulation.
func EncodeGraphMarkerWithID(value, width, height float64, id string) string {
	return WidgetMarker{
		Type:   WidgetTypeGraph,
		Value:  value,
		Width:  width,
		Height: height,
		ID:     id,
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

// ImageMarker encodes parameters for rendering an inline image.
// It is embedded in text content and parsed by the rendering layer.
type ImageMarker struct {
	// Path is the file path to the image.
	Path string
	// Width is the display width in pixels (0 = original width).
	Width float64
	// Height is the display height in pixels (0 = original height).
	Height float64
	// X is the absolute X position (-1 = inline with text).
	X float64
	// Y is the absolute Y position (-1 = inline with text).
	Y float64
	// NoCache disables image caching (for dynamic images).
	NoCache bool
}

// imageMarkerPrefix delimits image markers in text.
const imageMarkerPrefix = "\x00IMG:"

// Encode returns the string representation of the image marker.
// Format: \x00IMG:path:width:height:x:y:nocache\x00
func (im ImageMarker) Encode() string {
	noCache := 0
	if im.NoCache {
		noCache = 1
	}
	return fmt.Sprintf("%s%s:%.0f:%.0f:%.0f:%.0f:%d%s",
		imageMarkerPrefix,
		im.Path,
		im.Width,
		im.Height,
		im.X,
		im.Y,
		noCache,
		markerSuffix,
	)
}

// DecodeImageMarker parses an image marker string.
// Returns nil if the string is not a valid image marker.
func DecodeImageMarker(s string) *ImageMarker {
	if !strings.HasPrefix(s, imageMarkerPrefix) || !strings.HasSuffix(s, markerSuffix) {
		return nil
	}

	// Extract content between prefix and suffix
	content := s[len(imageMarkerPrefix) : len(s)-len(markerSuffix)]

	// Split into parts: path:width:height:x:y:nocache
	// Path may contain colons, so we need to parse carefully from the end
	parts := strings.Split(content, ":")
	if len(parts) < 6 {
		return nil
	}

	// Parse from the end where we know the format
	noCache, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return nil
	}
	y, err := strconv.ParseFloat(parts[len(parts)-2], 64)
	if err != nil {
		return nil
	}
	x, err := strconv.ParseFloat(parts[len(parts)-3], 64)
	if err != nil {
		return nil
	}
	height, err := strconv.ParseFloat(parts[len(parts)-4], 64)
	if err != nil {
		return nil
	}
	width, err := strconv.ParseFloat(parts[len(parts)-5], 64)
	if err != nil {
		return nil
	}

	// Path is everything before the last 5 parts (may contain colons)
	pathParts := parts[:len(parts)-5]
	path := strings.Join(pathParts, ":")

	return &ImageMarker{
		Path:    path,
		Width:   width,
		Height:  height,
		X:       x,
		Y:       y,
		NoCache: noCache == 1,
	}
}

// ContainsImageMarker checks if a string contains any image markers.
func ContainsImageMarker(s string) bool {
	return strings.Contains(s, imageMarkerPrefix)
}

// EncodeImageMarker creates an image marker string.
// Use x=-1 and y=-1 for inline positioning.
func EncodeImageMarker(path string, width, height, x, y float64, noCache bool) string {
	return ImageMarker{
		Path:    path,
		Width:   width,
		Height:  height,
		X:       x,
		Y:       y,
		NoCache: noCache,
	}.Encode()
}
