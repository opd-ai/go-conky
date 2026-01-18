package conky

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ErrorCategory represents the type of error for categorization purposes.
// Errors are categorized to enable targeted alerting and monitoring.
type ErrorCategory int

const (
	// ErrorCategoryUnknown is the default category for uncategorized errors.
	ErrorCategoryUnknown ErrorCategory = iota
	// ErrorCategoryConfig is for configuration parsing and validation errors.
	ErrorCategoryConfig
	// ErrorCategoryLua is for Lua script execution errors.
	ErrorCategoryLua
	// ErrorCategoryRender is for rendering and display errors.
	ErrorCategoryRender
	// ErrorCategoryMonitor is for system monitoring errors.
	ErrorCategoryMonitor
	// ErrorCategoryRemote is for SSH/remote monitoring connection errors.
	ErrorCategoryRemote
	// ErrorCategoryIO is for file and I/O errors.
	ErrorCategoryIO
	// ErrorCategoryNetwork is for network-related errors.
	ErrorCategoryNetwork
)

// String returns a human-readable name for the error category.
func (c ErrorCategory) String() string {
	switch c {
	case ErrorCategoryConfig:
		return "config"
	case ErrorCategoryLua:
		return "lua"
	case ErrorCategoryRender:
		return "render"
	case ErrorCategoryMonitor:
		return "monitor"
	case ErrorCategoryRemote:
		return "remote"
	case ErrorCategoryIO:
		return "io"
	case ErrorCategoryNetwork:
		return "network"
	default:
		return "unknown"
	}
}

// ErrorSeverity indicates the severity level of an error.
type ErrorSeverity int

const (
	// SeverityInfo is for informational messages that don't require action.
	SeverityInfo ErrorSeverity = iota
	// SeverityWarning is for non-critical issues that should be investigated.
	SeverityWarning
	// SeverityError is for errors that affect functionality but allow continued operation.
	SeverityError
	// SeverityCritical is for errors that require immediate attention.
	SeverityCritical
)

// String returns a human-readable name for the severity level.
func (s ErrorSeverity) String() string {
	switch s {
	case SeverityInfo:
		return "info"
	case SeverityWarning:
		return "warning"
	case SeverityError:
		return "error"
	case SeverityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// CategorizedError wraps an error with additional metadata for tracking and alerting.
type CategorizedError struct {
	// Err is the underlying error.
	Err error
	// Category classifies the type of error.
	Category ErrorCategory
	// Severity indicates the urgency level.
	Severity ErrorSeverity
	// Timestamp is when the error occurred.
	Timestamp time.Time
	// Context provides additional key-value metadata.
	Context map[string]string
}

// Error implements the error interface.
func (e *CategorizedError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("[%s/%s] (no error)", e.Severity, e.Category)
	}
	return fmt.Sprintf("[%s/%s] %s", e.Severity, e.Category, e.Err.Error())
}

// Unwrap returns the underlying error for errors.Is/As support.
func (e *CategorizedError) Unwrap() error {
	return e.Err
}

// NewCategorizedError creates a new CategorizedError with the given parameters.
func NewCategorizedError(err error, category ErrorCategory, severity ErrorSeverity) *CategorizedError {
	return &CategorizedError{
		Err:       err,
		Category:  category,
		Severity:  severity,
		Timestamp: time.Now(),
		Context:   make(map[string]string),
	}
}

// WithContext adds a key-value pair to the error context and returns the error.
func (e *CategorizedError) WithContext(key, value string) *CategorizedError {
	if e.Context == nil {
		e.Context = make(map[string]string)
	}
	e.Context[key] = value
	return e
}

// AlertCondition defines when an alert should be triggered.
type AlertCondition struct {
	// Category filters alerts to a specific error category.
	// Use ErrorCategoryUnknown to match all categories.
	Category ErrorCategory
	// MinSeverity is the minimum severity level to trigger the alert.
	MinSeverity ErrorSeverity
	// Threshold is the number of errors within the window to trigger.
	Threshold int
	// Window is the time window for counting errors.
	Window time.Duration
}

// AlertHandler is called when an alert condition is met.
// The handler receives the condition that triggered and the current error count.
// Implementations must not block; use goroutines for slow operations.
type AlertHandler func(condition AlertCondition, errorCount int, recentErrors []CategorizedError)

// ErrorTracker aggregates and monitors errors for alerting.
// It maintains a sliding window of recent errors and checks alert conditions.
// Thread-safe for concurrent use.
type ErrorTracker struct {
	mu            sync.RWMutex
	errors        []CategorizedError
	maxErrors     int           // Maximum number of errors to retain
	retentionTime time.Duration // How long to retain errors
	conditions    []AlertCondition
	handlers      []AlertHandler
	lastAlert     map[int]time.Time // Alert cooldown tracking (index = condition index)
	alertCooldown time.Duration     // Minimum time between alerts for same condition

	// Counters per category (atomic for fast access)
	categoryCounters [8]atomic.Int64 // One per ErrorCategory (0-7)
}

// ErrorTrackerConfig configures an ErrorTracker.
type ErrorTrackerConfig struct {
	// MaxErrors is the maximum number of errors to retain (default: 1000).
	MaxErrors int
	// RetentionTime is how long to retain errors (default: 1 hour).
	RetentionTime time.Duration
	// AlertCooldown is the minimum time between repeated alerts (default: 5 minutes).
	AlertCooldown time.Duration
}

// DefaultErrorTrackerConfig returns a configuration with sensible defaults.
func DefaultErrorTrackerConfig() ErrorTrackerConfig {
	return ErrorTrackerConfig{
		MaxErrors:     1000,
		RetentionTime: time.Hour,
		AlertCooldown: 5 * time.Minute,
	}
}

// NewErrorTracker creates a new ErrorTracker with the given configuration.
func NewErrorTracker(cfg ErrorTrackerConfig) *ErrorTracker {
	if cfg.MaxErrors <= 0 {
		cfg.MaxErrors = 1000
	}
	if cfg.RetentionTime <= 0 {
		cfg.RetentionTime = time.Hour
	}
	if cfg.AlertCooldown <= 0 {
		cfg.AlertCooldown = 5 * time.Minute
	}

	return &ErrorTracker{
		errors:        make([]CategorizedError, 0, cfg.MaxErrors),
		maxErrors:     cfg.MaxErrors,
		retentionTime: cfg.RetentionTime,
		conditions:    make([]AlertCondition, 0),
		handlers:      make([]AlertHandler, 0),
		lastAlert:     make(map[int]time.Time),
		alertCooldown: cfg.AlertCooldown,
	}
}

// AddCondition registers an alert condition to monitor.
func (t *ErrorTracker) AddCondition(cond AlertCondition) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.conditions = append(t.conditions, cond)
}

