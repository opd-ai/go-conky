package monitor

import (
	"os"
	"path/filepath"
	"testing"
)

// BenchmarkCPUReaderReadStats benchmarks CPU statistics reading from /proc/stat.
func BenchmarkCPUReaderReadStats(b *testing.B) {
	tmpDir := b.TempDir()

	// Create realistic mock /proc/stat content
	statContent := `cpu  10132153 290696 3084719 46828483 16683 0 25195 0 0 0
cpu0 1393280 32966 572056 13343292 6130 0 17875 0 0 0
cpu1 1335066 30711 557760 13343333 4566 0 4376 0 0 0
cpu2 1365604 30388 574665 13341851 3022 0 1463 0 0 0
cpu3 1354703 30627 576034 13342115 2966 0 1480 0 0 0
cpu4 1369634 28846 575873 13342456 0 0 0 0 0 0
cpu5 1369634 28846 575873 13342456 0 0 0 0 0 0
cpu6 1369634 28846 575873 13342456 0 0 0 0 0 0
cpu7 1369634 28846 575873 13342456 0 0 0 0 0 0
intr 199292316 6 0 0 0 0 0 4 0 1 0 0 0 0 0 0 0 39 0 0 0
ctxt 620113968
btime 1539008020
processes 80853
procs_running 2
procs_blocked 0
`
	statPath := filepath.Join(tmpDir, "stat")
	if err := os.WriteFile(statPath, []byte(statContent), 0o644); err != nil {
		b.Fatalf("failed to write mock stat: %v", err)
	}

	// Create mock /proc/cpuinfo
	cpuinfoContent := `processor	: 0
vendor_id	: GenuineIntel
cpu family	: 6
model		: 158
model name	: Intel(R) Core(TM) i7-8700 CPU @ 3.20GHz
cpu MHz		: 3200.000

processor	: 1
vendor_id	: GenuineIntel
cpu family	: 6
model		: 158
model name	: Intel(R) Core(TM) i7-8700 CPU @ 3.20GHz
cpu MHz		: 3200.000
`
	cpuinfoPath := filepath.Join(tmpDir, "cpuinfo")
	if err := os.WriteFile(cpuinfoPath, []byte(cpuinfoContent), 0o644); err != nil {
		b.Fatalf("failed to write mock cpuinfo: %v", err)
	}

	reader := &cpuReader{
		procStatPath: statPath,
		procInfoPath: cpuinfoPath,
	}

	// Warm up to establish baseline
	_, _ = reader.ReadStats()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = reader.ReadStats()
	}
}

// BenchmarkMemoryReaderReadStats benchmarks memory statistics reading from /proc/meminfo.
func BenchmarkMemoryReaderReadStats(b *testing.B) {
	tmpDir := b.TempDir()

	// Create realistic mock /proc/meminfo content
	meminfoContent := `MemTotal:       16342012 kB
MemFree:          234716 kB
MemAvailable:    8456792 kB
Buffers:          456780 kB
Cached:          6789012 kB
SwapCached:        12345 kB
Active:          8765432 kB
Inactive:        5432100 kB
Active(anon):    4567890 kB
Inactive(anon):   123456 kB
Active(file):    4197542 kB
Inactive(file):  5308644 kB
Unevictable:           0 kB
Mlocked:               0 kB
SwapTotal:       8388604 kB
SwapFree:        7654321 kB
Dirty:              1234 kB
Writeback:             0 kB
AnonPages:       4567890 kB
Mapped:           123456 kB
Shmem:            234567 kB
KReclaimable:     345678 kB
Slab:             567890 kB
SReclaimable:     345678 kB
SUnreclaim:       222212 kB
KernelStack:       12345 kB
PageTables:        45678 kB
NFS_Unstable:          0 kB
Bounce:                0 kB
WritebackTmp:          0 kB
CommitLimit:    16559608 kB
Committed_AS:   12345678 kB
VmallocTotal:   34359738367 kB
VmallocUsed:       23456 kB
VmallocChunk:          0 kB
Percpu:             4567 kB
HardwareCorrupted:     0 kB
AnonHugePages:         0 kB
ShmemHugePages:        0 kB
ShmemPmdMapped:        0 kB
FileHugePages:         0 kB
FilePmdMapped:         0 kB
HugePages_Total:       0
HugePages_Free:        0
HugePages_Rsvd:        0
HugePages_Surp:        0
Hugepagesize:       2048 kB
Hugetlb:               0 kB
DirectMap4k:      234567 kB
DirectMap2M:    10485760 kB
DirectMap1G:     7340032 kB
`
	meminfoPath := filepath.Join(tmpDir, "meminfo")
	if err := os.WriteFile(meminfoPath, []byte(meminfoContent), 0o644); err != nil {
		b.Fatalf("failed to write mock meminfo: %v", err)
	}

	reader := &memoryReader{
		procMemInfoPath: meminfoPath,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = reader.ReadStats()
	}
}

// BenchmarkNetworkReaderReadStats benchmarks network statistics reading from /proc/net/dev.
func BenchmarkNetworkReaderReadStats(b *testing.B) {
	tmpDir := b.TempDir()

	// Create mock /proc/net/dev
	netDevContent := `Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
    lo: 12345678   12345    0    0    0     0          0         0 12345678   12345    0    0    0     0       0          0
  eth0: 87654321   54321   10    5    0     0          0       100 43210987   32100    2    1    0     0       0          0
wlan0: 11111111   22222    1    2    0     0          0        50  5555555    6666    0    0    0     0       0          0
docker0: 9876543    8765    0    0    0     0          0         0  1234567    2345    0    0    0     0       0          0
`
	netDevPath := filepath.Join(tmpDir, "net_dev")
	if err := os.WriteFile(netDevPath, []byte(netDevContent), 0o644); err != nil {
		b.Fatalf("failed to write mock net_dev: %v", err)
	}

	reader := &networkReader{
		procNetDevPath: netDevPath,
		prevStats:      make(map[string]rawInterfaceStats),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = reader.ReadStats()
	}
}

