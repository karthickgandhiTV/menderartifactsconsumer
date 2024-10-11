package azblob

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/service"
	"github.com/google/uuid"
	"github.com/menderartifactsconsumer/internal/config"
)

type GenerateSASTokenResponse struct {
	ContainerName string `json:"containerName"`
	SASToken      string `json:"SASToken"`
}

func GetAzureBlobClient(cfg *config.Config) (*azblob.Client, error) {
	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Printf("Failed to create default credential: %v", err)
		return nil, err
	}

	serviceClient, err := azblob.NewClient(cfg.BlobStorageUrl, credential, nil)
	if err != nil {
		log.Printf("Failed to create service client: %v", err)
		return nil, err
	}

	return serviceClient, nil
}

func GetAzureBlobServiceClient(cfg *config.Config) (*service.Client, error) {
	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Printf("Failed to create default credential: %v", err)
		return nil, err
	}

	serviceClient, err := service.NewClient(cfg.BlobStorageUrl, credential, nil)
	if err != nil {
		log.Printf("Failed to create service client: %v", err)
		return nil, err
	}

	return serviceClient, nil
}

func CreateContainer(serviceClient *service.Client, ctx context.Context, containerName string) error {
	containerClient := serviceClient.NewContainerClient(containerName)
	_, err := containerClient.Create(ctx, &container.CreateOptions{})
	if err != nil {
		log.Printf("Failed to create container: %v", err)
		return err
	}

	return nil
}

func GenerateUniqueFileName() string {
	return fmt.Sprintf("artifact-%s.mender", uuid.New().String())
}

func GenerateSASStoken(cfg *config.Config, containerName string, blobName string, client *service.Client) (string, error) {
	containerClient := client.NewContainerClient(containerName)
	now := time.Now().UTC().Add(-10 * time.Second)
	expiry := now.Add(48 * time.Hour)
	info := service.KeyInfo{
		Start:  to.Ptr(now.UTC().Format(sas.TimeFormat)),
		Expiry: to.Ptr(expiry.UTC().Format(sas.TimeFormat)),
	}
	udc, err := client.GetUserDelegationCredential(context.Background(), info, nil)
	if err != nil {
		log.Printf("Failed to get user delegation credential: %s", err)
	}
	sasQueryParams, err := sas.BlobSignatureValues{
		Protocol:      sas.ProtocolHTTPS,
		BlobName:      blobName,
		ContainerName: containerName,
		StartTime:     time.Now().UTC().Add(-48 * time.Hour),
		ExpiryTime:    time.Now().UTC().Add(48 * time.Hour),
		Permissions: to.Ptr(sas.ContainerPermissions{
			Read:   false,
			Add:    false,
			Create: true,
			Write:  false,
		}).String(),
	}.SignWithUserDelegation(udc)

	if err != nil {
		log.Printf("Failed to generate SAS token: %v", err)
		return "", err
	}

	sasToken := sasQueryParams.Encode()

	return fmt.Sprintf("%s?%s", containerClient.URL(), sasToken), nil
}

func CreateSASToken(cfg *config.Config, containerNameReceived string) (*GenerateSASTokenResponse, error) {
	ctx := context.Background()
	serviceClient, err := GetAzureBlobServiceClient(cfg)
	if err != nil {
		log.Printf("Failed to get service client: %v", err)
		return nil, err
	}

	CreateContainer(serviceClient, ctx, containerNameReceived)
	sasToken, err := GenerateSASStoken(cfg, containerNameReceived, "raspberryimage", serviceClient)
	if err != nil {
		log.Printf("Failed to generate SAS token: %v", err)
		return nil, err
	}

	log.Print(sasToken)

	response := GenerateSASTokenResponse{
		ContainerName: containerNameReceived,
		SASToken:      sasToken,
	}

	return &response, nil
}
