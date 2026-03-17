package service

import (
	"context"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/werd-platform/werd/src/go/api/internal/integration"
	"github.com/werd-platform/werd/src/go/api/internal/storage"
)

var (
	ErrProcessingRuleNotFound = errors.New("processing rule not found")
	ErrInvalidPhase           = errors.New("invalid phase; must be filter or classify")
	ErrInvalidRuleType        = errors.New("invalid rule_type; must be keyword, regex, or llm")
)

// ProcessingRuleInfo is the service-layer representation of a processing rule.
type ProcessingRuleInfo struct {
	ID        string
	ProjectID string
	SourceID  string // empty string = project-wide
	Name      string
	Phase     string
	RuleType  string
	Config    map[string]any
	Priority  int
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ProcessingResult holds the enrichment data produced by the classify phase.
type ProcessingResult struct {
	Severity             string
	Tags                 []string
	ClassificationReason string
}

// ProcessingPipeline applies filter and classify rules to monitored items.
type ProcessingPipeline struct {
	q      *storage.Queries
	llmCli *LLMClient // nil if LLM disabled
}

func NewProcessingPipeline(q *storage.Queries, llmCli *LLMClient) *ProcessingPipeline {
	return &ProcessingPipeline{q: q, llmCli: llmCli}
}

// Process runs the full pipeline (filter + classify) on a batch of items.
// Returns items that passed filtering, each paired with its classification result.
func (p *ProcessingPipeline) Process(
	ctx context.Context,
	projectID uuid.UUID,
	sourceID uuid.UUID,
	sourceType string,
	items []integration.MonitoredItem,
) ([]integration.MonitoredItem, []ProcessingResult, error) {
	rules, err := p.q.ListRulesForSource(ctx, storage.ListRulesForSourceParams{
		SourceID:  pgtype.UUID{Bytes: sourceID, Valid: true},
		ProjectID: projectID,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("loading processing rules: %w", err)
	}

	// Separate rules by phase.
	var filterRules, classifyRules []storage.ProcessingRule
	for _, r := range rules {
		switch r.Phase {
		case "filter":
			filterRules = append(filterRules, r)
		case "classify":
			classifyRules = append(classifyRules, r)
		}
	}

	// Filter phase.
	filtered := p.applyFilters(filterRules, items)

	// Classify phase.
	results := make([]ProcessingResult, len(filtered))
	for i, item := range filtered {
		results[i] = p.applyClassify(ctx, classifyRules, sourceType, item)
	}

	return filtered, results, nil
}

// applyFilters applies filter rules to items.
// If no filter rules exist, all items pass through.
// Otherwise: item must match at least one "include" rule AND not match any "exclude" rule.
func (p *ProcessingPipeline) applyFilters(rules []storage.ProcessingRule, items []integration.MonitoredItem) []integration.MonitoredItem {
	if len(rules) == 0 {
		return items
	}

	var result []integration.MonitoredItem
	for _, item := range items {
		if p.itemPassesFilters(rules, item) {
			result = append(result, item)
		}
	}
	return result
}

func (p *ProcessingPipeline) itemPassesFilters(rules []storage.ProcessingRule, item integration.MonitoredItem) bool {
	hasIncludeRules := false
	matchedInclude := false

	for _, rule := range rules {
		var cfg filterConfig
		if err := json.Unmarshal(rule.Config, &cfg); err != nil {
			log.Printf("processing: invalid filter config for rule %s: %v", rule.ID, err)
			continue
		}

		matched := false
		switch rule.RuleType {
		case "keyword":
			matched = matchKeywordFilter(cfg, item)
		case "regex":
			matched = matchRegexFilter(cfg, item)
		}

		action := cfg.Action
		if action == "" {
			action = "include"
		}

		if action == "exclude" && matched {
			return false // excluded
		}

		if action == "include" {
			hasIncludeRules = true
			if matched {
				matchedInclude = true
			}
		}
	}

	if hasIncludeRules && !matchedInclude {
		return false
	}
	return true
}

type filterConfig struct {
	// Keyword filter fields.
	Keywords  []string `json:"keywords"`
	MatchType string   `json:"match_type"` // exact, substring, regex
	// Regex filter fields.
	Pattern string `json:"pattern"`
	// Common fields.
	Fields []string `json:"fields"` // title, content
	Action string   `json:"action"` // include, exclude
}

func getFieldValues(fields []string, item integration.MonitoredItem) []string {
	if len(fields) == 0 {
		fields = []string{"title", "content"}
	}
	var vals []string
	for _, f := range fields {
		switch f {
		case "title":
			vals = append(vals, item.Title)
		case "content":
			vals = append(vals, item.Content)
		case "author":
			vals = append(vals, item.Author)
		case "url":
			vals = append(vals, item.URL)
		}
	}
	return vals
}

func matchKeywordFilter(cfg filterConfig, item integration.MonitoredItem) bool {
	fieldValues := getFieldValues(cfg.Fields, item)
	matchType := cfg.MatchType
	if matchType == "" {
		matchType = "substring"
	}

	for _, kw := range cfg.Keywords {
		for _, val := range fieldValues {
			switch matchType {
			case "exact":
				if strings.EqualFold(val, kw) {
					return true
				}
			case "substring":
				if strings.Contains(strings.ToLower(val), strings.ToLower(kw)) {
					return true
				}
			case "regex":
				re, err := regexp.Compile("(?i)" + kw)
				if err != nil {
					continue
				}
				if re.MatchString(val) {
					return true
				}
			}
		}
	}
	return false
}

func matchRegexFilter(cfg filterConfig, item integration.MonitoredItem) bool {
	if cfg.Pattern == "" {
		return false
	}
	re, err := regexp.Compile(cfg.Pattern)
	if err != nil {
		log.Printf("processing: invalid regex pattern %q: %v", cfg.Pattern, err)
		return false
	}
	fieldValues := getFieldValues(cfg.Fields, item)
	for _, val := range fieldValues {
		if re.MatchString(val) {
			return true
		}
	}
	return false
}

// classifyConfig holds the config shape for classify rules.
type classifyConfig struct {
	// Keyword classify fields.
	Keywords    []string `json:"keywords"`
	MatchType   string   `json:"match_type"`
	Fields      []string `json:"fields"`
	SetSeverity string   `json:"set_severity"`
	AddTags     []string `json:"add_tags"`
	// LLM classify fields.
	PromptTemplate string `json:"prompt_template"`
	MaxTokens      int    `json:"max_tokens"`
	OnlyIfKeywords bool   `json:"only_if_keywords"`
}

func (p *ProcessingPipeline) applyClassify(ctx context.Context, rules []storage.ProcessingRule, sourceType string, item integration.MonitoredItem) ProcessingResult {
	result := ProcessingResult{
		Severity: "low",
		Tags:     []string{},
	}

	// Track whether any keyword filter matched (for only_if_keywords).
	keywordMatched := false

	for _, rule := range rules {
		var cfg classifyConfig
		if err := json.Unmarshal(rule.Config, &cfg); err != nil {
			log.Printf("processing: invalid classify config for rule %s: %v", rule.ID, err)
			continue
		}

		switch rule.RuleType {
		case "keyword":
			if matchKeywordClassify(cfg, item) {
				keywordMatched = true
				if cfg.SetSeverity != "" && severityRank(storage.AlertSeverity(cfg.SetSeverity)) > severityRank(storage.AlertSeverity(result.Severity)) {
					result.Severity = cfg.SetSeverity
				}
				for _, tag := range cfg.AddTags {
					if !containsString(result.Tags, tag) {
						result.Tags = append(result.Tags, tag)
					}
				}
				if result.ClassificationReason == "" {
					result.ClassificationReason = fmt.Sprintf("matched keyword rule %q", rule.Name)
				}
			}

		case "llm":
			if p.llmCli == nil {
				continue
			}
			// Respect only_if_keywords (default true when absent from config JSON).
			onlyIfKW := cfg.OnlyIfKeywords
			if !onlyIfKW && !bytes.Contains(rule.Config, []byte(`"only_if_keywords"`)) {
				onlyIfKW = true
			}
			if cfg.PromptTemplate != "" && (!onlyIfKW || keywordMatched) {
				llmResult, err := p.runLLMClassify(ctx, cfg, sourceType, item)
				if err != nil {
					log.Printf("processing: LLM classify failed for rule %s: %v", rule.ID, err)
					continue
				}
				if !llmResult.Relevant {
					// LLM says not relevant — we could drop the item but since we're
					// in classify phase (item already passed filters), just skip enrichment.
					continue
				}
				if llmResult.Severity != "" && severityRank(storage.AlertSeverity(llmResult.Severity)) > severityRank(storage.AlertSeverity(result.Severity)) {
					result.Severity = llmResult.Severity
				}
				for _, tag := range llmResult.Tags {
					if !containsString(result.Tags, tag) {
						result.Tags = append(result.Tags, tag)
					}
				}
				if llmResult.Reason != "" {
					result.ClassificationReason = llmResult.Reason
				}
			}
		}
	}

	return result
}

func matchKeywordClassify(cfg classifyConfig, item integration.MonitoredItem) bool {
	fieldValues := getFieldValues(cfg.Fields, item)
	matchType := cfg.MatchType
	if matchType == "" {
		matchType = "substring"
	}

	for _, kw := range cfg.Keywords {
		for _, val := range fieldValues {
			switch matchType {
			case "exact":
				if strings.EqualFold(val, kw) {
					return true
				}
			case "substring":
				if strings.Contains(strings.ToLower(val), strings.ToLower(kw)) {
					return true
				}
			case "regex":
				re, err := regexp.Compile("(?i)" + kw)
				if err != nil {
					continue
				}
				if re.MatchString(val) {
					return true
				}
			}
		}
	}
	return false
}

func (p *ProcessingPipeline) runLLMClassify(ctx context.Context, cfg classifyConfig, sourceType string, item integration.MonitoredItem) (*LLMClassifyResult, error) {
	prompt := cfg.PromptTemplate
	prompt = strings.ReplaceAll(prompt, "{{source_type}}", sourceType)
	prompt = strings.ReplaceAll(prompt, "{{title}}", item.Title)
	prompt = strings.ReplaceAll(prompt, "{{content}}", item.Content)
	prompt = strings.ReplaceAll(prompt, "{{url}}", item.URL)
	prompt = strings.ReplaceAll(prompt, "{{author}}", item.Author)

	maxTokens := cfg.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 200
	}

	return p.llmCli.Classify(ctx, prompt, maxTokens)
}

func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// --- CRUD Service ---

// ProcessingRuleService handles CRUD for processing rules.
type ProcessingRuleService struct {
	q *storage.Queries
}

func NewProcessingRuleService(q *storage.Queries) *ProcessingRuleService {
	return &ProcessingRuleService{q: q}
}

func (s *ProcessingRuleService) Create(ctx context.Context, projectID, sourceID, name, phase, ruleType string, config map[string]any, priority int, enabled bool) (*ProcessingRuleInfo, error) {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return nil, ErrProjectNotFound
	}

	if err := validatePhase(phase); err != nil {
		return nil, err
	}
	if err := validateRuleType(ruleType); err != nil {
		return nil, err
	}

	var sid pgtype.UUID
	if sourceID != "" {
		parsed, err := uuid.Parse(sourceID)
		if err != nil {
			return nil, fmt.Errorf("invalid source_id: %w", err)
		}
		sid = pgtype.UUID{Bytes: parsed, Valid: true}
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("marshaling config: %w", err)
	}

	rule, err := s.q.CreateProcessingRule(ctx, storage.CreateProcessingRuleParams{
		ProjectID: pid,
		SourceID:  sid,
		Name:      name,
		Phase:     phase,
		RuleType:  ruleType,
		Config:    configJSON,
		Priority:  int32(priority),
		Enabled:   enabled,
	})
	if err != nil {
		return nil, fmt.Errorf("creating processing rule: %w", err)
	}

	return storageRuleToProcessingInfo(rule), nil
}

