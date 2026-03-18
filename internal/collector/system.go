package collector

import (
	"strings"

	"github.com/avvvet/syswatch/pkg/models"
	"github.com/digitalocean/go-smbios/smbios"
)

// collectSystem reads system identity from SMBIOS/DMI tables.
// This is equivalent to dmidecode -t system and -t bios.
func collectSystem() (models.System, error) {
	stream, _, err := smbios.Stream()
	if err != nil {
		return models.System{}, err
	}
	defer stream.Close()

	decoder := smbios.NewDecoder(stream)
	structs, err := decoder.Decode()
	if err != nil {
		return models.System{}, err
	}

	system := models.System{}

	for _, s := range structs {
		switch s.Header.Type {
		case 1: // System Information
			if len(s.Strings) >= 5 {
				system.Manufacturer = clean(s.Strings[0])
				system.Model = clean(s.Strings[1])
				system.Serial = clean(s.Strings[3])
			}

		case 0: // BIOS Information
			if len(s.Strings) >= 2 {
				system.BIOSVersion = clean(s.Strings[1])
			}

		case 2: // Base Board (Motherboard) Information
			// Collect motherboard serial as fallback identifier
			// Only set if system serial is empty
			if system.Serial == "" && len(s.Strings) >= 4 {
				system.MotherboardSerial = clean(s.Strings[3])
			}

		case 3: // Chassis Information
			// u_height is not always available via SMBIOS
			// default to 1U — can be overridden via config
			system.UHeight = 1
		}
	}

	return system, nil
}

// clean removes whitespace and filters placeholder values
// that vendors put when data is not available.
func clean(s string) string {
	s = strings.TrimSpace(s)

	// Common placeholder values returned by vendors
	placeholders := []string{
		"Not Specified",
		"Not Present",
		"To Be Filled By O.E.M.",
		"Default string",
		"",
	}

	for _, p := range placeholders {
		if strings.EqualFold(s, p) {
			return ""
		}
	}

	return s
}
