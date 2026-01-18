package platform

import (
	"fmt"
	"strconv"
	"strings"
)

// remoteDarwinCPUProvider collects CPU metrics from remote macOS systems via SSH.
type remoteDarwinCPUProvider struct {
	runner commandRunner
}

func newRemoteDarwinCPUProvider(p *sshPlatform) *remoteDarwinCPUProvider {
	return &remoteDarwinCPUProvider{
		runner: p,
	}
}

// newTestableRemoteDarwinCPUProviderWithRunner creates a provider with an injectable runner for testing.
func newTestableRemoteDarwinCPUProviderWithRunner(runner commandRunner) *remoteDarwinCPUProvider {
	return &remoteDarwinCPUProvider{
		runner: runner,
	}
}

func (c *remoteDarwinCPUProvider) TotalUsage() (float64, error) {
	// Use iostat to get CPU usage on macOS
	output, err := c.runner.runCommand("iostat -c 2 | tail -n 1")
	if err != nil {
		return 0, fmt.Errorf("failed to read CPU stats: %w", err)
	}

	// iostat output format: us sy id
	fields := strings.Fields(output)
	if len(fields) < 3 {
		return 0, fmt.Errorf("unexpected iostat output: %s", output)
	}

	user, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse user CPU: %w", err)
	}

	sys, err := strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse system CPU: %w", err)
	}

	// Total usage is 100 - idle
	return user + sys, nil
}

func (c *remoteDarwinCPUProvider) Usage() ([]float64, error) {
	// macOS doesn't provide per-core usage easily via simple commands
	// Return total usage as a single value
	total, err := c.TotalUsage()
	if err != nil {
		return nil, err
	}

	// Get number of cores
	info, err := c.Info()
	if err != nil {
		return []float64{total}, nil
	}

	// Return the same usage for all cores as an approximation
	usages := make([]float64, info.Threads)
	for i := range usages {
		usages[i] = total
	}

	return usages, nil
}

func (c *remoteDarwinCPUProvider) LoadAverage() (float64, float64, float64, error) {
	output, err := c.runner.runCommand("sysctl -n vm.loadavg")
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to read load average: %w", err)
	}

	// Output format: { 1.5 2.0 2.5 }
	output = strings.TrimSpace(output)
	output = strings.Trim(output, "{}")
	fields := strings.Fields(output)

	if len(fields) < 3 {
		return 0, 0, 0, fmt.Errorf("unexpected loadavg format: %s", output)
	}

	load1, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to parse 1min load: %w", err)
	}

	load5, err := strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to parse 5min load: %w", err)
	}

	load15, err := strconv.ParseFloat(fields[2], 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to parse 15min load: %w", err)
	}

	return load1, load5, load15, nil
}

func (c *remoteDarwinCPUProvider) Info() (*CPUInfo, error) {
	info := &CPUInfo{}

	// Get CPU model
	output, err := c.runner.runCommand("sysctl -n machdep.cpu.brand_string")
	if err == nil {
		info.Model = strings.TrimSpace(output)
	}

	// Get CPU vendor
	output, err = c.runner.runCommand("sysctl -n machdep.cpu.vendor")
	if err == nil {
		info.Vendor = strings.TrimSpace(output)
	}

	// Get core count
	output, err = c.runner.runCommand("sysctl -n hw.physicalcpu")
	if err == nil {
		if cores, err := strconv.Atoi(strings.TrimSpace(output)); err == nil {
			info.Cores = cores
		}
	}

	// Get thread count
	output, err = c.runner.runCommand("sysctl -n hw.logicalcpu")
	if err == nil {
		if threads, err := strconv.Atoi(strings.TrimSpace(output)); err == nil {
			info.Threads = threads
		}
	}

	// Get cache size
	output, err = c.runner.runCommand("sysctl -n hw.l3cachesize")
	if err == nil {
		if cache, err := strconv.ParseInt(strings.TrimSpace(output), 10, 64); err == nil {
			info.CacheSize = cache
		}
	}

	return info, nil
}

func (c *remoteDarwinCPUProvider) Frequency() ([]float64, error) {
	// Get CPU frequency
	output, err := c.runner.runCommand("sysctl -n hw.cpufrequency")
	if err != nil {
		return nil, fmt.Errorf("failed to read CPU frequency: %w", err)
	}

	freqHz, err := strconv.ParseFloat(strings.TrimSpace(output), 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CPU frequency: %w", err)
	}

	// Convert Hz to MHz
	freqMHz := freqHz / 1000000

	// Get number of threads
	info, err := c.Info()
	if err != nil {
		return []float64{freqMHz}, nil
	}

	// Return the same frequency for all cores
	frequencies := make([]float64, info.Threads)
	for i := range frequencies {
		frequencies[i] = freqMHz
	}

	return frequencies, nil
}
