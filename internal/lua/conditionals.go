// Package lua provides Golua integration for conky-go.
// This file implements conditional variable parsing for ${if_*}...${else}...${endif} blocks.
package lua

import (
	"os"
	"regexp"
	"strings"
)

// conditionalPattern matches conditional blocks: ${if_*...}...${endif}
// It captures the conditional type and arguments in group 1, and the body in group 2.
// The regex is non-greedy to handle nested conditionals properly.
var conditionalPattern = regexp.MustCompile(`\$\{(if_[^}]+)\}`)

// endifPattern matches the ${endif} marker.
var endifPattern = regexp.MustCompile(`\$\{endif\}`)

// elsePattern matches the ${else} marker.
var elsePattern = regexp.MustCompile(`\$\{else\}`)

// parseConditionals processes conditional blocks in a template string.
// It handles ${if_up}, ${if_existing}, ${if_running}, ${if_match}, ${if_empty},
// and their corresponding ${else} and ${endif} blocks.
//
// Conditional syntax:
//   - ${if_up interface}content${endif}
//   - ${if_up interface}content${else}alternative${endif}
//   - ${if_existing path}content${endif}
//   - ${if_running process}content${endif}
//   - ${if_match value pattern}content${endif}
//   - ${if_empty value}content${endif}
//
// Conditionals can be nested.
func (api *ConkyAPI) parseConditionals(template string) string {
	// Process conditionals iteratively until no more are found
	// This handles nested conditionals by processing from innermost to outermost
	result := template
	maxIterations := 100 // Prevent infinite loops

	for i := 0; i < maxIterations; i++ {
		processed := api.processOneConditional(result)
		if processed == result {
			// No changes, we're done
			break
		}
		result = processed
	}

	return result
}

// processOneConditional finds and processes one conditional block.
// It finds the first ${if_*} and its matching ${endif}, then evaluates the condition.
func (api *ConkyAPI) processOneConditional(template string) string {
	// Find first conditional
	loc := conditionalPattern.FindStringIndex(template)
	if loc == nil {
		return template
	}

	ifStart := loc[0]
	ifEnd := loc[1]

	// Extract the conditional expression (e.g., "if_up eth0")
	match := conditionalPattern.FindStringSubmatch(template[ifStart:ifEnd])
	if len(match) < 2 {
		return template
	}
	condExpr := match[1]

	// Find matching ${endif} - need to count nesting
	endifIdx := api.findMatchingEndif(template[ifEnd:])
	if endifIdx == -1 {
		// No matching endif found, return as-is
		return template
	}
	endifIdx += ifEnd // Adjust to absolute position

	// Extract the body between ${if_*} and ${endif}
	body := template[ifEnd:endifIdx]

	// Find ${else} in the body (if present, only at our nesting level)
	elseIdx := api.findElseAtLevel(body)

	var thenPart, elsePart string
	if elseIdx == -1 {
		// No else clause
		thenPart = body
		elsePart = ""
	} else {
		thenPart = body[:elseIdx]
		// Skip past ${else} (7 characters)
		elsePart = body[elseIdx+7:]
	}

	// Evaluate the condition
	condResult := api.evaluateCondition(condExpr)

	// Select the appropriate content
	var replacement string
	if condResult {
		replacement = thenPart
	} else {
		replacement = elsePart
	}

	// Find the end of ${endif} (7 characters: ${endif})
	endifEnd := endifIdx + 8

	// Rebuild the template
	return template[:ifStart] + replacement + template[endifEnd:]
}

// findMatchingEndif finds the position of the matching ${endif} for a conditional,
// accounting for nested conditionals.
func (api *ConkyAPI) findMatchingEndif(s string) int {
	depth := 1
	pos := 0

	for pos < len(s) {
		// Find next ${if_* or ${endif}
		ifLoc := conditionalPattern.FindStringIndex(s[pos:])
		endifLoc := endifPattern.FindStringIndex(s[pos:])

		// Determine which comes first
		ifIdx := -1
		endifIdx := -1
		if ifLoc != nil {
			ifIdx = pos + ifLoc[0]
		}
		if endifLoc != nil {
			endifIdx = pos + endifLoc[0]
		}

		if ifIdx == -1 && endifIdx == -1 {
			// Neither found
			return -1
		}

		if endifIdx != -1 && (ifIdx == -1 || endifIdx < ifIdx) {
			// Found ${endif} first
			depth--
			if depth == 0 {
				return endifIdx
			}
			pos = endifIdx + 8 // Skip past ${endif}
		} else if ifIdx != -1 {
			// Found another ${if_*}
			depth++
			// Find end of this ${if_*}
			endBrace := strings.Index(s[ifIdx:], "}")
			if endBrace == -1 {
				return -1
			}
			pos = ifIdx + endBrace + 1
		}
	}

	return -1
}

