package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

// New returns a configured zerolog logger.
// Outputs structured JSON suitable for shipping to Loki in future phases.
func New(site, hostname string) zerolog.Logger {
	zerolog.TimeFieldFormat = time.RFC3339

	return zerolog.New(os.Stdout).
		With().
		Timestamp().
		Str("site", site).
		Str("hostname", hostname).
		Str("app", "syswatch").
		Logger()
}
