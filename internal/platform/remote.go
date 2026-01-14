package platform

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
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

	return &ssh.ClientConfig{
		User: p.config.User,
		Auth: authMethods,
		// TODO: SECURITY - Replace InsecureIgnoreHostKey with proper host key verification
		// Options for production use:
		// 1. Use ssh.FixedHostKey(knownHostKey) with a known host key
		// 2. Use knownhosts.New("/path/to/known_hosts") to read from known_hosts file
		// 3. Implement custom HostKeyCallback for CA-based verification
		// Current implementation is VULNERABLE to man-in-the-middle attacks
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}, nil
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
func (p *sshPlatform) runCommand(cmd string) (string, error) {
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
