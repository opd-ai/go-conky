// Package monitor provides NVIDIA GPU monitoring via nvidia-smi.
package monitor

import (
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// GPUStats contains NVIDIA GPU statistics.
type GPUStats struct {
	Name        string
	DriverVer   string
	Temperature int     // Celsius
	UtilGPU     int     // GPU utilization percentage
	UtilMem     int     // Memory utilization percentage
	MemUsed     uint64  // Bytes
	MemTotal    uint64  // Bytes
	MemFree     uint64  // Bytes
	FanSpeed    int     // Percentage
	PowerDraw   float64 // Watts
	PowerLimit  float64 // Watts
	Available   bool
}

// gpuReader reads NVIDIA GPU stats using nvidia-smi.
type gpuReader struct {
	mu            sync.RWMutex
	cache         GPUStats
	lastUpdate    time.Time
	cacheDuration time.Duration
	nvidiaSmiPath string
}

// newGPUReader creates a new gpuReader.
func newGPUReader() *gpuReader {
	// Try to find nvidia-smi
	path, err := exec.LookPath("nvidia-smi")
	if err != nil {
		path = ""
	}
	return &gpuReader{
		cacheDuration: 2 * time.Second,
		nvidiaSmiPath: path,
	}
}

// ReadStats reads current GPU statistics.
func (r *gpuReader) ReadStats() (GPUStats, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Return cached if fresh
	if time.Since(r.lastUpdate) < r.cacheDuration {
		return r.cache, nil
	}

	if r.nvidiaSmiPath == "" {
		r.cache = GPUStats{Available: false}
		return r.cache, nil
	}

	stats, err := r.queryNvidiaSmi()
	if err != nil {
		r.cache = GPUStats{Available: false}
		return r.cache, nil
	}

	r.cache = stats
	r.lastUpdate = time.Now()
	return r.cache, nil
}

// queryNvidiaSmi runs nvidia-smi and parses output.
func (r *gpuReader) queryNvidiaSmi() (GPUStats, error) {
	// Query specific fields in CSV format
	cmd := exec.Command(r.nvidiaSmiPath,
		"--query-gpu=name,driver_version,temperature.gpu,utilization.gpu,utilization.memory,memory.used,memory.total,memory.free,fan.speed,power.draw,power.limit",
		"--format=csv,noheader,nounits")

	output, err := cmd.Output()
	if err != nil {
		return GPUStats{}, err
	}

	return r.parseNvidiaSmiOutput(string(output))
}

// parseNvidiaSmiOutput parses CSV output from nvidia-smi.
func (r *gpuReader) parseNvidiaSmiOutput(output string) (GPUStats, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 {
		return GPUStats{}, nil
	}

	// Parse first GPU
	fields := strings.Split(lines[0], ", ")
	if len(fields) < 11 {
		return GPUStats{}, nil
	}

	stats := GPUStats{Available: true}

	stats.Name = strings.TrimSpace(fields[0])
	stats.DriverVer = strings.TrimSpace(fields[1])
	stats.Temperature, _ = strconv.Atoi(strings.TrimSpace(fields[2]))
	stats.UtilGPU, _ = strconv.Atoi(strings.TrimSpace(fields[3]))
	stats.UtilMem, _ = strconv.Atoi(strings.TrimSpace(fields[4]))

	// Memory in MiB from nvidia-smi
	memUsed, _ := strconv.ParseUint(strings.TrimSpace(fields[5]), 10, 64)
	memTotal, _ := strconv.ParseUint(strings.TrimSpace(fields[6]), 10, 64)
	memFree, _ := strconv.ParseUint(strings.TrimSpace(fields[7]), 10, 64)
	stats.MemUsed = memUsed * 1024 * 1024
	stats.MemTotal = memTotal * 1024 * 1024
	stats.MemFree = memFree * 1024 * 1024

	stats.FanSpeed, _ = strconv.Atoi(strings.TrimSpace(fields[8]))
	stats.PowerDraw, _ = strconv.ParseFloat(strings.TrimSpace(fields[9]), 64)
	stats.PowerLimit, _ = strconv.ParseFloat(strings.TrimSpace(fields[10]), 64)

	return stats, nil
}

// GetField returns a specific field value as string.
func (stats GPUStats) GetField(field string) string {
	if !stats.Available {
		return "N/A"
	}

	switch strings.ToLower(field) {
	case "gpuutil", "gpu", "utilization":
		return strconv.Itoa(stats.UtilGPU) + "%"
	case "memutil", "mem":
		return strconv.Itoa(stats.UtilMem) + "%"
	case "temp", "temperature":
		return strconv.Itoa(stats.Temperature) + "°C"
	case "driver", "driverversion":
		return stats.DriverVer
	case "name", "model":
		return stats.Name
	case "fan", "fanspeed":
		return strconv.Itoa(stats.FanSpeed) + "%"
	case "power", "powerdraw":
		return strconv.FormatFloat(stats.PowerDraw, 'f', 1, 64) + "W"
	case "powerlimit":
		return strconv.FormatFloat(stats.PowerLimit, 'f', 1, 64) + "W"
	case "memused":
		return formatGPUBytes(stats.MemUsed)
	case "memtotal":
		return formatGPUBytes(stats.MemTotal)
	case "memfree":
		return formatGPUBytes(stats.MemFree)
	case "memperc":
		if stats.MemTotal == 0 {
			return "0%"
		}
		return strconv.FormatFloat(float64(stats.MemUsed)*100/float64(stats.MemTotal), 'f', 1, 64) + "%"
	default:
		return ""
	}
}

// formatGPUBytes formats bytes to human-readable format.
func formatGPUBytes(bytes uint64) string {
	const (
		KiB = 1024
		MiB = 1024 * KiB
		GiB = 1024 * MiB
	)
	switch {
	case bytes >= GiB:
		return strconv.FormatFloat(float64(bytes)/float64(GiB), 'f', 1, 64) + "GiB"
	case bytes >= MiB:
		return strconv.FormatFloat(float64(bytes)/float64(MiB), 'f', 1, 64) + "MiB"
	case bytes >= KiB:
		return strconv.FormatFloat(float64(bytes)/float64(KiB), 'f', 1, 64) + "KiB"
	default:
		return strconv.FormatUint(bytes, 10) + "B"
	}
}
