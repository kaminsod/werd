package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

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
	Title       string
	Content     string
	URL         string
	PostType    string
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
func (s *Post) Create(ctx context.Context, projectID, title, content, postURL, postType string, platforms []string) (*PostInfo, error) {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return nil, ErrProjectNotFound
	}

	if len(platforms) == 0 {
		return nil, ErrNoPlatforms
	}
	if postType == "" {
		postType = "text"
	}

	// Validate all platforms have at least one registered adapter (api or browser).
	for _, p := range platforms {
		_, apiErr := s.registry.Get(p + ":api")
		_, browserErr := s.registry.Get(p + ":browser")
		if apiErr != nil && browserErr != nil {
			return nil, fmt.Errorf("%w: %s", ErrUnsupportedPlatform, p)
		}
	}

	post, err := s.q.CreatePublishedPost(ctx, storage.CreatePublishedPostParams{
		ProjectID: pid,
		Title:     title,
		Content:   content,
		Url:       postURL,
		PostType:  storage.PostType(postType),
		Platforms: platforms,
		Status:    storage.PostStatusDraft,
	})
	if err != nil {
		return nil, fmt.Errorf("creating post: %w", err)
	}

	return postFromCreate(post), nil
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

	var result []PostInfo
	var total int64

	if status != "" {
		ps := storage.PostStatus(status)
		posts, err := s.q.ListPublishedPostsByStatus(ctx, storage.ListPublishedPostsByStatusParams{
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
		result = make([]PostInfo, len(posts))
		for i, p := range posts {
			result[i] = *postFromListStatus(p)
		}
	} else {
		posts, err := s.q.ListPublishedPosts(ctx, storage.ListPublishedPostsParams{
			ProjectID: pid, Limit: limit, Offset: offset,
		})
		if err != nil {
			return nil, fmt.Errorf("listing posts: %w", err)
		}
		total, err = s.q.CountPublishedPosts(ctx, pid)
		if err != nil {
			return nil, fmt.Errorf("counting posts: %w", err)
		}
		result = make([]PostInfo, len(posts))
		for i, p := range posts {
			result[i] = *postFromList(p)
		}
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

	return postFromGet(post), nil
}

func (s *Post) Update(ctx context.Context, projectID, postID, title, content, postURL, postType string, platforms []string) (*PostInfo, error) {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return nil, ErrPostNotFound
	}
	poid, err := uuid.Parse(postID)
	if err != nil {
		return nil, ErrPostNotFound
	}
	if postType == "" {
		postType = "text"
	}

	for _, p := range platforms {
		_, apiErr := s.registry.Get(p + ":api")
		_, browserErr := s.registry.Get(p + ":browser")
		if apiErr != nil && browserErr != nil {
			return nil, fmt.Errorf("%w: %s", ErrUnsupportedPlatform, p)
		}
	}

	post, err := s.q.UpdatePublishedPost(ctx, storage.UpdatePublishedPostParams{
		ID: poid, ProjectID: pid, Title: title, Content: content,
		Url: postURL, PostType: storage.PostType(postType), Platforms: platforms,
	})
	if err != nil {
		// If the WHERE clause didn't match (not draft or not found), pgx returns ErrNoRows.
		return nil, ErrPostNotDraft
	}

	return postFromUpdate(post), nil
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

		// Look up the enabled connection (returns method + credentials).
		conn, err := s.platformSvc.GetConnectionForPublish(ctx, projectID, platform)
		if err != nil {
			results[i].Error = fmt.Sprintf("no enabled connection for %s", platform)
			anyFailed = true
			continue
		}

		// Use platform:method as the registry key.
		adapterKey := conn.Platform + ":" + conn.Method
		adapter, err := s.registry.Get(adapterKey)
		if err != nil {
			results[i].Error = fmt.Sprintf("unsupported platform/method: %s", adapterKey)
			anyFailed = true
			continue
		}

		pubContent := integration.PublishContent{
			Title:    post.Title,
			Body:     post.Content,
			URL:      post.Url,
			PostType: string(post.PostType),
		}
		// Backward compat: if no structured title, use content as body.
		if pubContent.Title == "" && pubContent.Body == "" {
			pubContent.Body = post.Content
		}

		result, err := adapter.Publish(ctx, pubContent, conn.Credentials)
		if err != nil {
			log.Printf("publish: %s (%s) failed for post %s: %v", platform, conn.Method, postID, err)
			results[i].Error = err.Error()
			anyFailed = true
			continue
		}

		results[i].Success = true
		results[i].PostID = result.PlatformPostID
		results[i].URL = result.URL

		// Persist per-platform result.
		now := time.Now()
		s.q.CreatePostPlatformResult(ctx, storage.CreatePostPlatformResultParams{
			PostID:         poid,
			Platform:       platform,
			PlatformPostID: result.PlatformPostID,
			PlatformUrl:    result.URL,
			Success:        true,
			PublishedAt:    pgtype.Timestamptz{Time: now, Valid: true},
		})
	}

	// Persist failed results too.
	for _, r := range results {
		if !r.Success && r.Error != "" {
			s.q.CreatePostPlatformResult(ctx, storage.CreatePostPlatformResultParams{
				PostID:       poid,
				Platform:     r.Platform,
				Success:      false,
				ErrorMessage: r.Error,
			})
		}
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

func makePostInfo(id, projectID uuid.UUID, title, content, url string, postType storage.PostType, platforms []string, scheduledAt, publishedAt pgtype.Timestamptz, status storage.PostStatus, createdAt, updatedAt pgtype.Timestamptz) *PostInfo {
	info := &PostInfo{
		ID:        id.String(),
		ProjectID: projectID.String(),
		Title:     title,
		Content:   content,
		URL:       url,
		PostType:  string(postType),
		Platforms: platforms,
		Status:    string(status),
		CreatedAt: createdAt.Time,
		UpdatedAt: updatedAt.Time,
	}
	if scheduledAt.Valid {
		t := scheduledAt.Time
		info.ScheduledAt = &t
	}
	if publishedAt.Valid {
		t := publishedAt.Time
		info.PublishedAt = &t
	}
	return info
}

func postFromCreate(p storage.CreatePublishedPostRow) *PostInfo {
	return makePostInfo(p.ID, p.ProjectID, p.Title, p.Content, p.Url, p.PostType, p.Platforms, p.ScheduledAt, p.PublishedAt, p.Status, p.CreatedAt, p.UpdatedAt)
}

func postFromGet(p storage.GetPublishedPostByIDRow) *PostInfo {
	return makePostInfo(p.ID, p.ProjectID, p.Title, p.Content, p.Url, p.PostType, p.Platforms, p.ScheduledAt, p.PublishedAt, p.Status, p.CreatedAt, p.UpdatedAt)
}

func postFromList(p storage.ListPublishedPostsRow) *PostInfo {
	return makePostInfo(p.ID, p.ProjectID, p.Title, p.Content, p.Url, p.PostType, p.Platforms, p.ScheduledAt, p.PublishedAt, p.Status, p.CreatedAt, p.UpdatedAt)
}

func postFromListStatus(p storage.ListPublishedPostsByStatusRow) *PostInfo {
	return makePostInfo(p.ID, p.ProjectID, p.Title, p.Content, p.Url, p.PostType, p.Platforms, p.ScheduledAt, p.PublishedAt, p.Status, p.CreatedAt, p.UpdatedAt)
}

func postFromUpdate(p storage.UpdatePublishedPostRow) *PostInfo {
	return makePostInfo(p.ID, p.ProjectID, p.Title, p.Content, p.Url, p.PostType, p.Platforms, p.ScheduledAt, p.PublishedAt, p.Status, p.CreatedAt, p.UpdatedAt)
}

func postFromStatus(p storage.UpdatePublishedPostStatusRow) *PostInfo {
	return makePostInfo(p.ID, p.ProjectID, p.Title, p.Content, p.Url, p.PostType, p.Platforms, p.ScheduledAt, p.PublishedAt, p.Status, p.CreatedAt, p.UpdatedAt)
}

func postFromPublished(p storage.SetPublishedPostPublishedRow) *PostInfo {
	return makePostInfo(p.ID, p.ProjectID, p.Title, p.Content, p.Url, p.PostType, p.Platforms, p.ScheduledAt, p.PublishedAt, p.Status, p.CreatedAt, p.UpdatedAt)
}
