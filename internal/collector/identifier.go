package collector

import (
	"os"
	"strings"

	"github.com/avvvet/syswatch/pkg/models"
)

// ResolveIdentifier tries multiple sources in order
// to find a reliable unique hardware identifier.
//
// Fallback chain:
// 1. SMBIOS system serial number
// 2. Motherboard serial number
// 3. Primary NIC MAC address
// 4. Machine ID (/etc/machine-id)
func ResolveIdentifier(systemSerial, motherboardSerial string, nics []nicForIdentifier) models.Identifier {
	// Attempt 1 — SMBIOS system serial
	if systemSerial != "" {
		return models.Identifier{
			Value:  systemSerial,
			Source: "smbios-serial",
		}
	}

	// Attempt 2 — Motherboard serial
	if motherboardSerial != "" {
		return models.Identifier{
			Value:  motherboardSerial,
			Source: "motherboard-serial",
		}
	}

	// Attempt 3 — Primary NIC MAC address
	for _, nic := range nics {
		if nic.MACAddress != "" && !isVirtualMAC(nic.MACAddress) {
			return models.Identifier{
				Value:  nic.MACAddress,
				Source: "mac-address",
			}
		}
	}

	// Attempt 4 — Machine ID
	if machineID := readMachineID(); machineID != "" {
		return models.Identifier{
			Value:  machineID,
			Source: "machine-id",
		}
	}

	// All attempts failed
	return models.Identifier{
		Value:  "",
		Source: "none",
	}
}

// readMachineID reads /etc/machine-id which is generated
// on first OS boot and is unique per installation.
func readMachineID() string {
	data, err := os.ReadFile("/etc/machine-id")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// isVirtualMAC returns true for MAC addresses known
// to belong to virtual interfaces.
func isVirtualMAC(mac string) bool {
	if len(mac) < 2 {
		return true
	}

	virtualPrefixes := []string{
		"00:00:00",
		"02:42",
		"52:54:00",
		"00:50:56",
		"00:0c:29",
		"00:15:5d",
		"08:00:27",
	}

	macLower := strings.ToLower(mac)
	for _, prefix := range virtualPrefixes {
		if strings.HasPrefix(macLower, prefix) {
			return true
		}
	}

	return false
}

// nicForIdentifier is a minimal NIC struct
// used only for identifier resolution.
type nicForIdentifier struct {
	Name       string
	MACAddress string
}
