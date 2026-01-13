package lua

import (
	"strings"
	"testing"
	"time"

	rt "github.com/arnodel/golua/runtime"

	"github.com/opd-ai/go-conky/internal/monitor"
)

// mockSystemDataProvider implements SystemDataProvider for testing.
type mockSystemDataProvider struct {
	cpu        monitor.CPUStats
	memory     monitor.MemoryStats
	uptime     monitor.UptimeStats
	network    monitor.NetworkStats
	filesystem monitor.FilesystemStats
	diskIO     monitor.DiskIOStats
	hwmon      monitor.HwmonStats
	process    monitor.ProcessStats
	battery    monitor.BatteryStats
	audio      monitor.AudioStats
}

func (m *mockSystemDataProvider) CPU() monitor.CPUStats               { return m.cpu }
func (m *mockSystemDataProvider) Memory() monitor.MemoryStats         { return m.memory }
func (m *mockSystemDataProvider) Uptime() monitor.UptimeStats         { return m.uptime }
func (m *mockSystemDataProvider) Network() monitor.NetworkStats       { return m.network }
func (m *mockSystemDataProvider) Filesystem() monitor.FilesystemStats { return m.filesystem }
func (m *mockSystemDataProvider) DiskIO() monitor.DiskIOStats         { return m.diskIO }
func (m *mockSystemDataProvider) Hwmon() monitor.HwmonStats           { return m.hwmon }
func (m *mockSystemDataProvider) Process() monitor.ProcessStats       { return m.process }
func (m *mockSystemDataProvider) Battery() monitor.BatteryStats       { return m.battery }
func (m *mockSystemDataProvider) Audio() monitor.AudioStats           { return m.audio }

func newMockProvider() *mockSystemDataProvider {
	return &mockSystemDataProvider{
		cpu: monitor.CPUStats{
			UsagePercent: 45.5,
			Cores:        []float64{50.0, 40.0, 55.0, 35.0},
			CPUCount:     4,
			ModelName:    "Intel Core i7-9700K",
			Frequency:    3600.0,
		},
		memory: monitor.MemoryStats{
			Total:        16 * 1024 * 1024 * 1024, // 16 GiB
			Used:         8 * 1024 * 1024 * 1024,  // 8 GiB
			Free:         4 * 1024 * 1024 * 1024,  // 4 GiB
			Available:    6 * 1024 * 1024 * 1024,  // 6 GiB
			Buffers:      512 * 1024 * 1024,       // 512 MiB
			Cached:       2 * 1024 * 1024 * 1024,  // 2 GiB
			SwapTotal:    8 * 1024 * 1024 * 1024,  // 8 GiB
			SwapUsed:     1 * 1024 * 1024 * 1024,  // 1 GiB
			SwapFree:     7 * 1024 * 1024 * 1024,  // 7 GiB
			UsagePercent: 50.0,
			SwapPercent:  12.5,
		},
		uptime: monitor.UptimeStats{
			Seconds:     90061, // 1 day, 1 hour, 1 minute, 1 second
			IdleSeconds: 45000,
			Duration:    time.Duration(90061) * time.Second,
		},
		network: monitor.NetworkStats{
			Interfaces: map[string]monitor.InterfaceStats{
				"eth0": {
					Name:          "eth0",
					RxBytes:       1024 * 1024 * 100, // 100 MiB
					TxBytes:       1024 * 1024 * 50,  // 50 MiB
					RxBytesPerSec: 1024 * 100,        // 100 KiB/s
					TxBytesPerSec: 1024 * 50,         // 50 KiB/s
				},
			},
			TotalRxBytes:       1024 * 1024 * 100,
			TotalTxBytes:       1024 * 1024 * 50,
			TotalRxBytesPerSec: 1024 * 100,
			TotalTxBytesPerSec: 1024 * 50,
		},
		filesystem: monitor.FilesystemStats{
			Mounts: map[string]monitor.MountStats{
				"/": {
					MountPoint:   "/",
					Total:        500 * 1024 * 1024 * 1024, // 500 GiB
					Used:         200 * 1024 * 1024 * 1024, // 200 GiB
					Free:         300 * 1024 * 1024 * 1024, // 300 GiB
					Available:    280 * 1024 * 1024 * 1024, // 280 GiB
					UsagePercent: 40.0,
				},
				"/home": {
					MountPoint:   "/home",
					Total:        1024 * 1024 * 1024 * 1024, // 1 TiB
					Used:         512 * 1024 * 1024 * 1024,  // 512 GiB
					Free:         512 * 1024 * 1024 * 1024,  // 512 GiB
					Available:    500 * 1024 * 1024 * 1024,  // 500 GiB
					UsagePercent: 50.0,
				},
			},
		},
		diskIO: monitor.DiskIOStats{
			Disks: map[string]monitor.DiskStats{
				"sda": {
					Name:            "sda",
					ReadBytesPerSec: 1024 * 1024, // 1 MiB/s
				},
			},
		},
		hwmon: monitor.HwmonStats{
			TempSensors: []monitor.TempSensor{
				{Label: "CPU", InputCelsius: 55.0},
				{Label: "GPU", InputCelsius: 65.0},
			},
		},
		process: monitor.ProcessStats{
			TotalProcesses:   150,
			RunningProcesses: 5,
			TotalThreads:     500,
		},
		battery: monitor.BatteryStats{
			Batteries: map[string]monitor.BatteryInfo{
				"BAT0": {
					Capacity: 85,
					Status:   "Discharging",
				},
			},
			TotalCapacity: 85.0,
			IsDischarging: true,
		},
		audio: monitor.AudioStats{
			HasAudio:     true,
			MasterVolume: 75.0,
		},
	}
}