// findElseAtLevel finds ${else} at the current nesting level (not inside nested conditionals).
func (api *ConkyAPI) findElseAtLevel(s string) int {
	depth := 0
	pos := 0

	for pos < len(s) {
		// Find next ${if_*, ${else}, or ${endif}
		ifLoc := conditionalPattern.FindStringIndex(s[pos:])
		elseLoc := elsePattern.FindStringIndex(s[pos:])
		endifLoc := endifPattern.FindStringIndex(s[pos:])

		// Determine which comes first
		type marker struct {
			idx  int
			kind string
		}
		var markers []marker

		if ifLoc != nil {
			markers = append(markers, marker{pos + ifLoc[0], "if"})
		}
		if elseLoc != nil {
			markers = append(markers, marker{pos + elseLoc[0], "else"})
		}
		if endifLoc != nil {
			markers = append(markers, marker{pos + endifLoc[0], "endif"})
		}

		if len(markers) == 0 {
			// No more markers
			return -1
		}

		// Find the earliest marker
		earliest := markers[0]
		for _, m := range markers[1:] {
			if m.idx < earliest.idx {
				earliest = m
			}
		}

		switch earliest.kind {
		case "if":
			depth++
			// Find end of this ${if_*}
			endBrace := strings.Index(s[earliest.idx:], "}")
			if endBrace == -1 {
				return -1
			}
			pos = earliest.idx + endBrace + 1
		case "else":
			if depth == 0 {
				// Found ${else} at our level
				return earliest.idx
			}
			pos = earliest.idx + 7 // Skip past ${else}
		case "endif":
			if depth > 0 {
				depth--
			}
			pos = earliest.idx + 8 // Skip past ${endif}
		}
	}

	return -1
}

// evaluateCondition evaluates a conditional expression and returns true or false.
// Supports: if_up, if_existing, if_running, if_match, if_empty
func (api *ConkyAPI) evaluateCondition(condExpr string) bool {
	parts := strings.Fields(condExpr)
	if len(parts) == 0 {
		return false
	}

	condType := parts[0]
	args := parts[1:]

	switch condType {
	case "if_up":
		return api.evalIfUp(args)
	case "if_existing":
		return api.evalIfExisting(args)
	case "if_running":
		return api.evalIfRunning(args)
	case "if_match":
		return api.evalIfMatch(args)
	case "if_empty":
		return api.evalIfEmpty(args)
	case "if_mounted":
		return api.evalIfMounted(args)
	case "if_mpd_playing":
		return api.evalIfMPDPlaying()
	case "if_mixer_mute":
		return api.evalIfMixerMute(args)
	default:
		// Unknown conditional, treat as false
		return false
	}
}

// evalIfUp checks if a network interface is up and has an IP address.
func (api *ConkyAPI) evalIfUp(args []string) bool {
	if len(args) == 0 {
		return false
	}

	api.mu.RLock()
	provider := api.sysProvider
	api.mu.RUnlock()

	if provider == nil {
		return false
	}

	netStats := provider.Network()
	iface, ok := netStats.Interfaces[args[0]]
	if !ok {
		return false
	}

	// Interface exists and has at least one IPv4 address
	return len(iface.IPv4Addrs) > 0
}

// evalIfExisting checks if a file or path exists.
func (api *ConkyAPI) evalIfExisting(args []string) bool {
	if len(args) == 0 {
		return false
	}

	_, err := os.Stat(args[0])
	return err == nil
}

// evalIfRunning checks if a process with the given name is running.
func (api *ConkyAPI) evalIfRunning(args []string) bool {
	if len(args) == 0 {
		return false
	}

	api.mu.RLock()
	provider := api.sysProvider
	api.mu.RUnlock()

	if provider == nil {
		return false
	}

	procStats := provider.Process()
	processName := args[0]

	// Check in top CPU processes (case-sensitive to match original Conky behavior)
	for _, p := range procStats.TopCPU {
		if strings.Contains(p.Name, processName) {
			return true
		}
	}

	// Check in top memory processes (case-sensitive to match original Conky behavior)
	for _, p := range procStats.TopMem {
		if strings.Contains(p.Name, processName) {
			return true
		}
	}

	return false
}

// evalIfMatch checks if a value matches a pattern.
// Syntax: ${if_match value pattern}
// Pattern is a string comparison (not regex for simplicity).
func (api *ConkyAPI) evalIfMatch(args []string) bool {
	if len(args) < 2 {
		return false
	}

	value := args[0]
	pattern := args[1]

	// Support simple comparison operators
	if strings.HasPrefix(pattern, "==") {
		return value == strings.TrimPrefix(pattern, "==")
	}
	if strings.HasPrefix(pattern, "!=") {
		return value != strings.TrimPrefix(pattern, "!=")
	}

	// Default to equality check
	return value == pattern
}

// evalIfEmpty checks if a string is empty.
func (api *ConkyAPI) evalIfEmpty(args []string) bool {
	if len(args) == 0 {
		return true
	}

	// Join all args and check if the result is empty
	value := strings.Join(args, " ")
	return strings.TrimSpace(value) == ""
}

// evalIfMounted checks if a path is a mount point.
func (api *ConkyAPI) evalIfMounted(args []string) bool {
	if len(args) == 0 {
		return false
	}

	api.mu.RLock()
	provider := api.sysProvider
	api.mu.RUnlock()

	if provider == nil {
		return false
	}

	fsStats := provider.Filesystem()
	_, ok := fsStats.Mounts[args[0]]
	return ok
}

// evalIfMPDPlaying checks if MPD is playing.
func (api *ConkyAPI) evalIfMPDPlaying() bool {
	api.mu.RLock()
	provider := api.sysProvider
	api.mu.RUnlock()

	if provider == nil {
		return false
	}

	mpdStats := provider.MPD()
	return mpdStats.IsPlaying()
}

// evalIfMixerMute checks if the audio mixer is muted.
func (api *ConkyAPI) evalIfMixerMute(_ []string) bool {
	api.mu.RLock()
	provider := api.sysProvider
	api.mu.RUnlock()

	if provider == nil {
		return false
	}

	audioStats := provider.Audio()
	return audioStats.MasterMuted
}
