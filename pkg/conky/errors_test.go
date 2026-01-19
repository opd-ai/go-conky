package conky

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestErrorCategory_String(t *testing.T) {
	tests := []struct {
		category ErrorCategory
		want     string
	}{
		{ErrorCategoryUnknown, "unknown"},
		{ErrorCategoryConfig, "config"},
		{ErrorCategoryLua, "lua"},
		{ErrorCategoryRender, "render"},
		{ErrorCategoryMonitor, "monitor"},
		{ErrorCategoryRemote, "remote"},
		{ErrorCategoryIO, "io"},
		{ErrorCategoryNetwork, "network"},
		{ErrorCategory(99), "unknown"}, // Invalid category
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.category.String(); got != tt.want {
				t.Errorf("ErrorCategory.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrorSeverity_String(t *testing.T) {
	tests := []struct {
		severity ErrorSeverity
		want     string
	}{
		{SeverityInfo, "info"},
		{SeverityWarning, "warning"},
		{SeverityError, "error"},
		{SeverityCritical, "critical"},
		{ErrorSeverity(99), "unknown"}, // Invalid severity
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.severity.String(); got != tt.want {
				t.Errorf("ErrorSeverity.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCategorizedError(t *testing.T) {
	t.Run("Error method", func(t *testing.T) {
		err := NewCategorizedError(
			errors.New("test error"),
			ErrorCategoryConfig,
			SeverityError,
		)

		got := err.Error()
		if got != "[error/config] test error" {
			t.Errorf("CategorizedError.Error() = %v, want [error/config] test error", got)
		}
	})

	t.Run("Error method with nil error", func(t *testing.T) {
		err := &CategorizedError{
			Category: ErrorCategoryLua,
			Severity: SeverityWarning,
		}

		got := err.Error()
		if got != "[warning/lua] (no error)" {
			t.Errorf("CategorizedError.Error() = %v, want [warning/lua] (no error)", got)
		}
	})

	t.Run("Unwrap returns underlying error", func(t *testing.T) {
		underlying := errors.New("underlying")
		err := NewCategorizedError(underlying, ErrorCategoryIO, SeverityCritical)

		if !errors.Is(err, underlying) {
			t.Error("errors.Is failed to match underlying error")
		}
	})

	t.Run("WithContext adds metadata", func(t *testing.T) {
		err := NewCategorizedError(errors.New("test"), ErrorCategoryRemote, SeverityError).
			WithContext("host", "server.example.com").
			WithContext("port", "22")

		if err.Context["host"] != "server.example.com" {
			t.Errorf("Context[host] = %v, want server.example.com", err.Context["host"])
		}
		if err.Context["port"] != "22" {
			t.Errorf("Context[port] = %v, want 22", err.Context["port"])
		}
	})

	t.Run("WithContext handles nil context", func(t *testing.T) {
		err := &CategorizedError{}
		err = err.WithContext("key", "value")

		if err.Context["key"] != "value" {
			t.Errorf("Context[key] = %v, want value", err.Context["key"])
		}
	})
}

func TestDeepCopyContext(t *testing.T) {
	t.Run("returns nil for nil input", func(t *testing.T) {
		result := deepCopyContext(nil)
		if result != nil {
			t.Errorf("deepCopyContext(nil) = %v, want nil", result)
		}
	})

	t.Run("creates independent copy", func(t *testing.T) {
		original := map[string]string{"key": "value"}
		copied := deepCopyContext(original)

		// Modify original
		original["key"] = "modified"
		original["new"] = "added"

		// Copied should be unaffected
		if copied["key"] != "value" {
			t.Errorf("copied[key] = %v, want value", copied["key"])
		}
		if _, exists := copied["new"]; exists {
			t.Error("copied should not have 'new' key")
		}
	})
}

func TestCategorizedErrorDeepCopy(t *testing.T) {
	t.Run("creates independent copy of Context", func(t *testing.T) {
		original := NewCategorizedError(errors.New("test"), ErrorCategoryConfig, SeverityError).
			WithContext("host", "original")

		copied := original.deepCopy()

		// Modify original
		original.Context["host"] = "modified"
		original.Context["new"] = "added"

		// Copied should be unaffected
		if copied.Context["host"] != "original" {
			t.Errorf("copied.Context[host] = %v, want original", copied.Context["host"])
		}
		if _, exists := copied.Context["new"]; exists {
			t.Error("copied.Context should not have 'new' key")
		}
	})
}

func TestErrorTracker_DeepCopyOnRecord(t *testing.T) {
	tracker := NewErrorTracker(DefaultErrorTrackerConfig())

	err := NewCategorizedError(errors.New("test"), ErrorCategoryConfig, SeverityError).
		WithContext("key", "original")

	tracker.Record(err)

	// Modify original after recording
	err.Context["key"] = "modified"

	// Tracker should have independent copy
	recent := tracker.RecentErrors(1)
	if len(recent) != 1 {
		t.Fatalf("Expected 1 recent error, got %d", len(recent))
	}
	if recent[0].Context["key"] != "original" {
		t.Errorf("Tracker error context[key] = %v, want original", recent[0].Context["key"])
	}
}

func TestErrorTracker_DeepCopyOnRecentErrors(t *testing.T) {
	tracker := NewErrorTracker(DefaultErrorTrackerConfig())

	tracker.Record(NewCategorizedError(errors.New("test"), ErrorCategoryConfig, SeverityError).
		WithContext("key", "original"))

	// Get recent errors
	recent1 := tracker.RecentErrors(1)
	recent1[0].Context["key"] = "modified"

	// Get recent errors again
	recent2 := tracker.RecentErrors(1)

	// Second retrieval should have original value
	if recent2[0].Context["key"] != "original" {
		t.Errorf("recent2[0].Context[key] = %v, want original", recent2[0].Context["key"])
	}
}

func TestNewErrorTracker(t *testing.T) {
	t.Run("default configuration", func(t *testing.T) {
		cfg := DefaultErrorTrackerConfig()
		tracker := NewErrorTracker(cfg)

		if tracker.maxErrors != 1000 {
			t.Errorf("maxErrors = %d, want 1000", tracker.maxErrors)
		}
		if tracker.retentionTime != time.Hour {
			t.Errorf("retentionTime = %v, want 1h", tracker.retentionTime)
		}
		if tracker.alertCooldown != 5*time.Minute {
			t.Errorf("alertCooldown = %v, want 5m", tracker.alertCooldown)
		}
	})

	t.Run("zero values use defaults", func(t *testing.T) {
		tracker := NewErrorTracker(ErrorTrackerConfig{})

		if tracker.maxErrors != 1000 {
			t.Errorf("maxErrors = %d, want 1000", tracker.maxErrors)
		}
	})
}

func TestErrorTracker_Record(t *testing.T) {
	t.Run("records error", func(t *testing.T) {
		tracker := NewErrorTracker(DefaultErrorTrackerConfig())
		err := NewCategorizedError(errors.New("test"), ErrorCategoryConfig, SeverityError)

		tracker.Record(err)

		stats := tracker.Stats()
		if stats.TotalErrors != 1 {
			t.Errorf("TotalErrors = %d, want 1", stats.TotalErrors)
		}
		if stats.ErrorsByCategory[ErrorCategoryConfig] != 1 {
			t.Errorf("ErrorsByCategory[Config] = %d, want 1", stats.ErrorsByCategory[ErrorCategoryConfig])
		}
	})

	t.Run("ignores nil error", func(t *testing.T) {
		tracker := NewErrorTracker(DefaultErrorTrackerConfig())
		tracker.Record(nil)

		stats := tracker.Stats()
		if stats.TotalErrors != 0 {
			t.Errorf("TotalErrors = %d, want 0", stats.TotalErrors)
		}
	})

	t.Run("updates category counters", func(t *testing.T) {
		tracker := NewErrorTracker(DefaultErrorTrackerConfig())

		tracker.Record(NewCategorizedError(errors.New("1"), ErrorCategoryLua, SeverityError))
		tracker.Record(NewCategorizedError(errors.New("2"), ErrorCategoryLua, SeverityError))
		tracker.Record(NewCategorizedError(errors.New("3"), ErrorCategoryConfig, SeverityError))

		stats := tracker.Stats()
		var luaCount, configCount int64
		for _, cc := range stats.TotalByCategory {
			if cc.Category == ErrorCategoryLua {
				luaCount = cc.Count
			}
			if cc.Category == ErrorCategoryConfig {
				configCount = cc.Count
			}
		}

		if luaCount != 2 {
			t.Errorf("Lua count = %d, want 2", luaCount)
		}
		if configCount != 1 {
			t.Errorf("Config count = %d, want 1", configCount)
		}
	})

	t.Run("prunes when over capacity", func(t *testing.T) {
		cfg := DefaultErrorTrackerConfig()
		cfg.MaxErrors = 5
		tracker := NewErrorTracker(cfg)

		for i := 0; i < 10; i++ {
			tracker.Record(NewCategorizedError(errors.New("test"), ErrorCategoryConfig, SeverityError))
		}

		stats := tracker.Stats()
		if stats.TotalErrors > 5 {
			t.Errorf("TotalErrors = %d, want <= 5", stats.TotalErrors)
		}
	})
}

func TestErrorTracker_AlertConditions(t *testing.T) {
	t.Run("triggers alert when threshold met", func(t *testing.T) {
		cfg := DefaultErrorTrackerConfig()
		cfg.AlertCooldown = 0 // Disable cooldown for testing
		tracker := NewErrorTracker(cfg)

		var alertTriggered atomic.Bool
		var alertCount atomic.Int32

		tracker.AddCondition(AlertCondition{
			Category:    ErrorCategoryConfig,
			MinSeverity: SeverityError,
			Threshold:   3,
			Window:      time.Minute,
		})

		tracker.SetAlertHandler(func(cond AlertCondition, count int, recent []CategorizedError) {
			alertTriggered.Store(true)
			alertCount.Store(int32(count))
		})

		// Record errors below threshold
		tracker.Record(NewCategorizedError(errors.New("1"), ErrorCategoryConfig, SeverityError))
		tracker.Record(NewCategorizedError(errors.New("2"), ErrorCategoryConfig, SeverityError))

		time.Sleep(10 * time.Millisecond)
		if alertTriggered.Load() {
			t.Error("Alert triggered before threshold")
		}

		// Record error to meet threshold
		tracker.Record(NewCategorizedError(errors.New("3"), ErrorCategoryConfig, SeverityError))

		time.Sleep(50 * time.Millisecond) // Wait for async handler
		if !alertTriggered.Load() {
			t.Error("Alert not triggered after threshold met")
		}
		if alertCount.Load() != 3 {
			t.Errorf("Alert count = %d, want 3", alertCount.Load())
		}
	})

	t.Run("respects category filter", func(t *testing.T) {
		cfg := DefaultErrorTrackerConfig()
		cfg.AlertCooldown = 0
		tracker := NewErrorTracker(cfg)

		var alertTriggered atomic.Bool

		tracker.AddCondition(AlertCondition{
			Category:    ErrorCategoryLua,
			MinSeverity: SeverityError,
			Threshold:   2,
			Window:      time.Minute,
		})

		tracker.SetAlertHandler(func(cond AlertCondition, count int, recent []CategorizedError) {
			alertTriggered.Store(true)
		})

		// Record non-matching category
		tracker.Record(NewCategorizedError(errors.New("1"), ErrorCategoryConfig, SeverityError))
		tracker.Record(NewCategorizedError(errors.New("2"), ErrorCategoryConfig, SeverityError))
		tracker.Record(NewCategorizedError(errors.New("3"), ErrorCategoryConfig, SeverityError))

		time.Sleep(50 * time.Millisecond)
		if alertTriggered.Load() {
			t.Error("Alert triggered for non-matching category")
		}
	})

	t.Run("respects severity filter", func(t *testing.T) {
		cfg := DefaultErrorTrackerConfig()
		cfg.AlertCooldown = 0
		tracker := NewErrorTracker(cfg)

		var alertTriggered atomic.Bool

		tracker.AddCondition(AlertCondition{
			Category:    ErrorCategoryUnknown, // Any category
			MinSeverity: SeverityCritical,
			Threshold:   2,
			Window:      time.Minute,
		})

		tracker.SetAlertHandler(func(cond AlertCondition, count int, recent []CategorizedError) {
			alertTriggered.Store(true)
		})

		// Record lower severity errors
		tracker.Record(NewCategorizedError(errors.New("1"), ErrorCategoryConfig, SeverityWarning))
		tracker.Record(NewCategorizedError(errors.New("2"), ErrorCategoryConfig, SeverityError))
		tracker.Record(NewCategorizedError(errors.New("3"), ErrorCategoryConfig, SeverityError))

		time.Sleep(50 * time.Millisecond)
		if alertTriggered.Load() {
			t.Error("Alert triggered for lower severity errors")
		}
	})

	t.Run("respects cooldown", func(t *testing.T) {
		cfg := DefaultErrorTrackerConfig()
		cfg.AlertCooldown = 100 * time.Millisecond
		tracker := NewErrorTracker(cfg)

		var alertCount atomic.Int32

		tracker.AddCondition(AlertCondition{
			Category:    ErrorCategoryUnknown,
			MinSeverity: SeverityError,
			Threshold:   1,
			Window:      time.Minute,
		})

		tracker.SetAlertHandler(func(cond AlertCondition, count int, recent []CategorizedError) {
			alertCount.Add(1)
		})

		// Trigger multiple alerts quickly
		for i := 0; i < 5; i++ {
			tracker.Record(NewCategorizedError(errors.New("test"), ErrorCategoryConfig, SeverityError))
		}

		time.Sleep(50 * time.Millisecond)
		if alertCount.Load() > 1 {
			t.Errorf("Alert count = %d, want 1 (cooldown should prevent more)", alertCount.Load())
		}
	})
}

func TestErrorTracker_ErrorRate(t *testing.T) {
	t.Run("calculates rate correctly", func(t *testing.T) {
		tracker := NewErrorTracker(DefaultErrorTrackerConfig())

		// Record 10 errors
		for i := 0; i < 10; i++ {
			tracker.Record(NewCategorizedError(errors.New("test"), ErrorCategoryConfig, SeverityError))
		}

		rate := tracker.ErrorRate(time.Second)
		// Rate should be approximately 10 errors per second (but >= 10 since window is tiny)
		if rate < 10 {
			t.Errorf("ErrorRate = %f, want >= 10", rate)
		}
	})

	t.Run("returns zero for zero window", func(t *testing.T) {
		tracker := NewErrorTracker(DefaultErrorTrackerConfig())
		tracker.Record(NewCategorizedError(errors.New("test"), ErrorCategoryConfig, SeverityError))

		rate := tracker.ErrorRate(0)
		if rate != 0 {
			t.Errorf("ErrorRate(0) = %f, want 0", rate)
		}
	})
}

func TestErrorTracker_ErrorRateByCategory(t *testing.T) {
	tracker := NewErrorTracker(DefaultErrorTrackerConfig())

	// Record mixed category errors
	for i := 0; i < 5; i++ {
		tracker.Record(NewCategorizedError(errors.New("lua"), ErrorCategoryLua, SeverityError))
	}
	for i := 0; i < 3; i++ {
		tracker.Record(NewCategorizedError(errors.New("config"), ErrorCategoryConfig, SeverityError))
	}

	luaRate := tracker.ErrorRateByCategory(ErrorCategoryLua, time.Second)
	configRate := tracker.ErrorRateByCategory(ErrorCategoryConfig, time.Second)

	if luaRate < 5 {
		t.Errorf("Lua rate = %f, want >= 5", luaRate)
	}
	if configRate < 3 {
		t.Errorf("Config rate = %f, want >= 3", configRate)
	}
}

func TestErrorTracker_RecentErrors(t *testing.T) {
	t.Run("returns recent errors", func(t *testing.T) {
		tracker := NewErrorTracker(DefaultErrorTrackerConfig())

		for i := 0; i < 10; i++ {
			tracker.Record(NewCategorizedError(errors.New("test"), ErrorCategoryConfig, SeverityError))
		}

		recent := tracker.RecentErrors(5)
		if len(recent) != 5 {
			t.Errorf("len(recent) = %d, want 5", len(recent))
		}
	})

	t.Run("handles empty tracker", func(t *testing.T) {
		tracker := NewErrorTracker(DefaultErrorTrackerConfig())

		recent := tracker.RecentErrors(5)
		if recent != nil {
			t.Errorf("RecentErrors on empty tracker = %v, want nil", recent)
		}
	})

	t.Run("handles zero limit", func(t *testing.T) {
		tracker := NewErrorTracker(DefaultErrorTrackerConfig())
		tracker.Record(NewCategorizedError(errors.New("test"), ErrorCategoryConfig, SeverityError))

		recent := tracker.RecentErrors(0)
		if recent != nil {
			t.Errorf("RecentErrors(0) = %v, want nil", recent)
		}
	})

	t.Run("returns all when limit exceeds count", func(t *testing.T) {
		tracker := NewErrorTracker(DefaultErrorTrackerConfig())

		for i := 0; i < 3; i++ {
			tracker.Record(NewCategorizedError(errors.New("test"), ErrorCategoryConfig, SeverityError))
		}

		recent := tracker.RecentErrors(10)
		if len(recent) != 3 {
			t.Errorf("len(recent) = %d, want 3", len(recent))
		}
	})
}

func TestErrorTracker_Clear(t *testing.T) {
	tracker := NewErrorTracker(DefaultErrorTrackerConfig())

	for i := 0; i < 5; i++ {
		tracker.Record(NewCategorizedError(errors.New("test"), ErrorCategoryConfig, SeverityError))
	}

	tracker.Clear()

	stats := tracker.Stats()
	if stats.TotalErrors != 0 {
		t.Errorf("TotalErrors after Clear = %d, want 0", stats.TotalErrors)
	}

	// Lifetime counters should NOT be reset by Clear
	var totalLifetime int64
	for _, cc := range stats.TotalByCategory {
		totalLifetime += cc.Count
	}
	if totalLifetime != 5 {
		t.Errorf("Lifetime total after Clear = %d, want 5 (lifetime counters should be preserved)", totalLifetime)
	}
}

func TestErrorTracker_ClearAll(t *testing.T) {
	tracker := NewErrorTracker(DefaultErrorTrackerConfig())

	for i := 0; i < 5; i++ {
		tracker.Record(NewCategorizedError(errors.New("test"), ErrorCategoryConfig, SeverityError))
	}

	tracker.ClearAll()

	stats := tracker.Stats()
	if stats.TotalErrors != 0 {
		t.Errorf("TotalErrors after ClearAll = %d, want 0", stats.TotalErrors)
	}

	// Lifetime counters SHOULD be reset by ClearAll
	var totalLifetime int64
	for _, cc := range stats.TotalByCategory {
		totalLifetime += cc.Count
	}
	if totalLifetime != 0 {
		t.Errorf("Lifetime total after ClearAll = %d, want 0", totalLifetime)
	}
}

func TestErrorTracker_ConcurrentAccess(t *testing.T) {
	tracker := NewErrorTracker(DefaultErrorTrackerConfig())

	var wg sync.WaitGroup
	const goroutines = 10
	const errorsPerGoroutine = 100

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < errorsPerGoroutine; j++ {
				tracker.Record(NewCategorizedError(errors.New("test"), ErrorCategoryConfig, SeverityError))
				_ = tracker.Stats()
				_ = tracker.ErrorRate(time.Second)
				_ = tracker.RecentErrors(5)
			}
		}()
	}

	wg.Wait()

	stats := tracker.Stats()
	// Some errors may be pruned, but counters should be accurate
	var totalCount int64
	for _, cc := range stats.TotalByCategory {
		totalCount += cc.Count
	}

	expected := int64(goroutines * errorsPerGoroutine)
	if totalCount != expected {
		t.Errorf("Total count = %d, want %d", totalCount, expected)
	}
}

func TestDefaultErrorTracker(t *testing.T) {
	// DefaultErrorTracker should return the same instance
	t1 := DefaultErrorTracker()
	t2 := DefaultErrorTracker()

	if t1 != t2 {
		t.Error("DefaultErrorTracker returned different instances")
	}
}

func TestErrorStats(t *testing.T) {
	tracker := NewErrorTracker(DefaultErrorTrackerConfig())

	tracker.Record(NewCategorizedError(errors.New("1"), ErrorCategoryLua, SeverityError))
	tracker.Record(NewCategorizedError(errors.New("2"), ErrorCategoryLua, SeverityCritical))
	tracker.Record(NewCategorizedError(errors.New("3"), ErrorCategoryConfig, SeverityWarning))

	stats := tracker.Stats()

	if stats.TotalErrors != 3 {
		t.Errorf("TotalErrors = %d, want 3", stats.TotalErrors)
	}

	if stats.ErrorsByCategory[ErrorCategoryLua] != 2 {
		t.Errorf("ErrorsByCategory[Lua] = %d, want 2", stats.ErrorsByCategory[ErrorCategoryLua])
	}
	if stats.ErrorsByCategory[ErrorCategoryConfig] != 1 {
		t.Errorf("ErrorsByCategory[Config] = %d, want 1", stats.ErrorsByCategory[ErrorCategoryConfig])
	}

	if stats.ErrorsBySeverity[SeverityError] != 1 {
		t.Errorf("ErrorsBySeverity[Error] = %d, want 1", stats.ErrorsBySeverity[SeverityError])
	}
	if stats.ErrorsBySeverity[SeverityCritical] != 1 {
		t.Errorf("ErrorsBySeverity[Critical] = %d, want 1", stats.ErrorsBySeverity[SeverityCritical])
	}
	if stats.ErrorsBySeverity[SeverityWarning] != 1 {
		t.Errorf("ErrorsBySeverity[Warning] = %d, want 1", stats.ErrorsBySeverity[SeverityWarning])
	}
}
