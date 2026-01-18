package conky

import (
	"expvar"
	"sync/atomic"
	"time"
)

// Metrics provides application-level metrics collection for go-conky.
// It uses Go's expvar package for exposition, which can be accessed via the
// /debug/vars HTTP endpoint when an HTTP server is running.
//
// Thread-safe for concurrent use.
//
// Example usage:
//
//	metrics := conky.NewMetrics()
//	metrics.IncrementConfigReloads()
//	metrics.RecordUpdateLatency(15 * time.Millisecond)
//
//	// For HTTP exposition, import expvar's HTTP handler:
//	// import _ "expvar"
//	// This registers /debug/vars automatically.
type Metrics struct {
	// Counters
	starts         atomic.Int64
	stops          atomic.Int64
	restarts       atomic.Int64
	configReloads  atomic.Int64
	updateCycles   atomic.Int64
	errorsTotal    atomic.Int64
	eventsEmitted  atomic.Int64
	luaExecutions  atomic.Int64
	luaErrors      atomic.Int64
	remoteCommands atomic.Int64

	// Latency tracking (stored as nanoseconds)
	updateLatencyNs    atomic.Int64
	updateLatencyCount atomic.Int64
	luaLatencyNs       atomic.Int64
	luaLatencyCount    atomic.Int64
	renderLatencyNs    atomic.Int64
	renderLatencyCount atomic.Int64

	// Current state gauges
	currentlyRunning atomic.Int32
	activeMonitors   atomic.Int32

	// Registration tracking to prevent duplicate expvar registration
	registered atomic.Bool
}

// NewMetrics creates a new Metrics instance.
// Call RegisterExpvar() to expose metrics via the /debug/vars endpoint.
func NewMetrics() *Metrics {
	return &Metrics{}
}

// RegisterExpvar registers all metrics with Go's expvar package.
// This makes metrics available at /debug/vars when an HTTP server is running.
// Safe to call multiple times; subsequent calls are no-ops.
func (m *Metrics) RegisterExpvar() {
	if m.registered.Swap(true) {
		return // Already registered
	}

	// Counters
	expvar.Publish("conky_starts_total", expvar.Func(func() any { return m.starts.Load() }))
	expvar.Publish("conky_stops_total", expvar.Func(func() any { return m.stops.Load() }))
	expvar.Publish("conky_restarts_total", expvar.Func(func() any { return m.restarts.Load() }))
	expvar.Publish("conky_config_reloads_total", expvar.Func(func() any { return m.configReloads.Load() }))
	expvar.Publish("conky_update_cycles_total", expvar.Func(func() any { return m.updateCycles.Load() }))
	expvar.Publish("conky_errors_total", expvar.Func(func() any { return m.errorsTotal.Load() }))
	expvar.Publish("conky_events_emitted_total", expvar.Func(func() any { return m.eventsEmitted.Load() }))
	expvar.Publish("conky_lua_executions_total", expvar.Func(func() any { return m.luaExecutions.Load() }))
	expvar.Publish("conky_lua_errors_total", expvar.Func(func() any { return m.luaErrors.Load() }))
	expvar.Publish("conky_remote_commands_total", expvar.Func(func() any { return m.remoteCommands.Load() }))

	// Gauges
	expvar.Publish("conky_running", expvar.Func(func() any { return m.currentlyRunning.Load() }))
	expvar.Publish("conky_active_monitors", expvar.Func(func() any { return m.activeMonitors.Load() }))

	// Latency averages (milliseconds)
	expvar.Publish("conky_update_latency_avg_ms", expvar.Func(func() any {
		count := m.updateLatencyCount.Load()
		if count == 0 {
			return float64(0)
		}
		return float64(m.updateLatencyNs.Load()) / float64(count) / 1e6
	}))
	expvar.Publish("conky_lua_latency_avg_ms", expvar.Func(func() any {
		count := m.luaLatencyCount.Load()
		if count == 0 {
			return float64(0)
		}
		return float64(m.luaLatencyNs.Load()) / float64(count) / 1e6
	}))
	expvar.Publish("conky_render_latency_avg_ms", expvar.Func(func() any {
		count := m.renderLatencyCount.Load()
		if count == 0 {
			return float64(0)
		}
		return float64(m.renderLatencyNs.Load()) / float64(count) / 1e6
	}))
}

// Snapshot returns a point-in-time copy of all metrics.
// Useful for testing or custom metric exposition.
func (m *Metrics) Snapshot() MetricsSnapshot {
	updateCount := m.updateLatencyCount.Load()
	luaCount := m.luaLatencyCount.Load()
	renderCount := m.renderLatencyCount.Load()

	return MetricsSnapshot{
		Starts:         m.starts.Load(),
		Stops:          m.stops.Load(),
		Restarts:       m.restarts.Load(),
		ConfigReloads:  m.configReloads.Load(),
		UpdateCycles:   m.updateCycles.Load(),
		ErrorsTotal:    m.errorsTotal.Load(),
		EventsEmitted:  m.eventsEmitted.Load(),
		LuaExecutions:  m.luaExecutions.Load(),
		LuaErrors:      m.luaErrors.Load(),
		RemoteCommands: m.remoteCommands.Load(),

		Running:        m.currentlyRunning.Load() > 0,
		ActiveMonitors: int(m.activeMonitors.Load()),

		UpdateLatencyAvg: safeDivide(m.updateLatencyNs.Load(), updateCount),
		LuaLatencyAvg:    safeDivide(m.luaLatencyNs.Load(), luaCount),
		RenderLatencyAvg: safeDivide(m.renderLatencyNs.Load(), renderCount),
	}
}

