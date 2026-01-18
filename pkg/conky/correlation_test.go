package conky

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestNewCorrelationID(t *testing.T) {
	// Test that NewCorrelationID generates unique IDs
	ids := make(map[CorrelationID]bool)
	for i := 0; i < 100; i++ {
		id := NewCorrelationID()
		if id == "" {
			t.Error("NewCorrelationID returned empty string")
		}
		if len(id) != 16 {
			t.Errorf("expected 16-character ID, got %d characters: %s", len(id), id)
		}
		if ids[id] {
			t.Errorf("duplicate ID generated: %s", id)
		}
		ids[id] = true
	}
}

func TestCorrelationID_String(t *testing.T) {
	id := CorrelationID("test-id-12345678")
	if id.String() != "test-id-12345678" {
		t.Errorf("expected 'test-id-12345678', got '%s'", id.String())
	}
}

func TestWithCorrelationID(t *testing.T) {
	ctx := context.Background()

	// Test with explicit ID
	id := CorrelationID("explicit-id")
	ctx = WithCorrelationID(ctx, id)
	got := CorrelationIDFromContext(ctx)
	if got != id {
		t.Errorf("expected '%s', got '%s'", id, got)
	}

	// Test with empty ID (should generate new one)
	ctx2 := context.Background()
	ctx2 = WithCorrelationID(ctx2, "")
	got2 := CorrelationIDFromContext(ctx2)
	if got2 == "" {
		t.Error("expected generated ID, got empty string")
	}
	if len(got2) != 16 {
		t.Errorf("expected 16-character generated ID, got %d: %s", len(got2), got2)
	}
}

func TestCorrelationIDFromContext(t *testing.T) {
	// Test with nil context
	got := CorrelationIDFromContext(nil)
	if got != "" {
		t.Errorf("expected empty string for nil context, got '%s'", got)
	}

	// Test with context without correlation ID
	ctx := context.Background()
	got = CorrelationIDFromContext(ctx)
	if got != "" {
		t.Errorf("expected empty string for context without ID, got '%s'", got)
	}

	// Test with context with correlation ID
	id := CorrelationID("my-correlation-id")
	ctx = WithCorrelationID(ctx, id)
	got = CorrelationIDFromContext(ctx)
	if got != id {
		t.Errorf("expected '%s', got '%s'", id, got)
	}
}

func TestEnsureCorrelationID(t *testing.T) {
	// Test with context without ID (should add one)
	ctx := context.Background()
	ctx = EnsureCorrelationID(ctx)
	id := CorrelationIDFromContext(ctx)
	if id == "" {
		t.Error("expected correlation ID to be added")
	}

	// Test with context that already has ID (should keep existing)
	existingID := CorrelationID("existing-id")
	ctx2 := WithCorrelationID(context.Background(), existingID)
	ctx2 = EnsureCorrelationID(ctx2)
	id = CorrelationIDFromContext(ctx2)
	if id != existingID {
		t.Errorf("expected existing ID '%s', got '%s'", existingID, id)
	}
}

func TestCorrelatedLogger(t *testing.T) {
	// Create a mock logger to capture calls
	calls := make([]struct {
		level string
		msg   string
		args  []any
	}, 0)

	mockLogger := &mockLoggerForCorrelation{
		calls: &calls,
	}

	// Test with correlation ID
	id := CorrelationID("test-correlation")
	ctx := WithCorrelationID(context.Background(), id)
	logger := NewCorrelatedLogger(ctx, mockLogger)

	logger.Debug("debug message", "key1", "value1")
	logger.Info("info message", "key2", "value2")
	logger.Warn("warn message", "key3", "value3")
	logger.Error("error message", "key4", "value4")

	if len(calls) != 4 {
		t.Fatalf("expected 4 calls, got %d", len(calls))
	}

	// Verify each call includes correlation_id
	for i, call := range calls {
		if len(call.args) < 2 {
			t.Errorf("call %d: expected at least 2 args, got %d", i, len(call.args))
			continue
		}
		if call.args[0] != "correlation_id" {
			t.Errorf("call %d: expected first arg to be 'correlation_id', got '%v'", i, call.args[0])
		}
		if call.args[1] != "test-correlation" {
			t.Errorf("call %d: expected correlation ID 'test-correlation', got '%v'", i, call.args[1])
		}
	}
}

