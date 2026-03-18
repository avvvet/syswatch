package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration for syswatch.
type Config struct {
	// NetBox
	NetBoxURL   string
	NetBoxToken string
	Site        string
	Role        string

	// Mode: standalone or api
	Mode string

	// API mode — Central Intelligence API
	APIUrl string
	APIKey string
}

// LoadFromEnvFile loads config from a .env file or environment variables.
func LoadFromEnvFile(path string) (*Config, error) {
	_ = godotenv.Load(path)

	cfg := &Config{
		NetBoxURL:   os.Getenv("NETBOX_URL"),
		NetBoxToken: os.Getenv("NETBOX_TOKEN"),
		Site:        os.Getenv("NETBOX_SITE"),
		Role:        os.Getenv("NETBOX_ROLE"),
		Mode:        strings.ToLower(getEnv("SYSWATCH_MODE", "standalone")),
		APIUrl:      os.Getenv("SYSWATCH_API_URL"),
		APIKey:      os.Getenv("SYSWATCH_API_KEY"),
	}

	return cfg, cfg.validate()
}

// IsAPIMode returns true when running in Central Intelligence API mode.
func (c *Config) IsAPIMode() bool {
	return c.Mode == "api"
}

func (c *Config) validate() error {
	var missing []string

	if c.NetBoxURL == "" {
		missing = append(missing, "NETBOX_URL")
	}
	if c.NetBoxToken == "" {
		missing = append(missing, "NETBOX_TOKEN")
	}
	if c.Site == "" {
		missing = append(missing, "NETBOX_SITE")
	}
	if c.Role == "" {
		missing = append(missing, "NETBOX_ROLE")
	}

	// API mode requires URL and key
	if c.Mode == "api" {
		if c.APIUrl == "" {
			missing = append(missing, "SYSWATCH_API_URL")
		}
		if c.APIKey == "" {
			missing = append(missing, "SYSWATCH_API_KEY")
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required config: %s", strings.Join(missing, ", "))
	}

	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
