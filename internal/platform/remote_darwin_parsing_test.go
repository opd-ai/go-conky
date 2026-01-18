package platform

import (
	"errors"
	"testing"
)

// Test remote Darwin CPU provider with mock command runner

func TestRemoteDarwinCPU_TotalUsage(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteDarwinCPUProviderWithRunner(mock)

	// iostat output format: us sy id
	mock.setCommandResult("iostat -c 2 | tail -n 1", "  15.5   8.2  76.3")

	usage, err := provider.TotalUsage()
	if err != nil {
		t.Fatalf("TotalUsage() error = %v", err)
	}

	// Total usage = user + system = 15.5 + 8.2 = 23.7
	expected := 23.7
	if usage != expected {
		t.Errorf("TotalUsage() = %f, want %f", usage, expected)
	}
}

func TestRemoteDarwinCPU_TotalUsage_Error(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteDarwinCPUProviderWithRunner(mock)

	mock.setCommandError("iostat -c 2 | tail -n 1", errors.New("command failed"))

	_, err := provider.TotalUsage()
	if err == nil {
		t.Error("TotalUsage() expected error for command failure")
	}
}

func TestRemoteDarwinCPU_TotalUsage_InvalidOutput(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteDarwinCPUProviderWithRunner(mock)

	// Only 2 fields instead of 3
	mock.setCommandResult("iostat -c 2 | tail -n 1", "15.5 8.2")

	_, err := provider.TotalUsage()
	if err == nil {
		t.Error("TotalUsage() expected error for invalid output format")
	}
}

func TestRemoteDarwinCPU_LoadAverage(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteDarwinCPUProviderWithRunner(mock)

	// macOS loadavg format: { 1.5 2.0 2.5 }
	mock.setCommandResult("sysctl -n vm.loadavg", "{ 1.50 2.00 2.50 }")

	load1, load5, load15, err := provider.LoadAverage()
	if err != nil {
		t.Fatalf("LoadAverage() error = %v", err)
	}

	if load1 != 1.50 {
		t.Errorf("LoadAverage() load1 = %f, want 1.50", load1)
	}
	if load5 != 2.00 {
		t.Errorf("LoadAverage() load5 = %f, want 2.00", load5)
	}
	if load15 != 2.50 {
		t.Errorf("LoadAverage() load15 = %f, want 2.50", load15)
	}
}

func TestRemoteDarwinCPU_LoadAverage_Error(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteDarwinCPUProviderWithRunner(mock)

	mock.setCommandError("sysctl -n vm.loadavg", errors.New("command failed"))

	_, _, _, err := provider.LoadAverage()
	if err == nil {
		t.Error("LoadAverage() expected error for command failure")
	}
}

func TestRemoteDarwinCPU_Info(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteDarwinCPUProviderWithRunner(mock)

	mock.setCommandResult("sysctl -n machdep.cpu.brand_string", "Apple M1 Pro")
	mock.setCommandResult("sysctl -n machdep.cpu.vendor", "Apple")
	mock.setCommandResult("sysctl -n hw.physicalcpu", "8")
	mock.setCommandResult("sysctl -n hw.logicalcpu", "10")
	mock.setCommandResult("sysctl -n hw.l3cachesize", "12582912")

	info, err := provider.Info()
	if err != nil {
		t.Fatalf("Info() error = %v", err)
	}

	if info.Model != "Apple M1 Pro" {
		t.Errorf("Info().Model = %s, want 'Apple M1 Pro'", info.Model)
	}
	if info.Vendor != "Apple" {
		t.Errorf("Info().Vendor = %s, want 'Apple'", info.Vendor)
	}
	if info.Cores != 8 {
		t.Errorf("Info().Cores = %d, want 8", info.Cores)
	}
	if info.Threads != 10 {
		t.Errorf("Info().Threads = %d, want 10", info.Threads)
	}
	if info.CacheSize != 12582912 {
		t.Errorf("Info().CacheSize = %d, want 12582912", info.CacheSize)
	}
}