func (s *ProcessingRuleService) List(ctx context.Context, projectID string) ([]ProcessingRuleInfo, error) {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return nil, ErrProjectNotFound
	}

	rules, err := s.q.ListProcessingRules(ctx, pid)
	if err != nil {
		return nil, fmt.Errorf("listing processing rules: %w", err)
	}

	result := make([]ProcessingRuleInfo, len(rules))
	for i, r := range rules {
		result[i] = *storageRuleToProcessingInfo(r)
	}
	return result, nil
}

func (s *ProcessingRuleService) Get(ctx context.Context, projectID, ruleID string) (*ProcessingRuleInfo, error) {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return nil, ErrProcessingRuleNotFound
	}
	rid, err := uuid.Parse(ruleID)
	if err != nil {
		return nil, ErrProcessingRuleNotFound
	}

	rule, err := s.q.GetProcessingRuleByID(ctx, storage.GetProcessingRuleByIDParams{
		ID: rid, ProjectID: pid,
	})
	if err != nil {
		return nil, ErrProcessingRuleNotFound
	}

	return storageRuleToProcessingInfo(rule), nil
}

func (s *ProcessingRuleService) Update(ctx context.Context, projectID, ruleID, sourceID, name, phase, ruleType string, config map[string]any, priority int, enabled bool) (*ProcessingRuleInfo, error) {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return nil, ErrProcessingRuleNotFound
	}
	rid, err := uuid.Parse(ruleID)
	if err != nil {
		return nil, ErrProcessingRuleNotFound
	}

	if err := validatePhase(phase); err != nil {
		return nil, err
	}
	if err := validateRuleType(ruleType); err != nil {
		return nil, err
	}

	var sid pgtype.UUID
	if sourceID != "" {
		parsed, err := uuid.Parse(sourceID)
		if err != nil {
			return nil, fmt.Errorf("invalid source_id: %w", err)
		}
		sid = pgtype.UUID{Bytes: parsed, Valid: true}
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("marshaling config: %w", err)
	}

	rule, err := s.q.UpdateProcessingRule(ctx, storage.UpdateProcessingRuleParams{
		ID: rid, ProjectID: pid,
		SourceID: sid,
		Name:     name,
		Phase:    phase,
		RuleType: ruleType,
		Config:   configJSON,
		Priority: int32(priority),
		Enabled:  enabled,
	})
	if err != nil {
		return nil, ErrProcessingRuleNotFound
	}

	return storageRuleToProcessingInfo(rule), nil
}

