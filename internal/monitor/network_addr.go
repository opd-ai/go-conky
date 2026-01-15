package monitor

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"strings"
)

// networkAddressReader reads network address information from the system.
// It uses the net package for interface addresses and parses /proc and /etc files
// for gateway and DNS information.
type networkAddressReader struct {
	procRoutePath   string
	resolvConfPath  string
	getInterfacesFn func() ([]net.Interface, error)
	getAddrsFn      func(iface *net.Interface) ([]net.Addr, error)
}

// newNetworkAddressReader creates a new networkAddressReader with default paths.
func newNetworkAddressReader() *networkAddressReader {
	return &networkAddressReader{
		procRoutePath:   "/proc/net/route",
		resolvConfPath:  "/etc/resolv.conf",
		getInterfacesFn: net.Interfaces,
		getAddrsFn:      func(iface *net.Interface) ([]net.Addr, error) { return iface.Addrs() },
	}
}

// ReadInterfaceAddresses returns a map of interface name to IP addresses.
func (r *networkAddressReader) ReadInterfaceAddresses() (map[string]InterfaceAddrs, error) {
	interfaces, err := r.getInterfacesFn()
	if err != nil {
		return nil, fmt.Errorf("getting interfaces: %w", err)
	}

	result := make(map[string]InterfaceAddrs, len(interfaces))

	for _, iface := range interfaces {
		addrs, err := r.getAddrsFn(&iface)
		if err != nil {
			// Skip interfaces we can't read
			continue
		}

		ifAddrs := InterfaceAddrs{
			IPv4: make([]string, 0),
			IPv6: make([]string, 0),
		}

		for _, addr := range addrs {
			ip, _, err := net.ParseCIDR(addr.String())
			if err != nil {
				continue
			}

			// Classify as IPv4 or IPv6
			if ip.To4() != nil {
				ifAddrs.IPv4 = append(ifAddrs.IPv4, ip.String())
			} else {
				ifAddrs.IPv6 = append(ifAddrs.IPv6, ip.String())
			}
		}

		result[iface.Name] = ifAddrs
	}

	return result, nil
}

// InterfaceAddrs holds IPv4 and IPv6 addresses for an interface.
type InterfaceAddrs struct {
	IPv4 []string
	IPv6 []string
}

// ReadDefaultGateway reads the default gateway from /proc/net/route.
// Returns the gateway IP address and the interface name.
func (r *networkAddressReader) ReadDefaultGateway() (string, string, error) {
	file, err := os.Open(r.procRoutePath)
	if err != nil {
		return "", "", fmt.Errorf("opening %s: %w", r.procRoutePath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		// Skip header line
		if lineNum == 1 {
			continue
		}

		line := scanner.Text()
		iface, gateway, err := parseRouteLine(line)
		if err != nil {
			continue
		}

		// Found default gateway (destination 00000000)
		if gateway != "" {
			return gateway, iface, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", "", fmt.Errorf("scanning %s: %w", r.procRoutePath, err)
	}

	return "", "", nil // No default gateway found
}

// parseRouteLine parses a line from /proc/net/route.
// Format: Iface Destination Gateway Flags RefCnt Use Metric Mask MTU Window IRTT
// Returns interface name and gateway IP if this is the default route, empty strings otherwise.
func parseRouteLine(line string) (string, string, error) {
	fields := strings.Fields(line)
	if len(fields) < 4 {
		return "", "", fmt.Errorf("insufficient fields: got %d, need 4", len(fields))
	}

	iface := fields[0]
	destination := fields[1]
	gatewayHex := fields[2]

	// Default route has destination 00000000
	if destination != "00000000" {
		return "", "", nil
	}

	// Parse gateway hex to IP
	gateway, err := hexToIP(gatewayHex)
	if err != nil {
		return "", "", fmt.Errorf("parsing gateway: %w", err)
	}

	return iface, gateway, nil
}

// hexToIP converts a hex string (little-endian) to an IP address string.
// Linux /proc/net/route uses little-endian hex representation.
func hexToIP(hexStr string) (string, error) {
	if len(hexStr) != 8 {
		return "", fmt.Errorf("invalid hex length: %d", len(hexStr))
	}

	// Parse hex string to uint32
	var ipBytes [4]byte
	for i := 0; i < 4; i++ {
		// Read bytes in pairs, from right to left (little-endian)
		byteIdx := 3 - i
		hexByte := hexStr[byteIdx*2 : byteIdx*2+2]
		var b uint8
		_, err := fmt.Sscanf(hexByte, "%02X", &b)
		if err != nil {
			return "", fmt.Errorf("parsing hex byte %s: %w", hexByte, err)
		}
		ipBytes[i] = b
	}

	ip := net.IPv4(ipBytes[0], ipBytes[1], ipBytes[2], ipBytes[3])
	return ip.String(), nil
}

// hexToIPLE converts a hex string in little-endian format to IP address.
// This is an alternative implementation using binary package.
func hexToIPLE(hexStr string) (string, error) {
	if len(hexStr) != 8 {
		return "", fmt.Errorf("invalid hex length: %d", len(hexStr))
	}

	var ipUint32 uint32
	_, err := fmt.Sscanf(hexStr, "%X", &ipUint32)
	if err != nil {
		return "", fmt.Errorf("parsing hex: %w", err)
	}

	// Convert from little-endian to byte array
	ipBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(ipBytes, ipUint32)

	ip := net.IPv4(ipBytes[0], ipBytes[1], ipBytes[2], ipBytes[3])
	return ip.String(), nil
}

// ReadNameservers reads DNS nameservers from /etc/resolv.conf.
func (r *networkAddressReader) ReadNameservers() ([]string, error) {
	file, err := os.Open(r.resolvConfPath)
	if err != nil {
		// resolv.conf might not exist in containers
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("opening %s: %w", r.resolvConfPath, err)
	}
	defer file.Close()

	var nameservers []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		// Parse nameserver lines
		if strings.HasPrefix(line, "nameserver") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				// Validate it's a valid IP
				if ip := net.ParseIP(parts[1]); ip != nil {
					nameservers = append(nameservers, parts[1])
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning %s: %w", r.resolvConfPath, err)
	}

	return nameservers, nil
}
