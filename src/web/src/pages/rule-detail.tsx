import { useState, type FormEvent } from "react";
import { useParams, Link, useNavigate } from "react-router";
import { useRule, useUpdateRule, useDeleteRule } from "@/hooks/use-rules";
import type { AlertSeverity, NotificationDestination, NotificationSourceType } from "@/types/api";

const SOURCE_TYPES: NotificationSourceType[] = ["all", "reddit", "hn", "web", "rss", "github"];
const SEVERITIES: AlertSeverity[] = ["low", "medium", "high", "critical"];
const DESTINATIONS: NotificationDestination[] = ["ntfy", "webhook", "email"];

const DEST_COLORS: Record<NotificationDestination, string> = {
  ntfy: "bg-indigo-100 text-indigo-700",
  webhook: "bg-teal-100 text-teal-700",
  email: "bg-gray-100 text-gray-500",
};

export default function RuleDetailPage() {
  const { id: projectId, ruleId } = useParams<{ id: string; ruleId: string }>();
  const navigate = useNavigate();
  const { data: rule, isLoading, error } = useRule(projectId!, ruleId!);
  const updateRule = useUpdateRule(projectId!);
  const deleteRule = useDeleteRule(projectId!);

  const [editing, setEditing] = useState(false);
  const [formSourceType, setFormSourceType] = useState<NotificationSourceType>("all");
  const [formSeverity, setFormSeverity] = useState<AlertSeverity>("low");
  const [formDest, setFormDest] = useState<NotificationDestination>("ntfy");
  const [formTopic, setFormTopic] = useState("");
  const [formUrl, setFormUrl] = useState("");
  const [formEnabled, setFormEnabled] = useState(true);

  function startEdit() {
    if (!rule) return;
    setFormSourceType(rule.source_type);
    setFormSeverity(rule.min_severity);
    setFormDest(rule.destination);
    setFormTopic((rule.config as Record<string, string>).topic ?? "");
    setFormUrl((rule.config as Record<string, string>).url ?? "");
    setFormEnabled(rule.enabled);
    setEditing(true);
  }

  function buildConfig(): Record<string, unknown> {
    if (formDest === "ntfy") return { topic: formTopic };
    if (formDest === "webhook") return { url: formUrl };
    return {};
  }

  function handleSave(e: FormEvent) {
    e.preventDefault();
    updateRule.mutate(
      { ruleId: ruleId!, source_type: formSourceType, min_severity: formSeverity, destination: formDest, config: buildConfig(), enabled: formEnabled },
      { onSuccess: () => setEditing(false) },
    );
  }

  function handleDelete() {
    if (confirm("Delete this notification rule?")) {
      deleteRule.mutate(ruleId!, { onSuccess: () => navigate("../rules") });
    }
  }

  function configSummary(): string {
    if (!rule) return "";
    const cfg = rule.config as Record<string, string>;
    if (rule.destination === "ntfy" && cfg.topic) return `topic: ${cfg.topic}`;
    if (rule.destination === "webhook" && cfg.url) return cfg.url;
    return JSON.stringify(rule.config);
  }

  if (isLoading) return <p className="text-gray-500">Loading rule...</p>;
  if (error) return <p className="text-red-600">Error: {error.message}</p>;
  if (!rule) return <p className="text-gray-500">Notification rule not found.</p>;

  return (
    <div>
      <Link to="../rules" className="mb-4 inline-block text-sm text-blue-600 hover:underline">&larr; Back to Notification Rules</Link>

      <div className="mb-4 flex items-center gap-3">
        <h2 className="text-xl font-semibold">Notification Rule</h2>
        <span className={`rounded px-2 py-0.5 text-xs font-medium ${DEST_COLORS[rule.destination]}`}>{rule.destination}</span>
        <span className={`rounded px-2 py-0.5 text-xs ${rule.enabled ? "bg-green-50 text-green-700" : "bg-gray-100 text-gray-400"}`}>
          {rule.enabled ? "enabled" : "disabled"}
        </span>
      </div>

      {editing ? (
        <form onSubmit={handleSave} className="space-y-3 rounded border bg-gray-50 p-4">
          <h3 className="text-sm font-medium">Edit Rule</h3>
          {updateRule.error && <p className="text-sm text-red-600">{updateRule.error.message}</p>}

          <div className="flex flex-wrap gap-3">
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-600">Source type</label>
              <select value={formSourceType} onChange={(e) => setFormSourceType(e.target.value as NotificationSourceType)} className="rounded border px-3 py-2 text-sm">
                {SOURCE_TYPES.map((t) => <option key={t} value={t}>{t}</option>)}
              </select>
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-600">Min severity</label>
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
              <label className="mb-1 block text-xs font-medium text-gray-600">ntfy topic</label>
              <input value={formTopic} onChange={(e) => setFormTopic(e.target.value)} required className="w-full rounded border px-3 py-2 text-sm" />
            </div>
          )}
          {formDest === "webhook" && (
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-600">Webhook URL</label>
              <input value={formUrl} onChange={(e) => setFormUrl(e.target.value)} required type="url" className="w-full rounded border px-3 py-2 text-sm" />
            </div>
          )}
          {formDest === "email" && <p className="text-xs text-gray-400">Email notifications are not yet implemented.</p>}

          <div className="flex gap-2">
            <button type="submit" disabled={updateRule.isPending} className="rounded bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50">Update</button>
            <button type="button" onClick={() => setEditing(false)} className="text-sm text-gray-500 hover:text-gray-700">Cancel</button>
          </div>
        </form>
      ) : (
        <div className="space-y-4 rounded border bg-white p-4">
          <div className="grid grid-cols-3 gap-4 text-sm">
            <div>
              <span className="font-medium text-gray-500">Source Type</span>
              <p className="mt-1">{rule.source_type === "all" ? "All sources" : rule.source_type}</p>
            </div>
            <div>
              <span className="font-medium text-gray-500">Min Severity</span>
              <p className="mt-1">&ge; {rule.min_severity}</p>
            </div>
            <div>
              <span className="font-medium text-gray-500">Destination</span>
              <p className="mt-1">{rule.destination}</p>
            </div>
          </div>

          <div>
            <span className="text-sm font-medium text-gray-500">Config</span>
            <p className="mt-1 font-mono text-sm text-gray-700">{configSummary()}</p>
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
