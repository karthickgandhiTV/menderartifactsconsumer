package main

import (
	"context"
	"log"
	"time"

	"github.com/menderartifactsconsumer/internal/config"
	"github.com/menderartifactsconsumer/internal/nats"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	nc, err := nats.Connect(cfg.NATSURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	js, err := nats.SetupJetStream(nc)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	nats.InitStreamAndConsumer(nc, ctx, js, cfg)

}
