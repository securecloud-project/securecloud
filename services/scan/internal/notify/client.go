package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	client  *http.Client
	logger  *slog.Logger
}

func New(baseURL string, timeout time.Duration, logger *slog.Logger) *Client {
	return newClient(baseURL, &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			Proxy:                 nil,
			TLSHandshakeTimeout:   timeout,
			ResponseHeaderTimeout: timeout,
		},
	}, logger)
}

func newClient(baseURL string, httpClient *http.Client, logger *slog.Logger) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), client: httpClient, logger: logger}
}

func (c *Client) Notify(ctx context.Context, scanID, message string) error {
	if c.baseURL == "" {
		return nil
	}
	payload, err := json.Marshal(map[string]string{"scan_id": scanID, "message": message})
	if err != nil {
		return fmt.Errorf("encode notification: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/notifications", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create notification request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := c.client.Do(request)
	if err != nil {
		return fmt.Errorf("send notification: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("notification service returned %s", response.Status)
	}
	return nil
}
