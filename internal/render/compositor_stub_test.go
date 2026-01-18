//go:build !linux

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

func TestDetectCompositor_NonLinux(t *testing.T) {
	// On non-Linux platforms, DetectCompositor should always return Active
	// because Windows (DWM) and macOS always have compositing
	status := DetectCompositor()

	if status != CompositorActive {
		t.Errorf("DetectCompositor() = %v, want CompositorActive on non-Linux platform", status)
	}
}

func TestIsWayland_NonLinux(t *testing.T) {
	// On non-Linux platforms, IsWayland should always return false
	if IsWayland() {
		t.Error("IsWayland() = true, want false on non-Linux platform")
	}
}

func TestCheckTransparencySupport_NonLinux(t *testing.T) {
	// On non-Linux platforms, CheckTransparencySupport should always return empty
	// because Windows and macOS always have compositing
	tests := []struct {
		name        string
		argbVisual  bool
		transparent bool
	}{
		{"no transparency", false, false},
		{"argb visual only", true, false},
		{"transparent only", false, true},
		{"both enabled", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckTransparencySupport(tt.argbVisual, tt.transparent)
			if result != "" {
				t.Errorf("CheckTransparencySupport(%v, %v) = %q, want empty on non-Linux",
					tt.argbVisual, tt.transparent, result)
			}
		})
	}
}
