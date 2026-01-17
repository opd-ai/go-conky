package conky

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
)

// correlationIDKey is the context key for storing correlation IDs.
type correlationIDKey struct{}

// CorrelationID represents a unique identifier for tracing operations.
// It is used to correlate log entries and events across different
// components and goroutines within a single logical operation.
type CorrelationID string

// String returns the string representation of the correlation ID.
func (c CorrelationID) String() string {
	return string(c)
}

// NewCorrelationID generates a new random correlation ID.
// The ID is a 16-character hex string (64 bits of randomness).
func NewCorrelationID() CorrelationID {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// Fallback to a deterministic ID if random generation fails
		return CorrelationID("00000000-fallback")
	}
	return CorrelationID(hex.EncodeToString(b))
}

// WithCorrelationID returns a new context with the given correlation ID.
// If id is empty, a new correlation ID is generated.
func WithCorrelationID(ctx context.Context, id CorrelationID) context.Context {
	if id == "" {
		id = NewCorrelationID()
	}
	return context.WithValue(ctx, correlationIDKey{}, id)
}

// CorrelationIDFromContext retrieves the correlation ID from the context.
// Returns an empty string if no correlation ID is present.
func CorrelationIDFromContext(ctx context.Context) CorrelationID {
	if ctx == nil {
		return ""
	}
	if id, ok := ctx.Value(correlationIDKey{}).(CorrelationID); ok {
		return id
	}
	return ""
}

// EnsureCorrelationID returns the context with a correlation ID,
// generating a new one if the context doesn't already have one.
// This is useful for ensuring all operations have a correlation ID.
func EnsureCorrelationID(ctx context.Context) context.Context {
	if CorrelationIDFromContext(ctx) != "" {
		return ctx
	}
	return WithCorrelationID(ctx, NewCorrelationID())
}

// CorrelatedLogger wraps a Logger to automatically include correlation IDs
// in all log messages when available from the context.
type CorrelatedLogger struct {
	logger Logger
	ctx    context.Context
}

// NewCorrelatedLogger creates a CorrelatedLogger that includes the correlation ID
// from the context in all log messages.
func NewCorrelatedLogger(ctx context.Context, logger Logger) *CorrelatedLogger {
	if logger == nil {
		logger = NopLogger()
	}
	return &CorrelatedLogger{
		logger: logger,
		ctx:    ctx,
	}
}

// withCorrelation prepends the correlation ID to the args if present.
func (c *CorrelatedLogger) withCorrelation(args []any) []any {
	id := CorrelationIDFromContext(c.ctx)
	if id != "" {
		return append([]any{"correlation_id", string(id)}, args...)
	}
	return args
}

// Debug logs a debug-level message with optional key-value pairs.
func (c *CorrelatedLogger) Debug(msg string, args ...any) {
	c.logger.Debug(msg, c.withCorrelation(args)...)
}

// Info logs an info-level message with optional key-value pairs.
func (c *CorrelatedLogger) Info(msg string, args ...any) {
	c.logger.Info(msg, c.withCorrelation(args)...)
}

// Warn logs a warning-level message with optional key-value pairs.
func (c *CorrelatedLogger) Warn(msg string, args ...any) {
	c.logger.Warn(msg, c.withCorrelation(args)...)
}

// Error logs an error-level message with optional key-value pairs.
func (c *CorrelatedLogger) Error(msg string, args ...any) {
	c.logger.Error(msg, c.withCorrelation(args)...)
}

// WithContext returns a new CorrelatedLogger with the given context.
// This is useful for updating the correlation ID as operations flow through.
func (c *CorrelatedLogger) WithContext(ctx context.Context) *CorrelatedLogger {
	return &CorrelatedLogger{
		logger: c.logger,
		ctx:    ctx,
	}
}

// CorrelatedSlogHandler wraps an slog.Handler to automatically extract and add
// correlation IDs from context. This integrates with Go's native context handling
// in slog when using slog.InfoContext, slog.DebugContext, etc.
type CorrelatedSlogHandler struct {
	inner slog.Handler
}

// NewCorrelatedSlogHandler creates a new handler that adds correlation IDs.
func NewCorrelatedSlogHandler(inner slog.Handler) *CorrelatedSlogHandler {
	return &CorrelatedSlogHandler{inner: inner}
}

// Enabled reports whether the handler handles records at the given level.
func (h *CorrelatedSlogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

// Handle processes the log record, adding correlation ID if present in context.
func (h *CorrelatedSlogHandler) Handle(ctx context.Context, r slog.Record) error {
	if id := CorrelationIDFromContext(ctx); id != "" {
		r = r.Clone()
		r.AddAttrs(slog.String("correlation_id", string(id)))
	}
	return h.inner.Handle(ctx, r)
}

// WithAttrs returns a new handler with the given attributes.
func (h *CorrelatedSlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &CorrelatedSlogHandler{inner: h.inner.WithAttrs(attrs)}
}

// WithGroup returns a new handler with the given group name.
func (h *CorrelatedSlogHandler) WithGroup(name string) slog.Handler {
	return &CorrelatedSlogHandler{inner: h.inner.WithGroup(name)}
}

// CorrelatedJSONLogger returns a Logger that outputs JSON-formatted logs
// with automatic correlation ID extraction from context.
// Use slog.InfoContext, slog.DebugContext, etc. to include correlation IDs.
func CorrelatedJSONLogger(level slog.Level) *slog.Logger {
	handler := slog.NewJSONHandler(nil, &slog.HandlerOptions{Level: level})
	return slog.New(NewCorrelatedSlogHandler(handler))
}
