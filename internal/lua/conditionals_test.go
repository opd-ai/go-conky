package lua

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/opd-ai/go-conky/internal/monitor"
)

// TestParseConditionals tests the conditional parsing functionality.
func TestParseConditionals(t *testing.T) {
	runtime, err := New(RuntimeConfig{})
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}

	provider := &mockSystemDataProvider{
		network: monitor.NetworkStats{
			Interfaces: map[string]monitor.InterfaceStats{
				"eth0": {
					IPv4Addrs: []string{"192.168.1.100"},
				},
			},
		},
		process: monitor.ProcessStats{
			TopCPU: []monitor.ProcessInfo{
				{Name: "firefox", PID: 1234},
			},
		},
		filesystem: monitor.FilesystemStats{
			Mounts: map[string]monitor.MountStats{
				"/home": {Total: 1000000000},
			},
		},
	}

	api, err := NewConkyAPI(runtime, provider)
	if err != nil {
		t.Fatalf("Failed to create API: %v", err)
	}

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "if_up true no else",
			template: "${if_up eth0}Interface up${endif}",
			expected: "Interface up",
		},
		{
			name:     "if_up false no else",
			template: "${if_up wlan0}Interface up${endif}",
			expected: "",
		},
		{
			name:     "if_up true with else",
			template: "${if_up eth0}eth0 up${else}eth0 down${endif}",
			expected: "eth0 up",
		},
		{
			name:     "if_up false with else",
			template: "${if_up wlan0}wlan0 up${else}wlan0 down${endif}",
			expected: "wlan0 down",
		},
		{
			name:     "if_running true",
			template: "${if_running firefox}Firefox running${endif}",
			expected: "Firefox running",
		},
		{
			name:     "if_running false",
			template: "${if_running chrome}Chrome running${else}Chrome not running${endif}",
			expected: "Chrome not running",
		},
		{
			name:     "if_mounted true",
			template: "${if_mounted /home}Home mounted${endif}",
			expected: "Home mounted",
		},
		{
			name:     "if_mounted false",
			template: "${if_mounted /mnt/usb}USB mounted${else}USB not mounted${endif}",
			expected: "USB not mounted",
		},
		{
			name:     "if_empty true",
			template: "${if_empty }Empty${else}Not empty${endif}",
			expected: "Empty",
		},
		{
			name:     "if_empty false",
			template: "${if_empty hello}Empty${else}Not empty${endif}",
			expected: "Not empty",
		},
		{
			name:     "if_match equal",
			template: "${if_match hello hello}Match${else}No match${endif}",
			expected: "Match",
		},
		{
			name:     "if_match not equal",
			template: "${if_match hello world}Match${else}No match${endif}",
			expected: "No match",
		},
		{
			name:     "nested conditionals",
			template: "${if_up eth0}Up ${if_running firefox}and FF${endif}${endif}",
			expected: "Up and FF",
		},
		{
			name:     "nested with else",
			template: "${if_up eth0}${if_running chrome}Chrome${else}No Chrome${endif}${endif}",
			expected: "No Chrome",
		},
		{
			name:     "conditional with variables",
			template: "${if_up eth0}CPU: ${cpu}%${endif}",
			expected: "CPU: 0%",
		},
		{
			name:     "no conditional",
			template: "Normal text ${cpu}%",
			expected: "Normal text 0%",
		},
		{
			name:     "mixed content",
			template: "Before ${if_up eth0}Middle${endif} After",
			expected: "Before Middle After",
		},
		{
			name:     "multiple conditionals",
			template: "${if_up eth0}A${endif} ${if_running firefox}B${endif}",
			expected: "A B",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := api.Parse(tc.template)
			if result != tc.expected {
				t.Errorf("Parse(%q) = %q, want %q", tc.template, result, tc.expected)
			}
		})
	}
}

// TestIfExisting tests the if_existing conditional with real files.
func TestIfExisting(t *testing.T) {
	runtime, err := New(RuntimeConfig{})
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}

	provider := &mockSystemDataProvider{}
	api, err := NewConkyAPI(runtime, provider)
	if err != nil {
		t.Fatalf("Failed to create API: %v", err)
	}

	// Create a temp file for testing
	tmpDir := t.TempDir()
	existingFile := filepath.Join(tmpDir, "existing.txt")
	if err := os.WriteFile(existingFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	nonExistingFile := filepath.Join(tmpDir, "nonexisting.txt")

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "file exists",
			template: "${if_existing " + existingFile + "}Found${else}Not found${endif}",
			expected: "Found",
		},
		{
			name:     "file does not exist",
			template: "${if_existing " + nonExistingFile + "}Found${else}Not found${endif}",
			expected: "Not found",
		},
		{
			name:     "directory exists",
			template: "${if_existing " + tmpDir + "}Dir exists${endif}",
			expected: "Dir exists",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := api.Parse(tc.template)
			if result != tc.expected {
				t.Errorf("Parse(%q) = %q, want %q", tc.template, result, tc.expected)
			}
		})
	}
}

