package collector

import (
	"fmt"
	"strings"

	"github.com/avvvet/syswatch/pkg/models"
	"github.com/digitalocean/go-smbios/smbios"
)

// collectMemory reads per-DIMM memory information from SMBIOS type 17.
// SMBIOS spec offsets are from start of structure.
// The go-smbios library strips the 4-byte header from Formatted,
// so all offsets must subtract 4 when indexing Formatted slice.
//
// Type 17 structure layout (spec offsets):
// 0x00-0x03  Header (stripped by library)
// 0x04-0x05  Physical Memory Array Handle
// 0x06-0x07  Memory Error Information Handle
// 0x08-0x09  Total Width
// 0x0A-0x0B  Data Width
// 0x0C-0x0D  Size  ← offset 0x0C in spec = index 0x08 in Formatted
// 0x0E       Form Factor
// 0x0F       Device Set
// 0x10       Device Locator String (index into strings)
// 0x11       Bank Locator String
// 0x12       Memory Type  ← offset 0x12 in spec = index 0x0E in Formatted
// 0x13-0x14  Type Detail
// 0x15-0x16  Speed  ← offset 0x15 in spec = index 0x11 in Formatted
// 0x17       Manufacturer String
// 0x18       Serial Number String
// 0x19       Asset Tag String
// 0x1A       Part Number String

func collectMemory() ([]models.MemoryModule, error) {
	stream, _, err := smbios.Stream()
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	decoder := smbios.NewDecoder(stream)
	structs, err := decoder.Decode()
	if err != nil {
		return nil, err
	}

	var modules []models.MemoryModule

	for _, s := range structs {
		if s.Header.Type != 17 {
			continue
		}

		// Need at least enough bytes to read size field
		// Size is at spec offset 0x0C = Formatted index 0x08
		if len(s.Formatted) < 0x0A {
			continue
		}

		// Size field — spec offset 0x0C, Formatted index 0x08
		sizeRaw := uint16(s.Formatted[0x08]) | uint16(s.Formatted[0x09])<<8

		// 0x0000 = no module installed — skip empty slots
		// 0xFFFF = unknown size
		if sizeRaw == 0 {
			continue
		}

		var sizeGB int
		if sizeRaw == 0xFFFF {
			// Extended size — check offset 0x1C (Formatted 0x18) for actual size in MB
			if len(s.Formatted) >= 0x1C {
				extSizeMB := uint32(s.Formatted[0x18]) |
					uint32(s.Formatted[0x19])<<8 |
					uint32(s.Formatted[0x1A])<<16 |
					uint32(s.Formatted[0x1B])<<24
				sizeGB = int(extSizeMB / 1024)
			}
		} else if sizeRaw&0x8000 != 0 {
			// bit 15 set = size in KB
			sizeKB := int(sizeRaw & 0x7FFF)
			sizeGB = sizeKB / (1024 * 1024)
		} else {
			// size in MB
			sizeMB := int(sizeRaw)
			sizeGB = sizeMB / 1024
		}

		module := models.MemoryModule{
			SizeGB: sizeGB,
		}

		// Memory type — spec offset 0x12, Formatted index 0x0E
		if len(s.Formatted) > 0x0E {
			module.Type = memoryType(s.Formatted[0x0E])
		}

		// Speed — spec offset 0x15, Formatted index 0x11 (16-bit MHz)
		if len(s.Formatted) > 0x12 {
			speed := uint16(s.Formatted[0x11]) | uint16(s.Formatted[0x12])<<8
			module.SpeedMHz = int(speed)
		}

		// String fields — indexes into s.Strings array
		// Device Locator is at spec offset 0x10, Formatted index 0x0C
		// The byte value is the 1-based string index
		if len(s.Formatted) > 0x0C && len(s.Strings) > 0 {
			locIdx := int(s.Formatted[0x0C])
			if locIdx > 0 && locIdx <= len(s.Strings) {
				module.Locator = clean(s.Strings[locIdx-1])
			}
		}

		// Manufacturer — spec offset 0x17, Formatted index 0x13
		if len(s.Formatted) > 0x13 && len(s.Strings) > 0 {
			mfrIdx := int(s.Formatted[0x13])
			if mfrIdx > 0 && mfrIdx <= len(s.Strings) {
				module.Manufacturer = clean(s.Strings[mfrIdx-1])
			}
		}

		// Serial — spec offset 0x18, Formatted index 0x14
		if len(s.Formatted) > 0x14 && len(s.Strings) > 0 {
			serIdx := int(s.Formatted[0x14])
			if serIdx > 0 && serIdx <= len(s.Strings) {
				module.Serial = clean(s.Strings[serIdx-1])
			}
		}

		// Part Number — spec offset 0x1A, Formatted index 0x16
		if len(s.Formatted) > 0x16 && len(s.Strings) > 0 {
			partIdx := int(s.Formatted[0x16])
			if partIdx > 0 && partIdx <= len(s.Strings) {
				module.PartNumber = clean(s.Strings[partIdx-1])
			}
		}

		// Fallback locator if empty
		if module.Locator == "" {
			module.Locator = fmt.Sprintf("DIMM-%d", len(modules))
		}

		modules = append(modules, module)
	}

	return modules, nil
}

// memoryType maps SMBIOS memory type codes to human readable strings.
func memoryType(code byte) string {
	types := map[byte]string{
		0x12: "Flash",
		0x13: "SDRAM",
		0x14: "SGRAM",
		0x15: "RDRAM",
		0x16: "DDR",
		0x17: "DDR2",
		0x18: "DDR3",
		0x19: "FBD2",
		0x1A: "DDR4",
		0x1B: "LPDDR",
		0x1C: "LPDDR2",
		0x1D: "LPDDR3",
		0x1E: "LPDDR4",
		0x1F: "Logical non-volatile",
		0x20: "HBM",
		0x21: "HBM2",
		0x22: "DDR5",
		0x23: "LPDDR5",
		0x24: "LPDDR5X",
	}

	if t, ok := types[code]; ok {
		return t
	}

	return fmt.Sprintf("Unknown(0x%02X)", code)
}

// totalRAMGB sums all installed memory modules.
func totalRAMGB(modules []models.MemoryModule) int {
	total := 0
	for _, m := range modules {
		total += m.SizeGB
	}
	return total
}

// ramSummary returns a human readable summary e.g. "16GB DDR5"
func ramSummary(modules []models.MemoryModule) string {
	if len(modules) == 0 {
		return ""
	}

	total := totalRAMGB(modules)
	memType := modules[0].Type

	for _, m := range modules {
		if m.Type != memType {
			memType = "Mixed"
			break
		}
	}

	return strings.TrimSpace(fmt.Sprintf("%dGB %s", total, memType))
}
