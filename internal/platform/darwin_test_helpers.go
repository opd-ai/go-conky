//go:build darwin
// +build darwin

package platform

import "strings"

// isDarwinCIError checks if the error is related to sysctl or other system calls
// that can fail on CI environments with different hardware configurations
// (e.g., ARM64 GitHub Actions runners, VMs without full hardware access).
// This function helps tests gracefully skip when hardware metrics are unavailable.
func isDarwinCIError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "sysctl") ||
		strings.Contains(errStr, "no such file or directory") ||
		strings.Contains(errStr, "cannot allocate memory") ||
		strings.Contains(errStr, "expected integer") ||
		strings.Contains(errStr, "parsing")
}
