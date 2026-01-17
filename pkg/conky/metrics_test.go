package conky

import (
	"testing"
	"time"
)

func TestNewMetrics(t *testing.T) {
	m := NewMetrics()
	if m == nil {
		t.Fatal("NewMetrics returned nil")
	}

	// Verify initial state is zero
	snap := m.Snapshot()
	if snap.Starts != 0 || snap.Stops != 0 || snap.ErrorsTotal != 0 {
		t.Error("New metrics should have zero values")
	}
}

func TestMetricsCounters(t *testing.T) {
	m := NewMetrics()

	// Test each counter increment
	m.IncrementStarts()
	m.IncrementStarts()
	m.IncrementStops()
	m.IncrementRestarts()
	m.IncrementConfigReloads()
	m.IncrementUpdateCycles()
	m.IncrementUpdateCycles()
	m.IncrementUpdateCycles()
	m.IncrementErrors()
	m.IncrementEventsEmitted()
	m.IncrementLuaExecutions()
	m.IncrementLuaErrors()
	m.IncrementRemoteCommands()

	snap := m.Snapshot()

	tests := []struct {
		name     string
		got      int64
		expected int64
	}{
		{"Starts", snap.Starts, 2},
		{"Stops", snap.Stops, 1},
		{"Restarts", snap.Restarts, 1},
		{"ConfigReloads", snap.ConfigReloads, 1},
		{"UpdateCycles", snap.UpdateCycles, 3},
		{"ErrorsTotal", snap.ErrorsTotal, 1},
		{"EventsEmitted", snap.EventsEmitted, 1},
		{"LuaExecutions", snap.LuaExecutions, 1},
		{"LuaErrors", snap.LuaErrors, 1},
		{"RemoteCommands", snap.RemoteCommands, 1},
	}

	for _, tt := range tests {
		if tt.got != tt.expected {
			t.Errorf("%s: got %d, expected %d", tt.name, tt.got, tt.expected)
		}
	}
}

func TestMetricsGauges(t *testing.T) {
	m := NewMetrics()

	// Test running gauge
	m.SetRunning(true)
	snap := m.Snapshot()
	if !snap.Running {
		t.Error("Running should be true after SetRunning(true)")
	}

	m.SetRunning(false)
	snap = m.Snapshot()
	if snap.Running {
		t.Error("Running should be false after SetRunning(false)")
	}

	// Test active monitors gauge
	m.SetActiveMonitors(5)
	snap = m.Snapshot()
	if snap.ActiveMonitors != 5 {
		t.Errorf("ActiveMonitors: got %d, expected 5", snap.ActiveMonitors)
	}

	m.SetActiveMonitors(0)
	snap = m.Snapshot()
	if snap.ActiveMonitors != 0 {
		t.Errorf("ActiveMonitors: got %d, expected 0", snap.ActiveMonitors)
	}
}

func TestMetricsLatency(t *testing.T) {
	m := NewMetrics()

	// Record some latencies
	m.RecordUpdateLatency(10 * time.Millisecond)
	m.RecordUpdateLatency(20 * time.Millisecond)
	m.RecordUpdateLatency(30 * time.Millisecond)

	snap := m.Snapshot()

	// Average of 10, 20, 30 = 20ms
	expectedAvg := 20 * time.Millisecond
	if snap.UpdateLatencyAvg != expectedAvg {
		t.Errorf("UpdateLatencyAvg: got %v, expected %v", snap.UpdateLatencyAvg, expectedAvg)
	}

	// Test Lua latency
	m.RecordLuaLatency(5 * time.Millisecond)
	m.RecordLuaLatency(15 * time.Millisecond)

	snap = m.Snapshot()
	expectedLuaAvg := 10 * time.Millisecond
	if snap.LuaLatencyAvg != expectedLuaAvg {
		t.Errorf("LuaLatencyAvg: got %v, expected %v", snap.LuaLatencyAvg, expectedLuaAvg)
	}

	// Test render latency
	m.RecordRenderLatency(16 * time.Millisecond)
	snap = m.Snapshot()
	if snap.RenderLatencyAvg != 16*time.Millisecond {
		t.Errorf("RenderLatencyAvg: got %v, expected 16ms", snap.RenderLatencyAvg)
	}
}

