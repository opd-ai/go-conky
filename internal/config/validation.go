// Package config provides configuration parsing and validation for conky-go.
// This file implements comprehensive validation for configuration values
// and text template variable resolution.
package config

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ValidationError represents a configuration validation error.
// It contains the field name and a description of the issue.
type ValidationError struct {
	Field   string
	Message string
}

// Error implements the error interface.
func (ve ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", ve.Field, ve.Message)
}

// ValidationResult holds the results of a configuration validation.
type ValidationResult struct {
	// Errors contains all validation errors found.
	Errors []ValidationError
	// Warnings contains non-fatal issues (e.g., unknown variables).
	Warnings []ValidationError
}

// IsValid returns true if there are no validation errors.
func (vr *ValidationResult) IsValid() bool {
	return len(vr.Errors) == 0
}

// Error returns a combined error message if there are errors, nil otherwise.
func (vr *ValidationResult) Error() error {
	if len(vr.Errors) == 0 {
		return nil
	}

	messages := make([]string, 0, len(vr.Errors))
	for _, e := range vr.Errors {
		messages = append(messages, e.Error())
	}
	return fmt.Errorf("validation failed: %s", strings.Join(messages, "; "))
}

// AddError adds a validation error.
func (vr *ValidationResult) AddError(field, message string) {
	vr.Errors = append(vr.Errors, ValidationError{Field: field, Message: message})
}

// AddWarning adds a validation warning.
func (vr *ValidationResult) AddWarning(field, message string) {
	vr.Warnings = append(vr.Warnings, ValidationError{Field: field, Message: message})
}

// Merge combines another ValidationResult into this one.
func (vr *ValidationResult) Merge(other *ValidationResult) {
	if other == nil {
		return
	}
	vr.Errors = append(vr.Errors, other.Errors...)
	vr.Warnings = append(vr.Warnings, other.Warnings...)
}

// Validator provides comprehensive configuration validation.
type Validator struct {
	// knownVariables is the set of recognized Conky template variables.
	knownVariables map[string]bool
	// strictMode enables strict validation (unknown variables are errors).
	strictMode bool
}

// NewValidator creates a new Validator with default settings.
func NewValidator() *Validator {
	return &Validator{
		knownVariables: defaultKnownVariables(),
		strictMode:     false,
	}
}

// WithStrictMode enables strict validation where unknown variables are errors.
func (v *Validator) WithStrictMode(strict bool) *Validator {
	v.strictMode = strict
	return v
}

// Validate performs comprehensive validation of a Config.
func (v *Validator) Validate(cfg *Config) *ValidationResult {
	result := &ValidationResult{}

	v.validateWindow(&cfg.Window, result)
	v.validateDisplay(&cfg.Display, result)
	v.validateColors(&cfg.Colors, result)
	v.validateText(&cfg.Text, result)

	return result
}

// validateWindow validates WindowConfig settings.
func (v *Validator) validateWindow(wc *WindowConfig, result *ValidationResult) {
	if wc.Width < 0 {
		result.AddError("window.width", fmt.Sprintf("must be non-negative, got %d", wc.Width))
	}
	if wc.Height < 0 {
		result.AddError("window.height", fmt.Sprintf("must be non-negative, got %d", wc.Height))
	}

	// Validate window dimensions are reasonable
	const maxDimension = 10000
	if wc.Width > maxDimension {
		result.AddWarning("window.width", fmt.Sprintf("unusually large value %d", wc.Width))
	}
	if wc.Height > maxDimension {
		result.AddWarning("window.height", fmt.Sprintf("unusually large value %d", wc.Height))
	}

	// Validate window type is known
	if wc.Type > WindowTypeOverride {
		result.AddError("window.type", fmt.Sprintf("unknown window type: %d", wc.Type))
	}

	// Validate alignment is known
	if wc.Alignment > AlignmentBottomRight {
		result.AddError("window.alignment", fmt.Sprintf("unknown alignment: %d", wc.Alignment))
	}

	// Validate hints
	for i, hint := range wc.Hints {
		if hint > WindowHintSkipPager {
			result.AddError("window.hints",
				fmt.Sprintf("unknown hint at index %d: %d", i, hint))
		}
	}
}

