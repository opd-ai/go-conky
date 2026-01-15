// Package lua provides Golua integration for conky-go.
// This file implements the Conky Lua API including conky_parse()
// and the conky.info table for system monitoring data.
package lua

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	rt "github.com/arnodel/golua/runtime"

	"github.com/opd-ai/go-conky/internal/monitor"
)

// Version is the conky-go version string.
const Version = "0.1.0"

// SystemDataProvider is an interface for accessing system monitoring data.
// This allows for easy mocking in tests.
type SystemDataProvider interface {
	CPU() monitor.CPUStats
	Memory() monitor.MemoryStats
	Uptime() monitor.UptimeStats
	Network() monitor.NetworkStats
	Filesystem() monitor.FilesystemStats
	DiskIO() monitor.DiskIOStats
	Hwmon() monitor.HwmonStats
	Process() monitor.ProcessStats
	Battery() monitor.BatteryStats
	Audio() monitor.AudioStats
	SysInfo() monitor.SystemInfo
}

// ConkyAPI provides the Conky Lua API implementation.
// It registers conky_parse() and other Conky functions in the Lua environment.
type ConkyAPI struct {
	runtime     *ConkyRuntime
	sysProvider SystemDataProvider
	mu          sync.RWMutex
}

// NewConkyAPI creates a new ConkyAPI instance and registers all Conky functions
// in the provided Lua runtime.
func NewConkyAPI(runtime *ConkyRuntime, provider SystemDataProvider) (*ConkyAPI, error) {
	if runtime == nil {
		return nil, fmt.Errorf("runtime cannot be nil")
	}

	api := &ConkyAPI{
		runtime:     runtime,
		sysProvider: provider,
	}

	api.registerFunctions()
	return api, nil
}

// SetSystemDataProvider updates the system data provider.
// This is useful for testing or when changing data sources.
func (api *ConkyAPI) SetSystemDataProvider(provider SystemDataProvider) {
	api.mu.Lock()
	defer api.mu.Unlock()
	api.sysProvider = provider
}

// registerFunctions registers all Conky API functions in the Lua environment.
func (api *ConkyAPI) registerFunctions() {
	// Register conky_parse function
	api.runtime.SetGoFunction("conky_parse", api.conkyParseLua, 1, false)

	// Setup the conky global table with info subtable
	api.setupConkyTable()
}

// setupConkyTable creates the conky global table with the info subtable.
func (api *ConkyAPI) setupConkyTable() {
	// Create main conky table
	conkyTable := rt.NewTable()

	// Create config subtable (empty for now, will be populated by config parser)
	configTable := rt.NewTable()
	conkyTable.Set(rt.StringValue("config"), rt.TableValue(configTable))

	// Create text field (empty for now)
	conkyTable.Set(rt.StringValue("text"), rt.StringValue(""))

	// Set the conky global
	api.runtime.SetGlobal("conky", rt.TableValue(conkyTable))
}

// conkyParseLua is the Lua-callable implementation of conky_parse.
// It takes a template string and returns the parsed result with variables replaced.
func (api *ConkyAPI) conkyParseLua(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	template, err := c.StringArg(0)
	if err != nil {
		return nil, fmt.Errorf("conky_parse: %w", err)
	}

	result := api.Parse(template)
	return c.PushingNext1(t.Runtime, rt.StringValue(result)), nil
}

// variablePattern matches Conky variables in the format ${variable} or ${variable arg}
var variablePattern = regexp.MustCompile(`\$\{([^}]+)\}`)

// Parse parses a Conky template string and replaces variables with their values.
// Supported formats:
//   - ${variable} - simple variable
//   - ${variable arg} - variable with argument
//   - ${variable arg1 arg2} - variable with multiple arguments
func (api *ConkyAPI) Parse(template string) string {
	api.mu.RLock()
	defer api.mu.RUnlock()

	return variablePattern.ReplaceAllStringFunc(template, func(match string) string {
		// Extract variable name and arguments from ${variable args...}
		inner := match[2 : len(match)-1] // Remove ${ and }
		parts := strings.Fields(inner)
		if len(parts) == 0 {
			return match
		}

		varName := parts[0]
		args := parts[1:]

		return api.resolveVariable(varName, args)
	})
}

