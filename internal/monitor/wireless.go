// Package monitor provides wireless network monitoring for Linux systems.
// It reads wireless statistics from /proc/net/wireless and wireless ESSID
// and access point information from /sys/class/net/<interface>/wireless/.
package monitor

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// WirelessInfo contains wireless network statistics for an interface.
type WirelessInfo struct {
	// ESSID is the Extended Service Set Identifier (network name).
	ESSID string
	// AccessPoint is the MAC address of the connected access point.
	AccessPoint string
	// LinkQuality is the link quality value (0-100 typically).
	LinkQuality int
	// LinkQualityMax is the maximum link quality value.
	LinkQualityMax int
	// SignalLevel is the signal level in dBm.
	SignalLevel int
	// NoiseLevel is the noise level in dBm.
	NoiseLevel int
	// BitRate is the bit rate in Mb/s.
	BitRate float64
	// Mode is the operating mode (Managed, Ad-Hoc, etc.).
	Mode string
	// IsWireless indicates if this interface is wireless.
	IsWireless bool
}

// wirelessReader reads wireless interface statistics from /proc and /sys.
type wirelessReader struct {
	mu                  sync.RWMutex
	procWirelessPath    string
	sysNetPath          string
	wirelessStatsCache  map[string]WirelessInfo
	operstatePath       func(iface string) string
	wirelessEssidPath   func(iface string) string
	wirelessApPath      func(iface string) string
	wirelessBitratePath func(iface string) string
	wirelessModePath    func(iface string) string
}

// newWirelessReader creates a new wirelessReader with default paths.
func newWirelessReader() *wirelessReader {
	const sysNetPathDefault = "/sys/class/net"
	return &wirelessReader{
		procWirelessPath:   "/proc/net/wireless",
		sysNetPath:         sysNetPathDefault,
		wirelessStatsCache: make(map[string]WirelessInfo),
		operstatePath: func(iface string) string {
			return sysNetPathDefault + "/" + iface + "/operstate"
		},
		wirelessEssidPath: func(iface string) string {
			// Modern path via iw command output or nl80211
			return sysNetPathDefault + "/" + iface + "/wireless/essid"
		},
		wirelessApPath: func(iface string) string {
			return sysNetPathDefault + "/" + iface + "/wireless/ap"
		},
		wirelessBitratePath: func(iface string) string {
			return sysNetPathDefault + "/" + iface + "/wireless/bitrate"
		},
		wirelessModePath: func(iface string) string {
			return sysNetPathDefault + "/" + iface + "/wireless/mode"
		},
	}
}

// ReadWirelessStats reads wireless statistics for all wireless interfaces.
func (r *wirelessReader) ReadWirelessStats() (map[string]WirelessInfo, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	result := make(map[string]WirelessInfo)

	// Parse /proc/net/wireless for basic stats
	procStats, err := r.parseProcWireless()
	if err != nil {
		// Not an error if file doesn't exist (no wireless interfaces)
		if os.IsNotExist(err) {
			return result, nil
		}
		return nil, fmt.Errorf("reading /proc/net/wireless: %w", err)
	}

	// Enhance with /sys data
	for iface, info := range procStats {
		enhanced := r.enhanceWithSysInfo(iface, info)
		result[iface] = enhanced
	}

	r.wirelessStatsCache = result
	return result, nil
}

// GetWirelessInfo returns wireless info for a specific interface.
func (r *wirelessReader) GetWirelessInfo(iface string) (WirelessInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	info, ok := r.wirelessStatsCache[iface]
	return info, ok
}

// parseProcWireless parses /proc/net/wireless for wireless statistics.
// Format:
// Inter-| sta-|   Quality        |   Discarded packets               | Missed | WE
//
//	face | tus | link level noise |  nwid  crypt   frag  retry   misc | beacon | 22
//
// wlan0: 0000  70.  -40.  -95.       0      0      0     0      0        0
func (r *wirelessReader) parseProcWireless() (map[string]WirelessInfo, error) {
	file, err := os.Open(r.procWirelessPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	result := make(map[string]WirelessInfo)
	scanner := bufio.NewScanner(file)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		// Skip header lines (first 2 lines)
		if lineNum <= 2 {
			continue
		}

		line := scanner.Text()
		info, iface, err := r.parseProcWirelessLine(line)
		if err != nil {
			continue // Skip malformed lines
		}

		result[iface] = info
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning /proc/net/wireless: %w", err)
	}

	return result, nil
}

