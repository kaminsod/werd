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

const defaultBskyHost = "https://bsky.social"

type BlueskyCredentials struct {
	Identifier  string `json:"identifier"`
	AppPassword string `json:"app_password"`
}

type blueskySession struct {
	DID       string `json:"did"`
	Handle    string `json:"handle"`
	AccessJwt string `json:"accessJwt"`
}

type blueskyCreateRecordResponse struct {
	URI string `json:"uri"`
	CID string `json:"cid"`
}

// Bluesky implements PlatformAdapter for the AT Protocol.
type Bluesky struct {
	host    string
	httpCli *http.Client
}

func NewBluesky(host string) *Bluesky {
	if host == "" {
		host = defaultBskyHost
	}
	return &Bluesky{
		host:    host,
		httpCli: &http.Client{Timeout: 15 * time.Second},
	}
}

func (b *Bluesky) ValidateCredentials(ctx context.Context, credentials json.RawMessage) error {
	creds, err := b.parseCreds(credentials)
	if err != nil {
		return err
	}
	_, err = b.createSession(ctx, creds)
	return err
}

func (b *Bluesky) Publish(ctx context.Context, content string, credentials json.RawMessage) (*PublishResult, error) {
	creds, err := b.parseCreds(credentials)
	if err != nil {
		return nil, err
	}

	session, err := b.createSession(ctx, creds)
	if err != nil {
		return nil, fmt.Errorf("bluesky: creating session: %w", err)
	}

	record, err := b.createPost(ctx, session, content)
	if err != nil {
		return nil, fmt.Errorf("bluesky: creating post: %w", err)
	}

	return &PublishResult{
		PlatformPostID: record.URI,
		URL:            b.atURIToWebURL(record.URI, session.Handle),
	}, nil
}

func (b *Bluesky) parseCreds(raw json.RawMessage) (*BlueskyCredentials, error) {
	var creds BlueskyCredentials
	if err := json.Unmarshal(raw, &creds); err != nil {
		return nil, fmt.Errorf("bluesky: invalid credentials JSON: %w", err)
	}
	if creds.Identifier == "" {
		return nil, fmt.Errorf("bluesky: identifier is required")
	}
	if creds.AppPassword == "" {
		return nil, fmt.Errorf("bluesky: app_password is required")
	}
	return &creds, nil
}

func (b *Bluesky) createSession(ctx context.Context, creds *BlueskyCredentials) (*blueskySession, error) {
	payload := map[string]string{
		"identifier": creds.Identifier,
		"password":   creds.AppPassword,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		b.host+"/xrpc/com.atproto.server.createSession", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.httpCli.Do(req)
	if err != nil {
		return nil, fmt.Errorf("session request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("session failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var session blueskySession
	if err := json.Unmarshal(respBody, &session); err != nil {
		return nil, fmt.Errorf("parsing session response: %w", err)
	}
	return &session, nil
}

func (b *Bluesky) createPost(ctx context.Context, session *blueskySession, content string) (*blueskyCreateRecordResponse, error) {
	record := map[string]any{
		"$type":     "app.bsky.feed.post",
		"text":      content,
		"createdAt": time.Now().UTC().Format(time.RFC3339Nano),
	}
	payload := map[string]any{
		"repo":       session.DID,
		"collection": "app.bsky.feed.post",
		"record":     record,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		b.host+"/xrpc/com.atproto.repo.createRecord", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+session.AccessJwt)

	resp, err := b.httpCli.Do(req)
	if err != nil {
		return nil, fmt.Errorf("post request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("post failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result blueskyCreateRecordResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	return &result, nil
}

func (b *Bluesky) atURIToWebURL(atURI, handle string) string {
	// AT URI: at://did:plc:xyz/app.bsky.feed.post/rkey
	// Web URL: https://bsky.app/profile/handle/post/rkey
	lastSlash := len(atURI) - 1
	for lastSlash >= 0 && atURI[lastSlash] != '/' {
		lastSlash--
	}
	if lastSlash < 0 {
		return ""
	}
	rkey := atURI[lastSlash+1:]
	return fmt.Sprintf("https://bsky.app/profile/%s/post/%s", handle, rkey)
}
