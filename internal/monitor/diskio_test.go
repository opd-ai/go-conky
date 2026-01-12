package monitor

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseDiskstatsLine(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		wantName  string
		wantStats parsedDiskLine
		wantErr   bool
	}{
		{
			name:     "valid sda line",
			line:     "   8       0 sda 12345 678 901234 567 89012 345 678901 234 5 6789 1234",
			wantName: "sda",
			wantStats: parsedDiskLine{
				readsCompleted:   12345,
				readsMerged:      678,
				sectorsRead:      901234,
				readTimeMs:       567,
				writesCompleted:  89012,
				writesMerged:     345,
				sectorsWritten:   678901,
				writeTimeMs:      234,
				ioInProgress:     5,
				ioTimeMs:         6789,
				weightedIOTimeMs: 1234,
			},
			wantErr: false,
		},
		{
			name:     "valid nvme line",
			line:     "259       0 nvme0n1 100 200 300 400 500 600 700 800 9 1000 1100",
			wantName: "nvme0n1",
			wantStats: parsedDiskLine{
				readsCompleted:   100,
				readsMerged:      200,
				sectorsRead:      300,
				readTimeMs:       400,
				writesCompleted:  500,
				writesMerged:     600,
				sectorsWritten:   700,
				writeTimeMs:      800,
				ioInProgress:     9,
				ioTimeMs:         1000,
				weightedIOTimeMs: 1100,
			},
			wantErr: false,
		},
		{
			name:    "insufficient fields",
			line:    "8 0 sda 123 456",
			wantErr: true,
		},
		{
			name:    "invalid number",
			line:    "8 0 sda abc 200 300 400 500 600 700 800 9 1000 1100",
			wantErr: true,
		},
		{
			name:    "empty device name",
			line:    "8 0  123 200 300 400 500 600 700 800 9 1000 1100",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotStats, err := parseDiskstatsLine(tt.line)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDiskstatsLine() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if gotName != tt.wantName {
					t.Errorf("parseDiskstatsLine() name = %q, want %q", gotName, tt.wantName)
				}
				if gotStats != tt.wantStats {
					t.Errorf("parseDiskstatsLine() stats = %+v, want %+v", gotStats, tt.wantStats)
				}
			}
		})
	}
}

func TestIsPhysicalDisk(t *testing.T) {
	tests := []struct {
		name     string
		disk     string
		expected bool
	}{
		// SCSI/SATA disks
		{"sda", "sda", true},
		{"sdb", "sdb", true},
		{"sda1 partition", "sda1", false},
		{"sda10 partition", "sda10", false},

		// IDE disks
		{"hda", "hda", true},
		{"hdb", "hdb", true},
		{"hda1 partition", "hda1", false},

		// VirtIO disks
		{"vda", "vda", true},
		{"vdb", "vdb", true},
		{"vda1 partition", "vda1", false},

		// Xen virtual disks
		{"xvda", "xvda", true},
		{"xvdb", "xvdb", true},
		{"xvda1 partition", "xvda1", false},

		// NVMe disks
		{"nvme0n1", "nvme0n1", true},
		{"nvme1n1", "nvme1n1", true},
		{"nvme0n1p1 partition", "nvme0n1p1", false},
		{"nvme0n1p10 partition", "nvme0n1p10", false},

		// MMC/SD cards
		{"mmcblk0", "mmcblk0", true},
		{"mmcblk1", "mmcblk1", true},
		{"mmcblk0p1 partition", "mmcblk0p1", false},

		// Loop devices
		{"loop0", "loop0", true},
		{"loop1", "loop1", true},

		// Other non-physical
		{"dm-0", "dm-0", false},
		{"ram0", "ram0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isPhysicalDisk(tt.disk)
			if got != tt.expected {
				t.Errorf("isPhysicalDisk(%q) = %v, want %v", tt.disk, got, tt.expected)
			}
		})
	}
}

func TestDiskIOReaderWithMockFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create mock /proc/diskstats
	diskstatsContent := `   8       0 sda 12345 678 901234 567 89012 345 678901 234 5 6789 1234
   8       1 sda1 1000 100 2000 50 500 50 1000 25 0 100 125
 259       0 nvme0n1 100 200 300 400 500 600 700 800 9 1000 1100
 259       1 nvme0n1p1 50 100 150 200 250 300 350 400 0 500 550
   7       0 loop0 10 0 20 5 0 0 0 0 0 5 5
`
	if err := os.WriteFile(filepath.Join(tmpDir, "diskstats"), []byte(diskstatsContent), 0644); err != nil {
		t.Fatalf("failed to write mock diskstats: %v", err)
	}

	reader := &diskIOReader{
		prevStats:         make(map[string]rawDiskStats),
		procDiskstatsPath: filepath.Join(tmpDir, "diskstats"),
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	// Should have 3 physical disks (sda, nvme0n1, loop0)
	if len(stats.Disks) != 3 {
		t.Errorf("got %d disks, want 3", len(stats.Disks))
	}

	// Check sda
	sda, ok := stats.Disks["sda"]
	if !ok {
		t.Fatal("missing sda disk")
	}
	if sda.ReadsCompleted != 12345 {
		t.Errorf("sda.ReadsCompleted = %d, want 12345", sda.ReadsCompleted)
	}
	if sda.SectorsRead != 901234 {
		t.Errorf("sda.SectorsRead = %d, want 901234", sda.SectorsRead)
	}

	// Check nvme0n1
	nvme, ok := stats.Disks["nvme0n1"]
	if !ok {
		t.Fatal("missing nvme0n1 disk")
	}
	if nvme.WritesCompleted != 500 {
		t.Errorf("nvme0n1.WritesCompleted = %d, want 500", nvme.WritesCompleted)
	}

	// Check loop0
	loop, ok := stats.Disks["loop0"]
	if !ok {
		t.Fatal("missing loop0 disk")
	}
	if loop.ReadsCompleted != 10 {
		t.Errorf("loop0.ReadsCompleted = %d, want 10", loop.ReadsCompleted)
	}

	// Partitions should be filtered
	if _, ok := stats.Disks["sda1"]; ok {
		t.Error("partition sda1 should be filtered")
	}
	if _, ok := stats.Disks["nvme0n1p1"]; ok {
		t.Error("partition nvme0n1p1 should be filtered")
	}
}

func TestDiskIOReaderRateCalculation(t *testing.T) {
	tmpDir := t.TempDir()
	diskstatsPath := filepath.Join(tmpDir, "diskstats")

	// First read with initial values
	diskstatsContent1 := `   8       0 sda 1000 100 2000 50 500 50 1000 25 0 100 125
`
	if err := os.WriteFile(diskstatsPath, []byte(diskstatsContent1), 0644); err != nil {
		t.Fatalf("failed to write mock diskstats: %v", err)
	}

	reader := &diskIOReader{
		prevStats:         make(map[string]rawDiskStats),
		procDiskstatsPath: diskstatsPath,
	}

	// First read - no rates available yet
	stats1, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("first ReadStats() error = %v", err)
	}

	sda := stats1.Disks["sda"]
	if sda.ReadBytesPerSec != 0.0 {
		t.Errorf("first read: ReadBytesPerSec = %f, want 0.0", sda.ReadBytesPerSec)
	}
	if sda.ReadsPerSec != 0.0 {
		t.Errorf("first read: ReadsPerSec = %f, want 0.0", sda.ReadsPerSec)
	}

	// Wait and write new values
	time.Sleep(100 * time.Millisecond)

	// Second read with increased values
	// sectorsRead: 2000 -> 4000 (delta: 2000 sectors = 1024000 bytes)
	// readsCompleted: 1000 -> 1100 (delta: 100 reads)
	diskstatsContent2 := `   8       0 sda 1100 100 4000 50 600 50 2000 25 0 100 125
`
	if err := os.WriteFile(diskstatsPath, []byte(diskstatsContent2), 0644); err != nil {
		t.Fatalf("failed to write mock diskstats (2nd): %v", err)
	}

	stats2, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("second ReadStats() error = %v", err)
	}

	sda = stats2.Disks["sda"]
	if sda.ReadBytesPerSec <= 0 {
		t.Errorf("second read: ReadBytesPerSec = %f, want > 0", sda.ReadBytesPerSec)
	}
	if sda.WriteBytesPerSec <= 0 {
		t.Errorf("second read: WriteBytesPerSec = %f, want > 0", sda.WriteBytesPerSec)
	}
	if sda.ReadsPerSec <= 0 {
		t.Errorf("second read: ReadsPerSec = %f, want > 0", sda.ReadsPerSec)
	}
	if sda.WritesPerSec <= 0 {
		t.Errorf("second read: WritesPerSec = %f, want > 0", sda.WritesPerSec)
	}
}

