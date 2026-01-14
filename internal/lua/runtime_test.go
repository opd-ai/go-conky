package lua

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	rt "github.com/arnodel/golua/runtime"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.CPULimit != 10_000_000 {
		t.Errorf("expected CPULimit 10000000, got %d", config.CPULimit)
	}

	if config.MemoryLimit != 50*1024*1024 {
		t.Errorf("expected MemoryLimit %d, got %d", 50*1024*1024, config.MemoryLimit)
	}

	if config.Stdout != os.Stdout {
		t.Error("expected Stdout to be os.Stdout")
	}
}

func TestNew(t *testing.T) {
	config := DefaultConfig()
	runtime, err := New(config)
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	if runtime.runtime == nil {
		t.Error("expected runtime to be initialized")
	}

	if runtime.output == nil {
		t.Error("expected output buffer to be initialized")
	}
}

func TestNewWithCustomStdout(t *testing.T) {
	buf := &bytes.Buffer{}
	config := RuntimeConfig{
		CPULimit:    1_000_000,
		MemoryLimit: 10 * 1024 * 1024,
		Stdout:      buf,
	}

	runtime, err := New(config)
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	// Execute print statement
	_, err = runtime.ExecuteString("test", `print("hello from lua")`)
	if err != nil {
		t.Fatalf("failed to execute Lua code: %v", err)
	}

	// Check that output was captured
	output := buf.String()
	if output != "hello from lua\n" {
		t.Errorf("expected 'hello from lua\\n', got %q", output)
	}
}

func TestLoadString(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	tests := []struct {
		name    string
		code    string
		wantErr bool
	}{
		{
			name:    "valid code",
			code:    "return 42",
			wantErr: false,
		},
		{
			name:    "valid function",
			code:    "function test() return 1 end",
			wantErr: false,
		},
		{
			name:    "syntax error",
			code:    "invalid lua syntax {{}}",
			wantErr: true,
		},
		{
			name:    "empty code",
			code:    "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			closure, err := runtime.LoadString(tt.name, tt.code)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && closure == nil {
				t.Error("expected closure to be non-nil")
			}
		})
	}
}

func TestExecuteString(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	tests := []struct {
		name       string
		code       string
		wantResult interface{}
		wantErr    bool
	}{
		{
			name:       "return integer",
			code:       "return 42",
			wantResult: int64(42),
			wantErr:    false,
		},
		{
			name:       "return string",
			code:       `return "hello"`,
			wantResult: "hello",
			wantErr:    false,
		},
		{
			name:       "return calculation",
			code:       "return 10 + 20 * 2",
			wantResult: int64(50),
			wantErr:    false,
		},
		{
			name:       "return nil",
			code:       "return nil",
			wantResult: nil,
			wantErr:    false,
		},
		{
			name:       "syntax error",
			code:       "return {{invalid",
			wantResult: nil,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := runtime.ExecuteString(tt.name, tt.code)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			switch expected := tt.wantResult.(type) {
			case int64:
				got, ok := rt.ToInt(result)
				if !ok || got != expected {
					t.Errorf("expected %d, got %v", expected, result)
				}
			case string:
				if result.AsString() != expected {
					t.Errorf("expected %q, got %q", expected, result.AsString())
				}
			case nil:
				if result != rt.NilValue {
					t.Errorf("expected nil, got %v", result)
				}
			}
		})
	}
}