// parseProcWirelessLine parses a single line from /proc/net/wireless.
func (r *wirelessReader) parseProcWirelessLine(line string) (WirelessInfo, string, error) {
	// Remove trailing dots from numbers (e.g., "70." -> "70")
	line = strings.ReplaceAll(line, ".", " ")

	// Split into fields
	fields := strings.Fields(line)
	if len(fields) < 4 {
		return WirelessInfo{}, "", fmt.Errorf("too few fields: %d", len(fields))
	}

	// First field is interface name with colon
	iface := strings.TrimSuffix(fields[0], ":")
	if iface == "" {
		return WirelessInfo{}, "", fmt.Errorf("empty interface name")
	}

	info := WirelessInfo{
		IsWireless:     true,
		LinkQualityMax: 100,       // Standard max
		Mode:           "Managed", // Default mode
	}

	// Parse status (field 1) - usually 0000
	// Parse link quality (field 2)
	if len(fields) > 2 {
		if val, err := strconv.Atoi(fields[2]); err == nil {
			info.LinkQuality = val
		}
	}

	// Parse signal level (field 3)
	if len(fields) > 3 {
		if val, err := strconv.Atoi(fields[3]); err == nil {
			info.SignalLevel = val
		}
	}

	// Parse noise level (field 4)
	if len(fields) > 4 {
		if val, err := strconv.Atoi(fields[4]); err == nil {
			info.NoiseLevel = val
		}
	}

	return info, iface, nil
}

// enhanceWithSysInfo adds ESSID, AP, bitrate info from /sys filesystem.
func (r *wirelessReader) enhanceWithSysInfo(iface string, info WirelessInfo) WirelessInfo {
	// Try to read ESSID from /sys/class/net/<iface>/wireless directory
	// Note: On modern systems, this may not be directly available in /sys
	// and requires iw or nl80211 for full info.

	// Check if wireless directory exists
	wirelessDir := filepath.Join(r.sysNetPath, iface, "wireless")
	if _, err := os.Stat(wirelessDir); err != nil {
		// Fallback: check operstate to see if interface is up
		if data, err := os.ReadFile(r.operstatePath(iface)); err == nil {
			state := strings.TrimSpace(string(data))
			if state == "down" {
				info.ESSID = ""
				info.AccessPoint = ""
			}
		}
		return info
	}

	// Try to read ESSID (may not be available via /sys)
	if data, err := os.ReadFile(r.wirelessEssidPath(iface)); err == nil {
		info.ESSID = strings.TrimSpace(string(data))
	}

	// Try to read access point MAC
	if data, err := os.ReadFile(r.wirelessApPath(iface)); err == nil {
		info.AccessPoint = strings.TrimSpace(string(data))
	}

	// Try to read bitrate
	if data, err := os.ReadFile(r.wirelessBitratePath(iface)); err == nil {
		if bitrate, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64); err == nil {
			info.BitRate = bitrate / 1000000 // Convert to Mb/s
		}
	}

	// Try to read mode
	if data, err := os.ReadFile(r.wirelessModePath(iface)); err == nil {
		info.Mode = strings.TrimSpace(string(data))
	}

	return info
}

// IsWirelessInterface checks if an interface is a wireless interface.
func (r *wirelessReader) IsWirelessInterface(iface string) bool {
	wirelessDir := filepath.Join(r.sysNetPath, iface, "wireless")
	if _, err := os.Stat(wirelessDir); err == nil {
		return true
	}

	// Also check /proc/net/wireless
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.wirelessStatsCache[iface]
	return ok
}

// LinkQualityPercent returns link quality as a percentage (0-100).
func (info WirelessInfo) LinkQualityPercent() int {
	if info.LinkQualityMax <= 0 {
		return 0
	}
	pct := (info.LinkQuality * 100) / info.LinkQualityMax
	if pct > 100 {
		pct = 100
	}
	if pct < 0 {
		pct = 0
	}
	return pct
}

// BitRateString returns the bit rate as a formatted string (e.g., "54Mb/s").
func (info WirelessInfo) BitRateString() string {
	if info.BitRate <= 0 {
		return "0Mb/s"
	}
	if info.BitRate >= 1000 {
		return fmt.Sprintf("%.1fGb/s", info.BitRate/1000)
	}
	return fmt.Sprintf("%.0fMb/s", info.BitRate)
}
