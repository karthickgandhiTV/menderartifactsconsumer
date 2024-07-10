package http

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"time"
)

const (
	StatusOK = http.StatusOK
)

func NewClient() *http.Client {
	return &http.Client{
		Timeout: time.Minute * 10,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
}

func NewRequestWithContext(ctx context.Context, method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func MakeRequestWithJWT(ctx context.Context, method, url, token string, body io.Reader) (*http.Response, error) {
	client := NewClient()

	req, err := NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	return client.Do(req)
}