func TestCorrelatedLogger_WithoutCorrelationID(t *testing.T) {
	calls := make([]struct {
		level string
		msg   string
		args  []any
	}, 0)

	mockLogger := &mockLoggerForCorrelation{
		calls: &calls,
	}

	// Test without correlation ID
	ctx := context.Background()
	logger := NewCorrelatedLogger(ctx, mockLogger)

	logger.Info("message", "key", "value")

	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}

	// Should not have correlation_id prepended
	if len(calls[0].args) != 2 {
		t.Errorf("expected 2 args (original only), got %d", len(calls[0].args))
	}
}

func TestCorrelatedLogger_NilLogger(t *testing.T) {
	ctx := context.Background()
	logger := NewCorrelatedLogger(ctx, nil)

	// Should not panic
	logger.Debug("test")
	logger.Info("test")
	logger.Warn("test")
	logger.Error("test")
}

func TestCorrelatedLogger_WithContext(t *testing.T) {
	calls := make([]struct {
		level string
		msg   string
		args  []any
	}, 0)

	mockLogger := &mockLoggerForCorrelation{
		calls: &calls,
	}

	// Create initial logger with one ID
	id1 := CorrelationID("id-1")
	ctx1 := WithCorrelationID(context.Background(), id1)
	logger := NewCorrelatedLogger(ctx1, mockLogger)

	logger.Info("first message")

	// Create new logger with different context
	id2 := CorrelationID("id-2")
	ctx2 := WithCorrelationID(context.Background(), id2)
	logger2 := logger.WithContext(ctx2)

	logger2.Info("second message")

	if len(calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(calls))
	}

	// First call should have id-1
	if calls[0].args[1] != "id-1" {
		t.Errorf("expected first call to have 'id-1', got '%v'", calls[0].args[1])
	}

	// Second call should have id-2
	if calls[1].args[1] != "id-2" {
		t.Errorf("expected second call to have 'id-2', got '%v'", calls[1].args[1])
	}
}

func TestCorrelatedSlogHandler(t *testing.T) {
	var buf bytes.Buffer
	innerHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	handler := NewCorrelatedSlogHandler(innerHandler)
	logger := slog.New(handler)

	// Log with correlation ID in context
	id := CorrelationID("slog-correlation-id")
	ctx := WithCorrelationID(context.Background(), id)
	logger.InfoContext(ctx, "test message", "extra", "data")

	// Parse the JSON output
	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	// Verify correlation_id is present
	if record["correlation_id"] != "slog-correlation-id" {
		t.Errorf("expected correlation_id 'slog-correlation-id', got '%v'", record["correlation_id"])
	}

	// Verify other fields are present
	if record["msg"] != "test message" {
		t.Errorf("expected msg 'test message', got '%v'", record["msg"])
	}
	if record["extra"] != "data" {
		t.Errorf("expected extra 'data', got '%v'", record["extra"])
	}
}

func TestCorrelatedSlogHandler_WithoutCorrelationID(t *testing.T) {
	var buf bytes.Buffer
	innerHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	handler := NewCorrelatedSlogHandler(innerHandler)
	logger := slog.New(handler)

	// Log without correlation ID
	ctx := context.Background()
	logger.InfoContext(ctx, "test message")

	// Parse the JSON output
	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	// Verify correlation_id is NOT present
	if _, ok := record["correlation_id"]; ok {
		t.Error("expected no correlation_id when not in context")
	}
}

func TestCorrelatedSlogHandler_Enabled(t *testing.T) {
	innerHandler := slog.NewTextHandler(nil, &slog.HandlerOptions{Level: slog.LevelWarn})
	handler := NewCorrelatedSlogHandler(innerHandler)

	ctx := context.Background()

	if handler.Enabled(ctx, slog.LevelDebug) {
		t.Error("expected Debug to be disabled")
	}
	if handler.Enabled(ctx, slog.LevelInfo) {
		t.Error("expected Info to be disabled")
	}
	if !handler.Enabled(ctx, slog.LevelWarn) {
		t.Error("expected Warn to be enabled")
	}
	if !handler.Enabled(ctx, slog.LevelError) {
		t.Error("expected Error to be enabled")
	}
}

