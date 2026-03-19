import { useState, type FormEvent } from "react";
import { useParams, Link, useNavigate } from "react-router";
import { useProcessingRule, useUpdateProcessingRule, useDeleteProcessingRule } from "@/hooks/use-processing-rules";
import { useSources } from "@/hooks/use-sources";
import type { ProcessingPhase, ProcessingRuleType } from "@/types/api";

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
  keywords: string;
  matchType: string;
  fields: string;
  action: string;
  pattern: string;
  setSeverity: string;
  addTags: string;
  promptTemplate: string;
  maxTokens: number;
  onlyIfKeywords: boolean;
}

const defaultForm: RuleFormState = {
  name: "", sourceId: "", phase: "filter", ruleType: "keyword", priority: 0, enabled: true,
  keywords: "", matchType: "substring", fields: "title,content", action: "include",
  pattern: "", setSeverity: "", addTags: "", promptTemplate: "", maxTokens: 200, onlyIfKeywords: true,
};

function formToConfig(form: RuleFormState): Record<string, unknown> {
  if (form.ruleType === "keyword") {
    const cfg: Record<string, unknown> = {
      keywords: form.keywords.split(",").map((k) => k.trim()).filter(Boolean),
      match_type: form.matchType,
      fields: form.fields.split(",").map((f) => f.trim()).filter(Boolean),
    };
    if (form.phase === "filter") cfg.action = form.action;
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
    return { prompt_template: form.promptTemplate, max_tokens: form.maxTokens, only_if_keywords: form.onlyIfKeywords };
  }
  return {};
}