// BenchmarkUptimeReaderReadStats benchmarks uptime reading from /proc/uptime.
func BenchmarkUptimeReaderReadStats(b *testing.B) {
	tmpDir := b.TempDir()

	// Create mock /proc/uptime
	uptimeContent := `123456.78 987654.32`
	uptimePath := filepath.Join(tmpDir, "uptime")
	if err := os.WriteFile(uptimePath, []byte(uptimeContent), 0o644); err != nil {
		b.Fatalf("failed to write mock uptime: %v", err)
	}

	reader := &uptimeReader{
		procUptimePath: uptimePath,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = reader.ReadStats()
	}
}

// BenchmarkDiskIOReaderReadStats benchmarks disk I/O statistics reading from /proc/diskstats.
func BenchmarkDiskIOReaderReadStats(b *testing.B) {
	tmpDir := b.TempDir()

	// Create mock /proc/diskstats with multiple devices
	diskstatsContent := `   8       0 sda 12345 6789 234567 8901 23456 7890 345678 9012 0 12345 67890 0 0 0 0 0 0
   8       1 sda1 1234 567 23456 890 2345 678 34567 901 0 1234 5678 0 0 0 0 0 0
   8       2 sda2 11111 6222 211111 8011 21111 7212 311111 8111 0 11111 62012 0 0 0 0 0 0
   259     0 nvme0n1 98765 4321 876543 2109 87654 3210 765432 1098 0 98765 43210 0 0 0 0 0 0
   259     1 nvme0n1p1 9876 432 87654 210 8765 321 76543 109 0 9876 4321 0 0 0 0 0 0
   259     2 nvme0n1p2 88889 3889 788889 1899 78889 2889 688889 989 0 88889 38889 0 0 0 0 0 0
`
	diskstatsPath := filepath.Join(tmpDir, "diskstats")
	if err := os.WriteFile(diskstatsPath, []byte(diskstatsContent), 0o644); err != nil {
		b.Fatalf("failed to write mock diskstats: %v", err)
	}

	reader := &diskIOReader{
		procDiskstatsPath: diskstatsPath,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = reader.ReadStats()
	}
}

// BenchmarkParseCPULine benchmarks parsing a single CPU line from /proc/stat.
func BenchmarkParseCPULine(b *testing.B) {
	fields := []string{"10132153", "290696", "3084719", "46828483", "16683", "0", "25195", "0"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parseCPULine(fields)
	}
}

// BenchmarkSystemDataGetCPU benchmarks concurrent access to CPU data.
func BenchmarkSystemDataGetCPU(b *testing.B) {
	sd := NewSystemData()
	sd.setCPU(CPUStats{
		UsagePercent: 45.5,
		Cores:        []float64{40.0, 50.0, 45.0, 46.0, 44.0, 47.0, 48.0, 43.0},
		CPUCount:     8,
		ModelName:    "Intel(R) Core(TM) i7-8700 CPU @ 3.20GHz",
		Frequency:    3200.0,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sd.GetCPU()
	}
}

// BenchmarkSystemDataGetMemory benchmarks concurrent access to memory data.
func BenchmarkSystemDataGetMemory(b *testing.B) {
	sd := NewSystemData()
	sd.setMemory(MemoryStats{
		Total:        16342012 * 1024,
		Used:         8000000 * 1024,
		Free:         234716 * 1024,
		Available:    8456792 * 1024,
		Buffers:      456780 * 1024,
		Cached:       6789012 * 1024,
		SwapTotal:    8388604 * 1024,
		SwapUsed:     734283 * 1024,
		SwapFree:     7654321 * 1024,
		UsagePercent: 48.9,
		SwapPercent:  8.8,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sd.GetMemory()
	}
}

// BenchmarkSystemDataGetNetwork benchmarks concurrent access to network data (includes deep copy).
func BenchmarkSystemDataGetNetwork(b *testing.B) {
	sd := NewSystemData()
	sd.setNetwork(NetworkStats{
		Interfaces: map[string]InterfaceStats{
			"lo":    {Name: "lo", RxBytes: 12345678, TxBytes: 12345678},
			"eth0":  {Name: "eth0", RxBytes: 87654321, TxBytes: 43210987},
			"wlan0": {Name: "wlan0", RxBytes: 11111111, TxBytes: 5555555},
		},
		TotalRxBytes:       111111110,
		TotalTxBytes:       61111542,
		TotalRxBytesPerSec: 1234.56,
		TotalTxBytesPerSec: 789.01,
		GatewayIP:          "192.168.1.1",
		GatewayInterface:   "eth0",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sd.GetNetwork()
	}
}

// BenchmarkSystemDataConcurrentAccess benchmarks concurrent read/write operations.
func BenchmarkSystemDataConcurrentAccess(b *testing.B) {
	sd := NewSystemData()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%2 == 0 {
				// Read operation
				_ = sd.GetCPU()
			} else {
				// Write operation
				sd.setCPU(CPUStats{UsagePercent: float64(i % 100)})
			}
			i++
		}
	})
}
