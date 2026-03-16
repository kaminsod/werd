import { useState, type FormEvent } from "react";
import { useParams } from "react-router";
import { useSources, useCreateSource, useUpdateSource, useDeleteSource } from "@/hooks/use-sources";
import InfoIcon from "@/components/info-icon";
import { sourceConfig as sourceConfigHelp } from "@/lib/help-content";
import type { MonitorType, Source } from "@/types/api";

const SOURCE_TYPES: MonitorType[] = ["reddit", "hn", "web", "rss", "github"];

const TYPE_COLORS: Record<MonitorType, string> = {
  reddit: "bg-orange-100 text-orange-700",
  hn: "bg-amber-100 text-amber-700",
  web: "bg-blue-100 text-blue-700",
  rss: "bg-green-100 text-green-700",
  github: "bg-gray-100 text-gray-700",
};

export default function SourcesPage() {
  const { id: projectId } = useParams<{ id: string }>();
  const { data: sources, isLoading, error } = useSources(projectId!);
  const createSource = useCreateSource(projectId!);
  const updateSource = useUpdateSource(projectId!);
  const deleteSource = useDeleteSource(projectId!);

  const [showForm, setShowForm] = useState(false);
  const [editId, setEditId] = useState<string | null>(null);
  const [formType, setFormType] = useState<MonitorType>("web");
  const [formConfig, setFormConfig] = useState("{}");
  const [formEnabled, setFormEnabled] = useState(true);
  // Structured config fields for reddit/hn.
  const [formMode, setFormMode] = useState("subreddit");
  const [formSubreddit, setFormSubreddit] = useState("");
  const [formThreadId, setFormThreadId] = useState("");
  const [formItemId, setFormItemId] = useState("");
  const [formKeywords, setFormKeywords] = useState("");
  const [formPollInterval, setFormPollInterval] = useState("300");

  const useStructuredForm = formType === "reddit" || formType === "hn";

  function resetForm() {
    setFormType("web");
    setFormConfig("{}");
    setFormEnabled(true);
    setFormMode("subreddit");
    setFormSubreddit("");
    setFormThreadId("");
    setFormItemId("");
    setFormKeywords("");
    setFormPollInterval("300");
    setShowForm(false);
    setEditId(null);
  }

  function startEdit(src: Source) {
    setEditId(src.id);
    setFormType(src.type);
    setFormEnabled(src.enabled);
    setShowForm(true);

    const cfg = src.config as Record<string, unknown>;
    if (src.type === "reddit" || src.type === "hn") {
      setFormMode((cfg.mode as string) || "subreddit");
      setFormSubreddit((cfg.subreddit as string) || "");
      setFormThreadId((cfg.thread_id as string) || "");
      setFormItemId(cfg.item_id ? String(cfg.item_id) : "");
      setFormKeywords(Array.isArray(cfg.keywords) ? (cfg.keywords as string[]).join(", ") : "");
      setFormPollInterval(cfg.poll_interval_secs ? String(cfg.poll_interval_secs) : "300");
      setFormConfig(JSON.stringify(cfg, null, 2));
    } else {
      setFormConfig(JSON.stringify(cfg, null, 2));
    }
  }

  function buildConfig(): Record<string, unknown> {
    if (formType === "reddit") {
      const base: Record<string, unknown> = {
        mode: formMode,
        poll_interval_secs: parseInt(formPollInterval) || 300,
      };
      if (formMode === "thread") {
        base.thread_id = formThreadId;
        base.subreddit = formSubreddit;
      } else {
        base.subreddit = formSubreddit;
        if (formKeywords.trim()) {
          base.keywords = formKeywords.split(",").map((k) => k.trim()).filter(Boolean);
        }
      }
      return base;
    }
    if (formType === "hn") {
      const base: Record<string, unknown> = {
        mode: formMode,
        poll_interval_secs: parseInt(formPollInterval) || 300,
      };
      if (formMode === "thread") {
        base.item_id = parseInt(formItemId) || 0;
      } else {
        if (formKeywords.trim()) {
          base.keywords = formKeywords.split(",").map((k) => k.trim()).filter(Boolean);
        }
      }
      return base;
    }
    // Fallback: parse JSON.
    try {
      return JSON.parse(formConfig);
    } catch {
      return {};
    }
  }

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    const config = buildConfig();

    if (editId) {
      updateSource.mutate(
        { sourceId: editId, type: formType, config, enabled: formEnabled },
        { onSuccess: resetForm },
      );
    } else {
      createSource.mutate(
        { type: formType, config, enabled: formEnabled },
        { onSuccess: resetForm },
      );
    }
  }

  function handleDelete(sourceId: string) {
    if (confirm("Delete this monitor source?")) {
      deleteSource.mutate(sourceId);
    }
  }

  if (isLoading) return <p className="text-gray-500">Loading sources...</p>;
  if (error) return <p className="text-red-600">Error: {error.message}</p>;

  return (
    <div>
      <div className="mb-4 flex items-center justify-between">
        <h2 className="text-xl font-semibold">Monitor Sources</h2>
        <button
          onClick={() => { resetForm(); setShowForm(!showForm); }}
          className="rounded bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
        >
          {showForm && !editId ? "Cancel" : "Add Source"}
        </button>
      </div>

      {showForm && (
        <form onSubmit={handleSubmit} className="mb-6 space-y-3 rounded border bg-gray-50 p-4">
          <h3 className="text-sm font-medium">{editId ? "Edit Source" : "New Source"}</h3>

          {(createSource.error || updateSource.error) && (
            <p className="text-sm text-red-600">
              {(createSource.error || updateSource.error)?.message}
            </p>
          )}

          <div className="flex gap-3">
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-600">Type</label>
              <select
                value={formType}
                onChange={(e) => setFormType(e.target.value as MonitorType)}
                className="rounded border px-3 py-2 text-sm"
              >
                {SOURCE_TYPES.map((t) => (
                  <option key={t} value={t}>{t}</option>
                ))}
              </select>
            </div>
            <label className="flex items-center gap-2 self-end">
              <input
                type="checkbox"
                checked={formEnabled}
                onChange={(e) => setFormEnabled(e.target.checked)}
              />
              <span className="text-sm">Enabled</span>
            </label>
          </div>

          {useStructuredForm ? (
            <>
              {/* Mode selector */}
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-600">Mode</label>
                <select value={formMode} onChange={(e) => setFormMode(e.target.value)} className="rounded border px-3 py-2 text-sm">
                  {formType === "reddit" && (
                    <>
                      <option value="subreddit">Subreddit (new posts)</option>
                      <option value="thread">Thread (comments)</option>
                    </>
                  )}
                  {formType === "hn" && (
                    <>
                      <option value="keywords">Keywords (new stories)</option>
                      <option value="thread">Thread (comments)</option>
                    </>
                  )}
                </select>
              </div>

              {/* Reddit fields */}
              {formType === "reddit" && formMode === "subreddit" && (
                <div className="space-y-2">
                  <div>
                    <label className="mb-1 block text-xs font-medium text-gray-600">Subreddit</label>
                    <input value={formSubreddit} onChange={(e) => setFormSubreddit(e.target.value)} required placeholder="golang" className="w-full rounded border px-3 py-2 text-sm" />
                  </div>
                  <div>
                    <label className="mb-1 block text-xs font-medium text-gray-600">Keywords (optional, comma-separated)</label>
                    <input value={formKeywords} onChange={(e) => setFormKeywords(e.target.value)} placeholder="self-hosted, monitoring" className="w-full rounded border px-3 py-2 text-sm" />
                  </div>
                </div>
              )}
              {formType === "reddit" && formMode === "thread" && (
                <div className="space-y-2">
                  <div>
                    <label className="mb-1 block text-xs font-medium text-gray-600">Thread ID</label>
                    <input value={formThreadId} onChange={(e) => setFormThreadId(e.target.value)} required placeholder="t3_abc123" className="w-full rounded border px-3 py-2 text-sm font-mono" />
                  </div>
                  <div>
                    <label className="mb-1 block text-xs font-medium text-gray-600">Subreddit</label>
                    <input value={formSubreddit} onChange={(e) => setFormSubreddit(e.target.value)} placeholder="golang" className="w-full rounded border px-3 py-2 text-sm" />
                  </div>
                </div>
              )}

              {/* HN fields */}
              {formType === "hn" && formMode === "keywords" && (
                <div>
                  <label className="mb-1 block text-xs font-medium text-gray-600">Keywords (comma-separated)</label>
                  <input value={formKeywords} onChange={(e) => setFormKeywords(e.target.value)} placeholder="Show HN, self-hosted" className="w-full rounded border px-3 py-2 text-sm" />
                </div>
              )}
              {formType === "hn" && formMode === "thread" && (
                <div>
                  <label className="mb-1 block text-xs font-medium text-gray-600">HN Item ID</label>
                  <input value={formItemId} onChange={(e) => setFormItemId(e.target.value)} required type="number" placeholder="12345678" className="w-full rounded border px-3 py-2 text-sm font-mono" />
                </div>
              )}

              {/* Poll interval */}
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-600">Poll interval (seconds)</label>
                <input value={formPollInterval} onChange={(e) => setFormPollInterval(e.target.value)} type="number" min="60" className="w-32 rounded border px-3 py-2 text-sm" />
              </div>
            </>
          ) : (
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-600">
                Config (JSON)
                <InfoIcon tooltip={sourceConfigHelp.tooltip}>{sourceConfigHelp.modal}</InfoIcon>
              </label>
              <textarea
                value={formConfig}
                onChange={(e) => setFormConfig(e.target.value)}
                rows={4}
                className="w-full rounded border px-3 py-2 font-mono text-sm"
                placeholder='{"urls": ["https://example.com/blog"]}'
              />
            </div>
          )}

          <button
            type="submit"
            disabled={createSource.isPending || updateSource.isPending}
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

      {sources!.length === 0 ? (
        <p className="text-gray-500">No monitor sources configured.</p>
      ) : (
        <div className="space-y-2">
          {sources!.map((src) => (
            <div key={src.id} className="flex items-center justify-between rounded border bg-white p-3">
              <div className="flex items-center gap-3">
                <span className={`rounded px-2 py-0.5 text-xs font-medium ${TYPE_COLORS[src.type]}`}>
                  {src.type}
                </span>
                <span className={`rounded px-2 py-0.5 text-xs ${src.enabled ? "bg-green-50 text-green-700" : "bg-gray-100 text-gray-400"}`}>
                  {src.enabled ? "enabled" : "disabled"}
                </span>
                <span className="text-xs text-gray-500 font-mono">
                  {JSON.stringify(src.config).slice(0, 60)}
                  {JSON.stringify(src.config).length > 60 ? "..." : ""}
                </span>
              </div>
              <div className="flex items-center gap-2">
                <button onClick={() => startEdit(src)} className="rounded px-2 py-1 text-xs text-blue-600 hover:bg-blue-50">
                  Edit
                </button>
                <button onClick={() => handleDelete(src.id)} className="rounded px-2 py-1 text-xs text-red-600 hover:bg-red-50">
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
