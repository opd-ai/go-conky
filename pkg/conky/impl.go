package conky

import (
	"context"
	"fmt"
	"image/color"
	"io/fs"
	"sync"
	"sync/atomic"
	"time"

	"github.com/opd-ai/go-conky/internal/config"
	"github.com/opd-ai/go-conky/internal/monitor"
	"github.com/opd-ai/go-conky/internal/render"
)

// defaultUpdateInterval is the default update interval for the monitor and render loop.
const defaultUpdateInterval = time.Second

// conkyImpl is the private implementation of the Conky interface.
type conkyImpl struct {
	// Configuration
	cfg           *config.Config
	opts          Options
	configSource  string
	configLoader  func() (*config.Config, error)
	fsys          fs.FS  // Embedded filesystem for Lua require() (nil for disk files)
	configContent []byte // Stored content for reader-based configs
	configFormat  string // Format for reader-based configs

	// Components
	monitor    *monitor.SystemMonitor
	gameRunner *gameRunner // For hot-reload support
	metrics    *Metrics    // Metrics collector

	// State
	running     atomic.Bool
	startTime   time.Time
	updateCount atomic.Uint64
	lastError   atomic.Value // stores error

	// Handlers
	errorHandler ErrorHandler
	eventHandler EventHandler

	// Synchronization
	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// Verify interface implementation at compile time.
var _ Conky = (*conkyImpl)(nil)

// Start begins the go-conky rendering loop.
func (c *conkyImpl) Start() error {
	c.mu.Lock()

	if c.running.Load() {
		c.mu.Unlock()
		return fmt.Errorf("conky instance already running")
	}

	// Create cancellable context
	c.ctx, c.cancel = context.WithCancel(context.Background())

	// Initialize components
	if err := c.initComponents(); err != nil {
		if c.cancel != nil {
			c.cancel()
		}
		c.mu.Unlock()
		return fmt.Errorf("failed to initialize: %w", err)
	}

	// Start monitor
	if err := c.monitor.Start(); err != nil {
		c.cleanup()
		c.mu.Unlock()
		return fmt.Errorf("failed to start monitor: %w", err)
	}

	// Set running state BEFORE starting goroutine to avoid race
	c.running.Store(true)
	c.startTime = time.Now()

	// Update metrics
	c.metrics.IncrementStarts()
	c.metrics.SetRunning(true)
	c.metrics.SetActiveMonitors(1)

	// Start update loop in goroutine (non-blocking)
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		defer c.cleanup()
		defer c.running.Store(false)
		defer c.metrics.SetRunning(false)
		defer c.metrics.SetActiveMonitors(0)

		if c.opts.Headless {
			// Headless mode: just wait for context cancellation
			<-c.ctx.Done()
		} else {
			// GUI mode: run the Ebiten rendering loop
			c.runRenderLoop()

			// Ensure context is cancelled when the render loop exits (e.g., when
			// user closes the window). This prevents a goroutine leak in the
			// monitor-stopping goroutine started in initComponents().
			if c.cancel != nil {
				c.cancel()
			}
		}

		c.emitEvent(EventStopped, "Instance stopped")
	}()

	// Release lock before emitting event to avoid deadlock
	c.mu.Unlock()

	c.emitEvent(EventStarted, "Instance started")

	return nil
}

// Stop gracefully shuts down the go-conky instance.
func (c *conkyImpl) Stop() error {
	if !c.running.Load() {
		return nil // Already stopped
	}

	// Signal stop
	c.mu.Lock()
	if c.cancel != nil {
		c.cancel()
	}
	c.mu.Unlock()

	// Wait for goroutines with timeout
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	// Use configured timeout or default
	timeout := c.opts.ShutdownTimeout
	if timeout <= 0 {
		timeout = DefaultShutdownTimeout
	}

	select {
	case <-done:
		c.metrics.IncrementStops()
		return nil
	case <-time.After(timeout):
		err := fmt.Errorf("shutdown timeout after %v: some goroutines did not stop", timeout)
		c.notifyError(err)
		return err
	}
}

// Restart performs a stop followed by a start.
func (c *conkyImpl) Restart() error {
	// Stop if running
	if err := c.Stop(); err != nil {
		wrappedErr := fmt.Errorf("stop failed: %w", err)
		c.notifyError(wrappedErr)
		return wrappedErr
	}

	// Reload configuration
	if c.configLoader != nil {
		cfg, err := c.configLoader()
		if err != nil {
			wrappedErr := fmt.Errorf("config reload failed: %w", err)
			c.notifyError(wrappedErr)
			return wrappedErr
		}
		c.mu.Lock()
		c.cfg = cfg
		c.mu.Unlock()
		c.emitEvent(EventConfigReloaded, "Configuration reloaded")
	}

	// Start again
	if err := c.Start(); err != nil {
		wrappedErr := fmt.Errorf("start failed: %w", err)
		c.notifyError(wrappedErr)
		return wrappedErr
	}

	c.metrics.IncrementRestarts()
	c.emitEvent(EventRestarted, "Instance restarted")
	return nil
}

