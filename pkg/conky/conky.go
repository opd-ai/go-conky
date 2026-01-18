package conky

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"

	"github.com/opd-ai/go-conky/internal/config"
)

// Configuration format constants for use with NewFromReader.
const (
	// FormatLegacy indicates the legacy .conkyrc text format.
	FormatLegacy = "legacy"
	// FormatLua indicates the modern Lua configuration format.
	FormatLua = "lua"
)

// Conky represents an embedded go-conky instance with full lifecycle control.
// It is safe for concurrent use from multiple goroutines.
type Conky interface {
	// Start begins the go-conky rendering loop.
	// It returns immediately after starting; the rendering runs in background goroutines.
	// Returns an error if already running or if initialization fails.
	Start() error

	// Stop gracefully shuts down the go-conky instance.
	// It waits for all goroutines to complete before returning.
	// Safe to call multiple times; subsequent calls are no-ops.
	Stop() error

	// Restart performs a stop followed by a start.
	// Configuration is reloaded from the original source.
	// Returns an error if restart fails; the instance will be in a stopped state.
	Restart() error

	// ReloadConfig reloads the configuration in-place without stopping.
	// This provides seamless hot-reload capability: the rendering continues
	// uninterrupted while configuration changes take effect.
	// Returns an error if configuration reload fails; the previous config remains active.
	ReloadConfig() error

	// IsRunning returns true if the go-conky instance is currently running.
	IsRunning() bool

	// Status returns detailed status information about the instance.
	Status() Status

	// SetErrorHandler registers a callback for runtime errors.
	// The handler is invoked asynchronously; do not block in the handler.
	// Implementations of Conky MUST recover from panics in the handler so that
	// a buggy handler cannot crash the embedding application.
	SetErrorHandler(handler ErrorHandler)

	// SetEventHandler registers a callback for lifecycle events.
	SetEventHandler(handler EventHandler)

	// Health returns a health check result for the Conky instance.
	// This can be used for monitoring, alerting, and debugging.
	Health() HealthCheck

	// Metrics returns the metrics collector for this instance.
	// Use Metrics().Snapshot() for a point-in-time copy of all metrics.
	// Use Metrics().RegisterExpvar() to expose metrics via /debug/vars.
	Metrics() *Metrics
}

// New creates a new Conky instance from a configuration file on disk.
// The configuration file can be in either legacy .conkyrc or modern Lua format.
// The instance is created but not started; call Start() to begin operation.
//
// Example:
//
//	c, err := conky.New("/home/user/.conkyrc", nil)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer c.Stop()
//	if err := c.Start(); err != nil {
//		log.Fatal(err)
//	}
func New(configPath string, opts *Options) (Conky, error) {
	if opts == nil {
		defaultOpts := DefaultOptions()
		opts = &defaultOpts
	}

	parser, err := config.NewParser()
	if err != nil {
		return nil, fmt.Errorf("parser init: %w", err)
	}
	defer parser.Close()

	cfg, err := parser.ParseFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &conkyImpl{
		cfg:          cfg,
		opts:         *opts,
		configSource: configPath,
		configLoader: func() (*config.Config, error) {
			p, err := config.NewParser()
			if err != nil {
				return nil, err
			}
			defer p.Close()
			return p.ParseFile(configPath)
		},
	}, nil
}

// NewFromFS creates a new Conky instance using configuration from an embedded filesystem.
// This enables bundling configuration files within the application binary using Go's embed package.
//
// The fsys parameter should contain the configuration files, and configPath is the path
// within the filesystem to the main configuration file.
//
// Example:
//
//	//go:embed configs/*
//	var configFS embed.FS
//
//	c, err := conky.NewFromFS(configFS, "configs/myconky.lua", nil)
//	if err != nil {
//		log.Fatal(err)
//	}
func NewFromFS(fsys fs.FS, configPath string, opts *Options) (Conky, error) {
	if opts == nil {
		defaultOpts := DefaultOptions()
		opts = &defaultOpts
	}

	parser, err := config.NewParser()
	if err != nil {
		return nil, fmt.Errorf("parser init: %w", err)
	}
	defer parser.Close()

	cfg, err := parser.ParseFromFS(fsys, configPath)
	if err != nil {
		return nil, fmt.Errorf("parse config from FS: %w", err)
	}

	// Store fsys for Lua require() support
	return &conkyImpl{
		cfg:          cfg,
		opts:         *opts,
		configSource: "embedded:" + configPath,
		fsys:         fsys,
		configLoader: func() (*config.Config, error) {
			p, err := config.NewParser()
			if err != nil {
				return nil, err
			}
			defer p.Close()
			return p.ParseFromFS(fsys, configPath)
		},
	}, nil
}

// NewFromReader creates a new Conky instance from configuration content provided as an io.Reader.
// The format parameter specifies whether the content is "legacy" or "lua" format.
// This is useful for dynamically generated configurations or network-loaded configs.
//
// Example:
//
//	config := strings.NewReader(`
//		conky.config = { update_interval = 1 }
//		conky.text = [[CPU: ${cpu}%]]
//	`)
//	c, err := conky.NewFromReader(config, conky.FormatLua, nil)
func NewFromReader(r io.Reader, format string, opts *Options) (Conky, error) {
	if opts == nil {
		defaultOpts := DefaultOptions()
		opts = &defaultOpts
	}

	if format != FormatLegacy && format != FormatLua {
		return nil, fmt.Errorf("invalid format: %s (expected '%s' or '%s')", format, FormatLua, FormatLegacy)
	}

	// Read content once (can't re-read a Reader)
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	parser, err := config.NewParser()
	if err != nil {
		return nil, fmt.Errorf("parser init: %w", err)
	}
	defer parser.Close()

	cfg, err := parser.ParseReader(bytes.NewReader(content), format)
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &conkyImpl{
		cfg:          cfg,
		opts:         *opts,
		configSource: "reader",
		configLoader: func() (*config.Config, error) {
			p, err := config.NewParser()
			if err != nil {
				return nil, err
			}
			defer p.Close()
			return p.ParseReader(bytes.NewReader(content), format)
		},
		configContent: content,
		configFormat:  format,
	}, nil
}
