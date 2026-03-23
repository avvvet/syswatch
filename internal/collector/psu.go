package collector

import (
	"fmt"

	"github.com/avvvet/syswatch/pkg/models"
	"github.com/digitalocean/go-smbios/smbios"
)

// collectPowerSupplies reads PSU information from SMBIOS Type 39.
// Equivalent to: dmidecode -t 39
func collectPowerSupplies() ([]models.PowerSupply, error) {
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

	var psus []models.PowerSupply
	index := 0

	for _, s := range structs {
		if s.Header.Type != 39 { // Type 39 = System Power Supply
			continue
		}

		psu := models.PowerSupply{
			Name:   fmt.Sprintf("PSU %d", index),
			Status: "present",
		}

		// SMBIOS Type 39 string fields:
		// [0] Location
		// [1] Device Name
		// [2] Manufacturer
		// [3] Serial Number
		// [4] Asset Tag
		// [5] Model Part Number
		if len(s.Strings) >= 3 {
			psu.Manufacturer = clean(s.Strings[2])
		}
		if len(s.Strings) >= 4 {
			psu.Serial = clean(s.Strings[3])
		}
		if len(s.Strings) >= 6 {
			psu.PartNumber = clean(s.Strings[5])
		}

		// Max wattage is in the structured data (bytes 14-15)
		// Value is in milliwatts, divide by 1000 for watts
		// Only valid if bit 1 of characteristics word is set
		if len(s.Formatted) >= 14 {
			maxPowerRaw := int(s.Formatted[12]) | int(s.Formatted[13])<<8
			if maxPowerRaw > 0 && maxPowerRaw != 0x8000 {
				psu.MaxWatts = maxPowerRaw / 1000
				if psu.MaxWatts == 0 {
					psu.MaxWatts = maxPowerRaw // some vendors report in watts directly
				}
			}
		}

		psus = append(psus, psu)
		index++
	}

	return psus, nil
}
