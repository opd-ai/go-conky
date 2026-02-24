package monitor

import (
	"errors"
	"fmt"
	"testing"
)

func TestComponentError(t *testing.T) {
	originalErr := errors.New("disk read failed")
	ce := NewComponentError(ErrorSourceDiskIO, false, originalErr)

	// Test Error() output format
	expected := "diskio: disk read failed"
	if ce.Error() != expected {
		t.Errorf("Error() = %q, want %q", ce.Error(), expected)
	}

	// Test platform error format
	pce := NewComponentError(ErrorSourceCPU, true, originalErr)
	expectedPlatform := "cpu (platform): disk read failed"
	if pce.Error() != expectedPlatform {
		t.Errorf("Error() = %q, want %q", pce.Error(), expectedPlatform)
	}

	// Test Unwrap
	if !errors.Is(ce, originalErr) {
		t.Error("errors.Is() should return true for wrapped error")
	}
}

func TestComponentError_Unwrap(t *testing.T) {
	// Test that errors.As works through ComponentError
	originalErr := errors.New("original")
	compErr := NewComponentError(ErrorSourceMemory, false, originalErr)

	// errors.Is should work through Unwrap
	if !errors.Is(compErr, originalErr) {
		t.Error("errors.Is() should find wrapped error")
	}

	// Test nested wrapping
	wrappedCompErr := fmt.Errorf("context: %w", compErr)
	if !errors.Is(wrappedCompErr, originalErr) {
		t.Error("errors.Is() should find deeply wrapped error")
	}
}

func TestUpdateError(t *testing.T) {
	errs := []*ComponentError{
		NewComponentError(ErrorSourceCPU, false, errors.New("cpu failed")),
		NewComponentError(ErrorSourceMemory, true, errors.New("memory failed")),
	}
	ue := &UpdateError{Errors: errs}

	// Test Error() output
	errStr := ue.Error()
	if len(errStr) == 0 {
		t.Error("Error() returned empty string")
	}
	if errStr[:12] != "update error" {
		t.Errorf("Error() should start with 'update error', got %q", errStr[:12])
	}

	// Test single error case
	singleUE := &UpdateError{Errors: []*ComponentError{
		NewComponentError(ErrorSourceCPU, false, errors.New("cpu failed")),
	}}
	singleErr := singleUE.Error()
	if singleErr != "update error: cpu: cpu failed" {
		t.Errorf("Single error format = %q, want 'update error: cpu: cpu failed'", singleErr)
	}

	// Test empty error case
	emptyUE := &UpdateError{Errors: nil}
	if emptyUE.Error() != "no errors" {
		t.Errorf("Empty error format = %q, want 'no errors'", emptyUE.Error())
	}
}

func TestUpdateError_HasSource(t *testing.T) {
	ue := &UpdateError{Errors: []*ComponentError{
		NewComponentError(ErrorSourceCPU, false, errors.New("cpu failed")),
		NewComponentError(ErrorSourceNetwork, true, errors.New("network failed")),
	}}

	if !ue.HasSource(ErrorSourceCPU) {
		t.Error("HasSource(CPU) should return true")
	}
	if !ue.HasSource(ErrorSourceNetwork) {
		t.Error("HasSource(Network) should return true")
	}
	if ue.HasSource(ErrorSourceMemory) {
		t.Error("HasSource(Memory) should return false")
	}
}

func TestUpdateError_BySource(t *testing.T) {
	ue := &UpdateError{Errors: []*ComponentError{
		NewComponentError(ErrorSourceCPU, false, errors.New("cpu failed 1")),
		NewComponentError(ErrorSourceMemory, false, errors.New("memory failed")),
		NewComponentError(ErrorSourceCPU, true, errors.New("cpu failed 2")),
	}}

	cpuErrs := ue.BySource(ErrorSourceCPU)
	if len(cpuErrs) != 2 {
		t.Errorf("BySource(CPU) returned %d errors, want 2", len(cpuErrs))
	}

	memErrs := ue.BySource(ErrorSourceMemory)
	if len(memErrs) != 1 {
		t.Errorf("BySource(Memory) returned %d errors, want 1", len(memErrs))
	}

	diskErrs := ue.BySource(ErrorSourceDiskIO)
	if len(diskErrs) != 0 {
		t.Errorf("BySource(DiskIO) returned %d errors, want 0", len(diskErrs))
	}
}