// formatUnknownVariable formats an unknown variable back to its original template form.
func formatUnknownVariable(name string, args []string) string {
	if len(args) > 0 {
		return fmt.Sprintf("${%s %s}", name, strings.Join(args, " "))
	}
	return fmt.Sprintf("${%s}", name)
}

// resolveVariable resolves a single Conky variable to its value.
func (api *ConkyAPI) resolveVariable(name string, args []string) string {
	// Handle case where there's no system data provider
	if api.sysProvider == nil {
		return formatUnknownVariable(name, args)
	}

	switch name {
	// CPU variables
	case "cpu":
		return api.resolveCPU(args)
	case "freq":
		return api.resolveCPUFreq(args)
	case "freq_g":
		return api.resolveCPUFreqGHz(args)
	case "cpu_model":
		return api.sysProvider.CPU().ModelName

	// Memory variables
	case "mem":
		return formatBytes(api.sysProvider.Memory().Used)
	case "memmax":
		return formatBytes(api.sysProvider.Memory().Total)
	case "memfree":
		return formatBytes(api.sysProvider.Memory().Free)
	case "memperc":
		return fmt.Sprintf("%.0f", api.sysProvider.Memory().UsagePercent)
	case "memeasyfree":
		return formatBytes(api.sysProvider.Memory().Available)
	case "buffers":
		return formatBytes(api.sysProvider.Memory().Buffers)
	case "cached":
		return formatBytes(api.sysProvider.Memory().Cached)
	case "swap":
		return formatBytes(api.sysProvider.Memory().SwapUsed)
	case "swapmax":
		return formatBytes(api.sysProvider.Memory().SwapTotal)
	case "swapfree":
		return formatBytes(api.sysProvider.Memory().SwapFree)
	case "swapperc":
		return fmt.Sprintf("%.0f", api.sysProvider.Memory().SwapPercent)

	// Uptime variables
	case "uptime":
		return api.formatUptime(api.sysProvider.Uptime())
	case "uptime_short":
		return api.formatUptimeShort(api.sysProvider.Uptime())

	// Network variables
	case "downspeed":
		return api.resolveNetworkSpeed(args, true)
	case "upspeed":
		return api.resolveNetworkSpeed(args, false)
	case "totaldown":
		return api.resolveNetworkTotal(args, true)
	case "totalup":
		return api.resolveNetworkTotal(args, false)

	// Filesystem variables
	case "fs_used":
		return api.resolveFSUsed(args)
	case "fs_size":
		return api.resolveFSSize(args)
	case "fs_free":
		return api.resolveFSFree(args)
	case "fs_used_perc":
		return api.resolveFSUsedPerc(args)

	// Disk I/O variables
	case "diskio":
		return api.resolveDiskIO(args)
	case "diskio_read":
		return api.resolveDiskIORead(args)
	case "diskio_write":
		return api.resolveDiskIOWrite(args)

	// Process variables
	case "processes":
		return strconv.Itoa(api.sysProvider.Process().TotalProcesses)
	case "running_processes":
		return strconv.Itoa(api.sysProvider.Process().RunningProcesses)
	case "threads":
		return strconv.Itoa(api.sysProvider.Process().TotalThreads)

	// Battery variables
	case "battery_percent":
		return api.resolveBatteryPercent(args)
	case "battery_short":
		return api.resolveBatteryShort(args)

	// Hardware monitoring
	case "hwmon":
		return api.resolveHwmon(args)

	// Audio variables
	case "mixer":
		return api.resolveMixer(args)

	// System info variables
	case "kernel":
		return api.sysProvider.SysInfo().Kernel
	case "nodename":
		return api.sysProvider.SysInfo().Hostname
	case "nodename_short":
		return api.sysProvider.SysInfo().HostnameShort
	case "sysname":
		return api.sysProvider.SysInfo().Sysname
	case "machine":
		return api.sysProvider.SysInfo().Machine
	case "conky_version":
		return Version
	case "conky_build_arch":
		return api.sysProvider.SysInfo().Machine

	// Load average variables
	case "loadavg":
		return api.resolveLoadAvg(args)

	// Time variables
	case "time":
		return api.resolveTime(args)

	default:
		// Return original if unknown variable
		return formatUnknownVariable(name, args)
	}
}

