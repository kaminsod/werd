package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"

	"github.com/werd-platform/werd/src/go/api/internal/storage"
)

var (
	ErrKeywordNotFound  = errors.New("keyword not found")
	ErrInvalidMatchType = errors.New("invalid match type")
	ErrInvalidRegex     = errors.New("invalid regex pattern")
)

type KeywordInfo struct {
	ID        string
	ProjectID string
	Keyword   string
	MatchType string
	CreatedAt time.Time
}

type Keyword struct {
	q *storage.Queries
}

func NewKeyword(q *storage.Queries) *Keyword {
	return &Keyword{q: q}
}

// Create adds a new keyword to a project. Validates regex patterns upfront.
func (s *Keyword) Create(ctx context.Context, projectID, keyword, matchType string) (*KeywordInfo, error) {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return nil, ErrProjectNotFound
	}

	if keyword == "" {
		return nil, fmt.Errorf("keyword is required")
	}

	mt, err := parseKeywordMatchType(matchType)
	if err != nil {
		return nil, ErrInvalidMatchType
	}

	if mt == storage.KeywordMatchTypeRegex {
		if _, err := regexp.Compile(keyword); err != nil {
			return nil, ErrInvalidRegex
		}
	}

	kw, err := s.q.CreateKeyword(ctx, storage.CreateKeywordParams{
		ProjectID: pid,
		Keyword:   keyword,
		MatchType: mt,
	})
	if err != nil {
		return nil, fmt.Errorf("creating keyword: %w", err)
	}

	return storageKeywordToInfo(kw), nil
}

// List returns all keywords for a project.
func (s *Keyword) List(ctx context.Context, projectID string) ([]KeywordInfo, error) {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return nil, ErrProjectNotFound
	}

	keywords, err := s.q.ListKeywords(ctx, pid)
	if err != nil {
		return nil, fmt.Errorf("listing keywords: %w", err)
	}

	result := make([]KeywordInfo, len(keywords))
	for i, kw := range keywords {
		result[i] = *storageKeywordToInfo(kw)
	}
	return result, nil
}

// Delete removes a keyword from a project.
func (s *Keyword) Delete(ctx context.Context, projectID, keywordID string) error {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return ErrProjectNotFound
	}
	kid, err := uuid.Parse(keywordID)
	if err != nil {
		return ErrKeywordNotFound
	}

	_, err = s.q.GetKeywordByID(ctx, storage.GetKeywordByIDParams{
		ID:        kid,
		ProjectID: pid,
	})
	if err != nil {
		return ErrKeywordNotFound
	}

	return s.q.DeleteKeyword(ctx, storage.DeleteKeywordParams{
		ID:        kid,
		ProjectID: pid,
	})
}

func parseKeywordMatchType(s string) (storage.KeywordMatchType, error) {
	switch storage.KeywordMatchType(s) {
	case storage.KeywordMatchTypeExact, storage.KeywordMatchTypeSubstring, storage.KeywordMatchTypeRegex:
		return storage.KeywordMatchType(s), nil
	default:
		return "", fmt.Errorf("invalid match type: %s", s)
	}
}

func storageKeywordToInfo(kw storage.Keyword) *KeywordInfo {
	return &KeywordInfo{
		ID:        kw.ID.String(),
		ProjectID: kw.ProjectID.String(),
		Keyword:   kw.Keyword,
		MatchType: string(kw.MatchType),
		CreatedAt: kw.CreatedAt.Time,
	}
}
