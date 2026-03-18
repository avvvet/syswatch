package collector

import (
	"fmt"
	"os"

	"github.com/avvvet/syswatch/pkg/models"
	"github.com/rs/zerolog"
)

// Collector orchestrates all hardware collectors.
type Collector struct {
	log zerolog.Logger
}

// New creates a new Collector instance.
func New(log zerolog.Logger) *Collector {
	return &Collector{log: log}
}

// CollectAll runs all collectors and returns a complete Hardware snapshot.
// Errors from individual collectors are logged but do not stop the process.
func (c *Collector) CollectAll() models.Hardware {
	hw := models.Hardware{}

	hostname, _ := os.Hostname()
	c.log.Info().Str("hostname", hostname).Msg("starting hardware collection")

	// System identity
	system, err := collectSystem()
	if err != nil {
		c.log.Warn().Err(err).Msg("system collection failed")
	} else {
		hw.System = system
		c.log.Info().
			Str("manufacturer", system.Manufacturer).
			Str("model", system.Model).
			Str("serial", system.Serial).
			Msg("system collected")
	}

	// CPU
	cpu, err := collectCPU()
	if err != nil {
		c.log.Warn().Err(err).Msg("cpu collection failed")
	} else {
		hw.CPU = cpu
		c.log.Info().
			Str("model", cpu.Model).
			Int("sockets", cpu.Sockets).
			Int("total_cores", cpu.TotalCores).
			Msg("cpu collected")
	}

	// Memory
	memory, err := collectMemory()
	if err != nil {
		c.log.Warn().Err(err).Msg("memory collection failed")
	} else {
		hw.Memory = memory
		c.log.Info().
			Int("dimm_count", len(memory)).
			Str("total", ramSummary(memory)).
			Msg("memory collected")
	}

	// Disks
	disks, err := collectDisks()
	if err != nil {
		c.log.Warn().Err(err).Msg("disk collection failed")
	} else {
		hw.Disks = disks
		c.log.Info().
			Int("disk_count", len(disks)).
			Msg("disks collected")
	}

	// NICs
	nics, err := collectNICs()
	if err != nil {
		c.log.Warn().Err(err).Msg("nic collection failed")
	} else {
		hw.NICs = nics
		c.log.Info().
			Int("nic_count", len(nics)).
			Msg("nics collected")
	}

	// OS
	osInfo, err := collectOS()
	if err != nil {
		c.log.Warn().Err(err).Msg("os collection failed")
	} else {
		hw.OS = osInfo
		c.log.Info().
			Str("os", fmt.Sprintf("%s %s", osInfo.Name, osInfo.Version)).
			Str("kernel", osInfo.Kernel).
			Msg("os collected")
	}

	// Resolve unique identifier using fallback chain
	nicsForID := make([]nicForIdentifier, len(hw.NICs))
	for i, nic := range hw.NICs {
		nicsForID[i] = nicForIdentifier{
			Name:       nic.Name,
			MACAddress: nic.MACAddress,
		}
	}

	hw.Identifier = ResolveIdentifier(
		hw.System.Serial,
		hw.System.MotherboardSerial,
		nicsForID,
	)

	// Log identifier resolution result
	if hw.Identifier.Value == "" {
		c.log.Error().Msg("could not resolve unique identifier — all fallbacks exhausted")
	} else {
		c.log.Info().
			Str("identifier", hw.Identifier.Value).
			Str("source", hw.Identifier.Source).
			Msg("identifier resolved")

		// Warn if we fell back from serial number
		if hw.Identifier.Source != "smbios-serial" {
			c.log.Warn().
				Str("source", hw.Identifier.Source).
				Msg("serial number not available — using fallback identifier — device will be tagged in NetBox")
		}
	}

	c.log.Info().Msg("hardware collection complete")
	return hw
}
