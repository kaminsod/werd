package poller

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/werd-platform/werd/src/go/monitor-reddit/internal/reddit"
	"github.com/werd-platform/werd/src/go/monitor-reddit/internal/webhook"
)

type Poller struct {
	client       *reddit.Client
	sender       *webhook.Sender
	projectID    string
	subreddits   []string
	pollInterval time.Duration
	lastSeen     map[string]string // subreddit -> last seen fullname
	initialized  bool
}

func New(client *reddit.Client, sender *webhook.Sender, projectID string, subreddits []string, pollInterval time.Duration) *Poller {
	return &Poller{
		client:       client,
		sender:       sender,
		projectID:    projectID,
		subreddits:   subreddits,
		pollInterval: pollInterval,
		lastSeen:     make(map[string]string),
	}
}

func (p *Poller) Run(ctx context.Context) error {
	log.Printf("polling %d subreddits every %s", len(p.subreddits), p.pollInterval)

	// Initial poll to set watermarks.
	p.poll(ctx)
	p.initialized = true

	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			p.poll(ctx)
		}
	}
}

func (p *Poller) poll(ctx context.Context) {
	for _, sub := range p.subreddits {
		posts, err := p.client.ListNewPosts(ctx, sub, 25)
		if err != nil {
			log.Printf("error polling r/%s: %v", sub, err)
			continue
		}

		if len(posts) == 0 {
			continue
		}

		lastSeen := p.lastSeen[sub]
		var newPosts []reddit.Post

		for _, post := range posts {
			if post.Fullname == lastSeen {
				break
			}
			newPosts = append(newPosts, post)
		}

		// Update watermark to the newest post.
		p.lastSeen[sub] = posts[0].Fullname

		if !p.initialized {
			log.Printf("r/%s: initialized watermark at %s (%d posts)", sub, posts[0].Fullname, len(posts))
			continue
		}

		for _, post := range newPosts {
			content := post.Selftext
			if len(content) > 2000 {
				content = content[:2000] + "..."
			}

			err := p.sender.Send(ctx, webhook.IngestPayload{
				ProjectID:  p.projectID,
				SourceType: "reddit",
				SourceID:   post.Fullname,
				Title:      post.Title,
				Content:    content,
				URL:        fmt.Sprintf("https://reddit.com%s", post.Permalink),
				Severity:   "low",
			})
			if err != nil {
				log.Printf("error sending alert for r/%s post %s: %v", sub, post.Fullname, err)
			} else {
				log.Printf("r/%s: sent alert for %s: %s", sub, post.Fullname, post.Title)
			}
		}
	}
}