// ReloadConfig reloads the configuration in-place without stopping.
// This provides seamless hot-reload: the rendering continues uninterrupted
// while configuration changes take effect immediately.
func (c *conkyImpl) ReloadConfig() error {
	if !c.running.Load() {
		return fmt.Errorf("conky instance not running")
	}

	if c.configLoader == nil {
		return fmt.Errorf("no config loader available")
	}

	// Load the new configuration
	newCfg, err := c.configLoader()
	if err != nil {
		wrappedErr := fmt.Errorf("config reload failed: %w", err)
		c.notifyError(wrappedErr)
		return wrappedErr
	}

	// Update the configuration atomically
	c.mu.Lock()
	oldCfg := c.cfg
	c.cfg = newCfg
	gameRunner := c.gameRunner
	c.mu.Unlock()

	// Update the render game if running in GUI mode
	if gameRunner != nil && gameRunner.game != nil {
		c.applyConfigToGame(gameRunner.game, newCfg, oldCfg)
	}

	c.metrics.IncrementConfigReloads()
	c.emitEvent(EventConfigReloaded, "Configuration reloaded in-place")
	return nil
}

// applyConfigToGame updates the game with new configuration values.
func (c *conkyImpl) applyConfigToGame(game *render.Game, newCfg, oldCfg *config.Config) {
	// Update text lines from the new template
	if len(newCfg.Text.Template) > 0 {
		textColor := newCfg.Colors.Default
		if textColor == (color.RGBA{}) {
			textColor = color.RGBA{R: 255, G: 255, B: 255, A: 255}
		}

		lines := make([]render.TextLine, 0, len(newCfg.Text.Template))
		y := defaultTextStartY
		for _, text := range newCfg.Text.Template {
			lines = append(lines, render.TextLine{
				Text:  text,
				X:     defaultTextStartX,
				Y:     y,
				Color: textColor,
			})
			y += defaultLineHeight
		}
		game.SetLines(lines)
	}

	// Update render config if dimensions or colors changed
	needsConfigUpdate := false
	currentConfig := game.Config()

	if newCfg.Window.Width > 0 && newCfg.Window.Width != currentConfig.Width {
		currentConfig.Width = newCfg.Window.Width
		needsConfigUpdate = true
	}
	if newCfg.Window.Height > 0 && newCfg.Window.Height != currentConfig.Height {
		currentConfig.Height = newCfg.Window.Height
		needsConfigUpdate = true
	}
	if newCfg.Display.UpdateInterval > 0 && newCfg.Display.UpdateInterval != currentConfig.UpdateInterval {
		currentConfig.UpdateInterval = newCfg.Display.UpdateInterval
		needsConfigUpdate = true
	}

	if needsConfigUpdate {
		game.SetConfig(currentConfig)
	}
}

// IsRunning returns true if the go-conky instance is currently running.
func (c *conkyImpl) IsRunning() bool {
	return c.running.Load()
}

// Status returns detailed status information about the instance.
func (c *conkyImpl) Status() Status {
	c.mu.RLock()
	startTime := c.startTime
	configSource := c.configSource
	c.mu.RUnlock()

	return Status{
		Running:      c.running.Load(),
		StartTime:    startTime,
		UpdateCount:  c.updateCount.Load(),
		LastError:    c.getError(),
		ConfigSource: configSource,
	}
}

// SetErrorHandler registers a callback for runtime errors.
func (c *conkyImpl) SetErrorHandler(handler ErrorHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.errorHandler = handler
}

// SetEventHandler registers a callback for lifecycle events.
func (c *conkyImpl) SetEventHandler(handler EventHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.eventHandler = handler
}

// initComponents initializes all components for operation.
func (c *conkyImpl) initComponents() error {
	// Validate config is not nil (should be guaranteed by factory functions)
	if c.cfg == nil {
		return fmt.Errorf("configuration is nil")
	}

	// Initialize metrics (use provided or default)
	if c.opts.Metrics != nil {
		c.metrics = c.opts.Metrics
	} else {
		c.metrics = DefaultMetrics()
	}

	// Determine update interval
	interval := c.cfg.Display.UpdateInterval
	if c.opts.UpdateInterval > 0 {
		interval = c.opts.UpdateInterval
	}
	if interval <= 0 {
		interval = time.Second // Default to 1 second
	}

	// Initialize system monitor
	c.monitor = monitor.NewSystemMonitor(interval)

	// Ensure the monitor is stopped when the conkyImpl context is cancelled.
	// This avoids a situation where c.ctx is cancelled but the monitor's own
	// internal context (created in NewSystemMonitor) remains active.
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		<-c.ctx.Done()

		// Best-effort stop; Stop should be safe to call multiple times.
		if c.monitor != nil {
			c.monitor.Stop()
		}
	}()
	return nil
}

