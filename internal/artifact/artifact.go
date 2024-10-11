package artifact

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	storageClient "github.com/menderartifactsconsumer/internal/azblob"
	"github.com/menderartifactsconsumer/internal/config"
	"github.com/menderartifactsconsumer/internal/http"
	nats "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type Request struct {
	RequestId string `json:"requestId"`
	Token     string `json:"token"`
	Domain    string `json:"domain"`
}

type Artifact struct {
	ContainerName string `json:"containerName"`
	BlobName      string `json:"blobName"`
}

type UploadArtifactRequest struct {
	AuthRequest  Request  `json:"request_data"`
	BlobMetadata Artifact `json:"Artifact"`
}

type GenerateSASTokenRequest struct {
	RequestId     string `json:"requestId"`
	ContainerName string `json:"containerName"`
	BlobName      string `json:"blobName"`
}

type UploadArtifactConsumerResponse struct {
	RequestId    string `json:"requestId"`
	UploadStatus string `json:"uploadStatus"`
}

type UploadArtifactTargetApplicationResponse struct {
	RequestId    string `json:"requestId"`
	UploadStatus string `json:"uploadStatus"`
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

func ParseGenerateSASRequest(msg jetstream.Msg) (*GenerateSASTokenRequest, error) {
	var request GenerateSASTokenRequest
	err := json.Unmarshal(msg.Data(), &request)

	if err != nil {
		log.Printf("Failed to parse generatesas request: %v", err)
	}

	return &request, nil
}

func UploadArtifact(ctx context.Context, js jetstream.JetStream, msg jetstream.Msg, serviceClient *azblob.Client, cfg *config.Config) (string, error) {
	log.Print(string(msg.Data()))
	request, err := ParseUploadArtifactRequest(msg)
	if err != nil {
		log.Printf("Failed to parse credentials: %v", err)
		return "", err
	}

	uploadArtifactConsumerResponse := UploadArtifactConsumerResponse{
		RequestId:    request.AuthRequest.RequestId,
		UploadStatus: "In Progress",
	}

	consumerResponseJson, _ := json.Marshal(uploadArtifactConsumerResponse)
	responseMsg := nats.NewMsg("artifact.uploadArtifactResponse." + request.AuthRequest.RequestId)
	responseMsg.Header.Set("StatusCode", "200")
	responseMsg.Data = append(responseMsg.Data, consumerResponseJson...)
	fmt.Println(responseMsg.Header.Get("StatusCode"))
	_, err = js.PublishMsgAsync(responseMsg)
	log.Print(err)
	if err != nil {
		log.Printf("Failed to publish : %v", err)
	}

	log.Print(request.AuthRequest.Domain)
	log.Printf("Received Request: %s", request.AuthRequest.RequestId)
	token := request.AuthRequest.Token
	msg.Ack()
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

	uploadArtifactTargetApplicationResponse := UploadArtifactTargetApplicationResponse{
		RequestId:    request.AuthRequest.RequestId,
		UploadStatus: "Finished",
	}

	targetApplicationResponseJson, _ := json.Marshal(uploadArtifactTargetApplicationResponse)
	targetApplicationresponseMsg := nats.NewMsg("artifact.uploadArtifactTargetApplicationResponse." + request.AuthRequest.RequestId)
	targetApplicationresponseMsg.Header.Set("StatusCode", string(resp.StatusCode))
	targetApplicationresponseMsg.Data = append(responseMsg.Data, targetApplicationResponseJson...)

	fmt.Println(responseMsg.Header.Get("StatusCode"))
	_, err = js.PublishMsgAsync(responseMsg)
	log.Print(err)
	if err != nil {
		log.Printf("Failed to publish : %v", err)
	}

	return "Blob uploaded successfully", nil
}

// func UploadArtifact(ctx context.Context, js jetstream.JetStream, msg jetstream.Msg, serviceClient *azblob.Client, cfg *config.Config) (string, error) {
// 	log.Print(string(msg.Data()))
// 	request, err := ParseUploadArtifactRequest(msg)
// 	if err != nil {
// 		log.Printf("Failed to parse credentials: %v", err)
// 		return "", err
// 	}

// 	uploadArtifactConsumerResponse := UploadArtifactConsumerResponse{
// 		RequestId:    request.AuthRequest.RequestId,
// 		UploadStatus: "In Progress",
// 	}
// 	consumerResponseJson, _ := json.Marshal(uploadArtifactConsumerResponse)
// 	responseMsg := nats.NewMsg("artifact.uploadArtifactResponse." + request.AuthRequest.RequestId)
// 	responseMsg.Header.Set("StatusCode", "200")
// 	responseMsg.Data = append(responseMsg.Data, consumerResponseJson...)
// 	fmt.Println(responseMsg.Header.Get("StatusCode"))
// 	_, err = js.PublishMsgAsync(responseMsg)
// 	if err != nil {
// 		log.Printf("Failed to publish initial response: %v", err)
// 	}

// 	msg.Ack()

// 	// Prepare the token and URL
// 	token := request.AuthRequest.Token
// 	apiURL := "https://" + request.AuthRequest.Domain + "/api/management/v1/deployments/artifacts"

// 	// Download the blob from Azure in chunks
// 	downloadResponse, err := serviceClient.DownloadStream(ctx, request.BlobMetadata.ContainerName, request.BlobMetadata.BlobName, nil)
// 	if err != nil {
// 		log.Printf("Failed to start blob download: %v", err)
// 		return "", err
// 	}
// 	defer downloadResponse.Body.Close()

// 	// Pipe to stream data without blocking the main thread
// 	reader, writer := io.Pipe()

// 	// Multipart writer to write data in chunks
// 	multipartWriter := multipart.NewWriter(writer)

// 	go func() {
// 		defer writer.Close()

// 		// Write metadata field
// 		metaPart, err := multipartWriter.CreateFormField("description")
// 		if err != nil {
// 			log.Printf("Failed to create metadata part: %v", err)
// 			writer.CloseWithError(err)
// 			return
// 		}
// 		_, err = metaPart.Write([]byte("Artifact description"))
// 		if err != nil {
// 			log.Printf("Failed to write to metadata part: %v", err)
// 			writer.CloseWithError(err)
// 			return
// 		}

// 		// Create artifact part
// 		artifactPart, err := multipartWriter.CreateFormFile("artifact", request.BlobMetadata.BlobName)
// 		if err != nil {
// 			log.Printf("Failed to create form file for artifact: %v", err)
// 			writer.CloseWithError(err)
// 			return
// 		}

// 		// Define chunk size
// 		const chunkSize = 5 * 1024 * 1024 // 5 MB
// 		buffer := make([]byte, chunkSize)

// 		// Read the blob in chunks and stream it to the multipart writer
// 		for {
// 			n, err := downloadResponse.Body.Read(buffer)
// 			if n > 0 {
// 				_, err := artifactPart.Write(buffer[:n])
// 				if err != nil {
// 					log.Printf("Failed to write chunk to multipart writer: %v", err)
// 					writer.CloseWithError(err)
// 					return
// 				}
// 			}
// 			if err != nil {
// 				if err == io.EOF {
// 					break
// 				}
// 				log.Printf("Failed to read blob: %v", err)
// 				writer.CloseWithError(err)
// 				return
// 			}
// 		}

// 		// Close the multipart writer once all chunks are written
// 		if err := multipartWriter.Close(); err != nil {
// 			log.Printf("Failed to close multipart writer: %v", err)
// 			writer.CloseWithError(err)
// 			return
// 		}
// 	}()

// 	// Prepare the HTTP request
// 	client := http.NewClient()
// 	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, reader)
// 	if err != nil {
// 		log.Printf("Failed to create request: %v", err)
// 		return "", err
// 	}

// 	// Set headers
// 	req.Header.Set("Content-Type", multipartWriter.FormDataContentType())
// 	req.Header.Set("Authorization", "Bearer "+token)

// 	// Send the request and process the response
// 	log.Print("Sending request")
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		log.Printf("Failed to send request: %v", err)
// 		return "", err
// 	}
// 	defer resp.Body.Close()

// 	// Log response body and status code for debugging
// 	responseBody, _ := io.ReadAll(resp.Body)
// 	log.Print(string(responseBody))
// 	log.Printf("StatusCode: %v", resp.StatusCode)

// 	uploadArtifactTargetApplicationResponse := UploadArtifactTargetApplicationResponse{
// 		RequestId:    request.AuthRequest.RequestId,
// 		UploadStatus: "Finished",
// 	}
// 	targetApplicationResponseJson, _ := json.Marshal(uploadArtifactTargetApplicationResponse)
// 	targetApplicationresponseMsg := nats.NewMsg("artifact.uploadArtifactTargetApplicationResponse." + request.AuthRequest.RequestId)
// 	targetApplicationresponseMsg.Header.Set("StatusCode", string(resp.StatusCode))
// 	targetApplicationresponseMsg.Data = append(targetApplicationresponseMsg.Data, targetApplicationResponseJson...)

// 	fmt.Println(targetApplicationresponseMsg.Header.Get("StatusCode"))
// 	_, err = js.PublishMsgAsync(targetApplicationresponseMsg)
// 	if err != nil {
// 		log.Printf("Failed to publish final response: %v", err)
// 	}

// 	return "Blob uploaded successfully", nil
// }

func GenerateNewSASToken(ctx context.Context, js jetstream.JetStream, msg jetstream.Msg, serviceClient *azblob.Client, cfg *config.Config) (string, error) {
	request, err := ParseGenerateSASRequest(msg)

	if err != nil {
		log.Printf("Failed to parse request GenerateNewSasToken")
		return "", err
	}

	log.Print(request.ContainerName)
	token, err := storageClient.CreateSASToken(cfg, request.ContainerName, request.BlobName)

	if err != nil {
		log.Printf("Failed to create sas token : %v", err)
		return "", err
	}

	tokenJson, _ := json.Marshal(*token)

	responseMsg := nats.NewMsg("artifact.createSASTokenResponse." + request.RequestId)
	responseMsg.Header.Set("StatusCode", "200")
	responseMsg.Data = append(responseMsg.Data, tokenJson...)
	fmt.Println(responseMsg.Header.Get("StatusCode"))
	_, err = js.PublishMsgAsync(responseMsg)
	log.Print(err)
	if err != nil {
		log.Printf("Failed to publish : %v", err)
	}

	return token.SASToken, nil
}
