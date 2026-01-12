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

// rawInterfaceStats stores raw counters for rate calculation.
type rawInterfaceStats struct {
	rxBytes   uint64
	rxPackets uint64
	rxErrors  uint64
	rxDropped uint64
	txBytes   uint64
	txPackets uint64
	txErrors  uint64
	txDropped uint64
}

// networkReader reads network interface statistics from /proc filesystem.
type networkReader struct {
	mu             sync.Mutex
	prevStats      map[string]rawInterfaceStats
	prevTime       time.Time
	procNetDevPath string
}

// newNetworkReader creates a new networkReader with default paths.
func newNetworkReader() *networkReader {
	return &networkReader{
		prevStats:      make(map[string]rawInterfaceStats),
		procNetDevPath: "/proc/net/dev",
	}
}

// ReadStats reads current network interface statistics from /proc/net/dev.
func (r *networkReader) ReadStats() (NetworkStats, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	currentStats, err := r.readProcNetDev()
	if err != nil {
		return NetworkStats{}, err
	}

	now := time.Now()
	elapsed := now.Sub(r.prevTime).Seconds()

	stats := NetworkStats{
		Interfaces: make(map[string]InterfaceStats, len(currentStats)),
	}

	for name, curr := range currentStats {
		ifStats := InterfaceStats{
			Name:      name,
			RxBytes:   curr.rxBytes,
			RxPackets: curr.rxPackets,
			RxErrors:  curr.rxErrors,
			RxDropped: curr.rxDropped,
			TxBytes:   curr.txBytes,
			TxPackets: curr.txPackets,
			TxErrors:  curr.txErrors,
			TxDropped: curr.txDropped,
		}

		// Calculate rates if we have previous data and elapsed time is valid
		if prev, ok := r.prevStats[name]; ok && elapsed > 0 {
			ifStats.RxBytesPerSec = r.calculateRate(prev.rxBytes, curr.rxBytes, elapsed)
			ifStats.TxBytesPerSec = r.calculateRate(prev.txBytes, curr.txBytes, elapsed)
		}

		stats.Interfaces[name] = ifStats

		// Update totals
		stats.TotalRxBytes += curr.rxBytes
		stats.TotalTxBytes += curr.txBytes
		stats.TotalRxBytesPerSec += ifStats.RxBytesPerSec
		stats.TotalTxBytesPerSec += ifStats.TxBytesPerSec
	}

	// Store current stats for next rate calculation
	r.prevStats = currentStats
	r.prevTime = now

	return stats, nil
}

// readProcNetDev parses /proc/net/dev and returns raw interface statistics.
func (r *networkReader) readProcNetDev() (map[string]rawInterfaceStats, error) {
	file, err := os.Open(r.procNetDevPath)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", r.procNetDevPath, err)
	}
	defer file.Close()

	result := make(map[string]rawInterfaceStats)
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		// Skip header lines (first two lines)
		if lineNum <= 2 {
			continue
		}

		line := scanner.Text()
		name, stats, err := parseNetDevLine(line)
		if err != nil {
			continue // Skip malformed lines
		}

		result[name] = stats
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning %s: %w", r.procNetDevPath, err)
	}

	return result, nil
}

// parseNetDevLine parses a single line from /proc/net/dev.
// Format: "iface: rxbytes rxpackets rxerrs rxdrop rxfifo rxframe rxcompressed rxmulticast
//
//	txbytes txpackets txerrs txdrop txfifo txcolls txcarrier txcompressed"
func parseNetDevLine(line string) (string, rawInterfaceStats, error) {
	// Split by colon to separate interface name from stats
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return "", rawInterfaceStats{}, fmt.Errorf("invalid line format: no colon separator")
	}

	name := strings.TrimSpace(parts[0])
	if name == "" {
		return "", rawInterfaceStats{}, fmt.Errorf("empty interface name")
	}

	fields := strings.Fields(parts[1])
	if len(fields) < 16 {
		return "", rawInterfaceStats{}, fmt.Errorf("insufficient fields: got %d, need 16", len(fields))
	}

	values := make([]uint64, 16)
	for i := 0; i < 16; i++ {
		v, err := strconv.ParseUint(fields[i], 10, 64)
		if err != nil {
			return "", rawInterfaceStats{}, fmt.Errorf("parsing field %d: %w", i, err)
		}
		values[i] = v
	}

	// Receive fields: bytes(0), packets(1), errs(2), drop(3), fifo(4), frame(5), compressed(6), multicast(7)
	// Transmit fields: bytes(8), packets(9), errs(10), drop(11), fifo(12), colls(13), carrier(14), compressed(15)
	return name, rawInterfaceStats{
		rxBytes:   values[0],
		rxPackets: values[1],
		rxErrors:  values[2],
		rxDropped: values[3],
		txBytes:   values[8],
		txPackets: values[9],
		txErrors:  values[10],
		txDropped: values[11],
	}, nil
}

// calculateRate calculates the rate of change per second.
// Returns 0 if counter wrapped around (new < old).
func (r *networkReader) calculateRate(prev, curr uint64, elapsed float64) float64 {
	if curr < prev {
		// Counter wrapped around or interface was reset
		return 0.0
	}
	if elapsed <= 0 {
		return 0.0
	}
	return float64(curr-prev) / elapsed
}
