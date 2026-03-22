import { useState, type FormEvent } from "react";
import { useParams, Link, useNavigate } from "react-router";
import { useSource, useUpdateSource, useDeleteSource } from "@/hooks/use-sources";
import { useProcessingRules } from "@/hooks/use-processing-rules";
import SourceConfigEditor from "@/components/source-config-editor";
import type { MonitorType } from "@/types/api";

const SOURCE_TYPES: MonitorType[] = ["reddit", "hn", "bluesky", "web", "rss", "github"];

const TYPE_LABELS: Record<MonitorType, string> = {
  reddit: "Reddit",
  hn: "Hacker News",
  bluesky: "Bluesky",
  web: "Web",
  rss: "RSS",
  github: "GitHub",
};

const TYPE_COLORS: Record<MonitorType, string> = {
  reddit: "bg-orange-100 text-orange-700",
  hn: "bg-amber-100 text-amber-700",
  bluesky: "bg-sky-100 text-sky-700",
  web: "bg-blue-100 text-blue-700",
  rss: "bg-green-100 text-green-700",
  github: "bg-gray-100 text-gray-700",
};

export default function SourceDetailPage() {
  const { id: projectId, sourceId } = useParams<{ id: string; sourceId: string }>();
  const navigate = useNavigate();
  const { data: source, isLoading, error } = useSource(projectId!, sourceId!);
  const { data: allRules } = useProcessingRules(projectId!);
  const updateSource = useUpdateSource(projectId!);
  const deleteSource = useDeleteSource(projectId!);

  const [editing, setEditing] = useState(false);
  const [formType, setFormType] = useState<MonitorType>("web");
  const [formConfig, setFormConfig] = useState("{}");
  const [formEnabled, setFormEnabled] = useState(true);

  const scopedRules = allRules?.filter((r) => r.source_id === sourceId) ?? [];

  function startEdit() {
    if (!source) return;
    setFormType(source.type);
    setFormConfig(JSON.stringify(source.config, null, 2));
    setFormEnabled(source.enabled);
    setEditing(true);
  }

  function handleSave(e: FormEvent) {
    e.preventDefault();
    let config: Record<string, unknown>;
    try {
      config = JSON.parse(formConfig);
    } catch {
      alert("Invalid JSON in config field");
      return;
    }
    updateSource.mutate(
      { sourceId: sourceId!, type: formType, config, enabled: formEnabled },
      { onSuccess: () => setEditing(false) },
    );
  }

  function handleDelete() {
    if (confirm("Delete this source?")) {
      deleteSource.mutate(sourceId!, { onSuccess: () => navigate("../sources") });
    }
  }

  if (isLoading) return <p className="text-gray-500">Loading source...</p>;
  if (error) return <p className="text-red-600">Error: {error.message}</p>;
  if (!source) return <p className="text-gray-500">Source not found.</p>;

  return (
    <div>
      <Link to="../sources" className="mb-4 inline-block text-sm text-blue-600 hover:underline">&larr; Back to Sources</Link>

      <div className="mb-4 flex items-center gap-3">
        <h2 className="text-xl font-semibold">Source</h2>
        <span className={`rounded px-2 py-0.5 text-xs font-medium ${TYPE_COLORS[source.type]}`}>{TYPE_LABELS[source.type] ?? source.type}</span>
        <span className={`rounded px-2 py-0.5 text-xs ${source.enabled ? "bg-green-50 text-green-700" : "bg-gray-100 text-gray-400"}`}>
          {source.enabled ? "enabled" : "disabled"}
        </span>
      </div>

      {editing ? (
        <form onSubmit={handleSave} className="space-y-3 rounded border bg-gray-50 p-4">
          <h3 className="text-sm font-medium">Edit Source</h3>
          {updateSource.error && <p className="text-sm text-red-600">{updateSource.error.message}</p>}

          <div className="flex gap-3">
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-600">Type</label>
              <select value={formType} onChange={(e) => setFormType(e.target.value as MonitorType)} className="rounded border px-3 py-2 text-sm">
                {SOURCE_TYPES.map((t) => <option key={t} value={t}>{TYPE_LABELS[t]}</option>)}
              </select>
            </div>
            <label className="flex items-center gap-2 self-end">
              <input type="checkbox" checked={formEnabled} onChange={(e) => setFormEnabled(e.target.checked)} />
              <span className="text-sm">Enabled</span>
            </label>
          </div>

          <SourceConfigEditor
            sourceType={formType}
            config={formConfig}
            onConfigChange={setFormConfig}
          />

          <div className="flex gap-2">
            <button type="submit" disabled={updateSource.isPending} className="rounded bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50">Update</button>
            <button type="button" onClick={() => setEditing(false)} className="text-sm text-gray-500 hover:text-gray-700">Cancel</button>
          </div>
        </form>
      ) : (
        <div className="space-y-4 rounded border bg-white p-4">
          <div>
            <span className="text-sm font-medium text-gray-500">Config</span>
            <pre className="mt-1 overflow-x-auto rounded bg-gray-50 p-3 font-mono text-sm text-gray-700">
              {JSON.stringify(source.config, null, 2)}
            </pre>
          </div>

          <div className="grid grid-cols-2 gap-4 text-sm">
            <div>
              <span className="font-medium text-gray-500">Created</span>
              <p className="mt-1">{new Date(source.created_at).toLocaleString()}</p>
            </div>
            <div>
              <span className="font-medium text-gray-500">Updated</span>
              <p className="mt-1">{new Date(source.updated_at).toLocaleString()}</p>
            </div>
          </div>

          {scopedRules.length > 0 && (
            <div>
              <span className="text-sm font-medium text-gray-500">Processing Rules (scoped to this source)</span>
              <div className="mt-2 space-y-1">
                {scopedRules.map((rule) => (
                  <Link key={rule.id} to={`../processing/${rule.id}`} className="block rounded border px-3 py-2 text-sm text-blue-600 hover:bg-gray-50">
                    {rule.name || "(unnamed)"} — {rule.phase}/{rule.rule_type}
                  </Link>
                ))}
              </div>
            </div>
          )}

          <div className="flex gap-2 border-t pt-4">
            <button onClick={startEdit} className="rounded px-2 py-1 text-xs text-blue-600 hover:bg-blue-50">Edit</button>
            <button onClick={handleDelete} className="rounded px-2 py-1 text-xs text-red-600 hover:bg-red-50">Delete</button>
          </div>
        </div>
      )}
    </div>
  );
}
