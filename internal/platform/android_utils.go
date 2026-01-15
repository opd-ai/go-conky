//go:build android
// +build android

package platform

import (
	"os"
	"strconv"
	"strings"
)

// readUint64File reads a uint64 value from a file.
// Returns the value and true if successful, 0 and false otherwise.
func readUint64File(path string) (uint64, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}

	value, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0, false
	}

	return value, true
}

// readStringFile reads a string value from a file.
// Returns the trimmed string and true if successful, empty string and false otherwise.
func readStringFile(path string) (string, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}

	return strings.TrimSpace(string(data)), true
}

// readInt64File reads an int64 value from a file.
// Returns the value and true if successful, 0 and false otherwise.
func readInt64File(path string) (int64, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}

	value, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0, false
	}

	return value, true
}
