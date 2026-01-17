package platform

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/opd-ai/go-conky/pkg/conky"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

// sshPlatform implements Platform for remote systems via SSH.
// It executes standard shell commands on the remote system and parses
// the output locally, eliminating the need for go-conky installation
// on the target system.
type sshPlatform struct {
	config     RemoteConfig
	client     *ssh.Client
	targetOS   string
	ctx        context.Context
	cancel     context.CancelFunc
	mu         sync.RWMutex
	cmdTimeout time.Duration

	// circuitBreaker protects SSH operations from cascading failures
	circuitBreaker *conky.CircuitBreaker

	// Providers for system monitoring
	cpu        CPUProvider
	memory     MemoryProvider
	network    NetworkProvider
	filesystem FilesystemProvider
	sensors    SensorProvider
}

// newSSHPlatform creates a new SSH-based remote platform.
func newSSHPlatform(config RemoteConfig) (*sshPlatform, error) {
	// Validate configuration
	if config.Host == "" {
		return nil, fmt.Errorf("host is required")
	}
	if config.User == "" {
		return nil, fmt.Errorf("user is required")
	}
	if config.AuthMethod == nil {
		return nil, fmt.Errorf("authentication method is required")
	}

	// Set defaults
	if config.Port == 0 {
		config.Port = 22
	}
	if config.CommandTimeout == 0 {
		config.CommandTimeout = 5 * time.Second
	}
	if config.ReconnectInterval == 0 {
		config.ReconnectInterval = 30 * time.Second
	}

	p := &sshPlatform{
		config:     config,
		cmdTimeout: config.CommandTimeout,
	}

	// Initialize circuit breaker if enabled (default: true)
	circuitEnabled := config.CircuitBreakerEnabled == nil || *config.CircuitBreakerEnabled
	if circuitEnabled {
		cbConfig := conky.CircuitBreakerConfig{
			FailureThreshold: config.CircuitBreakerFailureThreshold,
			Timeout:          config.CircuitBreakerTimeout,
			OnStateChange: func(from, to conky.CircuitState) {
				log.Printf("SSH circuit breaker for %s: %s -> %s",
					config.Host, from.String(), to.String())
			},
		}
		// Apply defaults if not configured
		if cbConfig.FailureThreshold == 0 {
			cbConfig.FailureThreshold = 5
		}
		if cbConfig.Timeout == 0 {
			cbConfig.Timeout = 30 * time.Second
		}
		p.circuitBreaker = conky.NewCircuitBreaker(cbConfig)
	}

	return p, nil
}

func (p *sshPlatform) Name() string {
	if p.targetOS != "" {
		return fmt.Sprintf("remote-%s", p.targetOS)
	}
	return "remote"
}

func (p *sshPlatform) Initialize(ctx context.Context) error {
	p.ctx, p.cancel = context.WithCancel(ctx)

	// Build SSH client configuration
	sshConfig, err := p.buildSSHConfig()
	if err != nil {
		return fmt.Errorf("failed to build SSH config: %w", err)
	}

	// Connect to remote host
	addr := fmt.Sprintf("%s:%d", p.config.Host, p.config.Port)
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", addr, err)
	}

	p.mu.Lock()
	p.client = client
	p.mu.Unlock()

	// Auto-detect target OS if not specified
	if p.config.TargetOS == "" {
		p.targetOS, err = p.detectOS()
		if err != nil {
			p.client.Close()
			return fmt.Errorf("failed to detect remote OS: %w", err)
		}
	} else {
		p.targetOS = p.config.TargetOS
	}

	// Initialize providers based on target OS
	switch p.targetOS {
	case "linux":
		p.cpu = newRemoteLinuxCPUProvider(p)
		p.memory = newRemoteLinuxMemoryProvider(p)
		p.network = newRemoteLinuxNetworkProvider(p)
		p.filesystem = newRemoteLinuxFilesystemProvider(p)
		p.sensors = newRemoteLinuxSensorProvider(p)
	case "darwin":
		p.cpu = newRemoteDarwinCPUProvider(p)
		p.memory = newRemoteDarwinMemoryProvider(p)
		p.network = newRemoteDarwinNetworkProvider(p)
		p.filesystem = newRemoteDarwinFilesystemProvider(p)
		p.sensors = nil // Limited sensor support on macOS
	case "windows":
		// Windows remote monitoring via PowerShell commands
		return fmt.Errorf("Windows remote monitoring not yet implemented")
	default:
		return fmt.Errorf("unsupported remote OS: %s", p.targetOS)
	}

	return nil
}

