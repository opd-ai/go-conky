package conky

import (
	"embed"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

//go:embed testdata/*
var testFS embed.FS

func TestEventTypeString(t *testing.T) {
	tests := []struct {
		eventType EventType
		expected  string
	}{
		{EventStarted, "started"},
		{EventStopped, "stopped"},
		{EventRestarted, "restarted"},
		{EventConfigReloaded, "config_reloaded"},
		{EventError, "error"},
		{EventType(100), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.eventType.String(); got != tt.expected {
				t.Errorf("EventType.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	if opts.UpdateInterval != 0 {
		t.Errorf("UpdateInterval = %v, want 0", opts.UpdateInterval)
	}
	if opts.Headless != false {
		t.Errorf("Headless = %v, want false", opts.Headless)
	}
	if opts.LuaCPULimit != 0 {
		t.Errorf("LuaCPULimit = %v, want 0", opts.LuaCPULimit)
	}
	if opts.LuaMemoryLimit != 0 {
		t.Errorf("LuaMemoryLimit = %v, want 0", opts.LuaMemoryLimit)
	}
	if opts.ShutdownTimeout != 0 {
		t.Errorf("ShutdownTimeout = %v, want 0", opts.ShutdownTimeout)
	}
}

func TestNewWithInvalidPath(t *testing.T) {
	_, err := New("/nonexistent/path/config.lua", nil)
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestNewFromReaderWithInvalidFormat(t *testing.T) {
	reader := strings.NewReader("some content")
	_, err := NewFromReader(reader, "invalid_format", nil)
	if err == nil {
		t.Error("expected error for invalid format, got nil")
	}
	if !strings.Contains(err.Error(), "invalid format") {
		t.Errorf("error should mention invalid format, got: %v", err)
	}
}

func TestNewFromReaderWithLegacyConfig(t *testing.T) {
	config := `# Minimal legacy config
TEXT
$uptime
`
	reader := strings.NewReader(config)
	c, err := NewFromReader(reader, "legacy", nil)
	if err != nil {
		t.Fatalf("NewFromReader failed: %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil Conky instance")
	}
	if c.IsRunning() {
		t.Error("new instance should not be running")
	}
}

func TestNewFromReaderWithLuaConfig(t *testing.T) {
	config := `
conky.config = {}
conky.text = [[$uptime]]
`
	reader := strings.NewReader(config)
	c, err := NewFromReader(reader, "lua", nil)
	if err != nil {
		t.Fatalf("NewFromReader failed: %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil Conky instance")
	}
}

func TestNewFromFS(t *testing.T) {
	c, err := NewFromFS(testFS, "testdata/minimal.conkyrc", nil)
	if err != nil {
		t.Fatalf("NewFromFS failed: %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil Conky instance")
	}

	status := c.Status()
	if !strings.HasPrefix(status.ConfigSource, "embedded:") {
		t.Errorf("ConfigSource should start with 'embedded:', got %s", status.ConfigSource)
	}
}

func TestNewFromFSWithInvalidPath(t *testing.T) {
	_, err := NewFromFS(testFS, "testdata/nonexistent.lua", nil)
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestLifecycleHeadless(t *testing.T) {
	config := `# Minimal config
TEXT
$uptime
`
	reader := strings.NewReader(config)
	c, err := NewFromReader(reader, "legacy", &Options{
		Headless: true,
	})
	if err != nil {
		t.Fatalf("NewFromReader failed: %v", err)
	}

	// Test initial state
	if c.IsRunning() {
		t.Error("instance should not be running before Start()")
	}

	// Test Start
	err = c.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if !c.IsRunning() {
		t.Error("instance should be running after Start()")
	}

	// Test double Start
	err = c.Start()
	if err == nil {
		t.Error("expected error on double Start()")
	}

	// Test Status
	status := c.Status()
	if !status.Running {
		t.Error("Status.Running should be true")
	}
	if status.StartTime.IsZero() {
		t.Error("Status.StartTime should not be zero")
	}
	if status.ConfigSource != "reader" {
		t.Errorf("ConfigSource = %s, want 'reader'", status.ConfigSource)
	}

	// Test Stop
	err = c.Stop()
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
	if c.IsRunning() {
		t.Error("instance should not be running after Stop()")
	}

	// Test double Stop (should be no-op)
	err = c.Stop()
	if err != nil {
		t.Errorf("double Stop should not error, got: %v", err)
	}
}

func TestRestart(t *testing.T) {
	config := `# Minimal config
TEXT
$uptime
`
	reader := strings.NewReader(config)
	c, err := NewFromReader(reader, "legacy", &Options{
		Headless: true,
	})
	if err != nil {
		t.Fatalf("NewFromReader failed: %v", err)
	}

	// Start
	err = c.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Restart
	err = c.Restart()
	if err != nil {
		t.Fatalf("Restart failed: %v", err)
	}
	if !c.IsRunning() {
		t.Error("instance should be running after Restart()")
	}

	// Clean up
	if err := c.Stop(); err != nil {
		t.Errorf("Stop failed: %v", err)
	}
}

func TestEventHandler(t *testing.T) {
	config := `# Minimal config
TEXT
$uptime
`
	reader := strings.NewReader(config)
	c, err := NewFromReader(reader, "legacy", &Options{
		Headless: true,
	})
	if err != nil {
		t.Fatalf("NewFromReader failed: %v", err)
	}

	// Track events
	var events []Event
	var eventsMu sync.Mutex
	eventsCh := make(chan struct{}, 10)

	c.SetEventHandler(func(e Event) {
		eventsMu.Lock()
		events = append(events, e)
		eventsMu.Unlock()
		eventsCh <- struct{}{}
	})

	// Start and wait for event
	if err := c.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	select {
	case <-eventsCh:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for start event")
	}

	// Stop and wait for event
	if err := c.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	select {
	case <-eventsCh:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for stop event")
	}

	// Check events
	eventsMu.Lock()
	defer eventsMu.Unlock()

	if len(events) < 2 {
		t.Fatalf("expected at least 2 events, got %d", len(events))
	}

	// First event should be Started
	if events[0].Type != EventStarted {
		t.Errorf("first event should be EventStarted, got %v", events[0].Type)
	}

	// Last event should be Stopped
	if events[len(events)-1].Type != EventStopped {
		t.Errorf("last event should be EventStopped, got %v", events[len(events)-1].Type)
	}
}

func TestConcurrentAccess(t *testing.T) {
	config := `# Minimal config
TEXT
$uptime
`
	reader := strings.NewReader(config)
	c, err := NewFromReader(reader, "legacy", &Options{
		Headless: true,
	})
	if err != nil {
		t.Fatalf("NewFromReader failed: %v", err)
	}

	if err := c.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() {
		if err := c.Stop(); err != nil {
			t.Errorf("Stop failed: %v", err)
		}
	}()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = c.IsRunning()
			_ = c.Status()
		}()
	}
	wg.Wait()
}

func TestOptionsWithCustomLogger(t *testing.T) {
	// Create a simple test logger
	var logs []string
	var logsMu sync.Mutex

	logger := &testLogger{
		logFn: func(level, msg string, args ...any) {
			logsMu.Lock()
			defer logsMu.Unlock()
			logs = append(logs, level+": "+msg)
		},
	}

	config := `# Minimal config
TEXT
$uptime
`
	reader := strings.NewReader(config)
	c, err := NewFromReader(reader, "legacy", &Options{
		Headless: true,
		Logger:   logger,
	})
	if err != nil {
		t.Fatalf("NewFromReader failed: %v", err)
	}

	if err := c.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if err := c.Stop(); err != nil {
		t.Errorf("Stop failed: %v", err)
	}

	// Logger was provided and instance was created successfully
	if c == nil {
		t.Error("expected non-nil Conky instance with custom logger")
	}
}

// testLogger implements the Logger interface for testing.
type testLogger struct {
	logFn func(level, msg string, args ...any)
}

func (l *testLogger) Debug(msg string, args ...any) {
	l.logFn("DEBUG", msg, args...)
}

func (l *testLogger) Info(msg string, args ...any) {
	l.logFn("INFO", msg, args...)
}

func (l *testLogger) Warn(msg string, args ...any) {
	l.logFn("WARN", msg, args...)
}

func (l *testLogger) Error(msg string, args ...any) {
	l.logFn("ERROR", msg, args...)
}

func TestStatus(t *testing.T) {
	config := `# Minimal config
TEXT
$uptime
`
	reader := strings.NewReader(config)
	c, err := NewFromReader(reader, "legacy", &Options{
		Headless: true,
	})
	if err != nil {
		t.Fatalf("NewFromReader failed: %v", err)
	}

	// Check initial status
	status := c.Status()
	if status.Running {
		t.Error("Running should be false before Start()")
	}
	if !status.StartTime.IsZero() {
		t.Error("StartTime should be zero before Start()")
	}
	if status.UpdateCount != 0 {
		t.Error("UpdateCount should be 0 before Start()")
	}
	if status.LastError != nil {
		t.Error("LastError should be nil initially")
	}

	// Start and check status
	if err := c.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	status = c.Status()
	if !status.Running {
		t.Error("Running should be true after Start()")
	}
	if status.StartTime.IsZero() {
		t.Error("StartTime should not be zero after Start()")
	}

	if err := c.Stop(); err != nil {
		t.Errorf("Stop failed: %v", err)
	}
}

func TestErrorHandler(t *testing.T) {
	config := `# Minimal config
TEXT
$uptime
`
	reader := strings.NewReader(config)
	c, err := NewFromReader(reader, "legacy", &Options{
		Headless: true,
	})
	if err != nil {
		t.Fatalf("NewFromReader failed: %v", err)
	}

	// Track errors received by the handler
	var receivedErrors []error
	var errorsMu sync.Mutex
	errorsCh := make(chan error, 10)

	c.SetErrorHandler(func(err error) {
		errorsMu.Lock()
		receivedErrors = append(receivedErrors, err)
		errorsMu.Unlock()
		errorsCh <- err
	})

	if err := c.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Access the impl to directly test notifyError
	impl := c.(*conkyImpl)
	testErr := fmt.Errorf("test runtime error")
	impl.notifyError(testErr)

	// Wait for the error to be received
	select {
	case receivedErr := <-errorsCh:
		if receivedErr != testErr {
			t.Errorf("received error = %v, want %v", receivedErr, testErr)
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for error handler to be called")
	}

	// Also verify the error is stored in status
	status := c.Status()
	if status.LastError == nil {
		t.Error("LastError should not be nil after notifyError")
	}

	if err := c.Stop(); err != nil {
		t.Errorf("Stop failed: %v", err)
	}
}

func TestReloadConfig(t *testing.T) {
	config := `# Minimal config
TEXT
$uptime
`
	reader := strings.NewReader(config)
	c, err := NewFromReader(reader, "legacy", &Options{
		Headless: true,
	})
	if err != nil {
		t.Fatalf("NewFromReader failed: %v", err)
	}

	// Start the instance
	if err := c.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Track events
	var events []Event
	var eventsMu sync.Mutex
	eventsCh := make(chan struct{}, 10)

	c.SetEventHandler(func(e Event) {
		eventsMu.Lock()
		events = append(events, e)
		eventsMu.Unlock()
		eventsCh <- struct{}{}
	})

	// Drain any existing events
	for len(eventsCh) > 0 {
		<-eventsCh
	}
	eventsMu.Lock()
	events = nil
	eventsMu.Unlock()

	// Reload config
	if err := c.ReloadConfig(); err != nil {
		t.Fatalf("ReloadConfig failed: %v", err)
	}

	// Should still be running
	if !c.IsRunning() {
		t.Error("instance should still be running after ReloadConfig()")
	}

	// Wait for config reload event
	select {
	case <-eventsCh:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for config reload event")
	}

	// Check that we got a config_reloaded event
	eventsMu.Lock()
	hasReloadEvent := false
	for _, e := range events {
		if e.Type == EventConfigReloaded {
			hasReloadEvent = true
			break
		}
	}
	eventsMu.Unlock()

	if !hasReloadEvent {
		t.Error("expected EventConfigReloaded event")
	}

	// Clean up
	if err := c.Stop(); err != nil {
		t.Errorf("Stop failed: %v", err)
	}
}

func TestReloadConfigWhenNotRunning(t *testing.T) {
	config := `# Minimal config
TEXT
$uptime
`
	reader := strings.NewReader(config)
	c, err := NewFromReader(reader, "legacy", &Options{
		Headless: true,
	})
	if err != nil {
		t.Fatalf("NewFromReader failed: %v", err)
	}

	// ReloadConfig should fail when not running
	err = c.ReloadConfig()
	if err == nil {
		t.Error("ReloadConfig should fail when instance is not running")
	}
	if !strings.Contains(err.Error(), "not running") {
		t.Errorf("error should mention 'not running', got: %v", err)
	}
}

func TestReloadConfigNoInterruption(t *testing.T) {
	// This test verifies that ReloadConfig doesn't interrupt execution
	// by checking the instance remains running throughout
	config := `# Minimal config
TEXT
$uptime
`
	reader := strings.NewReader(config)
	c, err := NewFromReader(reader, "legacy", &Options{
		Headless: true,
	})
	if err != nil {
		t.Fatalf("NewFromReader failed: %v", err)
	}

	if err := c.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Perform multiple reloads in quick succession
	for i := 0; i < 5; i++ {
		if !c.IsRunning() {
			t.Fatalf("instance stopped unexpectedly at reload %d", i)
		}
		if err := c.ReloadConfig(); err != nil {
			t.Fatalf("ReloadConfig %d failed: %v", i, err)
		}
		// Verify still running after each reload
		if !c.IsRunning() {
			t.Fatalf("instance stopped after reload %d", i)
		}
	}

	// Clean up
	if err := c.Stop(); err != nil {
		t.Errorf("Stop failed: %v", err)
	}
}

func TestMetricsAccessor(t *testing.T) {
	config := `# Minimal config
TEXT
$uptime
`
	reader := strings.NewReader(config)

	// Create with custom metrics
	customMetrics := NewMetrics()
	c, err := NewFromReader(reader, "legacy", &Options{
		Headless: true,
		Metrics:  customMetrics,
	})
	if err != nil {
		t.Fatalf("NewFromReader failed: %v", err)
	}

	// Metrics is nil before Start (initialized in Start)
	// This is the current behavior

	// Start should initialize metrics
	if err := c.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Get metrics accessor after Start
	metrics := c.Metrics()
	if metrics == nil {
		t.Fatal("Metrics() returned nil after Start()")
	}

	// Metrics should be the same instance we provided
	if metrics != customMetrics {
		t.Error("Metrics() should return the custom metrics instance")
	}

	snap := metrics.Snapshot()
	if snap.Starts < 1 {
		t.Errorf("expected Starts >= 1, got %d", snap.Starts)
	}

	// Stop should increment stops counter
	if err := c.Stop(); err != nil {
		t.Errorf("Stop failed: %v", err)
	}

	snap = metrics.Snapshot()
	if snap.Stops < 1 {
		t.Errorf("expected Stops >= 1, got %d", snap.Stops)
	}
}

func TestMetricsDefaultWhenNil(t *testing.T) {
	config := `# Minimal config
TEXT
$uptime
`
	reader := strings.NewReader(config)

	// Create without custom metrics (should use default)
	c, err := NewFromReader(reader, "legacy", &Options{
		Headless: true,
		// Metrics is nil, should use default
	})
	if err != nil {
		t.Fatalf("NewFromReader failed: %v", err)
	}

	// Start to initialize metrics
	if err := c.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Metrics should be non-nil after Start (uses default)
	metrics := c.Metrics()
	if metrics == nil {
		t.Fatal("Metrics() should not return nil after Start()")
	}

	if err := c.Stop(); err != nil {
		t.Errorf("Stop failed: %v", err)
	}
}

func TestEventWarning(t *testing.T) {
	config := `# Minimal config
TEXT
$uptime
`
	reader := strings.NewReader(config)
	c, err := NewFromReader(reader, "legacy", &Options{
		Headless: true,
	})
	if err != nil {
		t.Fatalf("NewFromReader failed: %v", err)
	}

	// Track warning events
	var events []Event
	var eventsMu sync.Mutex
	eventsCh := make(chan struct{}, 10)

	c.SetEventHandler(func(e Event) {
		eventsMu.Lock()
		events = append(events, e)
		eventsMu.Unlock()
		eventsCh <- struct{}{}
	})

	if err := c.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Emit a warning event directly
	impl := c.(*conkyImpl)
	impl.emitEvent(EventWarning, "test warning message")

	// Wait for the event
	select {
	case <-eventsCh:
	case <-time.After(time.Second):
		// May have already received start event
	}

	// Continue to receive any additional events
	select {
	case <-eventsCh:
	case <-time.After(100 * time.Millisecond):
	}

	// Check that we got a warning event
	eventsMu.Lock()
	hasWarningEvent := false
	for _, e := range events {
		if e.Type == EventWarning {
			hasWarningEvent = true
			if e.Message != "test warning message" {
				t.Errorf("expected message 'test warning message', got '%s'", e.Message)
			}
		}
	}
	eventsMu.Unlock()

	if !hasWarningEvent {
		t.Error("expected EventWarning event")
	}

	if err := c.Stop(); err != nil {
		t.Errorf("Stop failed: %v", err)
	}
}

func TestEmitEventWithoutHandler(t *testing.T) {
	config := `# Minimal config
TEXT
$uptime
`
	reader := strings.NewReader(config)
	c, err := NewFromReader(reader, "legacy", &Options{
		Headless: true,
	})
	if err != nil {
		t.Fatalf("NewFromReader failed: %v", err)
	}

	// Don't set any event handler

	if err := c.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Emit event directly - should not panic even without handler
	impl := c.(*conkyImpl)
	impl.emitEvent(EventWarning, "test warning")

	if err := c.Stop(); err != nil {
		t.Errorf("Stop failed: %v", err)
	}
}
