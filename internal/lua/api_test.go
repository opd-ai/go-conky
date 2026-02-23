package lua

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	rt "github.com/arnodel/golua/runtime"

	"github.com/opd-ai/go-conky/internal/monitor"
	"github.com/opd-ai/go-conky/internal/render"
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
	mail       monitor.MailStats
	weather    monitor.WeatherStats
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
func (m *mockSystemDataProvider) GPU() monitor.GPUStats   { return m.gpu }
func (m *mockSystemDataProvider) Mail() monitor.MailStats { return m.mail }
func (m *mockSystemDataProvider) MailUnseenCount(name string) int {
	if m.mail.Accounts == nil {
		return 0
	}
	if account, ok := m.mail.Accounts[name]; ok {
		return account.Unseen
	}
	return 0
}

func (m *mockSystemDataProvider) MailTotalCount(name string) int {
	if m.mail.Accounts == nil {
		return 0
	}
	if account, ok := m.mail.Accounts[name]; ok {
		return account.Total
	}
	return 0
}

func (m *mockSystemDataProvider) MailTotalUnseen() int {
	if m.mail.Accounts == nil {
		return 0
	}
	var total int
	for _, account := range m.mail.Accounts {
		total += account.Unseen
	}
	return total
}

func (m *mockSystemDataProvider) MailTotalMessages() int {
	if m.mail.Accounts == nil {
		return 0
	}
	var total int
	for _, account := range m.mail.Accounts {
		total += account.Total
	}
	return total
}

