package device

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/menderartifactsconsumer/internal/http"
	"github.com/nats-io/nats.go/jetstream"
)


type Request struct {
	RequestId string `json:"requestId"`
	Token     string `json:"token"`
	Domain    string `json:"domain"`
}


type Artifact struct {
	ContainerName string
	BlobName      string
}

type UploadArtifactRequest struct {
	AuthRequest  Request
	BlobMetadata Artifact
}

func ParseUploadArtifactRequest(msg jetstream.Msg) (*UploadArtifactRequest, error) {
	var request UploadArtifactRequest
	err := json.Unmarshal(msg.Data(), &request)
	if err != nil {
		log.Printf("Failed to parse credentials: %v", err)
		return nil, err
	}
	return &request, nil
}

func UploadArtifact(ctx context.Context, js jetstream.JetStream, msg jetstream.Msg) (string, error) {
	request, err := ParseUploadArtifactRequest(msg)
	if err != nil {
		log.Printf("Failed to parse credentials: %v", err)
		return "", err
	}

	log.Print(request.AuthRequest.Domain)
	log.Printf("Received Request: %s", request.AuthRequest.RequestId)
	token := request.AuthRequest.Token

	url := "https://menderartifactstorage.blob.core.windows.net/"

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Printf("Failed to crate default Credential: %v", err)
		return "", err
	}

	serviceClient, err := azblob.NewClient(url, credential, nil)
	if err != nil {
		log.Printf("Failed to create service client: %v", err)
		return "", err
	}

	downloadResponse, err := serviceClient.DownloadStream(ctx, request.BlobMetadata.ContainerName, request.BlobMetadata.BlobName, nil)
	if err != nil {
		log.Printf("Failed to start blob download: %v", err)
		return "", err
	}

	reader, writer := io.Pipe()
	multipartWriter := multipart.NewWriter(writer)

	go func() {
		defer writer.Close()
		defer downloadResponse.Body.Close()

		metaPart, err := multipartWriter.CreateFormField("description")
		if err != nil {
			log.Printf("Failed to create metadata part: %v", err)
			writer.CloseWithError(err)
			return
		}
		_, err = metaPart.Write([]byte("Artifact description"))
		if err != nil {
			log.Printf("Failed to write to metadata part: %v", err)
			writer.CloseWithError(err)
			return
		}

		artifactPart, err := multipartWriter.CreateFormFile("artifact", request.BlobMetadata.ContainerName)
		if err != nil {
			log.Printf("Failed to create form file for artifact: %v", err)
			writer.CloseWithError(err)
			return
		}

		if _, err = io.Copy(artifactPart, downloadResponse.Body); err != nil {
			log.Printf("Failed to copy blob data to form file: %v", err)
			writer.CloseWithError(err)
			return
		}

		if err := multipartWriter.Close(); err != nil {
			log.Printf("Failed to close multipart writer: %v", err)
			writer.CloseWithError(err)
			return
		}
	}()

	client := http.NewClient()
	apiURL := "https://" + request.AuthRequest.Domain + "/api/management/v1/deployments/artifacts"
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, reader)
	if err != nil {
		log.Printf("Failed to create request: %v", err)
		return "", err
	}

	req.Header.Set("Content-Type", multipartWriter.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	log.Print("Sending request")
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to send request: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	responseBody, _ := io.ReadAll(resp.Body)
	log.Print(string(responseBody))
	log.Printf("StatusCode: %v", resp.StatusCode)

	if resp.StatusCode != 201 {
		log.Printf("Failed to upload blob, server responded with status: %v", resp.StatusCode)
		return "", fmt.Errorf("upload failed with status: %d", resp.StatusCode)
	}

	return "Blob uploaded successfully", nil
}
