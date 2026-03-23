package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ChangedetectClient is an HTTP client for the changedetection.io REST API.
type ChangedetectClient struct {
	baseURL string
	apiKey  string
	httpCli *http.Client
}

func NewChangedetectClient(baseURL, apiKey string) *ChangedetectClient {
	return &ChangedetectClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		httpCli: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *ChangedetectClient) do(ctx context.Context, method, path string, body any) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("changedetect: marshal: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("changedetect: request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("x-api-key", c.apiKey)
	}

	resp, err := c.httpCli.Do(req)
	if err != nil {
		return nil, fmt.Errorf("changedetect: %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("changedetect: %s %s: status %d: %s", method, path, resp.StatusCode, string(data))
	}
	return data, nil
}

// CreateWatch creates a new watch and returns its UUID.
func (c *ChangedetectClient) CreateWatch(ctx context.Context, url, tag, title string) (string, error) {
	payload := map[string]any{
		"url":   url,
		"tag":   tag,
		"title": title,
	}
	data, err := c.do(ctx, http.MethodPost, "/api/v1/watch", payload)
	if err != nil {
		return "", err
	}

	var result map[string]string
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("changedetect: parse create response: %w", err)
	}
	if uuid, ok := result["uuid"]; ok {
		return uuid, nil
	}
	return "", fmt.Errorf("changedetect: no uuid in response: %s", string(data))
}

// DeleteWatch removes a watch by UUID.
func (c *ChangedetectClient) DeleteWatch(ctx context.Context, watchUUID string) error {
	_, err := c.do(ctx, http.MethodDelete, "/api/v1/watch/"+watchUUID, nil)
	return err
}

// GetWatchHistory returns a map of timestamp→endpoint for a watch's history.
func (c *ChangedetectClient) GetWatchHistory(ctx context.Context, watchUUID string) (map[string]string, error) {
	data, err := c.do(ctx, http.MethodGet, "/api/v1/watch/"+watchUUID+"/history", nil)
	if err != nil {
		return nil, err
	}

	var history map[string]string
	if err := json.Unmarshal(data, &history); err != nil {
		return nil, fmt.Errorf("changedetect: parse history: %w", err)
	}
	return history, nil
}

// GetSnapshot fetches the text content/diff for a specific history timestamp.
func (c *ChangedetectClient) GetSnapshot(ctx context.Context, watchUUID, timestamp string) (string, error) {
	data, err := c.do(ctx, http.MethodGet, "/api/v1/watch/"+watchUUID+"/history/"+timestamp, nil)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
