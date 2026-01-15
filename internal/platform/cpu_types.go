package platform

// cpuTimes stores raw CPU time values for calculating CPU usage.
// This type is shared between local Linux/Darwin providers and remote SSH providers.
//
// Field availability by platform:
// - Linux: All fields (user, nice, system, idle, iowait, irq, softirq, steal)
// - macOS/Darwin: Uses user, nice, system, idle only (other fields will be 0)
// - Windows: Not used (Windows uses different mechanism via WMI/PDH)
type cpuTimes struct {
	user    uint64
	nice    uint64
	system  uint64
	idle    uint64
	iowait  uint64 // Linux-specific
	irq     uint64 // Linux-specific
	softirq uint64 // Linux-specific
	steal   uint64 // Linux-specific
}