function configToForm(rule: { name: string; source_id?: string; phase: ProcessingPhase; rule_type: ProcessingRuleType; priority: number; enabled: boolean; config: Record<string, unknown> }): RuleFormState {
  const cfg = rule.config;
  return {
    name: rule.name, sourceId: rule.source_id || "", phase: rule.phase, ruleType: rule.rule_type, priority: rule.priority, enabled: rule.enabled,
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

export default function ProcessingRuleDetailPage() {
  const { id: projectId, ruleId } = useParams<{ id: string; ruleId: string }>();
  const navigate = useNavigate();
  const { data: rule, isLoading, error } = useProcessingRule(projectId!, ruleId!);
  const { data: sources } = useSources(projectId!);
  const updateRule = useUpdateProcessingRule(projectId!);
  const deleteRule = useDeleteProcessingRule(projectId!);

  const [editing, setEditing] = useState(false);
  const [form, setForm] = useState<RuleFormState>(defaultForm);

  function startEdit() {
    if (!rule) return;
    setForm(configToForm(rule));
    setEditing(true);
  }

  function handleSave(e: FormEvent) {
    e.preventDefault();
    updateRule.mutate(
      {
        ruleId: ruleId!,
        source_id: form.sourceId || undefined,
        name: form.name,
        phase: form.phase,
        rule_type: form.ruleType,
        config: formToConfig(form),
        priority: form.priority,
        enabled: form.enabled,
      },
      { onSuccess: () => setEditing(false) },
    );
  }

  function handleDelete() {
    if (confirm(`Delete processing rule "${rule?.name || ruleId}"?`)) {
      deleteRule.mutate(ruleId!, { onSuccess: () => navigate("../processing") });
    }
  }

  if (isLoading) return <p className="text-gray-500">Loading processing rule...</p>;
  if (error) return <p className="text-red-600">Error: {error.message}</p>;
  if (!rule) return <p className="text-gray-500">Processing rule not found.</p>;

  const sourceLabel = rule.source_id
    ? sources?.find((s) => s.id === rule.source_id)
    : null;

  return (
    <div>
      <Link to="../processing" className="mb-4 inline-block text-sm text-blue-600 hover:underline">&larr; Back to Processing Rules</Link>

      <div className="mb-4 flex items-center gap-3">
        <h2 className="text-xl font-semibold">{rule.name || "(unnamed)"}</h2>
        <span className={`rounded px-2 py-0.5 text-xs font-medium ${PHASE_COLORS[rule.phase]}`}>{rule.phase}</span>
        <span className={`rounded px-2 py-0.5 text-xs font-medium ${RULE_TYPE_COLORS[rule.rule_type]}`}>{rule.rule_type}</span>
        {!rule.enabled && <span className="rounded bg-gray-100 px-2 py-0.5 text-xs text-gray-400">disabled</span>}
      </div>

      {editing ? (
        <form onSubmit={handleSave} className="space-y-4 rounded border bg-white p-4">
          <h3 className="font-medium">Edit Rule</h3>
          {updateRule.error && <p className="text-sm text-red-600">{updateRule.error.message}</p>}

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="mb-1 block text-sm font-medium text-gray-700">Name</label>
              <input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} className="w-full rounded border px-3 py-2 text-sm" />
            </div>
            <div>
              <label className="mb-1 block text-sm font-medium text-gray-700">Source (optional)</label>
              <select value={form.sourceId} onChange={(e) => setForm({ ...form, sourceId: e.target.value })} className="w-full rounded border px-3 py-2 text-sm">
                <option value="">All sources (project-wide)</option>
                {sources?.map((s) => <option key={s.id} value={s.id}>{s.type}: {JSON.stringify(s.config).slice(0, 40)}</option>)}
              </select>
            </div>
          </div>

          <div className="grid grid-cols-3 gap-4">
            <div>
              <label className="mb-1 block text-sm font-medium text-gray-700">Phase</label>
              <select value={form.phase} onChange={(e) => setForm({ ...form, phase: e.target.value as ProcessingPhase })} className="w-full rounded border px-3 py-2 text-sm">
                {PHASES.map((p) => <option key={p} value={p}>{p}</option>)}
              </select>
            </div>
            <div>
              <label className="mb-1 block text-sm font-medium text-gray-700">Rule Type</label>
              <select value={form.ruleType} onChange={(e) => setForm({ ...form, ruleType: e.target.value as ProcessingRuleType })} className="w-full rounded border px-3 py-2 text-sm">
                {RULE_TYPES.map((t) => <option key={t} value={t}>{t}</option>)}
              </select>
            </div>
            <div>
              <label className="mb-1 block text-sm font-medium text-gray-700">Priority</label>
              <input type="number" value={form.priority} onChange={(e) => setForm({ ...form, priority: parseInt(e.target.value) || 0 })} className="w-full rounded border px-3 py-2 text-sm" />
            </div>
          </div>

          {form.ruleType === "keyword" && (
            <div className="space-y-3">
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700">Keywords (comma-separated)</label>
                <input value={form.keywords} onChange={(e) => setForm({ ...form, keywords: e.target.value })} required className="w-full rounded border px-3 py-2 text-sm" />
              </div>
              <div className="grid grid-cols-3 gap-4">
                <div>
                  <label className="mb-1 block text-sm font-medium text-gray-700">Match Type</label>
                  <select value={form.matchType} onChange={(e) => setForm({ ...form, matchType: e.target.value })} className="w-full rounded border px-3 py-2 text-sm">
                    <option value="substring">substring</option><option value="exact">exact</option><option value="regex">regex</option>
                  </select>
                </div>
                <div>
                  <label className="mb-1 block text-sm font-medium text-gray-700">Fields</label>
                  <input value={form.fields} onChange={(e) => setForm({ ...form, fields: e.target.value })} className="w-full rounded border px-3 py-2 text-sm" />
                </div>
                {form.phase === "filter" && (
                  <div>
                    <label className="mb-1 block text-sm font-medium text-gray-700">Action</label>
                    <select value={form.action} onChange={(e) => setForm({ ...form, action: e.target.value })} className="w-full rounded border px-3 py-2 text-sm">
                      <option value="include">include</option><option value="exclude">exclude</option>
                    </select>
                  </div>
                )}
              </div>
              {form.phase === "classify" && (
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="mb-1 block text-sm font-medium text-gray-700">Set Severity</label>
                    <select value={form.setSeverity} onChange={(e) => setForm({ ...form, setSeverity: e.target.value })} className="w-full rounded border px-3 py-2 text-sm">
                      <option value="">default (low)</option><option value="low">low</option><option value="medium">medium</option><option value="high">high</option><option value="critical">critical</option>
                    </select>
                  </div>
                  <div>
                    <label className="mb-1 block text-sm font-medium text-gray-700">Add Tags (comma-separated)</label>
                    <input value={form.addTags} onChange={(e) => setForm({ ...form, addTags: e.target.value })} className="w-full rounded border px-3 py-2 text-sm" />
                  </div>
                </div>
              )}
            </div>
          )}

          {form.ruleType === "regex" && (
            <div className="space-y-3">
              <div>
                <label className="mb-1 block text-sm font-medium text-gray-700">Regex Pattern</label>
                <input value={form.pattern} onChange={(e) => setForm({ ...form, pattern: e.target.value })} required className="w-full rounded border px-3 py-2 font-mono text-sm" />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="mb-1 block text-sm font-medium text-gray-700">Fields</label>
                  <input value={form.fields} onChange={(e) => setForm({ ...form, fields: e.target.value })} className="w-full rounded border px-3 py-2 text-sm" />
                </div>
                {form.phase === "filter" && (
                  <div>
                    <label className="mb-1 block text-sm font-medium text-gray-700">Action</label>
                    <select value={form.action} onChange={(e) => setForm({ ...form, action: e.target.value })} className="w-full rounded border px-3 py-2 text-sm">
                      <option value="include">include</option><option value="exclude">exclude</option>
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
                <textarea value={form.promptTemplate} onChange={(e) => setForm({ ...form, promptTemplate: e.target.value })} rows={5} required className="w-full rounded border px-3 py-2 font-mono text-sm" />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="mb-1 block text-sm font-medium text-gray-700">Max Tokens</label>
                  <input type="number" value={form.maxTokens} onChange={(e) => setForm({ ...form, maxTokens: parseInt(e.target.value) || 200 })} className="w-full rounded border px-3 py-2 text-sm" />
                </div>
                <div className="flex items-center gap-2 pt-6">
                  <input type="checkbox" checked={form.onlyIfKeywords} onChange={(e) => setForm({ ...form, onlyIfKeywords: e.target.checked })} id="editOnlyIfKeywords" />
                  <label htmlFor="editOnlyIfKeywords" className="text-sm text-gray-700">Only run if keywords matched</label>
                </div>
              </div>
            </div>
          )}

          <div className="flex items-center gap-2">
            <input type="checkbox" checked={form.enabled} onChange={(e) => setForm({ ...form, enabled: e.target.checked })} id="editRuleEnabled" />
            <label htmlFor="editRuleEnabled" className="text-sm text-gray-700">Enabled</label>
          </div>

          <div className="flex gap-2">
            <button type="submit" disabled={updateRule.isPending} className="rounded bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50">Update</button>
            <button type="button" onClick={() => setEditing(false)} className="rounded border px-4 py-2 text-sm text-gray-600 hover:bg-gray-50">Cancel</button>
          </div>
        </form>
      ) : (
        <div className="space-y-4 rounded border bg-white p-4">
          <div className="grid grid-cols-3 gap-4 text-sm">
            <div>
              <span className="font-medium text-gray-500">Priority</span>
              <p className="mt-1">{rule.priority}</p>
            </div>
            <div>
              <span className="font-medium text-gray-500">Enabled</span>
              <p className="mt-1">{rule.enabled ? "Yes" : "No"}</p>
            </div>
            <div>
              <span className="font-medium text-gray-500">Source</span>
              <p className="mt-1">
                {sourceLabel ? (
                  <Link to={`../sources/${rule.source_id}`} className="text-blue-600 hover:underline">
                    {sourceLabel.type}: {JSON.stringify(sourceLabel.config).slice(0, 40)}
                  </Link>
                ) : (
                  "All sources (project-wide)"
                )}
              </p>
            </div>
          </div>

          <div>
            <span className="text-sm font-medium text-gray-500">Config</span>
            {rule.rule_type === "keyword" && (
              <div className="mt-2 space-y-2 text-sm">
                <div>
                  <span className="text-gray-500">Keywords: </span>
                  <span className="flex flex-wrap gap-1 mt-1">
                    {(rule.config.keywords as string[] || []).map((kw) => (
                      <span key={kw} className="rounded bg-yellow-50 px-2 py-0.5 text-xs text-yellow-700">{kw}</span>
                    ))}
                  </span>
                </div>
                <p><span className="text-gray-500">Match type: </span>{rule.config.match_type as string}</p>
                <p><span className="text-gray-500">Fields: </span>{(rule.config.fields as string[] || []).join(", ")}</p>
                {!!rule.config.action && <p><span className="text-gray-500">Action: </span>{String(rule.config.action)}</p>}
                {!!rule.config.set_severity && <p><span className="text-gray-500">Set severity: </span>{String(rule.config.set_severity)}</p>}
                {!!rule.config.add_tags && (
                  <div>
                    <span className="text-gray-500">Add tags: </span>
                    {(rule.config.add_tags as string[]).map((t) => (
                      <span key={t} className="mr-1 rounded bg-blue-50 px-1.5 py-0.5 text-xs text-blue-700">{t}</span>
                    ))}
                  </div>
                )}
              </div>
            )}
            {rule.rule_type === "regex" && (
              <div className="mt-2 space-y-2 text-sm">
                <p><span className="text-gray-500">Pattern: </span><code className="rounded bg-gray-100 px-1.5 py-0.5 font-mono text-xs">{String(rule.config.pattern)}</code></p>
                <p><span className="text-gray-500">Fields: </span>{(rule.config.fields as string[] || []).join(", ")}</p>
                {!!rule.config.action && <p><span className="text-gray-500">Action: </span>{String(rule.config.action)}</p>}
              </div>
            )}
            {rule.rule_type === "llm" && (
              <div className="mt-2 space-y-2 text-sm">
                <div>
                  <span className="text-gray-500">Prompt template:</span>
                  <pre className="mt-1 overflow-x-auto rounded bg-gray-50 p-3 font-mono text-xs text-gray-700">{rule.config.prompt_template as string}</pre>
                </div>
                <p><span className="text-gray-500">Max tokens: </span>{rule.config.max_tokens as number}</p>
                <p><span className="text-gray-500">Only if keywords matched: </span>{rule.config.only_if_keywords !== false ? "Yes" : "No"}</p>
              </div>
            )}
          </div>

          <div className="grid grid-cols-2 gap-4 text-sm">
            <div>
              <span className="font-medium text-gray-500">Created</span>
              <p className="mt-1">{new Date(rule.created_at).toLocaleString()}</p>
            </div>
            <div>
              <span className="font-medium text-gray-500">Updated</span>
              <p className="mt-1">{new Date(rule.updated_at).toLocaleString()}</p>
            </div>
          </div>

          <div className="flex gap-2 border-t pt-4">
            <button onClick={startEdit} className="rounded px-2 py-1 text-xs text-blue-600 hover:bg-blue-50">Edit</button>
            <button onClick={handleDelete} className="rounded px-2 py-1 text-xs text-red-600 hover:bg-red-50">Delete</button>
          </div>
        </div>
      )}
    </div>
  );
}
