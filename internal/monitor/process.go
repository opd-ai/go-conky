package monitor

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// TopProcessCount is the default number of top processes to track.
const TopProcessCount = 10

// Constants for parsing /proc/[pid]/stat fields.
// The field indices are relative to the fields after the command name (comm).
// Field numbers in comments are from the proc(5) man page.
const (
	// statMinFields is the minimum number of fields required after comm.
	statMinFields = 22
	// statFieldState is the process state (field 3 in proc(5)).
	statFieldState = 0
	// statFieldUtime is user mode CPU time in clock ticks (field 14).
	statFieldUtime = 11
	// statFieldStime is kernel mode CPU time in clock ticks (field 15).
	statFieldStime = 12
	// statFieldNumThreads is the number of threads (field 20).
	statFieldNumThreads = 17
	// statFieldStarttime is process start time in clock ticks (field 22).
	statFieldStarttime = 19
	// statFieldVsize is virtual memory size in bytes (field 23).
	statFieldVsize = 20
	// statFieldRss is resident set size in pages (field 24).
	statFieldRss = 21
	// pageSize is the memory page size in bytes.
	// TODO: Consider using syscall.Getpagesize() for portability. Currently hardcoded
	// to 4096 which is correct for x86/x86_64 Linux but may differ on ARM64 (64KB pages).
	pageSize = 4096
)

// processReader reads process statistics from /proc filesystem.
type processReader struct {
	mu               sync.Mutex
	procPath         string
	lastCPUTimes     map[int]cpuTime // PID -> last CPU time for rate calculation
	lastTotalCPU     uint64          // Total CPU time at last measurement
	totalMemoryBytes uint64          // Total system memory in bytes
	clkTck           float64         // Clock ticks per second (typically 100)
}

// cpuTime stores CPU time information for rate calculation.
type cpuTime struct {
	utime uint64 // User mode time
	stime uint64 // Kernel mode time
	total uint64 // Combined time
}

// newProcessReader creates a new processReader with default paths.
func newProcessReader() *processReader {
	return &processReader{
		procPath:     "/proc",
		lastCPUTimes: make(map[int]cpuTime),
		clkTck:       100.0, // Standard Linux value for USER_HZ
	}
}

// ReadStats reads current process statistics from /proc.
func (r *processReader) ReadStats() (ProcessStats, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	stats := ProcessStats{
		TopCPU: make([]ProcessInfo, 0, TopProcessCount),
		TopMem: make([]ProcessInfo, 0, TopProcessCount),
	}

	// Read total memory for percentage calculation
	if r.totalMemoryBytes == 0 {
		if err := r.readTotalMemory(); err != nil {
			return stats, fmt.Errorf("reading total memory: %w", err)
		}
	}

	// Read total CPU time for rate calculation
	totalCPU, err := r.readTotalCPUTime()
	if err != nil {
		return stats, fmt.Errorf("reading total CPU time: %w", err)
	}

	// Get list of all PIDs
	entries, err := os.ReadDir(r.procPath)
	if err != nil {
		return stats, fmt.Errorf("reading %s: %w", r.procPath, err)
	}

	processes := make([]ProcessInfo, 0, len(entries))
	currentCPUTimes := make(map[int]cpuTime)
	cpuDelta := totalCPU - r.lastTotalCPU

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			// Not a PID directory
			continue
		}

		proc, ct, err := r.readProcess(pid, cpuDelta)
		if err != nil {
			// Process may have exited, skip it
			continue
		}

		currentCPUTimes[pid] = ct
		processes = append(processes, proc)

		// Count by state
		stats.TotalProcesses++
		stats.TotalThreads += proc.Threads
		switch proc.State {
		case "R":
			stats.RunningProcesses++
		case "S", "I":
			stats.SleepingProcesses++
		case "Z":
			stats.ZombieProcesses++
		case "T", "t":
			stats.StoppedProcesses++
		}
	}

	// Update cached values for next calculation
	r.lastCPUTimes = currentCPUTimes
	r.lastTotalCPU = totalCPU

	// Sort by CPU usage and get top N
	sort.Slice(processes, func(i, j int) bool {
		return processes[i].CPUPercent > processes[j].CPUPercent
	})
	for i := 0; i < len(processes) && i < TopProcessCount; i++ {
		stats.TopCPU = append(stats.TopCPU, processes[i])
	}

	// Sort by memory usage and get top N
	sort.Slice(processes, func(i, j int) bool {
		return processes[i].MemBytes > processes[j].MemBytes
	})
	for i := 0; i < len(processes) && i < TopProcessCount; i++ {
		stats.TopMem = append(stats.TopMem, processes[i])
	}

	return stats, nil
}

// readProcess reads information for a single process.
func (r *processReader) readProcess(pid int, cpuDelta uint64) (ProcessInfo, cpuTime, error) {
	proc := ProcessInfo{PID: pid}
	var ct cpuTime

	// Read /proc/[pid]/stat
	statPath := filepath.Join(r.procPath, strconv.Itoa(pid), "stat")
	statContent, err := os.ReadFile(statPath)
	if err != nil {
		return proc, ct, fmt.Errorf("reading stat: %w", err)
	}

	if err := r.parseProcessStat(&proc, &ct, string(statContent), cpuDelta); err != nil {
		return proc, ct, fmt.Errorf("parsing stat: %w", err)
	}

	return proc, ct, nil
}

