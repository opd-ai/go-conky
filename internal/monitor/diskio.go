package monitor

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// sectorSize is the standard sector size in bytes for Linux disk I/O.
const sectorSize = 512

// rawDiskStats stores raw counters for rate calculation.
type rawDiskStats struct {
	readsCompleted  uint64
	sectorsRead     uint64
	writesCompleted uint64
	sectorsWritten  uint64
}

// diskIOReader reads disk I/O statistics from /proc/diskstats.
type diskIOReader struct {
	mu               sync.Mutex
	prevStats        map[string]rawDiskStats
	prevTime         time.Time
	procDiskstatsPath string
}

// newDiskIOReader creates a new diskIOReader with default paths.
func newDiskIOReader() *diskIOReader {
	return &diskIOReader{
		prevStats:        make(map[string]rawDiskStats),
		procDiskstatsPath: "/proc/diskstats",
	}
}

// ReadStats reads current disk I/O statistics from /proc/diskstats.
func (r *diskIOReader) ReadStats() (DiskIOStats, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	currentStats, err := r.readProcDiskstats()
	if err != nil {
		return DiskIOStats{}, err
	}

	now := time.Now()
	elapsed := now.Sub(r.prevTime).Seconds()

	stats := DiskIOStats{
		Disks: make(map[string]DiskStats, len(currentStats)),
	}

	for name, curr := range currentStats {
		diskStats := curr.toDiskStats(name)

		// Calculate rates if we have previous data and elapsed time is valid
		if prev, ok := r.prevStats[name]; ok && elapsed > 0 {
			diskStats.ReadBytesPerSec = r.calculateByteRate(prev.sectorsRead, curr.sectorsRead, elapsed)
			diskStats.WriteBytesPerSec = r.calculateByteRate(prev.sectorsWritten, curr.sectorsWritten, elapsed)
			diskStats.ReadsPerSec = r.calculateRate(prev.readsCompleted, curr.readsCompleted, elapsed)
			diskStats.WritesPerSec = r.calculateRate(prev.writesCompleted, curr.writesCompleted, elapsed)
		}

		stats.Disks[name] = diskStats
	}

	// Store current stats for next rate calculation
	// Convert parsed lines to raw stats for rate calculation
	rawStats := make(map[string]rawDiskStats, len(currentStats))
	for name, curr := range currentStats {
		rawStats[name] = rawDiskStats{
			readsCompleted:  curr.readsCompleted,
			sectorsRead:     curr.sectorsRead,
			writesCompleted: curr.writesCompleted,
			sectorsWritten:  curr.sectorsWritten,
		}
	}
	r.prevStats = rawStats
	r.prevTime = now

	return stats, nil
}

// parsedDiskLine represents parsed data from a /proc/diskstats line.
type parsedDiskLine struct {
	readsCompleted   uint64
	readsMerged      uint64
	sectorsRead      uint64
	readTimeMs       uint64
	writesCompleted  uint64
	writesMerged     uint64
	sectorsWritten   uint64
	writeTimeMs      uint64
	ioInProgress     uint64
	ioTimeMs         uint64
	weightedIOTimeMs uint64
}

// toDiskStats converts parsedDiskLine to DiskStats.
func (p parsedDiskLine) toDiskStats(name string) DiskStats {
	return DiskStats{
		Name:             name,
		ReadsCompleted:   p.readsCompleted,
		ReadsMerged:      p.readsMerged,
		SectorsRead:      p.sectorsRead,
		ReadTimeMs:       p.readTimeMs,
		WritesCompleted:  p.writesCompleted,
		WritesMerged:     p.writesMerged,
		SectorsWritten:   p.sectorsWritten,
		WriteTimeMs:      p.writeTimeMs,
		IOInProgress:     p.ioInProgress,
		IOTimeMs:         p.ioTimeMs,
		WeightedIOTimeMs: p.weightedIOTimeMs,
	}
}

// readProcDiskstats parses /proc/diskstats and returns disk statistics.
func (r *diskIOReader) readProcDiskstats() (map[string]parsedDiskLine, error) {
	file, err := os.Open(r.procDiskstatsPath)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", r.procDiskstatsPath, err)
	}
	defer file.Close()

	result := make(map[string]parsedDiskLine)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		name, stats, err := parseDiskstatsLine(line)
		if err != nil {
			continue // Skip malformed lines
		}

		// Only include physical disks, not partitions
		// This filters based on naming conventions
		if !isPhysicalDisk(name) {
			continue
		}

		result[name] = stats
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning %s: %w", r.procDiskstatsPath, err)
	}

	return result, nil
}

