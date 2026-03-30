package config

import (
	"errors"
	"os"
	"strings"
	"time"
)

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	Port             string
	DataDir          string
	JWTSecret        string
	OpenRouterAPIKey string
	AccessTokenTTL   time.Duration
	RefreshTokenTTL  time.Duration
	BcryptCost       int
	LogLevel         string
}

// Load reads configuration from environment variables, applies defaults,
// and returns an error if any required fields are missing.
func Load() (*Config, error) {
	jwtSecret := os.Getenv("JWT_SECRET")
	if strings.TrimSpace(jwtSecret) == "" {
		return nil, errors.New("JWT_SECRET environment variable is required")
	}

	cfg := &Config{
		JWTSecret:        jwtSecret,
		OpenRouterAPIKey: os.Getenv("OPENROUTER_API_KEY"),
		AccessTokenTTL:   15 * time.Minute,
		RefreshTokenTTL:  7 * 24 * time.Hour,
		BcryptCost:       12,
	}

	if v := os.Getenv("PORT"); v != "" {
		cfg.Port = v
	} else {
		cfg.Port = "8080"
	}

	if v := os.Getenv("DATA_DIR"); v != "" {
		cfg.DataDir = v
	} else {
		cfg.DataDir = "/data"
	}

	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	} else {
		cfg.LogLevel = "info"
	}

	return cfg, nil
}