func TestRemoteDarwinCPU_Frequency(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteDarwinCPUProviderWithRunner(mock)

	// CPU frequency in Hz
	mock.setCommandResult("sysctl -n hw.cpufrequency", "3200000000")
	mock.setCommandResult("sysctl -n machdep.cpu.brand_string", "Intel Core i7")
	mock.setCommandResult("sysctl -n machdep.cpu.vendor", "Intel")
	mock.setCommandResult("sysctl -n hw.physicalcpu", "4")
	mock.setCommandResult("sysctl -n hw.logicalcpu", "8")
	mock.setCommandResult("sysctl -n hw.l3cachesize", "8388608")

	freqs, err := provider.Frequency()
	if err != nil {
		t.Fatalf("Frequency() error = %v", err)
	}

	// Should return 8 frequencies (one per thread)
	if len(freqs) != 8 {
		t.Errorf("Frequency() returned %d values, want 8", len(freqs))
	}

	// Each should be 3200 MHz
	for i, freq := range freqs {
		if freq != 3200.0 {
			t.Errorf("Frequency()[%d] = %f, want 3200.0", i, freq)
		}
	}
}

func TestRemoteDarwinCPU_Usage(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteDarwinCPUProviderWithRunner(mock)

	mock.setCommandResult("iostat -c 2 | tail -n 1", "  20.0   10.0  70.0")
	mock.setCommandResult("sysctl -n machdep.cpu.brand_string", "Intel Core i7")
	mock.setCommandResult("sysctl -n machdep.cpu.vendor", "Intel")
	mock.setCommandResult("sysctl -n hw.physicalcpu", "4")
	mock.setCommandResult("sysctl -n hw.logicalcpu", "4")
	mock.setCommandResult("sysctl -n hw.l3cachesize", "8388608")

	usages, err := provider.Usage()
	if err != nil {
		t.Fatalf("Usage() error = %v", err)
	}

	// Should return 4 usages (one per thread)
	if len(usages) != 4 {
		t.Errorf("Usage() returned %d values, want 4", len(usages))
	}

	// Each should be 30% (20 + 10)
	for i, usage := range usages {
		if usage != 30.0 {
			t.Errorf("Usage()[%d] = %f, want 30.0", i, usage)
		}
	}
}

// Test remote Darwin Memory provider with mock command runner

func TestRemoteDarwinMemory_Stats(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteDarwinMemoryProviderWithRunner(mock)

	// Total memory in bytes (16 GB)
	mock.setCommandResult("sysctl -n hw.memsize", "17179869184")
	mock.setCommandResult("vm_stat", `Mach Virtual Memory Statistics: (page size of 16384 bytes)
Pages free:                             100000.
Pages active:                           200000.
Pages inactive:                         150000.
Pages wired down:                       50000.
File-backed pages:                      80000.
`)

	stats, err := provider.Stats()
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}

	expectedTotal := uint64(17179869184)
	if stats.Total != expectedTotal {
		t.Errorf("Stats().Total = %d, want %d", stats.Total, expectedTotal)
	}

	// Free = (free + inactive) * pageSize = (100000 + 150000) * 16384
	expectedFree := uint64(250000 * 16384)
	if stats.Free != expectedFree {
		t.Errorf("Stats().Free = %d, want %d", stats.Free, expectedFree)
	}

	// Used = (active + wired) * pageSize = (200000 + 50000) * 16384
	expectedUsed := uint64(250000 * 16384)
	if stats.Used != expectedUsed {
		t.Errorf("Stats().Used = %d, want %d", stats.Used, expectedUsed)
	}

	// Cached = file-backed * pageSize = 80000 * 16384
	expectedCached := uint64(80000 * 16384)
	if stats.Cached != expectedCached {
		t.Errorf("Stats().Cached = %d, want %d", stats.Cached, expectedCached)
	}
}

func TestRemoteDarwinMemory_Stats_DefaultPageSize(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteDarwinMemoryProviderWithRunner(mock)

	mock.setCommandResult("sysctl -n hw.memsize", "8589934592")
	// vm_stat output without explicit page size (should use default 4096)
	mock.setCommandResult("vm_stat", `Pages free:                             100000.
Pages active:                           200000.
`)

	stats, err := provider.Stats()
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}

	// Free = 100000 * 4096 (default page size)
	expectedFree := uint64(100000 * 4096)
	if stats.Free != expectedFree {
		t.Errorf("Stats().Free = %d, want %d", stats.Free, expectedFree)
	}
}

