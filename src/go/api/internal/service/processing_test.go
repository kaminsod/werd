package service

import (
	"testing"
	"time"

	"github.com/werd-platform/werd/src/go/api/internal/integration"
	"github.com/werd-platform/werd/src/go/api/internal/storage"
)

func TestMatchKeywordFilter_Substring(t *testing.T) {
	cfg := filterConfig{
		Keywords:  []string{"golang", "rust"},
		MatchType: "substring",
		Fields:    []string{"title", "content"},
		Action:    "include",
	}

	item := integration.MonitoredItem{
		Title:   "Learning Golang in 2026",
		Content: "A guide to getting started",
	}

	if !matchKeywordFilter(cfg, item) {
		t.Error("expected match on 'golang' in title")
	}

	item2 := integration.MonitoredItem{
		Title:   "Python is great",
		Content: "No matching keywords here",
	}

	if matchKeywordFilter(cfg, item2) {
		t.Error("expected no match")
	}
}

func TestMatchKeywordFilter_Exact(t *testing.T) {
	cfg := filterConfig{
		Keywords:  []string{"Breaking News"},
		MatchType: "exact",
		Fields:    []string{"title"},
	}

	item := integration.MonitoredItem{Title: "Breaking News"}
	if !matchKeywordFilter(cfg, item) {
		t.Error("expected exact match on title")
	}

	item2 := integration.MonitoredItem{Title: "This is Breaking News today"}
	if matchKeywordFilter(cfg, item2) {
		t.Error("expected no exact match")
	}
}

func TestMatchKeywordFilter_Regex(t *testing.T) {
	cfg := filterConfig{
		Keywords:  []string{`CVE-\d{4}-\d+`},
		MatchType: "regex",
		Fields:    []string{"title", "content"},
	}

	item := integration.MonitoredItem{Title: "Security fix for CVE-2026-12345"}
	if !matchKeywordFilter(cfg, item) {
		t.Error("expected regex match")
	}

	item2 := integration.MonitoredItem{Title: "Just a normal post"}
	if matchKeywordFilter(cfg, item2) {
		t.Error("expected no regex match")
	}
}

func TestMatchRegexFilter(t *testing.T) {
	cfg := filterConfig{
		Pattern: `(?i)\b(security|vulnerability)\b`,
		Fields:  []string{"title", "content"},
	}

	item := integration.MonitoredItem{Title: "Major Security Update"}
	if !matchRegexFilter(cfg, item) {
		t.Error("expected regex match")
	}

	item2 := integration.MonitoredItem{Title: "Performance improvements"}
	if matchRegexFilter(cfg, item2) {
		t.Error("expected no regex match")
	}
}

func TestMatchRegexFilter_InvalidRegex(t *testing.T) {
	cfg := filterConfig{
		Pattern: `[invalid`,
		Fields:  []string{"title"},
	}

	item := integration.MonitoredItem{Title: "anything"}
	if matchRegexFilter(cfg, item) {
		t.Error("invalid regex should not match")
	}
}

func TestItemPassesFilters_NoRules(t *testing.T) {
	p := &ProcessingPipeline{}
	items := []integration.MonitoredItem{
		{Title: "Test", Content: "Content"},
	}

	result := p.applyFilters(nil, items)
	if len(result) != 1 {
		t.Errorf("expected 1 item, got %d", len(result))
	}
}

func TestItemPassesFilters_IncludeOnly(t *testing.T) {
	p := &ProcessingPipeline{}

	rules := []storage.ProcessingRule{
		{
			Phase:    "filter",
			RuleType: "keyword",
			Config:   []byte(`{"keywords":["golang"],"match_type":"substring","fields":["title"],"action":"include"}`),
		},
	}

	items := []integration.MonitoredItem{
		{Title: "Learning Golang", Content: "Guide"},
		{Title: "Python tutorial", Content: "Basics"},
		{Title: "Golang tips", Content: "Advanced"},
	}

	result := p.applyFilters(rules, items)
	if len(result) != 2 {
		t.Errorf("expected 2 items, got %d", len(result))
	}
}

func TestItemPassesFilters_ExcludeRule(t *testing.T) {
	p := &ProcessingPipeline{}

	rules := []storage.ProcessingRule{
		{
			Phase:    "filter",
			RuleType: "keyword",
			Config:   []byte(`{"keywords":["golang","python"],"match_type":"substring","fields":["title"],"action":"include"}`),
		},
		{
			Phase:    "filter",
			RuleType: "keyword",
			Config:   []byte(`{"keywords":["spam"],"match_type":"substring","fields":["content"],"action":"exclude"}`),
		},
	}

	items := []integration.MonitoredItem{
		{Title: "Learning Golang", Content: "Good content"},
		{Title: "Python spam", Content: "This is spam"},
		{Title: "Golang advanced", Content: "Quality content"},
	}

	result := p.applyFilters(rules, items)
	if len(result) != 2 {
		t.Errorf("expected 2 items (exclude spam), got %d", len(result))
	}
}

