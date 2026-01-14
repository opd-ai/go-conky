# SSH Remote Monitoring

This document describes how to use go-conky's SSH remote monitoring feature to collect system metrics from remote machines without requiring go-conky installation on the target system.

## Overview

The SSH remote monitoring feature allows go-conky to connect to remote systems via SSH and collect system metrics using standard shell commands. The data is parsed locally, eliminating the need for go-conky or any special agents to be installed on the target system.

### Supported Remote Operating Systems

- **Linux**: Full support via `/proc` filesystem and standard commands
- **macOS (Darwin)**: Full support via `sysctl`, `iostat`, `vm_stat`, and other standard utilities
- **Windows**: Planned (not yet implemented)

### Available Metrics

| Metric Category | Linux | macOS | Notes |
|----------------|-------|-------|-------|
| CPU Usage | ✓ | ✓ | Per-core and aggregate |
| CPU Info | ✓ | ✓ | Model, vendor, cores, threads, cache |
| CPU Frequency | ✓ | ✓ | MHz for all cores |
| Load Average | ✓ | ✓ | 1, 5, and 15 minute averages |
| Memory Stats | ✓ | ✓ | Total, used, free, available, cached |
| Swap Stats | ✓ | ✓ | Total, used, free |
| Network Interfaces | ✓ | ✓ | List of interfaces |
| Network Stats | ✓ | ✓ | Bytes/packets sent/received, errors |
| Filesystem Mounts | ✓ | ✓ | Mount points, devices, types |
| Filesystem Usage | ✓ | ✓ | Total, used, free space |
| Disk I/O | ✓ | Limited | Read/write bytes and counts |
| Temperature Sensors | ✓ | - | Requires hwmon support |
| Fan Sensors | ✓ | - | Requires hwmon support |

## Configuration

### Authentication Methods

go-conky supports three SSH authentication methods:

#### 1. Password Authentication

```go
config := platform.RemoteConfig{
    Host: "server.example.com",
    Port: 22,
    User: "username",
    AuthMethod: platform.PasswordAuth{
        Password: "your-password",
    },
}
```

**Security Note**: Storing passwords in code is not recommended for production use. Consider using key-based authentication instead.

#### 2. Private Key Authentication

```go
config := platform.RemoteConfig{
    Host: "server.example.com",
    Port: 22,
    User: "username",
    AuthMethod: platform.KeyAuth{
        PrivateKeyPath: "/home/user/.ssh/id_rsa",
        Passphrase:     "", // Optional, for encrypted keys
    },
}
```

#### 3. SSH Agent Authentication

```go
config := platform.RemoteConfig{
    Host: "server.example.com",
    Port: 22,
    User: "username",
    AuthMethod: platform.AgentAuth{},
}
```

Requires the `SSH_AUTH_SOCK` environment variable to be set.

### Optional Parameters

```go
config := platform.RemoteConfig{
    Host: "server.example.com",
    Port: 22,  // Default: 22
    User: "username",
    AuthMethod: /* ... */,
    
    // Optional: Auto-detected if not specified
    TargetOS: "linux",  // or "darwin"
    
    // Optional: Timeout for individual commands
    CommandTimeout: 10 * time.Second,  // Default: 5s
    
    // Optional: Interval for reconnection attempts
    ReconnectInterval: 30 * time.Second,  // Default: 30s
}
```

## Usage Example

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/opd-ai/go-conky/internal/platform"
)

func main() {
    // Configure remote connection
    config := platform.RemoteConfig{
        Host: "server.example.com",
        User: "monitoring",
        AuthMethod: platform.KeyAuth{
            PrivateKeyPath: "/home/user/.ssh/id_rsa",
        },
    }

    // Create remote platform
    remotePlatform, err := platform.NewRemotePlatform(config)
    if err != nil {
        log.Fatalf("Failed to create remote platform: %v", err)
    }
    defer remotePlatform.Close()

    // Initialize connection
    ctx := context.Background()
    if err := remotePlatform.Initialize(ctx); err != nil {
        log.Fatalf("Failed to initialize: %v", err)
    }

    fmt.Printf("Connected to %s\n", remotePlatform.Name())

    // Get CPU usage
    cpuUsage, err := remotePlatform.CPU().TotalUsage()
    if err != nil {
        log.Printf("CPU error: %v", err)
    } else {
        fmt.Printf("CPU Usage: %.2f%%\n", cpuUsage)
    }

    // Get memory stats
    memStats, err := remotePlatform.Memory().Stats()
    if err != nil {
        log.Printf("Memory error: %v", err)
    } else {
        fmt.Printf("Memory: %.2f%% used (%d/%d MB)\n",
            memStats.UsedPercent,
            memStats.Used/1024/1024,
            memStats.Total/1024/1024)
    }

    // Get network stats
    interfaces, err := remotePlatform.Network().Interfaces()
    if err != nil {
        log.Printf("Network error: %v", err)
    } else {
        fmt.Printf("Network Interfaces: %v\n", interfaces)
    }
}
```

### Monitoring Loop

```go
func monitorRemoteSystem(config platform.RemoteConfig) {
    remotePlatform, err := platform.NewRemotePlatform(config)
    if err != nil {
        log.Fatalf("Failed to create remote platform: %v", err)
    }
    defer remotePlatform.Close()

    ctx := context.Background()
    if err := remotePlatform.Initialize(ctx); err != nil {
        log.Fatalf("Failed to initialize: %v", err)
    }

    // Monitor every 5 seconds
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        // CPU
        if usage, err := remotePlatform.CPU().TotalUsage(); err == nil {
            fmt.Printf("CPU: %.2f%% | ", usage)
        }

        // Memory
        if stats, err := remotePlatform.Memory().Stats(); err == nil {
            fmt.Printf("Mem: %.2f%% | ", stats.UsedPercent)
        }

        // Load Average
        if load1, _, _, err := remotePlatform.CPU().LoadAverage(); err == nil {
            fmt.Printf("Load: %.2f\n", load1)
        }
    }
}
```

## Security Considerations

### Host Key Verification

**IMPORTANT**: The current implementation uses `ssh.InsecureIgnoreHostKey()` which does not verify the remote host's identity. This is vulnerable to man-in-the-middle attacks.

For production use, implement proper host key verification:

```go
// Example: Using known_hosts file
hostKeyCallback, err := knownhosts.New("/home/user/.ssh/known_hosts")
if err != nil {
    log.Fatal(err)
}

