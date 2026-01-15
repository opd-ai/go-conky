//go:build darwin
// +build darwin

package platform

import "strings"

// isDarwinCIError checks if the error is related to sysctl or other system calls
// that can fail in some environments with different hardware configurations
// (e.g., VMs without full hardware access, sandboxed environments).
// This function helps tests gracefully skip when hardware metrics are unavailable.
func isDarwinCIError(err error) bool {
	if err == nil {
		return false
	}

	// Normalize to lower case so matching is case-insensitive.
	errStr := strings.ToLower(err.Error())

	// Only treat errors as CI-related if they clearly reference sysctl or
	// similar kernel interfaces. This avoids accidentally classifying
	// unrelated parsing or filesystem errors as CI hardware issues.
	if !strings.Contains(errStr, "sysctl") {
		return false
	}

	// Within sysctl-related errors, look for known failure patterns that
	// typically occur on constrained environments (e.g., missing
	// hardware counters, restricted kernel interfaces, or limited memory).
	return strings.Contains(errStr, "no such file or directory") ||
		strings.Contains(errStr, "cannot allocate memory") ||
		strings.Contains(errStr, "expected integer") ||
		strings.Contains(errStr, "parsing")
}