// validateDisplay validates DisplayConfig settings.
func (v *Validator) validateDisplay(dc *DisplayConfig, result *ValidationResult) {
	if dc.UpdateInterval < 0 {
		result.AddError("display.update_interval",
			fmt.Sprintf("must be non-negative, got %v", dc.UpdateInterval))
	}

	// Warn on very fast update intervals (< 100ms)
	if dc.UpdateInterval > 0 && dc.UpdateInterval < 100*time.Millisecond {
		result.AddWarning("display.update_interval",
			fmt.Sprintf("very fast interval %v may cause high CPU usage", dc.UpdateInterval))
	}

	// Warn on very slow update intervals (> 1 hour)
	if dc.UpdateInterval > time.Hour {
		result.AddWarning("display.update_interval",
			fmt.Sprintf("very slow interval %v", dc.UpdateInterval))
	}

	// Validate font specification
	if dc.Font != "" {
		v.validateFont(dc.Font, result)
	}

	// Validate font size
	if dc.FontSize < 0 {
		result.AddError("display.font_size",
			fmt.Sprintf("must be non-negative, got %f", dc.FontSize))
	}
	if dc.FontSize > 200 {
		result.AddWarning("display.font_size",
			fmt.Sprintf("unusually large font size: %f", dc.FontSize))
	}
}

// validateFont validates a font specification string.
func (v *Validator) validateFont(font string, result *ValidationResult) {
	// Check for obviously invalid characters
	if strings.ContainsAny(font, "<>|&;$`") {
		result.AddError("display.font",
			"contains invalid characters")
		return
	}

	// Check for reasonable length
	if len(font) > 256 {
		result.AddError("display.font",
			"font specification too long")
		return
	}

	// Font names typically don't have numbers at the start
	if font != "" && font[0] >= '0' && font[0] <= '9' {
		result.AddWarning("display.font",
			"font name starts with a number")
	}
}

// validateColors validates ColorConfig settings.
func (v *Validator) validateColors(cc *ColorConfig, result *ValidationResult) {
	// Colors are validated during parsing, so we mainly check for
	// transparency issues here
	if cc.Default.A == 0 {
		result.AddWarning("colors.default",
			"fully transparent default color will be invisible")
	}
}

// validateText validates TextConfig and its template variables.
func (v *Validator) validateText(tc *TextConfig, result *ValidationResult) {
	for lineNum, line := range tc.Template {
		v.validateTemplateLine(line, lineNum+1, result)
	}
}

// variablePattern matches Conky variables: ${variable} or ${variable args}
var templateVariablePattern = regexp.MustCompile(`\$\{([^}]+)\}`)

// simpleVariablePattern matches simple Conky variables: $variable
var simpleVariablePattern = regexp.MustCompile(`\$([a-zA-Z_][a-zA-Z0-9_]*)`)

// validateTemplateLine validates a single template line for variable usage.
func (v *Validator) validateTemplateLine(line string, lineNum int, result *ValidationResult) {
	// Check ${variable} format
	matches := templateVariablePattern.FindAllStringSubmatch(line, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		inner := match[1]
		parts := strings.Fields(inner)
		if len(parts) == 0 {
			continue
		}
		varName := parts[0]
		v.checkVariable(varName, lineNum, result)
	}

	// Check $variable format (simple variables)
	simpleMatches := simpleVariablePattern.FindAllStringSubmatch(line, -1)
	for _, match := range simpleMatches {
		if len(match) < 2 {
			continue
		}
		varName := match[1]
		// Skip if this is part of a ${...} match
		if strings.Contains(line, "${"+varName) {
			continue
		}
		v.checkVariable(varName, lineNum, result)
	}
}

// checkVariable checks if a variable is known.
func (v *Validator) checkVariable(varName string, lineNum int, result *ValidationResult) {
	// Skip color and hr - they are display control commands, not data variables
	if varName == "color" || varName == "hr" || varName == "font" ||
		varName == "goto" || varName == "voffset" || varName == "alignr" ||
		varName == "alignc" || varName == "tab" || varName == "scroll" {
		return
	}

	if !v.knownVariables[varName] {
		field := fmt.Sprintf("text.template[line %d]", lineNum)
		msg := fmt.Sprintf("unknown variable: %s", varName)
		if v.strictMode {
			result.AddError(field, msg)
		} else {
			result.AddWarning(field, msg)
		}
	}
}

