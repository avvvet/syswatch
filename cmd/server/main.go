package main

import (
	"os"

	"github.com/avvvet/syswatch/internal/collector"
	"github.com/avvvet/syswatch/internal/config"
	"github.com/avvvet/syswatch/internal/intelligence"
	"github.com/avvvet/syswatch/internal/logger"
	"github.com/avvvet/syswatch/internal/netbox"
)

func main() {
	cfg, err := config.LoadFromEnvFile(".env")
	if err != nil {
		os.Stderr.WriteString("config error: " + err.Error() + "\n")
		os.Exit(1)
	}

	hostname, _ := os.Hostname()
	log := logger.New(cfg.Site, hostname)

	log.Info().
		Str("site", cfg.Site).
		Str("role", cfg.Role).
		Str("mode", cfg.Mode).
		Msg("syswatch server starting")

	// Collect hardware
	c := collector.New(log)
	hw := c.CollectAll()

	if hw.Identifier.Value == "" {
		log.Error().Msg("cannot identify this device — no identifier found")
		os.Exit(1)
	}

	// Build NetBox client
	nbClient := netbox.NewClient(cfg.NetBoxURL, cfg.NetBoxToken, log)

	// Build syncer based on mode
	var syncer *netbox.Syncer

	if cfg.IsAPIMode() {
		// API mode — use Central Intelligence API
		apiClient := intelligence.New(cfg.APIUrl, cfg.APIKey)

		// Verify API is reachable
		if err := apiClient.Ping(); err != nil {
			log.Error().Err(err).Msg("Central Intelligence API unreachable")
			os.Exit(1)
		}
		log.Info().Str("url", cfg.APIUrl).Msg("Central Intelligence API connected")

		syncer = netbox.NewSyncerWithAPI(nbClient, cfg.Site, cfg.Role, apiClient)
	} else {
		// Standalone mode
		log.Info().Msg("running in standalone mode")
		syncer = netbox.NewSyncer(nbClient, cfg.Site, cfg.Role)
	}

	// Sync to NetBox
	if err := syncer.Sync(hw); err != nil {
		log.Error().Err(err).Msg("NetBox sync failed")
		os.Exit(1)
	}

	log.Info().Msg("syswatch completed successfully")
}
