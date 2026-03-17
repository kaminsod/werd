package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// LLMClassifyResult is the expected structured response from an LLM classify call.
type LLMClassifyResult struct {
	Relevant bool     `json:"relevant"`
	Severity string   `json:"severity"`
	Tags     []string `json:"tags"`
	Reason   string   `json:"reason"`
}

// LLMClient calls an OpenAI-compatible chat completions endpoint.
type LLMClient struct {
	apiURL  string
	apiKey  string
	model   string
	httpCli *http.Client
}

// NewLLMClient creates a new LLM client. Returns nil if apiURL is empty.
func NewLLMClient(apiURL, apiKey, model string) *LLMClient {
	if apiURL == "" {
		return nil
	}
	if model == "" {
		model = "gpt-4o-mini"
	}
	return &LLMClient{
		apiURL: apiURL,
		apiKey: apiKey,
		model:  model,
		httpCli: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Classify sends a prompt to the LLM and parses the structured JSON response.
func (c *LLMClient) Classify(ctx context.Context, prompt string, maxTokens int) (*LLMClassifyResult, error) {
	if maxTokens <= 0 {
		maxTokens = 200
	}

	reqBody := map[string]any{
		"model": c.model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens":  maxTokens,
		"temperature": 0.1,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling LLM request: %w", err)
	}

	endpoint := c.apiURL + "/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating LLM request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpCli.Do(req)
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading LLM response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("LLM returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse OpenAI-compatible response.
	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("parsing LLM response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("LLM returned no choices")
	}

	content := chatResp.Choices[0].Message.Content

	// Parse the LLM's JSON response.
	var result LLMClassifyResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("parsing LLM classification JSON: %w (content: %s)", err, content)
	}

	return &result, nil
}
