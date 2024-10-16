package nats

import (
	"context"
	"log"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/menderartifactsconsumer/internal/config"
	nats "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func Connect(url string) (*nats.Conn, error) {
	nc, err := nats.Connect(url, nats.Name("Mender Producer"))
	if err != nil {
		return nil, err
	}
	return nc, nil
}

func SetupJetStream(nc *nats.Conn) (jetstream.JetStream, error) {
	js, err := jetstream.New(nc)
	if err != nil {
		return nil, err
	}
	return js, nil
}

func InitStreamAndConsumer(nc *nats.Conn, ctx context.Context, js jetstream.JetStream, azureServiceClient *azblob.Client, cfg *config.Config) {
	stream, err := js.Stream(ctx, "MenderUser")
	if err != nil {
		log.Fatal(err)
	}

	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Name:           "mender_artifact",
		Durable:        "mender_artifact",
		FilterSubjects: []string{"artifact.GenerateSASToken.>", "artifact.uploadArtifact.>"},
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Print("Waiting for messages..")
	cctx, err := consumer.Consume(func(msgs jetstream.Msg) {
		handleRequest(js, msgs, azureServiceClient, cfg)
		msgs.Ack()
	})
	if err != nil {
		log.Fatal(err)
	}
	defer cctx.Stop()
	select {}
}
