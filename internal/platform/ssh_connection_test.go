package platform

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestConnectionState_String(t *testing.T) {
	tests := []struct {
		state ConnectionState
		want  string
	}{
		{ConnectionStateDisconnected, "disconnected"},
		{ConnectionStateConnecting, "connecting"},
		{ConnectionStateConnected, "connected"},
		{ConnectionStateReconnecting, "reconnecting"},
		{ConnectionState(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.state.String(); got != tt.want {
				t.Errorf("ConnectionState.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSSHConnectionConfig_Defaults(t *testing.T) {
	// Create connection manager with empty config
	cm := newSSHConnectionManager("example.com:22", nil, SSHConnectionConfig{})

	// Check defaults are applied
	if cm.config.KeepAliveInterval != 30*time.Second {
		t.Errorf("KeepAliveInterval = %v, want 30s", cm.config.KeepAliveInterval)
	}
	if cm.config.KeepAliveTimeout != 15*time.Second {
		t.Errorf("KeepAliveTimeout = %v, want 15s", cm.config.KeepAliveTimeout)
	}
	if cm.config.InitialReconnectDelay != 1*time.Second {
		t.Errorf("InitialReconnectDelay = %v, want 1s", cm.config.InitialReconnectDelay)
	}
	if cm.config.MaxReconnectDelay != 5*time.Minute {
		t.Errorf("MaxReconnectDelay = %v, want 5m", cm.config.MaxReconnectDelay)
	}
	if cm.config.SessionPoolSize != 5 {
		t.Errorf("SessionPoolSize = %v, want 5", cm.config.SessionPoolSize)
	}
	if cm.config.SessionIdleTimeout != 1*time.Minute {
		t.Errorf("SessionIdleTimeout = %v, want 1m", cm.config.SessionIdleTimeout)
	}
}

func TestSSHConnectionConfig_CustomValues(t *testing.T) {
	config := SSHConnectionConfig{
		KeepAliveInterval:    1 * time.Minute,
		KeepAliveTimeout:     30 * time.Second,
		InitialReconnectDelay: 5 * time.Second,
		MaxReconnectDelay:    10 * time.Minute,
		SessionPoolSize:      10,
		SessionIdleTimeout:   5 * time.Minute,
		MaxReconnectAttempts: 3,
	}

	cm := newSSHConnectionManager("example.com:22", nil, config)

	if cm.config.KeepAliveInterval != 1*time.Minute {
		t.Errorf("KeepAliveInterval = %v, want 1m", cm.config.KeepAliveInterval)
	}
	if cm.config.MaxReconnectAttempts != 3 {
		t.Errorf("MaxReconnectAttempts = %v, want 3", cm.config.MaxReconnectAttempts)
	}
}

func TestSSHConnectionManager_StateTransitions(t *testing.T) {
	var transitions []struct{ from, to ConnectionState }

	config := SSHConnectionConfig{
		OnStateChange: func(from, to ConnectionState) {
			transitions = append(transitions, struct{ from, to ConnectionState }{from, to})
		},
	}

	cm := newSSHConnectionManager("example.com:22", nil, config)

	// Verify initial state
	if cm.State() != ConnectionStateDisconnected {
		t.Errorf("Initial state = %v, want disconnected", cm.State())
	}

	// Test state transition
	if !cm.setState(ConnectionStateDisconnected, ConnectionStateConnecting) {
		t.Error("setState should succeed for valid transition")
	}

	if cm.State() != ConnectionStateConnecting {
		t.Errorf("State = %v, want connecting", cm.State())
	}

	// Verify callback was called
	if len(transitions) != 1 {
		t.Errorf("Expected 1 transition, got %d", len(transitions))
	}
	if transitions[0].from != ConnectionStateDisconnected || transitions[0].to != ConnectionStateConnecting {
		t.Errorf("Transition = %v->%v, want disconnected->connecting",
			transitions[0].from, transitions[0].to)
	}
}

func TestSSHConnectionManager_Stats(t *testing.T) {
	cm := newSSHConnectionManager("example.com:22", nil, SSHConnectionConfig{})

	// Initial stats should be empty
	stats := cm.Stats()
	if stats.State != ConnectionStateDisconnected {
		t.Errorf("Initial state = %v, want disconnected", stats.State)
	}
	if stats.SessionsCreated != 0 {
		t.Errorf("Initial SessionsCreated = %d, want 0", stats.SessionsCreated)
	}
	if stats.ReconnectAttempts != 0 {
		t.Errorf("Initial ReconnectAttempts = %d, want 0", stats.ReconnectAttempts)
	}
}

func TestSSHConnectionManager_IsHealthy(t *testing.T) {
	cm := newSSHConnectionManager("example.com:22", nil, SSHConnectionConfig{})

	// Not healthy when disconnected
	if cm.IsHealthy() {
		t.Error("Should not be healthy when disconnected")
	}

	// Set state to connected
	cm.state.Store(int32(ConnectionStateConnected))
	if !cm.IsHealthy() {
		t.Error("Should be healthy when connected")
	}

	// Set state to reconnecting
	cm.state.Store(int32(ConnectionStateReconnecting))
	if cm.IsHealthy() {
		t.Error("Should not be healthy when reconnecting")
	}
}

func TestSSHConnectionManager_ConcurrentStateAccess(t *testing.T) {
	cm := newSSHConnectionManager("example.com:22", nil, SSHConnectionConfig{})

	// Run concurrent state accesses
	done := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ {
			_ = cm.State()
			_ = cm.IsHealthy()
			_ = cm.Stats()
		}
		close(done)
	}()

	for i := 0; i < 100; i++ {
		cm.state.Store(int32(i % 4))
	}

	<-done
}

func TestSSHConnectionManager_Close(t *testing.T) {
	cm := newSSHConnectionManager("example.com:22", nil, SSHConnectionConfig{})

	ctx, cancel := context.WithCancel(context.Background())
	cm.ctx = ctx
	cm.cancel = cancel

	// Close should not error
	if err := cm.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Verify context is cancelled
	select {
	case <-ctx.Done():
		// Expected
	default:
		t.Error("Context should be cancelled after Close()")
	}
}

func TestSSHConnectionManager_SetLastError(t *testing.T) {
	cm := newSSHConnectionManager("example.com:22", nil, SSHConnectionConfig{})

	testErr := &testError{msg: "test error"}
	cm.setLastError(testErr)

	stats := cm.Stats()
	if stats.LastError == nil {
		t.Error("LastError should not be nil")
	}
	if stats.LastError.Error() != "test error" {
		t.Errorf("LastError = %v, want 'test error'", stats.LastError)
	}
	if stats.LastErrorTime.IsZero() {
		t.Error("LastErrorTime should not be zero")
	}
}

func TestSSHConnectionManager_IsConnectionError(t *testing.T) {
	cm := newSSHConnectionManager("example.com:22", nil, SSHConnectionConfig{})

	tests := []struct {
		name    string
		err     error
		isError bool
	}{
		{"nil error", nil, false},
		{"connection refused", &testError{msg: "connection refused"}, true},
		{"connection reset", &testError{msg: "connection reset by peer"}, true},
		{"broken pipe", &testError{msg: "broken pipe"}, true},
		{"timeout", &testError{msg: "i/o timeout"}, true},
		{"EOF", &testError{msg: "EOF"}, true},
		{"closed connection", &testError{msg: "use of closed network connection"}, true},
		{"no route", &testError{msg: "no route to host"}, true},
		{"network unreachable", &testError{msg: "network is unreachable"}, true},
		{"regular error", &testError{msg: "some other error"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cm.isConnectionError(tt.err); got != tt.isError {
				t.Errorf("isConnectionError() = %v, want %v", got, tt.isError)
			}
		})
	}
}

func TestCalculateBackoff(t *testing.T) {
	tests := []struct {
		attempt int
		initial time.Duration
		max     time.Duration
		want    time.Duration
	}{
		{0, time.Second, time.Minute, time.Second},
		{1, time.Second, time.Minute, time.Second},
		{2, time.Second, time.Minute, 2 * time.Second},
		{3, time.Second, time.Minute, 4 * time.Second},
		{4, time.Second, time.Minute, 8 * time.Second},
		{10, time.Second, time.Minute, time.Minute}, // Capped at max
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := calculateBackoff(tt.attempt, tt.initial, tt.max)
			if got != tt.want {
				t.Errorf("calculateBackoff(%d, %v, %v) = %v, want %v",
					tt.attempt, tt.initial, tt.max, got, tt.want)
			}
		})
	}
}

func TestSSHConnectionManager_AtomicCounters(t *testing.T) {
	cm := newSSHConnectionManager("example.com:22", nil, SSHConnectionConfig{})

	// Test atomic counters
	cm.sessionsCreated.Add(5)
	cm.sessionsReused.Add(3)
	cm.keepalivesSent.Add(10)
	cm.keepalivesFailed.Add(1)
	cm.totalReconnects.Add(2)

	stats := cm.Stats()
	if stats.SessionsCreated != 5 {
		t.Errorf("SessionsCreated = %d, want 5", stats.SessionsCreated)
	}
	if stats.SessionsReused != 3 {
		t.Errorf("SessionsReused = %d, want 3", stats.SessionsReused)
	}
	if stats.KeepalivesSent != 10 {
		t.Errorf("KeepalivesSent = %d, want 10", stats.KeepalivesSent)
	}
	if stats.KeepalivesFailed != 1 {
		t.Errorf("KeepalivesFailed = %d, want 1", stats.KeepalivesFailed)
	}
	if stats.TotalReconnects != 2 {
		t.Errorf("TotalReconnects = %d, want 2", stats.TotalReconnects)
	}
}

func TestContainsFunctions(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   bool
	}{
		{"Connection refused", "connection refused", true},
		{"CONNECTION REFUSED", "connection refused", true},
		{"error: Connection Refused here", "connection refused", true},
		{"no match here", "connection refused", false},
		{"", "test", false},
		{"test", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			if got := contains(tt.s, tt.substr); got != tt.want {
				t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}

// testError is a simple error type for testing.
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestSSHConnectionManager_ReconnectAttemptsTracking(t *testing.T) {
	cm := newSSHConnectionManager("example.com:22", nil, SSHConnectionConfig{})

	// Simulate reconnection attempts
	for i := int64(1); i <= 5; i++ {
		cm.reconnectAttempts.Store(i)
		if cm.reconnectAttempts.Load() != i {
			t.Errorf("reconnectAttempts = %d, want %d", cm.reconnectAttempts.Load(), i)
		}
	}

	// Reset attempts (as would happen on successful reconnect)
	cm.reconnectAttempts.Store(0)
	cm.totalReconnects.Add(1)

	stats := cm.Stats()
	if stats.ReconnectAttempts != 0 {
		t.Errorf("ReconnectAttempts after reset = %d, want 0", stats.ReconnectAttempts)
	}
	if stats.TotalReconnects != 1 {
		t.Errorf("TotalReconnects = %d, want 1", stats.TotalReconnects)
	}
}

func TestSSHConnectionManager_SetStateWithCallback(t *testing.T) {
	callbackCalled := atomic.Bool{}
	
	config := SSHConnectionConfig{
		OnStateChange: func(from, to ConnectionState) {
			callbackCalled.Store(true)
		},
	}

	cm := newSSHConnectionManager("example.com:22", nil, config)

	// Valid transition should call callback
	cm.setState(ConnectionStateDisconnected, ConnectionStateConnecting)
	if !callbackCalled.Load() {
		t.Error("OnStateChange callback should be called")
	}

	// Invalid transition (wrong 'from' state) should not succeed
	callbackCalled.Store(false)
	if cm.setState(ConnectionStateConnected, ConnectionStateReconnecting) {
		t.Error("setState should fail for invalid 'from' state")
	}
	if callbackCalled.Load() {
		t.Error("OnStateChange should not be called for failed transition")
	}
}
