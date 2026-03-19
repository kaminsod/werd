import { useState, type FormEvent } from "react";
import { useParams, Link } from "react-router";
import { useRules, useCreateRule, useUpdateRule, useDeleteRule } from "@/hooks/use-rules";
import InfoIcon from "@/components/info-icon";
import { minSeverity as minSevHelp, ntfyTopic as ntfyHelp, webhookUrl as webhookHelp } from "@/lib/help-content";
import type { AlertSeverity, NotificationDestination, NotificationSourceType, Rule } from "@/types/api";

const SOURCE_TYPES: NotificationSourceType[] = ["all", "reddit", "hn", "web", "rss", "github"];
const SEVERITIES: AlertSeverity[] = ["low", "medium", "high", "critical"];
const DESTINATIONS: NotificationDestination[] = ["ntfy", "webhook", "email"];

const DEST_COLORS: Record<NotificationDestination, string> = {
  ntfy: "bg-indigo-100 text-indigo-700",
  webhook: "bg-teal-100 text-teal-700",
  email: "bg-gray-100 text-gray-500",
};

export default function RulesPage() {
  const { id: projectId } = useParams<{ id: string }>();
  const { data: rules, isLoading, error } = useRules(projectId!);
  const createRule = useCreateRule(projectId!);
  const updateRule = useUpdateRule(projectId!);
  const deleteRule = useDeleteRule(projectId!);

  const [showForm, setShowForm] = useState(false);
  const [editId, setEditId] = useState<string | null>(null);
  const [formSourceType, setFormSourceType] = useState<NotificationSourceType>("all");
  const [formSeverity, setFormSeverity] = useState<AlertSeverity>("low");
  const [formDest, setFormDest] = useState<NotificationDestination>("ntfy");
  const [formTopic, setFormTopic] = useState("");
  const [formUrl, setFormUrl] = useState("");
  const [formEnabled, setFormEnabled] = useState(true);

  function resetForm() {
    setFormSourceType("all");
    setFormSeverity("low");
    setFormDest("ntfy");
    setFormTopic("");
    setFormUrl("");
    setFormEnabled(true);
    setShowForm(false);
    setEditId(null);
  }

  function startEdit(rule: Rule) {
    setEditId(rule.id);
    setFormSourceType(rule.source_type);
    setFormSeverity(rule.min_severity);
    setFormDest(rule.destination);
    setFormTopic((rule.config as Record<string, string>).topic ?? "");
    setFormUrl((rule.config as Record<string, string>).url ?? "");
    setFormEnabled(rule.enabled);
    setShowForm(true);
  }

  function buildConfig(): Record<string, unknown> {
    if (formDest === "ntfy") return { topic: formTopic };
    if (formDest === "webhook") return { url: formUrl };
    return {};
  }

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    const data = {
      source_type: formSourceType,
      min_severity: formSeverity,
      destination: formDest,
      config: buildConfig(),
      enabled: formEnabled,
    };

    if (editId) {
      updateRule.mutate({ ruleId: editId, ...data }, { onSuccess: resetForm });
    } else {
      createRule.mutate(data, { onSuccess: resetForm });
    }
  }

  function handleDelete(ruleId: string) {
    if (confirm("Delete this notification rule?")) {
      deleteRule.mutate(ruleId);
    }
  }

  function configSummary(rule: Rule): string {
    const cfg = rule.config as Record<string, string>;
    if (rule.destination === "ntfy" && cfg.topic) return `topic: ${cfg.topic}`;
    if (rule.destination === "webhook" && cfg.url) return cfg.url;
    return JSON.stringify(rule.config);
  }

  if (isLoading) return <p className="text-gray-500">Loading rules...</p>;
  if (error) return <p className="text-red-600">Error: {error.message}</p>;

  return (
    <div>
      <div className="mb-4 flex items-center justify-between">
        <h2 className="text-xl font-semibold">Notification Rules</h2>
        <button
          onClick={() => { resetForm(); setShowForm(!showForm); }}
          className="rounded bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
        >
          {showForm && !editId ? "Cancel" : "Add Rule"}
        </button>
      </div>

      {showForm && (
        <form onSubmit={handleSubmit} className="mb-6 space-y-3 rounded border bg-gray-50 p-4">
          <h3 className="text-sm font-medium">{editId ? "Edit Rule" : "New Rule"}</h3>

          {(createRule.error || updateRule.error) && (
            <p className="text-sm text-red-600">
              {(createRule.error || updateRule.error)?.message}
            </p>
          )}

          <div className="flex flex-wrap gap-3">
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-600">Source type</label>
              <select value={formSourceType} onChange={(e) => setFormSourceType(e.target.value as NotificationSourceType)} className="rounded border px-3 py-2 text-sm">
                {SOURCE_TYPES.map((t) => <option key={t} value={t}>{t}</option>)}
              </select>
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-600">
                Min severity
                <InfoIcon tooltip={minSevHelp.tooltip}>{minSevHelp.modal}</InfoIcon>
              </label>
              <select value={formSeverity} onChange={(e) => setFormSeverity(e.target.value as AlertSeverity)} className="rounded border px-3 py-2 text-sm">
                {SEVERITIES.map((s) => <option key={s} value={s}>{s}</option>)}
              </select>
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-600">Destination</label>
              <select value={formDest} onChange={(e) => setFormDest(e.target.value as NotificationDestination)} className="rounded border px-3 py-2 text-sm">
                {DESTINATIONS.map((d) => <option key={d} value={d}>{d}</option>)}
              </select>
            </div>
            <label className="flex items-center gap-2 self-end">
              <input type="checkbox" checked={formEnabled} onChange={(e) => setFormEnabled(e.target.checked)} />
              <span className="text-sm">Enabled</span>
            </label>
          </div>

          {formDest === "ntfy" && (
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-600">
                ntfy topic
                <InfoIcon tooltip={ntfyHelp.tooltip}>{ntfyHelp.modal}</InfoIcon>
              </label>
              <input
                value={formTopic}
                onChange={(e) => setFormTopic(e.target.value)}
                required
                placeholder="werd-myproject-alerts"
                className="w-full rounded border px-3 py-2 text-sm"
              />
            </div>
          )}

          {formDest === "webhook" && (
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-600">
                Webhook URL
                <InfoIcon tooltip={webhookHelp.tooltip}>{webhookHelp.modal}</InfoIcon>
              </label>
              <input
                value={formUrl}
                onChange={(e) => setFormUrl(e.target.value)}
                required
                type="url"
                placeholder="https://example.com/webhook"
                className="w-full rounded border px-3 py-2 text-sm"
              />
            </div>
          )}

          {formDest === "email" && (
            <p className="text-xs text-gray-400">Email notifications are not yet implemented.</p>
          )}

          <button
            type="submit"
            disabled={createRule.isPending || updateRule.isPending}
            className="rounded bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
          >
            {editId ? "Update" : "Create"}
          </button>
          {editId && (
            <button type="button" onClick={resetForm} className="ml-2 text-sm text-gray-500 hover:text-gray-700">
              Cancel edit
            </button>
          )}
        </form>
      )}

      {rules!.length === 0 ? (
        <p className="text-gray-500">No notification rules configured. Add a rule to get notified when alerts match.</p>
      ) : (
        <div className="space-y-2">
          {rules!.map((rule) => (
            <div key={rule.id} className="flex items-center justify-between rounded border bg-white p-3">
              <div className="flex flex-wrap items-center gap-2">
                <span className={`rounded px-2 py-0.5 text-xs font-medium ${DEST_COLORS[rule.destination]}`}>
                  {rule.destination}
                </span>
                <span className="rounded bg-gray-50 px-2 py-0.5 text-xs text-gray-600">
                  {rule.source_type === "all" ? "all sources" : rule.source_type}
                </span>
                <span className="rounded bg-gray-50 px-2 py-0.5 text-xs text-gray-600">
                  &ge; {rule.min_severity}
                </span>
                <span className={`rounded px-2 py-0.5 text-xs ${rule.enabled ? "bg-green-50 text-green-700" : "bg-gray-100 text-gray-400"}`}>
                  {rule.enabled ? "enabled" : "disabled"}
                </span>
                <span className="text-xs text-gray-400 font-mono">{configSummary(rule)}</span>
              </div>
              <div className="flex items-center gap-2">
                <Link to={rule.id} className="rounded px-2 py-1 text-xs text-blue-600 hover:bg-blue-50">View</Link>
                <button onClick={() => startEdit(rule)} className="rounded px-2 py-1 text-xs text-blue-600 hover:bg-blue-50">
                  Edit
                </button>
                <button onClick={() => handleDelete(rule.id)} className="rounded px-2 py-1 text-xs text-red-600 hover:bg-red-50">
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
