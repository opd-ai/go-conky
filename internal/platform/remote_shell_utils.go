package platform

import (
	"strings"
)

// shellEscape escapes a string for safe use in shell commands.
// It wraps the string in single quotes and escapes any single quotes within it.
// This prevents command injection attacks.
func shellEscape(s string) string {
	// Replace any single quotes with '\'' which ends the quote, adds an escaped quote, and starts a new quote
	escaped := strings.ReplaceAll(s, "'", "'\\''")
	return "'" + escaped + "'"
}

// validatePath performs basic validation on file paths to prevent command injection.
// Returns true if the path appears safe (contains only alphanumeric, dash, underscore, slash, dot).
// Also rejects directory traversal attempts using "..".
func validatePath(path string) bool {
	// Reject empty paths
	if path == "" {
		return false
	}

	// Reject directory traversal attempts
	if strings.Contains(path, "..") {
		return false
	}

	// Check for safe characters only
	for _, c := range path {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '-' || c == '_' ||
			c == '/' || c == '.') {
			return false
		}
	}
	return true
}
