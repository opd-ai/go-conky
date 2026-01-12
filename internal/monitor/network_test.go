package monitor

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseNetDevLine(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		wantName  string
		wantStats rawInterfaceStats
		wantErr   bool
	}{
		{
			name: "valid eth0 line",
			line: "  eth0: 523861305  362702    0    0    0     0          0         1  7179696   49506    0    1    0     0       0          0",
			wantName: "eth0",
			wantStats: rawInterfaceStats{
				rxBytes:   523861305,
				rxPackets: 362702,
				rxErrors:  0,
				rxDropped: 0,
				txBytes:   7179696,
				txPackets: 49506,
				txErrors:  0,
				txDropped: 1,
			},
			wantErr: false,
		},
		{
			name: "valid lo line",
			line: "    lo: 34909048   19020    0    0    0     0          0         0 34909048   19020    0    0    0     0       0          0",
			wantName: "lo",
			wantStats: rawInterfaceStats{
				rxBytes:   34909048,
				rxPackets: 19020,
				rxErrors:  0,
				rxDropped: 0,
				txBytes:   34909048,
				txPackets: 19020,
				txErrors:  0,
				txDropped: 0,
			},
			wantErr: false,
		},
		{
			name: "line with errors",
			line: "  eth1: 1000 500 10 5 0 0 0 0 2000 800 20 8 0 0 0 0",
			wantName: "eth1",
			wantStats: rawInterfaceStats{
				rxBytes:   1000,
				rxPackets: 500,
				rxErrors:  10,
				rxDropped: 5,
				txBytes:   2000,
				txPackets: 800,
				txErrors:  20,
				txDropped: 8,
			},
			wantErr: false,
		},
		{
			name:    "no colon separator",
			line:    "eth0 123456 789",
			wantErr: true,
		},
		{
			name:    "empty interface name",
			line:    ": 123456 789 0 0 0 0 0 0 654321 456 0 0 0 0 0 0",
			wantErr: true,
		},
		{
			name:    "insufficient fields",
			line:    "eth0: 123 456 789",
			wantErr: true,
		},
		{
			name:    "invalid number",
			line:    "eth0: abc 456 0 0 0 0 0 0 789 123 0 0 0 0 0 0",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotStats, err := parseNetDevLine(tt.line)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseNetDevLine() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if gotName != tt.wantName {
					t.Errorf("parseNetDevLine() name = %q, want %q", gotName, tt.wantName)
				}
				if gotStats != tt.wantStats {
					t.Errorf("parseNetDevLine() stats = %+v, want %+v", gotStats, tt.wantStats)
				}
			}
		})
	}
}

func TestNetworkReaderWithMockFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create mock /proc/net/dev
	netDevContent := `Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
    lo: 34909048   19020    0    0    0     0          0         0 34909048   19020    0    0    0     0       0          0
  eth0: 523861305  362702    0    0    0     0          0         1  7179696   49506    0    1    0     0       0          0
docker0:       0       0    0    0    0     0          0         0        0       0    0    2    0     0       0          0
`
	if err := os.WriteFile(filepath.Join(tmpDir, "net_dev"), []byte(netDevContent), 0644); err != nil {
		t.Fatalf("failed to write mock net_dev: %v", err)
	}

	reader := &networkReader{
		prevStats:      make(map[string]rawInterfaceStats),
		procNetDevPath: filepath.Join(tmpDir, "net_dev"),
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	// Check that we got 3 interfaces
	if len(stats.Interfaces) != 3 {
		t.Errorf("got %d interfaces, want 3", len(stats.Interfaces))
	}

	// Check lo interface
	lo, ok := stats.Interfaces["lo"]
	if !ok {
		t.Fatal("missing lo interface")
	}
	if lo.RxBytes != 34909048 {
		t.Errorf("lo.RxBytes = %d, want 34909048", lo.RxBytes)
	}
	if lo.TxBytes != 34909048 {
		t.Errorf("lo.TxBytes = %d, want 34909048", lo.TxBytes)
	}

	// Check eth0 interface
	eth0, ok := stats.Interfaces["eth0"]
	if !ok {
		t.Fatal("missing eth0 interface")
	}
	if eth0.RxBytes != 523861305 {
		t.Errorf("eth0.RxBytes = %d, want 523861305", eth0.RxBytes)
	}
	if eth0.TxBytes != 7179696 {
		t.Errorf("eth0.TxBytes = %d, want 7179696", eth0.TxBytes)
	}
	if eth0.TxDropped != 1 {
		t.Errorf("eth0.TxDropped = %d, want 1", eth0.TxDropped)
	}

	// Check docker0 interface
	docker0, ok := stats.Interfaces["docker0"]
	if !ok {
		t.Fatal("missing docker0 interface")
	}
	if docker0.RxBytes != 0 {
		t.Errorf("docker0.RxBytes = %d, want 0", docker0.RxBytes)
	}
	if docker0.TxDropped != 2 {
		t.Errorf("docker0.TxDropped = %d, want 2", docker0.TxDropped)
	}

	// Check totals
	expectedTotalRx := uint64(34909048 + 523861305 + 0)
	expectedTotalTx := uint64(34909048 + 7179696 + 0)
	if stats.TotalRxBytes != expectedTotalRx {
		t.Errorf("TotalRxBytes = %d, want %d", stats.TotalRxBytes, expectedTotalRx)
	}
	if stats.TotalTxBytes != expectedTotalTx {
		t.Errorf("TotalTxBytes = %d, want %d", stats.TotalTxBytes, expectedTotalTx)
	}
}

func TestNetworkReaderRateCalculation(t *testing.T) {
	tmpDir := t.TempDir()
	netDevPath := filepath.Join(tmpDir, "net_dev")

	// First read with initial values
	netDevContent1 := `Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
  eth0: 1000  100    0    0    0     0          0         0  2000   200    0    0    0     0       0          0
`
	if err := os.WriteFile(netDevPath, []byte(netDevContent1), 0644); err != nil {
		t.Fatalf("failed to write mock net_dev: %v", err)
	}

	reader := &networkReader{
		prevStats:      make(map[string]rawInterfaceStats),
		procNetDevPath: netDevPath,
	}

	// First read - no rates available yet
	stats1, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("first ReadStats() error = %v", err)
	}

	eth0 := stats1.Interfaces["eth0"]
	if eth0.RxBytesPerSec != 0.0 {
		t.Errorf("first read: RxBytesPerSec = %f, want 0.0", eth0.RxBytesPerSec)
	}

	// Wait a bit and write new values
	time.Sleep(100 * time.Millisecond)

	// Second read with increased values
	netDevContent2 := `Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
  eth0: 2000  200    0    0    0     0          0         0  4000   400    0    0    0     0       0          0
`
	if err := os.WriteFile(netDevPath, []byte(netDevContent2), 0644); err != nil {
		t.Fatalf("failed to write mock net_dev (2nd): %v", err)
	}

	stats2, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("second ReadStats() error = %v", err)
	}

	eth0 = stats2.Interfaces["eth0"]
	// RxBytes increased by 1000, TxBytes increased by 2000
	// Rate should be positive (exact value depends on elapsed time)
	if eth0.RxBytesPerSec <= 0 {
		t.Errorf("second read: RxBytesPerSec = %f, want > 0", eth0.RxBytesPerSec)
	}
	if eth0.TxBytesPerSec <= 0 {
		t.Errorf("second read: TxBytesPerSec = %f, want > 0", eth0.TxBytesPerSec)
	}
}

func TestNetworkReaderMissingFile(t *testing.T) {
	reader := &networkReader{
		prevStats:      make(map[string]rawInterfaceStats),
		procNetDevPath: "/nonexistent/net/dev",
	}

	_, err := reader.ReadStats()
	if err == nil {
		t.Error("ReadStats() should return error for missing file")
	}
}

func TestNetworkReaderEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create empty file
	if err := os.WriteFile(filepath.Join(tmpDir, "net_dev"), []byte(""), 0644); err != nil {
		t.Fatalf("failed to write mock net_dev: %v", err)
	}

	reader := &networkReader{
		prevStats:      make(map[string]rawInterfaceStats),
		procNetDevPath: filepath.Join(tmpDir, "net_dev"),
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	if len(stats.Interfaces) != 0 {
		t.Errorf("got %d interfaces from empty file, want 0", len(stats.Interfaces))
	}
}

func TestNetworkReaderHeadersOnly(t *testing.T) {
	tmpDir := t.TempDir()

	// Create file with headers only
	netDevContent := `Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
`
	if err := os.WriteFile(filepath.Join(tmpDir, "net_dev"), []byte(netDevContent), 0644); err != nil {
		t.Fatalf("failed to write mock net_dev: %v", err)
	}

	reader := &networkReader{
		prevStats:      make(map[string]rawInterfaceStats),
		procNetDevPath: filepath.Join(tmpDir, "net_dev"),
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	if len(stats.Interfaces) != 0 {
		t.Errorf("got %d interfaces from headers-only file, want 0", len(stats.Interfaces))
	}
}

func TestNetworkReaderCalculateRate(t *testing.T) {
	reader := newNetworkReader()

	tests := []struct {
		name     string
		prev     uint64
		curr     uint64
		elapsed  float64
		expected float64
	}{
		{
			name:     "normal increase",
			prev:     1000,
			curr:     2000,
			elapsed:  1.0,
			expected: 1000.0,
		},
		{
			name:     "no change",
			prev:     1000,
			curr:     1000,
			elapsed:  1.0,
			expected: 0.0,
		},
		{
			name:     "counter wrap-around",
			prev:     2000,
			curr:     1000,
			elapsed:  1.0,
			expected: 0.0,
		},
		{
			name:     "zero elapsed time",
			prev:     1000,
			curr:     2000,
			elapsed:  0.0,
			expected: 0.0,
		},
		{
			name:     "negative elapsed time",
			prev:     1000,
			curr:     2000,
			elapsed:  -1.0,
			expected: 0.0,
		},
		{
			name:     "half second elapsed",
			prev:     1000,
			curr:     2000,
			elapsed:  0.5,
			expected: 2000.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reader.calculateRate(tt.prev, tt.curr, tt.elapsed)
			if got != tt.expected {
				t.Errorf("calculateRate() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNetworkReaderMalformedLines(t *testing.T) {
	tmpDir := t.TempDir()

	// Create file with some malformed lines mixed in
	netDevContent := `Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
  eth0: 1000  100    0    0    0     0          0         0  2000   200    0    0    0     0       0          0
malformed line without colon
  lo: 500   50    0    0    0     0          0         0  500    50    0    0    0     0       0          0
another: broken
`
	if err := os.WriteFile(filepath.Join(tmpDir, "net_dev"), []byte(netDevContent), 0644); err != nil {
		t.Fatalf("failed to write mock net_dev: %v", err)
	}

	reader := &networkReader{
		prevStats:      make(map[string]rawInterfaceStats),
		procNetDevPath: filepath.Join(tmpDir, "net_dev"),
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	// Should only have eth0 and lo - malformed lines should be skipped
	if len(stats.Interfaces) != 2 {
		t.Errorf("got %d interfaces, want 2 (eth0 and lo)", len(stats.Interfaces))
	}

	if _, ok := stats.Interfaces["eth0"]; !ok {
		t.Error("missing eth0 interface")
	}
	if _, ok := stats.Interfaces["lo"]; !ok {
		t.Error("missing lo interface")
	}
}

func TestNewNetworkReader(t *testing.T) {
	reader := newNetworkReader()

	if reader.procNetDevPath != "/proc/net/dev" {
		t.Errorf("procNetDevPath = %q, want %q", reader.procNetDevPath, "/proc/net/dev")
	}
	if reader.prevStats == nil {
		t.Error("prevStats should be initialized")
	}
}