func TestLoadFile(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	// Create a temporary Lua file
	tmpDir := t.TempDir()
	luaFile := filepath.Join(tmpDir, "test.lua")
	if err := os.WriteFile(luaFile, []byte("return 123"), 0o644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	// Test loading the file
	closure, err := runtime.LoadFile(luaFile)
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	if closure == nil {
		t.Error("expected closure to be non-nil")
	}

	// Test loading non-existent file
	_, err = runtime.LoadFile("/nonexistent/file.lua")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestExecuteFile(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	// Create a temporary Lua file
	tmpDir := t.TempDir()
	luaFile := filepath.Join(tmpDir, "test.lua")
	if err := os.WriteFile(luaFile, []byte("return 456"), 0o644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	// Test executing the file
	result, err := runtime.ExecuteFile(luaFile)
	if err != nil {
		t.Fatalf("ExecuteFile() error = %v", err)
	}

	got, ok := rt.ToInt(result)
	if !ok || got != 456 {
		t.Errorf("expected 456, got %v", result)
	}
}

func TestSetAndGetGlobal(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	// Set a global value
	runtime.SetGlobal("myVar", rt.IntValue(999))

	// Get the global value
	value := runtime.GetGlobal("myVar")
	got, ok := rt.ToInt(value)
	if !ok || got != 999 {
		t.Errorf("expected 999, got %v", value)
	}

	// Get non-existent global
	value = runtime.GetGlobal("nonexistent")
	if value != rt.NilValue {
		t.Errorf("expected nil for non-existent global, got %v", value)
	}
}

func TestSetGoFunction(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	// Register a Go function that adds two numbers
	addFunc := func(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
		a, _ := c.IntArg(0)
		b, _ := c.IntArg(1)
		return c.PushingNext1(t.Runtime, rt.IntValue(a+b)), nil
	}
	runtime.SetGoFunction("add", addFunc, 2, false)

	// Call the function from Lua
	result, err := runtime.ExecuteString("test", "return add(10, 20)")
	if err != nil {
		t.Fatalf("failed to execute Lua code: %v", err)
	}

	got, ok := rt.ToInt(result)
	if !ok || got != 30 {
		t.Errorf("expected 30, got %v", result)
	}
}

func TestCallFunction(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	// Define a Lua function
	_, err = runtime.ExecuteString("setup", `
		function multiply(a, b)
			return a * b
		end
	`)
	if err != nil {
		t.Fatalf("failed to define function: %v", err)
	}

	// Call the function
	result, err := runtime.CallFunction("multiply", rt.IntValue(5), rt.IntValue(7))
	if err != nil {
		t.Fatalf("CallFunction() error = %v", err)
	}

	got, ok := rt.ToInt(result)
	if !ok || got != 35 {
		t.Errorf("expected 35, got %v", result)
	}

	// Call non-existent function
	_, err = runtime.CallFunction("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent function")
	}
}

func TestOutput(t *testing.T) {
	buf := &bytes.Buffer{}
	config := RuntimeConfig{
		CPULimit:    1_000_000,
		MemoryLimit: 10 * 1024 * 1024,
		Stdout:      buf,
	}

	runtime, err := New(config)
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	// Execute print statements
	_, err = runtime.ExecuteString("test", `print("line1")`)
	if err != nil {
		t.Fatalf("failed to execute: %v", err)
	}
	_, err = runtime.ExecuteString("test", `print("line2")`)
	if err != nil {
		t.Fatalf("failed to execute: %v", err)
	}

	// Check output
	output := runtime.Output()
	if output != "line1\nline2\n" {
		t.Errorf("expected 'line1\\nline2\\n', got %q", output)
	}

	// Clear and check
	runtime.ClearOutput()
	output = runtime.Output()
	if output != "" {
		t.Errorf("expected empty output after clear, got %q", output)
	}
}

func TestConfig(t *testing.T) {
	config := RuntimeConfig{
		CPULimit:    5_000_000,
		MemoryLimit: 25 * 1024 * 1024,
		Stdout:      nil,
	}

	runtime, err := New(config)
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	got := runtime.Config()
	if got.CPULimit != config.CPULimit {
		t.Errorf("expected CPULimit %d, got %d", config.CPULimit, got.CPULimit)
	}
	if got.MemoryLimit != config.MemoryLimit {
		t.Errorf("expected MemoryLimit %d, got %d", config.MemoryLimit, got.MemoryLimit)
	}
}

func TestClose(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}

	// Close should not error
	if err := runtime.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Multiple closes should be safe
	if err := runtime.Close(); err != nil {
		t.Errorf("second Close() error = %v", err)
	}
}

func TestRuntimeReturnsUnderlyingRuntime(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	underlying := runtime.Runtime()
	if underlying == nil {
		t.Error("expected underlying runtime to be non-nil")
	}

	// Verify the runtime is functional by checking its global environment
	if underlying.GlobalEnv() == nil {
		t.Error("expected underlying runtime to have a valid global environment")
	}
}

func TestComplexLuaScript(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	// Test a more complex script with tables and loops
	code := `
		local sum = 0
		for i = 1, 10 do
			sum = sum + i
		end
		
		local data = {
			name = "test",
			value = sum
		}
		
		return data.value
	`

	result, err := runtime.ExecuteString("complex", code)
	if err != nil {
		t.Fatalf("failed to execute complex script: %v", err)
	}

	got, ok := rt.ToInt(result)
	if !ok || got != 55 {
		t.Errorf("expected 55 (sum of 1-10), got %v", result)
	}
}

func TestResourceLimits(t *testing.T) {
	// Create runtime with very low CPU limit
	config := RuntimeConfig{
		CPULimit:    100, // Very low limit
		MemoryLimit: 1 * 1024 * 1024,
		Stdout:      nil,
	}

	runtime, err := New(config)
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	// This should hit CPU limits with a tight loop
	// golua panics when limits are exceeded, so we need to recover
	code := `
		local sum = 0
		for i = 1, 100000 do
			sum = sum + i
		end
		return sum
	`

	// Expect the execution to panic due to CPU limit
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic due to CPU limit, but got none")
		} else {
			t.Logf("Caught expected panic: %v", r)
		}
	}()

	_, _ = runtime.ExecuteString("heavy", code)
}