func TestRemoteDarwinMemory_SwapStats(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteDarwinMemoryProviderWithRunner(mock)

	mock.setCommandResult("sysctl -n vm.swapusage", "total = 2048.00M  used = 512.00M  free = 1536.00M")

	stats, err := provider.SwapStats()
	if err != nil {
		t.Fatalf("SwapStats() error = %v", err)
	}

	// Total = 2048 MB = 2048 * 1024 * 1024
	expectedTotal := uint64(2048 * 1024 * 1024)
	if stats.Total != expectedTotal {
		t.Errorf("SwapStats().Total = %d, want %d", stats.Total, expectedTotal)
	}

	// Used = 512 MB
	expectedUsed := uint64(512 * 1024 * 1024)
	if stats.Used != expectedUsed {
		t.Errorf("SwapStats().Used = %d, want %d", stats.Used, expectedUsed)
	}

	// Free = 1536 MB
	expectedFree := uint64(1536 * 1024 * 1024)
	if stats.Free != expectedFree {
		t.Errorf("SwapStats().Free = %d, want %d", stats.Free, expectedFree)
	}

	// UsedPercent = 512 / 2048 * 100 = 25%
	if stats.UsedPercent != 25.0 {
		t.Errorf("SwapStats().UsedPercent = %f, want 25.0", stats.UsedPercent)
	}
}

