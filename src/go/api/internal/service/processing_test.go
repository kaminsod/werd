package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

// newMockLLMServer creates a mock OpenAI-compatible server returning the given result.
func newMockLLMServer(t *testing.T, result LLMClassifyResult) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		content, _ := json.Marshal(result)
		resp := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": string(content)}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
}

func TestApplyClassify_LLM(t *testing.T) {
	server := newMockLLMServer(t, LLMClassifyResult{
		Relevant: true,
		Severity: "high",
		Tags:     []string{"security", "cve"},
		Reason:   "Contains CVE reference",
	})
	defer server.Close()

	llmCli := NewLLMClient(server.URL, "test-key", "test-model")
	p := &ProcessingPipeline{llmCli: llmCli}

	rules := []storage.ProcessingRule{
		{
			Phase:    "classify",
			RuleType: "keyword",
			Name:     "kw-security",
			Config:   []byte(`{"keywords":["security"],"match_type":"substring","fields":["title"],"set_severity":"medium","add_tags":["sec-kw"]}`),
		},
		{
			Phase:    "classify",
			RuleType: "llm",
			Name:     "llm-classify",
			Config:   []byte(`{"prompt_template":"Classify: {{title}} {{content}}"}`),
		},
	}

	item := integration.MonitoredItem{
		Title:   "Security vulnerability CVE-2026-1234",
		Content: "Critical issue found",
	}

	result := p.applyClassify(context.Background(), rules, "reddit", item)

	if result.Severity != "high" {
		t.Errorf("expected severity 'high' from LLM, got %q", result.Severity)
	}
	if !containsString(result.Tags, "security") || !containsString(result.Tags, "cve") || !containsString(result.Tags, "sec-kw") {
		t.Errorf("expected tags to contain security, cve, sec-kw; got %v", result.Tags)
	}
	if result.ClassificationReason != "Contains CVE reference" {
		t.Errorf("expected LLM reason, got %q", result.ClassificationReason)
	}
}

func TestApplyClassify_LLM_NotRelevant(t *testing.T) {
	server := newMockLLMServer(t, LLMClassifyResult{
		Relevant: false,
		Severity: "low",
		Tags:     []string{},
		Reason:   "Not relevant",
	})
	defer server.Close()

	llmCli := NewLLMClient(server.URL, "", "model")
	p := &ProcessingPipeline{llmCli: llmCli}

	rules := []storage.ProcessingRule{
		{
			Phase:    "classify",
			RuleType: "keyword",
			Name:     "kw-match",
			Config:   []byte(`{"keywords":["test"],"match_type":"substring","fields":["title"],"set_severity":"medium","add_tags":["matched"]}`),
		},
		{
			Phase:    "classify",
			RuleType: "llm",
			Name:     "llm-rule",
			Config:   []byte(`{"prompt_template":"Classify: {{title}}"}`),
		},
	}

	item := integration.MonitoredItem{Title: "test post", Content: "content"}
	result := p.applyClassify(context.Background(), rules, "hn", item)

	// Keyword rule matched so severity is medium, but LLM said not relevant so no LLM enrichment.
	if result.Severity != "medium" {
		t.Errorf("expected severity 'medium' from keyword rule, got %q", result.Severity)
	}
	if !containsString(result.Tags, "matched") {
		t.Errorf("expected tag 'matched' from keyword rule, got %v", result.Tags)
	}
}

