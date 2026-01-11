package monitor

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

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

// total returns the total CPU time.
func (c cpuTimes) total() uint64 {
	return c.user + c.nice + c.system + c.idle + c.iowait + c.irq + c.softirq + c.steal
}

// idleTime returns the idle CPU time.
func (c cpuTimes) idleTime() uint64 {
	return c.idle + c.iowait
}

// cpuReader reads CPU statistics from /proc filesystem.
type cpuReader struct {
	prevTimes     cpuTimes
	prevCoreTimes []cpuTimes
	procStatPath  string
	procInfoPath  string
}

// newCPUReader creates a new cpuReader with default paths.
func newCPUReader() *cpuReader {
	return &cpuReader{
		procStatPath: "/proc/stat",
		procInfoPath: "/proc/cpuinfo",
	}
}

// ReadStats reads current CPU statistics.
func (r *cpuReader) ReadStats() (CPUStats, error) {
	currentTimes, coreTimes, err := r.readProcStat()
	if err != nil {
		return CPUStats{}, fmt.Errorf("reading /proc/stat: %w", err)
	}

	modelName, freq, err := r.readCPUInfo()
	if err != nil {
		// Non-fatal: continue without CPU info
		modelName = "Unknown"
		freq = 0.0
	}

	stats := CPUStats{
		UsagePercent: r.calculateUsage(r.prevTimes, currentTimes),
		Cores:        r.calculateCoreUsage(r.prevCoreTimes, coreTimes),
		CPUCount:     len(coreTimes),
		ModelName:    modelName,
		Frequency:    freq,
	}

	r.prevTimes = currentTimes
	r.prevCoreTimes = coreTimes

	return stats, nil
}

// readProcStat reads and parses /proc/stat for CPU times.
func (r *cpuReader) readProcStat() (cpuTimes, []cpuTimes, error) {
	file, err := os.Open(r.procStatPath)
	if err != nil {
		return cpuTimes{}, nil, fmt.Errorf("opening %s: %w", r.procStatPath, err)
	}
	defer file.Close()

	var totalTimes cpuTimes
	var coreTimes []cpuTimes

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 8 {
			continue
		}

		if fields[0] == "cpu" {
			t, err := parseCPULine(fields[1:])
			if err != nil {
				return cpuTimes{}, nil, fmt.Errorf("parsing cpu line: %w", err)
			}
			totalTimes = t
		} else if strings.HasPrefix(fields[0], "cpu") {
			t, err := parseCPULine(fields[1:])
			if err != nil {
				continue // Skip malformed core lines
			}
			coreTimes = append(coreTimes, t)
		}
	}

	if err := scanner.Err(); err != nil {
		return cpuTimes{}, nil, fmt.Errorf("scanning %s: %w", r.procStatPath, err)
	}

	return totalTimes, coreTimes, nil
}

// parseCPULine parses a single CPU line from /proc/stat.
func parseCPULine(fields []string) (cpuTimes, error) {
	if len(fields) < 7 {
		return cpuTimes{}, fmt.Errorf("insufficient fields: got %d, need at least 7", len(fields))
	}

	values := make([]uint64, 8)
	for i := 0; i < len(values) && i < len(fields); i++ {
		v, err := strconv.ParseUint(fields[i], 10, 64)
		if err != nil {
			return cpuTimes{}, fmt.Errorf("parsing field %d: %w", i, err)
		}
		values[i] = v
	}

	return cpuTimes{
		user:    values[0],
		nice:    values[1],
		system:  values[2],
		idle:    values[3],
		iowait:  values[4],
		irq:     values[5],
		softirq: values[6],
		steal:   values[7],
	}, nil
}

// calculateUsage calculates CPU usage percentage from time deltas.
func (r *cpuReader) calculateUsage(prev, curr cpuTimes) float64 {
	totalDelta := curr.total() - prev.total()
	if totalDelta == 0 {
		return 0.0
	}
	idleDelta := curr.idleTime() - prev.idleTime()
	usage := float64(totalDelta-idleDelta) / float64(totalDelta) * 100.0
	if usage < 0 {
		return 0.0
	}
	if usage > 100 {
		return 100.0
	}
	return usage
}

// calculateCoreUsage calculates per-core CPU usage percentages.
func (r *cpuReader) calculateCoreUsage(prev, curr []cpuTimes) []float64 {
	result := make([]float64, len(curr))
	for i, c := range curr {
		var p cpuTimes
		if i < len(prev) {
			p = prev[i]
		}
		result[i] = r.calculateUsage(p, c)
	}
	return result
}

// readCPUInfo reads CPU model name and frequency from /proc/cpuinfo.
func (r *cpuReader) readCPUInfo() (string, float64, error) {
	file, err := os.Open(r.procInfoPath)
	if err != nil {
		return "", 0, fmt.Errorf("opening %s: %w", r.procInfoPath, err)
	}
	defer file.Close()

	var modelName string
	var freq float64

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
			if modelName == "" {
				modelName = value
			}
		case "cpu MHz":
			if freq == 0 {
				f, err := strconv.ParseFloat(value, 64)
				if err == nil {
					freq = f
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", 0, fmt.Errorf("scanning %s: %w", r.procInfoPath, err)
	}

	return modelName, freq, nil
}
