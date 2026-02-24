package conky

import (
	"time"

	"github.com/opd-ai/go-conky/internal/monitor"
)

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

	// Platform sets a cross-platform monitoring provider.
	// If nil, Linux-specific monitoring is used.
	// Use cmd/conky-go platform wrapper to initialize this from internal/platform.
	Platform monitor.PlatformInterface

	// WatchConfig enables automatic configuration hot-reloading when the
	// configuration file changes on disk. When enabled, file modifications
	// trigger an in-place config reload (via ReloadConfig) without restarting.
	WatchConfig bool

	// WatchDebounce sets the debounce interval for file change events.
	// Multiple rapid file modifications within this window trigger only
	// a single reload. Zero means use the default (500ms).
	WatchDebounce time.Duration
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
