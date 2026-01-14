package platform

import (
	"testing"
)

func TestShellEscape(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple string",
			input: "hello",
			want:  "'hello'",
		},
		{
			name:  "string with spaces",
			input: "hello world",
			want:  "'hello world'",
		},
		{
			name:  "string with single quote",
			input: "it's",
			want:  "'it'\\''s'",
		},
		{
			name:  "string with multiple single quotes",
			input: "it's a 'test'",
			want:  "'it'\\''s a '\\''test'\\'''",
		},
		{
			name:  "string with semicolon",
			input: "test; rm -rf /",
			want:  "'test; rm -rf /'",
		},
		{
			name:  "string with backticks",
			input: "test`whoami`",
			want:  "'test`whoami`'",
		},
		{
			name:  "empty string",
			input: "",
			want:  "''",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shellEscape(tt.input)
			if got != tt.want {
				t.Errorf("shellEscape(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidatePath(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		valid bool
	}{
		{
			name:  "valid absolute path",
			path:  "/sys/class/hwmon/hwmon0/temp1_input",
			valid: true,
		},
		{
			name:  "valid relative path",
			path:  "temp1_input",
			valid: true,
		},
		{
			name:  "path with spaces",
			path:  "/path with spaces/file",
			valid: false,
		},
		{
			name:  "path with semicolon",
			path:  "/path;rm -rf /",
			valid: false,
		},
		{
			name:  "path with backtick",
			path:  "/path`whoami`",
			valid: false,
		},
		{
			name:  "path with dollar sign",
			path:  "/path/$var",
			valid: false,
		},
		{
			name:  "directory traversal",
			path:  "/path/../etc/passwd",
			valid: false,
		},
		{
			name:  "directory traversal relative",
			path:  "../../etc/passwd",
			valid: false,
		},
		{
			name:  "empty path",
			path:  "",
			valid: false,
		},
		{
			name:  "valid device name",
			path:  "sda1",
			valid: true,
		},
		{
			name:  "valid interface name",
			path:  "eth0",
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validatePath(tt.path)
			if got != tt.valid {
				t.Errorf("validatePath(%q) = %v, want %v", tt.path, got, tt.valid)
			}
		})
	}
}
