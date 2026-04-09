package collector

import (
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/avvvet/syswatch/pkg/models"
	"github.com/digitalocean/go-smbios/smbios"
)

// collectOOBIP detects the out-of-band management IP address.
// This is the iDRAC (Dell), iLO (HP), or BMC IP used for
// remote server management independent of the main OS.
//
// Detection uses two methods:
//  1. SMBIOS Type 38 — IPMI Device Info
//  2. Virtual BMC network interface scan (fallback)
func collectOOBIP() (models.OOBInterface, error) {
	// Method 1 — SMBIOS Type 38
	oob, err := collectOOBFromSMBIOS()
	if err == nil && oob.IPAddress != "" {
		return oob, nil
	}

	// Method 2 — Virtual BMC interface scan
	return collectOOBFromNetInterface()
}

// collectOOBFromSMBIOS reads BMC info from SMBIOS Type 38.
func collectOOBFromSMBIOS() (models.OOBInterface, error) {
	stream, _, err := smbios.Stream()
	if err != nil {
		return models.OOBInterface{}, err
	}
	defer stream.Close()

	decoder := smbios.NewDecoder(stream)
	structs, err := decoder.Decode()
	if err != nil {
		return models.OOBInterface{}, err
	}

	for _, s := range structs {
		if s.Header.Type != 38 {
			continue
		}
		if len(s.Formatted) >= 1 {
			return models.OOBInterface{
				Name: bmcInterfaceType(s.Formatted[0]),
			}, nil
		}
	}

	return models.OOBInterface{}, nil
}

// collectOOBFromNetInterface scans for virtual BMC network interfaces.
func collectOOBFromNetInterface() (models.OOBInterface, error) {
	bmcNames := []string{"idrac", "ilo", "bmc", "ipmi"}

	entries, err := os.ReadDir("/sys/class/net")
	if err != nil {
		return models.OOBInterface{}, err
	}

	for _, entry := range entries {
		name := strings.ToLower(entry.Name())

		for _, pattern := range bmcNames {
			if !strings.Contains(name, pattern) {
				continue
			}

			iface, err := net.InterfaceByName(entry.Name())
			if err != nil {
				continue
			}

			addrs, err := iface.Addrs()
			if err != nil || len(addrs) == 0 {
				continue
			}

			for _, addr := range addrs {
				ip, _, err := net.ParseCIDR(addr.String())
				if err != nil {
					continue
				}
				if ip.To4() != nil {
					return models.OOBInterface{
						Name:      entry.Name(),
						IPAddress: addr.String(),
					}, nil
				}
			}
		}
	}

	// Fallback — check USB ethernet (Dell iDRAC USB NIC)
	return collectOOBFromUSBEthernet()
}

// collectOOBFromUSBEthernet checks USB ethernet interfaces
// that Dell iDRAC creates for its virtual NIC.
func collectOOBFromUSBEthernet() (models.OOBInterface, error) {
	entries, err := os.ReadDir("/sys/class/net")
	if err != nil {
		return models.OOBInterface{}, err
	}

	for _, entry := range entries {
		devicePath := filepath.Join("/sys/class/net", entry.Name(), "device", "subsystem")
		target, err := os.Readlink(devicePath)
		if err != nil || !strings.Contains(target, "usb") {
			continue
		}

		iface, err := net.InterfaceByName(entry.Name())
		if err != nil {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil || len(addrs) == 0 {
			continue
		}

		for _, addr := range addrs {
			ip, _, err := net.ParseCIDR(addr.String())
			if err != nil || ip.To4() == nil {
				continue
			}
			ipStr := ip.String()
			if strings.HasPrefix(ipStr, "169.254.") || strings.HasPrefix(ipStr, "192.168.") {
				return models.OOBInterface{
					Name:      entry.Name(),
					IPAddress: addr.String(),
				}, nil
			}
		}
	}

	return models.OOBInterface{}, nil
}

// bmcInterfaceType returns a human readable BMC type name.
func bmcInterfaceType(t byte) string {
	switch t {
	case 1:
		return "KCS"
	case 2:
		return "SMIC"
	case 3:
		return "BT"
	case 4:
		return "SSIF"
	default:
		return "IPMI"
	}
}