// SetAlertHandler registers a handler for all alert conditions.
// Multiple handlers can be registered by calling this method multiple times.
func (t *ErrorTracker) SetAlertHandler(handler AlertHandler) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.handlers = append(t.handlers, handler)
}

// Record adds an error to the tracker and checks alert conditions.
func (t *ErrorTracker) Record(err *CategorizedError) {
	if err == nil {
		return
	}

	// Update category counter
	if int(err.Category) < len(t.categoryCounters) {
		t.categoryCounters[err.Category].Add(1)
	}

	t.mu.Lock()
	// Add error to the list
	t.errors = append(t.errors, *err)

	// Prune old errors if over capacity
	if len(t.errors) > t.maxErrors {
		t.errors = t.errors[len(t.errors)-t.maxErrors:]
	}

	// Prune expired errors
	t.pruneExpired()

	// Copy data needed for condition checking
	conditions := make([]AlertCondition, len(t.conditions))
	copy(conditions, t.conditions)
	handlers := make([]AlertHandler, len(t.handlers))
	copy(handlers, t.handlers)
	t.mu.Unlock()

	// Check alert conditions (outside lock to avoid blocking)
	for i, cond := range conditions {
		t.checkCondition(i, cond, handlers)
	}
}

// pruneExpired removes errors older than the retention time.
// Must be called with mu held.
func (t *ErrorTracker) pruneExpired() {
	if len(t.errors) == 0 {
		return
	}

	cutoff := time.Now().Add(-t.retentionTime)
	start := 0
	for i, err := range t.errors {
		if err.Timestamp.After(cutoff) {
			start = i
			break
		}
		start = i + 1
	}
	if start > 0 {
		t.errors = t.errors[start:]
	}
}