func TestUpdateError_PlatformAndFallbackErrors(t *testing.T) {
	ue := &UpdateError{Errors: []*ComponentError{
		NewComponentError(ErrorSourceCPU, true, errors.New("cpu platform failed")),
		NewComponentError(ErrorSourceCPU, false, errors.New("cpu linux failed")),
		NewComponentError(ErrorSourceMemory, true, errors.New("memory platform failed")),
	}}

	platformErrs := ue.PlatformErrors()
	if len(platformErrs) != 2 {
		t.Errorf("PlatformErrors() returned %d, want 2", len(platformErrs))
	}

	fallbackErrs := ue.FallbackErrors()
	if len(fallbackErrs) != 1 {
		t.Errorf("FallbackErrors() returned %d, want 1", len(fallbackErrs))
	}
}

func TestUpdateError_Unwrap(t *testing.T) {
	originalErr := errors.New("original error")
	ue := &UpdateError{Errors: []*ComponentError{
		NewComponentError(ErrorSourceCPU, false, originalErr),
		NewComponentError(ErrorSourceMemory, false, errors.New("other error")),
	}}

	// errors.Is should find the original error in the multi-error
	if !errors.Is(ue, originalErr) {
		t.Error("errors.Is() should find wrapped original error")
	}
}

func TestAsUpdateError(t *testing.T) {
	ue := &UpdateError{Errors: []*ComponentError{
		NewComponentError(ErrorSourceCPU, false, errors.New("cpu failed")),
	}}

	// Test with UpdateError
	result := AsUpdateError(ue)
	if result == nil {
		t.Error("AsUpdateError() should return non-nil for UpdateError")
	}
	if len(result.Errors) != 1 {
		t.Errorf("AsUpdateError() returned %d errors, want 1", len(result.Errors))
	}

	// Test with wrapped UpdateError
	wrapped := fmt.Errorf("wrapped: %w", ue)
	result2 := AsUpdateError(wrapped)
	if result2 == nil {
		t.Error("AsUpdateError() should return non-nil for wrapped UpdateError")
	}

	// Test with non-UpdateError
	result3 := AsUpdateError(errors.New("plain error"))
	if result3 != nil {
		t.Error("AsUpdateError() should return nil for non-UpdateError")
	}
}

func TestIsComponentError(t *testing.T) {
	originalErr := errors.New("cpu read failed")
	ce := NewComponentError(ErrorSourceCPU, false, originalErr)

	if !IsComponentError(ce, ErrorSourceCPU) {
		t.Error("IsComponentError() should return true for CPU source")
	}
	if IsComponentError(ce, ErrorSourceMemory) {
		t.Error("IsComponentError() should return false for Memory source")
	}

	// Test with wrapped error
	wrapped := fmt.Errorf("context: %w", ce)
	if !IsComponentError(wrapped, ErrorSourceCPU) {
		t.Error("IsComponentError() should return true for wrapped CPU error")
	}

	// Test with plain error
	if IsComponentError(originalErr, ErrorSourceCPU) {
		t.Error("IsComponentError() should return false for plain error")
	}
}

func TestErrorSourceConstants(t *testing.T) {
	// Verify all error sources are defined and distinct
	sources := []ErrorSource{
		ErrorSourceCPU,
		ErrorSourceMemory,
		ErrorSourceUptime,
		ErrorSourceNetwork,
		ErrorSourceFilesystem,
		ErrorSourceDiskIO,
		ErrorSourceHwmon,
		ErrorSourceProcess,
		ErrorSourceBattery,
		ErrorSourceAudio,
		ErrorSourceSysInfo,
		ErrorSourcePlatform,
	}

	seen := make(map[ErrorSource]bool)
	for _, s := range sources {
		if seen[s] {
			t.Errorf("Duplicate error source: %s", s)
		}
		seen[s] = true
		if len(s) == 0 {
			t.Error("Empty error source found")
		}
	}
}
