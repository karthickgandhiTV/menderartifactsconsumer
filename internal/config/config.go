package config

import (
	"log"
)

type Config struct {
	NATSURL         string `JSON:"NATS_URL"`
	NATSCredentials string `JSON:"NATS_CREDENTIALS"`
	// APIBaseURL      string `JSON:"API_BASE_URL"`
}

func Load() (*Config, error) {
	var cfg Config
	cfg.NATSURL = "nats://localhost:4222"
	// cfg.NATSCredentials = "NGS-Karthick-karthick.creds"

	if cfg.NATSURL == "" {
		log.Fatal("Critical configuration is missing")
	}

	return &cfg, nil
}