func (p *sshPlatform) buildSSHConfig() (*ssh.ClientConfig, error) {
	var authMethods []ssh.AuthMethod

	switch auth := p.config.AuthMethod.(type) {
	case PasswordAuth:
		authMethods = append(authMethods, ssh.Password(auth.Password))
	case KeyAuth:
		key, err := os.ReadFile(auth.PrivateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read private key: %w", err)
		}
		var signer ssh.Signer
		if auth.Passphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(key, []byte(auth.Passphrase))
		} else {
			signer, err = ssh.ParsePrivateKey(key)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	case AgentAuth:
		socket := os.Getenv("SSH_AUTH_SOCK")
		if socket == "" {
			return nil, fmt.Errorf("SSH_AUTH_SOCK not set")
		}
		// Use a callback to defer the agent connection until it's actually needed
		authMethods = append(authMethods, ssh.PublicKeysCallback(func() ([]ssh.Signer, error) {
			agentConn, err := net.Dial("unix", socket)
			if err != nil {
				return nil, fmt.Errorf("failed to connect to SSH agent: %w", err)
			}
			defer agentConn.Close()

			agentClient := agent.NewClient(agentConn)
			signers, err := agentClient.Signers()
			if err != nil {
				return nil, fmt.Errorf("failed to get signers from SSH agent: %w", err)
			}

			return signers, nil
		}))
	default:
		return nil, fmt.Errorf("unsupported auth method type: %T", auth)
	}

	// Build host key callback based on configuration
	hostKeyCallback, err := p.buildHostKeyCallback()
	if err != nil {
		return nil, fmt.Errorf("failed to build host key callback: %w", err)
	}

	return &ssh.ClientConfig{
		User:            p.config.User,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         10 * time.Second,
	}, nil
}

// buildHostKeyCallback creates the appropriate host key verification callback
// based on the RemoteConfig settings.
//
// Priority order:
// 1. Custom HostKeyCallback (if set)
// 2. InsecureIgnoreHostKey (if true, with warning)
// 3. KnownHostsPath (explicit path to known_hosts file)
// 4. Default known_hosts file (~/.ssh/known_hosts)
func (p *sshPlatform) buildHostKeyCallback() (ssh.HostKeyCallback, error) {
	// Priority 1: Use custom callback if provided
	if p.config.HostKeyCallback != nil {
		return p.config.HostKeyCallback, nil
	}

	// Priority 2: Use insecure mode if explicitly requested
	if p.config.InsecureIgnoreHostKey {
		log.Printf("WARNING: SSH host key verification is disabled for %s. " +
			"This makes the connection vulnerable to man-in-the-middle attacks. " +
			"Set InsecureIgnoreHostKey=false and use known_hosts for production.",
			p.config.Host)
		return ssh.InsecureIgnoreHostKey(), nil
	}

	// Priority 3 & 4: Use known_hosts file
	knownHostsPath := p.config.KnownHostsPath
	if knownHostsPath == "" {
		// Default to ~/.ssh/known_hosts
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		knownHostsPath = filepath.Join(homeDir, ".ssh", "known_hosts")
	}

	// Check if the known_hosts file exists
	if _, err := os.Stat(knownHostsPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("known_hosts file not found at %s. "+
			"Either add the host key to known_hosts, specify a custom path with KnownHostsPath, "+
			"provide a custom HostKeyCallback, or set InsecureIgnoreHostKey=true (not recommended)",
			knownHostsPath)
	}

	// Create callback from known_hosts file
	callback, err := knownhosts.New(knownHostsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read known_hosts file %s: %w", knownHostsPath, err)
	}

	return callback, nil
}

