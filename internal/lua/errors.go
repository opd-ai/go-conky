// Package lua provides Golua integration for conky-go.
// This file defines common error types used throughout the package.
package lua

import "errors"

var (
	// ErrNilRuntime is returned when a nil runtime is passed to a function that requires one.
	ErrNilRuntime = errors.New("runtime cannot be nil")

	// ErrInvalidLineCap is returned when an invalid line cap value is provided (must be 0-2).
	ErrInvalidLineCap = errors.New("invalid line cap value (must be 0-2)")

	// ErrInvalidLineJoin is returned when an invalid line join value is provided (must be 0-2).
	ErrInvalidLineJoin = errors.New("invalid line join value (must be 0-2)")
)
