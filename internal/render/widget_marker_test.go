package render

import (
	"testing"
)

func TestWidgetTypeString(t *testing.T) {
	tests := []struct {
		wType    WidgetType
		expected string
	}{
		{WidgetTypeBar, "bar"},
		{WidgetTypeGraph, "graph"},
		{WidgetTypeGauge, "gauge"},
		{WidgetType(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.wType.String(); got != tt.expected {
				t.Errorf("WidgetType.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestWidgetMarkerEncode(t *testing.T) {
	tests := []struct {
		name     string
		marker   WidgetMarker
		contains []string
	}{
		{
			name:     "bar 50%",
			marker:   WidgetMarker{Type: WidgetTypeBar, Value: 50, Width: 100, Height: 8},
			contains: []string{"\x00WGT:", "bar", "50.00", "100", "8", "\x00"},
		},
		{
			name:     "graph 75%",
			marker:   WidgetMarker{Type: WidgetTypeGraph, Value: 75.5, Width: 200, Height: 50},
			contains: []string{"\x00WGT:", "graph", "75.50", "200", "50", "\x00"},
		},
		{
			name:     "gauge 100%",
			marker:   WidgetMarker{Type: WidgetTypeGauge, Value: 100, Width: 30, Height: 30},
			contains: []string{"\x00WGT:", "gauge", "100.00", "30", "30", "\x00"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := tt.marker.Encode()
			for _, substr := range tt.contains {
				if !containsString(encoded, substr) {
					t.Errorf("Encode() = %q, missing %q", encoded, substr)
				}
			}
		})
	}
}

func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestDecodeWidgetMarker(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   *WidgetMarker
		wantOK bool
	}{
		{
			name:   "valid bar",
			input:  "\x00WGT:bar:50.00:100:8\x00",
			want:   &WidgetMarker{Type: WidgetTypeBar, Value: 50, Width: 100, Height: 8},
			wantOK: true,
		},
		{
			name:   "valid graph",
			input:  "\x00WGT:graph:75.50:200:50\x00",
			want:   &WidgetMarker{Type: WidgetTypeGraph, Value: 75.5, Width: 200, Height: 50},
			wantOK: true,
		},
		{
			name:   "valid gauge",
			input:  "\x00WGT:gauge:100.00:30:30\x00",
			want:   &WidgetMarker{Type: WidgetTypeGauge, Value: 100, Width: 30, Height: 30},
			wantOK: true,
		},
		{
			name:   "missing prefix",
			input:  "WGT:bar:50:100:8\x00",
			wantOK: false,
		},
		{
			name:   "missing suffix",
			input:  "\x00WGT:bar:50:100:8",
			wantOK: false,
		},
		{
			name:   "invalid type",
			input:  "\x00WGT:invalid:50:100:8\x00",
			wantOK: false,
		},
		{
			name:   "invalid value",
			input:  "\x00WGT:bar:abc:100:8\x00",
			wantOK: false,
		},
		{
			name:   "too few parts",
			input:  "\x00WGT:bar:50:100\x00",
			wantOK: false,
		},
		{
			name:   "empty string",
			input:  "",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DecodeWidgetMarker(tt.input)
			if tt.wantOK {
				if got == nil {
					t.Fatalf("DecodeWidgetMarker() returned nil, want non-nil")
				}
				if got.Type != tt.want.Type {
					t.Errorf("Type = %v, want %v", got.Type, tt.want.Type)
				}
				if got.Value != tt.want.Value {
					t.Errorf("Value = %v, want %v", got.Value, tt.want.Value)
				}
				if got.Width != tt.want.Width {
					t.Errorf("Width = %v, want %v", got.Width, tt.want.Width)
				}
				if got.Height != tt.want.Height {
					t.Errorf("Height = %v, want %v", got.Height, tt.want.Height)
				}
			} else if got != nil {
				t.Errorf("DecodeWidgetMarker() = %v, want nil", got)
			}
		})
	}
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	markers := []WidgetMarker{
		{Type: WidgetTypeBar, Value: 0, Width: 50, Height: 5},
		{Type: WidgetTypeBar, Value: 50.5, Width: 100, Height: 8},
		{Type: WidgetTypeBar, Value: 100, Width: 200, Height: 10},
		{Type: WidgetTypeGraph, Value: 33.33, Width: 150, Height: 40},
		{Type: WidgetTypeGauge, Value: 75, Width: 25, Height: 25},
	}

	for _, original := range markers {
		encoded := original.Encode()
		decoded := DecodeWidgetMarker(encoded)

		if decoded == nil {
			t.Errorf("Round trip failed for %+v: decode returned nil", original)
			continue
		}

		if decoded.Type != original.Type {
			t.Errorf("Type mismatch: got %v, want %v", decoded.Type, original.Type)
		}
		// Allow small floating point differences
		if diff := decoded.Value - original.Value; diff > 0.01 || diff < -0.01 {
			t.Errorf("Value mismatch: got %v, want %v", decoded.Value, original.Value)
		}
		if decoded.Width != original.Width {
			t.Errorf("Width mismatch: got %v, want %v", decoded.Width, original.Width)
		}
		if decoded.Height != original.Height {
			t.Errorf("Height mismatch: got %v, want %v", decoded.Height, original.Height)
		}
	}
}

func TestContainsWidgetMarker(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"plain text", false},
		{"\x00WGT:bar:50:100:8\x00", true},
		{"text with \x00WGT:bar:50:100:8\x00 embedded", true},
		{"no marker here", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := ContainsWidgetMarker(tt.input); got != tt.want {
				t.Errorf("ContainsWidgetMarker() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseWidgetSegments(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantLen int
		checkFn func(t *testing.T, segments []WidgetSegment)
	}{
		{
			name:    "plain text only",
			input:   "just plain text",
			wantLen: 1,
			checkFn: func(t *testing.T, segments []WidgetSegment) {
				if segments[0].IsWidget {
					t.Error("expected text segment, got widget")
				}
				if segments[0].Text != "just plain text" {
					t.Errorf("text = %q, want %q", segments[0].Text, "just plain text")
				}
			},
		},
		{
			name:    "widget only",
			input:   "\x00WGT:bar:50.00:100:8\x00",
			wantLen: 1,
			checkFn: func(t *testing.T, segments []WidgetSegment) {
				if !segments[0].IsWidget {
					t.Error("expected widget segment, got text")
				}
				if segments[0].Widget.Type != WidgetTypeBar {
					t.Errorf("widget type = %v, want bar", segments[0].Widget.Type)
				}
			},
		},
		{
			name:    "text then widget",
			input:   "CPU: \x00WGT:bar:75.00:100:8\x00",
			wantLen: 2,
			checkFn: func(t *testing.T, segments []WidgetSegment) {
				if segments[0].IsWidget || segments[0].Text != "CPU: " {
					t.Errorf("first segment wrong: isWidget=%v, text=%q", segments[0].IsWidget, segments[0].Text)
				}
				if !segments[1].IsWidget || segments[1].Widget.Value != 75 {
					t.Errorf("second segment wrong: isWidget=%v, value=%v", segments[1].IsWidget, segments[1].Widget)
				}
			},
		},
		{
			name:    "widget then text",
			input:   "\x00WGT:bar:50.00:100:8\x00 done",
			wantLen: 2,
			checkFn: func(t *testing.T, segments []WidgetSegment) {
				if !segments[0].IsWidget {
					t.Error("expected widget first")
				}
				if segments[1].IsWidget || segments[1].Text != " done" {
					t.Errorf("second segment wrong: %+v", segments[1])
				}
			},
		},
		{
			name:    "text widget text",
			input:   "Mem: \x00WGT:bar:80.00:100:8\x00 used",
			wantLen: 3,
			checkFn: func(t *testing.T, segments []WidgetSegment) {
				if segments[0].Text != "Mem: " {
					t.Errorf("first = %q", segments[0].Text)
				}
				if !segments[1].IsWidget {
					t.Error("middle should be widget")
				}
				if segments[2].Text != " used" {
					t.Errorf("last = %q", segments[2].Text)
				}
			},
		},
		{
			name:    "multiple widgets",
			input:   "CPU: \x00WGT:bar:50.00:100:8\x00 Mem: \x00WGT:bar:75.00:100:8\x00",
			wantLen: 4,
			checkFn: func(t *testing.T, segments []WidgetSegment) {
				if !segments[1].IsWidget || segments[1].Widget.Value != 50 {
					t.Error("first widget wrong")
				}
				if !segments[3].IsWidget || segments[3].Widget.Value != 75 {
					t.Error("second widget wrong")
				}
			},
		},
		{
			name:    "empty string",
			input:   "",
			wantLen: 1,
			checkFn: func(t *testing.T, segments []WidgetSegment) {
				if segments[0].IsWidget {
					t.Error("expected text segment for empty string")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			segments := ParseWidgetSegments(tt.input)
			if len(segments) != tt.wantLen {
				t.Errorf("len(segments) = %d, want %d", len(segments), tt.wantLen)
				for i, s := range segments {
					t.Logf("  segment[%d]: isWidget=%v, text=%q, widget=%+v", i, s.IsWidget, s.Text, s.Widget)
				}
				return
			}
			if tt.checkFn != nil {
				tt.checkFn(t, segments)
			}
		})
	}
}

func TestEncodeBarMarker(t *testing.T) {
	marker := EncodeBarMarker(50, 100, 8)
	decoded := DecodeWidgetMarker(marker)
	if decoded == nil {
		t.Fatal("EncodeBarMarker produced invalid marker")
	}
	if decoded.Type != WidgetTypeBar {
		t.Errorf("Type = %v, want bar", decoded.Type)
	}
	if decoded.Value != 50 {
		t.Errorf("Value = %v, want 50", decoded.Value)
	}
}

func TestEncodeGraphMarker(t *testing.T) {
	marker := EncodeGraphMarker(75, 200, 50)
	decoded := DecodeWidgetMarker(marker)
	if decoded == nil {
		t.Fatal("EncodeGraphMarker produced invalid marker")
	}
	if decoded.Type != WidgetTypeGraph {
		t.Errorf("Type = %v, want graph", decoded.Type)
	}
}

func TestEncodeGaugeMarker(t *testing.T) {
	marker := EncodeGaugeMarker(100, 30, 30)
	decoded := DecodeWidgetMarker(marker)
	if decoded == nil {
		t.Fatal("EncodeGaugeMarker produced invalid marker")
	}
	if decoded.Type != WidgetTypeGauge {
		t.Errorf("Type = %v, want gauge", decoded.Type)
	}
}

func TestEncodeGraphMarkerWithID(t *testing.T) {
	marker := EncodeGraphMarkerWithID(75, 200, 50, "cpu")
	decoded := DecodeWidgetMarker(marker)
	if decoded == nil {
		t.Fatal("EncodeGraphMarkerWithID produced invalid marker")
	}
	if decoded.Type != WidgetTypeGraph {
		t.Errorf("Type = %v, want graph", decoded.Type)
	}
	if decoded.ID != "cpu" {
		t.Errorf("ID = %q, want %q", decoded.ID, "cpu")
	}
	if decoded.Value != 75 {
		t.Errorf("Value = %v, want 75", decoded.Value)
	}
	if decoded.Width != 200 {
		t.Errorf("Width = %v, want 200", decoded.Width)
	}
	if decoded.Height != 50 {
		t.Errorf("Height = %v, want 50", decoded.Height)
	}
}

func TestDecodeWidgetMarkerWithID(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		wantID string
		wantOK bool
	}{
		{
			name:   "graph with ID",
			input:  "\x00WGT:graph:50.00:100:20:cpu\x00",
			wantID: "cpu",
			wantOK: true,
		},
		{
			name:   "graph with network ID",
			input:  "\x00WGT:graph:25.00:150:30:net_eth0_down\x00",
			wantID: "net_eth0_down",
			wantOK: true,
		},
		{
			name:   "bar without ID (legacy)",
			input:  "\x00WGT:bar:50.00:100:8\x00",
			wantID: "",
			wantOK: true,
		},
		{
			name:   "graph without ID (legacy)",
			input:  "\x00WGT:graph:75.50:200:50\x00",
			wantID: "",
			wantOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DecodeWidgetMarker(tt.input)
			if tt.wantOK {
				if got == nil {
					t.Fatal("DecodeWidgetMarker returned nil, want valid marker")
				}
				if got.ID != tt.wantID {
					t.Errorf("ID = %q, want %q", got.ID, tt.wantID)
				}
			} else {
				if got != nil {
					t.Errorf("DecodeWidgetMarker returned non-nil, want nil")
				}
			}
		})
	}
}

func TestWidgetMarkerEncodeWithID(t *testing.T) {
	marker := WidgetMarker{
		Type:   WidgetTypeGraph,
		Value:  65.5,
		Width:  100,
		Height: 20,
		ID:     "mem",
	}
	encoded := marker.Encode()
	
	// Should contain all parts including ID
	if !containsString(encoded, "mem") {
		t.Errorf("Encode() = %q, missing ID 'mem'", encoded)
	}
	
	// Decode and verify round-trip
	decoded := DecodeWidgetMarker(encoded)
	if decoded == nil {
		t.Fatal("Failed to decode marker with ID")
	}
	if decoded.ID != "mem" {
		t.Errorf("Round-trip ID = %q, want %q", decoded.ID, "mem")
	}
}

// Image marker tests

func TestImageMarkerEncode(t *testing.T) {
	tests := []struct {
		name     string
		marker   ImageMarker
		contains []string
	}{
		{
			name:     "basic image",
			marker:   ImageMarker{Path: "/path/to/image.png", Width: 100, Height: 50, X: -1, Y: -1, NoCache: false},
			contains: []string{"\x00IMG:", "/path/to/image.png", "100", "50", "-1", "0", "\x00"},
		},
		{
			name:     "absolute positioned",
			marker:   ImageMarker{Path: "/img.jpg", Width: 200, Height: 100, X: 10, Y: 20, NoCache: false},
			contains: []string{"\x00IMG:", "/img.jpg", "200", "100", "10", "20", "\x00"},
		},
		{
			name:     "no cache",
			marker:   ImageMarker{Path: "/dynamic.png", Width: 50, Height: 50, X: -1, Y: -1, NoCache: true},
			contains: []string{"\x00IMG:", "/dynamic.png", ":1\x00"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := tt.marker.Encode()
			for _, substr := range tt.contains {
				if !containsString(encoded, substr) {
					t.Errorf("Encode() = %q, missing %q", encoded, substr)
				}
			}
		})
	}
}

func TestDecodeImageMarker(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   *ImageMarker
		wantOK bool
	}{
		{
			name:   "valid image inline",
			input:  "\x00IMG:/path/to/image.png:100:50:-1:-1:0\x00",
			want:   &ImageMarker{Path: "/path/to/image.png", Width: 100, Height: 50, X: -1, Y: -1, NoCache: false},
			wantOK: true,
		},
		{
			name:   "valid image absolute position",
			input:  "\x00IMG:/img.jpg:200:100:10:20:0\x00",
			want:   &ImageMarker{Path: "/img.jpg", Width: 200, Height: 100, X: 10, Y: 20, NoCache: false},
			wantOK: true,
		},
		{
			name:   "valid image no cache",
			input:  "\x00IMG:/dynamic.png:50:50:-1:-1:1\x00",
			want:   &ImageMarker{Path: "/dynamic.png", Width: 50, Height: 50, X: -1, Y: -1, NoCache: true},
			wantOK: true,
		},
		{
			name:   "path with colon",
			input:  "\x00IMG:C:/Users/image.png:100:50:-1:-1:0\x00",
			want:   &ImageMarker{Path: "C:/Users/image.png", Width: 100, Height: 50, X: -1, Y: -1, NoCache: false},
			wantOK: true,
		},
		{
			name:   "missing prefix",
			input:  "IMG:/path.png:100:50:-1:-1:0\x00",
			wantOK: false,
		},
		{
			name:   "missing suffix",
			input:  "\x00IMG:/path.png:100:50:-1:-1:0",
			wantOK: false,
		},
		{
			name:   "too few parts",
			input:  "\x00IMG:/path.png:100:50\x00",
			wantOK: false,
		},
		{
			name:   "empty string",
			input:  "",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DecodeImageMarker(tt.input)
			if tt.wantOK {
				if got == nil {
					t.Fatalf("DecodeImageMarker() returned nil, want non-nil")
				}
				if got.Path != tt.want.Path {
					t.Errorf("Path = %v, want %v", got.Path, tt.want.Path)
				}
				if got.Width != tt.want.Width {
					t.Errorf("Width = %v, want %v", got.Width, tt.want.Width)
				}
				if got.Height != tt.want.Height {
					t.Errorf("Height = %v, want %v", got.Height, tt.want.Height)
				}
				if got.X != tt.want.X {
					t.Errorf("X = %v, want %v", got.X, tt.want.X)
				}
				if got.Y != tt.want.Y {
					t.Errorf("Y = %v, want %v", got.Y, tt.want.Y)
				}
				if got.NoCache != tt.want.NoCache {
					t.Errorf("NoCache = %v, want %v", got.NoCache, tt.want.NoCache)
				}
			} else if got != nil {
				t.Errorf("DecodeImageMarker() = %v, want nil", got)
			}
		})
	}
}

func TestImageMarkerRoundTrip(t *testing.T) {
	markers := []ImageMarker{
		{Path: "/simple.png", Width: 100, Height: 50, X: -1, Y: -1, NoCache: false},
		{Path: "/path/with/dirs/image.jpg", Width: 200, Height: 100, X: 10, Y: 20, NoCache: false},
		{Path: "/dynamic.gif", Width: 50, Height: 50, X: -1, Y: -1, NoCache: true},
		{Path: "C:/Windows/path.png", Width: 0, Height: 0, X: 100, Y: 200, NoCache: false},
	}

	for _, original := range markers {
		encoded := original.Encode()
		decoded := DecodeImageMarker(encoded)

		if decoded == nil {
			t.Errorf("Round trip failed for %+v: decode returned nil", original)
			continue
		}

		if decoded.Path != original.Path {
			t.Errorf("Path mismatch: got %v, want %v", decoded.Path, original.Path)
		}
		if decoded.Width != original.Width {
			t.Errorf("Width mismatch: got %v, want %v", decoded.Width, original.Width)
		}
		if decoded.Height != original.Height {
			t.Errorf("Height mismatch: got %v, want %v", decoded.Height, original.Height)
		}
		if decoded.X != original.X {
			t.Errorf("X mismatch: got %v, want %v", decoded.X, original.X)
		}
		if decoded.Y != original.Y {
			t.Errorf("Y mismatch: got %v, want %v", decoded.Y, original.Y)
		}
		if decoded.NoCache != original.NoCache {
			t.Errorf("NoCache mismatch: got %v, want %v", decoded.NoCache, original.NoCache)
		}
	}
}

func TestContainsImageMarker(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"plain text", false},
		{"\x00IMG:/path.png:100:50:-1:-1:0\x00", true},
		{"text with \x00IMG:/path.png:100:50:-1:-1:0\x00 embedded", true},
		{"no marker here", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := ContainsImageMarker(tt.input); got != tt.want {
				t.Errorf("ContainsImageMarker() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEncodeImageMarker(t *testing.T) {
	marker := EncodeImageMarker("/test/image.png", 100, 50, 10, 20, true)
	decoded := DecodeImageMarker(marker)
	if decoded == nil {
		t.Fatal("EncodeImageMarker produced invalid marker")
	}
	if decoded.Path != "/test/image.png" {
		t.Errorf("Path = %v, want /test/image.png", decoded.Path)
	}
	if decoded.Width != 100 {
		t.Errorf("Width = %v, want 100", decoded.Width)
	}
	if decoded.Height != 50 {
		t.Errorf("Height = %v, want 50", decoded.Height)
	}
	if decoded.X != 10 {
		t.Errorf("X = %v, want 10", decoded.X)
	}
	if decoded.Y != 20 {
		t.Errorf("Y = %v, want 20", decoded.Y)
	}
	if !decoded.NoCache {
		t.Errorf("NoCache = %v, want true", decoded.NoCache)
	}
}

func TestParseWidgetSegmentsWithImages(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantLen int
		checkFn func(t *testing.T, segments []WidgetSegment)
	}{
		{
			name:    "image only",
			input:   "\x00IMG:/path.png:100:50:-1:-1:0\x00",
			wantLen: 1,
			checkFn: func(t *testing.T, segments []WidgetSegment) {
				if !segments[0].IsImage {
					t.Error("expected image segment")
				}
				if segments[0].Image.Path != "/path.png" {
					t.Errorf("path = %q, want /path.png", segments[0].Image.Path)
				}
			},
		},
		{
			name:    "text then image",
			input:   "Icon: \x00IMG:/icon.png:16:16:-1:-1:0\x00",
			wantLen: 2,
			checkFn: func(t *testing.T, segments []WidgetSegment) {
				if segments[0].IsImage || segments[0].Text != "Icon: " {
					t.Errorf("first segment wrong: isImage=%v, text=%q", segments[0].IsImage, segments[0].Text)
				}
				if !segments[1].IsImage || segments[1].Image.Path != "/icon.png" {
					t.Error("second segment should be image")
				}
			},
		},
		{
			name:    "mixed widget and image",
			input:   "CPU: \x00WGT:bar:50.00:100:8\x00 \x00IMG:/icon.png:16:16:-1:-1:0\x00",
			wantLen: 4,
			checkFn: func(t *testing.T, segments []WidgetSegment) {
				if !segments[1].IsWidget {
					t.Error("expected widget at position 1")
				}
				if !segments[3].IsImage {
					t.Error("expected image at position 3")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			segments := ParseWidgetSegments(tt.input)
			if len(segments) != tt.wantLen {
				t.Errorf("len(segments) = %d, want %d", len(segments), tt.wantLen)
				for i, s := range segments {
					t.Logf("  segment[%d]: isWidget=%v, isImage=%v, text=%q", i, s.IsWidget, s.IsImage, s.Text)
				}
				return
			}
			if tt.checkFn != nil {
				tt.checkFn(t, segments)
			}
		})
	}
}

func TestWidgetTypeImageString(t *testing.T) {
	if got := WidgetTypeImage.String(); got != "image" {
		t.Errorf("WidgetTypeImage.String() = %q, want %q", got, "image")
	}
}
