package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// BrowserAdapter delegates platform operations to the browser automation service.
// A single struct handles all platforms — parameterized by platform name.
type BrowserAdapter struct {
	serviceURL     string
	platform       string
	internalAPIKey string
	httpCli        *http.Client
}

func NewBrowserAdapter(serviceURL, platform, internalAPIKey string) *BrowserAdapter {
	return &BrowserAdapter{
		serviceURL:     serviceURL,
		platform:       platform,
		internalAPIKey: internalAPIKey,
		httpCli:        &http.Client{Timeout: 60 * time.Second}, // browser ops are slow
	}
}

type browserPublishRequest struct {
	Platform    string         `json:"platform"`
	Credentials map[string]any `json:"credentials"`
	Content     string         `json:"content"`
	Options     map[string]any `json:"options"`
}

type browserPublishResponse struct {
	Success      bool   `json:"success"`
	PostID       string `json:"post_id"`
	URL          string `json:"url"`
	Error        string `json:"error"`
	ScreenshotB64 string `json:"screenshot_b64"`
}

type browserValidateRequest struct {
	Platform    string         `json:"platform"`
	Credentials map[string]any `json:"credentials"`
	Options     map[string]any `json:"options"`
}

type browserValidateResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

func (b *BrowserAdapter) ValidateCredentials(ctx context.Context, credentials json.RawMessage) error {
	var creds map[string]any
	if err := json.Unmarshal(credentials, &creds); err != nil {
		return fmt.Errorf("browser: invalid credentials JSON: %w", err)
	}

	req := browserValidateRequest{
		Platform:    b.platform,
		Credentials: creds,
		Options:     map[string]any{"headless": true, "timeout_secs": 30},
	}

	body, _ := json.Marshal(req)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		b.serviceURL+"/actions/validate", bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Internal-Key", b.internalAPIKey)

	resp, err := b.httpCli.Do(httpReq)
	if err != nil {
		return fmt.Errorf("browser service unreachable: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result browserValidateResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("browser: invalid response: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("browser validation failed: %s", result.Error)
	}
	return nil
}

func (b *BrowserAdapter) Publish(ctx context.Context, content PublishContent, credentials json.RawMessage) (*PublishResult, error) {
	var creds map[string]any
	if err := json.Unmarshal(credentials, &creds); err != nil {
		return nil, fmt.Errorf("browser: invalid credentials JSON: %w", err)
	}

	// Build content string for browser service — use structured fields if available.
	contentStr := content.Body
	if content.Title != "" {
		contentStr = content.Title + "\n" + content.Body
	}
	if content.PostType == "link" && content.URL != "" {
		contentStr = content.Title + "\n" + content.URL
	}

	req := browserPublishRequest{
		Platform:    b.platform,
		Credentials: creds,
		Content:     contentStr,
		Options:     map[string]any{"headless": true, "timeout_secs": 45, "screenshot_on_error": true},
	}

	body, _ := json.Marshal(req)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		b.serviceURL+"/actions/publish", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Internal-Key", b.internalAPIKey)

	resp, err := b.httpCli.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("browser service unreachable: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result browserPublishResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("browser: invalid response: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("browser publish failed: %s", result.Error)
	}

	return &PublishResult{
		PlatformPostID: result.PostID,
		URL:            result.URL,
	}, nil
}
