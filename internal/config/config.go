package config

import (
	"log"
)

type Config struct {
	NATSURL         string `JSON:"NATS_URL"`
	NATSCredentials string `JSON:"NATS_CREDENTIALS"`
	BlobStorageUrl  string `JSON:"BLOB_STORAGE_URL"`
}

func Load() (*Config, error) {
	var cfg Config
	cfg.NATSURL = "nats://20.52.109.144:4222"
	// cfg.NATSCredentials = "NGS-Karthick-karthick.creds"
	cfg.BlobStorageUrl = "https://mendertemporarystorage.blob.core.windows.net/"

	if cfg.NATSURL == "" {
		log.Fatal("Critical configuration is missing")
	}

	return &cfg, nil
}
