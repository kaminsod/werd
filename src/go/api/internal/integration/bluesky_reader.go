package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// BlueskyReader implements PlatformReader for Bluesky.
type BlueskyReader struct {
	host    string
	httpCli *http.Client
}

func NewBlueskyReader() *BlueskyReader {
	return &BlueskyReader{
		host:    defaultBskyHost,
		httpCli: &http.Client{Timeout: 15 * time.Second},
	}
}

func (r *BlueskyReader) GetReplies(ctx context.Context, platformPostID, sinceID string, credentials json.RawMessage) ([]PlatformReply, error) {
	var creds BlueskyCredentials
	if err := json.Unmarshal(credentials, &creds); err != nil {
		return nil, fmt.Errorf("bluesky reader: invalid credentials: %w", err)
	}

	// Create a session to authenticate.
	bsky := &Bluesky{host: r.host, httpCli: r.httpCli}
	session, err := bsky.createSession(ctx, &creds)
	if err != nil {
		return nil, fmt.Errorf("bluesky reader: auth failed: %w", err)
	}

	// Fetch the post thread.
	params := url.Values{"uri": {platformPostID}, "depth": {"10"}}
	reqURL := fmt.Sprintf("%s/xrpc/app.bsky.feed.getPostThread?%s", r.host, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+session.AccessJwt)

	resp, err := r.httpCli.Do(req)
	if err != nil {
		return nil, fmt.Errorf("bluesky reader: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bluesky reader: status %d: %s", resp.StatusCode, string(body))
	}

	// Parse thread response.
	var thread struct {
		Thread struct {
			Replies []struct {
				Post struct {
					URI    string `json:"uri"`
					Author struct {
						Handle string `json:"handle"`
					} `json:"author"`
					Record struct {
						Text      string `json:"text"`
						CreatedAt string `json:"createdAt"`
					} `json:"record"`
				} `json:"post"`
			} `json:"replies"`
		} `json:"thread"`
	}
	if err := json.Unmarshal(body, &thread); err != nil {
		return nil, fmt.Errorf("bluesky reader: parsing response: %w", err)
	}

	var replies []PlatformReply
	for _, r := range thread.Thread.Replies {
		replyURI := r.Post.URI
		if sinceID != "" && replyURI == sinceID {
			break // already seen this and everything before it
		}

		createdAt, _ := time.Parse(time.RFC3339Nano, r.Post.Record.CreatedAt)

		// Derive web URL from AT URI.
		bskyInst := &Bluesky{}
		webURL := bskyInst.atURIToWebURL(replyURI, r.Post.Author.Handle)

		replies = append(replies, PlatformReply{
			ID:        replyURI,
			Author:    r.Post.Author.Handle,
			Content:   r.Post.Record.Text,
			URL:       webURL,
			CreatedAt: createdAt,
			ParentID:  platformPostID,
		})
	}

	return replies, nil
}
