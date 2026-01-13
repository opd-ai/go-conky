//go:build integration

// Package integration provides end-to-end integration tests for conky-go.
// These tests verify that multiple components work together correctly.
//
// Note: Tests involving the lua package are excluded because it imports
// internal/render which depends on ebiten, and ebiten requires a display
// environment that is not available in CI.
package integration

import (
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/opd-ai/go-conky/internal/config"
	"github.com/opd-ai/go-conky/internal/monitor"
)

// getTestConfigsDir returns the path to the test configs directory.
func getTestConfigsDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "configs")
}

// TestConfigMonitorIntegration tests that parsed configs can be used with
// the system monitor to resolve template variables.
func TestConfigMonitorIntegration(t *testing.T) {
	parser, err := config.NewParser()
	if err != nil {
		t.Fatalf("NewParser failed: %v", err)
	}
	defer parser.Close()

	// Create a system monitor with a short interval
	mon := monitor.NewSystemMonitor(100 * time.Millisecond)
	if err := mon.Start(); err != nil {
		t.Fatalf("Monitor start failed: %v", err)
	}
	defer mon.Stop()

	// Wait for initial data collection
	time.Sleep(150 * time.Millisecond)

	configFiles := []string{
		"basic.conkyrc",
		"advanced.conkyrc",
		"minimal.conkyrc",
	}

	for _, cfgFile := range configFiles {
		t.Run(cfgFile, func(t *testing.T) {
			path := filepath.Join(getTestConfigsDir(), cfgFile)
			cfg, err := parser.ParseFile(path)
			if err != nil {
				t.Fatalf("ParseFile failed: %v", err)
			}

			// Validate the config
			if err := config.ValidateConfig(cfg); err != nil {
				t.Errorf("ValidateConfig failed: %v", err)
			}

			// Verify we can get monitor data for variables referenced in config
			data := mon.Data()

			// Check that CPU data is available (referenced in configs)
			if data.CPU.UsagePercent < 0 || data.CPU.UsagePercent > 100 {
				t.Errorf("CPU usage out of range: %f", data.CPU.UsagePercent)
			}

			// Check that memory data is available
			if data.Memory.Total == 0 {
				t.Error("Memory total should not be zero")
			}

			// Check that uptime data is available
			if data.Uptime.Seconds == 0 {
				t.Error("Uptime should not be zero")
			}

			// Verify config update interval is reasonable for monitoring
			if cfg.Display.UpdateInterval > 0 && cfg.Display.UpdateInterval < 50*time.Millisecond {
				t.Logf("Warning: Update interval %v is very fast", cfg.Display.UpdateInterval)
			}
		})
	}
}

// TestConfigVariableResolution tests that template variables in configs
// match the known variable set.
func TestConfigVariableResolution(t *testing.T) {
	parser, err := config.NewParser()
	if err != nil {
		t.Fatalf("NewParser failed: %v", err)
	}
	defer parser.Close()

	testConfigs := []struct {
		name        string
		file        string
		strictValid bool // should pass strict validation
	}{
		{
			name:        "basic legacy",
			file:        "basic.conkyrc",
			strictValid: true,
		},
		{
			name: "advanced legacy",
			file: "advanced.conkyrc",
			// Uses color0, color1 etc. which are formatting commands,
			// not data variables - strict validation will warn about these
			strictValid: false,
		},
		{
			name: "basic lua",
			file: "basic_lua.conkyrc",
			// Uses color1 which is a formatting command
			strictValid: false,
		},
		{
			name: "advanced lua",
			file: "advanced_lua.conkyrc",
			// Uses color0, color1 etc. which are formatting commands
			strictValid: false,
		},
	}

	for _, tc := range testConfigs {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(getTestConfigsDir(), tc.file)
			cfg, err := parser.ParseFile(path)
			if err != nil {
				t.Fatalf("ParseFile failed: %v", err)
			}

			// Non-strict validation should always pass for our test configs
			if err := config.ValidateConfig(cfg); err != nil {
				t.Errorf("Non-strict validation failed: %v", err)
			}

			// Strict validation should pass if expected
			if tc.strictValid {
				validator := config.NewValidator().WithStrictMode(true)
				result := validator.Validate(cfg)
				if !result.IsValid() {
					t.Errorf("Strict validation failed: %v", result.Errors)
				}
			}
		})
	}
}

