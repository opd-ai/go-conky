// Package config provides configuration parsing for conky-go.
// This file implements environment variable expansion support for configuration values.
package config

import (
	"os"
	"regexp"
	"strings"
)

// envVarPattern matches environment variable references in configuration values.
// Supports formats:
//   - ${VAR_NAME} - standard shell-like format
//   - ${VAR_NAME:-default} - with default value if unset or empty
//   - $VAR_NAME - simple format (word characters only)
var envVarPattern = regexp.MustCompile(`\$\{([^}]+)\}|\$([a-zA-Z_][a-zA-Z0-9_]*)`)

// ExpandEnv expands environment variable references in a string.
// It supports the following formats:
//   - ${VAR_NAME} - replaced with value of VAR_NAME
//   - ${VAR_NAME:-default} - replaced with VAR_NAME's value, or "default" if unset/empty
//   - $VAR_NAME - replaced with value of VAR_NAME (simple format)
//
// Unknown or unset variables without defaults are replaced with empty string.
func ExpandEnv(s string) string {
	return envVarPattern.ReplaceAllStringFunc(s, func(match string) string {
		// Check for ${VAR} or ${VAR:-default} format
		if strings.HasPrefix(match, "${") && strings.HasSuffix(match, "}") {
			inner := match[2 : len(match)-1]

			// Check for default value syntax: VAR:-default
			if idx := strings.Index(inner, ":-"); idx >= 0 {
				varName := inner[:idx]
				defaultVal := inner[idx+2:]
				if val := os.Getenv(varName); val != "" {
					return val
				}
				return defaultVal
			}

			// Simple variable reference
			return os.Getenv(inner)
		}

		// Handle $VAR format (simple variable)
		if strings.HasPrefix(match, "$") {
			varName := match[1:]
			return os.Getenv(varName)
		}

		return match
	})
}

// ExpandEnvConfig expands environment variables in all string configuration values.
// It modifies the Config in place, expanding ${VAR} and $VAR patterns in:
//   - Font specification
//   - Text template lines
//   - Template definitions
func ExpandEnvConfig(cfg *Config) {
	if cfg == nil {
		return
	}

	// Expand font
	cfg.Display.Font = ExpandEnv(cfg.Display.Font)

	// Expand text template lines
	for i, line := range cfg.Text.Template {
		cfg.Text.Template[i] = ExpandEnv(line)
	}

	// Expand template definitions
	for i, tmpl := range cfg.Text.Templates {
		cfg.Text.Templates[i] = ExpandEnv(tmpl)
	}
}

// EnvConfigOption is a functional option for environment variable expansion.
type EnvConfigOption func(*envConfigOptions)

type envConfigOptions struct {
	expandFont      bool
	expandTemplates bool
	expandText      bool
}

// defaultEnvConfigOptions returns the default options (all expansion enabled).
func defaultEnvConfigOptions() *envConfigOptions {
	return &envConfigOptions{
		expandFont:      true,
		expandTemplates: true,
		expandText:      true,
	}
}

// WithExpandFont controls whether font specifications should be expanded.
func WithExpandFont(expand bool) EnvConfigOption {
	return func(o *envConfigOptions) {
		o.expandFont = expand
	}
}

// WithExpandTemplates controls whether template definitions should be expanded.
func WithExpandTemplates(expand bool) EnvConfigOption {
	return func(o *envConfigOptions) {
		o.expandTemplates = expand
	}
}

// WithExpandText controls whether text template lines should be expanded.
func WithExpandText(expand bool) EnvConfigOption {
	return func(o *envConfigOptions) {
		o.expandText = expand
	}
}

// ExpandEnvConfigWithOptions expands environment variables with specific options.
func ExpandEnvConfigWithOptions(cfg *Config, opts ...EnvConfigOption) {
	if cfg == nil {
		return
	}

	options := defaultEnvConfigOptions()
	for _, opt := range opts {
		opt(options)
	}

	// Expand font if enabled
	if options.expandFont {
		cfg.Display.Font = ExpandEnv(cfg.Display.Font)
	}

	// Expand text template lines if enabled
	if options.expandText {
		for i, line := range cfg.Text.Template {
			cfg.Text.Template[i] = ExpandEnv(line)
		}
	}

	// Expand template definitions if enabled
	if options.expandTemplates {
		for i, tmpl := range cfg.Text.Templates {
			cfg.Text.Templates[i] = ExpandEnv(tmpl)
		}
	}
}
