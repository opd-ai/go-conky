package platform

import (
	"time"
)

// NewRemotePlatform creates a Platform that collects data from a remote system via SSH.
// The remote system does not need go-conky installed; data is collected using
// standard shell commands and parsed locally.
func NewRemotePlatform(config RemoteConfig) (Platform, error) {
	p, err := newSSHPlatform(config)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// RemoteConfig specifies connection parameters for remote monitoring.
type RemoteConfig struct {
	// Host is the hostname or IP address of the remote system.
	Host string

	// Port is the SSH port (default: 22).
	Port int

	// User is the SSH username.
	User string

	// AuthMethod specifies how to authenticate.
	AuthMethod AuthMethod

	// TargetOS specifies the operating system of the remote host.
	// Auto-detected if empty.
	TargetOS string

	// CommandTimeout is the timeout for individual commands (default: 5s).
	CommandTimeout time.Duration

	// ReconnectInterval is how often to attempt reconnection on failure (default: 30s).
	ReconnectInterval time.Duration
}

// AuthMethod defines SSH authentication methods.
type AuthMethod interface {
	isAuthMethod()
}

// PasswordAuth authenticates using a password.
type PasswordAuth struct {
	Password string
}

func (PasswordAuth) isAuthMethod() {}

// KeyAuth authenticates using an SSH private key.
type KeyAuth struct {
	PrivateKeyPath string
	Passphrase     string // optional, for encrypted keys
}

func (KeyAuth) isAuthMethod() {}

// AgentAuth authenticates using the SSH agent.
type AgentAuth struct{}

func (AgentAuth) isAuthMethod() {}