// TestMonitorDataConsistency tests that the system monitor returns
// consistent data across multiple reads.
func TestMonitorDataConsistency(t *testing.T) {
	mon := monitor.NewSystemMonitor(50 * time.Millisecond)
	if err := mon.Start(); err != nil {
		t.Fatalf("Monitor start failed: %v", err)
	}
	defer mon.Stop()

	// Wait for initial data
	time.Sleep(100 * time.Millisecond)

	// Take multiple readings
	readings := make([]monitor.SystemData, 5)
	for i := range readings {
		readings[i] = mon.Data()
		time.Sleep(60 * time.Millisecond)
	}

	// Verify consistency
	for i, data := range readings {
		// Memory total should be constant
		if i > 0 && data.Memory.Total != readings[0].Memory.Total {
			t.Errorf("Memory total changed between readings: %d vs %d",
				readings[0].Memory.Total, data.Memory.Total)
		}

		// CPU usage should be within valid range
		if data.CPU.UsagePercent < 0 || data.CPU.UsagePercent > 100 {
			t.Errorf("Reading %d: CPU usage out of range: %f", i, data.CPU.UsagePercent)
		}

		// Uptime should increase
		if i > 0 && data.Uptime.Seconds < readings[i-1].Uptime.Seconds {
			t.Errorf("Uptime decreased: %v -> %v",
				readings[i-1].Uptime.Seconds, data.Uptime.Seconds)
		}
	}
}

// TestConfigMigrationIntegration tests that migrated configs can be re-parsed.
func TestConfigMigrationIntegration(t *testing.T) {
	legacyConfigs := []string{
		"basic.conkyrc",
		"advanced.conkyrc",
	}

	for _, cfgFile := range legacyConfigs {
		t.Run(cfgFile, func(t *testing.T) {
			path := filepath.Join(getTestConfigsDir(), cfgFile)

			// Migrate to Lua format
			luaContent, err := config.MigrateLegacyFile(path)
			if err != nil {
				t.Fatalf("MigrateLegacyFile failed: %v", err)
			}

			// Parse the migrated content
			parser, err := config.NewParser()
			if err != nil {
				t.Fatalf("NewParser failed: %v", err)
			}
			defer parser.Close()

			migratedCfg, err := parser.Parse(luaContent)
			if err != nil {
				t.Fatalf("Parse migrated content failed: %v\nContent:\n%s", err, string(luaContent))
			}

			// Validate the migrated config
			if err := config.ValidateConfig(migratedCfg); err != nil {
				t.Errorf("Migrated config validation failed: %v", err)
			}
		})
	}
}

// TestFullPipelineIntegration tests the complete flow from config parsing
// through monitoring and validation.
func TestFullPipelineIntegration(t *testing.T) {
	// Parse a config
	parser, err := config.NewParser()
	if err != nil {
		t.Fatalf("NewParser failed: %v", err)
	}
	defer parser.Close()

	path := filepath.Join(getTestConfigsDir(), "advanced.conkyrc")
	cfg, err := parser.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// Create a monitor with the config's update interval
	updateInterval := cfg.Display.UpdateInterval
	if updateInterval == 0 {
		updateInterval = time.Second
	}
	mon := monitor.NewSystemMonitor(updateInterval)
	if err := mon.Start(); err != nil {
		t.Fatalf("Monitor start failed: %v", err)
	}
	defer mon.Stop()

	// Wait for data collection
	time.Sleep(updateInterval + 50*time.Millisecond)

	// Verify all components are working
	data := mon.Data()

	// Verify expected data fields based on config template variables
	// The advanced config uses: cpu, mem, memmax, memperc, uptime, fs_used, fs_size, etc.

	if data.CPU.UsagePercent < 0 {
		t.Error("CPU usage should be non-negative")
	}

	if data.Memory.Total == 0 {
		t.Error("Memory total should not be zero")
	}

	if data.Memory.Used > data.Memory.Total {
		t.Error("Memory used should not exceed total")
	}

	if data.Uptime.Seconds == 0 {
		t.Error("Uptime should not be zero")
	}

	// Validate the config one more time
	if err := config.ValidateConfig(cfg); err != nil {
		t.Errorf("Config validation failed: %v", err)
	}
}

// TestLuaConfigParsing tests that Lua config files can be parsed correctly.
func TestLuaConfigParsing(t *testing.T) {
	luaConfigs := []struct {
		name              string
		file              string
		expectNonEmptyTpl bool
	}{
		{"basic_lua", "basic_lua.conkyrc", true},
		{"advanced_lua", "advanced_lua.conkyrc", true},
		{"minimal_lua", "minimal_lua.conkyrc", false}, // minimal config may have empty template
	}

	for _, tc := range luaConfigs {
		t.Run(tc.name, func(t *testing.T) {
			parser, err := config.NewParser()
			if err != nil {
				t.Fatalf("NewParser failed: %v", err)
			}
			defer parser.Close()

			path := filepath.Join(getTestConfigsDir(), tc.file)
			cfg, err := parser.ParseFile(path)
			if err != nil {
				t.Fatalf("ParseFile failed: %v", err)
			}

			// Validate the config
			if err := config.ValidateConfig(cfg); err != nil {
				t.Errorf("ValidateConfig failed: %v", err)
			}

			// Verify that the config has valid text template if expected
			if tc.expectNonEmptyTpl && len(cfg.Text.Template) == 0 {
				t.Error("Expected non-empty text template")
			}
		})
	}
}