func (p *sshPlatform) detectOS() (string, error) {
	// Try uname first (works on Linux, macOS, BSD)
	output, err := p.runCommand("uname -s")
	if err == nil {
		os := strings.TrimSpace(output)
		switch strings.ToLower(os) {
		case "linux":
			return "linux", nil
		case "darwin":
			return "darwin", nil
		}
	}

	// Try Windows detection via PowerShell
	output, err = p.runCommand("echo %OS%")
	if err == nil && strings.Contains(output, "Windows") {
		return "windows", nil
	}

	return "", fmt.Errorf("unable to detect remote OS")
}

// runCommand executes a command on the remote system and returns the output.
// If a circuit breaker is configured, operations will be rejected when the
// circuit is open due to consecutive failures.
func (p *sshPlatform) runCommand(cmd string) (string, error) {
	// Check circuit breaker before attempting command
	if p.circuitBreaker != nil {
		var output string
		err := p.circuitBreaker.Execute(func() error {
			var cmdErr error
			output, cmdErr = p.runCommandInternal(cmd)
			return cmdErr
		})
		return output, err
	}

	// No circuit breaker, run directly
	return p.runCommandInternal(cmd)
}

// runCommandInternal executes a command on the remote system without circuit breaker protection.
func (p *sshPlatform) runCommandInternal(cmd string) (string, error) {
	p.mu.RLock()
	client := p.client
	p.mu.RUnlock()

	if client == nil {
		return "", fmt.Errorf("SSH client not connected")
	}

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	// Run command with timeout
	done := make(chan error, 1)
	go func() {
		done <- session.Run(cmd)
	}()

	select {
	case err := <-done:
		if err != nil {
			return "", fmt.Errorf("command failed: %w (stderr: %s)", err, stderr.String())
		}
		return stdout.String(), nil
	case <-time.After(p.cmdTimeout):
		// Ensure the remote command is actually terminated on timeout
		_ = session.Signal(ssh.SIGKILL)
		_ = session.Close()
		return "", fmt.Errorf("command timed out after %v", p.cmdTimeout)
	case <-p.ctx.Done():
		// Ensure the remote command is terminated when the platform context is cancelled
		_ = session.Signal(ssh.SIGKILL)
		_ = session.Close()
		return "", p.ctx.Err()
	}
}

func (p *sshPlatform) Close() error {
	if p.cancel != nil {
		p.cancel()
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.client != nil {
		err := p.client.Close()
		p.client = nil
		return err
	}
	return nil
}

func (p *sshPlatform) CPU() CPUProvider {
	return p.cpu
}

func (p *sshPlatform) Memory() MemoryProvider {
	return p.memory
}

func (p *sshPlatform) Network() NetworkProvider {
	return p.network
}

func (p *sshPlatform) Filesystem() FilesystemProvider {
	return p.filesystem
}

func (p *sshPlatform) Battery() BatteryProvider {
	// Battery monitoring typically not needed for remote servers
	return nil
}

func (p *sshPlatform) Sensors() SensorProvider {
	return p.sensors
}

// CircuitBreakerStats returns the current circuit breaker statistics.
// Returns nil if circuit breaker is not enabled.
func (p *sshPlatform) CircuitBreakerStats() *conky.CircuitBreakerStats {
	if p.circuitBreaker == nil {
		return nil
	}
	stats := p.circuitBreaker.Stats()
	return &stats
}

// ResetCircuitBreaker resets the circuit breaker to closed state.
// This is useful for manual recovery after addressing connectivity issues.
func (p *sshPlatform) ResetCircuitBreaker() {
	if p.circuitBreaker != nil {
		p.circuitBreaker.Reset()
	}
}