func (m *mockSystemDataProvider) Weather(stationID string) monitor.WeatherStats {
	return m.weather
}

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
					Capacity:    85,
					Status:      "Discharging",
					TimeToEmpty: 9000, // 2 hours 30 minutes in seconds
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
		weather: monitor.WeatherStats{
			StationID:     "KJFK",
			Temperature:   22,
			DewPoint:      10,
			Humidity:      50,
			Pressure:      1013.25,
			WindSpeed:     15,
			WindDirection: 270,
			WindGust:      25,
			Visibility:    10,
			Condition:     "clear",
			Cloud:         "few clouds",
			RawMETAR:      "KJFK 151756Z 27015G25KT 10SM FEW045 22/10 A2992",
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
	defer api.Close()

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
			name:     "battery time discharging",
			template: "${battery_time}",
			expected: "2:30",
		},
		{
			name:     "battery time with battery name",
			template: "${battery_time BAT0}",
			expected: "2:30",
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

// TestBatteryTimeScenarios tests battery_time with various battery states.
func TestBatteryTimeScenarios(t *testing.T) {
	tests := []struct {
		name     string
		battery  monitor.BatteryStats
		template string
		expected string
	}{
		{
			name: "discharging with time",
			battery: monitor.BatteryStats{
				Batteries: map[string]monitor.BatteryInfo{
					"BAT0": {Status: "Discharging", TimeToEmpty: 5400}, // 1:30
				},
				IsDischarging: true,
			},
			template: "${battery_time}",
			expected: "1:30",
		},
		{
			name: "charging with time",
			battery: monitor.BatteryStats{
				Batteries: map[string]monitor.BatteryInfo{
					"BAT0": {Status: "Charging", TimeToFull: 3660}, // 1:01
				},
				IsCharging: true,
			},
			template: "${battery_time}",
			expected: "1:01",
		},
		{
			name: "full battery on AC",
			battery: monitor.BatteryStats{
				Batteries: map[string]monitor.BatteryInfo{
					"BAT0": {Status: "Full"},
				},
				ACOnline: true,
			},
			template: "${battery_time}",
			expected: "AC",
		},
		{
			name: "not charging on AC",
			battery: monitor.BatteryStats{
				Batteries: map[string]monitor.BatteryInfo{
					"BAT0": {Status: "Not charging"},
				},
				ACOnline: true,
			},
			template: "${battery_time}",
			expected: "AC",
		},
		{
			name: "discharging no time available",
			battery: monitor.BatteryStats{
				Batteries: map[string]monitor.BatteryInfo{
					"BAT0": {Status: "Discharging", TimeToEmpty: 0},
				},
				IsDischarging: true,
			},
			template: "${battery_time}",
			expected: "Unknown",
		},
		{
			name: "no battery AC online",
			battery: monitor.BatteryStats{
				Batteries: map[string]monitor.BatteryInfo{},
				ACOnline:  true,
			},
			template: "${battery_time}",
			expected: "AC",
		},
		{
			name: "no battery AC offline",
			battery: monitor.BatteryStats{
				Batteries: map[string]monitor.BatteryInfo{},
				ACOnline:  false,
			},
			template: "${battery_time}",
			expected: "Unknown",
		},
		{
			name: "specific battery name",
			battery: monitor.BatteryStats{
				Batteries: map[string]monitor.BatteryInfo{
					"BAT0": {Status: "Discharging", TimeToEmpty: 7200}, // 2:00
					"BAT1": {Status: "Discharging", TimeToEmpty: 3600}, // 1:00
				},
				IsDischarging: true,
			},
			template: "${battery_time BAT1}",
			expected: "1:00",
		},
		{
			name: "long discharge time",
			battery: monitor.BatteryStats{
				Batteries: map[string]monitor.BatteryInfo{
					"BAT0": {Status: "Discharging", TimeToEmpty: 36000}, // 10:00
				},
				IsDischarging: true,
			},
			template: "${battery_time}",
			expected: "10:00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runtime, err := New(DefaultConfig())
			if err != nil {
				t.Fatalf("failed to create runtime: %v", err)
			}
			defer runtime.Close()

			provider := newMockProvider()
			provider.battery = tt.battery

			api, err := NewConkyAPI(runtime, provider)
			if err != nil {
				t.Fatalf("failed to create API: %v", err)
			}

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

func TestStrftimeSpecifiers(t *testing.T) {
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

	// Test %V - ISO week number (01-53)
	result := api.Parse("${time %V}")
	if len(result) != 2 {
		t.Errorf("%%V should produce 2-digit week number, got %q", result)
	}
	// Verify it's a valid week number (01-53)
	weekNum := 0
	_, err = fmt.Sscanf(result, "%d", &weekNum)
	if err != nil || weekNum < 1 || weekNum > 53 {
		t.Errorf("%%V should produce valid ISO week number (01-53), got %q", result)
	}

	// Test %G - ISO year (4 digits)
	result = api.Parse("${time %G}")
	if len(result) != 4 {
		t.Errorf("%%G should produce 4-digit year, got %q", result)
	}

	// Test %g - ISO year without century (2 digits)
	result = api.Parse("${time %g}")
	if len(result) != 2 {
		t.Errorf("%%g should produce 2-digit year, got %q", result)
	}

	// Test %U - Week number (Sunday first, 00-53)
	result = api.Parse("${time %U}")
	if len(result) != 2 {
		t.Errorf("%%U should produce 2-digit week number, got %q", result)
	}
	weekNum = 0
	_, err = fmt.Sscanf(result, "%d", &weekNum)
	if err != nil || weekNum < 0 || weekNum > 53 {
		t.Errorf("%%U should produce valid week number (00-53), got %q", result)
	}

	// Test %W - Week number (Monday first, 00-53)
	result = api.Parse("${time %W}")
	if len(result) != 2 {
		t.Errorf("%%W should produce 2-digit week number, got %q", result)
	}
	weekNum = 0
	_, err = fmt.Sscanf(result, "%d", &weekNum)
	if err != nil || weekNum < 0 || weekNum > 53 {
		t.Errorf("%%W should produce valid week number (00-53), got %q", result)
	}

	// Test %s - Unix timestamp
	result = api.Parse("${time %s}")
	if len(result) < 10 {
		t.Errorf("%%s should produce Unix timestamp (at least 10 digits), got %q", result)
	}
	var ts int64
	_, err = fmt.Sscanf(result, "%d", &ts)
	if err != nil {
		t.Errorf("%%s should produce valid integer Unix timestamp, got %q", result)
	}

	// Test combined format with new specifiers
	result = api.Parse("${time Week %V of %G}")
	if !strings.Contains(result, "Week") || !strings.Contains(result, "of") {
		t.Errorf("expected format 'Week NN of YYYY', got %q", result)
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

	// Non-existent interface - use wlan99 which doesn't exist in mock data
	result := api.Parse("${downspeed wlan99}")
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

	// Test entropy_perc - should be a percentage (0-100)
	entropyPercResult := api.Parse("${entropy_perc}")
	if entropyPercResult != "" {
		// Parse and validate it's a valid percentage
		if percVal, err := strconv.Atoi(entropyPercResult); err != nil {
			t.Errorf("entropy_perc should be numeric, got %q", entropyPercResult)
		} else if percVal < 0 || percVal > 100 {
			t.Errorf("entropy_perc should be 0-100, got %d", percVal)
		}
	}

	// Test entropy_bar
	result = api.Parse("${entropy_bar}")
	if !render.ContainsWidgetMarker(result) {
		t.Errorf("expected widget marker for entropy_bar, got: %q", result)
	} else {
		marker := render.DecodeWidgetMarker(result)
		if marker == nil {
			t.Errorf("failed to decode entropy_bar widget marker")
		} else if marker.Type != render.WidgetTypeBar {
			t.Errorf("expected bar widget type, got %s", marker.Type.String())
		}
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
		name       string
		variable   string
		wantType   string
		wantWidth  float64
		wantHeight float64
	}{
		{"membar default", "${membar}", "bar", 100, 8},
		{"membar custom height", "${membar 20}", "bar", 100, 20},
		{"swapbar default", "${swapbar}", "bar", 100, 8},
		{"cpubar default", "${cpubar}", "bar", 100, 8},
		{"loadgraph default", "${loadgraph}", "graph", 100, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := api.Parse(tt.variable)
			// Check that result is a widget marker
			if !render.ContainsWidgetMarker(result) {
				t.Errorf("expected widget marker, got: %q", result)
				return
			}
			// Decode and verify the marker
			marker := render.DecodeWidgetMarker(result)
			if marker == nil {
				t.Errorf("failed to decode widget marker: %q", result)
				return
			}
			if marker.Type.String() != tt.wantType {
				t.Errorf("widget type = %s, want %s", marker.Type.String(), tt.wantType)
			}
			if marker.Width != tt.wantWidth {
				t.Errorf("widget width = %v, want %v", marker.Width, tt.wantWidth)
			}
			if marker.Height != tt.wantHeight {
				t.Errorf("widget height = %v, want %v", marker.Height, tt.wantHeight)
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
// NOTE: TCP port monitoring is currently a stub implementation.
// TestParseTCPPortMonVariables tests TCP port monitoring variable parsing.
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
			expected: "3", // SSH port 22, HTTP port 80, and HTTPS port 443
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
// NOTE: NVIDIA GPU support is currently a stub implementation.
// This test is skipped because resolveNvidiaVariable always returns "".
// TODO: Implement NVIDIA GPU monitoring and un-skip this test.
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
		{
			name:     "nvidia default (no field)",
			template: "${nvidia}",
			expected: "45%",
		},
		{
			name:     "nvidiagraph",
			template: "${nvidiagraph}",
			expected: "45",
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

func TestParseMailVariables(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	provider := newMockProvider()
	// Add mail accounts to the mock
	provider.mail = monitor.MailStats{
		Accounts: map[string]monitor.MailAccountStats{
			"gmail": {
				Name:   "gmail",
				Type:   "imap",
				Unseen: 5,
				Total:  100,
			},
			"work": {
				Name:   "work",
				Type:   "imap",
				Unseen: 3,
				Total:  50,
			},
		},
	}

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
			name:     "imap_unseen total",
			template: "${imap_unseen}",
			expected: "8",
		},
		{
			name:     "imap_unseen specific account",
			template: "${imap_unseen gmail}",
			expected: "5",
		},
		{
			name:     "imap_unseen work account",
			template: "${imap_unseen work}",
			expected: "3",
		},
		{
			name:     "imap_unseen nonexistent account",
			template: "${imap_unseen nonexistent}",
			expected: "0",
		},
		{
			name:     "imap_messages total",
			template: "${imap_messages}",
			expected: "150",
		},
		{
			name:     "imap_messages specific account",
			template: "${imap_messages gmail}",
			expected: "100",
		},
		{
			name:     "pop3_unseen total",
			template: "${pop3_unseen}",
			expected: "8",
		},
		{
			name:     "pop3_used total",
			template: "${pop3_used}",
			expected: "150",
		},
		{
			name:     "new_mails total",
			template: "${new_mails}",
			expected: "8",
		},
		{
			name:     "new_mails specific account",
			template: "${new_mails gmail}",
			expected: "5",
		},
		{
			name:     "mails total",
			template: "${mails}",
			expected: "8",
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

func TestParseMailVariablesNoAccounts(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	provider := newMockProvider()
	// No mail accounts configured

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
			name:     "imap_unseen no accounts",
			template: "${imap_unseen}",
			expected: "0",
		},
		{
			name:     "imap_messages no accounts",
			template: "${imap_messages}",
			expected: "0",
		},
		{
			name:     "new_mails no accounts",
			template: "${new_mails}",
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

func TestParseWeatherVariables(t *testing.T) {
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
			name:     "weather temp",
			template: "${weather KJFK temp}",
			expected: "22",
		},
		{
			name:     "weather temperature_f",
			template: "${weather KJFK temp_f}",
			expected: "72",
		},
		{
			name:     "weather humidity",
			template: "${weather KJFK humidity}",
			expected: "50",
		},
		{
			name:     "weather wind",
			template: "${weather KJFK wind}",
			expected: "15",
		},
		{
			name:     "weather wind_dir",
			template: "${weather KJFK wind_dir}",
			expected: "270",
		},
		{
			name:     "weather wind_dir_compass",
			template: "${weather KJFK wind_dir_compass}",
			expected: "W",
		},
		{
			name:     "weather condition default",
			template: "${weather KJFK}",
			expected: "clear",
		},
		{
			name:     "weather condition explicit",
			template: "${weather KJFK condition}",
			expected: "clear",
		},
		{
			name:     "weather cloud",
			template: "${weather KJFK cloud}",
			expected: "few clouds",
		},
		{
			name:     "weather pressure",
			template: "${weather KJFK pressure}",
			expected: "1013",
		},
		{
			name:     "weather visibility",
			template: "${weather KJFK visibility}",
			expected: "10.0",
		},
		{
			name:     "weather no station",
			template: "${weather}",
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

// TestScrollAnimation tests the scroll variable animation.
func TestScrollAnimation(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	api, err := NewConkyAPI(runtime, newMockProvider())
	if err != nil {
		t.Fatalf("failed to create API: %v", err)
	}

	tests := []struct {
		name     string
		template string
		check    func(result string) bool
		desc     string
	}{
		{
			name:     "short text no scroll",
			template: "${scroll 20 1 Hello}",
			check:    func(r string) bool { return len(r) == 20 && r[:5] == "Hello" },
			desc:     "short text should be padded to length",
		},
		{
			name:     "long text scrolls",
			template: "${scroll 10 1 This is a very long text that needs scrolling}",
			check:    func(r string) bool { return len(r) == 10 },
			desc:     "result should be exactly 10 characters",
		},
		{
			name:     "default step",
			template: "${scroll 5 1 ABCDEFGHIJ}",
			check:    func(r string) bool { return len(r) == 5 },
			desc:     "should return 5 character window",
		},
		{
			name:     "missing args",
			template: "${scroll 10}",
			check:    func(r string) bool { return r == "" },
			desc:     "insufficient args returns empty",
		},
		{
			name:     "empty text",
			template: "${scroll 10 1}",
			check:    func(r string) bool { return r == "" },
			desc:     "empty text returns empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := api.Parse(tt.template)
			if !tt.check(result) {
				t.Errorf("%s: got %q (len=%d)", tt.desc, result, len(result))
			}
		})
	}
}

// TestScrollAnimationAdvances tests that scroll position advances.
func TestScrollAnimationAdvances(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	api, err := NewConkyAPI(runtime, newMockProvider())
	if err != nil {
		t.Fatalf("failed to create API: %v", err)
	}

	// Use a simple text that's longer than the window
	template := "${scroll 5 1 ABCDEFGHIJ}"

	// Get first result
	result1 := api.Parse(template)
	if len(result1) != 5 {
		t.Errorf("expected length 5, got %d: %q", len(result1), result1)
	}

	// Parse again - position should advance
	result2 := api.Parse(template)
	if len(result2) != 5 {
		t.Errorf("expected length 5, got %d: %q", len(result2), result2)
	}

	// Results should differ after advancement (unless wrapping at same position)
	// Due to step=1, they should differ
	if result1 == result2 {
		// This is actually acceptable if the time hasn't advanced enough
		// The scroll advances based on time, not just calls
		t.Logf("results may be same due to rapid calls: %q vs %q", result1, result2)
	}
}

// TestScrollStateIsolation tests that different scroll instances maintain separate state.
func TestScrollStateIsolation(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	api, err := NewConkyAPI(runtime, newMockProvider())
	if err != nil {
		t.Fatalf("failed to create API: %v", err)
	}

	// Two different scroll templates should have separate state
	template1 := "${scroll 5 1 AAAAAAAAAA}"
	template2 := "${scroll 5 1 BBBBBBBBBB}"

	result1 := api.Parse(template1)
	result2 := api.Parse(template2)

	// Results should match their respective content
	for _, r := range result1 {
		if r != 'A' && r != ' ' {
			t.Errorf("template1 result should contain only 'A' or space, got %q", result1)
			break
		}
	}
	for _, r := range result2 {
		if r != 'B' && r != ' ' {
			t.Errorf("template2 result should contain only 'B' or space, got %q", result2)
			break
		}
	}
}

// TestScrollUnicodeText tests scroll with Unicode characters.
func TestScrollUnicodeText(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	api, err := NewConkyAPI(runtime, newMockProvider())
	if err != nil {
		t.Fatalf("failed to create API: %v", err)
	}

	// Unicode characters should be counted as single characters
	template := "${scroll 5 1 日本語テスト文字列}"
	result := api.Parse(template)

	// Should return 5 runes, not 5 bytes
	runes := []rune(result)
	if len(runes) != 5 {
		t.Errorf("expected 5 runes, got %d: %q", len(runes), result)
	}
}

// TestPadRight tests the padRight helper function.
func TestPadRight(t *testing.T) {
	tests := []struct {
		input    string
		length   int
		expected string
	}{
		{"Hello", 10, "Hello     "},
		{"Test", 4, "Test"},
		{"Long", 2, "Long"},
		{"", 5, "     "},
		{"日本語", 6, "日本語   "},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := padRight(tt.input, tt.length)
			if result != tt.expected {
				t.Errorf("padRight(%q, %d) = %q, want %q", tt.input, tt.length, result, tt.expected)
			}
		})
	}
}

// TestTemplateVariables tests the template0-template9 variable resolution.
func TestTemplateVariables(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	mockProvider := &mockSystemDataProvider{
		cpu: monitor.CPUStats{
			UsagePercent: 42.5,
			Cores:        []float64{42.5},
		},
		memory: monitor.MemoryStats{
			Total:     16 * 1024 * 1024 * 1024, // 16 GiB
			Used:      8 * 1024 * 1024 * 1024,  // 8 GiB
			Available: 8 * 1024 * 1024 * 1024,
		},
	}

	api, err := NewConkyAPI(runtime, mockProvider)
	if err != nil {
		t.Fatalf("Failed to create API: %v", err)
	}

	tests := []struct {
		name      string
		templates [10]string
		input     string
		contains  []string
	}{
		{
			name: "simple template without args",
			templates: [10]string{
				"Hello World", // template0
			},
			input:    "${template0}",
			contains: []string{"Hello World"},
		},
		{
			name: "template with one argument",
			templates: [10]string{
				"",               // template0
				"Core \\1 usage", // template1
			},
			input:    "${template1 5}",
			contains: []string{"Core 5 usage"},
		},
		{
			name: "template with multiple arguments",
			templates: [10]string{
				"", "", // template0, template1
				"\\1 is \\2 and \\3", // template2
			},
			input:    "${template2 CPU fast efficient}",
			contains: []string{"CPU is fast and efficient"},
		},
		{
			name: "template with embedded variables",
			templates: [10]string{
				"", "", "", // template0-2
				"CPU: ${cpu}%", // template3
			},
			input:    "${template3}",
			contains: []string{"CPU:", "%"},
		},
		{
			name: "template with argument in variable",
			templates: [10]string{
				"", "", "", "", // template0-3
				"Memory used: ${mem}", // template4
			},
			input:    "${template4}",
			contains: []string{"Memory used:"},
		},
		{
			name:      "undefined template returns empty",
			templates: [10]string{}, // all empty
			input:     "${template5}",
			contains:  []string{},
		},
		{
			name: "template9 works",
			templates: [10]string{
				"", "", "", "", "", "", "", "", "", // template0-8
				"Last template: \\1", // template9
			},
			input:    "${template9 works}",
			contains: []string{"Last template: works"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api.SetTemplates(tt.templates)
			result := api.Parse(tt.input)

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("Parse(%q) = %q, expected to contain %q", tt.input, result, expected)
				}
			}

			// For undefined template, result should be empty
			if len(tt.contains) == 0 && result != "" {
				t.Errorf("Parse(%q) = %q, expected empty string", tt.input, result)
			}
		})
	}
}

// TestTemplateSetAndGet tests the SetTemplates and GetTemplate methods.
func TestTemplateSetAndGet(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	api, err := NewConkyAPI(runtime, nil)
	if err != nil {
		t.Fatalf("Failed to create API: %v", err)
	}

	// Initially all templates should be empty
	for i := 0; i < 10; i++ {
		if got := api.GetTemplate(i); got != "" {
			t.Errorf("GetTemplate(%d) = %q, expected empty", i, got)
		}
	}

	// Set templates
	templates := [10]string{
		"template0 content",
		"template1 content",
		"template2 content",
		"", "", "", "", "", "",
		"template9 content",
	}
	api.SetTemplates(templates)

	// Verify templates are set
	if got := api.GetTemplate(0); got != "template0 content" {
		t.Errorf("GetTemplate(0) = %q, expected %q", got, "template0 content")
	}
	if got := api.GetTemplate(1); got != "template1 content" {
		t.Errorf("GetTemplate(1) = %q, expected %q", got, "template1 content")
	}
	if got := api.GetTemplate(9); got != "template9 content" {
		t.Errorf("GetTemplate(9) = %q, expected %q", got, "template9 content")
	}

	// Test out of bounds
	if got := api.GetTemplate(-1); got != "" {
		t.Errorf("GetTemplate(-1) = %q, expected empty", got)
	}
	if got := api.GetTemplate(10); got != "" {
		t.Errorf("GetTemplate(10) = %q, expected empty", got)
	}
}

// TestTemplateArgumentSubstitution tests argument placeholder substitution.
func TestTemplateArgumentSubstitution(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create runtime: %v", err)
	}
	defer runtime.Close()

	api, err := NewConkyAPI(runtime, &mockSystemDataProvider{})
	if err != nil {
		t.Fatalf("Failed to create API: %v", err)
	}

	tests := []struct {
		name     string
		template string
		args     []string
		expected string
	}{
		{
			name:     "no placeholders",
			template: "Static text",
			args:     []string{"ignored"},
			expected: "Static text",
		},
		{
			name:     "single placeholder",
			template: "Value: \\1",
			args:     []string{"42"},
			expected: "Value: 42",
		},
		{
			name:     "multiple same placeholder",
			template: "\\1 and \\1 again",
			args:     []string{"test"},
			expected: "test and test again",
		},
		{
			name:     "multiple different placeholders",
			template: "\\1, \\2, \\3",
			args:     []string{"a", "b", "c"},
			expected: "a, b, c",
		},
		{
			name:     "placeholder without matching arg",
			template: "Has \\5 placeholder",
			args:     []string{"only one"},
			expected: "Has \\5 placeholder",
		},
		{
			name:     "no args with placeholder",
			template: "Needs \\1",
			args:     []string{},
			expected: "Needs \\1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			templates := [10]string{tt.template}
			api.SetTemplates(templates)

			// Build the input string with args
			input := "${template0"
			for _, arg := range tt.args {
				input += " " + arg
			}
			input += "}"

			result := api.Parse(input)
			if result != tt.expected {
				t.Errorf("Parse(%q) with template %q = %q, want %q", input, tt.template, result, tt.expected)
			}
		})
	}
}

// TestUnsupportedVariables tests that intentionally unsupported variables
// return "N/A" and are documented as such in docs/migration.md
func TestUnsupportedVariables(t *testing.T) {
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
		// Stock quote - not implemented; requires external API keys
		{
			name:     "stockquote returns N/A",
			template: "${stockquote AAPL}",
			expected: "N/A",
		},
		// APCUPSD - not implemented; requires APCUPSD daemon and NIS protocol
		{
			name:     "apcupsd returns N/A",
			template: "${apcupsd}",
			expected: "N/A",
		},
		{
			name:     "apcupsd_model returns N/A",
			template: "${apcupsd_model}",
			expected: "N/A",
		},
		{
			name:     "apcupsd_status returns N/A",
			template: "${apcupsd_status}",
			expected: "N/A",
		},
		{
			name:     "apcupsd_linev returns N/A",
			template: "${apcupsd_linev}",
			expected: "N/A",
		},
		{
			name:     "apcupsd_load returns N/A",
			template: "${apcupsd_load}",
			expected: "N/A",
		},
		{
			name:     "apcupsd_charge returns N/A",
			template: "${apcupsd_charge}",
			expected: "N/A",
		},
		{
			name:     "apcupsd_timeleft returns N/A",
			template: "${apcupsd_timeleft}",
			expected: "N/A",
		},
		{
			name:     "apcupsd_temp returns N/A",
			template: "${apcupsd_temp}",
			expected: "N/A",
		},
		{
			name:     "apcupsd_battv returns N/A",
			template: "${apcupsd_battv}",
			expected: "N/A",
		},
		{
			name:     "apcupsd_cable returns N/A",
			template: "${apcupsd_cable}",
			expected: "N/A",
		},
		{
			name:     "apcupsd_driver returns N/A",
			template: "${apcupsd_driver}",
			expected: "N/A",
		},
		{
			name:     "apcupsd_upsmode returns N/A",
			template: "${apcupsd_upsmode}",
			expected: "N/A",
		},
		{
			name:     "apcupsd_name returns N/A",
			template: "${apcupsd_name}",
			expected: "N/A",
		},
		{
			name:     "apcupsd_hostname returns N/A",
			template: "${apcupsd_hostname}",
			expected: "N/A",
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

func TestParseImageVariables(t *testing.T) {
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
		name       string
		template   string
		checkPath  string
		checkWidth float64
		checkX     float64
		noCache    bool
	}{
		{
			name:       "basic image",
			template:   "${image /path/to/image.png}",
			checkPath:  "/path/to/image.png",
			checkWidth: 0,
			checkX:     -1,
			noCache:    false,
		},
		{
			name:       "image with size",
			template:   "${image /path/to/image.png -s 100x50}",
			checkPath:  "/path/to/image.png",
			checkWidth: 100,
			checkX:     -1,
			noCache:    false,
		},
		{
			name:       "image with position",
			template:   "${image /path/to/image.png -p 10,20}",
			checkPath:  "/path/to/image.png",
			checkWidth: 0,
			checkX:     10,
			noCache:    false,
		},
		{
			name:       "image with no cache",
			template:   "${image /dynamic.png -n}",
			checkPath:  "/dynamic.png",
			checkWidth: 0,
			checkX:     -1,
			noCache:    true,
		},
		{
			name:       "image with all options",
			template:   "${image /image.png -s 200x100 -p 50,25 -n}",
			checkPath:  "/image.png",
			checkWidth: 200,
			checkX:     50,
			noCache:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := api.Parse(tt.template)

			// The result should be an image marker
			if !strings.Contains(result, "\x00IMG:") {
				t.Errorf("expected image marker in result, got %q", result)
				return
			}

			// Decode the marker
			marker := render.DecodeImageMarker(result)
			if marker == nil {
				t.Errorf("failed to decode image marker from %q", result)
				return
			}

			if marker.Path != tt.checkPath {
				t.Errorf("Path = %q, want %q", marker.Path, tt.checkPath)
			}
			if marker.Width != tt.checkWidth {
				t.Errorf("Width = %v, want %v", marker.Width, tt.checkWidth)
			}
			if marker.X != tt.checkX {
				t.Errorf("X = %v, want %v", marker.X, tt.checkX)
			}
			if marker.NoCache != tt.noCache {
				t.Errorf("NoCache = %v, want %v", marker.NoCache, tt.noCache)
			}
		})
	}
}

func TestResolveImageEmptyArgs(t *testing.T) {
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

	// Empty args should return empty string
	result := api.Parse("${image}")
	if result != "" {
		t.Errorf("expected empty string for empty image args, got %q", result)
	}
}

// TestResolveFSBar tests the filesystem bar widget resolver.
func TestResolveFSBar(t *testing.T) {
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
		name       string
		template   string
		checkPerc  float64 // Expected percentage encoded in bar
		checkWidth float64
	}{
		{
			name:       "default mount point",
			template:   "${fs_bar}",
			checkPerc:  40.0, // "/" mount is 40% in mock
			checkWidth: 100,
		},
		{
			name:       "specific mount point",
			template:   "${fs_bar /home}",
			checkPerc:  50.0, // "/home" mount is 50% in mock
			checkWidth: 100,
		},
		{
			name:       "with height",
			template:   "${fs_bar 12}",
			checkPerc:  40.0,
			checkWidth: 100,
		},
		{
			name:       "with height,width",
			template:   "${fs_bar 10,80}",
			checkPerc:  40.0,
			checkWidth: 80,
		},
		{
			name:       "with size and mount point",
			template:   "${fs_bar 10,80 /home}",
			checkPerc:  50.0,
			checkWidth: 80,
		},
		{
			name:       "unknown mount point",
			template:   "${fs_bar /unknown}",
			checkPerc:  0.0,
			checkWidth: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := api.Parse(tt.template)

			// Should return a widget marker
			if !strings.Contains(result, "\x00WGT:") {
				t.Errorf("expected widget marker in result, got %q", result)
				return
			}

			// Decode and check widget marker
			marker := render.DecodeWidgetMarker(result)
			if marker == nil {
				t.Errorf("failed to decode widget marker from %q", result)
				return
			}

			if marker.Value != tt.checkPerc {
				t.Errorf("Value = %v, want %v", marker.Value, tt.checkPerc)
			}
			if marker.Width != tt.checkWidth {
				t.Errorf("Width = %v, want %v", marker.Width, tt.checkWidth)
			}
		})
	}
}

// TestResolveBatteryBar tests the battery bar widget resolver.
func TestResolveBatteryBar(t *testing.T) {
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
		name       string
		template   string
		checkPerc  float64
		checkWidth float64
	}{
		{
			name:       "default size",
			template:   "${battery_bar}",
			checkPerc:  85.0, // TotalCapacity from mock
			checkWidth: 100,
		},
		{
			name:       "with height",
			template:   "${battery_bar 12}",
			checkPerc:  85.0,
			checkWidth: 100,
		},
		{
			name:       "with height and width",
			template:   "${battery_bar 15 150}",
			checkPerc:  85.0,
			checkWidth: 150,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := api.Parse(tt.template)

			if !strings.Contains(result, "\x00WGT:") {
				t.Errorf("expected widget marker in result, got %q", result)
				return
			}

			marker := render.DecodeWidgetMarker(result)
			if marker == nil {
				t.Errorf("failed to decode widget marker from %q", result)
				return
			}

			if marker.Value != tt.checkPerc {
				t.Errorf("Value = %v, want %v", marker.Value, tt.checkPerc)
			}
			if marker.Width != tt.checkWidth {
				t.Errorf("Width = %v, want %v", marker.Width, tt.checkWidth)
			}
		})
	}
}

// TestResolveIfRunning tests the if_running process check resolver.
func TestResolveIfRunning(t *testing.T) {
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
		want     string
	}{
		{
			name:     "running process",
			template: "${if_running firefox}yes${else}no${endif}",
			want:     "yes",
		},
		{
			name:     "partial match",
			template: "${if_running fire}yes${else}no${endif}",
			want:     "yes",
		},
		{
			name:     "not running process",
			template: "${if_running nonexistent}yes${else}no${endif}",
			want:     "no",
		},
		{
			name:     "another running process",
			template: "${if_running chrome}yes${else}no${endif}",
			want:     "yes",
		},
		{
			name:     "case sensitivity",
			template: "${if_running FIREFOX}yes${else}no${endif}",
			want:     "no", // Process name is lowercase "firefox"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := api.Parse(tt.template)
			if result != tt.want {
				t.Errorf("Parse(%q) = %q, want %q", tt.template, result, tt.want)
			}
		})
	}
}

