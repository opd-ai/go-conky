//go:build linux

package render

import (
	"testing"
)

func TestCompositorStatusString(t *testing.T) {
	tests := []struct {
		status CompositorStatus
		want   string
	}{
		{CompositorUnknown, "unknown"},
		{CompositorActive, "active"},
		{CompositorInactive, "inactive"},
		{CompositorStatus(99), "unknown"}, // Invalid value
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.status.String()
			if got != tt.want {
				t.Errorf("CompositorStatus(%d).String() = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestDetectCompositor(t *testing.T) {
	// DetectCompositor should return one of the valid statuses
	// We can't predict the result since it depends on the environment
	status := DetectCompositor()

	// Verify it's a valid status
	if status != CompositorUnknown && status != CompositorActive && status != CompositorInactive {
		t.Errorf("DetectCompositor() returned invalid status: %d", status)
	}

	// Just verify it doesn't panic
	t.Logf("DetectCompositor() = %s", status.String())
}

func TestIsWayland(t *testing.T) {
	// IsWayland checks environment variables
	// We can't easily test this without modifying the environment
	// Just verify it doesn't panic and returns a bool
	result := IsWayland()
	t.Logf("IsWayland() = %v", result)
}

func TestCheckTransparencySupport(t *testing.T) {
	tests := []struct {
		name        string
		argbVisual  bool
		transparent bool
		wantEmpty   bool // true if we expect empty string (no warning)
	}{
		{
			name:        "no transparency requested",
			argbVisual:  false,
			transparent: false,
			wantEmpty:   true,
		},
		{
			name:        "argb visual enabled",
			argbVisual:  true,
			transparent: false,
			wantEmpty:   false, // may have warning if no compositor
		},
		{
			name:        "transparent enabled",
			argbVisual:  false,
			transparent: true,
			wantEmpty:   false, // may have warning if no compositor
		},
		{
			name:        "both enabled",
			argbVisual:  true,
			transparent: true,
			wantEmpty:   false, // may have warning if no compositor
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckTransparencySupport(tt.argbVisual, tt.transparent)

			if tt.wantEmpty && result != "" {
				t.Errorf("CheckTransparencySupport(%v, %v) = %q, want empty string",
					tt.argbVisual, tt.transparent, result)
			}

			// For cases where we might get a warning, we just verify it doesn't panic
			// The actual warning depends on system state (compositor running or not)
			if !tt.wantEmpty {
				// Warning may or may not be empty depending on system state
				t.Logf("CheckTransparencySupport(%v, %v) = %q",
					tt.argbVisual, tt.transparent, result)
			}
		})
	}
}

func TestCheckTransparentSupport_NoTransparencyRequested(t *testing.T) {
	// When transparency is not requested, should always return empty
	result := CheckTransparencySupport(false, false)
	if result != "" {
		t.Errorf("CheckTransparencySupport(false, false) should return empty, got %q", result)
	}
}