func (s *ProcessingRuleService) Delete(ctx context.Context, projectID, ruleID string) error {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return ErrProcessingRuleNotFound
	}
	rid, err := uuid.Parse(ruleID)
	if err != nil {
		return ErrProcessingRuleNotFound
	}

	// Verify it exists.
	_, err = s.q.GetProcessingRuleByID(ctx, storage.GetProcessingRuleByIDParams{
		ID: rid, ProjectID: pid,
	})
	if err != nil {
		return ErrProcessingRuleNotFound
	}

	return s.q.DeleteProcessingRule(ctx, storage.DeleteProcessingRuleParams{
		ID: rid, ProjectID: pid,
	})
}

// --- Helpers ---

func validatePhase(phase string) error {
	switch phase {
	case "filter", "classify":
		return nil
	default:
		return ErrInvalidPhase
	}
}

func validateRuleType(ruleType string) error {
	switch ruleType {
	case "keyword", "regex", "llm":
		return nil
	default:
		return ErrInvalidRuleType
	}
}

func storageRuleToProcessingInfo(r storage.ProcessingRule) *ProcessingRuleInfo {
	var cfg map[string]any
	if len(r.Config) > 0 {
		json.Unmarshal(r.Config, &cfg)
	}
	if cfg == nil {
		cfg = map[string]any{}
	}

	sourceID := ""
	if r.SourceID.Valid {
		sourceID = uuid.UUID(r.SourceID.Bytes).String()
	}

	return &ProcessingRuleInfo{
		ID:        r.ID.String(),
		ProjectID: r.ProjectID.String(),
		SourceID:  sourceID,
		Name:      r.Name,
		Phase:     r.Phase,
		RuleType:  r.RuleType,
		Config:    cfg,
		Priority:  int(r.Priority),
		Enabled:   r.Enabled,
		CreatedAt: r.CreatedAt.Time,
		UpdatedAt: r.UpdatedAt.Time,
	}
}
