package config

import (
	"strings"
	"testing"
	"time"
)

func TestNewValidator(t *testing.T) {
	v := NewValidator()
	if v == nil {
		t.Fatal("NewValidator returned nil")
	}
	if v.knownVariables == nil {
		t.Error("knownVariables map is nil")
	}
	if v.strictMode {
		t.Error("strictMode should default to false")
	}
}

func TestValidatorWithStrictMode(t *testing.T) {
	v := NewValidator().WithStrictMode(true)
	if !v.strictMode {
		t.Error("strictMode should be true after WithStrictMode(true)")
	}

	v2 := NewValidator().WithStrictMode(false)
	if v2.strictMode {
		t.Error("strictMode should be false after WithStrictMode(false)")
	}
}

func TestValidationErrorError(t *testing.T) {
	ve := ValidationError{
		Field:   "test.field",
		Message: "test message",
	}
	expected := "test.field: test message"
	if ve.Error() != expected {
		t.Errorf("expected %q, got %q", expected, ve.Error())
	}
}

func TestValidationResultIsValid(t *testing.T) {
	tests := []struct {
		name   string
		result *ValidationResult
		want   bool
	}{
		{
			name:   "empty result is valid",
			result: &ValidationResult{},
			want:   true,
		},
		{
			name: "only warnings is valid",
			result: &ValidationResult{
				Warnings: []ValidationError{{Field: "f", Message: "m"}},
			},
			want: true,
		},
		{
			name: "with errors is invalid",
			result: &ValidationResult{
				Errors: []ValidationError{{Field: "f", Message: "m"}},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidationResultError(t *testing.T) {
	tests := []struct {
		name    string
		result  *ValidationResult
		wantErr bool
	}{
		{
			name:    "no errors returns nil",
			result:  &ValidationResult{},
			wantErr: false,
		},
		{
			name: "only warnings returns nil",
			result: &ValidationResult{
				Warnings: []ValidationError{{Field: "f", Message: "m"}},
			},
			wantErr: false,
		},
		{
			name: "with errors returns error",
			result: &ValidationResult{
				Errors: []ValidationError{{Field: "f", Message: "m"}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.result.Error()
			if (err != nil) != tt.wantErr {
				t.Errorf("Error() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidationResultAddErrorAndWarning(t *testing.T) {
	result := &ValidationResult{}

	result.AddError("field1", "error message")
	result.AddWarning("field2", "warning message")

	if len(result.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(result.Errors))
	}
	if len(result.Warnings) != 1 {
		t.Errorf("expected 1 warning, got %d", len(result.Warnings))
	}

	if result.Errors[0].Field != "field1" {
		t.Errorf("expected field1, got %s", result.Errors[0].Field)
	}
	if result.Warnings[0].Field != "field2" {
		t.Errorf("expected field2, got %s", result.Warnings[0].Field)
	}
}

func TestValidationResultMerge(t *testing.T) {
	result1 := &ValidationResult{}
	result1.AddError("f1", "e1")
	result1.AddWarning("f2", "w1")

	result2 := &ValidationResult{}
	result2.AddError("f3", "e2")
	result2.AddWarning("f4", "w2")

	result1.Merge(result2)

	if len(result1.Errors) != 2 {
		t.Errorf("expected 2 errors, got %d", len(result1.Errors))
	}
	if len(result1.Warnings) != 2 {
		t.Errorf("expected 2 warnings, got %d", len(result1.Warnings))
	}

	// Test merge with nil
	result1.Merge(nil)
	if len(result1.Errors) != 2 {
		t.Error("merge with nil should not change result")
	}
}

func TestValidatorValidateWindow(t *testing.T) {
	tests := []struct {
		name         string
		window       WindowConfig
		expectErrors int
		expectWarns  int
	}{
		{
			name:         "valid window config",
			window:       DefaultWindowConfig(),
			expectErrors: 0,
			expectWarns:  0,
		},
		{
			name: "negative width",
			window: WindowConfig{
				Width:     -1,
				Height:    100,
				Type:      WindowTypeNormal,
				Alignment: AlignmentTopLeft,
			},
			expectErrors: 1,
		},
		{
			name: "negative height",
			window: WindowConfig{
				Width:     100,
				Height:    -1,
				Type:      WindowTypeNormal,
				Alignment: AlignmentTopLeft,
			},
			expectErrors: 1,
		},
		{
			name: "very large width warning",
			window: WindowConfig{
				Width:     20000,
				Height:    100,
				Type:      WindowTypeNormal,
				Alignment: AlignmentTopLeft,
			},
			expectWarns: 1,
		},
		{
			name: "unknown window type",
			window: WindowConfig{
				Width:     100,
				Height:    100,
				Type:      WindowType(99),
				Alignment: AlignmentTopLeft,
			},
			expectErrors: 1,
		},
		{
			name: "unknown alignment",
			window: WindowConfig{
				Width:     100,
				Height:    100,
				Type:      WindowTypeNormal,
				Alignment: Alignment(99),
			},
			expectErrors: 1,
		},
		{
			name: "unknown hint",
			window: WindowConfig{
				Width:     100,
				Height:    100,
				Type:      WindowTypeNormal,
				Alignment: AlignmentTopLeft,
				Hints:     []WindowHint{WindowHint(99)},
			},
			expectErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			result := &ValidationResult{}
			v.validateWindow(&tt.window, result)

			if tt.expectErrors > 0 && len(result.Errors) == 0 {
				t.Error("expected errors but got none")
			}
			if tt.expectWarns > 0 && len(result.Warnings) == 0 {
				t.Error("expected warnings but got none")
			}
		})
	}
}

func TestValidatorValidateDisplay(t *testing.T) {
	tests := []struct {
		name         string
		display      DisplayConfig
		expectErrors int
		expectWarns  int
	}{
		{
			name:         "valid display config",
			display:      DefaultDisplayConfig(),
			expectErrors: 0,
			expectWarns:  0,
		},
		{
			name: "negative update interval",
			display: DisplayConfig{
				UpdateInterval: -time.Second,
			},
			expectErrors: 1,
		},
		{
			name: "very fast update interval",
			display: DisplayConfig{
				UpdateInterval: 10 * time.Millisecond,
			},
			expectWarns: 1,
		},
		{
			name: "very slow update interval",
			display: DisplayConfig{
				UpdateInterval: 2 * time.Hour,
			},
			expectWarns: 1,
		},
		{
			name: "negative font size",
			display: DisplayConfig{
				UpdateInterval: time.Second,
				FontSize:       -1,
			},
			expectErrors: 1,
		},
		{
			name: "very large font size warning",
			display: DisplayConfig{
				UpdateInterval: time.Second,
				FontSize:       300,
			},
			expectWarns: 1,
		},
		{
			name: "font with invalid characters",
			display: DisplayConfig{
				UpdateInterval: time.Second,
				Font:           "Font;rm -rf /",
			},
			expectErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			result := &ValidationResult{}
			v.validateDisplay(&tt.display, result)

			if tt.expectErrors > 0 && len(result.Errors) == 0 {
				t.Error("expected errors but got none")
			}
			if tt.expectWarns > 0 && len(result.Warnings) == 0 {
				t.Error("expected warnings but got none")
			}
		})
	}
}

func TestValidatorValidateColors(t *testing.T) {
	tests := []struct {
		name        string
		colors      ColorConfig
		expectWarns int
	}{
		{
			name:        "valid colors",
			colors:      DefaultColorConfig(),
			expectWarns: 0,
		},
		{
			name: "transparent default color warning",
			colors: ColorConfig{
				Default: TransparentColor,
			},
			expectWarns: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			result := &ValidationResult{}
			v.validateColors(&tt.colors, result)

			if tt.expectWarns > 0 && len(result.Warnings) == 0 {
				t.Error("expected warnings but got none")
			}
		})
	}
}

func TestValidatorValidateText(t *testing.T) {
	tests := []struct {
		name         string
		template     []string
		strictMode   bool
		expectErrors int
		expectWarns  int
	}{
		{
			name:         "empty template",
			template:     []string{},
			expectErrors: 0,
			expectWarns:  0,
		},
		{
			name:         "valid variables",
			template:     []string{"CPU: ${cpu}%", "MEM: ${mem}"},
			expectErrors: 0,
			expectWarns:  0,
		},
		{
			name:         "unknown variable warning",
			template:     []string{"${unknown_var}"},
			strictMode:   false,
			expectErrors: 0,
			expectWarns:  1,
		},
		{
			name:         "unknown variable error in strict mode",
			template:     []string{"${unknown_var}"},
			strictMode:   true,
			expectErrors: 1,
			expectWarns:  0,
		},
		{
			name:         "color is not a warning",
			template:     []string{"${color grey}Test$color"},
			expectErrors: 0,
			expectWarns:  0,
		},
		{
			name:         "hr is not a warning",
			template:     []string{"${hr 2}"},
			expectErrors: 0,
			expectWarns:  0,
		},
		{
			name:         "multiple variables",
			template:     []string{"${cpu} ${mem} ${uptime}"},
			expectErrors: 0,
			expectWarns:  0,
		},
		{
			name:         "variable with arguments",
			template:     []string{"${fs_used /home}"},
			expectErrors: 0,
			expectWarns:  0,
		},
		{
			name:         "simple var not confused with braced var prefix",
			template:     []string{"$cpu and ${cpu_model}"},
			strictMode:   false,
			expectErrors: 0,
			expectWarns:  0, // Both are valid, no false positive
		},
		{
			name:         "simple unknown var not skipped due to similar braced var",
			template:     []string{"$test ${test_var}"},
			strictMode:   true,
			expectErrors: 2, // Both are unknown, both should be reported
			expectWarns:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator().WithStrictMode(tt.strictMode)
			result := &ValidationResult{}
			v.validateText(&TextConfig{Template: tt.template}, result)

			if len(result.Errors) != tt.expectErrors {
				t.Errorf("expected %d errors, got %d: %v", tt.expectErrors, len(result.Errors), result.Errors)
			}
			if len(result.Warnings) != tt.expectWarns {
				t.Errorf("expected %d warnings, got %d: %v", tt.expectWarns, len(result.Warnings), result.Warnings)
			}
		})
	}
}

func TestValidatorValidateFull(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		isValid bool
	}{
		{
			name:    "valid default config",
			config:  DefaultConfig(),
			isValid: true,
		},
		{
			name: "invalid negative width",
			config: Config{
				Window: WindowConfig{Width: -1},
			},
			isValid: false,
		},
		{
			name: "invalid negative update interval",
			config: Config{
				Display: DisplayConfig{UpdateInterval: -time.Second},
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			result := v.Validate(&tt.config)

			if result.IsValid() != tt.isValid {
				t.Errorf("IsValid() = %v, want %v; errors: %v",
					result.IsValid(), tt.isValid, result.Errors)
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  func() *Config { c := DefaultConfig(); return &c }(),
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "invalid config",
			config: &Config{
				Window: WindowConfig{Width: -1},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateConfigStrict(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "valid config without template",
			config:  func() *Config { c := DefaultConfig(); return &c }(),
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "config with unknown variable",
			config: &Config{
				Window:  DefaultWindowConfig(),
				Display: DefaultDisplayConfig(),
				Colors:  DefaultColorConfig(),
				Text:    TextConfig{Template: []string{"${unknown_variable}"}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfigStrict(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfigStrict() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestKnownConkyVariables(t *testing.T) {
	// Check some common variables are present in the package-level map
	expectedVars := []string{
		"cpu", "mem", "memmax", "memperc", "uptime",
		"downspeed", "upspeed", "fs_used", "fs_size",
		"battery_percent", "hwmon", "processes",
	}

	for _, v := range expectedVars {
		if !knownConkyVariables[v] {
			t.Errorf("expected variable %q to be in knownConkyVariables", v)
		}
	}
}

func TestValidateFontSpecification(t *testing.T) {
	tests := []struct {
		name         string
		font         string
		expectErrors int
		expectWarns  int
	}{
		{
			name:         "valid simple font",
			font:         "DejaVu Sans Mono",
			expectErrors: 0,
			expectWarns:  0,
		},
		{
			name:         "valid font with size",
			font:         "DejaVu Sans Mono:size=10",
			expectErrors: 0,
			expectWarns:  0,
		},
		{
			name:         "font with shell injection attempt",
			font:         "Font;rm -rf /",
			expectErrors: 1,
			expectWarns:  0,
		},
		{
			name:         "font starting with number",
			font:         "12pt Font",
			expectErrors: 0,
			expectWarns:  1,
		},
		{
			name:         "empty font is ok",
			font:         "",
			expectErrors: 0,
			expectWarns:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			display := DisplayConfig{
				UpdateInterval: time.Second,
				Font:           tt.font,
				FontSize:       10,
			}
			result := &ValidationResult{}
			v.validateDisplay(&display, result)

			if len(result.Errors) != tt.expectErrors {
				t.Errorf("expected %d errors, got %d: %v", tt.expectErrors, len(result.Errors), result.Errors)
			}
			if len(result.Warnings) != tt.expectWarns {
				t.Errorf("expected %d warnings, got %d: %v", tt.expectWarns, len(result.Warnings), result.Warnings)
			}
		})
	}
}

func TestVariablePatternMatching(t *testing.T) {
	tests := []struct {
		input        string
		expectedVars []string
	}{
		{
			input:        "${cpu}",
			expectedVars: []string{"cpu"},
		},
		{
			input:        "${cpu 0}",
			expectedVars: []string{"cpu"},
		},
		{
			input:        "${fs_used /home}",
			expectedVars: []string{"fs_used"},
		},
		{
			input:        "${cpu} ${mem} ${uptime}",
			expectedVars: []string{"cpu", "mem", "uptime"},
		},
		{
			input:        "${color grey}Test${color}",
			expectedVars: []string{"color", "color"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			matches := templateVariablePattern.FindAllStringSubmatch(tt.input, -1)

			if len(matches) != len(tt.expectedVars) {
				t.Errorf("expected %d matches, got %d", len(tt.expectedVars), len(matches))
				return
			}

			for i, match := range matches {
				parts := strings.Fields(match[1])
				if len(parts) == 0 {
					t.Errorf("match %d: no variable name extracted", i)
					continue
				}
				if parts[0] != tt.expectedVars[i] {
					t.Errorf("match %d: expected %q, got %q", i, tt.expectedVars[i], parts[0])
				}
			}
		})
	}
}

func TestValidatorARGBSettings(t *testing.T) {
	tests := []struct {
		name         string
		argbVisual   bool
		argbValue    int
		wantErrors   bool
		wantWarnings bool
	}{
		{
			name:         "valid settings - disabled",
			argbVisual:   false,
			argbValue:    255,
			wantErrors:   false,
			wantWarnings: false,
		},
		{
			name:         "valid settings - enabled with full opacity",
			argbVisual:   true,
			argbValue:    255,
			wantErrors:   false,
			wantWarnings: false,
		},
		{
			name:         "valid settings - enabled with transparency",
			argbVisual:   true,
			argbValue:    128,
			wantErrors:   false,
			wantWarnings: false,
		},
		{
			name:         "warning - argb value set but visual disabled",
			argbVisual:   false,
			argbValue:    128,
			wantErrors:   false,
			wantWarnings: true,
		},
		{
			name:         "valid - fully transparent with argb enabled",
			argbVisual:   true,
			argbValue:    0,
			wantErrors:   false,
			wantWarnings: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Window.ARGBVisual = tt.argbVisual
			cfg.Window.ARGBValue = tt.argbValue

			v := NewValidator()
			result := v.Validate(&cfg)

			hasErrors := len(result.Errors) > 0
			hasWarnings := len(result.Warnings) > 0

			if hasErrors != tt.wantErrors {
				t.Errorf("expected hasErrors=%v, got %v; errors: %v", tt.wantErrors, hasErrors, result.Errors)
			}
			if hasWarnings != tt.wantWarnings {
				t.Errorf("expected hasWarnings=%v, got %v; warnings: %v", tt.wantWarnings, hasWarnings, result.Warnings)
			}
		})
	}
}

func TestValidatorGradientValidation(t *testing.T) {
	tests := []struct {
		name         string
		gradient     GradientConfig
		expectErrors int
		expectWarns  int
		warnContains string
	}{
		{
			name: "valid gradient with different colors",
			gradient: GradientConfig{
				StartColor: DefaultTextColor, // White
				EndColor:   DefaultGrey,      // Grey
				Direction:  GradientDirectionVertical,
			},
			expectErrors: 0,
			expectWarns:  0,
		},
		{
			name: "both colors unset - invisible gradient",
			gradient: GradientConfig{
				StartColor: TransparentColor,
				EndColor:   TransparentColor,
				Direction:  GradientDirectionVertical,
			},
			expectErrors: 0,
			expectWarns:  1,
			warnContains: "both start_color and end_color are unset",
		},
		{
			name: "only start color unset",
			gradient: GradientConfig{
				StartColor: TransparentColor,
				EndColor:   DefaultTextColor,
				Direction:  GradientDirectionVertical,
			},
			expectErrors: 0,
			expectWarns:  1,
			warnContains: "start_color is unset",
		},
		{
			name: "only end color unset",
			gradient: GradientConfig{
				StartColor: DefaultTextColor,
				EndColor:   TransparentColor,
				Direction:  GradientDirectionVertical,
			},
			expectErrors: 0,
			expectWarns:  1,
			warnContains: "end_color is unset",
		},
		{
			name: "identical colors - effectively solid",
			gradient: GradientConfig{
				StartColor: DefaultTextColor,
				EndColor:   DefaultTextColor,
				Direction:  GradientDirectionHorizontal,
			},
			expectErrors: 0,
			expectWarns:  1,
			warnContains: "start_color and end_color are identical",
		},
		{
			name: "unknown gradient direction",
			gradient: GradientConfig{
				StartColor: DefaultTextColor,
				EndColor:   DefaultGrey,
				Direction:  GradientDirection(99),
			},
			expectErrors: 1,
			expectWarns:  0,
		},
		{
			name: "valid radial gradient",
			gradient: GradientConfig{
				StartColor: DefaultTextColor,
				EndColor:   DefaultBackgroundColour,
				Direction:  GradientDirectionRadial,
			},
			expectErrors: 0,
			expectWarns:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			result := &ValidationResult{}
			v.validateGradient(&tt.gradient, result)

			if len(result.Errors) != tt.expectErrors {
				t.Errorf("expected %d errors, got %d: %v", tt.expectErrors, len(result.Errors), result.Errors)
			}
			if len(result.Warnings) != tt.expectWarns {
				t.Errorf("expected %d warnings, got %d: %v", tt.expectWarns, len(result.Warnings), result.Warnings)
			}

			// Check warning contains expected text
			if tt.warnContains != "" && len(result.Warnings) > 0 {
				found := false
				for _, w := range result.Warnings {
					if strings.Contains(w.Message, tt.warnContains) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected warning to contain %q, got %v", tt.warnContains, result.Warnings)
				}
			}
		})
	}
}

func TestValidatorGradientModeTriggersValidation(t *testing.T) {
	// Test that gradient validation is only triggered when background mode is gradient
	tests := []struct {
		name           string
		backgroundMode BackgroundMode
		expectWarns    int
	}{
		{
			name:           "solid mode - no gradient validation",
			backgroundMode: BackgroundModeSolid,
			expectWarns:    0,
		},
		{
			name:           "transparent mode - no gradient validation",
			backgroundMode: BackgroundModeTransparent,
			expectWarns:    0,
		},
		{
			name:           "gradient mode - triggers validation with unset colors",
			backgroundMode: BackgroundModeGradient,
			expectWarns:    1, // Both colors are zero/unset in default config
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Window.BackgroundMode = tt.backgroundMode
			// Default config has zero/unset gradient colors

			v := NewValidator()
			result := v.Validate(&cfg)

			if len(result.Warnings) != tt.expectWarns {
				t.Errorf("expected %d warnings, got %d: %v", tt.expectWarns, len(result.Warnings), result.Warnings)
			}
		})
	}
}