// Modify the SSH client config in remote.go to use:
// HostKeyCallback: hostKeyCallback,
```

Alternatively, implement a custom `HostKeyCallback` that validates against a trusted Certificate Authority.

### Authentication Best Practices

1. **Use SSH keys instead of passwords** when possible
2. **Encrypt private keys** with a strong passphrase
3. **Use SSH agent** for better key management
4. **Limit SSH user permissions** on remote systems
5. **Use dedicated monitoring users** with minimal privileges
6. **Rotate credentials regularly**

### Required Remote Permissions

The remote user needs read access to:

- `/proc/*` (Linux)
- `/sys/class/hwmon/*` (Linux, for sensors)
- Ability to run: `df`, `mount`, `netstat`, `iostat`, `vm_stat`, `sysctl` (platform-dependent)

Consider creating a dedicated monitoring user with these minimal permissions:

```bash
# Linux example
sudo useradd -r -s /bin/bash monitoring
# Grant read access to /proc and /sys (usually granted by default)
```

## Troubleshooting

### Connection Timeout

If connections timeout, increase the `CommandTimeout`:

```go
config.CommandTimeout = 30 * time.Second
```

### SSH Agent Not Found

Ensure `SSH_AUTH_SOCK` is set:

```bash
echo $SSH_AUTH_SOCK
# Should print something like /tmp/ssh-XXX/agent.123
```

If using SSH agent, start it:

```bash
eval $(ssh-agent)
ssh-add ~/.ssh/id_rsa
```

### Permission Denied

Verify:
1. User exists on remote system
2. SSH key is authorized in `~/.ssh/authorized_keys`
3. SSH service is running on the remote host
4. Firewall allows SSH connections (port 22)

### OS Detection Fails

If automatic OS detection fails, specify it explicitly:

```go
config.TargetOS = "linux"  // or "darwin"
```

### Missing Metrics

Some metrics may not be available depending on:
- Operating system (e.g., fan sensors on macOS)
- System configuration (e.g., no swap configured)
- User permissions (e.g., restricted access to hwmon)

Check error messages for specific issues.

## Performance Considerations

### Command Overhead

Each metric collection executes one or more shell commands via SSH. Consider:

1. **Batch related operations** when possible
2. **Cache values** if you don't need real-time updates
3. **Adjust update intervals** to balance freshness vs. load
4. **Monitor SSH connection count** and reuse connections

### Network Latency

SSH command execution includes network round-trip time. For high-latency connections:

1. Increase `CommandTimeout` to avoid timeouts
2. Reduce monitoring frequency
3. Consider local agents if latency is critical

### Resource Usage

Monitor the impact on remote systems:
- Each command spawns a process
- Parsing large `/proc` files can be CPU-intensive
- Frequent connections may impact SSH daemon performance

## Future Enhancements

Planned improvements:

- [ ] Windows remote monitoring via PowerShell
- [ ] Command batching to reduce SSH overhead
- [ ] Connection pooling and keep-alive
- [ ] Custom command execution for extensibility
- [ ] Metrics caching with TTL
- [ ] Multi-system monitoring dashboard

## Example Configurations

### Multiple Remote Hosts

```go
hosts := []platform.RemoteConfig{
    {
        Host: "web1.example.com",
        User: "monitoring",
        AuthMethod: platform.AgentAuth{},
    },
    {
        Host: "db1.example.com",
        User: "monitoring",
        AuthMethod: platform.AgentAuth{},
    },
}

for _, config := range hosts {
    go monitorHost(config)  // Monitor in parallel
}
```

### Fallback Authentication

```go
func connectWithFallback(host, user string) (platform.Platform, error) {
    // Try SSH agent first
    config := platform.RemoteConfig{
        Host: host,
        User: user,
        AuthMethod: platform.AgentAuth{},
    }
    
    p, err := platform.NewRemotePlatform(config)
    if err == nil {
        return p, nil
    }
    
    // Fallback to key-based auth
    config.AuthMethod = platform.KeyAuth{
        PrivateKeyPath: os.ExpandEnv("$HOME/.ssh/id_rsa"),
    }
    
    return platform.NewRemotePlatform(config)
}
```

## See Also

- [Platform Interface Documentation](platform.go)
- [Linux Platform Implementation](linux.go)
- [macOS Platform Implementation](darwin.go)
- [Integration Tests](remote_integration_test.go)
