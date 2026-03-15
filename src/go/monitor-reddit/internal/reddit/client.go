package reddit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type Post struct {
	ID        string  `json:"id"`
	Fullname  string  `json:"name"`
	Title     string  `json:"title"`
	Selftext  string  `json:"selftext"`
	Author    string  `json:"author"`
	Permalink string  `json:"permalink"`
	URL       string  `json:"url"`
	Created   float64 `json:"created_utc"`
	Subreddit string  `json:"subreddit"`
}

type Client struct {
	authHost     string
	apiHost      string
	clientID     string
	clientSecret string
	username     string
	password     string
	userAgent    string
	httpCli      *http.Client

	mu    sync.Mutex
	token string
	expAt time.Time
}

func NewClient(clientID, clientSecret, username, password, userAgent string) *Client {
	return &Client{
		authHost:     "https://www.reddit.com",
		apiHost:      "https://oauth.reddit.com",
		clientID:     clientID,
		clientSecret: clientSecret,
		username:     username,
		password:     password,
		userAgent:    userAgent,
		httpCli:      &http.Client{Timeout: 15 * time.Second},
	}
}

// NewClientWithHosts creates a client with custom hosts (for testing).
func NewClientWithHosts(clientID, clientSecret, username, password, userAgent, authHost, apiHost string) *Client {
	c := NewClient(clientID, clientSecret, username, password, userAgent)
	c.authHost = authHost
	c.apiHost = apiHost
	return c
}

func (c *Client) getToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.token != "" && time.Now().Before(c.expAt) {
		return c.token, nil
	}

	form := url.Values{
		"grant_type": {"password"},
		"username":   {c.username},
		"password":   {c.password},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.authHost+"/api/v1/access_token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(c.clientID, c.clientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpCli.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token request failed (%d): %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("parsing token: %w", err)
	}

	c.token = tokenResp.AccessToken
	c.expAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn-60) * time.Second)
	return c.token, nil
}

// ListNewPosts fetches the newest posts from a subreddit.
func (c *Client) ListNewPosts(ctx context.Context, subreddit string, limit int) ([]Post, error) {
	token, err := c.getToken(ctx)
	if err != nil {
		return nil, err
	}

	u := fmt.Sprintf("%s/r/%s/new.json?limit=%d&raw_json=1", c.apiHost, subreddit, limit)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpCli.Do(req)
	if err != nil {
		return nil, fmt.Errorf("listing posts: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("listing posts (%d): %s", resp.StatusCode, string(body))
	}

	var listing struct {
		Data struct {
			Children []struct {
				Data Post `json:"data"`
			} `json:"children"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &listing); err != nil {
		return nil, fmt.Errorf("parsing listing: %w", err)
	}

	posts := make([]Post, len(listing.Data.Children))
	for i, c := range listing.Data.Children {
		posts[i] = c.Data
	}
	return posts, nil
}
