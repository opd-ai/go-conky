package platform

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

// linuxCPUProvider implements CPUProvider for Linux systems by reading /proc/stat and /proc/cpuinfo.
type linuxCPUProvider struct {
	mu             sync.Mutex
	prevStats      map[int]cpuTimes
	procStatPath   string
	procInfoPath   string
	procLoadavgPath string
}

// cpuTimes stores raw CPU time values from /proc/stat.
type cpuTimes struct {
	user    uint64
	nice    uint64
	system  uint64
	idle    uint64
	iowait  uint64
	irq     uint64
	softirq uint64
	steal   uint64
}

func newLinuxCPUProvider() *linuxCPUProvider {
	return &linuxCPUProvider{
		prevStats:       make(map[int]cpuTimes),
		procStatPath:    "/proc/stat",
		procInfoPath:    "/proc/cpuinfo",
		procLoadavgPath: "/proc/loadavg",
	}
}

func (c *linuxCPUProvider) Usage() ([]float64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	currentStats, err := c.readProcStat()
	if err != nil {
		return nil, err
	}

	var usages []float64
	for cpuNum, current := range currentStats {
		prev, exists := c.prevStats[cpuNum]
		c.prevStats[cpuNum] = current

		if !exists {
			usages = append(usages, 0)
			continue
		}

		usage := c.calculateUsage(prev, current)
		usages = append(usages, usage)
	}

	return usages, nil
}

func (c *linuxCPUProvider) TotalUsage() (float64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Read aggregate CPU line (cpu) from /proc/stat
	file, err := os.Open(c.procStatPath)
	if err != nil {
		return 0, fmt.Errorf("opening %s: %w", c.procStatPath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return 0, fmt.Errorf("reading %s: empty file", c.procStatPath)
	}

	line := scanner.Text()
	fields := strings.Fields(line)
	if len(fields) < 8 || fields[0] != "cpu" {
		return 0, fmt.Errorf("unexpected format in %s", c.procStatPath)
	}

	current := cpuTimes{
		user:    parseUint64(fields[1]),
		nice:    parseUint64(fields[2]),
		system:  parseUint64(fields[3]),
		idle:    parseUint64(fields[4]),
		iowait:  parseUint64(fields[5]),
		irq:     parseUint64(fields[6]),
		softirq: parseUint64(fields[7]),
	}
	if len(fields) > 8 {
		current.steal = parseUint64(fields[8])
	}

	// Use -1 as key for aggregate CPU stats
	prev, exists := c.prevStats[-1]
	c.prevStats[-1] = current

	if !exists {
		return 0, nil
	}

	return c.calculateUsage(prev, current), nil
}

func (c *linuxCPUProvider) Frequency() ([]float64, error) {
	// Read CPU frequencies from /proc/cpuinfo
	file, err := os.Open(c.procInfoPath)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", c.procInfoPath, err)
	}
	defer file.Close()

	var frequencies []float64
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "cpu MHz") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				freqStr := strings.TrimSpace(parts[1])
				if freq, err := strconv.ParseFloat(freqStr, 64); err == nil {
					frequencies = append(frequencies, freq)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning %s: %w", c.procInfoPath, err)
	}

	return frequencies, nil
}

func (c *linuxCPUProvider) Info() (*CPUInfo, error) {
	file, err := os.Open(c.procInfoPath)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", c.procInfoPath, err)
	}
	defer file.Close()

	info := &CPUInfo{}
	var coreCount int
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "model name":
			if info.Model == "" {
				info.Model = value
			}
		case "vendor_id":
			if info.Vendor == "" {
				info.Vendor = value
			}
		case "cpu cores":
			if cores, err := strconv.Atoi(value); err == nil && info.Cores == 0 {
				info.Cores = cores
			}
		case "siblings":
			if siblings, err := strconv.Atoi(value); err == nil && info.Threads == 0 {
				info.Threads = siblings
			}
		case "cache size":
			if info.CacheSize == 0 {
				// Parse cache size (e.g., "8192 KB")
				cacheParts := strings.Fields(value)
				if len(cacheParts) >= 2 {
					if size, err := strconv.ParseInt(cacheParts[0], 10, 64); err == nil {
						// Convert KB to bytes
						info.CacheSize = size * 1024
					}
				}
			}
		case "processor":
			coreCount++
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning %s: %w", c.procInfoPath, err)
	}

	// If cores and threads were not found, use processor count
	if info.Cores == 0 {
		info.Cores = coreCount
	}
	if info.Threads == 0 {
		info.Threads = coreCount
	}

	return info, nil
}

func (c *linuxCPUProvider) LoadAverage() (float64, float64, float64, error) {
	file, err := os.Open(c.procLoadavgPath)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("opening %s: %w", c.procLoadavgPath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return 0, 0, 0, fmt.Errorf("reading %s: empty file", c.procLoadavgPath)
	}

	fields := strings.Fields(scanner.Text())
	if len(fields) < 3 {
		return 0, 0, 0, fmt.Errorf("unexpected format in %s", c.procLoadavgPath)
	}

	load1, err1 := strconv.ParseFloat(fields[0], 64)
	load5, err2 := strconv.ParseFloat(fields[1], 64)
	load15, err3 := strconv.ParseFloat(fields[2], 64)

	if err1 != nil || err2 != nil || err3 != nil {
		return 0, 0, 0, fmt.Errorf("parsing load averages from %s", c.procLoadavgPath)
	}

	return load1, load5, load15, nil
}

// readProcStat reads and parses /proc/stat for per-CPU times.
// Returns a map of CPU number to cpuTimes (excluding the aggregate "cpu" line).
func (c *linuxCPUProvider) readProcStat() (map[int]cpuTimes, error) {
	file, err := os.Open(c.procStatPath)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", c.procStatPath, err)
	}
	defer file.Close()

	stats := make(map[int]cpuTimes)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 8 {
			continue
		}

		// Skip aggregate "cpu" line, only process "cpu0", "cpu1", etc.
		if fields[0] == "cpu" {
			continue
		}

		if !strings.HasPrefix(fields[0], "cpu") {
			break
		}

		cpuNum, err := strconv.Atoi(strings.TrimPrefix(fields[0], "cpu"))
		if err != nil {
			continue
		}

		current := cpuTimes{
			user:    parseUint64(fields[1]),
			nice:    parseUint64(fields[2]),
			system:  parseUint64(fields[3]),
			idle:    parseUint64(fields[4]),
			iowait:  parseUint64(fields[5]),
			irq:     parseUint64(fields[6]),
			softirq: parseUint64(fields[7]),
		}
		if len(fields) > 8 {
			current.steal = parseUint64(fields[8])
		}

		stats[cpuNum] = current
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning %s: %w", c.procStatPath, err)
	}

	return stats, nil
}

// calculateUsage calculates CPU usage percentage from two cpuTimes snapshots.
func (c *linuxCPUProvider) calculateUsage(prev, current cpuTimes) float64 {
	prevTotal := prev.user + prev.nice + prev.system + prev.idle + prev.iowait + prev.irq + prev.softirq + prev.steal
	currentTotal := current.user + current.nice + current.system + current.idle + current.iowait + current.irq + current.softirq + current.steal

	prevIdle := prev.idle + prev.iowait
	currentIdle := current.idle + current.iowait

	totalDelta := currentTotal - prevTotal
	idleDelta := currentIdle - prevIdle

	if totalDelta == 0 {
		return 0
	}

	return 100.0 * float64(totalDelta-idleDelta) / float64(totalDelta)
}

func parseUint64(s string) uint64 {
	v, _ := strconv.ParseUint(s, 10, 64)
	return v
}
