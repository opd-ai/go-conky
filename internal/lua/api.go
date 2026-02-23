// Package lua provides Golua integration for conky-go.
// This file implements the Conky Lua API including conky_parse()
// and the conky.info table for system monitoring data.
package lua

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	rt "github.com/arnodel/golua/runtime"

	"github.com/opd-ai/go-conky/internal/monitor"
	"github.com/opd-ai/go-conky/internal/render"
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
	GPU() monitor.GPUStats
	Mail() monitor.MailStats
	MailUnseenCount(name string) int
	MailTotalCount(name string) int
	MailTotalUnseen() int
	MailTotalMessages() int
	Weather(stationID string) monitor.WeatherStats
	TCP() monitor.TCPStats
	TCPCountInRange(minPort, maxPort int) int
	TCPConnectionByIndex(minPort, maxPort, index int) *monitor.TCPConnection
}

// execCacheEntry stores cached output from execi commands.
type execCacheEntry struct {
	output       string
	expiresAt    time.Time
	lastAccessed time.Time // Tracks when this entry was last accessed
}

// scrollState tracks the current scroll position for a scroll instance.
type scrollState struct {
	position     int       // Current scroll offset
	lastUpdate   time.Time // When the scroll was last advanced
	lastAccessed time.Time // Tracks when this state was last accessed
}

// CacheCleanupConfig holds configuration for cache cleanup behavior.
type CacheCleanupConfig struct {
	// MaxAge is the maximum time an unused cache entry can exist before cleanup.
	MaxAge time.Duration
	// CleanupInterval is how often the background cleanup runs.
	CleanupInterval time.Duration
}

// DefaultCacheCleanupConfig returns sensible defaults for cache cleanup.
func DefaultCacheCleanupConfig() CacheCleanupConfig {
	return CacheCleanupConfig{
		MaxAge:          5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
	}
}

// ConkyAPI provides the Conky Lua API implementation.
// It registers conky_parse() and other Conky functions in the Lua environment.
type ConkyAPI struct {
	runtime        *ConkyRuntime
	sysProvider    SystemDataProvider
	execCache      map[string]*execCacheEntry
	scrollStates   map[string]*scrollState
	templates      [10]string // template0-template9 definitions
	mu             sync.RWMutex
	cleanupConfig  CacheCleanupConfig
	cleanupStop    chan struct{}
	cleanupRunning bool
}

