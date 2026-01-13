package lua

import (
	"testing"

	rt "github.com/arnodel/golua/runtime"
)

func TestHookTypeString(t *testing.T) {
	tests := []struct {
		hookType HookType
		expected string
	}{
		{HookStartup, "startup"},
		{HookShutdown, "shutdown"},
		{HookMain, "main"},
		{HookDrawPre, "draw_pre"},
		{HookDrawPost, "draw_post"},
		{HookType(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.hookType.String()
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestHookTypeLuaFunctionName(t *testing.T) {
	tests := []struct {
		hookType HookType
		expected string
	}{
		{HookStartup, "conky_startup"},
		{HookShutdown, "conky_shutdown"},
		{HookMain, "conky_main"},
		{HookDrawPre, "conky_draw_pre"},
		{HookDrawPost, "conky_draw_post"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.hookType.LuaFunctionName()
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestParseHookType(t *testing.T) {
	tests := []struct {
		input    string
		expected HookType
		wantErr  bool
	}{
		{"startup", HookStartup, false},
		{"shutdown", HookShutdown, false},
		{"main", HookMain, false},
		{"draw_pre", HookDrawPre, false},
		{"draw_post", HookDrawPost, false},
		{"invalid", HookStartup, true},
		{"", HookStartup, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParseHookType(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseHookType(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestNewHookManager(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	hm, err := NewHookManager(runtime)
	if err != nil {
		t.Fatalf("failed to create hook manager: %v", err)
	}

	if hm == nil {
		t.Error("expected hook manager to be non-nil")
	}
}

func TestNewHookManagerWithNilRuntime(t *testing.T) {
	_, err := NewHookManager(nil)
	if err == nil {
		t.Error("expected error for nil runtime")
	}
}

func TestRegisterHook(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	// Define a Lua function
	_, err = runtime.ExecuteString("setup", `
		function conky_main()
			return "main called"
		end
	`)
	if err != nil {
		t.Fatalf("failed to define Lua function: %v", err)
	}

	hm, err := NewHookManager(runtime)
	if err != nil {
		t.Fatalf("failed to create hook manager: %v", err)
	}

	// Register the hook
	err = hm.RegisterHook(HookMain, "main")
	if err != nil {
		t.Fatalf("failed to register hook: %v", err)
	}

	// Verify it's registered
	if !hm.IsRegistered(HookMain) {
		t.Error("expected HookMain to be registered")
	}

	funcName := hm.GetRegisteredFunctionName(HookMain)
	if funcName != "main" {
		t.Errorf("expected 'main', got %q", funcName)
	}
}

func TestRegisterHookNonExistentFunction(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	hm, err := NewHookManager(runtime)
	if err != nil {
		t.Fatalf("failed to create hook manager: %v", err)
	}

	// Try to register a non-existent function
	err = hm.RegisterHook(HookMain, "nonexistent")
	if err == nil {
		t.Error("expected error for non-existent function")
	}
}

func TestRegisterHookNonFunction(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	// Set a global that's not a function
	runtime.SetGlobal("conky_notfunc", rt.StringValue("not a function"))

	hm, err := NewHookManager(runtime)
	if err != nil {
		t.Fatalf("failed to create hook manager: %v", err)
	}

	// Try to register a non-function
	err = hm.RegisterHook(HookMain, "notfunc")
	if err == nil {
		t.Error("expected error for non-function value")
	}
}

func TestUnregisterHook(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	// Define a Lua function
	_, err = runtime.ExecuteString("setup", `
		function conky_startup()
			return true
		end
	`)
	if err != nil {
		t.Fatalf("failed to define Lua function: %v", err)
	}

	hm, err := NewHookManager(runtime)
	if err != nil {
		t.Fatalf("failed to create hook manager: %v", err)
	}

	// Register and then unregister
	err = hm.RegisterHook(HookStartup, "startup")
	if err != nil {
		t.Fatalf("failed to register hook: %v", err)
	}

	if !hm.IsRegistered(HookStartup) {
		t.Error("expected HookStartup to be registered")
	}

	hm.UnregisterHook(HookStartup)

	if hm.IsRegistered(HookStartup) {
		t.Error("expected HookStartup to be unregistered")
	}
}

func TestCallHook(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	// Define a Lua function that returns a value
	_, err = runtime.ExecuteString("setup", `
		function conky_main()
			return 42
		end
	`)
	if err != nil {
		t.Fatalf("failed to define Lua function: %v", err)
	}

	hm, err := NewHookManager(runtime)
	if err != nil {
		t.Fatalf("failed to create hook manager: %v", err)
	}

	err = hm.RegisterHook(HookMain, "main")
	if err != nil {
		t.Fatalf("failed to register hook: %v", err)
	}

	// Call the hook
	result, err := hm.Call(HookMain)
	if err != nil {
		t.Fatalf("failed to call hook: %v", err)
	}

	got, ok := rt.ToInt(result)
	if !ok || got != 42 {
		t.Errorf("expected 42, got %v", result)
	}
}

func TestCallUnregisteredHook(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	hm, err := NewHookManager(runtime)
	if err != nil {
		t.Fatalf("failed to create hook manager: %v", err)
	}

	// Call an unregistered hook should return nil without error
	result, err := hm.Call(HookMain)
	if err != nil {
		t.Errorf("expected no error for unregistered hook, got %v", err)
	}
	if result != rt.NilValue {
		t.Errorf("expected NilValue, got %v", result)
	}
}

func TestCallIfExists(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	// Define standard hook functions
	_, err = runtime.ExecuteString("setup", `
		function conky_startup()
			return "started"
		end
		function conky_shutdown()
			return "stopped"
		end
	`)
	if err != nil {
		t.Fatalf("failed to define Lua functions: %v", err)
	}

	hm, err := NewHookManager(runtime)
	if err != nil {
		t.Fatalf("failed to create hook manager: %v", err)
	}

	// Call existing hook
	result, err := hm.CallIfExists(HookStartup)
	if err != nil {
		t.Fatalf("failed to call hook: %v", err)
	}
	if result.AsString() != "started" {
		t.Errorf("expected 'started', got %q", result.AsString())
	}

	// Call existing hook
	result, err = hm.CallIfExists(HookShutdown)
	if err != nil {
		t.Fatalf("failed to call hook: %v", err)
	}
	if result.AsString() != "stopped" {
		t.Errorf("expected 'stopped', got %q", result.AsString())
	}

	// Call non-existing hook should return nil without error
	result, err = hm.CallIfExists(HookMain)
	if err != nil {
		t.Errorf("expected no error for non-existing hook, got %v", err)
	}
	if result != rt.NilValue {
		t.Errorf("expected NilValue, got %v", result)
	}
}

func TestAutoRegisterHooks(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	// Define some standard hook functions
	_, err = runtime.ExecuteString("setup", `
		function conky_startup()
			return "startup"
		end
		function conky_main()
			return "main"
		end
		function conky_shutdown()
			return "shutdown"
		end
	`)
	if err != nil {
		t.Fatalf("failed to define Lua functions: %v", err)
	}

	hm, err := NewHookManager(runtime)
	if err != nil {
		t.Fatalf("failed to create hook manager: %v", err)
	}

	// Auto-register hooks
	registered := hm.AutoRegisterHooks()

	// Verify the expected hooks were registered
	if len(registered) != 3 {
		t.Errorf("expected 3 hooks registered, got %d", len(registered))
	}

	if !hm.IsRegistered(HookStartup) {
		t.Error("expected HookStartup to be registered")
	}
	if !hm.IsRegistered(HookMain) {
		t.Error("expected HookMain to be registered")
	}
	if !hm.IsRegistered(HookShutdown) {
		t.Error("expected HookShutdown to be registered")
	}
	if hm.IsRegistered(HookDrawPre) {
		t.Error("expected HookDrawPre to NOT be registered")
	}
	if hm.IsRegistered(HookDrawPost) {
		t.Error("expected HookDrawPost to NOT be registered")
	}
}

func TestRegisteredHooks(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = runtime.ExecuteString("setup", `
		function conky_main()
			return true
		end
		function conky_draw_pre()
			return true
		end
	`)
	if err != nil {
		t.Fatalf("failed to define Lua functions: %v", err)
	}

	hm, err := NewHookManager(runtime)
	if err != nil {
		t.Fatalf("failed to create hook manager: %v", err)
	}

	err = hm.RegisterHook(HookMain, "main")
	if err != nil {
		t.Fatalf("failed to register hook: %v", err)
	}
	err = hm.RegisterHook(HookDrawPre, "draw_pre")
	if err != nil {
		t.Fatalf("failed to register hook: %v", err)
	}

	hooks := hm.RegisteredHooks()
	if len(hooks) != 2 {
		t.Errorf("expected 2 registered hooks, got %d", len(hooks))
	}
}

func TestClear(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = runtime.ExecuteString("setup", `
		function conky_main()
			return true
		end
	`)
	if err != nil {
		t.Fatalf("failed to define Lua function: %v", err)
	}

	hm, err := NewHookManager(runtime)
	if err != nil {
		t.Fatalf("failed to create hook manager: %v", err)
	}

	err = hm.RegisterHook(HookMain, "main")
	if err != nil {
		t.Fatalf("failed to register hook: %v", err)
	}

	if !hm.IsRegistered(HookMain) {
		t.Error("expected HookMain to be registered")
	}

	hm.Clear()

	if hm.IsRegistered(HookMain) {
		t.Error("expected no hooks after Clear")
	}

	hooks := hm.RegisteredHooks()
	if len(hooks) != 0 {
		t.Errorf("expected 0 hooks after Clear, got %d", len(hooks))
	}
}

func TestCallHookWithArguments(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	// Define a function that takes arguments
	_, err = runtime.ExecuteString("setup", `
		function conky_draw_pre(x, y)
			return x + y
		end
	`)
	if err != nil {
		t.Fatalf("failed to define Lua function: %v", err)
	}

	hm, err := NewHookManager(runtime)
	if err != nil {
		t.Fatalf("failed to create hook manager: %v", err)
	}

	err = hm.RegisterHook(HookDrawPre, "draw_pre")
	if err != nil {
		t.Fatalf("failed to register hook: %v", err)
	}

	// Call with arguments
	result, err := hm.Call(HookDrawPre, rt.IntValue(10), rt.IntValue(20))
	if err != nil {
		t.Fatalf("failed to call hook with arguments: %v", err)
	}

	got, ok := rt.ToInt(result)
	if !ok || got != 30 {
		t.Errorf("expected 30, got %v", result)
	}
}

func TestCallIfExistsWithArguments(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	// Define a function that takes arguments
	_, err = runtime.ExecuteString("setup", `
		function conky_draw_post(message)
			return "received: " .. message
		end
	`)
	if err != nil {
		t.Fatalf("failed to define Lua function: %v", err)
	}

	hm, err := NewHookManager(runtime)
	if err != nil {
		t.Fatalf("failed to create hook manager: %v", err)
	}

	// Call with arguments
	result, err := hm.CallIfExists(HookDrawPost, rt.StringValue("hello"))
	if err != nil {
		t.Fatalf("failed to call hook: %v", err)
	}

	expected := "received: hello"
	if result.AsString() != expected {
		t.Errorf("expected %q, got %q", expected, result.AsString())
	}
}

func TestHookManagerConcurrency(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = runtime.ExecuteString("setup", `
		function conky_main()
			return 1
		end
	`)
	if err != nil {
		t.Fatalf("failed to define Lua function: %v", err)
	}

	hm, err := NewHookManager(runtime)
	if err != nil {
		t.Fatalf("failed to create hook manager: %v", err)
	}

	err = hm.RegisterHook(HookMain, "main")
	if err != nil {
		t.Fatalf("failed to register hook: %v", err)
	}

	// Test concurrent reads
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			_ = hm.IsRegistered(HookMain)
			_ = hm.GetRegisteredFunctionName(HookMain)
			_ = hm.RegisteredHooks()
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestGetRegisteredFunctionNameUnregistered(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	hm, err := NewHookManager(runtime)
	if err != nil {
		t.Fatalf("failed to create hook manager: %v", err)
	}

	// Get function name for unregistered hook
	funcName := hm.GetRegisteredFunctionName(HookMain)
	if funcName != "" {
		t.Errorf("expected empty string for unregistered hook, got %q", funcName)
	}
}

func TestMultipleHookRegistrations(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	_, err = runtime.ExecuteString("setup", `
		function conky_startup()
			return "startup"
		end
		function conky_main()
			return "main"
		end
		function conky_shutdown()
			return "shutdown"
		end
		function conky_draw_pre()
			return "draw_pre"
		end
		function conky_draw_post()
			return "draw_post"
		end
	`)
	if err != nil {
		t.Fatalf("failed to define Lua functions: %v", err)
	}

	hm, err := NewHookManager(runtime)
	if err != nil {
		t.Fatalf("failed to create hook manager: %v", err)
	}

	// Register all hooks
	hooks := []struct {
		hookType HookType
		name     string
	}{
		{HookStartup, "startup"},
		{HookMain, "main"},
		{HookShutdown, "shutdown"},
		{HookDrawPre, "draw_pre"},
		{HookDrawPost, "draw_post"},
	}

	for _, h := range hooks {
		err := hm.RegisterHook(h.hookType, h.name)
		if err != nil {
			t.Fatalf("failed to register %s: %v", h.name, err)
		}
	}

	// Call all hooks and verify results
	for _, h := range hooks {
		result, err := hm.Call(h.hookType)
		if err != nil {
			t.Fatalf("failed to call %s: %v", h.name, err)
		}
		if result.AsString() != h.name {
			t.Errorf("expected %q, got %q", h.name, result.AsString())
		}
	}
}
