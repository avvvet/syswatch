package collector

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/avvvet/syswatch/pkg/models"
)

// collectNICs reads network interface information from the OS.
// Uses standard library net package plus /sys/class/net/ for speed.
func collectNICs() ([]models.NIC, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var nics []models.NIC

	for _, iface := range ifaces {
		// Skip loopback and virtual interfaces
		if isVirtualInterface(iface) {
			continue
		}

		nic := models.NIC{
			Name:       iface.Name,
			MACAddress: iface.HardwareAddr.String(),
			SpeedMbps:  readInterfaceSpeed(iface.Name),
		}

		nic.Type = speedToNetBoxType(nic.SpeedMbps)

		// Get assigned IP address if any
		nic.IPAddress = readInterfaceIP(iface)

		nics = append(nics, nic)
	}

	return nics, nil
}

// isVirtualInterface returns true for non-physical interfaces.
func isVirtualInterface(iface net.Interface) bool {
	// Skip loopback
	if iface.Flags&net.FlagLoopback != 0 {
		return true
	}

	// Skip interfaces with no MAC (tunnels, bridges etc)
	if iface.HardwareAddr == nil || len(iface.HardwareAddr) == 0 {
		return true
	}

	// Skip common virtual interface prefixes
	virtual := []string{"lo", "docker", "veth", "virbr", "br-", "tun", "tap", "vlan", "bond"}
	for _, prefix := range virtual {
		if strings.HasPrefix(iface.Name, prefix) {
			return true
		}
	}

	// Check /sys/class/net/<iface>/device — physical NICs have this
	devicePath := fmt.Sprintf("/sys/class/net/%s/device", iface.Name)
	if _, err := os.Stat(devicePath); os.IsNotExist(err) {
		return true
	}

	return false
}

// readInterfaceSpeed reads link speed from /sys/class/net/<iface>/speed
// Returns 0 if not available.
func readInterfaceSpeed(name string) int {
	path := fmt.Sprintf("/sys/class/net/%s/speed", name)
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}

	speed, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || speed <= 0 {
		return 0
	}

	return speed
}

// readInterfaceIP returns the first assigned IPv4 address with prefix.
func readInterfaceIP(iface net.Interface) string {
	addrs, err := iface.Addrs()
	if err != nil {
		return ""
	}

	for _, addr := range addrs {
		// Only return IPv4
		if ipnet, ok := addr.(*net.IPNet); ok {
			if ip4 := ipnet.IP.To4(); ip4 != nil {
				ones, _ := ipnet.Mask.Size()
				return fmt.Sprintf("%s/%d", ip4.String(), ones)
			}
		}
	}

	return ""
}

// speedToNetBoxType maps link speed to NetBox interface type string.
// These are the valid NetBox type values we confirmed via API testing.
func speedToNetBoxType(speedMbps int) string {
	switch {
	case speedMbps >= 100000:
		return "100gbase-x-qsfp28"
	case speedMbps >= 40000:
		return "40gbase-x-qsfpp"
	case speedMbps >= 25000:
		return "25gbase-x-sfp28"
	case speedMbps >= 10000:
		return "10gbase-x-sfpp"
	case speedMbps >= 1000:
		return "1000base-t"
	default:
		return "other"
	}
}