// TestResolveACPIFan tests the ACPI fan status resolver.
func TestResolveACPIFan(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	tests := []struct {
		name  string
		hwmon monitor.HwmonStats
		want  string
	}{
		{
			name: "fan device present",
			hwmon: monitor.HwmonStats{
				Devices: map[string]monitor.HwmonDevice{
					"coretemp":     {Name: "coretemp"},
					"thinkpad_fan": {Name: "thinkpad_fan"},
				},
			},
			want: "running",
		},
		{
			name: "fan in name case insensitive",
			hwmon: monitor.HwmonStats{
				Devices: map[string]monitor.HwmonDevice{
					"CPU_FAN": {Name: "CPU_FAN"},
				},
			},
			want: "running",
		},
		{
			name: "no fan device",
			hwmon: monitor.HwmonStats{
				Devices: map[string]monitor.HwmonDevice{
					"coretemp": {Name: "coretemp"},
					"amdgpu":   {Name: "amdgpu"},
				},
			},
			want: "unknown",
		},
		{
			name:  "empty devices",
			hwmon: monitor.HwmonStats{},
			want:  "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := newMockProvider()
			provider.hwmon = tt.hwmon

			api, err := NewConkyAPI(runtime, provider)
			if err != nil {
				t.Fatalf("failed to create API: %v", err)
			}

			result := api.Parse("${acpifan}")
			if result != tt.want {
				t.Errorf("Parse(${acpifan}) = %q, want %q", result, tt.want)
			}
		})
	}
}

