package lua

import (
	"testing"

	rt "github.com/arnodel/golua/runtime"
)

// TestLuaVariables tests the lua and lua_parse variables.
func TestLuaVariables(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	provider := newMockProvider()
	api, err := NewConkyAPI(runtime, provider)
	if err != nil {
		t.Fatalf("failed to create API: %v", err)
	}

	// Define test Lua functions in the runtime
	luaCode := `
function test_hello()
	return "Hello, World!"
end

function test_add(a, b)
	return tonumber(a) + tonumber(b)
end

function test_concat(...)
	local args = {...}
	return table.concat(args, "-")
end

function test_int()
	return 42
end

function test_float()
	return 3.14159
end

function test_bool_true()
	return true
end

function test_bool_false()
	return false
end

function test_nil()
	return nil
end

function test_conky_var()
	return "${cpu}"
end

function test_nested_conky()
	return "Memory: ${mem} / ${memmax}"
end
`

	_, err = runtime.ExecuteString("test_functions", luaCode)
	if err != nil {
		t.Fatalf("failed to load test Lua functions: %v", err)
	}

	tests := []struct {
		name     string
		template string
		expected string
	}{
		// Basic ${lua} tests
		{
			name:     "lua calls simple function",
			template: "${lua test_hello}",
			expected: "Hello, World!",
		},
		{
			name:     "lua calls function with arguments",
			template: "${lua test_add 10 20}",
			expected: "30",
		},
		{
			name:     "lua calls function with multiple args",
			template: "${lua test_concat a b c}",
			expected: "a-b-c",
		},
		{
			name:     "lua returns integer",
			template: "${lua test_int}",
			expected: "42",
		},
		{
			name:     "lua returns float",
			template: "${lua test_float}",
			expected: "3.14159",
		},
		{
			name:     "lua returns true",
			template: "${lua test_bool_true}",
			expected: "true",
		},
		{
			name:     "lua returns false",
			template: "${lua test_bool_false}",
			expected: "false",
		},
		{
			name:     "lua returns nil as empty",
			template: "${lua test_nil}",
			expected: "",
		},
		{
			name:     "lua with nonexistent function returns empty",
			template: "${lua nonexistent_function}",
			expected: "",
		},
		{
			name:     "lua with no function name returns empty",
			template: "${lua}",
			expected: "",
		},
		// ${lua_parse} tests - parses the result for Conky variables
		{
			name:     "lua_parse parses conky variable",
			template: "${lua_parse test_conky_var}",
			expected: "46", // CPU is 45.5, formatted as "46" (rounded)
		},
		{
			name:     "lua_parse parses nested variables",
			template: "${lua_parse test_nested_conky}",
			expected: "Memory: 8.0GiB / 16.0GiB",
		},
		// ${lua} does not parse
		{
			name:     "lua does not parse result",
			template: "${lua test_conky_var}",
			expected: "${cpu}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := api.Parse(tt.template)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestLuaValueToString tests the luaValueToString helper function.
func TestLuaValueToString(t *testing.T) {
	tests := []struct {
		name     string
		value    rt.Value
		expected string
	}{
		{
			name:     "string value",
			value:    rt.StringValue("hello"),
			expected: "hello",
		},
		{
			name:     "int value",
			value:    rt.IntValue(12345),
			expected: "12345",
		},
		{
			name:     "negative int",
			value:    rt.IntValue(-999),
			expected: "-999",
		},
		{
			name:     "float value",
			value:    rt.FloatValue(2.718),
			expected: "2.718",
		},
		{
			name:     "bool true",
			value:    rt.BoolValue(true),
			expected: "true",
		},
		{
			name:     "bool false",
			value:    rt.BoolValue(false),
			expected: "false",
		},
		{
			name:     "nil value",
			value:    rt.NilValue,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := luaValueToString(tt.value)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestLuaWithConkyParse tests that ${lua} works together with other features.
func TestLuaWithConkyParse(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	provider := newMockProvider()
	api, err := NewConkyAPI(runtime, provider)
	if err != nil {
		t.Fatalf("failed to create API: %v", err)
	}

	// Define function that formats system info
	luaCode := `
function format_temp(value)
	return value .. "°C"
end

function double(x)
	return tonumber(x) * 2
end
`

	_, err = runtime.ExecuteString("format_functions", luaCode)
	if err != nil {
		t.Fatalf("failed to load Lua functions: %v", err)
	}

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "lua used inline with text",
			template: "Temperature: ${lua format_temp 45}",
			expected: "Temperature: 45°C",
		},
		{
			name:     "multiple lua calls in template",
			template: "A: ${lua double 5}, B: ${lua double 10}",
			expected: "A: 10, B: 20",
		},
		{
			name:     "lua mixed with system variables",
			template: "CPU: ${cpu}% (doubled: ${lua double 46})",
			expected: "CPU: 46% (doubled: 92)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := api.Parse(tt.template)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