func TestMetricsLatencyZeroCount(t *testing.T) {
	m := NewMetrics()

	// Snapshot with no latency recordings should not panic
	snap := m.Snapshot()

	if snap.UpdateLatencyAvg != 0 {
		t.Errorf("UpdateLatencyAvg should be 0 with no recordings, got %v", snap.UpdateLatencyAvg)
	}
	if snap.LuaLatencyAvg != 0 {
		t.Errorf("LuaLatencyAvg should be 0 with no recordings, got %v", snap.LuaLatencyAvg)
	}
	if snap.RenderLatencyAvg != 0 {
		t.Errorf("RenderLatencyAvg should be 0 with no recordings, got %v", snap.RenderLatencyAvg)
	}
}

func TestMetricsReset(t *testing.T) {
	m := NewMetrics()

	// Add some values
	m.IncrementStarts()
	m.IncrementErrors()
	m.SetRunning(true)
	m.SetActiveMonitors(3)
	m.RecordUpdateLatency(100 * time.Millisecond)

	// Verify they're set
	snap := m.Snapshot()
	if snap.Starts == 0 || snap.ErrorsTotal == 0 {
		t.Error("Metrics should have values before reset")
	}

	// Reset
	m.Reset()

	// Verify all zero
	snap = m.Snapshot()
	if snap.Starts != 0 {
		t.Errorf("Starts should be 0 after reset, got %d", snap.Starts)
	}
	if snap.Stops != 0 {
		t.Errorf("Stops should be 0 after reset, got %d", snap.Stops)
	}
	if snap.ErrorsTotal != 0 {
		t.Errorf("ErrorsTotal should be 0 after reset, got %d", snap.ErrorsTotal)
	}
	if snap.Running {
		t.Error("Running should be false after reset")
	}
	if snap.ActiveMonitors != 0 {
		t.Errorf("ActiveMonitors should be 0 after reset, got %d", snap.ActiveMonitors)
	}
	if snap.UpdateLatencyAvg != 0 {
		t.Errorf("UpdateLatencyAvg should be 0 after reset, got %v", snap.UpdateLatencyAvg)
	}
}

func TestDefaultMetrics(t *testing.T) {
	m1 := DefaultMetrics()
	m2 := DefaultMetrics()

	if m1 != m2 {
		t.Error("DefaultMetrics should return the same instance")
	}

	if m1 == nil {
		t.Error("DefaultMetrics should not return nil")
	}
}

func TestMetricsConcurrentAccess(t *testing.T) {
	m := NewMetrics()
	done := make(chan bool)

	// Concurrent increments
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				m.IncrementStarts()
				m.IncrementErrors()
				m.RecordUpdateLatency(time.Millisecond)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	snap := m.Snapshot()
	if snap.Starts != 1000 {
		t.Errorf("Expected 1000 starts, got %d", snap.Starts)
	}
	if snap.ErrorsTotal != 1000 {
		t.Errorf("Expected 1000 errors, got %d", snap.ErrorsTotal)
	}
}

func TestMetricsSnapshotIsIsolated(t *testing.T) {
	m := NewMetrics()
	m.IncrementStarts()

	snap1 := m.Snapshot()

	// Modify metrics after snapshot
	m.IncrementStarts()
	m.IncrementStarts()

	// Original snapshot should be unchanged
	if snap1.Starts != 1 {
		t.Errorf("Snapshot should be isolated, got Starts=%d", snap1.Starts)
	}

	// New snapshot should have updated values
	snap2 := m.Snapshot()
	if snap2.Starts != 3 {
		t.Errorf("New snapshot should have Starts=3, got %d", snap2.Starts)
	}
}

func TestSafeDivide(t *testing.T) {
	tests := []struct {
		total    int64
		count    int64
		expected time.Duration
	}{
		{100, 10, 10 * time.Nanosecond},
		{0, 0, 0},
		{100, 0, 0}, // Divide by zero returns 0
		{0, 10, 0},
	}

	for _, tt := range tests {
		result := safeDivide(tt.total, tt.count)
		if result != tt.expected {
			t.Errorf("safeDivide(%d, %d) = %v, expected %v",
				tt.total, tt.count, result, tt.expected)
		}
	}
}

func TestRegisterExpvarIdempotent(t *testing.T) {
	m := NewMetrics()

	// Should not panic when called multiple times
	m.RegisterExpvar()
	m.RegisterExpvar()
	m.RegisterExpvar()

	// Verify the registered flag is set
	if !m.registered.Load() {
		t.Error("registered should be true after RegisterExpvar")
	}
}
