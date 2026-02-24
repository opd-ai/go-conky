package monitor

import (
	"errors"
	"fmt"
	"strings"
)

// ErrorSource identifies which component produced an error.
type ErrorSource string

const (
	ErrorSourceCPU        ErrorSource = "cpu"
	ErrorSourceMemory     ErrorSource = "memory"
	ErrorSourceUptime     ErrorSource = "uptime"
	ErrorSourceNetwork    ErrorSource = "network"
	ErrorSourceFilesystem ErrorSource = "filesystem"
	ErrorSourceDiskIO     ErrorSource = "diskio"
	ErrorSourceHwmon      ErrorSource = "hwmon"
	ErrorSourceProcess    ErrorSource = "process"
	ErrorSourceBattery    ErrorSource = "battery"
	ErrorSourceAudio      ErrorSource = "audio"
	ErrorSourceSysInfo    ErrorSource = "sysinfo"
	ErrorSourcePlatform   ErrorSource = "platform"
)

// ComponentError wraps an error with source information.
// It preserves the original error for inspection via errors.Is/errors.As.
type ComponentError struct {
	Source     ErrorSource
	IsPlatform bool // true if error came from platform adapter
	Err        error
}

// Error implements the error interface.
func (e *ComponentError) Error() string {
	if e.IsPlatform {
		return fmt.Sprintf("%s (platform): %v", e.Source, e.Err)
	}
	return fmt.Sprintf("%s: %v", e.Source, e.Err)
}

// Unwrap returns the underlying error for errors.Is/errors.As support.
func (e *ComponentError) Unwrap() error {
	return e.Err
}

// NewComponentError creates a new ComponentError.
func NewComponentError(source ErrorSource, isPlatform bool, err error) *ComponentError {
	return &ComponentError{
		Source:     source,
		IsPlatform: isPlatform,
		Err:        err,
	}
}

// UpdateError aggregates multiple component errors from a single Update() call.
// It preserves all individual errors, allowing callers to inspect each one.
type UpdateError struct {
	Errors []*ComponentError
}

// Error implements the error interface.
func (e *UpdateError) Error() string {
	if len(e.Errors) == 0 {
		return "no errors"
	}
	if len(e.Errors) == 1 {
		return fmt.Sprintf("update error: %v", e.Errors[0])
	}
	msgs := make([]string, len(e.Errors))
	for i, ce := range e.Errors {
		msgs[i] = ce.Error()
	}
	return fmt.Sprintf("update errors (%d): %s", len(e.Errors), strings.Join(msgs, "; "))
}

// Unwrap returns the underlying errors slice for multi-error support.
// This enables errors.Is to check against any wrapped error.
func (e *UpdateError) Unwrap() []error {
	errs := make([]error, len(e.Errors))
	for i, ce := range e.Errors {
		errs[i] = ce
	}
	return errs
}

// HasSource returns true if any error originated from the given source.
func (e *UpdateError) HasSource(source ErrorSource) bool {
	for _, ce := range e.Errors {
		if ce.Source == source {
			return true
		}
	}
	return false
}

// BySource returns all errors from the specified source.
func (e *UpdateError) BySource(source ErrorSource) []*ComponentError {
	var result []*ComponentError
	for _, ce := range e.Errors {
		if ce.Source == source {
			result = append(result, ce)
		}
	}
	return result
}

// PlatformErrors returns all errors that came from the platform adapter.
func (e *UpdateError) PlatformErrors() []*ComponentError {
	var result []*ComponentError
	for _, ce := range e.Errors {
		if ce.IsPlatform {
			result = append(result, ce)
		}
	}
	return result
}

// FallbackErrors returns all errors from Linux fallback readers.
func (e *UpdateError) FallbackErrors() []*ComponentError {
	var result []*ComponentError
	for _, ce := range e.Errors {
		if !ce.IsPlatform {
			result = append(result, ce)
		}
	}
	return result
}

// AsUpdateError attempts to extract an UpdateError from an error.
// Returns nil if the error is not an UpdateError.
func AsUpdateError(err error) *UpdateError {
	var ue *UpdateError
	if errors.As(err, &ue) {
		return ue
	}
	return nil
}

// IsComponentError returns true if err wraps or is a ComponentError with the given source.
func IsComponentError(err error, source ErrorSource) bool {
	var ce *ComponentError
	for errors.As(err, &ce) {
		if ce.Source == source {
			return true
		}
		// Try to unwrap further
		err = ce.Err
	}
	return false
}
