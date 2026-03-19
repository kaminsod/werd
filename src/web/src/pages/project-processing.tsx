import { useState, type FormEvent } from "react";
import { useParams, Link } from "react-router";
import {
  useProcessingRules,
  useCreateProcessingRule,
  useUpdateProcessingRule,
  useDeleteProcessingRule,
} from "@/hooks/use-processing-rules";
import { useSources } from "@/hooks/use-sources";
import type { ProcessingRule, ProcessingPhase, ProcessingRuleType } from "@/types/api";

const PHASES: ProcessingPhase[] = ["filter", "classify"];
const RULE_TYPES: ProcessingRuleType[] = ["keyword", "regex", "llm"];

const PHASE_COLORS: Record<ProcessingPhase, string> = {
  filter: "bg-blue-100 text-blue-700",
  classify: "bg-purple-100 text-purple-700",
};

const RULE_TYPE_COLORS: Record<ProcessingRuleType, string> = {
  keyword: "bg-green-100 text-green-700",
  regex: "bg-yellow-100 text-yellow-700",
  llm: "bg-indigo-100 text-indigo-700",
};

interface RuleFormState {
  name: string;
  sourceId: string;
  phase: ProcessingPhase;
  ruleType: ProcessingRuleType;
  priority: number;
  enabled: boolean;
  // Keyword filter/classify fields
  keywords: string;
  matchType: string;
  fields: string;
  action: string;
  // Regex fields
  pattern: string;
  // Classify keyword fields
  setSeverity: string;
  addTags: string;
  // LLM fields
  promptTemplate: string;
  maxTokens: number;
  onlyIfKeywords: boolean;
}

const defaultForm: RuleFormState = {
  name: "",
  sourceId: "",
  phase: "filter",
  ruleType: "keyword",
  priority: 0,
  enabled: true,
  keywords: "",
  matchType: "substring",
  fields: "title,content",
  action: "include",
  pattern: "",
  setSeverity: "",
  addTags: "",
  promptTemplate: "",
  maxTokens: 200,
  onlyIfKeywords: true,
};

function formToConfig(form: RuleFormState): Record<string, unknown> {
  if (form.ruleType === "keyword") {
    const cfg: Record<string, unknown> = {
      keywords: form.keywords.split(",").map((k) => k.trim()).filter(Boolean),
      match_type: form.matchType,
      fields: form.fields.split(",").map((f) => f.trim()).filter(Boolean),
    };
    if (form.phase === "filter") {
      cfg.action = form.action;
    }
    if (form.phase === "classify") {
      if (form.setSeverity) cfg.set_severity = form.setSeverity;
      if (form.addTags) cfg.add_tags = form.addTags.split(",").map((t) => t.trim()).filter(Boolean);
    }
    return cfg;
  }

  if (form.ruleType === "regex") {
    return {
      pattern: form.pattern,
      fields: form.fields.split(",").map((f) => f.trim()).filter(Boolean),
      action: form.phase === "filter" ? form.action : undefined,
    };
  }

  if (form.ruleType === "llm") {
    return {
      prompt_template: form.promptTemplate,
      max_tokens: form.maxTokens,
      only_if_keywords: form.onlyIfKeywords,
    };
  }

  return {};
}

function configToForm(rule: ProcessingRule): RuleFormState {
  const cfg = rule.config;
  return {
    name: rule.name,
    sourceId: rule.source_id || "",
    phase: rule.phase,
    ruleType: rule.rule_type,
    priority: rule.priority,
    enabled: rule.enabled,
    keywords: Array.isArray(cfg.keywords) ? (cfg.keywords as string[]).join(", ") : "",
    matchType: (cfg.match_type as string) || "substring",
    fields: Array.isArray(cfg.fields) ? (cfg.fields as string[]).join(", ") : "title, content",
    action: (cfg.action as string) || "include",
    pattern: (cfg.pattern as string) || "",
    setSeverity: (cfg.set_severity as string) || "",
    addTags: Array.isArray(cfg.add_tags) ? (cfg.add_tags as string[]).join(", ") : "",
    promptTemplate: (cfg.prompt_template as string) || "",
    maxTokens: (cfg.max_tokens as number) || 200,
    onlyIfKeywords: cfg.only_if_keywords !== false,
  };
}