// resolveCPU resolves the ${cpu} or ${cpu N} variable.
func (api *ConkyAPI) resolveCPU(args []string) string {
	cpuStats := api.sysProvider.CPU()

	if len(args) == 0 {
		return fmt.Sprintf("%.0f", cpuStats.UsagePercent)
	}

	// Parse CPU core number
	coreNum, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Sprintf("%.0f", cpuStats.UsagePercent)
	}

	// Conky uses 1-based indexing for cores
	coreIdx := coreNum - 1
	if coreIdx < 0 || coreIdx >= len(cpuStats.Cores) {
		return "0"
	}

	return fmt.Sprintf("%.0f", cpuStats.Cores[coreIdx])
}

// resolveCPUFreq resolves the ${freq} variable (MHz).
func (api *ConkyAPI) resolveCPUFreq(_ []string) string {
	return fmt.Sprintf("%.0f", api.sysProvider.CPU().Frequency)
}

// resolveCPUFreqGHz resolves the ${freq_g} variable (GHz).
func (api *ConkyAPI) resolveCPUFreqGHz(_ []string) string {
	return fmt.Sprintf("%.2f", api.sysProvider.CPU().Frequency/1000)
}

// formatUptime formats uptime in the format "Xd Xh Xm Xs".
func (api *ConkyAPI) formatUptime(stats monitor.UptimeStats) string {
	seconds := int64(stats.Seconds)
	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, secs)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, secs)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, secs)
	}
	return fmt.Sprintf("%ds", secs)
}