func TestDiskIOReaderMissingFile(t *testing.T) {
	reader := &diskIOReader{
		prevStats:         make(map[string]rawDiskStats),
		procDiskstatsPath: "/nonexistent/diskstats",
	}

	_, err := reader.ReadStats()
	if err == nil {
		t.Error("ReadStats() should return error for missing file")
	}
}

func TestDiskIOReaderEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmpDir, "diskstats"), []byte(""), 0644); err != nil {
		t.Fatalf("failed to write mock diskstats: %v", err)
	}

	reader := &diskIOReader{
		prevStats:         make(map[string]rawDiskStats),
		procDiskstatsPath: filepath.Join(tmpDir, "diskstats"),
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	if len(stats.Disks) != 0 {
		t.Errorf("got %d disks from empty file, want 0", len(stats.Disks))
	}
}

func TestDiskIOReaderCalculateRate(t *testing.T) {
	reader := newDiskIOReader()

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

func TestDiskIOReaderCalculateByteRate(t *testing.T) {
	reader := newDiskIOReader()

	tests := []struct {
		name        string
		prevSectors uint64
		currSectors uint64
		elapsed     float64
		expected    float64
	}{
		{
			name:        "normal increase",
			prevSectors: 1000,
			currSectors: 2000,
			elapsed:     1.0,
			expected:    512000.0, // 1000 sectors * 512 bytes
		},
		{
			name:        "no change",
			prevSectors: 1000,
			currSectors: 1000,
			elapsed:     1.0,
			expected:    0.0,
		},
		{
			name:        "counter wrap-around",
			prevSectors: 2000,
			currSectors: 1000,
			elapsed:     1.0,
			expected:    0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reader.calculateByteRate(tt.prevSectors, tt.currSectors, tt.elapsed)
			if got != tt.expected {
				t.Errorf("calculateByteRate() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNewDiskIOReader(t *testing.T) {
	reader := newDiskIOReader()

	if reader.procDiskstatsPath != "/proc/diskstats" {
		t.Errorf("procDiskstatsPath = %q, want %q", reader.procDiskstatsPath, "/proc/diskstats")
	}
	if reader.prevStats == nil {
		t.Error("prevStats should be initialized")
	}
}

func TestDiskIOReaderMalformedLines(t *testing.T) {
	tmpDir := t.TempDir()

	// Mix valid and invalid lines
	diskstatsContent := `   8       0 sda 1000 100 2000 50 500 50 1000 25 0 100 125
malformed line
   8       1 sdb 2000 200 4000 100 1000 100 2000 50 0 200 250
short line 1 2 3
`
	if err := os.WriteFile(filepath.Join(tmpDir, "diskstats"), []byte(diskstatsContent), 0644); err != nil {
		t.Fatalf("failed to write mock diskstats: %v", err)
	}

	reader := &diskIOReader{
		prevStats:         make(map[string]rawDiskStats),
		procDiskstatsPath: filepath.Join(tmpDir, "diskstats"),
	}

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats() error = %v", err)
	}

	// Should have 2 valid disks
	if len(stats.Disks) != 2 {
		t.Errorf("got %d disks, want 2", len(stats.Disks))
	}

	if _, ok := stats.Disks["sda"]; !ok {
		t.Error("missing sda disk")
	}
	if _, ok := stats.Disks["sdb"]; !ok {
		t.Error("missing sdb disk")
	}
}
