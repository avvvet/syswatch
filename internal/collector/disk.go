package collector

import (
	"strings"

	"github.com/avvvet/syswatch/pkg/models"
	"github.com/jaypipes/ghw"
	"github.com/jaypipes/ghw/pkg/block"
)

// collectDisks reads storage device information using ghw.
func collectDisks() ([]models.Disk, error) {
	b, err := ghw.Block()
	if err != nil {
		return nil, err
	}

	var disks []models.Disk

	for _, d := range b.Disks {
		// Skip loop devices, ram disks, and device mapper
		if isVirtualDisk(d.Name) {
			continue
		}

		disk := models.Disk{
			Name:         d.Name,
			Model:        clean(d.Model),
			Serial:       clean(d.SerialNumber),
			SizeGB:       int(d.SizeBytes / (1024 * 1024 * 1024)),
			Type:         diskType(d),
			Manufacturer: extractManufacturer(d.Vendor, d.Model),
		}

		disks = append(disks, disk)
	}

	return disks, nil
}

// isVirtualDisk returns true for non-physical block devices.
func isVirtualDisk(name string) bool {
	virtual := []string{"loop", "ram", "dm-", "md", "sr", "fd"}
	for _, prefix := range virtual {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// diskType determines storage type using ghw DriveType field.
func diskType(d *ghw.Disk) string {
	// NVMe drives always start with nvme
	if strings.HasPrefix(d.Name, "nvme") {
		return "NVMe"
	}

	switch d.DriveType {
	case block.DRIVE_TYPE_SSD:
		return "SSD"
	case block.DRIVE_TYPE_HDD:
		return "HDD"
	default:
		return "Unknown"
	}
}

// extractManufacturer tries vendor field first then model string.
func extractManufacturer(vendor, model string) string {
	vendor = clean(vendor)
	if vendor != "" {
		return vendor
	}

	prefixes := map[string]string{
		"Samsung": "Samsung",
		"WDC":     "Western Digital",
		"WD":      "Western Digital",
		"ST":      "Seagate",
		"HGST":    "HGST",
		"TOSHIBA": "Toshiba",
		"INTEL":   "Intel",
		"MICRON":  "Micron",
		"KIOXIA":  "Kioxia",
	}

	modelUpper := strings.ToUpper(model)
	for prefix, mfr := range prefixes {
		if strings.HasPrefix(modelUpper, strings.ToUpper(prefix)) {
			return mfr
		}
	}

	return ""
}