func TestParseMemorySize(t *testing.T) {
	tests := []struct {
		input    string
		expected uint64
		wantErr  bool
	}{
		{"1024K", 1024 * 1024, false},
		{"512M", 512 * 1024 * 1024, false},
		{"2G", 2 * 1024 * 1024 * 1024, false},
		{"1024k", 1024 * 1024, false},
		{"2048.00M", 2048 * 1024 * 1024, false},
		{"1.5G", 1610612736, false}, // 1.5 * 1024 * 1024 * 1024
		{"", 0, true},
		{"abc", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseMemorySize(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseMemorySize(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.expected {
				t.Errorf("parseMemorySize(%q) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}

// Test remote Darwin Network provider with mock command runner

func TestRemoteDarwinNetwork_Interfaces(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteDarwinNetworkProviderWithRunner(mock)

	mock.setCommandResult("netstat -i | tail -n +2 | awk '{print $1}' | sort -u", `en0
en1
lo0
`)

	interfaces, err := provider.Interfaces()
	if err != nil {
		t.Fatalf("Interfaces() error = %v", err)
	}

	if len(interfaces) != 3 {
		t.Errorf("Interfaces() returned %d interfaces, want 3", len(interfaces))
	}

	expected := []string{"en0", "en1", "lo0"}
	for i, iface := range expected {
		if interfaces[i] != iface {
			t.Errorf("Interfaces()[%d] = %s, want %s", i, interfaces[i], iface)
		}
	}
}

func TestRemoteDarwinNetwork_Stats(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteDarwinNetworkProviderWithRunner(mock)

	// netstat -ib output format:
	// Name Mtu Network Address Ipkts Ierrs Ibytes Opkts Oerrs Obytes Coll
	mock.setCommandResult("netstat -ib -I 'en0' | tail -n 1",
		"en0   1500  <Link#6>  aa:bb:cc:dd:ee:ff  123456  10  987654321  654321  5  123456789  0")

	stats, err := provider.Stats("en0")
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}

	if stats.PacketsRecv != 123456 {
		t.Errorf("Stats().PacketsRecv = %d, want 123456", stats.PacketsRecv)
	}
	if stats.ErrorsIn != 10 {
		t.Errorf("Stats().ErrorsIn = %d, want 10", stats.ErrorsIn)
	}
	if stats.BytesRecv != 987654321 {
		t.Errorf("Stats().BytesRecv = %d, want 987654321", stats.BytesRecv)
	}
	if stats.PacketsSent != 654321 {
		t.Errorf("Stats().PacketsSent = %d, want 654321", stats.PacketsSent)
	}
	if stats.ErrorsOut != 5 {
		t.Errorf("Stats().ErrorsOut = %d, want 5", stats.ErrorsOut)
	}
	if stats.BytesSent != 123456789 {
		t.Errorf("Stats().BytesSent = %d, want 123456789", stats.BytesSent)
	}
}

func TestRemoteDarwinNetwork_AllStats(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteDarwinNetworkProviderWithRunner(mock)

	mock.setCommandResult("netstat -i | tail -n +2 | awk '{print $1}' | sort -u", "en0\nlo0")
	mock.setCommandResult("netstat -ib -I 'en0' | tail -n 1",
		"en0   1500  <Link#6>  aa:bb:cc:dd  1000  0  50000  500  0  25000  0")
	mock.setCommandResult("netstat -ib -I 'lo0' | tail -n 1",
		"lo0   16384  <Link#1>  127.0.0.1  2000  0  100000  2000  0  100000  0")

	allStats, err := provider.AllStats()
	if err != nil {
		t.Fatalf("AllStats() error = %v", err)
	}

	if len(allStats) != 2 {
		t.Errorf("AllStats() returned %d entries, want 2", len(allStats))
	}

	if allStats["en0"].BytesRecv != 50000 {
		t.Errorf("AllStats()[en0].BytesRecv = %d, want 50000", allStats["en0"].BytesRecv)
	}
	if allStats["lo0"].BytesRecv != 100000 {
		t.Errorf("AllStats()[lo0].BytesRecv = %d, want 100000", allStats["lo0"].BytesRecv)
	}
}

// Test remote Darwin Filesystem provider with mock command runner

func TestRemoteDarwinFilesystem_Mounts(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteDarwinFilesystemProviderWithRunner(mock)

	mock.setCommandResult("mount", `/dev/disk1s1 on / (apfs, local, journaled)
/dev/disk1s2 on /System/Volumes/Data (apfs, local, journaled)
devfs on /dev (devfs, local, nobrowse)
`)

	mounts, err := provider.Mounts()
	if err != nil {
		t.Fatalf("Mounts() error = %v", err)
	}

	if len(mounts) != 3 {
		t.Errorf("Mounts() returned %d mounts, want 3", len(mounts))
	}

	// Check first mount
	if mounts[0].Device != "/dev/disk1s1" {
		t.Errorf("Mounts()[0].Device = %s, want /dev/disk1s1", mounts[0].Device)
	}
	if mounts[0].MountPoint != "/" {
		t.Errorf("Mounts()[0].MountPoint = %s, want /", mounts[0].MountPoint)
	}
	if mounts[0].FSType != "apfs" {
		t.Errorf("Mounts()[0].FSType = %s, want apfs", mounts[0].FSType)
	}
}

func TestRemoteDarwinFilesystem_Stats(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteDarwinFilesystemProviderWithRunner(mock)

	// df -k output format:
	// Filesystem 1K-blocks Used Available Capacity Mounted
	mock.setCommandResult("df -k '/' | tail -n 1",
		"/dev/disk1s1  500000000  250000000  250000000    50%   /")
	mock.setCommandResult("df -i '/' | tail -n 1",
		"/dev/disk1s1  10000000  5000000  5000000    50%   /")

	stats, err := provider.Stats("/")
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}

	// Total = 500000000 KB * 1024
	expectedTotal := uint64(500000000 * 1024)
	if stats.Total != expectedTotal {
		t.Errorf("Stats().Total = %d, want %d", stats.Total, expectedTotal)
	}

	// Used = 250000000 KB * 1024
	expectedUsed := uint64(250000000 * 1024)
	if stats.Used != expectedUsed {
		t.Errorf("Stats().Used = %d, want %d", stats.Used, expectedUsed)
	}

	// Free = 250000000 KB * 1024
	expectedFree := uint64(250000000 * 1024)
	if stats.Free != expectedFree {
		t.Errorf("Stats().Free = %d, want %d", stats.Free, expectedFree)
	}

	// UsedPercent = 50%
	if stats.UsedPercent != 50.0 {
		t.Errorf("Stats().UsedPercent = %f, want 50.0", stats.UsedPercent)
	}

	// Inode stats
	if stats.InodesTotal != 10000000 {
		t.Errorf("Stats().InodesTotal = %d, want 10000000", stats.InodesTotal)
	}
}

func TestRemoteDarwinFilesystem_DiskIO(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteDarwinFilesystemProviderWithRunner(mock)

	// DiskIO is not supported on macOS
	_, err := provider.DiskIO("disk0")
	if err == nil {
		t.Error("DiskIO() expected error on macOS")
	}
}

// Test remote Linux Sensor provider with mock command runner

func TestRemoteLinuxSensor_Temperatures(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteLinuxSensorProviderWithRunner(mock)

	mock.setCommandResult("find /sys/class/hwmon -name 'temp*_input' 2>/dev/null",
		"/sys/class/hwmon/hwmon0/temp1_input\n/sys/class/hwmon/hwmon0/temp2_input")

	// Temperature values in millidegrees
	mock.setCommandResult("cat '/sys/class/hwmon/hwmon0/temp1_input'", "45000")
	mock.setCommandResult("cat '/sys/class/hwmon/hwmon0/temp2_input'", "55000")

	// Labels
	mock.setCommandResult("cat '/sys/class/hwmon/hwmon0/temp1_label' 2>/dev/null || echo ''", "Core 0")
	mock.setCommandResult("cat '/sys/class/hwmon/hwmon0/temp2_label' 2>/dev/null || echo ''", "Core 1")

	// Critical thresholds
	mock.setCommandResult("cat '/sys/class/hwmon/hwmon0/temp1_crit' 2>/dev/null || echo ''", "100000")
	mock.setCommandResult("cat '/sys/class/hwmon/hwmon0/temp2_crit' 2>/dev/null || echo ''", "100000")

	temps, err := provider.Temperatures()
	if err != nil {
		t.Fatalf("Temperatures() error = %v", err)
	}

	if len(temps) != 2 {
		t.Errorf("Temperatures() returned %d readings, want 2", len(temps))
	}

	// Check first temperature
	if temps[0].Value != 45.0 {
		t.Errorf("Temperatures()[0].Value = %f, want 45.0", temps[0].Value)
	}
	if temps[0].Label != "Core 0" {
		t.Errorf("Temperatures()[0].Label = %s, want 'Core 0'", temps[0].Label)
	}
	if temps[0].Critical != 100.0 {
		t.Errorf("Temperatures()[0].Critical = %f, want 100.0", temps[0].Critical)
	}
	if temps[0].Unit != "°C" {
		t.Errorf("Temperatures()[0].Unit = %s, want '°C'", temps[0].Unit)
	}
}

func TestRemoteLinuxSensor_Temperatures_NoSensors(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteLinuxSensorProviderWithRunner(mock)

	mock.setCommandResult("find /sys/class/hwmon -name 'temp*_input' 2>/dev/null", "")

	temps, err := provider.Temperatures()
	if err != nil {
		t.Fatalf("Temperatures() error = %v", err)
	}

	if len(temps) != 0 {
		t.Errorf("Temperatures() returned %d readings, want 0", len(temps))
	}
}

func TestRemoteLinuxSensor_Fans(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteLinuxSensorProviderWithRunner(mock)

	mock.setCommandResult("find /sys/class/hwmon -name 'fan*_input' 2>/dev/null",
		"/sys/class/hwmon/hwmon1/fan1_input")

	// Fan speed in RPM
	mock.setCommandResult("cat '/sys/class/hwmon/hwmon1/fan1_input'", "1500")

	// Label
	mock.setCommandResult("cat '/sys/class/hwmon/hwmon1/fan1_label' 2>/dev/null || echo ''", "CPU Fan")

	fans, err := provider.Fans()
	if err != nil {
		t.Fatalf("Fans() error = %v", err)
	}

	if len(fans) != 1 {
		t.Errorf("Fans() returned %d readings, want 1", len(fans))
	}

	if fans[0].Value != 1500.0 {
		t.Errorf("Fans()[0].Value = %f, want 1500.0", fans[0].Value)
	}
	if fans[0].Label != "CPU Fan" {
		t.Errorf("Fans()[0].Label = %s, want 'CPU Fan'", fans[0].Label)
	}
	if fans[0].Unit != "RPM" {
		t.Errorf("Fans()[0].Unit = %s, want 'RPM'", fans[0].Unit)
	}
}

func TestRemoteLinuxSensor_Fans_NoSensors(t *testing.T) {
	mock := newMockSSHPlatform()
	provider := newTestableRemoteLinuxSensorProviderWithRunner(mock)

	mock.setCommandResult("find /sys/class/hwmon -name 'fan*_input' 2>/dev/null", "")

	fans, err := provider.Fans()
	if err != nil {
		t.Fatalf("Fans() error = %v", err)
	}

	if len(fans) != 0 {
		t.Errorf("Fans() returned %d readings, want 0", len(fans))
	}
}

func TestExtractSensorName(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/sys/class/hwmon/hwmon0/temp1_input", "hwmon0_temp1"},
		{"/sys/class/hwmon/hwmon2/fan1_input", "hwmon2_fan1"},
		{"/sys/class/hwmon/hwmon1/temp3_input", "hwmon1_temp3"},
		{"short_input", "short_input"}, // Edge case
		{"/a_input", "_a"},             // Edge case with minimal path
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := extractSensorName(tt.path)
			if got != tt.expected {
				t.Errorf("extractSensorName(%q) = %q, want %q", tt.path, got, tt.expected)
			}
		})
	}
}
