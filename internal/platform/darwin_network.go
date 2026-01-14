//go:build darwin
// +build darwin

package platform

import (
	"fmt"
	"net"
	"syscall"
	"unsafe"
)

// Sysctl MIB constants for Darwin network operations
const (
	ctlNet         = 4  // CTL_NET
	pfLink         = 18 // PF_LINK
	netlinkGeneric = 0  // NETLINK_GENERIC
	ifmibIfdata    = 2  // IFMIB_IFDATA
	ifdataGeneral  = 1  // IFDATA_GENERAL
)

// darwinNetworkProvider implements NetworkProvider for macOS/Darwin systems using getifaddrs and sysctl.
type darwinNetworkProvider struct{}

func newDarwinNetworkProvider() *darwinNetworkProvider {
	return &darwinNetworkProvider{}
}

// Interfaces returns a list of network interface names.
func (n *darwinNetworkProvider) Interfaces() ([]string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("getting network interfaces: %w", err)
	}

	names := make([]string, 0, len(ifaces))
	for _, iface := range ifaces {
		// Skip loopback interfaces
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		names = append(names, iface.Name)
	}

	return names, nil
}

// Stats returns network statistics for a specific interface.
func (n *darwinNetworkProvider) Stats(interfaceName string) (*NetworkStats, error) {
	// Get interface index
	iface, err := net.InterfaceByName(interfaceName)
	if err != nil {
		return nil, fmt.Errorf("interface %s not found: %w", interfaceName, err)
	}

	// Get network statistics using sysctl
	// macOS uses sysctl net.link.generic.ifdata to get interface stats
	stats, err := n.getIfData(iface.Index)
	if err != nil {
		return nil, fmt.Errorf("getting stats for %s: %w", interfaceName, err)
	}

	return stats, nil
}

// AllStats returns network statistics for all interfaces.
func (n *darwinNetworkProvider) AllStats() (map[string]*NetworkStats, error) {
	ifaces, err := n.Interfaces()
	if err != nil {
		return nil, err
	}

	allStats := make(map[string]*NetworkStats)
	for _, name := range ifaces {
		stats, err := n.Stats(name)
		if err != nil {
			// Skip interfaces that we can't get stats for
			continue
		}
		allStats[name] = stats
	}

	return allStats, nil
}

// ifData represents network interface statistics structure on macOS.
// This matches the if_data structure from <net/if.h>
type ifData struct {
	iType       uint8
	iTypelen    uint8
	iPhysical   uint8
	iAddrlen    uint8
	iHdrlen     uint8
	iRecvquota  uint8
	iXmitquota  uint8
	iUnused1    uint8
	iMtu        uint32
	iMetric     uint32
	iBaudrate   uint32
	iIpackets   uint32
	iIerrors    uint32
	iOpackets   uint32
	iOerrors    uint32
	iCollisions uint32
	iIbytes     uint32
	iObytes     uint32
	iImcasts    uint32
	iOmcasts    uint32
	iIqdrops    uint32
	iNoproto    uint32
	iRecvtiming uint32
	iXmittiming uint32
	iLastchange syscall.Timeval
	iUnused2    uint32
	iHwassist   uint32
	iReserved1  uint32
	iReserved2  uint32
}

// getIfData retrieves interface statistics using sysctl.
func (n *darwinNetworkProvider) getIfData(ifIndex int) (*NetworkStats, error) {
	// Use sysctl with net.link.generic.ifdata.INDEX.general
	// MIB: CTL_NET, PF_LINK, NETLINK_GENERIC, IFMIB_IFDATA, ifIndex, IFDATA_GENERAL
	mib := []int32{
		ctlNet,
		pfLink,
		netlinkGeneric,
		ifmibIfdata,
		int32(ifIndex),
		ifdataGeneral,
	}

	var data ifData
	dataSize := uintptr(unsafe.Sizeof(data))

	_, _, errno := syscall.Syscall6(
		syscall.SYS___SYSCTL,
		uintptr(unsafe.Pointer(&mib[0])),
		uintptr(len(mib)),
		uintptr(unsafe.Pointer(&data)),
		uintptr(unsafe.Pointer(&dataSize)),
		0,
		0,
	)

	if errno != 0 {
		return nil, fmt.Errorf("sysctl for interface %d failed: %w", ifIndex, errno)
	}

	return &NetworkStats{
		BytesRecv:   uint64(data.iIbytes),
		BytesSent:   uint64(data.iObytes),
		PacketsRecv: uint64(data.iIpackets),
		PacketsSent: uint64(data.iOpackets),
		ErrorsIn:    uint64(data.iIerrors),
		ErrorsOut:   uint64(data.iOerrors),
		DropIn:      uint64(data.iIqdrops),
		DropOut:     0, // macOS doesn't track output drops separately
	}, nil
}
