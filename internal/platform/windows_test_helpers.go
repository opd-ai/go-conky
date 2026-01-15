//go:build windows
// +build windows

package platform

import "strings"

// isWindowsCIError checks if the error is related to PDH (Performance Data Helper)
// counter initialization failures, which can occur in some environments without
// full Windows performance monitoring capabilities (e.g., sandboxed VMs).
// This function helps tests gracefully skip when hardware metrics are unavailable.
func isWindowsCIError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// Check for specific PDH error patterns to avoid matching unrelated errors.
	pdhSignatures := []string{
		"PdhOpenQuery",
		"PdhAddCounter",
		"PdhCollectQueryData",
		"PdhGetFormattedCounterValue",
	}
	for _, sig := range pdhSignatures {
		if strings.Contains(errStr, sig) {
			return true
		}
	}
	return false
}
