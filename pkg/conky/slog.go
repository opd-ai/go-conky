package conky

import (
	"io"
	"log/slog"
	"os"
)

// SlogAdapter wraps a *slog.Logger to implement the Logger interface.
// This enables integration with Go's structured logging facilities.
//
// Example:
//
//	// Use default slog logger
//	opts := conky.DefaultOptions()
//	opts.Logger = conky.NewSlogAdapter(slog.Default())
//
//	// Use a custom slog handler
//	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
//	opts.Logger = conky.NewSlogAdapter(slog.New(handler))
type SlogAdapter struct {
	logger *slog.Logger
}

// NewSlogAdapter creates a Logger adapter from a *slog.Logger.
// If logger is nil, slog.Default() is used.
func NewSlogAdapter(logger *slog.Logger) *SlogAdapter {
	if logger == nil {
		logger = slog.Default()
	}
	return &SlogAdapter{logger: logger}
}

// Debug logs a debug-level message with optional key-value pairs.
func (s *SlogAdapter) Debug(msg string, args ...any) {
	s.logger.Debug(msg, args...)
}

// Info logs an info-level message with optional key-value pairs.
func (s *SlogAdapter) Info(msg string, args ...any) {
	s.logger.Info(msg, args...)
}

// Warn logs a warning-level message with optional key-value pairs.
func (s *SlogAdapter) Warn(msg string, args ...any) {
	s.logger.Warn(msg, args...)
}

// Error logs an error-level message with optional key-value pairs.
func (s *SlogAdapter) Error(msg string, args ...any) {
	s.logger.Error(msg, args...)
}

// DefaultLogger returns a Logger configured for typical use cases.
// It logs to stderr with text format at Info level.
// For more control, use NewSlogAdapter with a custom slog.Handler.
func DefaultLogger() Logger {
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	return &SlogAdapter{logger: slog.New(handler)}
}

// DebugLogger returns a Logger configured for debugging.
// It logs to stderr with text format at Debug level, including source location.
func DebugLogger() Logger {
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	})
	return &SlogAdapter{logger: slog.New(handler)}
}

// JSONLogger returns a Logger that outputs JSON-formatted logs.
// This is suitable for production environments with log aggregation systems.
func JSONLogger(w io.Writer, level slog.Level) Logger {
	if w == nil {
		w = os.Stderr
	}
	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: level,
	})
	return &SlogAdapter{logger: slog.New(handler)}
}

// NopLogger returns a Logger that discards all log messages.
// Use this when logging should be completely disabled.
func NopLogger() Logger {
	return &nopLogger{}
}

// nopLogger implements Logger but discards all messages.
type nopLogger struct{}

func (n *nopLogger) Debug(msg string, args ...any) {}
func (n *nopLogger) Info(msg string, args ...any)  {}
func (n *nopLogger) Warn(msg string, args ...any)  {}
func (n *nopLogger) Error(msg string, args ...any) {}