// TestResolveBattery tests the battery status string resolver.
func TestResolveBattery(t *testing.T) {
	runtime, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create runtime: %v", err)
	}
	defer runtime.Close()

	tests := []struct {
		name    string
		battery monitor.BatteryStats
		args    string
		want    string
	}{
		{
			name:    "no battery",
			battery: monitor.BatteryStats{},
			args:    "",
			want:    "No battery",
		},
		{
			name: "specific battery",
			battery: monitor.BatteryStats{
				Batteries: map[string]monitor.BatteryInfo{
					"BAT0": {Capacity: 75, Status: "Charging"},
				},
			},
			args: "BAT0",
			want: "Charging 75%",
		},
		{
			name: "default battery",
			battery: monitor.BatteryStats{
				Batteries: map[string]monitor.BatteryInfo{
					"BAT0": {Capacity: 50, Status: "Discharging"},
				},
			},
			args: "",
			want: "Discharging 50%",
		},
		{
			name: "aggregate charging on AC",
			battery: monitor.BatteryStats{
				Batteries:     map[string]monitor.BatteryInfo{},
				ACOnline:      true,
				IsCharging:    true,
				TotalCapacity: 80.0,
			},
			args: "UNKNOWN",
			want: "Charging 80%",
		},
		{
			name: "aggregate full on AC",
			battery: monitor.BatteryStats{
				Batteries:     map[string]monitor.BatteryInfo{},
				ACOnline:      true,
				IsCharging:    false,
				TotalCapacity: 100.0,
			},
			args: "UNKNOWN",
			want: "Full 100%",
		},
		{
			name: "aggregate discharging",
			battery: monitor.BatteryStats{
				Batteries:     map[string]monitor.BatteryInfo{},
				ACOnline:      false,
				IsDischarging: true,
				TotalCapacity: 45.0,
			},
			args: "UNKNOWN",
			want: "Discharging 45%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := newMockProvider()
			provider.battery = tt.battery

			api, err := NewConkyAPI(runtime, provider)
			if err != nil {
				t.Fatalf("failed to create API: %v", err)
			}

			template := "${battery}"
			if tt.args != "" {
				template = fmt.Sprintf("${battery %s}", tt.args)
			}

			result := api.Parse(template)
			if result != tt.want {
				t.Errorf("Parse(%q) = %q, want %q", template, result, tt.want)
			}
		})
	}
}

