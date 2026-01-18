// Package monitor provides system monitoring functionality for Linux systems.
// This file implements system information collection including kernel version,
// hostname, load averages, and OS information.
package monitor

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
)

// SystemInfo contains static and dynamic system information.
type SystemInfo struct {
	// Kernel is the kernel version (e.g., "5.15.0-generic").
	Kernel string
	// Hostname is the full hostname.
	Hostname string
	// HostnameShort is the short hostname (before first dot).
	HostnameShort string
	// Sysname is the OS name (e.g., "Linux").
	Sysname string
	// Machine is the machine hardware name (e.g., "x86_64").
	Machine string
	// LoadAvg1 is the 1-minute load average.
	LoadAvg1 float64
	// LoadAvg5 is the 5-minute load average.
	LoadAvg5 float64
	// LoadAvg15 is the 15-minute load average.
	LoadAvg15 float64
}

// sysInfoReader reads system information from /proc and other sources.
type sysInfoReader struct {
	procLoadavgPath string
	procVersionPath string
	procHostname    string
}

// newSysInfoReader creates a new sysInfoReader with default paths.
func newSysInfoReader() *sysInfoReader {
	return &sysInfoReader{
		procLoadavgPath: "/proc/loadavg",
		procVersionPath: "/proc/sys/kernel/osrelease",
		procHostname:    "/proc/sys/kernel/hostname",
	}
}

// ReadSystemInfo reads all system information.
func (r *sysInfoReader) ReadSystemInfo() (SystemInfo, error) {
	info := SystemInfo{
		Sysname: getSysname(),
		Machine: r.getMachine(),
	}

	// Read kernel version
	kernel, err := r.readKernelVersion()
	if err == nil {
		info.Kernel = kernel
	}

	// Read hostname
	hostname, err := r.readHostname()
	if err == nil {
		info.Hostname = hostname
		// Short hostname is before the first dot
		if idx := strings.Index(hostname, "."); idx > 0 {
			info.HostnameShort = hostname[:idx]
		} else {
			info.HostnameShort = hostname
		}
	}

	// Read load averages
	load1, load5, load15, err := r.readLoadAvg()
	if err == nil {
		info.LoadAvg1 = load1
		info.LoadAvg5 = load5
		info.LoadAvg15 = load15
	}

	return info, nil
}

// ReadLoadAvg reads the current load averages.
func (r *sysInfoReader) ReadLoadAvg() (load1, load5, load15 float64, err error) {
	return r.readLoadAvg()
}

// readKernelVersion reads the kernel version from /proc/sys/kernel/osrelease.
func (r *sysInfoReader) readKernelVersion() (string, error) {
	data, err := os.ReadFile(r.procVersionPath)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// readHostname reads the hostname from /proc/sys/kernel/hostname.
func (r *sysInfoReader) readHostname() (string, error) {
	data, err := os.ReadFile(r.procHostname)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// readLoadAvg reads load averages from /proc/loadavg.
func (r *sysInfoReader) readLoadAvg() (load1, load5, load15 float64, err error) {
	file, err := os.Open(r.procLoadavgPath)
	if err != nil {
		return 0, 0, 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return 0, 0, 0, scanner.Err()
	}

	fields := strings.Fields(scanner.Text())
	if len(fields) < 3 {
		return 0, 0, 0, nil
	}

	var err1, err2, err3 error
	load1, err1 = strconv.ParseFloat(fields[0], 64)
	load5, err2 = strconv.ParseFloat(fields[1], 64)
	load15, err3 = strconv.ParseFloat(fields[2], 64)

	// Return first error if any parsing failed (but return partial results)
	if err1 != nil {
		return 0, 0, 0, fmt.Errorf("parsing load1: %w", err1)
	}
	if err2 != nil {
		return load1, 0, 0, fmt.Errorf("parsing load5: %w", err2)
	}
	if err3 != nil {
		return load1, load5, 0, fmt.Errorf("parsing load15: %w", err3)
	}

	return load1, load5, load15, nil
}

// getMachine returns the machine hardware name.
// Uses Go's runtime.GOARCH for reliable architecture detection.
func (r *sysInfoReader) getMachine() string {
	// Use Go's built-in architecture detection which is reliable
	// and aligns with the 'lazy programmer' philosophy
	switch runtime.GOARCH {
	case "amd64":
		return "x86_64"
	case "386":
		return "i686"
	case "arm64":
		return "aarch64"
	case "arm":
		return "armv7l"
	default:
		return runtime.GOARCH
	}
}

// getSysname returns the system name (OS name) based on runtime.GOOS.
// This maps Go's OS identifiers to conventional system names matching
// what uname -s would return on POSIX systems.
func getSysname() string {
	switch runtime.GOOS {
	case "linux":
		return "Linux"
	case "darwin":
		return "Darwin"
	case "windows":
		return "Windows"
	case "freebsd":
		return "FreeBSD"
	case "openbsd":
		return "OpenBSD"
	case "netbsd":
		return "NetBSD"
	case "dragonfly":
		return "DragonFly"
	case "android":
		// Android uses Linux kernel, so return Linux for compatibility
		return "Linux"
	case "solaris", "illumos":
		// Both Solaris and illumos report "SunOS" from uname -s for compatibility.
		// illumos is a fork of OpenSolaris but shares the SunOS kernel interface.
		return "SunOS"
	default:
		// For unknown platforms, return GOOS as-is.
		// Go's runtime.GOOS values are lowercase identifiers, but for unknown
		// platforms it's safer to return them unchanged rather than attempting
		// case conversion that could fail on non-ASCII or edge cases.
		return runtime.GOOS
	}
}