// NewConkyAPI creates a new ConkyAPI instance and registers all Conky functions
// in the provided Lua runtime.
func NewConkyAPI(runtime *ConkyRuntime, provider SystemDataProvider) (*ConkyAPI, error) {
	if runtime == nil {
		return nil, fmt.Errorf("runtime cannot be nil")
	}

	api := &ConkyAPI{
		runtime:       runtime,
		sysProvider:   provider,
		execCache:     make(map[string]*execCacheEntry),
		scrollStates:  make(map[string]*scrollState),
		cleanupConfig: DefaultCacheCleanupConfig(),
		cleanupStop:   make(chan struct{}),
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

// SetTemplates sets the template0-template9 definitions.
// Templates can use \1, \2, etc. as argument placeholders.
func (api *ConkyAPI) SetTemplates(templates [10]string) {
	api.mu.Lock()
	defer api.mu.Unlock()
	api.templates = templates
}

// GetTemplate returns the template at the given index (0-9).
func (api *ConkyAPI) GetTemplate(index int) string {
	if index < 0 || index > 9 {
		return ""
	}
	api.mu.RLock()
	defer api.mu.RUnlock()
	return api.templates[index]
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
//
// Conditional blocks are also supported:
//   - ${if_up interface}content${endif}
//   - ${if_up interface}content${else}alternative${endif}
//   - ${if_existing path}content${endif}
//   - ${if_running process}content${endif}
//   - ${if_match value pattern}content${endif}
//   - ${if_empty value}content${endif}
//   - ${if_mounted path}content${endif}
//
// Thread safety: This function does not hold locks during parsing to avoid
// deadlocks when resolving variables that need to acquire locks (e.g., execi
// with cache). The sysProvider is accessed via resolveVariable which does a
// brief RLock to read the pointer. The provider's methods are expected to be
// thread-safe. SetSystemDataProvider should not be called concurrently with
// Parse in production; it's intended for initialization and testing.
func (api *ConkyAPI) Parse(template string) string {
	// First, process conditional blocks
	processed := api.parseConditionals(template)

	// Then replace variables
	return variablePattern.ReplaceAllStringFunc(processed, func(match string) string {
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
// Thread safety: Reads sysProvider with a brief RLock. The individual provider
// methods (CPU(), Memory(), etc.) are expected to be thread-safe. The brief
// lock protects against the provider pointer changing during nil check.
func (api *ConkyAPI) resolveVariable(name string, args []string) string {
	// Read sysProvider with read lock for safe concurrent access.
	// We only lock briefly to get the pointer; provider methods are thread-safe.
	api.mu.RLock()
	provider := api.sysProvider
	api.mu.RUnlock()

	// Handle case where there's no system data provider
	if provider == nil {
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
	case "addr":
		return api.resolveAddr(args)
	case "addrs":
		return api.resolveAddrs(args)
	case "gw_ip":
		return api.sysProvider.Network().GatewayIP
	case "gw_iface":
		return api.sysProvider.Network().GatewayInterface
	case "nameserver":
		return api.resolveNameserver(args)

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

	// Top process variables
	case "top":
		return api.resolveTop(args, false)
	case "top_mem":
		return api.resolveTop(args, true)

	// Exec variables - execute shell commands
	case "exec":
		return api.resolveExec(args)
	case "execp":
		return api.resolveExec(args) // Same as exec, parsing handled elsewhere
	case "execi":
		return api.resolveExeci(args)
	case "execpi":
		return api.resolveExeci(args) // Same as execi, parsing handled elsewhere

	// Text formatting variables (return empty/spacing - actual formatting in renderer)
	case "color", "color0", "color1", "color2", "color3", "color4",
		"color5", "color6", "color7", "color8", "color9":
		return "" // Colors handled by renderer
	case "font":
		return "" // Font changes handled by renderer
	case "alignr":
		return "" // Right alignment marker
	case "alignc":
		return "" // Center alignment marker
	case "voffset":
		return "" // Vertical offset handled by renderer
	case "offset":
		return "" // Horizontal offset handled by renderer
	case "goto":
		return "" // Goto position handled by renderer
	case "tab":
		return "\t"
	case "hr":
		return api.resolveHR(args)

	// Additional filesystem variables
	case "fs_bar":
		return api.resolveFSBar(args)
	case "fs_type":
		return api.resolveFSType(args)

	// Additional CPU variables
	case "cpu_count", "cpu_cores":
		return strconv.Itoa(api.sysProvider.CPU().CPUCount)

	// Additional memory variables
	case "memwithbuffers":
		mem := api.sysProvider.Memory()
		return formatBytes(mem.Used - mem.Buffers - mem.Cached)

	// Additional battery variables
	case "battery":
		return api.resolveBattery(args)
	case "battery_bar":
		return api.resolveBatteryBar(args)
	case "battery_time":
		return api.resolveBatteryTime(args)

	// Platform/environment variables
	case "user_names", "user_name":
		return os.Getenv("USER")
	case "desktop_name":
		return os.Getenv("XDG_CURRENT_DESKTOP")
	case "uid":
		return strconv.Itoa(os.Getuid())
	case "gid":
		return strconv.Itoa(os.Getgid())

	// Additional network variables
	case "downspeedf":
		return api.resolveNetworkSpeedF(args, true)
	case "upspeedf":
		return api.resolveNetworkSpeedF(args, false)

	// Conditional stubs (return content as-is, conditions evaluated elsewhere)
	case "if_up":
		return api.resolveIfUp(args)

	// Additional network variables
	case "wireless_essid":
		return api.resolveWirelessESSID(args)
	case "wireless_link_qual":
		return api.resolveWirelessLinkQual(args)
	case "wireless_link_qual_perc":
		return api.resolveWirelessLinkQualPerc(args)
	case "wireless_link_qual_max":
		return "100" // Standard max
	case "wireless_bitrate":
		return api.resolveWirelessBitrate(args)
	case "wireless_ap":
		return api.resolveWirelessAP(args)
	case "wireless_mode":
		return api.resolveWirelessMode(args)

	// Network packet/error variables
	case "tcp_portmon":
		return api.resolveTCPPortMon(args)
	case "if_existing":
		return api.resolveIfExisting(args)
	case "if_running":
		return api.resolveIfRunning(args)

	// Process stats variables
	case "running_threads":
		return strconv.Itoa(api.sysProvider.Process().TotalThreads)
	case "top_io":
		return api.resolveTop(args, false) // Alias to top for now

	// Entropy/random variables
	case "entropy_avail":
		return api.resolveEntropy()
	case "entropy_poolsize":
		return "4096" // Standard Linux entropy pool size
	case "entropy_perc":
		return api.resolveEntropyPerc()
	case "entropy_bar":
		return api.resolveEntropyBar(args)

	// Date/time aliases
	case "tztime":
		return api.resolveTime(args)
	case "utime":
		return strconv.FormatInt(time.Now().Unix(), 10)

	// Misc formatting/info variables
	case "stippled_hr":
		return api.resolveStippledHR(args)
	case "scroll":
		return api.resolveScroll(args)
	case "lua":
		return api.resolveLua(args, false)
	case "lua_parse":
		return api.resolveLua(args, true)
	case "template0":
		return api.resolveTemplate(0, args)
	case "template1":
		return api.resolveTemplate(1, args)
	case "template2":
		return api.resolveTemplate(2, args)
	case "template3":
		return api.resolveTemplate(3, args)
	case "template4":
		return api.resolveTemplate(4, args)
	case "template5":
		return api.resolveTemplate(5, args)
	case "template6":
		return api.resolveTemplate(6, args)
	case "template7":
		return api.resolveTemplate(7, args)
	case "template8":
		return api.resolveTemplate(8, args)
	case "template9":
		return api.resolveTemplate(9, args)

	// Pre/post text markers
	case "pre_exec":
		return api.resolveExec(args)
	case "texeci":
		return api.resolveExec(args) // Threaded exec - treat as regular exec

	// Inode variables
	case "fs_inodes":
		return api.resolveFSInodes(args)
	case "fs_inodes_free":
		return api.resolveFSInodesFree(args)
	case "fs_inodes_perc":
		return api.resolveFSInodesPerc(args)

	// Additional memory variables
	case "memgauge", "membar":
		return api.resolveMemBar(args)
	case "swapbar":
		return api.resolveSwapBar(args)
	case "shmem":
		return formatBytes(api.sysProvider.Memory().Cached) // Shared memory approx

	// Additional CPU variables
	case "cpubar":
		return api.resolveCPUBar(args)
	case "loadgraph":
		return api.resolveLoadGraph(args)
	case "freq_dyn":
		return api.resolveCPUFreq(args) // Dynamic frequency
	case "freq_dyn_g":
		return api.resolveCPUFreqGHz(args)

	// Platform variables
	case "platform":
		return api.sysProvider.SysInfo().Sysname

	// Acpi/thermal variables
	case "acpitemp":
		return api.resolveHwmon(args)
	case "acpifan":
		return api.resolveACPIFan()
	case "acpiacadapter":
		if api.sysProvider.Battery().ACOnline {
			return "on-line"
		}
		return "off-line"

	// Nvidia GPU monitoring
	case "nvidia":
		return api.resolveNvidia(args)
	case "nvidiagraph":
		return api.resolveNvidiaGraph(args)
	case "nvidia_temp":
		return api.resolveNvidia([]string{"temp"})
	case "nvidia_gpu":
		return api.resolveNvidia([]string{"gpuutil"})
	case "nvidia_fan":
		return api.resolveNvidia([]string{"fan"})
	case "nvidia_mem":
		return api.resolveNvidia([]string{"memutil"})
	case "nvidia_memused":
		return api.resolveNvidia([]string{"memused"})
	case "nvidia_memtotal":
		return api.resolveNvidia([]string{"memtotal"})
	case "nvidia_driver":
		return api.resolveNvidia([]string{"driver"})
	case "nvidia_power":
		return api.resolveNvidia([]string{"power"})
	case "nvidia_name":
		return api.resolveNvidia([]string{"name"})

	// Apcupsd (UPS) stubs - not implemented; requires APCUPSD daemon and NIS protocol.
	// Users should use ${execi} with apcaccess command. See docs/migration.md.
	case "apcupsd", "apcupsd_model", "apcupsd_status", "apcupsd_linev",
		"apcupsd_load", "apcupsd_charge", "apcupsd_timeleft", "apcupsd_temp",
		"apcupsd_battv", "apcupsd_cable", "apcupsd_driver", "apcupsd_upsmode",
		"apcupsd_name", "apcupsd_hostname":
		return "N/A"

	// IMAP/POP3/mail variables
	case "imap_unseen":
		return api.resolveImapUnseen(args)
	case "imap_messages":
		return api.resolveImapMessages(args)
	case "pop3_unseen":
		return api.resolvePop3Unseen(args)
	case "pop3_used":
		return api.resolvePop3Used(args)
	case "new_mails", "mails":
		return api.resolveTotalMails(args)

	// Weather variables
	case "weather":
		return api.resolveWeather(args)

	// Image variable
	case "image":
		return api.resolveImage(args)

	// Stock ticker stub - not implemented; requires external API keys.
	// Users should use ${execi} with custom scripts. See docs/migration.md.
	case "stockquote":
		return "N/A"

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

	// ISO 8601 week number and year (%V, %G, %g)
	isoYear, isoWeek := t.ISOWeek()
	result = strings.ReplaceAll(result, "%V", fmt.Sprintf("%02d", isoWeek))
	result = strings.ReplaceAll(result, "%G", fmt.Sprintf("%04d", isoYear))
	result = strings.ReplaceAll(result, "%g", fmt.Sprintf("%02d", isoYear%100))

	// Week number with Sunday as first day of week (%U)
	// Week 01 starts on the first Sunday of the year
	result = strings.ReplaceAll(result, "%U", fmt.Sprintf("%02d", sundayWeekNumber(t)))

	// Week number with Monday as first day of week (%W)
	// Week 01 starts on the first Monday of the year
	result = strings.ReplaceAll(result, "%W", fmt.Sprintf("%02d", mondayWeekNumber(t)))

	// Seconds since Unix epoch (%s)
	result = strings.ReplaceAll(result, "%s", fmt.Sprintf("%d", t.Unix()))

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

// sundayWeekNumber calculates the week number with Sunday as the first day of the week.
// Week 01 starts on the first Sunday of the year. Days before the first Sunday are week 00.
func sundayWeekNumber(t time.Time) int {
	yearStart := time.Date(t.Year(), time.January, 1, 0, 0, 0, 0, t.Location())
	// Days until first Sunday (0 = Sunday, so we need 0 days if Jan 1 is Sunday)
	daysUntilFirstSunday := int((7 - yearStart.Weekday()) % 7)
	firstSunday := yearStart.AddDate(0, 0, daysUntilFirstSunday)

	if t.Before(firstSunday) {
		return 0
	}
	// Days since first Sunday
	daysSinceFirstSunday := int(t.Sub(firstSunday).Hours() / 24)
	return (daysSinceFirstSunday / 7) + 1
}

// mondayWeekNumber calculates the week number with Monday as the first day of the week.
// Week 01 starts on the first Monday of the year. Days before the first Monday are week 00.
func mondayWeekNumber(t time.Time) int {
	yearStart := time.Date(t.Year(), time.January, 1, 0, 0, 0, 0, t.Location())
	// Days until first Monday (1 = Monday)
	daysUntilFirstMonday := (8 - int(yearStart.Weekday())) % 7
	if yearStart.Weekday() == time.Monday {
		daysUntilFirstMonday = 0
	}
	firstMonday := yearStart.AddDate(0, 0, daysUntilFirstMonday)

	if t.Before(firstMonday) {
		return 0
	}
	// Days since first Monday
	daysSinceFirstMonday := int(t.Sub(firstMonday).Hours() / 24)
	return (daysSinceFirstMonday / 7) + 1
}

// resolveAddr resolves the ${addr interface} variable.
// Returns the first IPv4 address for the specified interface.
func (api *ConkyAPI) resolveAddr(args []string) string {
	if len(args) == 0 {
		return ""
	}

	netStats := api.sysProvider.Network()
	iface, ok := netStats.Interfaces[args[0]]
	if !ok {
		return ""
	}

	if len(iface.IPv4Addrs) > 0 {
		return iface.IPv4Addrs[0]
	}
	return ""
}

// resolveAddrs resolves the ${addrs interface} variable.
// Returns all IP addresses (IPv4 and IPv6) for the specified interface, space-separated.
func (api *ConkyAPI) resolveAddrs(args []string) string {
	if len(args) == 0 {
		return ""
	}

	netStats := api.sysProvider.Network()
	iface, ok := netStats.Interfaces[args[0]]
	if !ok {
		return ""
	}

	var addrs []string
	addrs = append(addrs, iface.IPv4Addrs...)
	addrs = append(addrs, iface.IPv6Addrs...)

	return strings.Join(addrs, " ")
}

// resolveNameserver resolves the ${nameserver index} variable.
// Returns the DNS nameserver at the specified index (0-based).
// With no arguments, returns the first nameserver.
func (api *ConkyAPI) resolveNameserver(args []string) string {
	netStats := api.sysProvider.Network()

	if len(netStats.Nameservers) == 0 {
		return ""
	}

	index := 0
	if len(args) > 0 {
		var err error
		index, err = strconv.Atoi(args[0])
		if err != nil || index < 0 {
			return ""
		}
	}

	if index >= len(netStats.Nameservers) {
		return ""
	}

	return netStats.Nameservers[index]
}

// resolveTop resolves ${top} and ${top_mem} variables.
// Format: ${top field index} where field is name/pid/cpu/mem and index is 1-based.
func (api *ConkyAPI) resolveTop(args []string, byMem bool) string {
	if len(args) < 2 {
		return ""
	}

	field := strings.ToLower(args[0])
	index, err := strconv.Atoi(args[1])
	if err != nil || index < 1 {
		return ""
	}
	index-- // Convert to 0-based

	procStats := api.sysProvider.Process()
	var processes []monitor.ProcessInfo
	if byMem {
		processes = procStats.TopMem
	} else {
		processes = procStats.TopCPU
	}

	if index >= len(processes) {
		return ""
	}

	proc := processes[index]
	switch field {
	case "name":
		return proc.Name
	case "pid":
		return strconv.Itoa(proc.PID)
	case "cpu":
		return fmt.Sprintf("%.1f", proc.CPUPercent)
	case "mem":
		return fmt.Sprintf("%.1f", proc.MemPercent)
	case "mem_res":
		return formatBytes(proc.MemBytes)
	case "mem_vsize":
		return formatBytes(proc.VirtBytes)
	case "threads", "time":
		return strconv.Itoa(proc.Threads)
	default:
		return ""
	}
}

// resolveExec executes a shell command and returns its output.
// Usage: ${exec command}
func (api *ConkyAPI) resolveExec(args []string) string {
	if len(args) == 0 {
		return ""
	}

	cmdStr := strings.Join(args, " ")
	cmd := exec.Command("sh", "-c", cmdStr)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	// Trim trailing newlines
	return strings.TrimRight(string(output), "\n\r")
}

// resolveExeci executes a shell command with interval-based caching.
// Usage: ${execi interval command}
// The command output is cached for 'interval' seconds before re-execution.
func (api *ConkyAPI) resolveExeci(args []string) string {
	if len(args) < 2 {
		return ""
	}

	// Parse interval from first argument
	interval, err := strconv.Atoi(args[0])
	if err != nil || interval < 0 {
		return ""
	}

	// Build command string from remaining arguments
	cmdStr := strings.Join(args[1:], " ")

	now := time.Now()

	// Check cache with read lock
	api.mu.RLock()
	entry, exists := api.execCache[cmdStr]
	api.mu.RUnlock()

	if exists && now.Before(entry.expiresAt) {
		// Update lastAccessed time
		api.mu.Lock()
		entry.lastAccessed = now
		api.mu.Unlock()
		return entry.output
	}

	// Cache miss or expired - execute command
	cmd := exec.Command("sh", "-c", cmdStr)
	output, err := cmd.Output()
	if err != nil {
		// On error, return cached value if available, otherwise empty
		if exists {
			api.mu.Lock()
			entry.lastAccessed = now
			api.mu.Unlock()
			return entry.output
		}
		return ""
	}

	result := strings.TrimRight(string(output), "\n\r")

	// Update cache with write lock
	api.mu.Lock()
	api.execCache[cmdStr] = &execCacheEntry{
		output:       result,
		expiresAt:    now.Add(time.Duration(interval) * time.Second),
		lastAccessed: now,
	}
	api.mu.Unlock()

	return result
}

// resolveHR returns a horizontal rule of specified length.
// Usage: ${hr height} - returns dashes (actual rendering in display layer).
func (api *ConkyAPI) resolveHR(args []string) string {
	width := 10
	if len(args) > 0 {
		if w, err := strconv.Atoi(args[0]); err == nil && w > 0 {
			width = w
		}
	}
	return strings.Repeat("-", width)
}

// resolveFSBar returns a graphical bar widget for filesystem usage.
// Usage: ${fs_bar height,width mountpoint}
func (api *ConkyAPI) resolveFSBar(args []string) string {
	mountPoint := "/"
	width := float64(100) // Default width in pixels
	height := float64(8)  // Default height in pixels

	if len(args) > 0 {
		// Check for size,mountpoint format (height,width or just height)
		parts := strings.Split(args[0], ",")
		if len(parts) >= 1 {
			if h, err := strconv.ParseFloat(parts[0], 64); err == nil {
				height = h
			}
		}
		if len(parts) >= 2 {
			if w, err := strconv.ParseFloat(parts[1], 64); err == nil {
				width = w
			}
		}
		// Last arg is mount point if more than one arg
		if len(args) > 1 {
			mountPoint = args[len(args)-1]
		} else if len(parts) == 1 {
			// Single arg could be height or mount point
			if _, err := strconv.ParseFloat(args[0], 64); err != nil {
				mountPoint = args[0]
			}
		}
	}

	fsStats := api.sysProvider.Filesystem()
	mount, ok := fsStats.Mounts[mountPoint]
	if !ok {
		return render.EncodeBarMarker(0, width, height)
	}

	return render.EncodeBarMarker(mount.UsagePercent, width, height)
}

// resolveFSType returns the filesystem type for a mount point.
func (api *ConkyAPI) resolveFSType(args []string) string {
	mountPoint := "/"
	if len(args) > 0 {
		mountPoint = args[0]
	}

	fsStats := api.sysProvider.Filesystem()
	if mount, ok := fsStats.Mounts[mountPoint]; ok {
		return mount.FSType
	}
	return ""
}

// resolveBattery returns battery status string.
func (api *ConkyAPI) resolveBattery(args []string) string {
	batStats := api.sysProvider.Battery()

	batName := "BAT0"
	if len(args) > 0 {
		batName = args[0]
	}

	// Try to find specific battery first
	if bat, ok := batStats.Batteries[batName]; ok {
		return fmt.Sprintf("%s %d%%", bat.Status, bat.Capacity)
	}

	// If a specific battery was requested but not found, or if no batteries exist,
	// try aggregate status if available (based on ACOnline, IsCharging, IsDischarging, TotalCapacity)
	hasAggregateInfo := batStats.ACOnline || batStats.IsCharging || batStats.IsDischarging || batStats.TotalCapacity > 0
	if hasAggregateInfo {
		status := "Unknown"
		if batStats.ACOnline {
			if batStats.IsCharging {
				status = "Charging"
			} else {
				status = "Full"
			}
		} else if batStats.IsDischarging {
			status = "Discharging"
		}
		return fmt.Sprintf("%s %.0f%%", status, batStats.TotalCapacity)
	}

	return "No battery"
}

// resolveBatteryBar returns a graphical bar widget for battery level.
func (api *ConkyAPI) resolveBatteryBar(args []string) string {
	width := float64(100) // Default width in pixels
	height := float64(8)  // Default height in pixels

	if len(args) > 0 {
		if h, err := strconv.ParseFloat(args[0], 64); err == nil {
			height = h
		}
	}
	if len(args) > 1 {
		if w, err := strconv.ParseFloat(args[1], 64); err == nil {
			width = w
		}
	}

	batStats := api.sysProvider.Battery()
	percent := batStats.TotalCapacity

	return render.EncodeBarMarker(percent, width, height)
}

// resolveBatteryTime returns estimated battery time remaining.
// Returns time in "H:MM" format for discharging/charging, "AC" when fully charged on AC,
// or "Unknown" when time cannot be calculated.
func (api *ConkyAPI) resolveBatteryTime(args []string) string {
	batStats := api.sysProvider.Battery()

	// Determine which battery to query (default: first found or specified)
	batteryName := ""
	if len(args) > 0 {
		batteryName = args[0]
	}

	// Find the target battery
	var targetBattery *monitor.BatteryInfo
	if batteryName != "" {
		if bat, ok := batStats.Batteries[batteryName]; ok {
			targetBattery = &bat
		}
	} else {
		// Use first battery found
		for _, bat := range batStats.Batteries {
			b := bat // Create a copy to take address of
			targetBattery = &b
			break
		}
	}

	// No battery found
	if targetBattery == nil {
		if batStats.ACOnline {
			return "AC"
		}
		return "Unknown"
	}

	// Calculate and format time based on charging status
	var timeSeconds float64
	switch targetBattery.Status {
	case "Discharging":
		timeSeconds = targetBattery.TimeToEmpty
	case "Charging":
		timeSeconds = targetBattery.TimeToFull
	case "Full", "Not charging":
		return "AC"
	default:
		if batStats.ACOnline {
			return "AC"
		}
		return "Unknown"
	}

	// If no time estimate available
	if timeSeconds <= 0 {
		return "Unknown"
	}

	// Format as "H:MM"
	hours := int(timeSeconds / 3600)
	minutes := int((timeSeconds - float64(hours)*3600) / 60)
	return fmt.Sprintf("%d:%02d", hours, minutes)
}

// resolveNetworkSpeedF returns network speed as a float value.
func (api *ConkyAPI) resolveNetworkSpeedF(args []string, isDownload bool) string {
	netStats := api.sysProvider.Network()

	var speed float64
	if len(args) == 0 {
		if isDownload {
			speed = netStats.TotalRxBytesPerSec
		} else {
			speed = netStats.TotalTxBytesPerSec
		}
	} else {
		iface, ok := netStats.Interfaces[args[0]]
		if !ok {
			return "0.00"
		}
		if isDownload {
			speed = iface.RxBytesPerSec
		} else {
			speed = iface.TxBytesPerSec
		}
	}

	// Convert to KiB/s
	return fmt.Sprintf("%.2f", speed/1024)
}

// resolveIfUp checks if a network interface is up.
func (api *ConkyAPI) resolveIfUp(args []string) string {
	if len(args) == 0 {
		return ""
	}

	netStats := api.sysProvider.Network()
	if _, ok := netStats.Interfaces[args[0]]; ok {
		return "1" // Interface exists/is up
	}
	return "0"
}

// --- Additional Resolver Functions ---

// resolveWirelessESSID returns the ESSID of a wireless interface.
// Reads wireless info from /proc/net/wireless and /sys/class/net/<iface>/wireless/.
func (api *ConkyAPI) resolveWirelessESSID(args []string) string {
	if len(args) == 0 {
		return ""
	}
	netStats := api.sysProvider.Network()
	if ifStats, ok := netStats.Interfaces[args[0]]; ok {
		if ifStats.Wireless != nil {
			return ifStats.Wireless.ESSID
		}
	}
	return ""
}

// resolveWirelessLinkQual returns wireless link quality (raw value).
func (api *ConkyAPI) resolveWirelessLinkQual(args []string) string {
	if len(args) == 0 {
		return "0"
	}
	netStats := api.sysProvider.Network()
	if ifStats, ok := netStats.Interfaces[args[0]]; ok {
		if ifStats.Wireless != nil {
			return strconv.Itoa(ifStats.Wireless.LinkQuality)
		}
	}
	return "0"
}

// resolveWirelessLinkQualPerc returns wireless link quality as percentage.
func (api *ConkyAPI) resolveWirelessLinkQualPerc(args []string) string {
	if len(args) == 0 {
		return "0"
	}
	netStats := api.sysProvider.Network()
	if ifStats, ok := netStats.Interfaces[args[0]]; ok {
		if ifStats.Wireless != nil {
			return strconv.Itoa(ifStats.Wireless.LinkQualityPercent())
		}
	}
	return "0"
}

// resolveWirelessBitrate returns wireless bitrate as formatted string.
func (api *ConkyAPI) resolveWirelessBitrate(args []string) string {
	if len(args) == 0 {
		return "0Mb/s"
	}
	netStats := api.sysProvider.Network()
	if ifStats, ok := netStats.Interfaces[args[0]]; ok {
		if ifStats.Wireless != nil {
			return ifStats.Wireless.BitRateString()
		}
	}
	return "0Mb/s"
}

// resolveWirelessAP returns wireless access point MAC address.
func (api *ConkyAPI) resolveWirelessAP(args []string) string {
	if len(args) == 0 {
		return "00:00:00:00:00:00"
	}
	netStats := api.sysProvider.Network()
	if ifStats, ok := netStats.Interfaces[args[0]]; ok {
		if ifStats.Wireless != nil && ifStats.Wireless.AccessPoint != "" {
			return ifStats.Wireless.AccessPoint
		}
	}
	return "00:00:00:00:00:00"
}

// resolveWirelessMode returns the operating mode of a wireless interface.
func (api *ConkyAPI) resolveWirelessMode(args []string) string {
	if len(args) == 0 {
		return "Managed"
	}
	netStats := api.sysProvider.Network()
	if ifStats, ok := netStats.Interfaces[args[0]]; ok {
		if ifStats.Wireless != nil && ifStats.Wireless.Mode != "" {
			return ifStats.Wireless.Mode
		}
	}
	return "Managed"
}

// resolveTCPPortMon monitors TCP connections in a port range.
// Syntax: ${tcp_portmon port_begin port_end item [index]}
// Items: count, lip, lport, lservice, rip, rport, rservice, rhost
func (api *ConkyAPI) resolveTCPPortMon(args []string) string {
	// Minimum args: port_begin, port_end, item
	if len(args) < 3 {
		return "0"
	}

	minPort, err := strconv.Atoi(args[0])
	if err != nil {
		return "0"
	}

	maxPort, err := strconv.Atoi(args[1])
	if err != nil {
		return "0"
	}

	item := strings.ToLower(args[2])

	// count doesn't need an index
	if item == "count" {
		count := api.sysProvider.TCPCountInRange(minPort, maxPort)
		return strconv.Itoa(count)
	}

	// All other items require an index
	if len(args) < 4 {
		return "0"
	}

	index, err := strconv.Atoi(args[3])
	if err != nil {
		return "0"
	}

	conn := api.sysProvider.TCPConnectionByIndex(minPort, maxPort, index)
	if conn == nil {
		return ""
	}

	switch item {
	case "lip":
		return conn.LocalIP
	case "lport":
		return strconv.Itoa(conn.LocalPort)
	case "lservice":
		return portToService(conn.LocalPort)
	case "rip":
		return conn.RemoteIP
	case "rport":
		return strconv.Itoa(conn.RemotePort)
	case "rservice":
		return portToService(conn.RemotePort)
	case "rhost":
		// Return remote IP as hostname (DNS lookup could be added)
		return conn.RemoteIP
	default:
		return ""
	}
}

// portToService maps well-known port numbers to service names.
func portToService(port int) string {
	services := map[int]string{
		20:    "ftp-data",
		21:    "ftp",
		22:    "ssh",
		23:    "telnet",
		25:    "smtp",
		53:    "domain",
		67:    "bootps",
		68:    "bootpc",
		69:    "tftp",
		80:    "http",
		110:   "pop3",
		119:   "nntp",
		123:   "ntp",
		143:   "imap",
		161:   "snmp",
		162:   "snmptrap",
		194:   "irc",
		389:   "ldap",
		443:   "https",
		445:   "microsoft-ds",
		465:   "smtps",
		514:   "syslog",
		587:   "submission",
		636:   "ldaps",
		993:   "imaps",
		995:   "pop3s",
		1080:  "socks",
		1433:  "ms-sql-s",
		1521:  "oracle",
		3306:  "mysql",
		3389:  "ms-wbt-server",
		5432:  "postgresql",
		5900:  "vnc",
		6379:  "redis",
		8080:  "http-proxy",
		8443:  "https-alt",
		9200:  "elasticsearch",
		27017: "mongodb",
	}
	if name, ok := services[port]; ok {
		return name
	}
	return strconv.Itoa(port)
}

// resolveIfExisting checks if a file or path exists.
func (api *ConkyAPI) resolveIfExisting(args []string) string {
	if len(args) == 0 {
		return "0"
	}
	if _, err := os.Stat(args[0]); err == nil {
		return "1"
	}
	return "0"
}

// resolveIfRunning checks if a process is running.
func (api *ConkyAPI) resolveIfRunning(args []string) string {
	if len(args) == 0 {
		return "0"
	}
	// Check if process name exists in top processes
	procStats := api.sysProvider.Process()
	for _, p := range procStats.TopCPU {
		if strings.Contains(p.Name, args[0]) {
			return "1"
		}
	}
	return "0"
}

// resolveEntropy returns available entropy.
func (api *ConkyAPI) resolveEntropy() string {
	data, err := os.ReadFile("/proc/sys/kernel/random/entropy_avail")
	if err != nil {
		return "0"
	}
	return strings.TrimSpace(string(data))
}

// resolveEntropyPerc returns entropy as percentage.
func (api *ConkyAPI) resolveEntropyPerc() string {
	data, err := os.ReadFile("/proc/sys/kernel/random/entropy_avail")
	if err != nil {
		return "0"
	}
	avail, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return "0"
	}
	perc := float64(avail) / 4096.0 * 100.0
	if perc > 100 {
		perc = 100
	}
	return fmt.Sprintf("%.0f", perc)
}

// resolveEntropyBar returns a graphical bar widget for entropy.
func (api *ConkyAPI) resolveEntropyBar(args []string) string {
	width := float64(100) // Default width in pixels
	height := float64(8)  // Default height in pixels

	if len(args) > 0 {
		if h, err := strconv.ParseFloat(args[0], 64); err == nil {
			height = h
		}
	}
	if len(args) > 1 {
		if w, err := strconv.ParseFloat(args[1], 64); err == nil {
			width = w
		}
	}

	data, err := os.ReadFile("/proc/sys/kernel/random/entropy_avail")
	if err != nil {
		return render.EncodeBarMarker(0, width, height)
	}
	avail, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return render.EncodeBarMarker(0, width, height)
	}
	perc := float64(avail) / 4096.0 * 100
	if perc > 100 {
		perc = 100
	}
	return render.EncodeBarMarker(perc, width, height)
}

// resolveStippledHR returns a stippled horizontal rule.
func (api *ConkyAPI) resolveStippledHR(args []string) string {
	width := 10
	if len(args) > 0 {
		if w, err := strconv.Atoi(args[0]); err == nil {
			width = w
		}
	}
	result := ""
	for i := 0; i < width; i++ {
		if i%2 == 0 {
			result += "-"
		} else {
			result += " "
		}
	}
	return result
}

// resolveScroll returns scrolling text with animation.
// Usage: ${scroll length step text}
// - length: visible width in characters
// - step: number of characters to scroll per update (default 1)
// - text: the text to scroll (can contain spaces)
// The text scrolls left by 'step' characters each update cycle, wrapping around.
func (api *ConkyAPI) resolveScroll(args []string) string {
	if len(args) < 2 {
		return ""
	}

	// Parse length (visible width)
	length, err := strconv.Atoi(args[0])
	if err != nil || length <= 0 {
		length = 20 // default width
	}

	// Parse step (characters to scroll per update)
	step := 1
	if len(args) >= 2 {
		if s, err := strconv.Atoi(args[1]); err == nil && s > 0 {
			step = s
		}
	}

	// Get the text to scroll (join remaining args)
	text := ""
	if len(args) >= 3 {
		text = strings.Join(args[2:], " ")
	}

	// Empty text returns empty
	if text == "" {
		return ""
	}

	// If text is shorter than or equal to length, no scrolling needed
	textLen := len([]rune(text))
	if textLen <= length {
		// Pad to length for consistent width
		return padRight(text, length)
	}

	// Create a unique key for this scroll instance (based on the original args)
	scrollKey := strings.Join(args, "|")

	now := time.Now()

	api.mu.Lock()
	defer api.mu.Unlock()

	// Get or create scroll state
	state, exists := api.scrollStates[scrollKey]
	if !exists {
		state = &scrollState{
			position:     0,
			lastUpdate:   now,
			lastAccessed: now,
		}
		api.scrollStates[scrollKey] = state
	}

	// Update lastAccessed time
	state.lastAccessed = now

	// Advance scroll position (on each call, as update cycle advances)
	// Use circular scrolling: when we reach the end, wrap around
	if now.Sub(state.lastUpdate) > 0 {
		state.position = (state.position + step) % textLen
		state.lastUpdate = now
	}

	// Create the scrolled view by extracting a window from the text
	// For smooth wrapping, duplicate the text
	runes := []rune(text)
	doubledRunes := make([]rune, len(runes)*2)
	copy(doubledRunes, runes)
	copy(doubledRunes[len(runes):], runes)

	start := state.position % textLen
	end := start + length
	if end > len(doubledRunes) {
		end = len(doubledRunes)
	}

	return string(doubledRunes[start:end])
}

// resolveTemplate resolves a template variable by index with argument substitution.
// Templates are defined as template0-template9 in the config and can contain
// \1, \2, etc. placeholders that are replaced with the provided arguments.
func (api *ConkyAPI) resolveTemplate(index int, args []string) string {
	template := api.GetTemplate(index)
	if template == "" {
		return ""
	}

	// Substitute \1, \2, etc. with the provided arguments
	result := template
	for i, arg := range args {
		placeholder := fmt.Sprintf("\\%d", i+1)
		result = strings.ReplaceAll(result, placeholder, arg)
	}

	// Parse the result to resolve any embedded Conky variables
	return api.Parse(result)
}

// padRight pads a string with spaces to reach the desired length.
func padRight(s string, length int) string {
	runes := []rune(s)
	if len(runes) >= length {
		return s
	}
	return s + strings.Repeat(" ", length-len(runes))
}

// resolveFSInodes returns total inodes for a mount point.
func (api *ConkyAPI) resolveFSInodes(args []string) string {
	mountPoint := "/"
	if len(args) > 0 {
		mountPoint = args[0]
	}
	fsStats := api.sysProvider.Filesystem()
	if mount, ok := fsStats.Mounts[mountPoint]; ok {
		return formatNumber(mount.InodesTotal)
	}
	return "0"
}

// resolveFSInodesFree returns free inodes for a mount point.
func (api *ConkyAPI) resolveFSInodesFree(args []string) string {
	mountPoint := "/"
	if len(args) > 0 {
		mountPoint = args[0]
	}
	fsStats := api.sysProvider.Filesystem()
	if mount, ok := fsStats.Mounts[mountPoint]; ok {
		return formatNumber(mount.InodesFree)
	}
	return "0"
}

// resolveFSInodesPerc returns inode usage percentage.
func (api *ConkyAPI) resolveFSInodesPerc(args []string) string {
	mountPoint := "/"
	if len(args) > 0 {
		mountPoint = args[0]
	}
	fsStats := api.sysProvider.Filesystem()
	if mount, ok := fsStats.Mounts[mountPoint]; ok {
		return fmt.Sprintf("%.0f", mount.InodesPercent)
	}
	return "0"
}

// resolveMemBar returns a graphical bar widget for memory usage.
func (api *ConkyAPI) resolveMemBar(args []string) string {
	width := float64(100) // Default width in pixels
	height := float64(8)  // Default height in pixels

	if len(args) > 0 {
		if h, err := strconv.ParseFloat(args[0], 64); err == nil {
			height = h
		}
	}
	if len(args) > 1 {
		if w, err := strconv.ParseFloat(args[1], 64); err == nil {
			width = w
		}
	}

	mem := api.sysProvider.Memory()
	return render.EncodeBarMarker(mem.UsagePercent, width, height)
}

// resolveSwapBar returns a graphical bar widget for swap usage.
func (api *ConkyAPI) resolveSwapBar(args []string) string {
	width := float64(100) // Default width in pixels
	height := float64(8)  // Default height in pixels

	if len(args) > 0 {
		if h, err := strconv.ParseFloat(args[0], 64); err == nil {
			height = h
		}
	}
	if len(args) > 1 {
		if w, err := strconv.ParseFloat(args[1], 64); err == nil {
			width = w
		}
	}

	mem := api.sysProvider.Memory()
	return render.EncodeBarMarker(mem.SwapPercent, width, height)
}

// resolveCPUBar returns a graphical bar widget for CPU usage.
func (api *ConkyAPI) resolveCPUBar(args []string) string {
	width := float64(100) // Default width in pixels
	height := float64(8)  // Default height in pixels
	cpuIdx := -1          // -1 means overall

	// Parse arguments: ${cpubar cpu# height,width} or ${cpubar height}
	if len(args) > 0 {
		// First arg could be CPU number or height
		if strings.HasPrefix(strings.ToLower(args[0]), "cpu") {
			// It's a CPU specifier like "cpu0"
			if idx, err := strconv.Atoi(strings.TrimPrefix(strings.ToLower(args[0]), "cpu")); err == nil {
				cpuIdx = idx
			}
		} else if idx, err := strconv.Atoi(args[0]); err == nil {
			// Check if it's a small number (likely a CPU core) or larger (likely height)
			if idx <= 16 {
				cpuIdx = idx - 1 // Convert to 0-based
			} else {
				height = float64(idx)
			}
		} else if h, err := strconv.ParseFloat(args[0], 64); err == nil {
			height = h
		}
	}
	if len(args) > 1 {
		if h, err := strconv.ParseFloat(args[1], 64); err == nil {
			height = h
		}
	}
	if len(args) > 2 {
		if w, err := strconv.ParseFloat(args[2], 64); err == nil {
			width = w
		}
	}

	cpuStats := api.sysProvider.CPU()
	var percent float64
	if cpuIdx >= 0 && cpuIdx < len(cpuStats.Cores) {
		percent = cpuStats.Cores[cpuIdx]
	} else {
		percent = cpuStats.UsagePercent
	}

	return render.EncodeBarMarker(percent, width, height)
}

// resolveLoadGraph returns a graphical representation of load.
func (api *ConkyAPI) resolveLoadGraph(args []string) string {
	width := float64(100) // Default width in pixels
	height := float64(20) // Default height in pixels

	if len(args) > 0 {
		if h, err := strconv.ParseFloat(args[0], 64); err == nil {
			height = h
		}
	}
	if len(args) > 1 {
		if w, err := strconv.ParseFloat(args[1], 64); err == nil {
			width = w
		}
	}

	sysInfo := api.sysProvider.SysInfo()
	// Normalize load to percentage based on CPU count
	cpuCount := api.sysProvider.CPU().CPUCount
	if cpuCount == 0 {
		cpuCount = 1
	}
	loadPerc := sysInfo.LoadAvg1 / float64(cpuCount) * 100
	if loadPerc > 100 {
		loadPerc = 100
	}

	return render.EncodeGraphMarker(loadPerc, width, height)
}

// resolveACPIFan returns ACPI fan status.
func (api *ConkyAPI) resolveACPIFan() string {
	hwmon := api.sysProvider.Hwmon()
	for _, dev := range hwmon.Devices {
		if strings.Contains(strings.ToLower(dev.Name), "fan") {
			return "running"
		}
	}
	return "unknown"
}

// resolveNvidia resolves ${nvidia} variables.
// Supports various GPU monitoring fields from nvidia-smi.
// Usage: ${nvidia [field]}
// Fields: temp, gpuutil, memutil, mem, fan, power, name, driver, memused, memtotal, memfree, memperc
// Without a field, returns GPU utilization percentage.
func (api *ConkyAPI) resolveNvidia(args []string) string {
	gpuStats := api.sysProvider.GPU()

	if !gpuStats.Available {
		return "N/A"
	}

	// Default to GPU utilization if no field specified
	if len(args) == 0 {
		return gpuStats.GetField("gpuutil")
	}

	return gpuStats.GetField(args[0])
}

// resolveNvidiaGraph is a placeholder for ${nvidiagraph} which would render a graph.
// Graph rendering is not yet implemented; returns the GPU utilization value.
func (api *ConkyAPI) resolveNvidiaGraph(args []string) string {
	gpuStats := api.sysProvider.GPU()

	if !gpuStats.Available {
		return "N/A"
	}

	// For now, return GPU utilization as a number (without %)
	// Graphs need to be rendered by the renderer, not as text
	return strconv.Itoa(gpuStats.UtilGPU)
}

// formatNumber formats a number with commas for readability.
func formatNumber(n uint64) string {
	s := strconv.FormatUint(n, 10)
	if len(s) <= 3 {
		return s
	}
	// Insert commas
	result := ""
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result += ","
		}
		result += string(c)
	}
	return result
}

// resolveImapUnseen resolves the ${imap_unseen} variable.
// Accepts an optional account name argument.
// Without argument, returns the total unseen count across all accounts.
func (api *ConkyAPI) resolveImapUnseen(args []string) string {
	if len(args) > 0 {
		// Get unseen count for specific IMAP account
		return strconv.Itoa(api.sysProvider.MailUnseenCount(args[0]))
	}
	// Return total unseen across all accounts
	return strconv.Itoa(api.sysProvider.MailTotalUnseen())
}

// resolveImapMessages resolves the ${imap_messages} variable.
// Accepts an optional account name argument.
// Without argument, returns the total message count across all accounts.
func (api *ConkyAPI) resolveImapMessages(args []string) string {
	if len(args) > 0 {
		return strconv.Itoa(api.sysProvider.MailTotalCount(args[0]))
	}
	return strconv.Itoa(api.sysProvider.MailTotalMessages())
}

// resolvePop3Unseen resolves the ${pop3_unseen} variable.
// Accepts an optional account name argument.
// For POP3, unseen and total are the same since POP3 doesn't track read status.
func (api *ConkyAPI) resolvePop3Unseen(args []string) string {
	if len(args) > 0 {
		return strconv.Itoa(api.sysProvider.MailUnseenCount(args[0]))
	}
	return strconv.Itoa(api.sysProvider.MailTotalUnseen())
}

// resolvePop3Used resolves the ${pop3_used} variable.
// Accepts an optional account name argument.
// Returns the total message count for the POP3 account.
func (api *ConkyAPI) resolvePop3Used(args []string) string {
	if len(args) > 0 {
		return strconv.Itoa(api.sysProvider.MailTotalCount(args[0]))
	}
	return strconv.Itoa(api.sysProvider.MailTotalMessages())
}

// resolveTotalMails resolves the ${new_mails} and ${mails} variables.
// ${new_mails} returns the total unseen count.
// ${mails} returns the total message count.
func (api *ConkyAPI) resolveTotalMails(args []string) string {
	if len(args) > 0 {
		// First arg could be account name
		return strconv.Itoa(api.sysProvider.MailUnseenCount(args[0]))
	}
	return strconv.Itoa(api.sysProvider.MailTotalUnseen())
}

// resolveWeather resolves the ${weather} variable.
// Syntax: ${weather station_id field}
// Example: ${weather KJFK temp} returns temperature at JFK airport
// Supported fields: temp, temp_f, dewpoint, humidity, pressure, pressure_inhg,
// wind, wind_dir, wind_dir_compass, wind_gust, visibility, condition, cloud, raw
func (api *ConkyAPI) resolveWeather(args []string) string {
	if len(args) < 1 {
		return ""
	}

	stationID := args[0]
	field := "condition" // default field
	if len(args) >= 2 {
		field = args[1]
	}

	weather := api.sysProvider.Weather(stationID)
	if weather.Error != "" && weather.RawMETAR == "" {
		return "N/A"
	}

	result := weather.GetField(field)
	if result == "" {
		return "N/A"
	}
	return result
}

// resolveLua calls a Lua function and returns its result.
// Usage: ${lua function_name arg1 arg2 ...}
// If parse is true (${lua_parse}), the result is parsed for Conky variables.
// Arguments are passed as strings to the Lua function.
func (api *ConkyAPI) resolveLua(args []string, parse bool) string {
	if len(args) == 0 {
		return ""
	}

	funcName := args[0]

	// Build Lua arguments from remaining args
	luaArgs := make([]rt.Value, len(args)-1)
	for i, arg := range args[1:] {
		luaArgs[i] = rt.StringValue(arg)
	}

	// Call the Lua function
	result, err := api.runtime.CallFunction(funcName, luaArgs...)
	if err != nil {
		// Function not found or error - return empty string
		return ""
	}

	// Convert result to string
	resultStr := luaValueToString(result)

	// If lua_parse, parse the result for Conky variables
	if parse && resultStr != "" {
		return api.Parse(resultStr)
	}

	return resultStr
}

// resolveImage resolves the ${image} variable for embedding images.
// Syntax: ${image path [-s widthxheight] [-p x,y] [-n]}
// - path: file path to the image
// - -s widthxheight: resize to specified dimensions
// - -p x,y: absolute position from top-left corner
// - -n: disable caching (for dynamic images)
// Returns an image marker that the rendering layer will process.
func (api *ConkyAPI) resolveImage(args []string) string {
	if len(args) == 0 {
		return ""
	}

	path := args[0]
	var width, height float64
	x, y := float64(-1), float64(-1) // -1 means inline
	noCache := false

	// Parse optional arguments
	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-s":
			// Size: widthxheight
			if i+1 < len(args) {
				i++
				sizeStr := args[i]
				parts := strings.Split(sizeStr, "x")
				if len(parts) == 2 {
					w, err1 := strconv.ParseFloat(parts[0], 64)
					h, err2 := strconv.ParseFloat(parts[1], 64)
					if err1 == nil && err2 == nil {
						width = w
						height = h
					}
				}
			}
		case "-p":
			// Position: x,y
			if i+1 < len(args) {
				i++
				posStr := args[i]
				parts := strings.Split(posStr, ",")
				if len(parts) == 2 {
					px, err1 := strconv.ParseFloat(parts[0], 64)
					py, err2 := strconv.ParseFloat(parts[1], 64)
					if err1 == nil && err2 == nil {
						x = px
						y = py
					}
				}
			}
		case "-n":
			// No cache
			noCache = true
		}
	}

	return render.EncodeImageMarker(path, width, height, x, y, noCache)
}

// luaValueToString converts a Lua value to its string representation.
func luaValueToString(v rt.Value) string {
	switch v.Type() {
	case rt.StringType:
		s, _ := v.TryString()
		return s
	case rt.IntType:
		i, _ := v.TryInt()
		return strconv.FormatInt(i, 10)
	case rt.FloatType:
		f, _ := v.TryFloat()
		return strconv.FormatFloat(f, 'f', -1, 64)
	case rt.BoolType:
		if v.AsBool() {
			return "true"
		}
		return "false"
	case rt.NilType:
		return ""
	default:
		// For tables and other types, return empty
		return ""
	}
}

// SetCacheCleanupConfig updates the cache cleanup configuration.
func (api *ConkyAPI) SetCacheCleanupConfig(cfg CacheCleanupConfig) {
	api.mu.Lock()
	defer api.mu.Unlock()
	api.cleanupConfig = cfg
}

// CleanupCaches removes stale entries from both execCache and scrollStates.
// An entry is considered stale if it hasn't been accessed within MaxAge.
// Returns the number of entries removed from each cache.
func (api *ConkyAPI) CleanupCaches() (execRemoved, scrollRemoved int) {
	api.mu.Lock()
	defer api.mu.Unlock()

	now := time.Now()
	maxAge := api.cleanupConfig.MaxAge

	// Cleanup execCache - remove entries that are expired AND haven't been accessed recently
	for key, entry := range api.execCache {
		if now.Sub(entry.lastAccessed) > maxAge {
			delete(api.execCache, key)
			execRemoved++
		}
	}

	// Cleanup scrollStates - remove states that haven't been accessed recently
	for key, state := range api.scrollStates {
		if now.Sub(state.lastAccessed) > maxAge {
			delete(api.scrollStates, key)
			scrollRemoved++
		}
	}

	return execRemoved, scrollRemoved
}

// StartCacheCleanup starts a background goroutine that periodically cleans caches.
// Call StopCacheCleanup to stop the background cleanup.
func (api *ConkyAPI) StartCacheCleanup() {
	api.mu.Lock()
	if api.cleanupRunning {
		api.mu.Unlock()
		return
	}
	api.cleanupRunning = true
	api.cleanupStop = make(chan struct{})
	interval := api.cleanupConfig.CleanupInterval
	api.mu.Unlock()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				api.CleanupCaches()
			case <-api.cleanupStop:
				return
			}
		}
	}()
}

// StopCacheCleanup stops the background cache cleanup goroutine.
func (api *ConkyAPI) StopCacheCleanup() {
	api.mu.Lock()
	defer api.mu.Unlock()

	if !api.cleanupRunning {
		return
	}
	close(api.cleanupStop)
	api.cleanupRunning = false
}

// CacheStats returns the current number of entries in each cache.
func (api *ConkyAPI) CacheStats() (execCount, scrollCount int) {
	api.mu.RLock()
	defer api.mu.RUnlock()
	return len(api.execCache), len(api.scrollStates)
}