// formatUptimeShort formats uptime in short format "Xd Xh Xm".
func (api *ConkyAPI) formatUptimeShort(stats monitor.UptimeStats) string {
	seconds := int64(stats.Seconds)
	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	minutes := (seconds % 3600) / 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

// resolveNetworkSpeed resolves ${downspeed} or ${upspeed} variables.
func (api *ConkyAPI) resolveNetworkSpeed(args []string, isDownload bool) string {
	netStats := api.sysProvider.Network()

	var speed float64
	if len(args) == 0 {
		// Total across all interfaces
		if isDownload {
			speed = netStats.TotalRxBytesPerSec
		} else {
			speed = netStats.TotalTxBytesPerSec
		}
	} else {
		// Specific interface
		iface, ok := netStats.Interfaces[args[0]]
		if !ok {
			return "0B"
		}
		if isDownload {
			speed = iface.RxBytesPerSec
		} else {
			speed = iface.TxBytesPerSec
		}
	}

	return formatSpeed(speed)
}

// resolveNetworkTotal resolves ${totaldown} or ${totalup} variables.
func (api *ConkyAPI) resolveNetworkTotal(args []string, isDownload bool) string {
	netStats := api.sysProvider.Network()

	var total uint64
	if len(args) == 0 {
		if isDownload {
			total = netStats.TotalRxBytes
		} else {
			total = netStats.TotalTxBytes
		}
	} else {
		iface, ok := netStats.Interfaces[args[0]]
		if !ok {
			return "0B"
		}
		if isDownload {
			total = iface.RxBytes
		} else {
			total = iface.TxBytes
		}
	}

	return formatBytes(total)
}

// resolveFSUsed resolves ${fs_used} variable.
func (api *ConkyAPI) resolveFSUsed(args []string) string {
	mountPoint := "/"
	if len(args) > 0 {
		mountPoint = args[0]
	}

	fsStats := api.sysProvider.Filesystem()
	mount, ok := fsStats.Mounts[mountPoint]
	if !ok {
		return "0B"
	}
	return formatBytes(mount.Used)
}

// resolveFSSize resolves ${fs_size} variable.
func (api *ConkyAPI) resolveFSSize(args []string) string {
	mountPoint := "/"
	if len(args) > 0 {
		mountPoint = args[0]
	}

	fsStats := api.sysProvider.Filesystem()
	mount, ok := fsStats.Mounts[mountPoint]
	if !ok {
		return "0B"
	}
	return formatBytes(mount.Total)
}

// resolveFSFree resolves ${fs_free} variable.
func (api *ConkyAPI) resolveFSFree(args []string) string {
	mountPoint := "/"
	if len(args) > 0 {
		mountPoint = args[0]
	}

	fsStats := api.sysProvider.Filesystem()
	mount, ok := fsStats.Mounts[mountPoint]
	if !ok {
		return "0B"
	}
	return formatBytes(mount.Available)
}

// resolveFSUsedPerc resolves ${fs_used_perc} variable.
func (api *ConkyAPI) resolveFSUsedPerc(args []string) string {
	mountPoint := "/"
	if len(args) > 0 {
		mountPoint = args[0]
	}

	fsStats := api.sysProvider.Filesystem()
	mount, ok := fsStats.Mounts[mountPoint]
	if !ok {
		return "0"
	}
	return fmt.Sprintf("%.0f", mount.UsagePercent)
}

// resolveBatteryPercent resolves ${battery_percent} variable.
func (api *ConkyAPI) resolveBatteryPercent(args []string) string {
	batStats := api.sysProvider.Battery()

	if len(args) > 0 {
		// Specific battery
		bat, ok := batStats.Batteries[args[0]]
		if !ok {
			return "0"
		}
		return strconv.Itoa(bat.Capacity)
	}

	// Return average capacity across all batteries
	return fmt.Sprintf("%.0f", batStats.TotalCapacity)
}

// resolveBatteryShort resolves ${battery_short} variable.
func (api *ConkyAPI) resolveBatteryShort(_ []string) string {
	batStats := api.sysProvider.Battery()

	var status string
	switch {
	case batStats.IsCharging:
		status = "C"
	case batStats.IsDischarging:
		status = "D"
	default:
		status = "F"
	}

	return fmt.Sprintf("%s %.0f%%", status, batStats.TotalCapacity)
}

// resolveHwmon resolves ${hwmon} variable.
func (api *ConkyAPI) resolveHwmon(args []string) string {
	hwmonStats := api.sysProvider.Hwmon()

	// Default to first temperature sensor if no args
	if len(hwmonStats.TempSensors) == 0 {
		return "0"
	}

	// If we have temp sensors, return the first one's current value
	if len(args) == 0 {
		return fmt.Sprintf("%.0f", hwmonStats.TempSensors[0].InputCelsius)
	}

	// Try to find sensor by index
	idx, err := strconv.Atoi(args[0])
	if err != nil || idx < 0 || idx >= len(hwmonStats.TempSensors) {
		return "0"
	}

	return fmt.Sprintf("%.0f", hwmonStats.TempSensors[idx].InputCelsius)
}

// resolveDiskIO resolves ${diskio} variable.
// Returns total disk I/O speed (read + write) for a device.
// If no device is specified, returns total for all devices.
// Format: "R+W speed" (e.g., "1.2MiB/s").
func (api *ConkyAPI) resolveDiskIO(args []string) string {
	diskStats := api.sysProvider.DiskIO()

	if len(args) > 0 {
		// Return I/O for specific device
		device := args[0]
		disk, ok := diskStats.Disks[device]
		if !ok {
			return "0B/s"
		}
		totalSpeed := disk.ReadBytesPerSec + disk.WriteBytesPerSec
		return formatSpeed(totalSpeed)
	}

	// Return total I/O across all devices
	var totalRead, totalWrite float64
	for _, disk := range diskStats.Disks {
		totalRead += disk.ReadBytesPerSec
		totalWrite += disk.WriteBytesPerSec
	}

	return formatSpeed(totalRead + totalWrite)
}

// resolveDiskIORead resolves ${diskio_read} variable.
// Returns disk read speed for a device.
// If no device is specified, returns total for all devices.
func (api *ConkyAPI) resolveDiskIORead(args []string) string {
	diskStats := api.sysProvider.DiskIO()

	if len(args) > 0 {
		// Return read speed for specific device
		device := args[0]
		disk, ok := diskStats.Disks[device]
		if !ok {
			return "0B/s"
		}
		return formatSpeed(disk.ReadBytesPerSec)
	}

	// Return total read speed across all devices
	var totalRead float64
	for _, disk := range diskStats.Disks {
		totalRead += disk.ReadBytesPerSec
	}

	return formatSpeed(totalRead)
}

// resolveDiskIOWrite resolves ${diskio_write} variable.
// Returns disk write speed for a device.
// If no device is specified, returns total for all devices.
func (api *ConkyAPI) resolveDiskIOWrite(args []string) string {
	diskStats := api.sysProvider.DiskIO()

	if len(args) > 0 {
		// Return write speed for specific device
		device := args[0]
		disk, ok := diskStats.Disks[device]
		if !ok {
			return "0B/s"
		}
		return formatSpeed(disk.WriteBytesPerSec)
	}

	// Return total write speed across all devices
	var totalWrite float64
	for _, disk := range diskStats.Disks {
		totalWrite += disk.WriteBytesPerSec
	}

	return formatSpeed(totalWrite)
}

// resolveMixer resolves ${mixer} variable.
func (api *ConkyAPI) resolveMixer(_ []string) string {
	audioStats := api.sysProvider.Audio()

	if !audioStats.HasAudio {
		return "0"
	}

	return fmt.Sprintf("%.0f", audioStats.MasterVolume)
}

// formatBytes formats bytes to human-readable format (e.g., "1.5GiB").
func formatBytes(bytes uint64) string {
	const (
		_ = 1 << (10 * iota)
		KiB
		MiB
		GiB
		TiB
	)

	switch {
	case bytes >= TiB:
		return fmt.Sprintf("%.1fTiB", float64(bytes)/TiB)
	case bytes >= GiB:
		return fmt.Sprintf("%.1fGiB", float64(bytes)/GiB)
	case bytes >= MiB:
		return fmt.Sprintf("%.1fMiB", float64(bytes)/MiB)
	case bytes >= KiB:
		return fmt.Sprintf("%.1fKiB", float64(bytes)/KiB)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

// formatSpeed formats speed to human-readable format (e.g., "1.5KiB/s").
func formatSpeed(bytesPerSec float64) string {
	const (
		_ = 1 << (10 * iota)
		KiB
		MiB
		GiB
	)

	switch {
	case bytesPerSec >= GiB:
		return fmt.Sprintf("%.1fGiB/s", bytesPerSec/GiB)
	case bytesPerSec >= MiB:
		return fmt.Sprintf("%.1fMiB/s", bytesPerSec/MiB)
	case bytesPerSec >= KiB:
		return fmt.Sprintf("%.1fKiB/s", bytesPerSec/KiB)
	default:
		return fmt.Sprintf("%.0fB/s", bytesPerSec)
	}
}

// resolveLoadAvg resolves the ${loadavg} variable.
// Accepts an optional argument to select which load average to return:
// - No argument: all three load averages ("load1 load5 load15")
// - "1": 1-minute load average
// - "5": 5-minute load average
// - "15": 15-minute load average
func (api *ConkyAPI) resolveLoadAvg(args []string) string {
	sysInfo := api.sysProvider.SysInfo()

	if len(args) == 0 {
		// Default: return all three load averages
		return fmt.Sprintf("%.2f %.2f %.2f", sysInfo.LoadAvg1, sysInfo.LoadAvg5, sysInfo.LoadAvg15)
	}

	switch args[0] {
	case "1":
		return fmt.Sprintf("%.2f", sysInfo.LoadAvg1)
	case "5":
		return fmt.Sprintf("%.2f", sysInfo.LoadAvg5)
	case "15":
		return fmt.Sprintf("%.2f", sysInfo.LoadAvg15)
	default:
		// Return all three for any other argument
		return fmt.Sprintf("%.2f %.2f %.2f", sysInfo.LoadAvg1, sysInfo.LoadAvg5, sysInfo.LoadAvg15)
	}
}

// resolveTime resolves the ${time} variable.
// Accepts an optional format string argument.
// If no format is provided, uses "%c" (locale-appropriate date and time).
// Supports standard strftime format specifiers.
func (api *ConkyAPI) resolveTime(args []string) string {
	now := time.Now()

	if len(args) == 0 {
		// Default format: Mon Jan 2 15:04:05 2006
		return now.Format("Mon Jan 2 15:04:05 2006")
	}

	// Join args in case the format string has spaces
	format := strings.Join(args, " ")

	// Convert strftime format to Go format
	return formatTime(now, format)
}

// formatTime converts a strftime format string to Go time format.
// This supports common strftime specifiers used in Conky configurations.
func formatTime(t time.Time, format string) string {
	result := format

	// Handle %% first to avoid conflicts with other specifiers
	result = strings.ReplaceAll(result, "%%", "\x00PERCENT\x00")

	// Handle special cases that need calculation (do these before static replacements)
	result = strings.ReplaceAll(result, "%C", fmt.Sprintf("%02d", t.Year()/100))
	result = strings.ReplaceAll(result, "%j", fmt.Sprintf("%03d", t.YearDay()))
	result = strings.ReplaceAll(result, "%u", fmt.Sprintf("%d", (int(t.Weekday())+6)%7+1))
	result = strings.ReplaceAll(result, "%w", fmt.Sprintf("%d", int(t.Weekday())))

	// Map of strftime specifiers to Go time format values
	// These are replaced with the formatted time value directly
	staticReplacements := []struct {
		strftime string
		goFormat string
	}{
		{"%A", "Monday"},                  // Full weekday name
		{"%a", "Mon"},                     // Abbreviated weekday name
		{"%B", "January"},                 // Full month name
		{"%b", "Jan"},                     // Abbreviated month name
		{"%c", "Mon Jan 2 15:04:05 2006"}, // Locale date and time
		{"%D", "01/02/06"},                // Equivalent to %m/%d/%y
		{"%d", "02"},                      // Day of month (01-31)
		{"%e", "_2"},                      // Day of month, space padded
		{"%F", "2006-01-02"},              // Equivalent to %Y-%m-%d
		{"%H", "15"},                      // Hour (00-23)
		{"%I", "03"},                      // Hour (01-12)
		{"%k", "15"},                      // Hour (0-23), space padded
		{"%l", "3"},                       // Hour (1-12), space padded
		{"%M", "04"},                      // Minute (00-59)
		{"%m", "01"},                      // Month (01-12)
		{"%n", "\n"},                      // Newline
		{"%P", "pm"},                      // am/pm
		{"%p", "PM"},                      // AM/PM
		{"%R", "15:04"},                   // 24-hour HH:MM
		{"%r", "03:04:05 PM"},             // 12-hour time
		{"%S", "05"},                      // Second (00-59)
		{"%T", "15:04:05"},                // 24-hour HH:MM:SS
		{"%t", "\t"},                      // Tab
		{"%X", "15:04:05"},                // Locale time
		{"%x", "01/02/06"},                // Locale date
		{"%Y", "2006"},                    // Year with century
		{"%y", "06"},                      // Year without century
		{"%Z", "MST"},                     // Timezone name
		{"%z", "-0700"},                   // Timezone offset
	}

	// Replace each strftime specifier with formatted time value
	for _, repl := range staticReplacements {
		if strings.Contains(result, repl.strftime) {
			result = strings.ReplaceAll(result, repl.strftime, t.Format(repl.goFormat))
		}
	}

	// Restore literal percent signs
	result = strings.ReplaceAll(result, "\x00PERCENT\x00", "%")

	return result
}