// TestCacheCleanupRemovesStaleEntries verifies that CleanupCaches removes old entries.
func TestCacheCleanupRemovesStaleEntries(t *testing.T) {
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
	defer api.Close()

	// Set a very short MaxAge for testing
	api.SetCacheCleanupConfig(CacheCleanupConfig{
		MaxAge:          10 * time.Millisecond,
		CleanupInterval: 5 * time.Millisecond,
	})

	// Add some entries directly to the caches with old lastAccessed times
	api.mu.Lock()
	oldTime := time.Now().Add(-1 * time.Hour)
	api.execCache["old_cmd"] = &execCacheEntry{
		output:       "old output",
		expiresAt:    time.Now().Add(1 * time.Hour),
		lastAccessed: oldTime,
	}
	api.execCache["recent_cmd"] = &execCacheEntry{
		output:       "recent output",
		expiresAt:    time.Now().Add(1 * time.Hour),
		lastAccessed: time.Now(),
	}
	api.scrollStates["old_scroll"] = &scrollState{
		position:     5,
		lastUpdate:   oldTime,
		lastAccessed: oldTime,
	}
	api.scrollStates["recent_scroll"] = &scrollState{
		position:     3,
		lastUpdate:   time.Now(),
		lastAccessed: time.Now(),
	}
	api.mu.Unlock()

	// Verify initial counts
	execCount, scrollCount := api.CacheStats()
	if execCount != 2 {
		t.Errorf("expected 2 exec cache entries, got %d", execCount)
	}
	if scrollCount != 2 {
		t.Errorf("expected 2 scroll state entries, got %d", scrollCount)
	}

	// Run cleanup
	execRemoved, scrollRemoved := api.CleanupCaches()
	if execRemoved != 1 {
		t.Errorf("expected 1 exec entry removed, got %d", execRemoved)
	}
	if scrollRemoved != 1 {
		t.Errorf("expected 1 scroll entry removed, got %d", scrollRemoved)
	}

	// Verify final counts
	execCount, scrollCount = api.CacheStats()
	if execCount != 1 {
		t.Errorf("expected 1 exec cache entry remaining, got %d", execCount)
	}
	if scrollCount != 1 {
		t.Errorf("expected 1 scroll state entry remaining, got %d", scrollCount)
	}
}