// TestConfigAndMonitorVariableMapping tests that config template variables
// can be resolved using monitor data.
func TestConfigAndMonitorVariableMapping(t *testing.T) {
	// Parse a config with many variables
	parser, err := config.NewParser()
	if err != nil {
		t.Fatalf("NewParser failed: %v", err)
	}
	defer parser.Close()

	path := filepath.Join(getTestConfigsDir(), "advanced.conkyrc")
	cfg, err := parser.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// Create a monitor
	mon := monitor.NewSystemMonitor(100 * time.Millisecond)
	if err := mon.Start(); err != nil {
		t.Fatalf("Monitor start failed: %v", err)
	}
	defer mon.Stop()

	// Wait for data
	time.Sleep(150 * time.Millisecond)

	// Get monitor data
	data := mon.Data()

	// The advanced config uses these variables, verify monitor provides data:
	// CPU: ${cpu}, ${cpu cpu0}, ${cpu cpu1}, ${freq_g}, ${loadavg}
	if len(data.CPU.Cores) == 0 {
		t.Error("Expected at least one CPU core")
	}
	if data.CPU.Frequency <= 0 {
		t.Logf("CPU frequency not available (may be expected): %f", data.CPU.Frequency)
	}

	// Memory: ${mem}, ${memmax}, ${memperc}, ${swap}, ${swapmax}, ${swapperc}
	if data.Memory.Total == 0 {
		t.Error("Memory total should not be zero")
	}
	if data.Memory.UsagePercent < 0 || data.Memory.UsagePercent > 100 {
		t.Errorf("Memory usage percent out of range: %f", data.Memory.UsagePercent)
	}

	// Filesystem: ${fs_used /}, ${fs_size /}
	if len(data.Filesystem.Mounts) == 0 {
		t.Error("Expected at least one filesystem mount")
	}
	if _, ok := data.Filesystem.Mounts["/"]; !ok {
		t.Error("Expected root filesystem mount")
	}

	// Network: ${downspeed}, ${upspeed}, ${totaldown}, ${totalup}
	// Network interfaces may vary, just check the map exists
	if data.Network.Interfaces == nil {
		t.Error("Network interfaces map should not be nil")
	}

	// Process: ${processes}, ${running_processes}
	if data.Process.TotalProcesses <= 0 {
		t.Error("Expected at least one process")
	}

	// Uptime: ${uptime}
	if data.Uptime.Seconds == 0 {
		t.Error("Uptime should not be zero")
	}

	// Validate config matches expected patterns
	if len(cfg.Text.Template) == 0 {
		t.Error("Config should have text template")
	}
}

// TestMonitorStartStop tests that the monitor can be started and stopped cleanly.
func TestMonitorStartStop(t *testing.T) {
	mon := monitor.NewSystemMonitor(100 * time.Millisecond)

	// Start
	if err := mon.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if !mon.IsRunning() {
		t.Error("Monitor should be running after Start")
	}

	// Wait for some updates
	time.Sleep(250 * time.Millisecond)

	// Stop
	mon.Stop()

	if mon.IsRunning() {
		t.Error("Monitor should not be running after Stop")
	}

	// Start again (should work)
	if err := mon.Start(); err != nil {
		t.Fatalf("Second start failed: %v", err)
	}
	defer mon.Stop()

	if !mon.IsRunning() {
		t.Error("Monitor should be running after second Start")
	}
}

// TestConfigParsingErrors tests that config parsing reports errors correctly.
func TestConfigParsingErrors(t *testing.T) {
	parser, err := config.NewParser()
	if err != nil {
		t.Fatalf("NewParser failed: %v", err)
	}
	defer parser.Close()

	invalidConfigs := []struct {
		name    string
		content string
	}{
		{
			name:    "invalid window type",
			content: "own_window_type invalid_type",
		},
		{
			name:    "invalid alignment",
			content: "alignment invalid_alignment",
		},
		{
			name:    "invalid color",
			content: "default_color gggggg",
		},
		{
			name:    "invalid width",
			content: "minimum_width not_a_number",
		},
	}

	for _, tc := range invalidConfigs {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parser.Parse([]byte(tc.content))
			if err == nil {
				t.Errorf("Expected error for %s, got nil", tc.name)
			}
		})
	}
}
