package config

import (
	"fmt"

	"github.com/caarlos0/env/v10"
	"github.com/joho/godotenv"
)

func Load() (*Config, error) {
	_ = godotenv.Load() // Ignore error if .env doesn't exist

	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("config.Load: parse env: %w", err)
	}

	return cfg, nil
}
