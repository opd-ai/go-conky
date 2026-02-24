//go:build linux

package render

import (
	"testing"

	"github.com/jezek/xgb/xproto"
)

func TestApplyWindowHints_NoHints(t *testing.T) {
	// When both hints are false, should return nil immediately
	err := ApplyWindowHints(false, false)
	if err != nil {
		t.Errorf("ApplyWindowHints(false, false) = %v, want nil", err)
	}
}

func TestApplyWindowHints_SkipTaskbar(t *testing.T) {
	// Test with skip_taskbar enabled - should not panic
	// Actual effect depends on X11 being available
	err := ApplyWindowHints(true, false)
	// Should not return error even if X11 not available (silently ignores)
	if err != nil {
		t.Errorf("ApplyWindowHints(true, false) = %v, want nil", err)
	}
}

func TestApplyWindowHints_SkipPager(t *testing.T) {
	// Test with skip_pager enabled - should not panic
	// Actual effect depends on X11 being available
	err := ApplyWindowHints(false, true)
	// Should not return error even if X11 not available (silently ignores)
	if err != nil {
		t.Errorf("ApplyWindowHints(false, true) = %v, want nil", err)
	}
}

func TestApplyWindowHints_Both(t *testing.T) {
	// Test with both hints enabled - should not panic
	err := ApplyWindowHints(true, true)
	// Should not return error even if X11 not available (silently ignores)
	if err != nil {
		t.Errorf("ApplyWindowHints(true, true) = %v, want nil", err)
	}
}

func TestWindowHintApplier_Close(t *testing.T) {
	// Create a fresh applier and close it
	applier := &WindowHintApplier{
		atoms: make(map[string]xproto.Atom),
	}

	// Should not panic when closing without initialization
	applier.Close()

	// Should be safe to close multiple times
	applier.Close()

	// Verify state is reset
	if applier.initDone {
		t.Error("initDone should be false after Close()")
	}
	if applier.conn != nil {
		t.Error("conn should be nil after Close()")
	}
}

func TestCloseWindowHints(t *testing.T) {
	// Should not panic
	CloseWindowHints()
}
