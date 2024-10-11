package nats

import (
	"context"
	"log"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/menderartifactsconsumer/internal/artifact"
	"github.com/menderartifactsconsumer/internal/config"
	"github.com/nats-io/nats.go/jetstream"
)

func handleRequest(js jetstream.JetStream, msg jetstream.Msg, azureServiceClient *azblob.Client, cfg *config.Config) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	log.Print("Received message with subject " + msg.Subject())
	switch msg.Subject() {
	case "artifact.uploadArtifact.>":
		{
			_, err := artifact.UploadArtifact(ctx, js, msg, azureServiceClient, cfg)
			if err != nil {
				log.Printf("Failed to upload Artifact: %v", err)
				msg.Ack()
				return
			}
			return

		}

	case "artifact.GenerateSASToken.>":
		{
			log.Print(msg.Data())
			_, err := artifact.GenerateNewSASToken(ctx, js, msg, azureServiceClient, cfg)
			if err != nil {
				log.Printf("Failed to upload Artifact: %v", err)
				msg.Ack()
				return
			}
			return
		}

	}

}

