//go:build integration

package platform

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestSSHRemoteIntegration tests SSH remote monitoring against a real SSH server.
// This test requires the following environment variables:
// - SSH_TEST_HOST: hostname or IP address of SSH server
// - SSH_TEST_USER: SSH username
// - SSH_TEST_KEY: path to SSH private key OR
// - SSH_TEST_PASSWORD: SSH password
func TestSSHRemoteIntegration(t *testing.T) {
	host := os.Getenv("SSH_TEST_HOST")
	user := os.Getenv("SSH_TEST_USER")
	keyPath := os.Getenv("SSH_TEST_KEY")
	password := os.Getenv("SSH_TEST_PASSWORD")

	if host == "" || user == "" {
		t.Skip("SSH_TEST_HOST and SSH_TEST_USER must be set for integration tests")
	}

	var authMethod AuthMethod
	if keyPath != "" {
		authMethod = KeyAuth{
			PrivateKeyPath: keyPath,
		}
	} else if password != "" {
		authMethod = PasswordAuth{
			Password: password,
		}
	} else {
		t.Skip("Either SSH_TEST_KEY or SSH_TEST_PASSWORD must be set for integration tests")
	}

	config := RemoteConfig{
		Host:           host,
		User:           user,
		AuthMethod:     authMethod,
		CommandTimeout: 10 * time.Second,
	}

	t.Run("Connection", func(t *testing.T) {
		platform, err := NewRemotePlatform(config)
		if err != nil {
			t.Fatalf("Failed to create remote platform: %v", err)
		}
		defer platform.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := platform.Initialize(ctx); err != nil {
			t.Fatalf("Failed to initialize platform: %v", err)
		}

		t.Logf("Connected to %s as %s, detected OS: %s", host, user, platform.Name())
	})

	t.Run("CPU_Stats", func(t *testing.T) {
		platform, err := NewRemotePlatform(config)
		if err != nil {
			t.Fatalf("Failed to create remote platform: %v", err)
		}
		defer platform.Close()

		ctx := context.Background()
		if err := platform.Initialize(ctx); err != nil {
			t.Fatalf("Failed to initialize platform: %v", err)
		}

		cpu := platform.CPU()
		if cpu == nil {
			t.Fatal("CPU provider is nil")
		}

		// Test CPU info
		info, err := cpu.Info()
		if err != nil {
			t.Errorf("Failed to get CPU info: %v", err)
		} else {
			t.Logf("CPU Info: Model=%s, Cores=%d, Threads=%d", info.Model, info.Cores, info.Threads)
		}

		// Test load average
		load1, load5, load15, err := cpu.LoadAverage()
		if err != nil {
			t.Logf("Load average not available: %v", err)
		} else {
			t.Logf("Load Average: 1min=%.2f, 5min=%.2f, 15min=%.2f", load1, load5, load15)
		}

		// Test CPU usage (requires two samples)
		time.Sleep(1 * time.Second)
		usage, err := cpu.TotalUsage()
		if err != nil {
			t.Errorf("Failed to get CPU usage: %v", err)
		} else {
			t.Logf("CPU Usage: %.2f%%", usage)
		}
	})

	t.Run("Memory_Stats", func(t *testing.T) {
		platform, err := NewRemotePlatform(config)
		if err != nil {
			t.Fatalf("Failed to create remote platform: %v", err)
		}
		defer platform.Close()

		ctx := context.Background()
		if err := platform.Initialize(ctx); err != nil {
			t.Fatalf("Failed to initialize platform: %v", err)
		}

		memory := platform.Memory()
		if memory == nil {
			t.Fatal("Memory provider is nil")
		}

		stats, err := memory.Stats()
		if err != nil {
			t.Errorf("Failed to get memory stats: %v", err)
		} else {
			t.Logf("Memory: Total=%dMB, Used=%dMB, Free=%dMB, %.2f%% used",
				stats.Total/1024/1024, stats.Used/1024/1024, stats.Free/1024/1024, stats.UsedPercent)
		}

		swapStats, err := memory.SwapStats()
		if err != nil {
			t.Logf("Swap stats not available: %v", err)
		} else {
			t.Logf("Swap: Total=%dMB, Used=%dMB, Free=%dMB, %.2f%% used",
				swapStats.Total/1024/1024, swapStats.Used/1024/1024, swapStats.Free/1024/1024, swapStats.UsedPercent)
		}
	})

	t.Run("Network_Stats", func(t *testing.T) {
		platform, err := NewRemotePlatform(config)
		if err != nil {
			t.Fatalf("Failed to create remote platform: %v", err)
		}
		defer platform.Close()

		ctx := context.Background()
		if err := platform.Initialize(ctx); err != nil {
			t.Fatalf("Failed to initialize platform: %v", err)
		}

		network := platform.Network()
		if network == nil {
			t.Fatal("Network provider is nil")
		}

		interfaces, err := network.Interfaces()
		if err != nil {
			t.Errorf("Failed to get network interfaces: %v", err)
		} else {
			t.Logf("Network Interfaces: %v", interfaces)

			// Test stats for first interface
			if len(interfaces) > 0 {
				stats, err := network.Stats(interfaces[0])
				if err != nil {
					t.Errorf("Failed to get stats for %s: %v", interfaces[0], err)
				} else {
					t.Logf("Interface %s: RX=%d bytes, TX=%d bytes, Errors=%d/%d",
						interfaces[0], stats.BytesRecv, stats.BytesSent, stats.ErrorsIn, stats.ErrorsOut)
				}
			}
		}
	})

	t.Run("Filesystem_Stats", func(t *testing.T) {
		platform, err := NewRemotePlatform(config)
		if err != nil {
			t.Fatalf("Failed to create remote platform: %v", err)
		}
		defer platform.Close()

		ctx := context.Background()
		if err := platform.Initialize(ctx); err != nil {
			t.Fatalf("Failed to initialize platform: %v", err)
		}

		filesystem := platform.Filesystem()
		if filesystem == nil {
			t.Fatal("Filesystem provider is nil")
		}

		mounts, err := filesystem.Mounts()
		if err != nil {
			t.Errorf("Failed to get mounts: %v", err)
		} else {
			t.Logf("Found %d mounts", len(mounts))

			// Test stats for root filesystem
			for _, mount := range mounts {
				if mount.MountPoint == "/" {
					stats, err := filesystem.Stats("/")
					if err != nil {
						t.Errorf("Failed to get stats for /: %v", err)
					} else {
						t.Logf("Root FS: Total=%dGB, Used=%dGB, Free=%dGB, %.2f%% used",
							stats.Total/1024/1024/1024, stats.Used/1024/1024/1024, stats.Free/1024/1024/1024, stats.UsedPercent)
					}
					break
				}
			}
		}
	})

	t.Run("Sensors", func(t *testing.T) {
		platform, err := NewRemotePlatform(config)
		if err != nil {
			t.Fatalf("Failed to create remote platform: %v", err)
		}
		defer platform.Close()

		ctx := context.Background()
		if err := platform.Initialize(ctx); err != nil {
			t.Fatalf("Failed to initialize platform: %v", err)
		}

		sensors := platform.Sensors()
		if sensors == nil {
			t.Log("Sensor monitoring not available on this platform")
			return
		}

		temps, err := sensors.Temperatures()
		if err != nil {
			t.Logf("Temperature sensors not available: %v", err)
		} else if len(temps) == 0 {
			t.Log("No temperature sensors found")
		} else {
			t.Logf("Found %d temperature sensors", len(temps))
			for _, temp := range temps {
				t.Logf("  %s: %.1f%s", temp.Label, temp.Value, temp.Unit)
			}
		}

		fans, err := sensors.Fans()
		if err != nil {
			t.Logf("Fan sensors not available: %v", err)
		} else if len(fans) == 0 {
			t.Log("No fan sensors found")
		} else {
			t.Logf("Found %d fan sensors", len(fans))
			for _, fan := range fans {
				t.Logf("  %s: %.0f%s", fan.Label, fan.Value, fan.Unit)
			}
		}
	})
}