func TestItemPassesFilters_RegexInclude(t *testing.T) {
	p := &ProcessingPipeline{}

	rules := []storage.ProcessingRule{
		{
			Phase:    "filter",
			RuleType: "regex",
			Config:   []byte(`{"pattern":"(?i)\\b(go|rust)\\b","fields":["title"],"action":"include"}`),
		},
	}

	items := []integration.MonitoredItem{
		{Title: "Go 1.23 released"},
		{Title: "Python 3.14 released"},
		{Title: "Rust 2.0 announced"},
	}

	result := p.applyFilters(rules, items)
	if len(result) != 2 {
		t.Errorf("expected 2 items, got %d", len(result))
	}
}

func TestApplyClassify_KeywordSeverity(t *testing.T) {
	p := &ProcessingPipeline{}

	rules := []storage.ProcessingRule{
		{
			Phase:    "classify",
			RuleType: "keyword",
			Name:     "security-critical",
			Config:   []byte(`{"keywords":["critical","security"],"match_type":"substring","fields":["title","content"],"set_severity":"high","add_tags":["security"]}`),
		},
		{
			Phase:    "classify",
			RuleType: "keyword",
			Name:     "bug-report",
			Config:   []byte(`{"keywords":["bug"],"match_type":"substring","fields":["title"],"set_severity":"medium","add_tags":["bug"]}`),
		},
	}

	item := integration.MonitoredItem{
		Title:   "Critical security bug found",
		Content: "Urgent fix needed",
	}

	result := p.applyClassify(nil, rules, "reddit", item)

	if result.Severity != "high" {
		t.Errorf("expected severity 'high', got %q", result.Severity)
	}
	if len(result.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d: %v", len(result.Tags), result.Tags)
	}
	if !containsString(result.Tags, "security") || !containsString(result.Tags, "bug") {
		t.Errorf("expected tags to contain 'security' and 'bug', got %v", result.Tags)
	}
}

func TestApplyClassify_NoMatch(t *testing.T) {
	p := &ProcessingPipeline{}

	rules := []storage.ProcessingRule{
		{
			Phase:    "classify",
			RuleType: "keyword",
			Config:   []byte(`{"keywords":["critical"],"match_type":"substring","fields":["title"],"set_severity":"high","add_tags":["urgent"]}`),
		},
	}

	item := integration.MonitoredItem{
		Title:   "Regular update",
		Content: "Nothing special",
	}

	result := p.applyClassify(nil, rules, "hn", item)

	if result.Severity != "low" {
		t.Errorf("expected default severity 'low', got %q", result.Severity)
	}
	if len(result.Tags) != 0 {
		t.Errorf("expected 0 tags, got %d: %v", len(result.Tags), result.Tags)
	}
}

func TestApplyClassify_HighestSeverityWins(t *testing.T) {
	p := &ProcessingPipeline{}

	rules := []storage.ProcessingRule{
		{
			Phase:    "classify",
			RuleType: "keyword",
			Config:   []byte(`{"keywords":["update"],"match_type":"substring","fields":["title"],"set_severity":"medium","add_tags":["update"]}`),
		},
		{
			Phase:    "classify",
			RuleType: "keyword",
			Config:   []byte(`{"keywords":["critical"],"match_type":"substring","fields":["title"],"set_severity":"critical","add_tags":["critical"]}`),
		},
	}

	item := integration.MonitoredItem{
		Title:   "Critical update released",
		Content: "Details inside",
	}

	result := p.applyClassify(nil, rules, "reddit", item)

	if result.Severity != "critical" {
		t.Errorf("expected severity 'critical' (highest wins), got %q", result.Severity)
	}
}

func TestGetFieldValues_DefaultFields(t *testing.T) {
	item := integration.MonitoredItem{
		Title:   "Test Title",
		Content: "Test Content",
		Author:  "testuser",
		URL:     "https://example.com",
	}

	// Empty fields = default to title + content.
	vals := getFieldValues(nil, item)
	if len(vals) != 2 {
		t.Errorf("expected 2 default fields, got %d", len(vals))
	}

	// Explicit fields.
	vals = getFieldValues([]string{"author", "url"}, item)
	if len(vals) != 2 || vals[0] != "testuser" || vals[1] != "https://example.com" {
		t.Errorf("expected [testuser, https://example.com], got %v", vals)
	}
}

func TestContainsString(t *testing.T) {
	slice := []string{"a", "b", "c"}
	if !containsString(slice, "b") {
		t.Error("expected true for existing element")
	}
	if containsString(slice, "d") {
		t.Error("expected false for missing element")
	}
}

func TestMatchKeywordFilter_DefaultFields(t *testing.T) {
	// When no fields specified, should default to title + content.
	cfg := filterConfig{
		Keywords:  []string{"test"},
		MatchType: "substring",
		Action:    "include",
	}

	item := integration.MonitoredItem{
		Title:     "No match here",
		Content:   "But test is in content",
		CreatedAt: time.Now(),
	}

	if !matchKeywordFilter(cfg, item) {
		t.Error("expected match on content with default fields")
	}
}
