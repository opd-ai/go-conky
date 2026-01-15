// Package lua provides Golua integration for conky-go.
// This file defines common error types used throughout the package.
package lua

import "errors"

var (
	// ErrNilRuntime is returned when a nil runtime is passed to a function that requires one.
	ErrNilRuntime = errors.New("runtime cannot be nil")

	// ErrInvalidLineCap is returned when an invalid line cap value is provided.
	ErrInvalidLineCap = errors.New("invalid line cap value")

	// ErrInvalidLineJoin is returned when an invalid line join value is provided.
	ErrInvalidLineJoin = errors.New("invalid line join value")

	// ErrInvalidSurface is returned when an invalid surface userdata is provided.
	ErrInvalidSurface = errors.New("expected surface userdata")

	// ErrContextCreation is returned when creating a Cairo context fails.
	ErrContextCreation = errors.New("failed to create context (surface may be destroyed)")
)