// TestDeeplyNestedConditionals tests multiple levels of nesting.
func TestDeeplyNestedConditionals(t *testing.T) {
	runtime, err := New(RuntimeConfig{})
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}

	provider := &mockSystemDataProvider{
		network: monitor.NetworkStats{
			Interfaces: map[string]monitor.InterfaceStats{
				"eth0": {IPv4Addrs: []string{"192.168.1.1"}},
				"eth1": {IPv4Addrs: []string{"10.0.0.1"}},
			},
		},
	}

	api, err := NewConkyAPI(runtime, provider)
	if err != nil {
		t.Fatalf("Failed to create API: %v", err)
	}

	// Test 3 levels of nesting
	template := "${if_up eth0}L1${if_up eth1}L2${if_empty }L3${endif}${endif}${endif}"
	expected := "L1L2L3"

	result := api.Parse(template)
	if result != expected {
		t.Errorf("Parse(%q) = %q, want %q", template, result, expected)
	}
}

// TestEvaluateCondition tests individual condition evaluation.
func TestEvaluateCondition(t *testing.T) {
	runtime, err := New(RuntimeConfig{})
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}

	provider := &mockSystemDataProvider{
		network: monitor.NetworkStats{
			Interfaces: map[string]monitor.InterfaceStats{
				"eth0": {IPv4Addrs: []string{"192.168.1.100"}},
			},
		},
		audio: monitor.AudioStats{
			MasterMuted: true,
		},
	}

	api, err := NewConkyAPI(runtime, provider)
	if err != nil {
		t.Fatalf("Failed to create API: %v", err)
	}

	tests := []struct {
		name     string
		condExpr string
		expected bool
	}{
		{"if_up eth0", "if_up eth0", true},
		{"if_up wlan0", "if_up wlan0", false},
		{"if_running empty", "if_running", false},
		{"if_match equal", "if_match foo foo", true},
		{"if_match not equal", "if_match foo bar", false},
		{"if_match with ==", "if_match foo ==foo", true},
		{"if_match with !=", "if_match foo !=bar", true},
		{"if_empty empty", "if_empty ", true},
		{"if_empty not empty", "if_empty hello", false},
		{"if_mixer_mute muted", "if_mixer_mute", true},
		{"unknown conditional", "if_unknown arg", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := api.evaluateCondition(tc.condExpr)
			if result != tc.expected {
				t.Errorf("evaluateCondition(%q) = %v, want %v", tc.condExpr, result, tc.expected)
			}
		})
	}
}

// TestFindMatchingEndif tests the endif finding logic.
func TestFindMatchingEndif(t *testing.T) {
	runtime, err := New(RuntimeConfig{})
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}

	api, err := NewConkyAPI(runtime, nil)
	if err != nil {
		t.Fatalf("Failed to create API: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "simple endif",
			input:    "content${endif}",
			expected: 7,
		},
		{
			name:     "nested endif",
			input:    "${if_up eth0}inner${endif}outer${endif}",
			expected: 31,
		},
		{
			name:     "no endif",
			input:    "no endif here",
			expected: -1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := api.findMatchingEndif(tc.input)
			if result != tc.expected {
				t.Errorf("findMatchingEndif(%q) = %d, want %d", tc.input, result, tc.expected)
			}
		})
	}
}

// TestFindElseAtLevel tests the else finding logic.
func TestFindElseAtLevel(t *testing.T) {
	runtime, err := New(RuntimeConfig{})
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}

	api, err := NewConkyAPI(runtime, nil)
	if err != nil {
		t.Fatalf("Failed to create API: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "simple else",
			input:    "then${else}else",
			expected: 4,
		},
		{
			name:     "no else",
			input:    "no else here",
			expected: -1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := api.findElseAtLevel(tc.input)
			if result != tc.expected {
				t.Errorf("findElseAtLevel(%q) = %d, want %d", tc.input, result, tc.expected)
			}
		})
	}
}

// TestConditionalEdgeCases tests edge cases in conditional parsing.
func TestConditionalEdgeCases(t *testing.T) {
	runtime, err := New(RuntimeConfig{})
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}

	provider := &mockSystemDataProvider{
		network: monitor.NetworkStats{
			Interfaces: map[string]monitor.InterfaceStats{
				"eth0": {IPv4Addrs: []string{"192.168.1.1"}},
			},
		},
	}

	api, err := NewConkyAPI(runtime, provider)
	if err != nil {
		t.Fatalf("Failed to create API: %v", err)
	}

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "unclosed if falls back to variable",
			template: "${if_up eth0}no endif",
			// Unclosed conditional: ${if_up eth0} is resolved as variable (returns "1" when up)
			expected: "1no endif",
		},
		{
			name:     "orphan endif",
			template: "orphan ${endif}",
			expected: "orphan ${endif}",
		},
		{
			name:     "orphan else",
			template: "orphan ${else}",
			expected: "orphan ${else}",
		},
		{
			name:     "empty if",
			template: "${if_up eth0}${endif}",
			expected: "",
		},
		{
			name:     "empty else",
			template: "${if_up wlan0}then${else}${endif}",
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := api.Parse(tc.template)
			if result != tc.expected {
				t.Errorf("Parse(%q) = %q, want %q", tc.template, result, tc.expected)
			}
		})
	}
}
