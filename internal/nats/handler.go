package nats

import (
	"context"
	"log"
	"time"

	"github.com/menderartifactsconsumer/internal/device"
	"github.com/nats-io/nats.go/jetstream"
)

func handleRequest(js jetstream.JetStream, msg jetstream.Msg) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	if msg.Subject() == "device.uploadArtifact.>" {
		_, err := device.UploadArtifact(ctx, js, msg)
		if err != nil {
			log.Printf("Failed to upload Artifact: %v", err)
			msg.Ack()
			return
		}
		return
	}
}
