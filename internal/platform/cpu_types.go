package platform

// cpuTimes stores raw CPU time values for calculating CPU usage.
// This type is shared between local Linux providers and remote SSH providers.
type cpuTimes struct {
	user    uint64
	nice    uint64
	system  uint64
	idle    uint64
	iowait  uint64
	irq     uint64
	softirq uint64
	steal   uint64
}
