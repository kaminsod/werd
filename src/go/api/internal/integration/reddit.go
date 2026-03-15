package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultRedditAuthHost = "https://www.reddit.com"
	defaultRedditAPIHost  = "https://oauth.reddit.com"
)

type RedditCredentials struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	UserAgent    string `json:"user_agent"`
	Subreddit    string `json:"subreddit"`
}

type redditTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// Reddit implements PlatformAdapter for Reddit cross-posting.
type Reddit struct {
	authHost string
	apiHost  string
	httpCli  *http.Client
}

// NewReddit creates a Reddit adapter with default hosts.
// Use NewRedditWithHosts for testing with mock servers.
func NewReddit() *Reddit {
	return &Reddit{
		authHost: defaultRedditAuthHost,
		apiHost:  defaultRedditAPIHost,
		httpCli:  &http.Client{Timeout: 15 * time.Second},
	}
}

// NewRedditWithHosts creates a Reddit adapter with custom hosts (for testing).
func NewRedditWithHosts(authHost, apiHost string) *Reddit {
	r := NewReddit()
	if authHost != "" {
		r.authHost = authHost
	}
	if apiHost != "" {
		r.apiHost = apiHost
	}
	return r
}

func (r *Reddit) ValidateCredentials(ctx context.Context, credentials json.RawMessage) error {
	creds, err := r.parseCreds(credentials)
	if err != nil {
		return err
	}
	_, err = r.getAccessToken(ctx, creds)
	return err
}

// Publish creates a post on Reddit. Supports text posts (title + body)
// and link posts (title + URL).
func (r *Reddit) Publish(ctx context.Context, content PublishContent, credentials json.RawMessage) (*PublishResult, error) {
	creds, err := r.parseCreds(credentials)
	if err != nil {
		return nil, err
	}

	token, err := r.getAccessToken(ctx, creds)
	if err != nil {
		return nil, fmt.Errorf("reddit: getting access token: %w", err)
	}

	// Use structured fields if title is set, otherwise fall back to splitting body.
	title := content.Title
	body := content.Body
	if title == "" && body != "" {
		title, body = splitTitleBody(body)
	}

	var name, postURL string
	if content.PostType == "link" && content.URL != "" {
		name, postURL, err = r.submitLinkPost(ctx, token, creds, title, content.URL)
	} else {
		name, postURL, err = r.submitPost(ctx, token, creds, title, body)
	}
	if err != nil {
		return nil, fmt.Errorf("reddit: submitting post: %w", err)
	}

	return &PublishResult{
		PlatformPostID: name,
		URL:            postURL,
	}, nil
}

func (r *Reddit) parseCreds(raw json.RawMessage) (*RedditCredentials, error) {
	var creds RedditCredentials
	if err := json.Unmarshal(raw, &creds); err != nil {
		return nil, fmt.Errorf("reddit: invalid credentials JSON: %w", err)
	}
	if creds.ClientID == "" {
		return nil, fmt.Errorf("reddit: client_id is required")
	}
	if creds.ClientSecret == "" {
		return nil, fmt.Errorf("reddit: client_secret is required")
	}
	if creds.Username == "" {
		return nil, fmt.Errorf("reddit: username is required")
	}
	if creds.Password == "" {
		return nil, fmt.Errorf("reddit: password is required")
	}
	if creds.UserAgent == "" {
		return nil, fmt.Errorf("reddit: user_agent is required")
	}
	if creds.Subreddit == "" {
		return nil, fmt.Errorf("reddit: subreddit is required")
	}
	return &creds, nil
}

func (r *Reddit) getAccessToken(ctx context.Context, creds *RedditCredentials) (string, error) {
	form := url.Values{
		"grant_type": {"password"},
		"username":   {creds.Username},
		"password":   {creds.Password},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		r.authHost+"/api/v1/access_token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(creds.ClientID, creds.ClientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", creds.UserAgent)

	resp, err := r.httpCli.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token request failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var tokenResp redditTokenResponse
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return "", fmt.Errorf("parsing token response: %w", err)
	}
	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("empty access token in response: %s", string(respBody))
	}

	return tokenResp.AccessToken, nil
}

func (r *Reddit) submitPost(ctx context.Context, token string, creds *RedditCredentials, title, body string) (string, string, error) {
	form := url.Values{
		"api_type": {"json"},
		"kind":     {"self"},
		"sr":       {creds.Subreddit},
		"title":    {title},
		"text":     {body},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		r.apiHost+"/api/submit", strings.NewReader(form.Encode()))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", creds.UserAgent)

	resp, err := r.httpCli.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("submit request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("reading submit response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("submit failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	// Reddit wraps the response: {"json": {"errors": [], "data": {"name": "t3_...", "url": "..."}}}
	var result struct {
		JSON struct {
			Errors [][]string `json:"errors"`
			Data   struct {
				Name string `json:"name"`
				URL  string `json:"url"`
			} `json:"data"`
		} `json:"json"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", "", fmt.Errorf("parsing submit response: %w", err)
	}

	if len(result.JSON.Errors) > 0 {
		return "", "", fmt.Errorf("reddit API errors: %v", result.JSON.Errors)
	}

	return result.JSON.Data.Name, result.JSON.Data.URL, nil
}

func (r *Reddit) submitLinkPost(ctx context.Context, token string, creds *RedditCredentials, title, linkURL string) (string, string, error) {
	form := url.Values{
		"api_type": {"json"},
		"kind":     {"link"},
		"sr":       {creds.Subreddit},
		"title":    {title},
		"url":      {linkURL},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		r.apiHost+"/api/submit", strings.NewReader(form.Encode()))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", creds.UserAgent)

	resp, err := r.httpCli.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("link submit failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("reading submit response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("link submit failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		JSON struct {
			Errors [][]string `json:"errors"`
			Data   struct {
				Name string `json:"name"`
				URL  string `json:"url"`
			} `json:"data"`
		} `json:"json"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", "", fmt.Errorf("parsing submit response: %w", err)
	}
	if len(result.JSON.Errors) > 0 {
		return "", "", fmt.Errorf("reddit API errors: %v", result.JSON.Errors)
	}

	return result.JSON.Data.Name, result.JSON.Data.URL, nil
}

// splitTitleBody splits content into title (first line) and body (remaining lines).
func splitTitleBody(content string) (string, string) {
	parts := strings.SplitN(content, "\n", 2)
	title := strings.TrimSpace(parts[0])
	body := ""
	if len(parts) > 1 {
		body = strings.TrimSpace(parts[1])
	}
	if title == "" {
		title = "Post from Werd"
	}
	return title, body
}