func TestCorrelatedSlogHandler_WithAttrs(t *testing.T) {
	var buf bytes.Buffer
	innerHandler := slog.NewJSONHandler(&buf, nil)
	handler := NewCorrelatedSlogHandler(innerHandler)

	// Add attributes
	handlerWithAttrs := handler.WithAttrs([]slog.Attr{
		slog.String("component", "test"),
	})
	logger := slog.New(handlerWithAttrs)

	id := CorrelationID("attrs-test")
	ctx := WithCorrelationID(context.Background(), id)
	logger.InfoContext(ctx, "message")

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	// Verify both correlation_id and component are present
	if record["correlation_id"] != "attrs-test" {
		t.Errorf("expected correlation_id 'attrs-test', got '%v'", record["correlation_id"])
	}
	if record["component"] != "test" {
		t.Errorf("expected component 'test', got '%v'", record["component"])
	}
}

func TestCorrelatedSlogHandler_WithGroup(t *testing.T) {
	var buf bytes.Buffer
	innerHandler := slog.NewJSONHandler(&buf, nil)
	handler := NewCorrelatedSlogHandler(innerHandler)

	// Add group
	handlerWithGroup := handler.WithGroup("request")
	logger := slog.New(handlerWithGroup)

	id := CorrelationID("group-test")
	ctx := WithCorrelationID(context.Background(), id)
	logger.InfoContext(ctx, "message", "path", "/api/test")

	output := buf.String()
	// Verify the output contains correlation_id (at root level) and the grouped attribute
	if !strings.Contains(output, "correlation_id") {
		t.Error("expected correlation_id in output")
	}
	if !strings.Contains(output, "request") {
		t.Error("expected request group in output")
	}
}

// mockLoggerForCorrelation is a test mock for the Logger interface
type mockLoggerForCorrelation struct {
	calls *[]struct {
		level string
		msg   string
		args  []any
	}
}

func (m *mockLoggerForCorrelation) Debug(msg string, args ...any) {
	*m.calls = append(*m.calls, struct {
		level string
		msg   string
		args  []any
	}{"debug", msg, args})
}

func (m *mockLoggerForCorrelation) Info(msg string, args ...any) {
	*m.calls = append(*m.calls, struct {
		level string
		msg   string
		args  []any
	}{"info", msg, args})
}

func (m *mockLoggerForCorrelation) Warn(msg string, args ...any) {
	*m.calls = append(*m.calls, struct {
		level string
		msg   string
		args  []any
	}{"warn", msg, args})
}

func (m *mockLoggerForCorrelation) Error(msg string, args ...any) {
	*m.calls = append(*m.calls, struct {
		level string
		msg   string
		args  []any
	}{"error", msg, args})
}

func TestCorrelatedJSONLogger(t *testing.T) {
	// CorrelatedJSONLogger with nil writer outputs to stderr
	// Just verify it creates a valid logger that doesn't panic
	logger := CorrelatedJSONLogger(slog.LevelInfo)
	if logger == nil {
		t.Error("CorrelatedJSONLogger returned nil")
	}
	// Note: This logger writes to stderr since nil is passed as writer.
	// We can only test that it doesn't panic on creation.
}

func TestCorrelatedJSONLogger_Levels(t *testing.T) {
	// Test different log levels
	tests := []struct {
		level slog.Level
		name  string
	}{
		{slog.LevelDebug, "debug"},
		{slog.LevelInfo, "info"},
		{slog.LevelWarn, "warn"},
		{slog.LevelError, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := CorrelatedJSONLogger(tt.level)
			if logger == nil {
				t.Errorf("CorrelatedJSONLogger(%v) returned nil", tt.level)
			}
		})
	}
}
