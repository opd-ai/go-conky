package monitor

import (
	"os"
	"testing"
)

func TestSysInfoReaderReadSystemInfo(t *testing.T) {
	reader := newSysInfoReader()
	info, err := reader.ReadSystemInfo()
	if err != nil {
		t.Fatalf("ReadSystemInfo failed: %v", err)
	}

	// Sysname should always be "Linux"
	if info.Sysname != "Linux" {
		t.Errorf("expected Sysname to be 'Linux', got %q", info.Sysname)
	}

	// Kernel version should not be empty
	if info.Kernel == "" {
		t.Error("expected Kernel to be non-empty")
	}

	// Hostname should not be empty
	if info.Hostname == "" {
		t.Error("expected Hostname to be non-empty")
	}

	// HostnameShort should not be empty
	if info.HostnameShort == "" {
		t.Error("expected HostnameShort to be non-empty")
	}

	// HostnameShort should not contain dots
	for _, c := range info.HostnameShort {
		if c == '.' {
			t.Error("HostnameShort should not contain dots")
			break
		}
	}
}

func TestSysInfoReaderReadLoadAvg(t *testing.T) {
	reader := newSysInfoReader()
	load1, load5, load15, err := reader.ReadLoadAvg()
	if err != nil {
		t.Fatalf("ReadLoadAvg failed: %v", err)
	}

	// Load averages should be non-negative
	if load1 < 0 {
		t.Errorf("expected load1 to be non-negative, got %f", load1)
	}
	if load5 < 0 {
		t.Errorf("expected load5 to be non-negative, got %f", load5)
	}
	if load15 < 0 {
		t.Errorf("expected load15 to be non-negative, got %f", load15)
	}
}

func TestSysInfoReaderWithMockPaths(t *testing.T) {
	// Create temp files for testing
	tmpDir := t.TempDir()

	// Create mock /proc/loadavg
	loadavgPath := tmpDir + "/loadavg"
	if err := os.WriteFile(loadavgPath, []byte("1.50 1.25 1.00 1/123 456\n"), 0o644); err != nil {
		t.Fatalf("failed to create mock loadavg: %v", err)
	}

	// Create mock /proc/sys/kernel/osrelease
	osreleasePath := tmpDir + "/osrelease"
	if err := os.WriteFile(osreleasePath, []byte("5.15.0-test-kernel\n"), 0o644); err != nil {
		t.Fatalf("failed to create mock osrelease: %v", err)
	}

	// Create mock /proc/sys/kernel/hostname
	hostnamePath := tmpDir + "/hostname"
	if err := os.WriteFile(hostnamePath, []byte("testhost.example.com\n"), 0o644); err != nil {
		t.Fatalf("failed to create mock hostname: %v", err)
	}

	reader := &sysInfoReader{
		procLoadavgPath: loadavgPath,
		procVersionPath: osreleasePath,
		procHostname:    hostnamePath,
	}

	info, err := reader.ReadSystemInfo()
	if err != nil {
		t.Fatalf("ReadSystemInfo failed: %v", err)
	}

	if info.Kernel != "5.15.0-test-kernel" {
		t.Errorf("expected Kernel to be '5.15.0-test-kernel', got %q", info.Kernel)
	}

	if info.Hostname != "testhost.example.com" {
		t.Errorf("expected Hostname to be 'testhost.example.com', got %q", info.Hostname)
	}

	if info.HostnameShort != "testhost" {
		t.Errorf("expected HostnameShort to be 'testhost', got %q", info.HostnameShort)
	}

	load1, load5, load15, err := reader.ReadLoadAvg()
	if err != nil {
		t.Fatalf("ReadLoadAvg failed: %v", err)
	}

	if load1 != 1.50 {
		t.Errorf("expected load1 to be 1.50, got %f", load1)
	}
	if load5 != 1.25 {
		t.Errorf("expected load5 to be 1.25, got %f", load5)
	}
	if load15 != 1.00 {
		t.Errorf("expected load15 to be 1.00, got %f", load15)
	}
}

func TestSysInfoReaderHostnameWithoutDot(t *testing.T) {
	tmpDir := t.TempDir()

	// Create mock hostname without domain
	hostnamePath := tmpDir + "/hostname"
	if err := os.WriteFile(hostnamePath, []byte("simplehost\n"), 0o644); err != nil {
		t.Fatalf("failed to create mock hostname: %v", err)
	}

	reader := &sysInfoReader{
		procLoadavgPath: "/proc/loadavg", // Use real path for load
		procVersionPath: "/proc/sys/kernel/osrelease",
		procHostname:    hostnamePath,
	}

	info, err := reader.ReadSystemInfo()
	if err != nil {
		t.Fatalf("ReadSystemInfo failed: %v", err)
	}

	if info.Hostname != "simplehost" {
		t.Errorf("expected Hostname to be 'simplehost', got %q", info.Hostname)
	}

	// For a hostname without a dot, short name should be the same
	if info.HostnameShort != "simplehost" {
		t.Errorf("expected HostnameShort to be 'simplehost', got %q", info.HostnameShort)
	}
}
