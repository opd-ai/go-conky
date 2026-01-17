package config

import (
	"os"
	"testing"
)

func TestExpandEnv(t *testing.T) {
	// Set up test environment variables
	os.Setenv("TEST_CONKY_VAR", "test_value")
	os.Setenv("TEST_CONKY_FONT", "DejaVu Sans Mono")
	os.Setenv("TEST_CONKY_PATH", "/home/user/.config")
	defer func() {
		os.Unsetenv("TEST_CONKY_VAR")
		os.Unsetenv("TEST_CONKY_FONT")
		os.Unsetenv("TEST_CONKY_PATH")
	}()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no variables",
			input:    "plain text without variables",
			expected: "plain text without variables",
		},
		{
			name:     "simple ${VAR} format",
			input:    "prefix ${TEST_CONKY_VAR} suffix",
			expected: "prefix test_value suffix",
		},
		{
			name:     "simple $VAR format",
			input:    "prefix $TEST_CONKY_VAR suffix",
			expected: "prefix test_value suffix",
		},
		{
			name:     "unset variable becomes empty",
			input:    "prefix ${UNSET_VAR_12345} suffix",
			expected: "prefix  suffix",
		},
		{
			name:     "unset variable with default",
			input:    "prefix ${UNSET_VAR_12345:-default_value} suffix",
			expected: "prefix default_value suffix",
		},
		{
			name:     "set variable ignores default",
			input:    "font: ${TEST_CONKY_FONT:-fallback}",
			expected: "font: DejaVu Sans Mono",
		},
		{
			name:     "empty default",
			input:    "${UNSET_VAR_12345:-}",
			expected: "",
		},
		{
			name:     "multiple variables",
			input:    "${TEST_CONKY_PATH}/config and ${TEST_CONKY_FONT}",
			expected: "/home/user/.config/config and DejaVu Sans Mono",
		},
		{
			name:     "mixed formats",
			input:    "$TEST_CONKY_VAR and ${TEST_CONKY_FONT}",
			expected: "test_value and DejaVu Sans Mono",
		},
		{
			name:     "adjacent variables",
			input:    "${TEST_CONKY_PATH}/${TEST_CONKY_VAR}",
			expected: "/home/user/.config/test_value",
		},
		{
			name:     "variable at start",
			input:    "${TEST_CONKY_VAR} at start",
			expected: "test_value at start",
		},
		{
			name:     "variable at end",
			input:    "at end ${TEST_CONKY_VAR}",
			expected: "at end test_value",
		},
		{
			name:     "preserve non-matching patterns",
			input:    "literal ${cpu} and ${mem}",
			expected: "literal  and ", // cpu and mem are not env vars
		},
		{
			name:     "default with special chars",
			input:    "${UNSET:-/path/to/file.txt}",
			expected: "/path/to/file.txt",
		},
		{
			name:     "default with colon",
			input:    "${UNSET:-value:with:colons}",
			expected: "value:with:colons",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandEnv(tt.input)
			if result != tt.expected {
				t.Errorf("ExpandEnv(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExpandEnvConfig(t *testing.T) {
	// Set up test environment variables
	os.Setenv("CONKY_TEST_FONT", "Test Font:size=12")
	os.Setenv("CONKY_TEST_TEXT", "dynamic content")
	os.Setenv("CONKY_TEST_TEMPLATE", "CPU: ${cpu}")
	defer func() {
		os.Unsetenv("CONKY_TEST_FONT")
		os.Unsetenv("CONKY_TEST_TEXT")
		os.Unsetenv("CONKY_TEST_TEMPLATE")
	}()

	cfg := &Config{
		Display: DisplayConfig{
			Font: "${CONKY_TEST_FONT}",
		},
		Text: TextConfig{
			Template: []string{
				"Line with ${CONKY_TEST_TEXT}",
				"Static line",
				"$CONKY_TEST_TEXT again",
			},
			Templates: [10]string{
				"${CONKY_TEST_TEMPLATE}", // template0
				"No variables here",      // template1
			},
		},
	}

	ExpandEnvConfig(cfg)

	// Verify font expansion
	if cfg.Display.Font != "Test Font:size=12" {
		t.Errorf("Font not expanded correctly, got %q", cfg.Display.Font)
	}

	// Verify template line expansion
	expectedLines := []string{
		"Line with dynamic content",
		"Static line",
		"dynamic content again",
	}
	for i, expected := range expectedLines {
		if cfg.Text.Template[i] != expected {
			t.Errorf("Template[%d] = %q, want %q", i, cfg.Text.Template[i], expected)
		}
	}

	// Verify template definition expansion
	if cfg.Text.Templates[0] != "CPU: ${cpu}" {
		t.Errorf("Templates[0] = %q, want %q", cfg.Text.Templates[0], "CPU: ${cpu}")
	}
	if cfg.Text.Templates[1] != "No variables here" {
		t.Errorf("Templates[1] = %q, want %q", cfg.Text.Templates[1], "No variables here")
	}
}

func TestExpandEnvConfigNil(t *testing.T) {
	// Should not panic on nil config
	ExpandEnvConfig(nil)
}

func TestExpandEnvConfigWithOptions(t *testing.T) {
	os.Setenv("OPT_TEST_VAR", "expanded")
	defer os.Unsetenv("OPT_TEST_VAR")

	t.Run("disable font expansion", func(t *testing.T) {
		cfg := &Config{
			Display: DisplayConfig{
				Font: "${OPT_TEST_VAR}",
			},
			Text: TextConfig{
				Template: []string{"${OPT_TEST_VAR}"},
			},
		}

		ExpandEnvConfigWithOptions(cfg, WithExpandFont(false))

		if cfg.Display.Font != "${OPT_TEST_VAR}" {
			t.Errorf("Font should not be expanded, got %q", cfg.Display.Font)
		}
		if cfg.Text.Template[0] != "expanded" {
			t.Errorf("Template should be expanded, got %q", cfg.Text.Template[0])
		}
	})

	t.Run("disable text expansion", func(t *testing.T) {
		cfg := &Config{
			Display: DisplayConfig{
				Font: "${OPT_TEST_VAR}",
			},
			Text: TextConfig{
				Template: []string{"${OPT_TEST_VAR}"},
			},
		}

		ExpandEnvConfigWithOptions(cfg, WithExpandText(false))

		if cfg.Display.Font != "expanded" {
			t.Errorf("Font should be expanded, got %q", cfg.Display.Font)
		}
		if cfg.Text.Template[0] != "${OPT_TEST_VAR}" {
			t.Errorf("Template should not be expanded, got %q", cfg.Text.Template[0])
		}
	})

	t.Run("disable template expansion", func(t *testing.T) {
		cfg := &Config{
			Text: TextConfig{
				Templates: [10]string{"${OPT_TEST_VAR}"},
			},
		}

		ExpandEnvConfigWithOptions(cfg, WithExpandTemplates(false))

		if cfg.Text.Templates[0] != "${OPT_TEST_VAR}" {
			t.Errorf("Templates should not be expanded, got %q", cfg.Text.Templates[0])
		}
	})

	t.Run("multiple options", func(t *testing.T) {
		cfg := &Config{
			Display: DisplayConfig{
				Font: "${OPT_TEST_VAR}",
			},
			Text: TextConfig{
				Template:  []string{"${OPT_TEST_VAR}"},
				Templates: [10]string{"${OPT_TEST_VAR}"},
			},
		}

		ExpandEnvConfigWithOptions(cfg,
			WithExpandFont(false),
			WithExpandText(false),
			WithExpandTemplates(false),
		)

		if cfg.Display.Font != "${OPT_TEST_VAR}" {
			t.Errorf("Font should not be expanded")
		}
		if cfg.Text.Template[0] != "${OPT_TEST_VAR}" {
			t.Errorf("Template should not be expanded")
		}
		if cfg.Text.Templates[0] != "${OPT_TEST_VAR}" {
			t.Errorf("Templates should not be expanded")
		}
	})
}

func TestExpandEnvConfigWithOptionsNil(t *testing.T) {
	// Should not panic on nil config
	ExpandEnvConfigWithOptions(nil)
	ExpandEnvConfigWithOptions(nil, WithExpandFont(true))
}

func TestExpandEnvEmptyString(t *testing.T) {
	result := ExpandEnv("")
	if result != "" {
		t.Errorf("ExpandEnv(%q) = %q, want empty string", "", result)
	}
}

func TestExpandEnvVariableNameValidation(t *testing.T) {
	os.Setenv("VALID_VAR", "valid")
	defer os.Unsetenv("VALID_VAR")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid variable name",
			input:    "$VALID_VAR",
			expected: "valid",
		},
		{
			name:     "variable with underscore",
			input:    "$VALID_VAR",
			expected: "valid",
		},
		{
			name:     "variable with numbers",
			input:    "${VALID_VAR}",
			expected: "valid",
		},
		{
			name:     "variable cannot start with number",
			input:    "$123VAR",
			expected: "$123VAR", // not matched as variable
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandEnv(tt.input)
			if result != tt.expected {
				t.Errorf("ExpandEnv(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
