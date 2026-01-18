package platform

import (
	"fmt"
)

// commandRunner defines the interface for running remote commands.
// This allows testing without an actual SSH connection.
type commandRunner interface {
	runCommand(cmd string) (string, error)
}

// remoteLinuxMemoryProvider collects memory metrics from remote Linux systems via SSH.
type remoteLinuxMemoryProvider struct {
	runner commandRunner
}

func newRemoteLinuxMemoryProvider(p *sshPlatform) *remoteLinuxMemoryProvider {
	return &remoteLinuxMemoryProvider{
		runner: p,
	}
}

// newTestableRemoteLinuxMemoryProviderWithRunner creates a provider with an injectable runner for testing.
func newTestableRemoteLinuxMemoryProviderWithRunner(runner commandRunner) *remoteLinuxMemoryProvider {
	return &remoteLinuxMemoryProvider{
		runner: runner,
	}
}

// Stats returns memory statistics from a remote Linux system.
// Uses parseMemInfoOutput helper for parsing logic.
func (m *remoteLinuxMemoryProvider) Stats() (*MemoryStats, error) {
	output, err := m.runner.runCommand("cat /proc/meminfo")
	if err != nil {
		return nil, fmt.Errorf("failed to read /proc/meminfo: %w", err)
	}

	return parseMemInfoOutput(output)
}

// SwapStats returns swap statistics from a remote Linux system.
// Uses parseSwapOutput helper for parsing logic.
func (m *remoteLinuxMemoryProvider) SwapStats() (*SwapStats, error) {
	output, err := m.runner.runCommand("cat /proc/meminfo | grep '^Swap'")
	if err != nil {
		return nil, fmt.Errorf("failed to read swap stats: %w", err)
	}

	return parseSwapOutput(output)
}
