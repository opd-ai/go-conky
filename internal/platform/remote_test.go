package platform

import (
	"context"
	"testing"
	"time"
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
