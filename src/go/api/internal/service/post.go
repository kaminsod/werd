package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/werd-platform/werd/src/go/api/internal/integration"
	"github.com/werd-platform/werd/src/go/api/internal/storage"
)

var (
	ErrPostNotFound  = errors.New("post not found")
	ErrPostNotDraft  = errors.New("only draft posts can be modified")
	ErrNoPlatforms   = errors.New("no platforms specified")
	ErrPublishFailed = errors.New("publish failed on one or more platforms")
)

type PostInfo struct {
	ID          string
	ProjectID   string
	Content     string
	Platforms   []string
	ScheduledAt *time.Time
	PublishedAt *time.Time
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type PostListResult struct {
	Posts []PostInfo
	Total int64
}

// PlatformPublishResult holds per-platform publish outcome.
type PlatformPublishResult struct {
	Platform string `json:"platform"`
	Success  bool   `json:"success"`
	PostID   string `json:"post_id,omitempty"`
	URL      string `json:"url,omitempty"`
	Error    string `json:"error,omitempty"`
}

type Post struct {
	q           *storage.Queries
	platformSvc *Platform
	registry    *integration.Registry
}

func NewPost(q *storage.Queries, platformSvc *Platform, registry *integration.Registry) *Post {
	return &Post{q: q, platformSvc: platformSvc, registry: registry}
}

// Create creates a new draft post.
func (s *Post) Create(ctx context.Context, projectID, content string, platforms []string) (*PostInfo, error) {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return nil, ErrProjectNotFound
	}

	if len(platforms) == 0 {
		return nil, ErrNoPlatforms
	}

	// Validate all platforms have registered adapters.
	for _, p := range platforms {
		if _, err := s.registry.Get(p); err != nil {
			return nil, fmt.Errorf("%w: %s", ErrUnsupportedPlatform, p)
		}
	}

	post, err := s.q.CreatePublishedPost(ctx, storage.CreatePublishedPostParams{
		ProjectID: pid,
		Content:   content,
		Platforms: platforms,
		Status:    storage.PostStatusDraft,
	})
	if err != nil {
		return nil, fmt.Errorf("creating post: %w", err)
	}

	return storagePostToInfo(post), nil
}

func (s *Post) List(ctx context.Context, projectID, status string, limit, offset int32) (*PostListResult, error) {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return nil, ErrProjectNotFound
	}

	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	var posts []storage.PublishedPost
	var total int64

	if status != "" {
		ps := storage.PostStatus(status)
		posts, err = s.q.ListPublishedPostsByStatus(ctx, storage.ListPublishedPostsByStatusParams{
			ProjectID: pid, Status: ps, Limit: limit, Offset: offset,
		})
		if err != nil {
			return nil, fmt.Errorf("listing posts: %w", err)
		}
		total, err = s.q.CountPublishedPostsByStatus(ctx, storage.CountPublishedPostsByStatusParams{
			ProjectID: pid, Status: ps,
		})
		if err != nil {
			return nil, fmt.Errorf("counting posts: %w", err)
		}
	} else {
		posts, err = s.q.ListPublishedPosts(ctx, storage.ListPublishedPostsParams{
			ProjectID: pid, Limit: limit, Offset: offset,
		})
		if err != nil {
			return nil, fmt.Errorf("listing posts: %w", err)
		}
		total, err = s.q.CountPublishedPosts(ctx, pid)
		if err != nil {
			return nil, fmt.Errorf("counting posts: %w", err)
		}
	}

	result := make([]PostInfo, len(posts))
	for i, p := range posts {
		result[i] = *storagePostToInfo(p)
	}
	return &PostListResult{Posts: result, Total: total}, nil
}

func (s *Post) Get(ctx context.Context, projectID, postID string) (*PostInfo, error) {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return nil, ErrPostNotFound
	}
	poid, err := uuid.Parse(postID)
	if err != nil {
		return nil, ErrPostNotFound
	}

	post, err := s.q.GetPublishedPostByID(ctx, storage.GetPublishedPostByIDParams{
		ID: poid, ProjectID: pid,
	})
	if err != nil {
		return nil, ErrPostNotFound
	}

	return storagePostToInfo(post), nil
}

