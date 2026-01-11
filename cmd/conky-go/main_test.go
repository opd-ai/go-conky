package main

import (
	"os"
	"testing"
)

func TestVersion(t *testing.T) {
	if Version == "" {
		t.Error("Version should not be empty")
	}
}

func TestConfigFileNotFound(t *testing.T) {
	// Test that we can check for a non-existent file
	_, err := os.Stat("/nonexistent/config/file.conkyrc")
	if !os.IsNotExist(err) {
		t.Error("Expected IsNotExist error for non-existent file")
	}
}