// cleanup releases all resources.
func (c *conkyImpl) cleanup() {
	if c.monitor != nil {
		c.monitor.Stop()
	}
}

// getError retrieves the last error.
func (c *conkyImpl) getError() error {
	if v := c.lastError.Load(); v != nil {
		if err, ok := v.(error); ok {
			return err
		}
	}
	return nil
}

// notifyError stores an error and invokes the error handler if registered.
// This method should be called when runtime errors occur during operation.
func (c *conkyImpl) notifyError(err error) {
	// Store the error for Status() retrieval
	c.lastError.Store(err)

	// Update metrics
	if c.metrics != nil {
		c.metrics.IncrementErrors()
	}

	c.mu.RLock()
	handler := c.errorHandler
	logger := c.opts.Logger
	c.mu.RUnlock()

	if handler != nil {
		go func() {
			defer func() {
				// Recover from panics in error handler to prevent crashing
				if r := recover(); r != nil {
					if logger != nil {
						logger.Error("error handler panicked", "panic", r, "original_error", err)
					}
				}
			}()
			handler(err)
		}()
	}

	// Also emit an error event
	c.emitEvent(EventError, err.Error())
}

// emitEvent sends an event to the event handler if configured.
func (c *conkyImpl) emitEvent(eventType EventType, message string) {
	// Update metrics
	if c.metrics != nil {
		c.metrics.IncrementEventsEmitted()
	}

	c.mu.RLock()
	handler := c.eventHandler
	c.mu.RUnlock()

	if handler != nil {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					// Recover from panics in the handler to avoid crashing the embedding application.
					c.mu.RLock()
					errHandler := c.errorHandler
					c.mu.RUnlock()
					if errHandler != nil {
						if err, ok := r.(error); ok {
							errHandler(fmt.Errorf("panic in event handler: %w", err))
						} else {
							errHandler(fmt.Errorf("panic in event handler: %v", r))
						}
					}
				}
			}()

			handler(Event{
				Type:      eventType,
				Timestamp: time.Now(),
				Message:   message,
			})
		}()
	}
}

// Health returns a health check result for the Conky instance.
func (c *conkyImpl) Health() HealthCheck {
	now := time.Now()
	components := make(map[string]ComponentHealth)

	// Check running state
	running := c.running.Load()

	// Calculate uptime
	var uptime time.Duration
	c.mu.RLock()
	if running && !c.startTime.IsZero() {
		uptime = now.Sub(c.startTime)
	}
	c.mu.RUnlock()

	// Check instance component
	if running {
		components["instance"] = ComponentHealth{
			Status:      HealthOK,
			Message:     "Instance is running",
			LastUpdated: now,
		}
	} else {
		components["instance"] = ComponentHealth{
			Status:      HealthUnhealthy,
			Message:     "Instance is not running",
			LastUpdated: now,
		}
	}

	// Check monitor component
	switch {
	case c.monitor != nil && running:
		components["monitor"] = ComponentHealth{
			Status:      HealthOK,
			Message:     fmt.Sprintf("Monitor active, %d updates completed", c.updateCount.Load()),
			LastUpdated: now,
		}
	case c.monitor != nil:
		components["monitor"] = ComponentHealth{
			Status:      HealthDegraded,
			Message:     "Monitor initialized but not active",
			LastUpdated: now,
		}
	default:
		components["monitor"] = ComponentHealth{
			Status:      HealthUnhealthy,
			Message:     "Monitor not initialized",
			LastUpdated: now,
		}
	}

	// Check for errors
	lastErr := c.getError()
	if lastErr != nil {
		components["errors"] = ComponentHealth{
			Status:      HealthDegraded,
			Message:     lastErr.Error(),
			LastUpdated: now,
		}
	} else {
		components["errors"] = ComponentHealth{
			Status:      HealthOK,
			Message:     "No recent errors",
			LastUpdated: now,
		}
	}

	// Determine overall status
	overallStatus := HealthOK
	var message string

	switch {
	case !running:
		overallStatus = HealthUnhealthy
		message = "Instance is not running"
	case lastErr != nil:
		overallStatus = HealthDegraded
		message = "Running with recent errors"
	default:
		message = "All components healthy"
	}

	return HealthCheck{
		Status:     overallStatus,
		Timestamp:  now,
		Uptime:     uptime,
		Components: components,
		Message:    message,
	}
}

// Metrics returns the metrics collector for this instance.
func (c *conkyImpl) Metrics() *Metrics {
	return c.metrics
}
