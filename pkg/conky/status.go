package conky

import "time"

// Status represents the current state of a Conky instance.
type Status struct {
	// Running indicates if the instance is currently active.
	Running bool
	// StartTime is when the instance was last started (zero if never started).
	StartTime time.Time
	// UpdateCount is the number of update cycles completed since last start.
	UpdateCount uint64
	// LastError is the most recent error encountered (nil if none).
	LastError error
	// ConfigSource describes the configuration source (file path or "embedded").
	ConfigSource string
}

// ErrorHandler is a callback for runtime errors.
// It is called asynchronously when errors occur during operation.
// Do not block in the handler; perform only quick, non-blocking operations.
type ErrorHandler func(err error)

// EventHandler is a callback for lifecycle events.
// It is called asynchronously; do not block in the handler.
type EventHandler func(event Event)

// Event represents a lifecycle event.
type Event struct {
	Type      EventType
	Timestamp time.Time
	Message   string
}

// EventType enumerates lifecycle event types.
// The underlying integer values are implementation details and should not
// be relied upon for serialization. Use the constant names for comparison.
type EventType int

const (
	// EventStarted is emitted when the instance starts successfully.
	EventStarted EventType = iota
	// EventStopped is emitted when the instance stops.
	EventStopped
	// EventRestarted is emitted after a successful restart.
	EventRestarted
	// EventConfigReloaded is emitted when configuration is reloaded.
	EventConfigReloaded
	// EventError is emitted when a recoverable error occurs.
	EventError
)

// String returns a human-readable representation of the event type.
func (e EventType) String() string {
	switch e {
	case EventStarted:
		return "started"
	case EventStopped:
		return "stopped"
	case EventRestarted:
		return "restarted"
	case EventConfigReloaded:
		return "config_reloaded"
	case EventError:
		return "error"
	default:
		return "unknown"
	}
}