// parseProcessStat parses /proc/[pid]/stat content.
// The format is: pid (comm) state ppid pgrp session tty_nr tpgid flags
// minflt cminflt majflt cmajflt utime stime cutime cstime priority nice
// num_threads itrealvalue starttime vsize rss ...
func (r *processReader) parseProcessStat(proc *ProcessInfo, ct *cpuTime, content string, cpuDelta uint64) error {
	// Find the command name between parentheses
	openParen := strings.IndexByte(content, '(')
	closeParen := strings.LastIndexByte(content, ')')
	if openParen == -1 || closeParen == -1 || closeParen <= openParen {
		return fmt.Errorf("invalid stat format: missing parentheses")
	}

	proc.Name = content[openParen+1 : closeParen]

	// Fields after the closing parenthesis
	fields := strings.Fields(content[closeParen+2:])
	if len(fields) < statMinFields {
		return fmt.Errorf("invalid stat format: not enough fields (got %d, need %d)", len(fields), statMinFields)
	}

	// Parse state (first field after comm)
	proc.State = fields[statFieldState]

	// Parse utime (user mode CPU time in clock ticks)
	utime, err := strconv.ParseUint(fields[statFieldUtime], 10, 64)
	if err != nil {
		return fmt.Errorf("parsing utime: %w", err)
	}

	// Parse stime (kernel mode CPU time in clock ticks)
	stime, err := strconv.ParseUint(fields[statFieldStime], 10, 64)
	if err != nil {
		return fmt.Errorf("parsing stime: %w", err)
	}

	// Parse num_threads (number of threads)
	threads, err := strconv.Atoi(fields[statFieldNumThreads])
	if err != nil {
		return fmt.Errorf("parsing num_threads: %w", err)
	}
	proc.Threads = threads

	// Parse starttime (process start time in clock ticks)
	starttime, err := strconv.ParseUint(fields[statFieldStarttime], 10, 64)
	if err != nil {
		return fmt.Errorf("parsing starttime: %w", err)
	}
	proc.StartTime = starttime

	// Parse vsize (virtual memory size in bytes)
	vsize, err := strconv.ParseUint(fields[statFieldVsize], 10, 64)
	if err != nil {
		return fmt.Errorf("parsing vsize: %w", err)
	}
	proc.VirtBytes = vsize

	// Parse rss (resident set size in pages, convert to bytes)
	rss, err := strconv.ParseUint(fields[statFieldRss], 10, 64)
	if err != nil {
		return fmt.Errorf("parsing rss: %w", err)
	}
	proc.MemBytes = rss * pageSize

	// Calculate memory percentage
	if r.totalMemoryBytes > 0 {
		proc.MemPercent = float64(proc.MemBytes) / float64(r.totalMemoryBytes) * 100.0
	}

	// Store CPU times for rate calculation
	ct.utime = utime
	ct.stime = stime
	ct.total = utime + stime

	// Calculate CPU percentage based on delta since last sample
	if cpuDelta > 0 {
		lastCT, exists := r.lastCPUTimes[proc.PID]
		if exists {
			procCPUDelta := ct.total - lastCT.total
			// CPU percentage = (process CPU delta / total CPU delta) * 100
			proc.CPUPercent = float64(procCPUDelta) / float64(cpuDelta) * 100.0
			// Clamp to reasonable values
			if proc.CPUPercent < 0 {
				proc.CPUPercent = 0
			}
			if proc.CPUPercent > 100 {
				proc.CPUPercent = 100
			}
		}
	}

	return nil
}

// readTotalCPUTime reads the total CPU time from /proc/stat.
func (r *processReader) readTotalCPUTime() (uint64, error) {
	file, err := os.Open(filepath.Join(r.procPath, "stat"))
	if err != nil {
		return 0, fmt.Errorf("opening /proc/stat: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			return 0, fmt.Errorf("invalid cpu line format")
		}

		var total uint64
		// Sum user, nice, system, idle, iowait, irq, softirq, etc.
		for i := 1; i < len(fields); i++ {
			val, err := strconv.ParseUint(fields[i], 10, 64)
			if err != nil {
				continue
			}
			total += val
		}
		return total, nil
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("scanning /proc/stat: %w", err)
	}

	return 0, fmt.Errorf("cpu line not found in /proc/stat")
}

// readTotalMemory reads the total system memory from /proc/meminfo.
func (r *processReader) readTotalMemory() error {
	file, err := os.Open(filepath.Join(r.procPath, "meminfo"))
	if err != nil {
		return fmt.Errorf("opening /proc/meminfo: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "MemTotal:") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			return fmt.Errorf("invalid MemTotal format")
		}

		val, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return fmt.Errorf("parsing MemTotal: %w", err)
		}

		r.totalMemoryBytes = val * 1024 // Convert from kB to bytes
		return nil
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanning /proc/meminfo: %w", err)
	}

	return fmt.Errorf("MemTotal not found in /proc/meminfo")
}