export default function ProcessingPage() {
  const { id: projectId } = useParams<{ id: string }>();
  const { data: rules, isLoading, error } = useProcessingRules(projectId!);
  const { data: sources } = useSources(projectId!);
  const createRule = useCreateProcessingRule(projectId!);
  const updateRule = useUpdateProcessingRule(projectId!);
  const deleteRule = useDeleteProcessingRule(projectId!);

  const [showForm, setShowForm] = useState(false);
  const [editId, setEditId] = useState<string | null>(null);
  const [form, setForm] = useState<RuleFormState>(defaultForm);

  function handleCreate(e: FormEvent) {
    e.preventDefault();
    createRule.mutate(
      {
        source_id: form.sourceId || undefined,
        name: form.name,
        phase: form.phase,
        rule_type: form.ruleType,
        config: formToConfig(form),
        priority: form.priority,
        enabled: form.enabled,
      },
      {
        onSuccess: () => {
          setForm(defaultForm);
          setShowForm(false);
        },
      },
    );
  }

  function handleUpdate(e: FormEvent) {
    e.preventDefault();
    if (!editId) return;
    updateRule.mutate(
      {
        ruleId: editId,
        source_id: form.sourceId || undefined,
        name: form.name,
        phase: form.phase,
        rule_type: form.ruleType,
        config: formToConfig(form),
        priority: form.priority,
        enabled: form.enabled,
      },
      {
        onSuccess: () => {
          setForm(defaultForm);
          setEditId(null);
        },
      },
    );
  }

  function handleEdit(rule: ProcessingRule) {
    setForm(configToForm(rule));
    setEditId(rule.id);
    setShowForm(false);
  }

  function handleDelete(ruleId: string, name: string) {
    if (confirm(`Delete processing rule "${name || ruleId}"?`)) {
      deleteRule.mutate(ruleId);
    }
  }

  function handleCancel() {
    setForm(defaultForm);
    setEditId(null);
    setShowForm(false);
  }

  if (isLoading) return <p className="text-gray-500">Loading processing rules...</p>;
  if (error) return <p className="text-red-600">Error: {error.message}</p>;

  const isEditing = editId !== null;

  return (
    <div>
      <h2 className="mb-2 text-xl font-semibold">Processing Rules</h2>
      <p className="mb-4 text-sm text-gray-500">
        Processing rules sit between monitor polling and alert ingestion. <strong>Filter</strong> rules
        control which items become alerts. <strong>Classify</strong> rules enrich alerts with severity,
        tags, and classification reasons. Rules can be scoped to a specific source or apply project-wide.
      </p>

      {!showForm && !isEditing && (
        <button
          onClick={() => setShowForm(true)}
          className="mb-4 rounded bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
        >
          Add Rule
        </button>
      )}

      {(showForm || isEditing) && (
        <form onSubmit={isEditing ? handleUpdate : handleCreate} className="mb-6 space-y-4 rounded border bg-white p-4">
          <h3 className="font-medium">{isEditing ? "Edit Rule" : "New Processing Rule"}</h3>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="mb-1 block text-sm font-medium text-gray-700">Name</label>
              <input
                value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
                placeholder="e.g. security-filter"
                className="w-full rounded border px-3 py-2 text-sm"
              />
            </div>
            <div>
              <label className="mb-1 block text-sm font-medium text-gray-700">Source (optional)</label>
              <select
                value={form.sourceId}
                onChange={(e) => setForm({ ...form, sourceId: e.target.value })}
                className="w-full rounded border px-3 py-2 text-sm"
              >
                <option value="">All sources (project-wide)</option>
                {sources?.map((s) => (
                  <option key={s.id} value={s.id}>
                    {s.type}: {JSON.stringify(s.config).slice(0, 40)}
                  </option>
                ))}
              </select>
            </div>
          </div>

          <div className="grid grid-cols-3 gap-4">
            <div>
              <label className="mb-1 block text-sm font-medium text-gray-700">Phase</label>
              <select
                value={form.phase}
                onChange={(e) => setForm({ ...form, phase: e.target.value as ProcessingPhase })}
                className="w-full rounded border px-3 py-2 text-sm"
              >
                {PHASES.map((p) => (
                  <option key={p} value={p}>{p}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="mb-1 block text-sm font-medium text-gray-700">Rule Type</label>
              <select
                value={form.ruleType}
                onChange={(e) => setForm({ ...form, ruleType: e.target.value as ProcessingRuleType })}
                className="w-full rounded border px-3 py-2 text-sm"
              >
                {RULE_TYPES.map((t) => (
                  <option key={t} value={t}>{t}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="mb-1 block text-sm font-medium text-gray-700">Priority</label>
              <input
                type="number"
                value={form.priority}
                onChange={(e) => setForm({ ...form, priority: parseInt(e.target.value) || 0 })}
                className="w-full rounded border px-3 py-2 text-sm"
              />
            </div>
          </div>

          {/* Dynamic config fields based on rule type */}
          {form.ruleType === "keyword" && (
            <div className="space-y-3">
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700">Keywords (comma-separated)</label>
                <input
                  value={form.keywords}
                  onChange={(e) => setForm({ ...form, keywords: e.target.value })}
                  placeholder="golang, self-hosted, security"
                  required
                  className="w-full rounded border px-3 py-2 text-sm"
                />
              </div>
              <div className="grid grid-cols-3 gap-4">
                <div>
                  <label className="mb-1 block text-sm font-medium text-gray-700">Match Type</label>
                  <select
                    value={form.matchType}
                    onChange={(e) => setForm({ ...form, matchType: e.target.value })}
                    className="w-full rounded border px-3 py-2 text-sm"
                  >
                    <option value="substring">substring</option>
                    <option value="exact">exact</option>
                    <option value="regex">regex</option>
                  </select>
                </div>
                <div>
                  <label className="mb-1 block text-sm font-medium text-gray-700">Fields</label>
                  <input
                    value={form.fields}
                    onChange={(e) => setForm({ ...form, fields: e.target.value })}
                    placeholder="title, content"
                    className="w-full rounded border px-3 py-2 text-sm"
                  />
                </div>
                {form.phase === "filter" && (
                  <div>
                    <label className="mb-1 block text-sm font-medium text-gray-700">Action</label>
                    <select
                      value={form.action}
                      onChange={(e) => setForm({ ...form, action: e.target.value })}
                      className="w-full rounded border px-3 py-2 text-sm"
                    >
                      <option value="include">include</option>
                      <option value="exclude">exclude</option>
                    </select>
                  </div>
                )}
              </div>
              {form.phase === "classify" && (
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="mb-1 block text-sm font-medium text-gray-700">Set Severity</label>
                    <select
                      value={form.setSeverity}
                      onChange={(e) => setForm({ ...form, setSeverity: e.target.value })}
                      className="w-full rounded border px-3 py-2 text-sm"
                    >
                      <option value="">default (low)</option>
                      <option value="low">low</option>
                      <option value="medium">medium</option>
                      <option value="high">high</option>
                      <option value="critical">critical</option>
                    </select>
                  </div>
                  <div>
                    <label className="mb-1 block text-sm font-medium text-gray-700">Add Tags (comma-separated)</label>
                    <input
                      value={form.addTags}
                      onChange={(e) => setForm({ ...form, addTags: e.target.value })}
                      placeholder="security, urgent"
                      className="w-full rounded border px-3 py-2 text-sm"
                    />
                  </div>
                </div>
              )}
            </div>
          )}

          {form.ruleType === "regex" && (
            <div className="space-y-3">
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700">Regex Pattern</label>
                <input
                  value={form.pattern}
                  onChange={(e) => setForm({ ...form, pattern: e.target.value })}
                  placeholder="(?i)\b(security|CVE-\d+)\b"
                  required
                  className="w-full rounded border px-3 py-2 font-mono text-sm"
                />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="mb-1 block text-sm font-medium text-gray-700">Fields</label>
                  <input
                    value={form.fields}
                    onChange={(e) => setForm({ ...form, fields: e.target.value })}
                    placeholder="title, content"
                    className="w-full rounded border px-3 py-2 text-sm"
                  />
                </div>
                {form.phase === "filter" && (
                  <div>
                    <label className="mb-1 block text-sm font-medium text-gray-700">Action</label>
                    <select
                      value={form.action}
                      onChange={(e) => setForm({ ...form, action: e.target.value })}
                      className="w-full rounded border px-3 py-2 text-sm"
                    >
                      <option value="include">include</option>
                      <option value="exclude">exclude</option>
                    </select>
                  </div>
                )}
              </div>
            </div>
          )}

          {form.ruleType === "llm" && (
            <div className="space-y-3">
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700">Prompt Template</label>
                <textarea
                  value={form.promptTemplate}
                  onChange={(e) => setForm({ ...form, promptTemplate: e.target.value })}
                  placeholder={'Analyze this {{source_type}} post for relevance...\nTitle: {{title}}\nContent: {{content}}\n\nRespond with JSON: {"relevant": bool, "severity": "low|medium|high|critical", "tags": [...], "reason": "..."}'}
                  rows={5}
                  required
                  className="w-full rounded border px-3 py-2 font-mono text-sm"
                />
                <p className="mt-1 text-xs text-gray-400">
                  Available variables: {"{{title}}"}, {"{{content}}"}, {"{{url}}"}, {"{{author}}"}, {"{{source_type}}"}
                </p>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="mb-1 block text-sm font-medium text-gray-700">Max Tokens</label>
                  <input
                    type="number"
                    value={form.maxTokens}
                    onChange={(e) => setForm({ ...form, maxTokens: parseInt(e.target.value) || 200 })}
                    className="w-full rounded border px-3 py-2 text-sm"
                  />
                </div>
                <div className="flex items-center gap-2 pt-6">
                  <input
                    type="checkbox"
                    checked={form.onlyIfKeywords}
                    onChange={(e) => setForm({ ...form, onlyIfKeywords: e.target.checked })}
                    id="onlyIfKeywords"
                  />
                  <label htmlFor="onlyIfKeywords" className="text-sm text-gray-700">
                    Only run if keywords matched (cost control)
                  </label>
                </div>
              </div>
            </div>
          )}

          <div className="flex items-center gap-2">
            <input
              type="checkbox"
              checked={form.enabled}
              onChange={(e) => setForm({ ...form, enabled: e.target.checked })}
              id="ruleEnabled"
            />
            <label htmlFor="ruleEnabled" className="text-sm text-gray-700">Enabled</label>
          </div>

          {(createRule.error || updateRule.error) && (
            <p className="text-sm text-red-600">{(createRule.error || updateRule.error)?.message}</p>
          )}

          <div className="flex gap-2">
            <button
              type="submit"
              disabled={createRule.isPending || updateRule.isPending}
              className="rounded bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
            >
              {isEditing ? "Update" : "Create"}
            </button>
            <button
              type="button"
              onClick={handleCancel}
              className="rounded border px-4 py-2 text-sm text-gray-600 hover:bg-gray-50"
            >
              Cancel
            </button>
          </div>
        </form>
      )}

      {/* Rules list */}
      {rules!.length === 0 ? (
        <p className="text-gray-500">No processing rules. All monitored items will be ingested as alerts.</p>
      ) : (
        <div className="space-y-2">
          {rules!.map((rule) => (
            <div key={rule.id} className={`rounded border bg-white p-3 ${!rule.enabled ? "opacity-50" : ""}`}>
              <div className="flex items-center gap-3">
                <span className={`shrink-0 rounded px-2 py-0.5 text-xs font-medium ${PHASE_COLORS[rule.phase]}`}>
                  {rule.phase}
                </span>
                <span className={`shrink-0 rounded px-2 py-0.5 text-xs font-medium ${RULE_TYPE_COLORS[rule.rule_type]}`}>
                  {rule.rule_type}
                </span>
                <Link to={rule.id} className="min-w-0 flex-1 text-sm font-medium text-blue-600 hover:underline">
                  {rule.name || "(unnamed)"}
                </Link>
                {rule.source_id && (
                  <span className="shrink-0 rounded bg-gray-50 px-2 py-0.5 text-xs text-gray-500">
                    source-scoped
                  </span>
                )}
                <span className="shrink-0 text-xs text-gray-400">pri: {rule.priority}</span>
                <ConfigSummary rule={rule} />
                <button
                  onClick={() => handleEdit(rule)}
                  className="shrink-0 rounded px-2 py-1 text-xs text-blue-600 hover:bg-blue-50"
                >
                  Edit
                </button>
                <button
                  onClick={() => handleDelete(rule.id, rule.name)}
                  disabled={deleteRule.isPending}
                  className="shrink-0 rounded px-2 py-1 text-xs text-red-600 hover:bg-red-50"
                >
                  Delete
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function ConfigSummary({ rule }: { rule: ProcessingRule }) {
  const cfg = rule.config;

  if (rule.rule_type === "keyword") {
    const keywords = cfg.keywords as string[] | undefined;
    const action = (cfg.action as string) || (rule.phase === "filter" ? "include" : "");
    return (
      <span className="shrink-0 text-xs text-gray-400">
        {action && `${action}: `}
        {keywords?.slice(0, 3).join(", ")}
        {keywords && keywords.length > 3 && ` +${keywords.length - 3}`}
      </span>
    );
  }

  if (rule.rule_type === "regex") {
    const pattern = cfg.pattern as string;
    return (
      <span className="shrink-0 font-mono text-xs text-gray-400">
        /{pattern?.slice(0, 30)}{pattern && pattern.length > 30 ? "..." : ""}/
      </span>
    );
  }

  if (rule.rule_type === "llm") {
    return (
      <span className="shrink-0 text-xs text-gray-400">
        LLM classify
      </span>
    );
  }

  return null;
}
