package conky

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestSlogAdapter(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	adapter := NewSlogAdapter(slog.New(handler))

	// Test Debug
	buf.Reset()
	adapter.Debug("debug message", "key", "value")
	if !strings.Contains(buf.String(), "debug message") {
		t.Errorf("Debug() did not log message, got: %s", buf.String())
	}
	if !strings.Contains(buf.String(), "key=value") {
		t.Errorf("Debug() did not log key-value pair, got: %s", buf.String())
	}

	// Test Info
	buf.Reset()
	adapter.Info("info message", "count", 42)
	if !strings.Contains(buf.String(), "info message") {
		t.Errorf("Info() did not log message, got: %s", buf.String())
	}
	if !strings.Contains(buf.String(), "count=42") {
		t.Errorf("Info() did not log key-value pair, got: %s", buf.String())
	}

	// Test Warn
	buf.Reset()
	adapter.Warn("warn message")
	if !strings.Contains(buf.String(), "warn message") {
		t.Errorf("Warn() did not log message, got: %s", buf.String())
	}

	// Test Error
	buf.Reset()
	adapter.Error("error message", "err", "something failed")
	if !strings.Contains(buf.String(), "error message") {
		t.Errorf("Error() did not log message, got: %s", buf.String())
	}
	if !strings.Contains(buf.String(), "err=\"something failed\"") && !strings.Contains(buf.String(), "err=something") {
		t.Errorf("Error() did not log key-value pair, got: %s", buf.String())
	}
}

func TestNewSlogAdapterNil(t *testing.T) {
	// Should not panic when nil is passed
	adapter := NewSlogAdapter(nil)
	if adapter == nil {
		t.Fatal("NewSlogAdapter(nil) returned nil")
	}
	if adapter.logger == nil {
		t.Error("NewSlogAdapter(nil) should use slog.Default()")
	}
}

func TestDefaultLogger(t *testing.T) {
	logger := DefaultLogger()
	if logger == nil {
		t.Error("DefaultLogger() returned nil")
	}

	// Should not panic when logging
	logger.Debug("test debug")
	logger.Info("test info")
	logger.Warn("test warn")
	logger.Error("test error")
}

func TestDebugLogger(t *testing.T) {
	logger := DebugLogger()
	if logger == nil {
		t.Error("DebugLogger() returned nil")
	}

	// Should not panic when logging
	logger.Debug("test debug with source")
	logger.Info("test info")
}

func TestJSONLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := JSONLogger(&buf, slog.LevelInfo)

	logger.Info("json test", "field", "value")

	output := buf.String()
	if !strings.Contains(output, `"msg":"json test"`) {
		t.Errorf("JSONLogger did not produce JSON output, got: %s", output)
	}
	if !strings.Contains(output, `"field":"value"`) {
		t.Errorf("JSONLogger did not include field in JSON output, got: %s", output)
	}
}

func TestJSONLoggerNilWriter(t *testing.T) {
	// Should not panic when nil writer is passed
	logger := JSONLogger(nil, slog.LevelInfo)
	if logger == nil {
		t.Error("JSONLogger(nil, ...) returned nil")
	}

	// Should not panic when logging (will write to stderr)
	logger.Info("test")
}

func TestJSONLoggerLevel(t *testing.T) {
	var buf bytes.Buffer
	// Create logger at Warn level
	logger := JSONLogger(&buf, slog.LevelWarn)

	// Debug and Info should be filtered out
	logger.Debug("should not appear")
	logger.Info("should not appear")
	logger.Warn("should appear")

	output := buf.String()
	if strings.Contains(output, "should not appear") {
		t.Errorf("JSONLogger logged messages below its level, got: %s", output)
	}
	if !strings.Contains(output, "should appear") {
		t.Errorf("JSONLogger did not log warn message, got: %s", output)
	}
}

func TestNopLogger(t *testing.T) {
	logger := NopLogger()
	if logger == nil {
		t.Error("NopLogger() returned nil")
	}

	// Should not panic and should not output anything
	logger.Debug("test debug", "key", "value")
	logger.Info("test info", "count", 42)
	logger.Warn("test warn")
	logger.Error("test error", "err", "something failed")
}

func TestNopLoggerInterface(t *testing.T) {
	// Verify nopLogger implements Logger interface
	var logger Logger = NopLogger()
	if logger == nil {
		t.Error("NopLogger() should implement Logger interface")
	}
}

func TestSlogAdapterInterface(t *testing.T) {
	// Verify SlogAdapter implements Logger interface at compile time
	var _ Logger = (*SlogAdapter)(nil)
}
