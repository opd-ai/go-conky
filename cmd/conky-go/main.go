// Package main provides the entry point for the conky-go system monitor.
// This is a reimplementation of Conky in Go using Ebiten for rendering
// and Golua for Lua script compatibility.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/opd-ai/go-conky/internal/profiling"
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
	memProfile := flag.String("memprofile", "", "Write memory profile to file on exit")
	flag.Parse()

	if *version {
		fmt.Printf("conky-go version %s\n", Version)
		return 0
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
	fmt.Println("Note: This is a development build. Full functionality is not yet implemented.")

	// TODO: Phase 1 - Initialize system monitor
	// TODO: Phase 3 - Initialize Ebiten rendering engine
	// TODO: Phase 4 - Initialize Golua runtime
	// TODO: Phase 5 - Parse configuration file

	return 0
}
