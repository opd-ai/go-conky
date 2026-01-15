//go:build windows
// +build windows

package platform

import "strings"

// isWindowsCIError checks if the error is related to PDH (Performance Data Helper)
// counter initialization failures, which can occur on CI environments without
// full Windows performance monitoring capabilities (e.g., GitHub Actions runners).
// This function helps tests gracefully skip when hardware metrics are unavailable.
func isWindowsCIError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "Pdh")
}
