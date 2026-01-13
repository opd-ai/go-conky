package main

import (
	"testing"
)

func TestVersion(t *testing.T) {
	if Version == "" {
		t.Error("Version should not be empty")
	}
}

func TestVersionFormat(t *testing.T) {
	// Version should contain at least one character
	if len(Version) < 1 {
		t.Error("Version should have at least one character")
	}
}