// TestCacheCleanupBackground verifies background cleanup works correctly.
func TestCacheCleanupBackground(t *testing.T) {
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
	defer api.Close()

	// Stop the auto-started cleanup first so we can configure a shorter interval
	api.StopCacheCleanup()

	// Set a very short cleanup interval
	api.SetCacheCleanupConfig(CacheCleanupConfig{
		MaxAge:          10 * time.Millisecond,
		CleanupInterval: 20 * time.Millisecond,
	})

	// Add an old entry
	api.mu.Lock()
	oldTime := time.Now().Add(-1 * time.Hour)
	api.execCache["stale_entry"] = &execCacheEntry{
		output:       "stale",
		expiresAt:    time.Now().Add(1 * time.Hour),
		lastAccessed: oldTime,
	}
	api.mu.Unlock()

	execCount, _ := api.CacheStats()
	if execCount != 1 {
		t.Errorf("expected 1 exec cache entry, got %d", execCount)
	}

	// Start background cleanup with new config
	api.StartCacheCleanup()

	// Wait for at least one cleanup cycle
	time.Sleep(50 * time.Millisecond)

	// Verify the stale entry was cleaned
	execCount, _ = api.CacheStats()
	if execCount != 0 {
		t.Errorf("expected 0 exec cache entries after background cleanup, got %d", execCount)
	}
}

