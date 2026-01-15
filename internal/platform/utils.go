package platform

import (
	"os"
	"strconv"
	"strings"
)

// parseUint64 parses a string to uint64, returning 0 on error.
// This utility is used by remote SSH providers across all platforms.
func parseUint64(s string) uint64 {
	v, _ := strconv.ParseUint(s, 10, 64)
	return v
}

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

// safeMultiplyDivide multiplies two uint64 values and divides by a divisor,
// with overflow protection. Returns 0 if any intermediate calculation would overflow.
func safeMultiplyDivide(a, b, divisor uint64) uint64 {
	if divisor == 0 {
		return 0
	}

	maxUint := ^uint64(0)

	// Fast path: check for potential overflow in a * b.
	// If b == 0, the result is 0 and cannot overflow.
	if b == 0 || (b > 0 && a <= maxUint/b) {
		return (a * b) / divisor
	}

	// Slow path: a * b would overflow. Compute (a * b) / divisor as:
	// (a / divisor) * b + (a % divisor) * b / divisor
	// with additional overflow checks on the intermediate multiplications
	// and the final addition. If any of these would overflow, return 0.
	q := a / divisor
	r := a % divisor

	// Check q * b for overflow.
	if b > 0 && q > maxUint/b {
		return 0
	}
	part1 := q * b

	// Check r * b for overflow.
	if b > 0 && r > maxUint/b {
		return 0
	}
	part2 := (r * b) / divisor

	// Check part1 + part2 for overflow.
	if part1 > maxUint-part2 {
		return 0
	}

	return part1 + part2
}