func TestLoadFileFromFS(t *testing.T) {
	config := DefaultConfig()
	runtime, err := New(config)
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	// Create a test filesystem with embedded Lua files
	testFS := fstest.MapFS{
		"scripts/hello.lua": &fstest.MapFile{
			Data: []byte(`return "Hello from embedded FS"`),
		},
		"scripts/math.lua": &fstest.MapFile{
			Data: []byte(`return 2 + 2`),
		},
	}

	tests := []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{
			name:    "Load hello script",
			path:    "scripts/hello.lua",
			want:    "Hello from embedded FS",
			wantErr: false,
		},
		{
			name:    "Load math script",
			path:    "scripts/math.lua",
			want:    "4", // 2 + 2 = 4
			wantErr: false,
		},
		{
			name:    "Nonexistent file",
			path:    "scripts/nonexistent.lua",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			closure, err := runtime.LoadFileFromFS(testFS, tt.path)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("LoadFileFromFS failed: %v", err)
			}

			if closure == nil {
				t.Fatal("expected non-nil closure")
			}

			// Execute the closure
			result, err := runtime.Execute(closure)
			if err != nil {
				t.Fatalf("Execute failed: %v", err)
			}

			if tt.want != "" {
				// Convert result to string based on its type
				var resultStr string
				switch v := result.Interface().(type) {
				case string:
					resultStr = v
				case int64:
					resultStr = fmt.Sprintf("%d", v)
				case float64:
					resultStr = fmt.Sprintf("%g", v)
				default:
					resultStr = result.AsString()
				}
				if resultStr != tt.want {
					t.Errorf("expected result %q, got %q", tt.want, resultStr)
				}
			}
		})
	}
}

func TestSetFS(t *testing.T) {
	config := DefaultConfig()
	runtime, err := New(config)
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	// Create a test filesystem
	testFS := fstest.MapFS{
		"test.lua": &fstest.MapFile{
			Data: []byte(`return "test"`),
		},
	}

	// SetFS should not return an error
	runtime.SetFS(testFS)

	// Verify that fsys is set
	runtime.mu.RLock()
	if runtime.fsys == nil {
		t.Error("expected fsys to be set")
	}
	runtime.mu.RUnlock()

	// Verify we can still load files
	closure, err := runtime.LoadFileFromFS(testFS, "test.lua")
	if err != nil {
		t.Fatalf("LoadFileFromFS failed after SetFS: %v", err)
	}

	result, err := runtime.Execute(closure)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.AsString() != "test" {
		t.Errorf("expected result %q, got %q", "test", result.AsString())
	}
}

func TestSetFSNil(t *testing.T) {
	config := DefaultConfig()
	runtime, err := New(config)
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	// Setting nil should be allowed
	runtime.SetFS(nil)

	runtime.mu.RLock()
	if runtime.fsys != nil {
		t.Error("expected fsys to be nil")
	}
	runtime.mu.RUnlock()
}