func TestNewConkyAPI(t *testing.T) {
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

	if api == nil {
		t.Error("expected API to be non-nil")
	}
}

func TestNewConkyAPIWithNilRuntime(t *testing.T) {
	_, err := NewConkyAPI(nil, nil)
	if err == nil {
		t.Error("expected error for nil runtime")
	}
}

func TestConkyAPIRegistersConkyParse(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	provider := newMockProvider()
	_, err = NewConkyAPI(runtime, provider)
	if err != nil {
		t.Fatalf("failed to create API: %v", err)
	}

	// Verify conky_parse is registered
	val := runtime.GetGlobal("conky_parse")
	if val == rt.NilValue {
		t.Error("expected conky_parse to be registered")
	}
}

func TestConkyAPIRegistersConkyTable(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	provider := newMockProvider()
	_, err = NewConkyAPI(runtime, provider)
	if err != nil {
		t.Fatalf("failed to create API: %v", err)
	}

	// Verify conky table is registered
	val := runtime.GetGlobal("conky")
	if val == rt.NilValue {
		t.Error("expected conky table to be registered")
	}
}

func TestConkyParseFromLua(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	provider := newMockProvider()
	_, err = NewConkyAPI(runtime, provider)
	if err != nil {
		t.Fatalf("failed to create API: %v", err)
	}

	// Test calling conky_parse from Lua
	result, err := runtime.ExecuteString("test", `return conky_parse("CPU: ${cpu}%")`)
	if err != nil {
		t.Fatalf("failed to execute Lua: %v", err)
	}

	got := result.AsString()
	expected := "CPU: 46%"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestParseCPUVariables(t *testing.T) {
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

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "cpu usage",
			template: "${cpu}",
			expected: "46",
		},
		{
			name:     "cpu core 1",
			template: "${cpu 1}",
			expected: "50",
		},
		{
			name:     "cpu core 2",
			template: "${cpu 2}",
			expected: "40",
		},
		{
			name:     "cpu frequency",
			template: "${freq}",
			expected: "3600",
		},
		{
			name:     "cpu frequency GHz",
			template: "${freq_g}",
			expected: "3.60",
		},
		{
			name:     "cpu model",
			template: "${cpu_model}",
			expected: "Intel Core i7-9700K",
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

func TestParseMemoryVariables(t *testing.T) {
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

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "mem used",
			template: "${mem}",
			expected: "8.0GiB",
		},
		{
			name:     "mem total",
			template: "${memmax}",
			expected: "16.0GiB",
		},
		{
			name:     "mem free",
			template: "${memfree}",
			expected: "4.0GiB",
		},
		{
			name:     "mem percent",
			template: "${memperc}",
			expected: "50",
		},
		{
			name:     "mem available",
			template: "${memeasyfree}",
			expected: "6.0GiB",
		},
		{
			name:     "swap used",
			template: "${swap}",
			expected: "1.0GiB",
		},
		{
			name:     "swap total",
			template: "${swapmax}",
			expected: "8.0GiB",
		},
		{
			name:     "swap percent",
			template: "${swapperc}",
			expected: "12",
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

func TestParseUptimeVariables(t *testing.T) {
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

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "uptime full",
			template: "${uptime}",
			expected: "1d 1h 1m 1s",
		},
		{
			name:     "uptime short",
			template: "${uptime_short}",
			expected: "1d 1h 1m",
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

func TestParseNetworkVariables(t *testing.T) {
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

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "download speed total",
			template: "${downspeed}",
			expected: "100.0KiB/s",
		},
		{
			name:     "upload speed total",
			template: "${upspeed}",
			expected: "50.0KiB/s",
		},
		{
			name:     "download speed eth0",
			template: "${downspeed eth0}",
			expected: "100.0KiB/s",
		},
		{
			name:     "total downloaded",
			template: "${totaldown}",
			expected: "100.0MiB",
		},
		{
			name:     "total uploaded",
			template: "${totalup}",
			expected: "50.0MiB",
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

func TestParseFilesystemVariables(t *testing.T) {
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

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "fs used root",
			template: "${fs_used /}",
			expected: "200.0GiB",
		},
		{
			name:     "fs size root",
			template: "${fs_size /}",
			expected: "500.0GiB",
		},
		{
			name:     "fs free root",
			template: "${fs_free /}",
			expected: "280.0GiB",
		},
		{
			name:     "fs used percent root",
			template: "${fs_used_perc /}",
			expected: "40",
		},
		{
			name:     "fs used home",
			template: "${fs_used /home}",
			expected: "512.0GiB",
		},
		{
			name:     "fs size home",
			template: "${fs_size /home}",
			expected: "1.0TiB",
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

func TestParseProcessVariables(t *testing.T) {
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

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "total processes",
			template: "${processes}",
			expected: "150",
		},
		{
			name:     "running processes",
			template: "${running_processes}",
			expected: "5",
		},
		{
			name:     "total threads",
			template: "${threads}",
			expected: "500",
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

func TestParseBatteryVariables(t *testing.T) {
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

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "battery percent",
			template: "${battery_percent}",
			expected: "85",
		},
		{
			name:     "battery short",
			template: "${battery_short}",
			expected: "D 85%",
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

func TestParseHwmonVariables(t *testing.T) {
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

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "hwmon default",
			template: "${hwmon}",
			expected: "55",
		},
		{
			name:     "hwmon sensor 0",
			template: "${hwmon 0}",
			expected: "55",
		},
		{
			name:     "hwmon sensor 1",
			template: "${hwmon 1}",
			expected: "65",
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

func TestParseMixerVariable(t *testing.T) {
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

	result := api.Parse("${mixer}")
	expected := "75"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestParseComplexTemplate(t *testing.T) {
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

	template := "CPU: ${cpu}% | RAM: ${mem}/${memmax} (${memperc}%) | Up: ${uptime_short}"
	result := api.Parse(template)
	expected := "CPU: 46% | RAM: 8.0GiB/16.0GiB (50%) | Up: 1d 1h 1m"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestParseUnknownVariable(t *testing.T) {
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

	// Unknown variables should be returned as-is
	result := api.Parse("${unknown_var}")
	expected := "${unknown_var}"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestParseWithNilProvider(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	api, err := NewConkyAPI(runtime, nil)
	if err != nil {
		t.Fatalf("failed to create API: %v", err)
	}

	// With nil provider, variables should be returned as-is
	result := api.Parse("${cpu}")
	expected := "${cpu}"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}

	// With nil provider, variables with args should also include args
	result = api.Parse("${cpu 1}")
	expected = "${cpu 1}"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestSetSystemDataProvider(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	api, err := NewConkyAPI(runtime, nil)
	if err != nil {
		t.Fatalf("failed to create API: %v", err)
	}

	// Initially nil, should return variable as-is
	result := api.Parse("${cpu}")
	if result != "${cpu}" {
		t.Errorf("expected ${cpu}, got %q", result)
	}

	// Set provider and check again
	provider := newMockProvider()
	api.SetSystemDataProvider(provider)

	result = api.Parse("${cpu}")
	expected := "46"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    uint64
		expected string
	}{
		{0, "0B"},
		{512, "512B"},
		{1024, "1.0KiB"},
		{1536, "1.5KiB"},
		{1024 * 1024, "1.0MiB"},
		{1024 * 1024 * 1024, "1.0GiB"},
		{1024 * 1024 * 1024 * 1024, "1.0TiB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatBytes(tt.bytes)
			if result != tt.expected {
				t.Errorf("formatBytes(%d) = %q, want %q", tt.bytes, result, tt.expected)
			}
		})
	}
}

func TestFormatSpeed(t *testing.T) {
	tests := []struct {
		speed    float64
		expected string
	}{
		{0, "0B/s"},
		{512, "512B/s"},
		{1024, "1.0KiB/s"},
		{1024 * 1024, "1.0MiB/s"},
		{1024 * 1024 * 1024, "1.0GiB/s"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatSpeed(tt.speed)
			if result != tt.expected {
				t.Errorf("formatSpeed(%f) = %q, want %q", tt.speed, result, tt.expected)
			}
		})
	}
}

func TestParseInvalidCPUCore(t *testing.T) {
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

	// Core 10 doesn't exist (only 4 cores)
	result := api.Parse("${cpu 10}")
	expected := "0"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}

	// Invalid core number (not a number)
	result = api.Parse("${cpu abc}")
	expected = "46" // Falls back to total CPU
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestParseInvalidFilesystemMount(t *testing.T) {
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

	// Non-existent mount point
	result := api.Parse("${fs_used /nonexistent}")
	expected := "0B"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestParseInvalidNetworkInterface(t *testing.T) {
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

	// Non-existent interface
	result := api.Parse("${downspeed wlan0}")
	expected := "0B"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestParseEmptyTemplate(t *testing.T) {
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

	result := api.Parse("")
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestParseNoVariables(t *testing.T) {
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

	result := api.Parse("Just plain text")
	if result != "Just plain text" {
		t.Errorf("expected 'Just plain text', got %q", result)
	}
}

func TestParseUptimeShortFormats(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	tests := []struct {
		name     string
		uptime   monitor.UptimeStats
		template string
		contains string
	}{
		{
			name:     "just minutes",
			uptime:   monitor.UptimeStats{Seconds: 300}, // 5 minutes
			template: "${uptime_short}",
			contains: "5m",
		},
		{
			name:     "hours and minutes",
			uptime:   monitor.UptimeStats{Seconds: 7260}, // 2h 1m
			template: "${uptime_short}",
			contains: "2h 1m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &mockSystemDataProvider{
				uptime: tt.uptime,
			}
			api, err := NewConkyAPI(runtime, provider)
			if err != nil {
				t.Fatalf("failed to create API: %v", err)
			}

			result := api.Parse(tt.template)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("expected result to contain %q, got %q", tt.contains, result)
			}
		})
	}
}
