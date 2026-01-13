// Package lua provides Golua integration for conky-go.
// This file implements the Conky Lua event hook system, which allows
// Lua scripts to define callbacks that are invoked at specific lifecycle points.
package lua

import (
	"fmt"
	"sync"

	rt "github.com/arnodel/golua/runtime"
)

// HookType represents the different types of Conky lifecycle hooks.
type HookType int

const (
	// HookInvalid represents an invalid or unknown hook type.
	// This is returned by ParseHookType when parsing fails.
	HookInvalid HookType = iota

	// HookStartup is called once when Conky starts or reloads configuration.
	// Use for initialization code, resource allocation, and setup.
	HookStartup

	// HookShutdown is called once when Conky exits or configuration is reloaded.
	// Use for cleanup code, closing files, and freeing resources.
	HookShutdown

	// HookMain is called every update cycle.
	// Use for drawing and processing that needs to happen every refresh.
	HookMain

	// HookDrawPre is called before each redraw of the window.
	// Use for pre-rendering setup or calculations.
	HookDrawPre

	// HookDrawPost is called after each redraw of the window.
	// Use for post-rendering cleanup or overlays.
	HookDrawPost
)

// String returns the string representation of a HookType.
func (h HookType) String() string {
	switch h {
	case HookStartup:
		return "startup"
	case HookShutdown:
		return "shutdown"
	case HookMain:
		return "main"
	case HookDrawPre:
		return "draw_pre"
	case HookDrawPost:
		return "draw_post"
	case HookInvalid:
		return "invalid"
	default:
		return "unknown"
	}
}

// LuaFunctionName returns the Lua function name for a hook type.
// Conky convention is that all hook functions start with "conky_" prefix.
func (h HookType) LuaFunctionName() string {
	return "conky_" + h.String()
}

// ParseHookType parses a string into a HookType.
// Returns HookInvalid and an error if the string is not a valid hook type.
func ParseHookType(s string) (HookType, error) {
	switch s {
	case "startup":
		return HookStartup, nil
	case "shutdown":
		return HookShutdown, nil
	case "main":
		return HookMain, nil
	case "draw_pre":
		return HookDrawPre, nil
	case "draw_post":
		return HookDrawPost, nil
	default:
		return HookInvalid, fmt.Errorf("unknown hook type: %s", s)
	}
}

// HookManager manages Conky Lua lifecycle hooks.
// It provides thread-safe hook registration and invocation.
type HookManager struct {
	runtime *ConkyRuntime
	hooks   map[HookType]string // Maps hook type to function name
	mu      sync.RWMutex
}

// NewHookManager creates a new HookManager for the given runtime.
func NewHookManager(runtime *ConkyRuntime) (*HookManager, error) {
	if runtime == nil {
		return nil, fmt.Errorf("runtime cannot be nil")
	}

	return &HookManager{
		runtime: runtime,
		hooks:   make(map[HookType]string),
	}, nil
}

// RegisterHook registers a Lua function to be called for a specific hook type.
// The funcName is the name of the Lua function without the "conky_" prefix.
// For example, to register "conky_main", pass "main" as funcName.
func (hm *HookManager) RegisterHook(hookType HookType, funcName string) error {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	// Build the full function name with conky_ prefix
	fullName := "conky_" + funcName

	// Verify the function exists in the Lua environment
	fn := hm.runtime.GetGlobal(fullName)
	if fn == rt.NilValue {
		return fmt.Errorf("Lua function %s not found", fullName)
	}

	// Verify it's actually a function
	if fn.Type() != rt.FunctionType {
		return fmt.Errorf("%s is not a function (type: %v)", fullName, fn.Type())
	}

	hm.hooks[hookType] = funcName
	return nil
}

// UnregisterHook removes a hook registration.
func (hm *HookManager) UnregisterHook(hookType HookType) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	delete(hm.hooks, hookType)
}

// IsRegistered returns true if a hook is registered for the given type.
func (hm *HookManager) IsRegistered(hookType HookType) bool {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	_, ok := hm.hooks[hookType]
	return ok
}

// GetRegisteredFunctionName returns the registered function name for a hook type.
// Returns empty string if no hook is registered.
func (hm *HookManager) GetRegisteredFunctionName(hookType HookType) string {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	return hm.hooks[hookType]
}

// Call invokes the registered hook function for the given hook type.
// Returns nil if no hook is registered for the type.
// Returns an error if the hook function fails during execution.
func (hm *HookManager) Call(hookType HookType, args ...rt.Value) (rt.Value, error) {
	hm.mu.RLock()
	funcName, ok := hm.hooks[hookType]
	hm.mu.RUnlock()

	if !ok {
		return rt.NilValue, nil // No hook registered, not an error
	}

	// Call the hook function using the runtime's CallFunction method
	fullName := "conky_" + funcName
	result, err := hm.runtime.CallFunction(fullName, args...)
	if err != nil {
		return rt.NilValue, fmt.Errorf("hook %s execution failed: %w", hookType.String(), err)
	}

	return result, nil
}

// CallIfExists invokes the hook only if the function exists in Lua.
// Unlike Call, this doesn't require prior registration with RegisterHook.
// It looks up the function by the standard conky_<hooktype> naming convention.
func (hm *HookManager) CallIfExists(hookType HookType, args ...rt.Value) (rt.Value, error) {
	fullName := hookType.LuaFunctionName()

	// Check if the function exists
	fn := hm.runtime.GetGlobal(fullName)
	if fn == rt.NilValue {
		return rt.NilValue, nil // Function doesn't exist, not an error
	}

	// Call the function
	result, err := hm.runtime.CallFunction(fullName, args...)
	if err != nil {
		return rt.NilValue, fmt.Errorf("hook %s execution failed: %w", hookType.String(), err)
	}

	return result, nil
}

// AutoRegisterHooks scans the Lua environment for standard Conky hook functions
// and automatically registers them. It looks for conky_startup, conky_shutdown,
// conky_main, conky_draw_pre, and conky_draw_post.
func (hm *HookManager) AutoRegisterHooks() []HookType {
	hookTypes := []HookType{
		HookStartup,
		HookShutdown,
		HookMain,
		HookDrawPre,
		HookDrawPost,
	}

	// Collect valid hooks first without holding the lock
	foundHooks := make([]HookType, 0, len(hookTypes))

	for _, hookType := range hookTypes {
		fullName := hookType.LuaFunctionName()
		fn := hm.runtime.GetGlobal(fullName)
		if fn != rt.NilValue && fn.Type() == rt.FunctionType {
			foundHooks = append(foundHooks, hookType)
		}
	}

	// Now acquire lock once to update all hooks
	hm.mu.Lock()
	for _, hookType := range foundHooks {
		hm.hooks[hookType] = hookType.String()
	}
	hm.mu.Unlock()

	return foundHooks
}

// RegisteredHooks returns a list of all currently registered hook types.
func (hm *HookManager) RegisteredHooks() []HookType {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	hooks := make([]HookType, 0, len(hm.hooks))
	for hookType := range hm.hooks {
		hooks = append(hooks, hookType)
	}
	return hooks
}

// Clear removes all hook registrations.
func (hm *HookManager) Clear() {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	hm.hooks = make(map[HookType]string)
}
