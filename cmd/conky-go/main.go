// Package main provides the entry point for the conky-go system monitor.
// This is a reimplementation of Conky in Go using Ebiten for rendering
// and Golua for Lua script compatibility.
package main

import (
	"flag"
	"fmt"
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

func main() {
	os.Exit(run())
}

func run() int {
	// Parse command-line flags
	configPath := flag.String("c", "", "Path to configuration file (.conkyrc or Lua config)")
	version := flag.Bool("v", false, "Print version and exit")
	cpuProfile := flag.String("cpuprofile", "", "Write CPU profile to file")
	memProfile := flag.String("memprofile", "", "Write memory profile to file")
	convert := flag.String("convert", "", "Convert legacy .conkyrc to Lua format and print to stdout")
	flag.Parse()

	if *version {
		fmt.Printf("conky-go version %s\n", Version)
		return 0
	}

	// Handle --convert flag for legacy config migration
	if *convert != "" {
		return runConvert(*convert)
	}

	// Initialize profiling if requested
	profConfig := profiling.Config{
		CPUProfilePath: *cpuProfile,
		MemProfilePath: *memProfile,
	}
	profiler := profiling.New(profConfig)

	if profConfig.ProfilingEnabled() {
		if err := profiler.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start profiling: %v\n", err)
			return 1
		}
		defer func() {
			if err := profiler.Stop(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to stop profiling: %v\n", err)
			}
		}()
	}

	if *configPath == "" {
		fmt.Fprintln(os.Stderr, "No configuration file specified. Use -c to specify a config file.")
		fmt.Fprintln(os.Stderr, "Usage: conky-go -c <config-file>")
		return 1
	}

	// Verify config file exists and is accessible
	if _, err := os.Stat(*configPath); err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Configuration file not found: %s\n", *configPath)
		} else {
			fmt.Fprintf(os.Stderr, "Error accessing configuration file %s: %v\n", *configPath, err)
		}
		return 1
	}

	fmt.Printf("conky-go %s starting with config: %s\n", Version, *configPath)

	// Create and start using public API
	c, err := conky.New(*configPath, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating conky instance: %v\n", err)
		return 1
	}

	// Set up error handling
	c.SetErrorHandler(func(err error) {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	})

	// Set up event handling for lifecycle events
	c.SetEventHandler(func(e conky.Event) {
		fmt.Printf("[%s] %s: %s\n", e.Timestamp.Format("15:04:05"), e.Type, e.Message)
	})

	if err := c.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start: %v\n", err)
		return 1
	}

	// Wait for termination signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(sigCh)

	for sig := range sigCh {
		switch sig {
		case syscall.SIGHUP:
			fmt.Println("Received SIGHUP, reloading configuration...")
			if err := c.Restart(); err != nil {
				fmt.Fprintf(os.Stderr, "Restart failed: %v\n", err)
			}
		default:
			fmt.Println("Shutting down...")
			if err := c.Stop(); err != nil {
				fmt.Fprintf(os.Stderr, "Stop error: %v\n", err)
			}
			return 0
		}
	}

	return 0
}

// runConvert converts a legacy .conkyrc file to Lua format and outputs to stdout.
// This implements the --convert CLI flag documented in docs/migration.md.
func runConvert(path string) int {
	// Verify file exists
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Configuration file not found: %s\n", path)
		} else {
			fmt.Fprintf(os.Stderr, "Error accessing configuration file %s: %v\n", path, err)
		}
		return 1
	}

	// Migrate the legacy config to Lua format
	luaContent, err := config.MigrateLegacyFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error converting configuration: %v\n", err)
		return 1
	}

	// Output to stdout (user can redirect to file)
	fmt.Print(string(luaContent))
	return 0
}