func (s *Post) Update(ctx context.Context, projectID, postID, content string, platforms []string) (*PostInfo, error) {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return nil, ErrPostNotFound
	}
	poid, err := uuid.Parse(postID)
	if err != nil {
		return nil, ErrPostNotFound
	}

	for _, p := range platforms {
		if _, err := s.registry.Get(p); err != nil {
			return nil, fmt.Errorf("%w: %s", ErrUnsupportedPlatform, p)
		}
	}

	post, err := s.q.UpdatePublishedPost(ctx, storage.UpdatePublishedPostParams{
		ID: poid, ProjectID: pid, Content: content, Platforms: platforms,
	})
	if err != nil {
		// If the WHERE clause didn't match (not draft or not found), pgx returns ErrNoRows.
		return nil, ErrPostNotDraft
	}

	return storagePostToInfo(post), nil
}

func (s *Post) Delete(ctx context.Context, projectID, postID string) error {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return ErrPostNotFound
	}
	poid, err := uuid.Parse(postID)
	if err != nil {
		return ErrPostNotFound
	}

	// Verify existence.
	post, err := s.q.GetPublishedPostByID(ctx, storage.GetPublishedPostByIDParams{
		ID: poid, ProjectID: pid,
	})
	if err != nil {
		return ErrPostNotFound
	}
	if post.Status != storage.PostStatusDraft {
		return ErrPostNotDraft
	}

	return s.q.DeletePublishedPost(ctx, storage.DeletePublishedPostParams{
		ID: poid, ProjectID: pid,
	})
}

// Publish publishes a draft post to all its target platforms synchronously.
// Returns per-platform results and an error if any platform failed.
func (s *Post) Publish(ctx context.Context, projectID, postID string) ([]PlatformPublishResult, error) {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return nil, ErrPostNotFound
	}
	poid, err := uuid.Parse(postID)
	if err != nil {
		return nil, ErrPostNotFound
	}

	post, err := s.q.GetPublishedPostByID(ctx, storage.GetPublishedPostByIDParams{
		ID: poid, ProjectID: pid,
	})
	if err != nil {
		return nil, ErrPostNotFound
	}
	if post.Status != storage.PostStatusDraft {
		return nil, ErrPostNotDraft
	}
	if len(post.Platforms) == 0 {
		return nil, ErrNoPlatforms
	}

	// Set status to publishing.
	_, err = s.q.UpdatePublishedPostStatus(ctx, storage.UpdatePublishedPostStatusParams{
		ID: poid, ProjectID: pid, Status: storage.PostStatusPublishing,
	})
	if err != nil {
		return nil, fmt.Errorf("setting publishing status: %w", err)
	}

	// Publish to each platform.
	results := make([]PlatformPublishResult, len(post.Platforms))
	anyFailed := false

	for i, platform := range post.Platforms {
		results[i] = PlatformPublishResult{Platform: platform}

		adapter, err := s.registry.Get(platform)
		if err != nil {
			results[i].Error = fmt.Sprintf("unsupported platform: %s", platform)
			anyFailed = true
			continue
		}

		creds, err := s.platformSvc.GetCredentials(ctx, projectID, platform)
		if err != nil {
			results[i].Error = fmt.Sprintf("no enabled connection for %s", platform)
			anyFailed = true
			continue
		}

		result, err := adapter.Publish(ctx, post.Content, creds)
		if err != nil {
			log.Printf("publish: %s failed for post %s: %v", platform, postID, err)
			results[i].Error = err.Error()
			anyFailed = true
			continue
		}

		results[i].Success = true
		results[i].PostID = result.PlatformPostID
		results[i].URL = result.URL
	}

	// Update final status.
	if anyFailed {
		s.q.UpdatePublishedPostStatus(ctx, storage.UpdatePublishedPostStatusParams{
			ID: poid, ProjectID: pid, Status: storage.PostStatusFailed,
		})
		return results, ErrPublishFailed
	}

	s.q.SetPublishedPostPublished(ctx, storage.SetPublishedPostPublishedParams{
		ID: poid, ProjectID: pid,
	})
	return results, nil
}

func storagePostToInfo(p storage.PublishedPost) *PostInfo {
	info := &PostInfo{
		ID:        p.ID.String(),
		ProjectID: p.ProjectID.String(),
		Content:   p.Content,
		Platforms: p.Platforms,
		Status:    string(p.Status),
		CreatedAt: p.CreatedAt.Time,
		UpdatedAt: p.UpdatedAt.Time,
	}
	if p.ScheduledAt.Valid {
		t := p.ScheduledAt.Time
		info.ScheduledAt = &t
	}
	if p.PublishedAt.Valid {
		t := p.PublishedAt.Time
		info.PublishedAt = &t
	}
	return info
}