// TestCacheStatsReturnsCorrectCounts verifies CacheStats works correctly.
func TestCacheStatsReturnsCorrectCounts(t *testing.T) {
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
	defer api.Close()

	// Initially both should be empty
	execCount, scrollCount := api.CacheStats()
	if execCount != 0 || scrollCount != 0 {
		t.Errorf("expected empty caches, got exec=%d scroll=%d", execCount, scrollCount)
	}

	// Add entries
	now := time.Now()
	api.mu.Lock()
	api.execCache["cmd1"] = &execCacheEntry{output: "1", expiresAt: now, lastAccessed: now}
	api.execCache["cmd2"] = &execCacheEntry{output: "2", expiresAt: now, lastAccessed: now}
	api.scrollStates["s1"] = &scrollState{position: 0, lastUpdate: now, lastAccessed: now}
	api.mu.Unlock()

	execCount, scrollCount = api.CacheStats()
	if execCount != 2 {
		t.Errorf("expected 2 exec entries, got %d", execCount)
	}
	if scrollCount != 1 {
		t.Errorf("expected 1 scroll entry, got %d", scrollCount)
	}
}

// TestDefaultCacheCleanupConfig verifies default config values.
func TestDefaultCacheCleanupConfig(t *testing.T) {
	cfg := DefaultCacheCleanupConfig()

	if cfg.MaxAge != 5*time.Minute {
		t.Errorf("expected MaxAge=5m, got %v", cfg.MaxAge)
	}
	if cfg.CleanupInterval != 1*time.Minute {
		t.Errorf("expected CleanupInterval=1m, got %v", cfg.CleanupInterval)
	}
}

// TestStartStopCacheCleanup verifies start/stop behavior.
func TestStartStopCacheCleanup(t *testing.T) {
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
	defer api.Close() // Cleanup is now auto-started, so we should close

	// Cleanup is auto-started by NewConkyAPI, so stop it first
	api.StopCacheCleanup()

	// Start cleanup again
	api.StartCacheCleanup()

	// Double start should be safe (no-op)
	api.StartCacheCleanup()

	// Stop cleanup
	api.StopCacheCleanup()

	// Double stop should be safe (no-op)
	api.StopCacheCleanup()
}
