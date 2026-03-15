package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type IngestPayload struct {
	ProjectID  string `json:"project_id"`
	SourceType string `json:"source_type"`
	SourceID   string `json:"source_id"`
	Title      string `json:"title"`
	Content    string `json:"content"`
	URL        string `json:"url"`
	Severity   string `json:"severity"`
}

type Sender struct {
	apiURL  string
	apiKey  string
	httpCli *http.Client
}

func NewSender(apiURL, apiKey string) *Sender {
	return &Sender{
		apiURL:  apiURL,
		apiKey:  apiKey,
		httpCli: &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *Sender) Send(ctx context.Context, payload IngestPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		s.apiURL+"/api/webhooks/ingest", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Secret", s.apiKey)

	resp, err := s.httpCli.Do(req)
	if err != nil {
		return fmt.Errorf("sending webhook: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	return nil
}