// MetricsSnapshot is a point-in-time copy of all metrics.
type MetricsSnapshot struct {
	// Counters
	Starts         int64
	Stops          int64
	Restarts       int64
	ConfigReloads  int64
	UpdateCycles   int64
	ErrorsTotal    int64
	EventsEmitted  int64
	LuaExecutions  int64
	LuaErrors      int64
	RemoteCommands int64

	// Gauges
	Running        bool
	ActiveMonitors int

	// Latency averages
	UpdateLatencyAvg time.Duration
	LuaLatencyAvg    time.Duration
	RenderLatencyAvg time.Duration
}

// Counter increment methods

// IncrementStarts records a start operation.
func (m *Metrics) IncrementStarts() {
	m.starts.Add(1)
}

// IncrementStops records a stop operation.
func (m *Metrics) IncrementStops() {
	m.stops.Add(1)
}

// IncrementRestarts records a restart operation.
func (m *Metrics) IncrementRestarts() {
	m.restarts.Add(1)
}

// IncrementConfigReloads records a configuration reload.
func (m *Metrics) IncrementConfigReloads() {
	m.configReloads.Add(1)
}

// IncrementUpdateCycles records an update cycle completion.
func (m *Metrics) IncrementUpdateCycles() {
	m.updateCycles.Add(1)
}

// IncrementErrors records an error occurrence.
func (m *Metrics) IncrementErrors() {
	m.errorsTotal.Add(1)
}

// IncrementEventsEmitted records an event emission.
func (m *Metrics) IncrementEventsEmitted() {
	m.eventsEmitted.Add(1)
}

// IncrementLuaExecutions records a Lua script execution.
func (m *Metrics) IncrementLuaExecutions() {
	m.luaExecutions.Add(1)
}

// IncrementLuaErrors records a Lua script error.
func (m *Metrics) IncrementLuaErrors() {
	m.luaErrors.Add(1)
}

// IncrementRemoteCommands records a remote SSH command execution.
func (m *Metrics) IncrementRemoteCommands() {
	m.remoteCommands.Add(1)
}

// Gauge methods

// SetRunning updates the running state gauge.
func (m *Metrics) SetRunning(running bool) {
	if running {
		m.currentlyRunning.Store(1)
	} else {
		m.currentlyRunning.Store(0)
	}
}

// SetActiveMonitors updates the active monitors gauge.
func (m *Metrics) SetActiveMonitors(count int) {
	m.activeMonitors.Store(int32(count))
}

// Latency recording methods

// RecordUpdateLatency records the duration of an update cycle.
func (m *Metrics) RecordUpdateLatency(d time.Duration) {
	m.updateLatencyNs.Add(d.Nanoseconds())
	m.updateLatencyCount.Add(1)
}

// RecordLuaLatency records the duration of a Lua execution.
func (m *Metrics) RecordLuaLatency(d time.Duration) {
	m.luaLatencyNs.Add(d.Nanoseconds())
	m.luaLatencyCount.Add(1)
}

// RecordRenderLatency records the duration of a render operation.
func (m *Metrics) RecordRenderLatency(d time.Duration) {
	m.renderLatencyNs.Add(d.Nanoseconds())
	m.renderLatencyCount.Add(1)
}

// Reset clears all metrics. Useful for testing.
func (m *Metrics) Reset() {
	m.starts.Store(0)
	m.stops.Store(0)
	m.restarts.Store(0)
	m.configReloads.Store(0)
	m.updateCycles.Store(0)
	m.errorsTotal.Store(0)
	m.eventsEmitted.Store(0)
	m.luaExecutions.Store(0)
	m.luaErrors.Store(0)
	m.remoteCommands.Store(0)

	m.updateLatencyNs.Store(0)
	m.updateLatencyCount.Store(0)
	m.luaLatencyNs.Store(0)
	m.luaLatencyCount.Store(0)
	m.renderLatencyNs.Store(0)
	m.renderLatencyCount.Store(0)

	m.currentlyRunning.Store(0)
	m.activeMonitors.Store(0)
}

// safeDivide performs safe division, returning 0 for divide by zero.
func safeDivide(total, count int64) time.Duration {
	if count == 0 {
		return 0
	}
	return time.Duration(total / count)
}

// defaultMetrics is a global metrics instance for convenience.
var defaultMetrics = NewMetrics()

// DefaultMetrics returns the global default Metrics instance.
// This can be used when a single application-wide metrics collector is sufficient.
func DefaultMetrics() *Metrics {
	return defaultMetrics
}
