# SSH Remote Monitoring - Quick Start

This guide provides a quick introduction to using go-conky's SSH remote monitoring feature.

## What is SSH Remote Monitoring?

SSH Remote Monitoring allows go-conky to collect system metrics from remote machines via SSH without requiring go-conky or any agent software to be installed on the target system. It works by executing standard shell commands over SSH and parsing the output locally.

## Quick Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/opd-ai/go-conky/internal/platform"
)

func main() {
    // Configure SSH connection
    config := platform.RemoteConfig{
        Host: "server.example.com",
        User: "monitoring",
        AuthMethod: platform.KeyAuth{
            PrivateKeyPath: "/home/user/.ssh/id_rsa",
        },
    }
    
    // Create and initialize remote platform
    remote, err := platform.NewRemotePlatform(config)
    if err != nil {
        log.Fatalf("Failed to create remote platform: %v", err)
    }
    defer remote.Close()
    
    if err := remote.Initialize(context.Background()); err != nil {
        log.Fatalf("Failed to initialize: %v", err)
    }
    
    // Collect metrics
    cpu, err := remote.CPU().TotalUsage()
    if err != nil {
        log.Fatalf("Failed to get CPU usage: %v", err)
    }
    fmt.Printf("CPU Usage: %.2f%%\n", cpu)
    
    mem, err := remote.Memory().Stats()
    if err != nil {
        log.Fatalf("Failed to get memory stats: %v", err)
    }
    fmt.Printf("Memory: %.2f%% used\n", mem.UsedPercent)
}
```

## Authentication Methods

### 1. SSH Private Key (Recommended)

```go
AuthMethod: platform.KeyAuth{
    PrivateKeyPath: "/home/user/.ssh/id_rsa",
    Passphrase:     "", // Optional
}
```

### 2. SSH Agent

```go
AuthMethod: platform.AgentAuth{}
```

### 3. Password (Not Recommended)

```go
AuthMethod: platform.PasswordAuth{
    Password: "your-password",
}
```

## Supported Systems

| OS | Status | Notes |
|----|--------|-------|
| Linux | ✅ Full Support | All metrics available |
| macOS | ✅ Full Support | Limited sensor support |
| Windows | ⏳ Planned | Not yet implemented |

## Available Metrics

- **CPU**: Usage, load average, frequency, core info
- **Memory**: Total, used, free, cached, swap
- **Network**: Bytes/packets sent/received per interface
- **Filesystem**: Disk usage, mount points, I/O stats
- **Sensors**: Temperature and fan speeds (Linux only)

## Next Steps

- Read the [full documentation](ssh-remote-monitoring.md)
- Run the [integration tests](../internal/platform/remote_integration_test.go)
- Check [security considerations](ssh-remote-monitoring.md#security-considerations)

## Testing

Run unit tests:
```bash
go test ./internal/platform/... -v
```

Run integration tests (requires SSH server):
```bash
export SSH_TEST_HOST=server.example.com
export SSH_TEST_USER=testuser
export SSH_TEST_KEY=/path/to/key
go test ./internal/platform/... -v -tags=integration
```

## Troubleshooting

**Connection timeout?** Increase `CommandTimeout`:
```go
config.CommandTimeout = 30 * time.Second
```

**Permission denied?** Check:
- SSH key is in `~/.ssh/authorized_keys` on remote host
- User has permissions to read `/proc` and run system commands

**Missing metrics?** Some metrics require specific permissions or may not be available on all systems.

See the [full documentation](ssh-remote-monitoring.md#troubleshooting) for more help.
