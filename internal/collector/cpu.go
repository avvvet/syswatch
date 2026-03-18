package collector

import (
	"github.com/jaypipes/ghw"
	"github.com/avvvet/syswatch/pkg/models"
)

// collectCPU reads processor information using ghw.
func collectCPU() (models.CPU, error) {
	cpu, err := ghw.CPU()
	if err != nil {
		return models.CPU{}, err
	}

	if len(cpu.Processors) == 0 {
		return models.CPU{}, nil
	}

	// Use first processor for model name
	// All sockets in a server are typically the same model
	first := cpu.Processors[0]

	result := models.CPU{
		Model:   first.Model,
		Sockets: len(cpu.Processors),
	}

	// Count physical cores per socket
	result.Cores = int(first.NumCores)
	result.TotalCores = result.Cores * result.Sockets

	return result, nil
}
