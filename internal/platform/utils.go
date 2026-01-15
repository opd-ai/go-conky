package platform

import (
	"strconv"
)

// parseUint64 parses a string to uint64, returning 0 on error.
// This utility is used by remote SSH providers across all platforms.
func parseUint64(s string) uint64 {
	v, _ := strconv.ParseUint(s, 10, 64)
	return v
}
