package collector

import (
	"strings"

	"github.com/avvvet/syswatch/pkg/models"
	"github.com/jaypipes/ghw"
)

// collectGPUs reads GPU information using ghw PCI device enumeration.
// Identifies display controllers (PCI class 0x03).
// Includes both integrated and discrete GPUs.
func collectGPUs() ([]models.GPU, error) {
	pci, err := ghw.PCI()
	if err != nil {
		return nil, err
	}

	var gpus []models.GPU

	for _, device := range pci.Devices {
		// PCI class 0x03 = Display Controller
		// Subclass 0x00 = VGA compatible
		// Subclass 0x02 = 3D controller (compute GPUs like A100)
		if device.Class == nil {
			continue
		}

		classID := device.Class.ID
		if classID != "0300" && classID != "0302" && classID != "03" {
			continue
		}

		gpu := models.GPU{
			Address: device.Address,
		}

		// Vendor name — fallback to vendor ID if name not resolved
		if device.Vendor != nil {
			name := cleanGPUString(device.Vendor.Name)
			if name == "" || name == "unknown" {
				gpu.Manufacturer = device.Vendor.ID
			} else {
				gpu.Manufacturer = name
			}
		}

		// Product name — fallback to product ID if name not resolved
		if device.Product != nil {
			name := cleanGPUString(device.Product.Name)
			if name == "" || name == "unknown" {
				gpu.Name = device.Product.ID
			} else {
				gpu.Name = name
			}
		}

		// Skip generic VGA fallbacks and virtual GPUs
		nameLower := strings.ToLower(gpu.Name)
		if gpu.Name == "" ||
			strings.Contains(nameLower, "bochs") ||
			strings.Contains(nameLower, "virtualbox") ||
			strings.Contains(nameLower, "vmware") {
			continue
		}

		gpus = append(gpus, gpu)
	}

	return gpus, nil
}

// cleanGPUString removes common vendor suffixes from GPU names.
func cleanGPUString(s string) string {
	s = strings.TrimSpace(s)
	// Remove common suffixes that add noise
	noise := []string{
		" Corporation",
		" Technologies",
		" Inc.",
		" Inc",
		" Ltd.",
		" Ltd",
		" [",
	}
	for _, n := range noise {
		if idx := strings.Index(s, n); idx > 0 {
			s = s[:idx]
		}
	}
	return strings.TrimSpace(s)
}