// parseDiskstatsLine parses a single line from /proc/diskstats.
// Format: major minor name reads_completed reads_merged sectors_read read_time
//         writes_completed writes_merged sectors_written write_time io_in_progress
//         io_time weighted_io_time
func parseDiskstatsLine(line string) (string, parsedDiskLine, error) {
	fields := strings.Fields(line)
	if len(fields) < 14 {
		return "", parsedDiskLine{}, fmt.Errorf("insufficient fields: got %d, need at least 14", len(fields))
	}

	name := fields[2]
	if name == "" {
		return "", parsedDiskLine{}, fmt.Errorf("empty device name")
	}

	values := make([]uint64, 11)
	for i := 0; i < 11; i++ {
		v, err := strconv.ParseUint(fields[i+3], 10, 64)
		if err != nil {
			return "", parsedDiskLine{}, fmt.Errorf("parsing field %d: %w", i+3, err)
		}
		values[i] = v
	}

	return name, parsedDiskLine{
		readsCompleted:   values[0],
		readsMerged:      values[1],
		sectorsRead:      values[2],
		readTimeMs:       values[3],
		writesCompleted:  values[4],
		writesMerged:     values[5],
		sectorsWritten:   values[6],
		writeTimeMs:      values[7],
		ioInProgress:     values[8],
		ioTimeMs:         values[9],
		weightedIOTimeMs: values[10],
	}, nil
}

// isPhysicalDisk returns true if the device name represents a physical disk.
// It filters out partitions (e.g., sda1) and keeps only whole disks (e.g., sda).
func isPhysicalDisk(name string) bool {
	// Common disk naming patterns:
	// sd[a-z] - SCSI/SATA disks
	// hd[a-z] - IDE disks
	// vd[a-z] - VirtIO disks
	// nvme[0-9]+n[0-9]+ - NVMe disks (without partition suffix p[0-9]+)
	// xvd[a-z] - Xen virtual disks
	// mmcblk[0-9]+ - MMC/SD cards (without partition suffix p[0-9]+)
	// loop[0-9]+ - Loop devices

	// Check for standard disk patterns without partition numbers
	if len(name) >= 3 {
		prefix := name[:2]
		switch prefix {
		case "sd", "hd", "vd":
			// Valid if name is exactly 3 chars (e.g., "sda")
			return len(name) == 3
		case "xv":
			// xvd[a-z] pattern
			if len(name) >= 4 && name[2] == 'd' {
				return len(name) == 4
			}
		}
	}

	// NVMe drives: nvme0n1, nvme1n1, etc. (no partition suffix like p1, p2)
	if strings.HasPrefix(name, "nvme") && strings.Contains(name, "n") {
		// Check if it ends with a digit after 'n' and has no 'p' partition suffix
		if !strings.Contains(name, "p") {
			return true
		}
		return false
	}

	// MMC/SD cards: mmcblk0, mmcblk1, etc. (no partition suffix like p1, p2)
	if strings.HasPrefix(name, "mmcblk") {
		if !strings.Contains(name[6:], "p") {
			return true
		}
		return false
	}

	// Loop devices: loop0, loop1, etc.
	if strings.HasPrefix(name, "loop") {
		return true
	}

	return false
}

// calculateRate calculates the rate of change per second.
// Returns 0 if counter wrapped around (new < old).
func (r *diskIOReader) calculateRate(prev, curr uint64, elapsed float64) float64 {
	if curr < prev {
		// Counter wrapped around or device was reset
		return 0.0
	}
	if elapsed <= 0 {
		return 0.0
	}
	return float64(curr-prev) / elapsed
}

// calculateByteRate calculates the byte rate from sector counts.
func (r *diskIOReader) calculateByteRate(prevSectors, currSectors uint64, elapsed float64) float64 {
	if currSectors < prevSectors {
		return 0.0
	}
	if elapsed <= 0 {
		return 0.0
	}
	sectorDelta := currSectors - prevSectors
	return float64(sectorDelta*sectorSize) / elapsed
}
