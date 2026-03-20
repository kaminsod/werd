package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
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

func (b *Bluesky) Publish(ctx context.Context, content PublishContent, credentials json.RawMessage) (*PublishResult, error) {
	creds, err := b.parseCreds(credentials)
	if err != nil {
		return nil, err
	}

	session, err := b.createSession(ctx, creds)
	if err != nil {
		return nil, fmt.Errorf("bluesky: creating session: %w", err)
	}

	// Reply mode: reply to an existing Bluesky post.
	if content.ReplyToURL != "" {
		handle, rkey, err := parseBlueskyURL(content.ReplyToURL)
		if err != nil {
			return nil, fmt.Errorf("bluesky: %w", err)
		}
		parentURI, parentCID, rootURI, rootCID, err := b.resolveBlueskyPost(ctx, session, handle, rkey)
		if err != nil {
			return nil, fmt.Errorf("bluesky: resolving parent post: %w", err)
		}
		record, err := b.createReply(ctx, session, content.Body, parentURI, parentCID, rootURI, rootCID)
		if err != nil {
			return nil, fmt.Errorf("bluesky: creating reply: %w", err)
		}
		return &PublishResult{
			PlatformPostID: record.URI,
			URL:            b.atURIToWebURL(record.URI, session.Handle),
		}, nil
	}

	// Bluesky uses body as post text. Append URL if it's a link post.
	text := content.Body
	if content.URL != "" {
		if text != "" {
			text += "\n"
		}
		text += content.URL
	}

	record, err := b.createPost(ctx, session, text)
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

// parseBlueskyURL extracts handle and rkey from a bsky.app URL.
func parseBlueskyURL(rawURL string) (handle, rkey string, err error) {
	// Expected format: https://bsky.app/profile/{handle}/post/{rkey}
	rawURL = strings.TrimRight(rawURL, "/")
	parts := strings.Split(rawURL, "/")
	// Find "profile" and "post" segments.
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == "profile" && i+3 < len(parts) && parts[i+2] == "post" {
			return parts[i+1], parts[i+3], nil
		}
	}
	return "", "", fmt.Errorf("invalid Bluesky URL: %s", rawURL)
}

// resolveBlueskyPost resolves a handle + rkey to the AT URI/CID of the post and its root.
func (b *Bluesky) resolveBlueskyPost(ctx context.Context, session *blueskySession, handle, rkey string) (parentURI, parentCID, rootURI, rootCID string, err error) {
	// Resolve handle to DID.
	did, err := b.resolveHandle(ctx, handle)
	if err != nil {
		return "", "", "", "", fmt.Errorf("resolving handle %s: %w", handle, err)
	}

	// Fetch the post record to get its CID.
	parentURI = fmt.Sprintf("at://%s/app.bsky.feed.post/%s", did, rkey)
	thread, err := b.getPostThread(ctx, session, parentURI)
	if err != nil {
		return "", "", "", "", fmt.Errorf("fetching thread: %w", err)
	}

	parentCID = thread.Post.CID

	// Walk to root: if the post itself is a reply, its reply.root is the root.
	// Otherwise, this post IS the root.
	if thread.Post.Record.Reply != nil {
		rootURI = thread.Post.Record.Reply.Root.URI
		rootCID = thread.Post.Record.Reply.Root.CID
	} else {
		rootURI = parentURI
		rootCID = parentCID
	}
	return parentURI, parentCID, rootURI, rootCID, nil
}

func (b *Bluesky) resolveHandle(ctx context.Context, handle string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		b.host+"/xrpc/com.atproto.identity.resolveHandle?handle="+handle, nil)
	if err != nil {
		return "", err
	}
	resp, err := b.httpCli.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("resolve handle failed (status %d): %s", resp.StatusCode, string(body))
	}
	var result struct {
		DID string `json:"did"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	return result.DID, nil
}

type blueskyThreadResponse struct {
	Post struct {
		URI    string `json:"uri"`
		CID    string `json:"cid"`
		Record struct {
			Reply *struct {
				Root struct {
					URI string `json:"uri"`
					CID string `json:"cid"`
				} `json:"root"`
				Parent struct {
					URI string `json:"uri"`
					CID string `json:"cid"`
				} `json:"parent"`
			} `json:"reply"`
		} `json:"record"`
	} `json:"post"`
}

type blueskyGetThreadResponse struct {
	Thread blueskyThreadResponse `json:"thread"`
}

func (b *Bluesky) getPostThread(ctx context.Context, session *blueskySession, uri string) (*blueskyThreadResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		b.host+"/xrpc/app.bsky.feed.getPostThread?uri="+url.QueryEscape(uri)+"&depth=0", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+session.AccessJwt)

	resp, err := b.httpCli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("getPostThread failed (status %d): %s", resp.StatusCode, string(body))
	}

	var result blueskyGetThreadResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return &result.Thread, nil
}

// createReply creates a reply post on Bluesky with proper root/parent threading.
func (b *Bluesky) createReply(ctx context.Context, session *blueskySession, text, parentURI, parentCID, rootURI, rootCID string) (*blueskyCreateRecordResponse, error) {
	record := map[string]any{
		"$type":     "app.bsky.feed.post",
		"text":      text,
		"createdAt": time.Now().UTC().Format(time.RFC3339Nano),
		"reply": map[string]any{
			"root": map[string]any{
				"uri": rootURI,
				"cid": rootCID,
			},
			"parent": map[string]any{
				"uri": parentURI,
				"cid": parentCID,
			},
		},
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
		return nil, fmt.Errorf("reply request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("reply failed (status %d): %s", resp.StatusCode, string(respBody))
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