// defaultKnownVariables returns the set of recognized Conky template variables.
func defaultKnownVariables() map[string]bool {
	return map[string]bool{
		// CPU variables
		"cpu":       true,
		"cpu0":      true,
		"cpu1":      true,
		"cpu2":      true,
		"cpu3":      true,
		"cpu4":      true,
		"cpu5":      true,
		"cpu6":      true,
		"cpu7":      true,
		"cpubar":    true,
		"cpugauge":  true,
		"cpugraph":  true,
		"freq":      true,
		"freq_g":    true,
		"cpu_model": true,
		"loadavg":   true,

		// Memory variables
		"mem":         true,
		"memmax":      true,
		"memfree":     true,
		"memperc":     true,
		"memeasyfree": true,
		"membar":      true,
		"memgauge":    true,
		"memgraph":    true,
		"buffers":     true,
		"cached":      true,
		"swap":        true,
		"swapmax":     true,
		"swapfree":    true,
		"swapperc":    true,
		"swapbar":     true,

		// Uptime variables
		"uptime":       true,
		"uptime_short": true,

		// Network variables
		"downspeed":               true,
		"downspeedf":              true,
		"downspeedgraph":          true,
		"upspeed":                 true,
		"upspeedf":                true,
		"upspeedgraph":            true,
		"totaldown":               true,
		"totalup":                 true,
		"addr":                    true,
		"addrs":                   true,
		"wireless_essid":          true,
		"wireless_link_qual":      true,
		"wireless_link_qual_max":  true,
		"wireless_link_qual_perc": true,

		// Filesystem variables
		"fs_used":      true,
		"fs_size":      true,
		"fs_free":      true,
		"fs_used_perc": true,
		"fs_bar":       true,
		"fs_free_perc": true,
		"fs_type":      true,

		// Disk I/O variables
		"diskio":       true,
		"diskio_read":  true,
		"diskio_write": true,
		"diskiograph":  true,

		// Process variables
		"processes":         true,
		"running_processes": true,
		"threads":           true,
		"running_threads":   true,
		"top":               true,
		"top_mem":           true,
		"top_time":          true,
		"top_io":            true,

		// Battery variables
		"battery":         true,
		"battery_bar":     true,
		"battery_percent": true,
		"battery_short":   true,
		"battery_status":  true,
		"battery_time":    true,

		// Hardware monitoring
		"hwmon":    true,
		"acpitemp": true,
		"platform": true,

		// Audio variables
		"mixer":     true,
		"mixerbar":  true,
		"mixerl":    true,
		"mixerr":    true,
		"mixerlbar": true,
		"mixerrbar": true,

		// Time and date variables
		"time":        true,
		"utime":       true,
		"tztime":      true,
		"format_time": true,

		// System info
		"kernel":           true,
		"machine":          true,
		"nodename":         true,
		"nodename_short":   true,
		"sysname":          true,
		"conky_version":    true,
		"conky_build_date": true,
		"conky_build_arch": true,

		// X11 variables
		"desktop":        true,
		"desktop_name":   true,
		"desktop_number": true,

		// Display control (not data variables, but commonly used)
		"if_empty":     true,
		"if_match":     true,
		"if_existing":  true,
		"if_running":   true,
		"if_mounted":   true,
		"if_updatenr":  true,
		"else":         true,
		"endif":        true,
		"template":     true,
		"exec":         true,
		"execp":        true,
		"execi":        true,
		"execpi":       true,
		"execbar":      true,
		"execgauge":    true,
		"execgraph":    true,
		"texeci":       true,
		"lua":          true,
		"lua_parse":    true,
		"lua_bar":      true,
		"lua_gauge":    true,
		"lua_graph":    true,
		"pre_exec":     true,
		"image":        true,
		"stippled_hr":  true,
		"offset":       true,
		"shadecolor":   true,
		"outlinecolor": true,
		"to_bytes":     true,
		"eval":         true,
		"head":         true,
		"tail":         true,
		"lines":        true,
		"words":        true,
		"cat":          true,
	}
}

// ValidateConfig is a convenience function to validate a Config with default settings.
// Returns nil if the config is valid, or an error describing validation failures.
func ValidateConfig(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	validator := NewValidator()
	result := validator.Validate(cfg)
	return result.Error()
}

// ValidateConfigStrict validates a Config with strict mode enabled.
// Unknown variables in templates are treated as errors.
func ValidateConfigStrict(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	validator := NewValidator().WithStrictMode(true)
	result := validator.Validate(cfg)
	return result.Error()
}
