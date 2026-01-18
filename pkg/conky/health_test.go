package conky

import (
	"bytes"
	"testing"
	"time"
)

func TestHealthCheck_IsHealthy(t *testing.T) {
	tests := []struct {
		name     string
		status   HealthStatus
		expected bool
	}{
		{"ok status", HealthOK, true},
		{"degraded status", HealthDegraded, false},
		{"unhealthy status", HealthUnhealthy, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := HealthCheck{Status: tt.status}
			if got := h.IsHealthy(); got != tt.expected {
				t.Errorf("IsHealthy() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestHealthCheck_IsDegraded(t *testing.T) {
	tests := []struct {
		name     string
		status   HealthStatus
		expected bool
	}{
		{"ok status", HealthOK, false},
		{"degraded status", HealthDegraded, true},
		{"unhealthy status", HealthUnhealthy, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := HealthCheck{Status: tt.status}
			if got := h.IsDegraded(); got != tt.expected {
				t.Errorf("IsDegraded() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestHealthCheck_IsUnhealthy(t *testing.T) {
	tests := []struct {
		name     string
		status   HealthStatus
		expected bool
	}{
		{"ok status", HealthOK, false},
		{"degraded status", HealthDegraded, false},
		{"unhealthy status", HealthUnhealthy, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := HealthCheck{Status: tt.status}
			if got := h.IsUnhealthy(); got != tt.expected {
				t.Errorf("IsUnhealthy() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestHealth_NotRunning(t *testing.T) {
	cfg := []byte(`
conky.config = {
    update_interval = 1,
}
conky.text = [[test]]
`)
	c, err := NewFromReader(bytes.NewReader(cfg), FormatLua, nil)
	if err != nil {
		t.Fatalf("NewFromReader() error = %v", err)
	}

	// Check health before starting
	h := c.Health()

	if h.Status != HealthUnhealthy {
		t.Errorf("Health().Status = %v, want %v", h.Status, HealthUnhealthy)
	}

	if h.IsHealthy() {
		t.Error("Health().IsHealthy() = true for stopped instance")
	}

	if h.Uptime != 0 {
		t.Errorf("Health().Uptime = %v, want 0", h.Uptime)
	}

	if h.Timestamp.IsZero() {
		t.Error("Health().Timestamp is zero")
	}

	// Check components
	if inst, ok := h.Components["instance"]; !ok {
		t.Error("Health().Components missing 'instance'")
	} else if inst.Status != HealthUnhealthy {
		t.Errorf("instance component status = %v, want %v", inst.Status, HealthUnhealthy)
	}

	if mon, ok := h.Components["monitor"]; !ok {
		t.Error("Health().Components missing 'monitor'")
	} else if mon.Status != HealthUnhealthy {
		t.Errorf("monitor component status = %v, want %v", mon.Status, HealthUnhealthy)
	}
}

func TestHealth_Running(t *testing.T) {
	cfg := []byte(`
conky.config = {
    update_interval = 1,
}
conky.text = [[test]]
`)
	opts := DefaultOptions()
	opts.Headless = true

	c, err := NewFromReader(bytes.NewReader(cfg), FormatLua, &opts)
	if err != nil {
		t.Fatalf("NewFromReader() error = %v", err)
	}

	if err := c.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() { _ = c.Stop() }()

	// Wait for monitor to initialize
	time.Sleep(50 * time.Millisecond)

	h := c.Health()

	if h.Status != HealthOK {
		t.Errorf("Health().Status = %v, want %v", h.Status, HealthOK)
	}

	if !h.IsHealthy() {
		t.Error("Health().IsHealthy() = false for running instance")
	}

	if h.Uptime <= 0 {
		t.Errorf("Health().Uptime = %v, want > 0", h.Uptime)
	}

	// Check components
	if inst, ok := h.Components["instance"]; !ok {
		t.Error("Health().Components missing 'instance'")
	} else if inst.Status != HealthOK {
		t.Errorf("instance component status = %v, want %v", inst.Status, HealthOK)
	}

	if mon, ok := h.Components["monitor"]; !ok {
		t.Error("Health().Components missing 'monitor'")
	} else if mon.Status != HealthOK {
		t.Errorf("monitor component status = %v, want %v", mon.Status, HealthOK)
	}

	if errs, ok := h.Components["errors"]; !ok {
		t.Error("Health().Components missing 'errors'")
	} else if errs.Status != HealthOK {
		t.Errorf("errors component status = %v, want %v", errs.Status, HealthOK)
	}
}

func TestHealth_AfterStop(t *testing.T) {
	cfg := []byte(`
conky.config = {
    update_interval = 1,
}
conky.text = [[test]]
`)
	opts := DefaultOptions()
	opts.Headless = true

	c, err := NewFromReader(bytes.NewReader(cfg), FormatLua, &opts)
	if err != nil {
		t.Fatalf("NewFromReader() error = %v", err)
	}

	if err := c.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Stop the instance
	if err := c.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	h := c.Health()

	if h.Status != HealthUnhealthy {
		t.Errorf("Health().Status = %v, want %v after stop", h.Status, HealthUnhealthy)
	}

	if h.Uptime != 0 {
		t.Errorf("Health().Uptime = %v, want 0 after stop", h.Uptime)
	}
}

func TestHealthStatus_Values(t *testing.T) {
	// Verify the string values are as documented
	if HealthOK != "ok" {
		t.Errorf("HealthOK = %q, want %q", HealthOK, "ok")
	}
	if HealthDegraded != "degraded" {
		t.Errorf("HealthDegraded = %q, want %q", HealthDegraded, "degraded")
	}
	if HealthUnhealthy != "unhealthy" {
		t.Errorf("HealthUnhealthy = %q, want %q", HealthUnhealthy, "unhealthy")
	}
}
