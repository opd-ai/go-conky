package conky

import "time"

// DefaultShutdownTimeout is the default timeout for graceful shutdown.
// This can be overridden via Options.ShutdownTimeout.
const DefaultShutdownTimeout = 5 * time.Second

// Options configures the Conky instance behavior.
type Options struct {
	// UpdateInterval overrides the configuration file's update_interval.
	// Zero means use the configuration file's value.
	UpdateInterval time.Duration

	// WindowTitle overrides the window title.
	// Empty string means use the configuration file's value.
	WindowTitle string

	// Headless runs without creating a visible window.
	// Useful for testing or when only system data is needed.
	Headless bool

	// LuaCPULimit overrides the Lua CPU instruction limit.
	// Zero means use the default (10 million instructions).
	LuaCPULimit uint64

	// LuaMemoryLimit overrides the Lua memory limit in bytes.
	// Zero means use the default (50 MB).
	LuaMemoryLimit uint64

	// ShutdownTimeout sets the maximum time to wait for graceful shutdown.
	// Zero means use DefaultShutdownTimeout (5 seconds).
	ShutdownTimeout time.Duration

	// Logger sets a custom logger for debug/info messages.
	// If nil, no logging is performed.
	Logger Logger

	// Metrics sets a custom metrics collector for operational metrics.
	// If nil, DefaultMetrics() is used.
	// Metrics can be exposed via /debug/vars by calling Metrics.RegisterExpvar().
	Metrics *Metrics

	// ErrorTracker sets a custom error tracker for error aggregation and alerting.
	// If nil, DefaultErrorTracker() is used.
	// Use ErrorTracker.AddCondition() to set up alerts.
	// Use ErrorTracker.SetAlertHandler() to receive alert notifications.
	ErrorTracker *ErrorTracker
}

// DefaultOptions returns Options with sensible defaults.
func DefaultOptions() Options {
	return Options{
		UpdateInterval:  0, // Use config file value
		Headless:        false,
		LuaCPULimit:     0, // Use default
		LuaMemoryLimit:  0, // Use default
		ShutdownTimeout: 0, // Use DefaultShutdownTimeout
	}
}

// Logger interface for custom logging.
// It follows the slog-style signature for compatibility with Go's structured logging.
type Logger interface {
	// Debug logs a debug-level message with optional key-value pairs.
	Debug(msg string, args ...any)
	// Info logs an info-level message with optional key-value pairs.
	Info(msg string, args ...any)
	// Warn logs a warning-level message with optional key-value pairs.
	Warn(msg string, args ...any)
	// Error logs an error-level message with optional key-value pairs.
	Error(msg string, args ...any)
}