// checkCondition evaluates a single alert condition and triggers handlers if met.
func (t *ErrorTracker) checkCondition(index int, cond AlertCondition, handlers []AlertHandler) {
	t.mu.RLock()
	// Check cooldown
	lastTime, exists := t.lastAlert[index]
	if exists && time.Since(lastTime) < t.alertCooldown {
		t.mu.RUnlock()
		return
	}

	// Count matching errors within window
	cutoff := time.Now().Add(-cond.Window)
	var count int
	var matching []CategorizedError

	for _, err := range t.errors {
		if err.Timestamp.Before(cutoff) {
			continue
		}
		if cond.Category != ErrorCategoryUnknown && err.Category != cond.Category {
			continue
		}
		if err.Severity < cond.MinSeverity {
			continue
		}
		count++
		if len(matching) < 10 { // Keep up to 10 recent examples
			matching = append(matching, err)
		}
	}
	t.mu.RUnlock()

	// Check if threshold is met
	if count >= cond.Threshold {
		// Update last alert time
		t.mu.Lock()
		t.lastAlert[index] = time.Now()
		t.mu.Unlock()

		// Call handlers (outside lock)
		for _, handler := range handlers {
			// Call in goroutine to avoid blocking
			go func(h AlertHandler) {
				defer func() {
					recover() // Prevent panic in handler from crashing
				}()
				h(cond, count, matching)
			}(handler)
		}
	}
}

// ErrorRate calculates the error rate (errors per second) within the given window.
func (t *ErrorTracker) ErrorRate(window time.Duration) float64 {
	if window <= 0 {
		return 0
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	cutoff := time.Now().Add(-window)
	count := 0
	for _, err := range t.errors {
		if err.Timestamp.After(cutoff) {
			count++
		}
	}

	return float64(count) / window.Seconds()
}

// ErrorRateByCategory calculates the error rate for a specific category.
func (t *ErrorTracker) ErrorRateByCategory(category ErrorCategory, window time.Duration) float64 {
	if window <= 0 {
		return 0
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	cutoff := time.Now().Add(-window)
	count := 0
	for _, err := range t.errors {
		if err.Timestamp.After(cutoff) && err.Category == category {
			count++
		}
	}

	return float64(count) / window.Seconds()
}

// Stats returns a snapshot of error statistics.
func (t *ErrorTracker) Stats() ErrorStats {
	t.mu.RLock()
	defer t.mu.RUnlock()

	stats := ErrorStats{
		TotalErrors:     len(t.errors),
		ErrorsByCategory: make(map[ErrorCategory]int),
		ErrorsBySeverity: make(map[ErrorSeverity]int),
	}

	for _, err := range t.errors {
		stats.ErrorsByCategory[err.Category]++
		stats.ErrorsBySeverity[err.Severity]++
	}

	// Add total counts from atomic counters
	for i := 0; i < len(t.categoryCounters); i++ {
		stats.TotalByCategory = append(stats.TotalByCategory, CategoryCount{
			Category: ErrorCategory(i),
			Count:    t.categoryCounters[i].Load(),
		})
	}

	return stats
}

// RecentErrors returns the most recent errors, up to the specified limit.
func (t *ErrorTracker) RecentErrors(limit int) []CategorizedError {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if limit <= 0 || len(t.errors) == 0 {
		return nil
	}

	start := len(t.errors) - limit
	if start < 0 {
		start = 0
	}

	result := make([]CategorizedError, len(t.errors)-start)
	copy(result, t.errors[start:])
	return result
}

// Clear removes all tracked errors.
func (t *ErrorTracker) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.errors = t.errors[:0]
	t.lastAlert = make(map[int]time.Time)
}

// ErrorStats provides a summary of error statistics.
type ErrorStats struct {
	// TotalErrors is the number of errors currently retained.
	TotalErrors int
	// ErrorsByCategory counts errors by category in the current retention window.
	ErrorsByCategory map[ErrorCategory]int
	// ErrorsBySeverity counts errors by severity in the current retention window.
	ErrorsBySeverity map[ErrorSeverity]int
	// TotalByCategory contains lifetime totals per category (from atomic counters).
	TotalByCategory []CategoryCount
}

// CategoryCount pairs a category with its count.
type CategoryCount struct {
	Category ErrorCategory
	Count    int64
}

// defaultErrorTracker is a global error tracker for convenience.
var defaultErrorTracker *ErrorTracker
var defaultErrorTrackerOnce sync.Once

// DefaultErrorTracker returns the global default ErrorTracker instance.
// The tracker is lazily initialized with default configuration.
func DefaultErrorTracker() *ErrorTracker {
	defaultErrorTrackerOnce.Do(func() {
		defaultErrorTracker = NewErrorTracker(DefaultErrorTrackerConfig())
	})
	return defaultErrorTracker
}
