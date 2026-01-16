package platform

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

func TestNewSSHPlatform(t *testing.T) {
	tests := []struct {
		name    string
		config  RemoteConfig
		wantErr bool
	}{
		{
			name: "valid config with password auth",
			config: RemoteConfig{
				Host: "example.com",
				User: "testuser",
				AuthMethod: PasswordAuth{
					Password: "testpass",
				},
			},
			wantErr: false,
		},
		{
			name: "valid config with key auth",
			config: RemoteConfig{
				Host: "example.com",
				User: "testuser",
				AuthMethod: KeyAuth{
					PrivateKeyPath: "/path/to/key",
				},
			},
			wantErr: false,
		},
		{
			name: "missing host",
			config: RemoteConfig{
				User: "testuser",
				AuthMethod: PasswordAuth{
					Password: "testpass",
				},
			},
			wantErr: true,
		},
		{
			name: "missing user",
			config: RemoteConfig{
				Host: "example.com",
				AuthMethod: PasswordAuth{
					Password: "testpass",
				},
			},
			wantErr: true,
		},
		{
			name: "missing auth method",
			config: RemoteConfig{
				Host: "example.com",
				User: "testuser",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newSSHPlatform(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("newSSHPlatform() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSSHPlatform_Name(t *testing.T) {
	tests := []struct {
		name     string
		targetOS string
		want     string
	}{
		{
			name:     "linux platform",
			targetOS: "linux",
			want:     "remote-linux",
		},
		{
			name:     "darwin platform",
			targetOS: "darwin",
			want:     "remote-darwin",
		},
		{
			name:     "no target os",
			targetOS: "",
			want:     "remote",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &sshPlatform{
				targetOS: tt.targetOS,
			}
			if got := p.Name(); got != tt.want {
				t.Errorf("sshPlatform.Name() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRemoteConfig_Defaults(t *testing.T) {
	config := RemoteConfig{
		Host: "example.com",
		User: "testuser",
		AuthMethod: PasswordAuth{
			Password: "testpass",
		},
	}

	p, err := newSSHPlatform(config)
	if err != nil {
		t.Fatalf("newSSHPlatform() error = %v", err)
	}

	// Check defaults
	if p.config.Port != 22 {
		t.Errorf("Default port = %d, want 22", p.config.Port)
	}

	if p.config.CommandTimeout != 5*time.Second {
		t.Errorf("Default command timeout = %v, want 5s", p.config.CommandTimeout)
	}

	if p.config.ReconnectInterval != 30*time.Second {
		t.Errorf("Default reconnect interval = %v, want 30s", p.config.ReconnectInterval)
	}
}

func TestAuthMethod_Interface(t *testing.T) {
	// Test that all auth methods implement the interface
	var _ AuthMethod = PasswordAuth{}
	var _ AuthMethod = KeyAuth{}
	var _ AuthMethod = AgentAuth{}
}

func TestSSHPlatform_Close(t *testing.T) {
	p := &sshPlatform{
		ctx: context.Background(),
	}

	// Create a cancel function
	ctx, cancel := context.WithCancel(context.Background())
	p.ctx = ctx
	p.cancel = cancel

	// Close should not error
	if err := p.Close(); err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}

	// Calling Close again should be safe
	if err := p.Close(); err != nil {
		t.Errorf("Close() second call error = %v, want nil", err)
	}
}

func TestSSHPlatform_Providers(t *testing.T) {
	tests := []struct {
		name     string
		targetOS string
		wantCPU  bool
		wantMem  bool
		wantNet  bool
		wantFS   bool
		wantBat  bool
		wantSens bool
	}{
		{
			name:     "linux platform",
			targetOS: "linux",
			wantCPU:  true,
			wantMem:  true,
			wantNet:  true,
			wantFS:   true,
			wantBat:  false,
			wantSens: true,
		},
		{
			name:     "darwin platform",
			targetOS: "darwin",
			wantCPU:  true,
			wantMem:  true,
			wantNet:  true,
			wantFS:   true,
			wantBat:  false,
			wantSens: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &sshPlatform{
				targetOS: tt.targetOS,
			}

			// Initialize providers based on target OS
			switch tt.targetOS {
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
			}

			if (p.CPU() != nil) != tt.wantCPU {
				t.Errorf("CPU() != nil = %v, want %v", p.CPU() != nil, tt.wantCPU)
			}
			if (p.Memory() != nil) != tt.wantMem {
				t.Errorf("Memory() != nil = %v, want %v", p.Memory() != nil, tt.wantMem)
			}
			if (p.Network() != nil) != tt.wantNet {
				t.Errorf("Network() != nil = %v, want %v", p.Network() != nil, tt.wantNet)
			}
			if (p.Filesystem() != nil) != tt.wantFS {
				t.Errorf("Filesystem() != nil = %v, want %v", p.Filesystem() != nil, tt.wantFS)
			}
			if (p.Battery() != nil) != tt.wantBat {
				t.Errorf("Battery() != nil = %v, want %v", p.Battery() != nil, tt.wantBat)
			}
			if (p.Sensors() != nil) != tt.wantSens {
				t.Errorf("Sensors() != nil = %v, want %v", p.Sensors() != nil, tt.wantSens)
			}
		})
	}
}

// TestCommandInjectionPrevention tests that user-controlled parameters are properly escaped
func TestCommandInjectionPrevention(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		shouldErr bool
	}{
		{
			name:      "safe interface name",
			input:     "eth0",
			shouldErr: false,
		},
		{
			name:      "interface with semicolon",
			input:     "eth0; rm -rf /",
			shouldErr: true,
		},
		{
			name:      "interface with backticks",
			input:     "eth0`whoami`",
			shouldErr: true,
		},
		{
			name:      "safe mount point",
			input:     "/mnt/data",
			shouldErr: false,
		},
		{
			name:      "mount point with command injection",
			input:     "/mnt'; rm -rf /; echo '",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that validatePath properly rejects malicious inputs
			// When shouldErr is true, we expect validatePath to return false (invalid)
			// When shouldErr is false, we expect validatePath to return true (valid)
			isValid := validatePath(tt.input)
			if isValid == tt.shouldErr {
				t.Errorf("validatePath(%q) = %v, expected %v", tt.input, isValid, !tt.shouldErr)
			}
		})
	}
}

// TestBuildHostKeyCallback tests the host key verification callback building logic.
func TestBuildHostKeyCallback(t *testing.T) {
	tests := []struct {
		name      string
		config    RemoteConfig
		wantErr   bool
		errSubstr string
	}{
		{
			name: "custom callback takes precedence",
			config: RemoteConfig{
				Host:       "example.com",
				User:       "testuser",
				AuthMethod: PasswordAuth{Password: "testpass"},
				HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
					return nil
				},
			},
			wantErr: false,
		},
		{
			name: "insecure mode with explicit flag",
			config: RemoteConfig{
				Host:                  "example.com",
				User:                  "testuser",
				AuthMethod:            PasswordAuth{Password: "testpass"},
				InsecureIgnoreHostKey: true,
			},
			wantErr: false,
		},
		{
			name: "nonexistent known_hosts file",
			config: RemoteConfig{
				Host:           "example.com",
				User:           "testuser",
				AuthMethod:     PasswordAuth{Password: "testpass"},
				KnownHostsPath: "/nonexistent/path/known_hosts",
			},
			wantErr:   true,
			errSubstr: "known_hosts file not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := newSSHPlatform(tt.config)
			if err != nil {
				t.Fatalf("newSSHPlatform() error = %v", err)
			}

			callback, err := p.buildHostKeyCallback()
			if (err != nil) != tt.wantErr {
				t.Errorf("buildHostKeyCallback() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errSubstr != "" {
				if !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("buildHostKeyCallback() error = %v, want error containing %q", err, tt.errSubstr)
				}
			}
			if !tt.wantErr && callback == nil {
				t.Errorf("buildHostKeyCallback() returned nil callback without error")
			}
		})
	}
}

// TestBuildHostKeyCallbackWithValidKnownHosts tests known_hosts file reading.
func TestBuildHostKeyCallbackWithValidKnownHosts(t *testing.T) {
	// Create a temporary known_hosts file
	tmpDir := t.TempDir()
	knownHostsPath := filepath.Join(tmpDir, "known_hosts")

	// Write a valid known_hosts entry using ed25519 key
	// This is a valid OpenSSH public key format
	knownHostsContent := "example.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIBaLR4I4jx/L5oqjNBl0r/QJLCC0BFmPdCLzU4mQD8vS\n"
	if err := os.WriteFile(knownHostsPath, []byte(knownHostsContent), 0600); err != nil {
		t.Fatalf("Failed to write known_hosts file: %v", err)
	}

	config := RemoteConfig{
		Host:           "example.com",
		User:           "testuser",
		AuthMethod:     PasswordAuth{Password: "testpass"},
		KnownHostsPath: knownHostsPath,
	}

	p, err := newSSHPlatform(config)
	if err != nil {
		t.Fatalf("newSSHPlatform() error = %v", err)
	}

	callback, err := p.buildHostKeyCallback()
	if err != nil {
		t.Errorf("buildHostKeyCallback() error = %v", err)
	}
	if callback == nil {
		t.Errorf("buildHostKeyCallback() returned nil callback")
	}
}

// TestRemoteConfigHostKeyFields tests that the new host key fields are properly set.
func TestRemoteConfigHostKeyFields(t *testing.T) {
	// Test with InsecureIgnoreHostKey
	config := RemoteConfig{
		Host:                  "example.com",
		User:                  "testuser",
		AuthMethod:            PasswordAuth{Password: "testpass"},
		InsecureIgnoreHostKey: true,
	}

	if !config.InsecureIgnoreHostKey {
		t.Error("InsecureIgnoreHostKey should be true")
	}

	// Test with KnownHostsPath
	config = RemoteConfig{
		Host:           "example.com",
		User:           "testuser",
		AuthMethod:     PasswordAuth{Password: "testpass"},
		KnownHostsPath: "/custom/path/known_hosts",
	}

	if config.KnownHostsPath != "/custom/path/known_hosts" {
		t.Errorf("KnownHostsPath = %v, want /custom/path/known_hosts", config.KnownHostsPath)
	}

	// Test with custom HostKeyCallback
	customCallback := func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		return nil
	}

	config = RemoteConfig{
		Host:            "example.com",
		User:            "testuser",
		AuthMethod:      PasswordAuth{Password: "testpass"},
		HostKeyCallback: customCallback,
	}

	if config.HostKeyCallback == nil {
		t.Error("HostKeyCallback should not be nil")
	}
}
