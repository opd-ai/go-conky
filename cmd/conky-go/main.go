// Package main provides the entry point for the conky-go system monitor.
// This is a reimplementation of Conky in Go using Ebiten for rendering
// and Golua for Lua script compatibility.
package main

import (
	"flag"
	"fmt"
	"os"
)

// Version is the current version of conky-go.
// This default value can be overridden at build time using:
//
//	go build -ldflags "-X main.Version=x.y.z"
var Version = "0.1.0-dev"

func main() {
	// Parse command-line flags
	configPath := flag.String("c", "", "Path to configuration file (.conkyrc or Lua config)")
	version := flag.Bool("v", false, "Print version and exit")
	flag.Parse()

	if *version {
		fmt.Printf("conky-go version %s\n", Version)
		os.Exit(0)
	}

	if *configPath == "" {
		fmt.Fprintln(os.Stderr, "No configuration file specified. Use -c to specify a config file.")
		fmt.Fprintln(os.Stderr, "Usage: conky-go -c <config-file>")
		os.Exit(1)
	}

	// Verify config file exists
	if _, err := os.Stat(*configPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Configuration file not found: %s\n", *configPath)
		os.Exit(1)
	}

	fmt.Printf("conky-go %s starting with config: %s\n", Version, *configPath)
	fmt.Println("Note: This is a development build. Full functionality is not yet implemented.")

	// TODO: Phase 1 - Initialize system monitor
	// TODO: Phase 3 - Initialize Ebiten rendering engine
	// TODO: Phase 4 - Initialize Golua runtime
	// TODO: Phase 5 - Parse configuration file
}