func TestApplyClassify_LLM_OnlyIfKeywords_DefaultTrue(t *testing.T) {
	// LLM rule without explicit only_if_keywords should default to true,
	// meaning LLM is skipped when no keyword rules matched.
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		content := `{"relevant":true,"severity":"high","tags":["llm"],"reason":"LLM ran"}`
		resp := map[string]any{
			"choices": []map[string]any{
				{"message": map[string]string{"content": content}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	llmCli := NewLLMClient(server.URL, "", "model")
	p := &ProcessingPipeline{llmCli: llmCli}

	rules := []storage.ProcessingRule{
		{
			Phase:    "classify",
			RuleType: "keyword",
			Name:     "no-match-kw",
			Config:   []byte(`{"keywords":["nomatch"],"match_type":"substring","fields":["title"],"set_severity":"medium"}`),
		},
		{
			Phase:    "classify",
			RuleType: "llm",
			Name:     "llm-rule",
			// only_if_keywords absent — should default to true.
			Config: []byte(`{"prompt_template":"Classify: {{title}}"}`),
		},
	}

	item := integration.MonitoredItem{Title: "unrelated post", Content: "nothing here"}
	result := p.applyClassify(context.Background(), rules, "reddit", item)

	if called {
		t.Error("LLM should NOT have been called when only_if_keywords defaults true and no keywords matched")
	}
	if result.Severity != "low" {
		t.Errorf("expected default severity 'low', got %q", result.Severity)
	}
}

func TestApplyClassify_LLM_OnlyIfKeywords_ExplicitFalse(t *testing.T) {
	// When only_if_keywords is explicitly false, LLM should run even without keyword matches.
	server := newMockLLMServer(t, LLMClassifyResult{
		Relevant: true,
		Severity: "high",
		Tags:     []string{"llm-tag"},
		Reason:   "LLM classified",
	})
	defer server.Close()

	llmCli := NewLLMClient(server.URL, "", "model")
	p := &ProcessingPipeline{llmCli: llmCli}

	rules := []storage.ProcessingRule{
		{
			Phase:    "classify",
			RuleType: "llm",
			Name:     "llm-always",
			Config:   []byte(`{"prompt_template":"Classify: {{title}}","only_if_keywords":false}`),
		},
	}

	item := integration.MonitoredItem{Title: "some post", Content: "content"}
	result := p.applyClassify(context.Background(), rules, "reddit", item)

	if result.Severity != "high" {
		t.Errorf("expected severity 'high' from LLM, got %q", result.Severity)
	}
	if !containsString(result.Tags, "llm-tag") {
		t.Errorf("expected tag 'llm-tag', got %v", result.Tags)
	}
}

func TestCombinedFilterAndClassify(t *testing.T) {
	// Tests the combined filter→classify pipeline flow using the public methods.
	server := newMockLLMServer(t, LLMClassifyResult{
		Relevant: true,
		Severity: "critical",
		Tags:     []string{"urgent"},
		Reason:   "LLM says critical",
	})
	defer server.Close()

	llmCli := NewLLMClient(server.URL, "", "model")
	p := &ProcessingPipeline{llmCli: llmCli}

	filterRules := []storage.ProcessingRule{
		{
			Phase:    "filter",
			RuleType: "keyword",
			Config:   []byte(`{"keywords":["golang","security"],"match_type":"substring","fields":["title"],"action":"include"}`),
		},
		{
			Phase:    "filter",
			RuleType: "keyword",
			Config:   []byte(`{"keywords":["spam"],"match_type":"substring","fields":["content"],"action":"exclude"}`),
		},
	}

	classifyRules := []storage.ProcessingRule{
		{
			Phase:    "classify",
			RuleType: "keyword",
			Name:     "sec-kw",
			Config:   []byte(`{"keywords":["security"],"match_type":"substring","fields":["title"],"set_severity":"high","add_tags":["security"]}`),
		},
		{
			Phase:    "classify",
			RuleType: "llm",
			Name:     "llm-rule",
			Config:   []byte(`{"prompt_template":"Classify: {{title}}"}`),
		},
	}

	items := []integration.MonitoredItem{
		{Title: "Golang security issue", Content: "Important fix"},
		{Title: "Golang performance", Content: "Benchmark results"},
		{Title: "Security alert", Content: "This is spam content"},
		{Title: "Python update", Content: "Not matching"},
	}

	// Filter phase.
	filtered := p.applyFilters(filterRules, items)

	// Should keep items 0 and 1 (match "golang" or "security"), exclude item 2 (spam), drop item 3 (no match).
	if len(filtered) != 2 {
		t.Fatalf("expected 2 filtered items, got %d", len(filtered))
	}
	if filtered[0].Title != "Golang security issue" {
		t.Errorf("expected first item 'Golang security issue', got %q", filtered[0].Title)
	}
	if filtered[1].Title != "Golang performance" {
		t.Errorf("expected second item 'Golang performance', got %q", filtered[1].Title)
	}

	// Classify phase on filtered items.
	results := make([]ProcessingResult, len(filtered))
	for i, item := range filtered {
		results[i] = p.applyClassify(context.Background(), classifyRules, "reddit", item)
	}

	// First item matches "security" keyword → severity=high, then LLM upgrades to critical.
	if results[0].Severity != "critical" {
		t.Errorf("expected severity 'critical' for first item, got %q", results[0].Severity)
	}
	if !containsString(results[0].Tags, "security") || !containsString(results[0].Tags, "urgent") {
		t.Errorf("expected tags [security, urgent] for first item, got %v", results[0].Tags)
	}

	// Second item: no keyword match, LLM should NOT run (only_if_keywords defaults true).
	if results[1].Severity != "low" {
		t.Errorf("expected severity 'low' for second item (no keyword match), got %q", results[1].Severity)
	}
}
