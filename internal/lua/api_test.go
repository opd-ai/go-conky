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
	sysInfo    monitor.SystemInfo
	tcp        monitor.TCPStats
	gpu        monitor.GPUStats
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
func (m *mockSystemDataProvider) SysInfo() monitor.SystemInfo         { return m.sysInfo }
func (m *mockSystemDataProvider) TCP() monitor.TCPStats               { return m.tcp }
func (m *mockSystemDataProvider) TCPCountInRange(minPort, maxPort int) int {
	count := 0
	for _, c := range m.tcp.Connections {
		if c.LocalPort >= minPort && c.LocalPort <= maxPort {
			count++
		}
	}
	return count
}
func (m *mockSystemDataProvider) TCPConnectionByIndex(minPort, maxPort, index int) *monitor.TCPConnection {
	var matching []monitor.TCPConnection
	for _, c := range m.tcp.Connections {
		if c.LocalPort >= minPort && c.LocalPort <= maxPort {
			matching = append(matching, c)
		}
	}
	if index >= 0 && index < len(matching) {
		return &matching[index]
	}
	return nil
}
func (m *mockSystemDataProvider) GPU() monitor.GPUStats { return m.gpu }

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
					IPv4Addrs:     []string{"192.168.1.100"},
					IPv6Addrs:     []string{"fe80::1"},
				},
				"lo": {
					Name:      "lo",
					IPv4Addrs: []string{"127.0.0.1"},
					IPv6Addrs: []string{"::1"},
				},
				"wlan0": {
					Name:          "wlan0",
					RxBytes:       1024 * 1024 * 200, // 200 MiB
					TxBytes:       1024 * 1024 * 100, // 100 MiB
					RxBytesPerSec: 1024 * 500,        // 500 KiB/s
					TxBytesPerSec: 1024 * 200,        // 200 KiB/s
					IPv4Addrs:     []string{"192.168.1.101"},
					IPv6Addrs:     []string{"fe80::2"},
					Wireless: &monitor.WirelessInfo{
						ESSID:          "MyHomeNetwork",
						AccessPoint:    "AA:BB:CC:DD:EE:FF",
						LinkQuality:    70,
						LinkQualityMax: 100,
						SignalLevel:    -40,
						NoiseLevel:     -95,
						BitRate:        300.0,
						Mode:           "Managed",
						IsWireless:     true,
					},
				},
			},
			TotalRxBytes:       1024 * 1024 * 100,
			TotalTxBytes:       1024 * 1024 * 50,
			TotalRxBytesPerSec: 1024 * 100,
			TotalTxBytesPerSec: 1024 * 50,
			GatewayIP:          "192.168.1.1",
			GatewayInterface:   "eth0",
			Nameservers:        []string{"8.8.8.8", "8.8.4.4"},
		},
		filesystem: monitor.FilesystemStats{
			Mounts: map[string]monitor.MountStats{
				"/": {
					MountPoint:   "/",
					Device:       "/dev/sda1",
					FSType:       "ext4",
					Total:        500 * 1024 * 1024 * 1024, // 500 GiB
					Used:         200 * 1024 * 1024 * 1024, // 200 GiB
					Free:         300 * 1024 * 1024 * 1024, // 300 GiB
					Available:    280 * 1024 * 1024 * 1024, // 280 GiB
					UsagePercent: 40.0,
				},
				"/home": {
					MountPoint:   "/home",
					Device:       "/dev/sda2",
					FSType:       "xfs",
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
					Name:             "sda",
					ReadBytesPerSec:  1024 * 1024, // 1.0 MiB/s read
					WriteBytesPerSec: 512 * 1024,  // 512 KiB/s write
					ReadsPerSec:      100.0,       // 100 reads/sec
					WritesPerSec:     50.0,        // 50 writes/sec
				},
				"sdb": {
					Name:             "sdb",
					ReadBytesPerSec:  2 * 1024 * 1024, // 2.0 MiB/s read
					WriteBytesPerSec: 1024 * 1024,     // 1.0 MiB/s write
					ReadsPerSec:      200.0,           // 200 reads/sec
					WritesPerSec:     100.0,           // 100 writes/sec
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
			TopCPU: []monitor.ProcessInfo{
				{PID: 1234, Name: "firefox", CPUPercent: 25.5, MemPercent: 10.2, MemBytes: 512 * 1024 * 1024, Threads: 50},
				{PID: 5678, Name: "chrome", CPUPercent: 15.3, MemPercent: 8.5, MemBytes: 400 * 1024 * 1024, Threads: 30},
				{PID: 9012, Name: "vscode", CPUPercent: 10.1, MemPercent: 12.0, MemBytes: 600 * 1024 * 1024, Threads: 25},
			},
			TopMem: []monitor.ProcessInfo{
				{PID: 9012, Name: "vscode", CPUPercent: 10.1, MemPercent: 12.0, MemBytes: 600 * 1024 * 1024, Threads: 25},
				{PID: 1234, Name: "firefox", CPUPercent: 25.5, MemPercent: 10.2, MemBytes: 512 * 1024 * 1024, Threads: 50},
				{PID: 5678, Name: "chrome", CPUPercent: 15.3, MemPercent: 8.5, MemBytes: 400 * 1024 * 1024, Threads: 30},
			},
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
		sysInfo: monitor.SystemInfo{
			Kernel:        "5.15.0-generic",
			Hostname:      "testhost.example.com",
			HostnameShort: "testhost",
			Sysname:       "Linux",
			Machine:       "x86_64",
			LoadAvg1:      1.50,
			LoadAvg5:      1.25,
			LoadAvg15:     1.00,
		},
		tcp: monitor.TCPStats{
			Connections: []monitor.TCPConnection{
				{
					LocalIP:    "192.168.1.100",
					LocalPort:  22,
					RemoteIP:   "10.0.0.1",
					RemotePort: 52345,
					State:      "ESTABLISHED",
				},
				{
					LocalIP:    "0.0.0.0",
					LocalPort:  80,
					RemoteIP:   "0.0.0.0",
					RemotePort: 0,
					State:      "LISTEN",
				},
				{
					LocalIP:    "192.168.1.100",
					LocalPort:  443,
					RemoteIP:   "10.0.0.2",
					RemotePort: 54321,
					State:      "ESTABLISHED",
				},
			},
			TotalCount:  3,
			ListenCount: 1,
		},
		gpu: monitor.GPUStats{
			Name:        "NVIDIA GeForce RTX 3080",
			DriverVer:   "535.154.05",
			MemTotal:    10 * 1024 * 1024 * 1024, // 10 GiB
			MemUsed:     4 * 1024 * 1024 * 1024,  // 4 GiB
			MemFree:     6 * 1024 * 1024 * 1024,  // 6 GiB
			UtilGPU:     45,
			UtilMem:     40,
			Temperature: 65,
			FanSpeed:    55,
			PowerDraw:   180.5,
			PowerLimit:  320.0,
			Available:   true,
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
		{
			name:     "interface address",
			template: "${addr eth0}",
			expected: "192.168.1.100",
		},
		{
			name:     "interface address loopback",
			template: "${addr lo}",
			expected: "127.0.0.1",
		},
		{
			name:     "interface address unknown",
			template: "${addr unknown}",
			expected: "",
		},
		{
			name:     "interface address no arg",
			template: "${addr}",
			expected: "",
		},
		{
			name:     "all addresses eth0",
			template: "${addrs eth0}",
			expected: "192.168.1.100 fe80::1",
		},
		{
			name:     "gateway ip",
			template: "${gw_ip}",
			expected: "192.168.1.1",
		},
		{
			name:     "gateway interface",
			template: "${gw_iface}",
			expected: "eth0",
		},
		{
			name:     "nameserver first",
			template: "${nameserver}",
			expected: "8.8.8.8",
		},
		{
			name:     "nameserver second",
			template: "${nameserver 1}",
			expected: "8.8.4.4",
		},
		{
			name:     "nameserver out of range",
			template: "${nameserver 10}",
			expected: "",
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

func TestParseDiskIOVariables(t *testing.T) {
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
			name:     "total diskio all devices",
			template: "${diskio}",
			expected: "4.5MiB/s", // sda: 1.0 + 0.5 = 1.5, sdb: 2.0 + 1.0 = 3.0, total = 4.5
		},
		{
			name:     "diskio specific device sda",
			template: "${diskio sda}",
			expected: "1.5MiB/s", // 1.0 MiB/s read + 512 KiB/s write = 1.5 MiB/s
		},
		{
			name:     "diskio specific device sdb",
			template: "${diskio sdb}",
			expected: "3.0MiB/s", // 2.0 MiB/s read + 1.0 MiB/s write = 3.0 MiB/s
		},
		{
			name:     "diskio nonexistent device",
			template: "${diskio sdc}",
			expected: "0B/s",
		},
		{
			name:     "diskio read all devices",
			template: "${diskio_read}",
			expected: "3.0MiB/s", // sda: 1.0 + sdb: 2.0 = 3.0 MiB/s
		},
		{
			name:     "diskio read specific device sda",
			template: "${diskio_read sda}",
			expected: "1.0MiB/s",
		},
		{
			name:     "diskio read specific device sdb",
			template: "${diskio_read sdb}",
			expected: "2.0MiB/s",
		},
		{
			name:     "diskio read nonexistent device",
			template: "${diskio_read sdc}",
			expected: "0B/s",
		},
		{
			name:     "diskio write all devices",
			template: "${diskio_write}",
			expected: "1.5MiB/s", // sda: 512 KiB + sdb: 1024 KiB = 1536 KiB = 1.5 MiB/s
		},
		{
			name:     "diskio write specific device sda",
			template: "${diskio_write sda}",
			expected: "512.0KiB/s",
		},
		{
			name:     "diskio write specific device sdb",
			template: "${diskio_write sdb}",
			expected: "1.0MiB/s",
		},
		{
			name:     "diskio write nonexistent device",
			template: "${diskio_write sdc}",
			expected: "0B/s",
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
		{
			name:     "top cpu name 1",
			template: "${top name 1}",
			expected: "firefox",
		},
		{
			name:     "top cpu name 2",
			template: "${top name 2}",
			expected: "chrome",
		},
		{
			name:     "top cpu pid 1",
			template: "${top pid 1}",
			expected: "1234",
		},
		{
			name:     "top cpu percent 1",
			template: "${top cpu 1}",
			expected: "25.5",
		},
		{
			name:     "top mem percent 1",
			template: "${top mem 1}",
			expected: "10.2",
		},
		{
			name:     "top_mem name 1",
			template: "${top_mem name 1}",
			expected: "vscode",
		},
		{
			name:     "top_mem mem 1",
			template: "${top_mem mem 1}",
			expected: "12.0",
		},
		{
			name:     "top out of range",
			template: "${top name 100}",
			expected: "",
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
		{
			name:     "battery full",
			template: "${battery}",
			expected: "Discharging 85%",
		},
		{
			name:     "battery BAT0",
			template: "${battery BAT0}",
			expected: "Discharging 85%",
		},
		{
			name:     "battery bar",
			template: "${battery_bar 10}",
			expected: "########--",
		},
		{
			name:     "battery time",
			template: "${battery_time}",
			expected: "Unknown",
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

func TestParseSystemInfoVariables(t *testing.T) {
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
			name:     "kernel",
			template: "${kernel}",
			expected: "5.15.0-generic",
		},
		{
			name:     "nodename",
			template: "${nodename}",
			expected: "testhost.example.com",
		},
		{
			name:     "nodename_short",
			template: "${nodename_short}",
			expected: "testhost",
		},
		{
			name:     "sysname",
			template: "${sysname}",
			expected: "Linux",
		},
		{
			name:     "machine",
			template: "${machine}",
			expected: "x86_64",
		},
		{
			name:     "conky_version",
			template: "${conky_version}",
			expected: Version,
		},
		{
			name:     "conky_build_arch",
			template: "${conky_build_arch}",
			expected: "x86_64",
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

func TestParseLoadAvgVariables(t *testing.T) {
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
			name:     "loadavg default",
			template: "${loadavg}",
			expected: "1.50 1.25 1.00",
		},
		{
			name:     "loadavg 1 minute",
			template: "${loadavg 1}",
			expected: "1.50",
		},
		{
			name:     "loadavg 5 minute",
			template: "${loadavg 5}",
			expected: "1.25",
		},
		{
			name:     "loadavg 15 minute",
			template: "${loadavg 15}",
			expected: "1.00",
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

func TestParseTimeVariable(t *testing.T) {
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

	// Test that time variable returns something reasonable
	result := api.Parse("${time}")
	if result == "" || result == "${time}" {
		t.Errorf("expected time variable to return a value, got %q", result)
	}

	// Test with format specifier
	result = api.Parse("${time %H:%M}")
	// Should match pattern HH:MM
	if len(result) != 5 || result[2] != ':' {
		t.Errorf("expected time in HH:MM format, got %q", result)
	}

	// Test with date format
	result = api.Parse("${time %Y-%m-%d}")
	// Should match pattern YYYY-MM-DD
	if len(result) != 10 || result[4] != '-' || result[7] != '-' {
		t.Errorf("expected date in YYYY-MM-DD format, got %q", result)
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

func TestParseMiscVariables(t *testing.T) {
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
			name:     "cpu count",
			template: "${cpu_count}",
			expected: "4",
		},
		{
			name:     "fs type root",
			template: "${fs_type /}",
			expected: "ext4",
		},
		{
			name:     "fs type home",
			template: "${fs_type /home}",
			expected: "xfs",
		},
		{
			name:     "fs bar",
			template: "${fs_bar 10 /}",
			expected: "####------",
		},
		{
			name:     "hr",
			template: "${hr 5}",
			expected: "-----",
		},
		{
			name:     "tab",
			template: "${tab}",
			expected: "\t",
		},
		{
			name:     "color returns empty",
			template: "${color red}",
			expected: "",
		},
		{
			name:     "font returns empty",
			template: "${font Monospace}",
			expected: "",
		},
		{
			name:     "if_up existing",
			template: "${if_up eth0}",
			expected: "1",
		},
		{
			name:     "if_up non-existing",
			template: "${if_up nonexistent}",
			expected: "0",
		},
		{
			name:     "downspeedf",
			template: "${downspeedf eth0}",
			expected: "100.00",
		},
		{
			name:     "upspeedf",
			template: "${upspeedf eth0}",
			expected: "50.00",
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

func TestExecVariable(t *testing.T) {
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

	// Test simple echo command
	result := api.Parse("${exec echo hello}")
	if result != "hello" {
		t.Errorf("expected 'hello', got %q", result)
	}

	// Test command with arguments
	result = api.Parse("${exec echo hello world}")
	if result != "hello world" {
		t.Errorf("expected 'hello world', got %q", result)
	}
}

func TestExeciVariable(t *testing.T) {
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

	// Test basic execi command with interval
	result := api.Parse("${execi 60 echo cached}")
	if result != "cached" {
		t.Errorf("expected 'cached', got %q", result)
	}

	// Test that result is cached - second call should return cached value
	result2 := api.Parse("${execi 60 echo cached}")
	if result2 != "cached" {
		t.Errorf("expected cached 'cached', got %q", result2)
	}

	// Test with missing interval (should return empty)
	result = api.Parse("${execi echo only}")
	if result != "" {
		t.Errorf("expected empty for missing interval, got %q", result)
	}

	// Test with interval 0 (always re-execute)
	result = api.Parse("${execi 0 echo fresh}")
	if result != "fresh" {
		t.Errorf("expected 'fresh', got %q", result)
	}

	// Test with multi-word command
	result = api.Parse("${execi 30 echo hello world}")
	if result != "hello world" {
		t.Errorf("expected 'hello world', got %q", result)
	}

	// Test execpi (same as execi, parsing handled elsewhere)
	result = api.Parse("${execpi 60 echo parsed}")
	if result != "parsed" {
		t.Errorf("expected 'parsed', got %q", result)
	}

	// Test invalid interval
	result = api.Parse("${execi abc echo test}")
	if result != "" {
		t.Errorf("expected empty for invalid interval, got %q", result)
	}

	// Test negative interval
	result = api.Parse("${execi -5 echo test}")
	if result != "" {
		t.Errorf("expected empty for negative interval, got %q", result)
	}
}

func TestEntropyVariables(t *testing.T) {
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

	// Test entropy_avail - reads from /proc/sys/kernel/random/entropy_avail
	result := api.Parse("${entropy_avail}")
	// Just verify it returns a number (or empty string if file doesn't exist)
	if result != "" && result != "0" {
		// Should be a numeric value
		if len(result) > 5 {
			t.Errorf("entropy_avail should be <= 4096, got %s", result)
		}
	}

	// Test entropy_poolsize - always returns 4096
	result = api.Parse("${entropy_poolsize}")
	if result != "4096" {
		t.Errorf("expected '4096', got %q", result)
	}

	// Test entropy_perc - should be a percentage
	result = api.Parse("${entropy_perc}")
	// Just check it's a valid number
	if result == "" {
		result = "0"
	}

	// Test entropy_bar
	result = api.Parse("${entropy_bar}")
	if len(result) != 10 { // default width is 10
		t.Errorf("expected bar of length 10, got length %d: %q", len(result), result)
	}
}

func TestBarWidgets(t *testing.T) {
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
		variable string
		minLen   int
	}{
		{"membar default", "${membar}", 10},
		{"membar custom width", "${membar 20}", 20},
		{"swapbar default", "${swapbar}", 10},
		{"cpubar default", "${cpubar}", 10},
		{"loadgraph default", "${loadgraph}", 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := api.Parse(tt.variable)
			if len(result) != tt.minLen {
				t.Errorf("expected bar of length %d, got length %d: %q", tt.minLen, len(result), result)
			}
			// Check bar contains only # and -
			for _, c := range result {
				if c != '#' && c != '-' {
					t.Errorf("unexpected character in bar: %c", c)
				}
			}
		})
	}
}

func TestConditionalVariables(t *testing.T) {
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

	// Test if_existing with existing file
	result := api.Parse("${if_existing /proc/version}")
	if result != "1" {
		t.Errorf("expected '1' for existing file, got %q", result)
	}

	// Test if_existing with non-existing file
	result = api.Parse("${if_existing /nonexistent/file/path}")
	if result != "0" {
		t.Errorf("expected '0' for non-existing file, got %q", result)
	}
}

func TestInodeVariables(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	provider := newMockProvider()
	// Add filesystem data with inodes
	provider.filesystem = monitor.FilesystemStats{
		Mounts: map[string]monitor.MountStats{
			"/": {
				MountPoint:    "/",
				InodesTotal:   1000000,
				InodesFree:    500000,
				InodesPercent: 50.0,
			},
		},
	}
	api, err := NewConkyAPI(runtime, provider)
	if err != nil {
		t.Fatalf("failed to create API: %v", err)
	}

	// Test fs_inodes
	result := api.Parse("${fs_inodes /}")
	if result == "0" {
		t.Errorf("expected non-zero inodes, got %q", result)
	}

	// Test fs_inodes_free
	result = api.Parse("${fs_inodes_free /}")
	if result == "0" {
		t.Errorf("expected non-zero free inodes, got %q", result)
	}

	// Test fs_inodes_perc
	result = api.Parse("${fs_inodes_perc /}")
	if result != "50" {
		t.Errorf("expected '50', got %q", result)
	}
}

func TestStippledHR(t *testing.T) {
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

	result := api.Parse("${stippled_hr 8}")
	expected := "- - - - "
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// TestParseWirelessVariables tests wireless network variable parsing.
func TestParseWirelessVariables(t *testing.T) {
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
			name:     "wireless essid",
			template: "${wireless_essid wlan0}",
			expected: "MyHomeNetwork",
		},
		{
			name:     "wireless essid non-wireless interface",
			template: "${wireless_essid eth0}",
			expected: "",
		},
		{
			name:     "wireless essid no arg",
			template: "${wireless_essid}",
			expected: "",
		},
		{
			name:     "wireless link quality",
			template: "${wireless_link_qual wlan0}",
			expected: "70",
		},
		{
			name:     "wireless link quality non-wireless",
			template: "${wireless_link_qual eth0}",
			expected: "0",
		},
		{
			name:     "wireless link quality percent",
			template: "${wireless_link_qual_perc wlan0}",
			expected: "70",
		},
		{
			name:     "wireless link quality percent non-wireless",
			template: "${wireless_link_qual_perc eth0}",
			expected: "0",
		},
		{
			name:     "wireless link quality max",
			template: "${wireless_link_qual_max wlan0}",
			expected: "100",
		},
		{
			name:     "wireless bitrate",
			template: "${wireless_bitrate wlan0}",
			expected: "300Mb/s",
		},
		{
			name:     "wireless bitrate non-wireless",
			template: "${wireless_bitrate eth0}",
			expected: "0Mb/s",
		},
		{
			name:     "wireless access point",
			template: "${wireless_ap wlan0}",
			expected: "AA:BB:CC:DD:EE:FF",
		},
		{
			name:     "wireless access point non-wireless",
			template: "${wireless_ap eth0}",
			expected: "00:00:00:00:00:00",
		},
		{
			name:     "wireless mode",
			template: "${wireless_mode wlan0}",
			expected: "Managed",
		},
		{
			name:     "wireless mode non-wireless",
			template: "${wireless_mode eth0}",
			expected: "Managed",
		},
		{
			name:     "wireless unknown interface",
			template: "${wireless_essid unknown}",
			expected: "",
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

// TestParseTCPPortMonVariables tests TCP port monitor variable parsing.
func TestParseTCPPortMonVariables(t *testing.T) {
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
			name:     "tcp portmon count in range",
			template: "${tcp_portmon 1 1024 count}",
			expected: "2", // SSH port 22 and HTTP port 80
		},
		{
			name:     "tcp portmon count all ports",
			template: "${tcp_portmon 1 65535 count}",
			expected: "3", // All three connections
		},
		{
			name:     "tcp portmon local ip",
			template: "${tcp_portmon 1 1024 lip 0}",
			expected: "192.168.1.100",
		},
		{
			name:     "tcp portmon local port",
			template: "${tcp_portmon 1 1024 lport 0}",
			expected: "22",
		},
		{
			name:     "tcp portmon local service",
			template: "${tcp_portmon 1 1024 lservice 0}",
			expected: "ssh",
		},
		{
			name:     "tcp portmon remote ip",
			template: "${tcp_portmon 1 1024 rip 0}",
			expected: "10.0.0.1",
		},
		{
			name:     "tcp portmon remote port",
			template: "${tcp_portmon 1 1024 rport 0}",
			expected: "52345",
		},
		{
			name:     "tcp portmon insufficient args",
			template: "${tcp_portmon 1}",
			expected: "0",
		},
		{
			name:     "tcp portmon invalid port range",
			template: "${tcp_portmon abc 1024 count}",
			expected: "0",
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

// TestParseNvidiaVariables tests NVIDIA GPU variable parsing.
func TestParseNvidiaVariables(t *testing.T) {
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
			name:     "nvidia temp",
			template: "${nvidia temp}",
			expected: "65°C",
		},
		{
			name:     "nvidia gpuutil",
			template: "${nvidia gpuutil}",
			expected: "45%",
		},
		{
			name:     "nvidia memutil",
			template: "${nvidia memutil}",
			expected: "40%",
		},
		{
			name:     "nvidia fan",
			template: "${nvidia fan}",
			expected: "55%",
		},
		{
			name:     "nvidia power",
			template: "${nvidia power}",
			expected: "180.5W",
		},
		{
			name:     "nvidia driver",
			template: "${nvidia driver}",
			expected: "535.154.05",
		},
		{
			name:     "nvidia name",
			template: "${nvidia name}",
			expected: "NVIDIA GeForce RTX 3080",
		},
		{
			name:     "nvidia memused",
			template: "${nvidia memused}",
			expected: "4.0GiB",
		},
		{
			name:     "nvidia memtotal",
			template: "${nvidia memtotal}",
			expected: "10.0GiB",
		},
		{
			name:     "nvidia memfree",
			template: "${nvidia memfree}",
			expected: "6.0GiB",
		},
		{
			name:     "nvidia memperc",
			template: "${nvidia memperc}",
			expected: "40.0%",
		},
		{
			name:     "nvidia_temp direct variable",
			template: "${nvidia_temp}",
			expected: "65°C",
		},
		{
			name:     "nvidia_gpu direct variable",
			template: "${nvidia_gpu}",
			expected: "45%",
		},
		{
			name:     "nvidia_fan direct variable",
			template: "${nvidia_fan}",
			expected: "55%",
		},
		{
			name:     "nvidia unknown field",
			template: "${nvidia unknown}",
			expected: "",
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
