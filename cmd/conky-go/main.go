// Package main provides the entry point for the conky-go system monitor.
// This is a reimplementation of Conky in Go using Ebiten for rendering
// and Golua for Lua script compatibility.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/opd-ai/go-conky/internal/config"
	"github.com/opd-ai/go-conky/internal/profiling"
	"github.com/opd-ai/go-conky/pkg/conky"
)

// Version is the current version of conky-go.
// This default value can be overridden at build time using:
//
//	go build -ldflags "-X main.Version=x.y.z"
var Version = "0.1.0-dev"

// parsedFlags holds parsed command-line flags.
type parsedFlags struct {
	configPath string
	version    bool
	cpuProfile string
	memProfile string
	convert    string
}

// parseFlags parses command-line arguments and returns the parsed flags.
// If args is nil, uses os.Args[1:].
func parseFlags(args []string) (*parsedFlags, error) {
	fs := flag.NewFlagSet("conky-go", flag.ContinueOnError)
	fs.SetOutput(io.Discard) // Suppress default error output for testing

	configPath := fs.String("c", "", "Path to configuration file (.conkyrc or Lua config)")
	version := fs.Bool("v", false, "Print version and exit")
	cpuProfile := fs.String("cpuprofile", "", "Write CPU profile to file")
	memProfile := fs.String("memprofile", "", "Write memory profile to file")
	convert := fs.String("convert", "", "Convert legacy .conkyrc to Lua format and print to stdout")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	return &parsedFlags{
		configPath: *configPath,
		version:    *version,
		cpuProfile: *cpuProfile,
		memProfile: *memProfile,
		convert:    *convert,
	}, nil
}

func main() {
	os.Exit(run())
}

func run() int {
	return runWithArgs(os.Args[1:], os.Stdout, os.Stderr)
}

// runWithArgs is the main application logic, taking args and writers for testing.
func runWithArgs(args []string, stdout, stderr io.Writer) int {
	flags, err := parseFlags(args)
	if err != nil {
		fmt.Fprintf(stderr, "Error parsing flags: %v\n", err)
		return 1
	}

	if flags.version {
		fmt.Fprintf(stdout, "conky-go version %s\n", Version)
		return 0
	}

	// Handle --convert flag for legacy config migration
	if flags.convert != "" {
		return runConvertWithWriter(flags.convert, stdout, stderr)
	}

	// Initialize profiling if requested
	profConfig := profiling.Config{
		CPUProfilePath: flags.cpuProfile,
		MemProfilePath: flags.memProfile,
	}
	profiler := profiling.New(profConfig)

	if profConfig.ProfilingEnabled() {
		if err := profiler.Start(); err != nil {
			fmt.Fprintf(stderr, "Failed to start profiling: %v\n", err)
			return 1
		}
		defer func() {
			if err := profiler.Stop(); err != nil {
				fmt.Fprintf(stderr, "Warning: failed to stop profiling: %v\n", err)
			}
		}()
	}

	if flags.configPath == "" {
		fmt.Fprintln(stderr, "No configuration file specified. Use -c to specify a config file.")
		fmt.Fprintln(stderr, "Usage: conky-go -c <config-file>")
		return 1
	}

	// Verify config file exists and is accessible
	if _, err := os.Stat(flags.configPath); err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(stderr, "Configuration file not found: %s\n", flags.configPath)
		} else {
			fmt.Fprintf(stderr, "Error accessing configuration file %s: %v\n", flags.configPath, err)
		}
		return 1
	}

	fmt.Fprintf(stdout, "conky-go %s starting with config: %s\n", Version, flags.configPath)

	// Create and start using public API
	c, err := conky.New(flags.configPath, nil)
	if err != nil {
		fmt.Fprintf(stderr, "Error creating conky instance: %v\n", err)
		return 1
	}

	// Set up error handling
	c.SetErrorHandler(func(err error) {
		fmt.Fprintf(stderr, "Warning: %v\n", err)
	})

	// Set up event handling for lifecycle events
	c.SetEventHandler(func(e conky.Event) {
		fmt.Fprintf(stdout, "[%s] %s: %s\n", e.Timestamp.Format("15:04:05"), e.Type, e.Message)
	})

	if err := c.Start(); err != nil {
		fmt.Fprintf(stderr, "Failed to start: %v\n", err)
		return 1
	}

	// Wait for termination signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(sigCh)

	for sig := range sigCh {
		switch sig {
		case syscall.SIGHUP:
			fmt.Fprintln(stdout, "Received SIGHUP, reloading configuration...")
			if err := c.Restart(); err != nil {
				fmt.Fprintf(stderr, "Restart failed: %v\n", err)
			}
		default:
			fmt.Fprintln(stdout, "Shutting down...")
			if err := c.Stop(); err != nil {
				fmt.Fprintf(stderr, "Stop error: %v\n", err)
			}
			return 0
		}
	}

	return 0
}

// runConvertWithWriter converts a legacy .conkyrc file to Lua format using provided writers.
func runConvertWithWriter(path string, stdout, stderr io.Writer) int {
	// Verify file exists
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(stderr, "Configuration file not found: %s\n", path)
		} else {
			fmt.Fprintf(stderr, "Error accessing configuration file %s: %v\n", path, err)
		}
		return 1
	}

	// Migrate the legacy config to Lua format
	luaContent, err := config.MigrateLegacyFile(path)
	if err != nil {
		fmt.Fprintf(stderr, "Error converting configuration: %v\n", err)
		return 1
	}

	// Output to stdout (user can redirect to file)
	fmt.Fprint(stdout, string(luaContent))
	return 0
}

// runConvert converts a legacy .conkyrc file to Lua format and outputs to stdout.
// This implements the --convert CLI flag documented in docs/migration.md.
func runConvert(path string) int {
	return runConvertWithWriter(path, os.Stdout, os.Stderr)
}
