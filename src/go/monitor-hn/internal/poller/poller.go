package poller

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/werd-platform/werd/src/go/monitor-hn/internal/hn"
	"github.com/werd-platform/werd/src/go/monitor-hn/internal/webhook"
)

type Poller struct {
	client       *hn.Client
	sender       *webhook.Sender
	projectID    string
	keywords     []string
	pollInterval time.Duration
	maxSeenID    int
	initialized  bool
}

func New(client *hn.Client, sender *webhook.Sender, projectID string, keywords []string, pollInterval time.Duration) *Poller {
	return &Poller{
		client:       client,
		sender:       sender,
		projectID:    projectID,
		keywords:     keywords,
		pollInterval: pollInterval,
	}
}

func (p *Poller) Run(ctx context.Context) error {
	log.Printf("polling HN every %s (keywords: %v)", p.pollInterval, p.keywords)

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
	ids, err := p.client.GetNewStoryIDs(ctx)
	if err != nil {
		log.Printf("error fetching HN new stories: %v", err)
		return
	}

	if len(ids) == 0 {
		return
	}

	// Find new IDs (greater than watermark).
	var newIDs []int
	for _, id := range ids {
		if id > p.maxSeenID {
			newIDs = append(newIDs, id)
		}
	}

	// Update watermark.
	maxID := ids[0]
	for _, id := range ids {
		if id > maxID {
			maxID = id
		}
	}
	p.maxSeenID = maxID

	if !p.initialized {
		log.Printf("HN: initialized watermark at %d (%d stories)", p.maxSeenID, len(ids))
		return
	}

	if len(newIDs) == 0 {
		return
	}

	// Limit concurrent fetches.
	const maxConcurrent = 5
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	for _, id := range newIDs {
		sem <- struct{}{}
		wg.Add(1)
		go func(storyID int) {
			defer wg.Done()
			defer func() { <-sem }()

			item, err := p.client.GetItem(ctx, storyID)
			if err != nil {
				log.Printf("error fetching HN item %d: %v", storyID, err)
				return
			}
			if item == nil {
				return // deleted/dead item
			}
			if item.Type != "story" {
				return
			}

			// Optional keyword pre-filter.
			if len(p.keywords) > 0 && !p.matchesKeywords(item) {
				return
			}

			url := item.URL
			if url == "" {
				url = fmt.Sprintf("https://news.ycombinator.com/item?id=%d", item.ID)
			}

			content := item.Text
			if content == "" && item.URL != "" {
				content = item.URL
			}

			err = p.sender.Send(ctx, webhook.IngestPayload{
				ProjectID:  p.projectID,
				SourceType: "hn",
				SourceID:   fmt.Sprintf("hn_%d", item.ID),
				Title:      item.Title,
				Content:    content,
				URL:        fmt.Sprintf("https://news.ycombinator.com/item?id=%d", item.ID),
				Severity:   "low",
			})
			if err != nil {
				log.Printf("error sending HN alert %d: %v", item.ID, err)
			} else {
				log.Printf("HN: sent alert for %d: %s", item.ID, item.Title)
			}
		}(id)
	}

	wg.Wait()
}

func (p *Poller) matchesKeywords(item *hn.Item) bool {
	text := strings.ToLower(item.Title + " " + item.Text + " " + item.URL)
	for _, kw := range p.keywords {
		if strings.Contains(text, kw) {
			return true
		}
	}
	return false
}
