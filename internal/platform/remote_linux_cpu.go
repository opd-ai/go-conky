package platform

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// remoteLinuxCPUProvider collects CPU metrics from remote Linux systems via SSH.
type remoteLinuxCPUProvider struct {
	runner    commandRunner
	mu        sync.Mutex
	prevStats map[int]cpuTimes
}

func newRemoteLinuxCPUProvider(p *sshPlatform) *remoteLinuxCPUProvider {
	return &remoteLinuxCPUProvider{
		runner:    p,
		prevStats: make(map[int]cpuTimes),
	}
}

// newTestableRemoteLinuxCPUProviderWithRunner creates a provider with an injectable runner for testing.
func newTestableRemoteLinuxCPUProviderWithRunner(runner commandRunner) *remoteLinuxCPUProvider {
	return &remoteLinuxCPUProvider{
		runner:    runner,
		prevStats: make(map[int]cpuTimes),
	}
}

// TotalUsage returns the aggregate CPU usage percentage.
// Note: The first call will return 0 because there are no previous stats to compare against.
// At least two samples separated by a time interval are needed to calculate CPU usage.
// Uses parseTotalCPUUsage helper for parsing logic.
func (c *remoteLinuxCPUProvider) TotalUsage() (float64, error) {
	output, err := c.runner.runCommand("cat /proc/stat | head -1")
	if err != nil {
		return 0, fmt.Errorf("failed to read /proc/stat: %w", err)
	}

	return parseTotalCPUUsage(output, c.prevStats, &c.mu)
}

func (c *remoteLinuxCPUProvider) Usage() ([]float64, error) {
	output, err := c.runner.runCommand("cat /proc/stat | grep '^cpu[0-9]'")
	if err != nil {
		return nil, fmt.Errorf("failed to read /proc/stat: %w", err)
	}

	var usages []float64
	lines := strings.Split(strings.TrimSpace(output), "\n")

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 8 {
			continue
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

		prev, exists := c.prevStats[cpuNum]
		c.prevStats[cpuNum] = current

		if !exists {
			usages = append(usages, 0)
			continue
		}

		// Calculate usage percentage
		totalDelta := float64(
			(current.user - prev.user) +
				(current.nice - prev.nice) +
				(current.system - prev.system) +
				(current.idle - prev.idle) +
				(current.iowait - prev.iowait) +
				(current.irq - prev.irq) +
				(current.softirq - prev.softirq) +
				(current.steal - prev.steal))

		idleDelta := float64(current.idle - prev.idle + current.iowait - prev.iowait)

		if totalDelta > 0 {
			usages = append(usages, 100*(1-idleDelta/totalDelta))
		} else {
			usages = append(usages, 0)
		}
	}

	return usages, nil
}

// LoadAverage returns load averages from a remote Linux system.
// Uses parseLoadAverage helper for parsing logic.
func (c *remoteLinuxCPUProvider) LoadAverage() (float64, float64, float64, error) {
	output, err := c.runner.runCommand("cat /proc/loadavg")
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to read /proc/loadavg: %w", err)
	}

	return parseLoadAverage(output)
}

func (c *remoteLinuxCPUProvider) Info() (*CPUInfo, error) {
	output, err := c.runner.runCommand("cat /proc/cpuinfo")
	if err != nil {
		return nil, fmt.Errorf("failed to read /proc/cpuinfo: %w", err)
	}

	info := &CPUInfo{}
	var cores int
	var threads int

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

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
			if n, err := strconv.Atoi(value); err == nil && cores == 0 {
				cores = n
			}
		case "processor":
			threads++
		case "cache size":
			if info.CacheSize == 0 {
				// Parse "6144 KB" format
				parts := strings.Fields(value)
				if len(parts) >= 2 {
					size, err := strconv.ParseInt(parts[0], 10, 64)
					if err == nil {
						switch strings.ToLower(parts[1]) {
						case "kb":
							info.CacheSize = size * 1024
						case "mb":
							info.CacheSize = size * 1024 * 1024
						}
					}
				}
			}
		}
	}

	info.Cores = cores
	info.Threads = threads

	return info, nil
}

func (c *remoteLinuxCPUProvider) Frequency() ([]float64, error) {
	// Try to read from /proc/cpuinfo first
	output, err := c.runner.runCommand("cat /proc/cpuinfo | grep 'cpu MHz'")
	if err != nil {
		return nil, fmt.Errorf("failed to read CPU frequency: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	frequencies := make([]float64, 0, len(lines))

	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		value := strings.TrimSpace(parts[1])
		freq, err := strconv.ParseFloat(value, 64)
		if err != nil {
			continue
		}

		frequencies = append(frequencies, freq)
	}

	if len(frequencies) == 0 {
		return nil, fmt.Errorf("no CPU frequency information available")
	}

	return frequencies, nil
}
