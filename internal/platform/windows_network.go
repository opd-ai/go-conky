// +build windows

package platform

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	modIphlpapi       = syscall.NewLazyDLL("iphlpapi.dll")
	procGetIfTable2   = modIphlpapi.NewProc("GetIfTable2")
	procFreeMibTable  = modIphlpapi.NewProc("FreeMibTable")
)

const (
	MAX_INTERFACE_NAME_LEN = 256
	IF_MAX_STRING_SIZE     = 256
)

// mibIfRow2 matches a subset of the Windows MIB_IF_ROW2 structure
type mibIfRow2 struct {
	InterfaceLuid          uint64
	InterfaceIndex         uint32
	InterfaceGuid          [16]byte
	Alias                  [MAX_INTERFACE_NAME_LEN + 1]uint16
	Description            [MAX_INTERFACE_NAME_LEN + 1]uint16
	PhysicalAddressLength  uint32
	PhysicalAddress        [32]byte
	PermanentPhysicalAddress [32]byte
	Mtu                    uint32
	Type                   uint32
	TunnelType             uint32
	MediaType              uint32
	PhysicalMediumType     uint32
	AccessType             uint32
	DirectionType          uint32
	InterfaceAndOperStatusFlags byte
	OperStatus             uint32
	AdminStatus            uint32
	MediaConnectState      uint32
	NetworkGuid            [16]byte
	ConnectionType         uint32
	TransmitLinkSpeed      uint64
	ReceiveLinkSpeed       uint64
	InOctets               uint64
	InUcastPkts            uint64
	InNUcastPkts           uint64
	InDiscards             uint64
	InErrors               uint64
	InUnknownProtos        uint64
	InUcastOctets          uint64
	InMulticastOctets      uint64
	InBroadcastOctets      uint64
	OutOctets              uint64
	OutUcastPkts           uint64
	OutNUcastPkts          uint64
	OutDiscards            uint64
	OutErrors              uint64
	OutUcastOctets         uint64
	OutMulticastOctets     uint64
	OutBroadcastOctets     uint64
	OutQLen                uint64
}

// mibIfTable2 matches the Windows MIB_IF_TABLE2 structure
type mibIfTable2 struct {
	NumEntries uint32
	Table      [1]mibIfRow2 // Variable-length array
}

// windowsNetworkProvider implements NetworkProvider for Windows systems using GetIfTable2
type windowsNetworkProvider struct{}

func newWindowsNetworkProvider() *windowsNetworkProvider {
	return &windowsNetworkProvider{}
}

// getInterfaceTable retrieves the network interface table from Windows.
// Note: This uses unsafe pointer arithmetic to access the variable-length array
// in the MIB_IF_TABLE2 structure. The structure layout must match the Windows API
// definition exactly. If Windows changes the structure layout in future versions,
// this could cause memory corruption.
func (n *windowsNetworkProvider) getInterfaceTable() ([]mibIfRow2, error) {
	var table *mibIfTable2
	ret, _, _ := procGetIfTable2.Call(uintptr(unsafe.Pointer(&table)))
	if ret != 0 {
		return nil, fmt.Errorf("GetIfTable2 failed with status 0x%x", ret)
	}
	defer procFreeMibTable.Call(uintptr(unsafe.Pointer(table)))

	if table == nil || table.NumEntries == 0 {
		return nil, nil
	}

	// Extract interface rows from the table
	// We use unsafe pointer arithmetic to access the variable-length array
	// Verify the structure size matches expected size
	const expectedRowSize = unsafe.Sizeof(mibIfRow2{})
	rows := make([]mibIfRow2, table.NumEntries)
	tablePtr := uintptr(unsafe.Pointer(&table.Table[0]))
	rowSize := expectedRowSize

	for i := uint32(0); i < table.NumEntries; i++ {
		rowPtr := tablePtr + uintptr(i)*rowSize
		rows[i] = *(*mibIfRow2)(unsafe.Pointer(rowPtr))
	}

	return rows, nil
}

func (n *windowsNetworkProvider) Interfaces() ([]string, error) {
	rows, err := n.getInterfaceTable()
	if err != nil {
		return nil, err
	}

	interfaces := make([]string, 0, len(rows))
	for _, row := range rows {
		// Convert UTF-16 alias to string
		name := syscall.UTF16ToString(row.Alias[:])
		if name != "" {
			interfaces = append(interfaces, name)
		}
	}

	return interfaces, nil
}

func (n *windowsNetworkProvider) Stats(interfaceName string) (*NetworkStats, error) {
	rows, err := n.getInterfaceTable()
	if err != nil {
		return nil, err
	}

	// Find the interface by name
	for _, row := range rows {
		name := syscall.UTF16ToString(row.Alias[:])
		if name == interfaceName {
			return &NetworkStats{
				BytesRecv:   row.InOctets,
				BytesSent:   row.OutOctets,
				PacketsRecv: row.InUcastPkts + row.InNUcastPkts,
				PacketsSent: row.OutUcastPkts + row.OutNUcastPkts,
				ErrorsIn:    row.InErrors,
				ErrorsOut:   row.OutErrors,
				DropIn:      row.InDiscards,
				DropOut:     row.OutDiscards,
			}, nil
		}
	}

	return nil, fmt.Errorf("interface %s not found", interfaceName)
}

func (n *windowsNetworkProvider) AllStats() (map[string]*NetworkStats, error) {
	rows, err := n.getInterfaceTable()
	if err != nil {
		return nil, err
	}

	stats := make(map[string]*NetworkStats)
	for _, row := range rows {
		name := syscall.UTF16ToString(row.Alias[:])
		if name != "" {
			stats[name] = &NetworkStats{
				BytesRecv:   row.InOctets,
				BytesSent:   row.OutOctets,
				PacketsRecv: row.InUcastPkts + row.InNUcastPkts,
				PacketsSent: row.OutUcastPkts + row.OutNUcastPkts,
				ErrorsIn:    row.InErrors,
				ErrorsOut:   row.OutErrors,
				DropIn:      row.InDiscards,
				DropOut:     row.OutDiscards,
			}
		}
	}

	return stats, nil
}
