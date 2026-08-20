package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	client  *http.Client
}

func New(baseURL string, timeout time.Duration) *Client {
	return newClient(baseURL, &http.Client{
		Timeout:   timeout,
		Transport: &http.Transport{Proxy: nil, TLSHandshakeTimeout: timeout, ResponseHeaderTimeout: timeout},
	})
}

func newClient(baseURL string, httpClient *http.Client) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), client: httpClient}
}

func (c *Client) Verify(ctx context.Context, authorization string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/verify", nil)
	if err != nil {
		return fmt.Errorf("create auth verification request: %w", err)
	}
	request.Header.Set("Authorization", authorization)
	response, err := c.client.Do(request)
	if err != nil {
		return fmt.Errorf("verify authorization: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("authorization rejected")
	}
	return nil
}
