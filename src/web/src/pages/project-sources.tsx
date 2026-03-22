import { useState, type FormEvent } from "react";
import { useParams, Link } from "react-router";
import { useSources, useCreateSource, useUpdateSource, useDeleteSource } from "@/hooks/use-sources";
import InfoIcon from "@/components/info-icon";
import { navSources as sourcesHelp } from "@/lib/help-content";
import SourceConfigEditor, { defaultConfigForType, parseConfig } from "@/components/source-config-editor";
import type { MonitorType, Source } from "@/types/api";

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

export default function SourcesPage() {
  const { id: projectId } = useParams<{ id: string }>();
  const { data: sources, isLoading, error } = useSources(projectId!);
  const createSource = useCreateSource(projectId!);
  const updateSource = useUpdateSource(projectId!);
  const deleteSource = useDeleteSource(projectId!);

  const [showForm, setShowForm] = useState(false);
  const [editId, setEditId] = useState<string | null>(null);
  const [formType, setFormType] = useState<MonitorType>("reddit");
  const [formConfig, setFormConfig] = useState(() => defaultConfigForType("reddit"));
  const [formEnabled, setFormEnabled] = useState(true);

  function resetForm() {
    setFormType("reddit");
    setFormConfig(defaultConfigForType("reddit"));
    setFormEnabled(true);
    setShowForm(false);
    setEditId(null);
  }

  function handleTypeChange(newType: MonitorType) {
    setFormType(newType);
    setFormConfig(defaultConfigForType(newType));
  }

  function startEdit(src: Source) {
    setEditId(src.id);
    setFormType(src.type);
    setFormEnabled(src.enabled);
    setFormConfig(JSON.stringify(src.config, null, 2));
    setShowForm(true);
  }

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    const config = parseConfig(formConfig);
    if (!config) {
      alert("Invalid JSON in config");
      return;
    }

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
    if (confirm("Delete this source?")) {
      deleteSource.mutate(sourceId);
    }
  }

  if (isLoading) return <p className="text-gray-500">Loading sources...</p>;
  if (error) return <p className="text-red-600">Error: {error.message}</p>;

  return (
    <div>
      <div className="mb-4 flex items-center justify-between">
        <h2 className="text-xl font-semibold">
          Sources
          <InfoIcon tooltip={sourcesHelp.tooltip}>{sourcesHelp.modal}</InfoIcon>
        </h2>
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
                onChange={(e) => handleTypeChange(e.target.value as MonitorType)}
                className="rounded border px-3 py-2 text-sm"
              >
                {SOURCE_TYPES.map((t) => (
                  <option key={t} value={t}>{TYPE_LABELS[t]}</option>
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

          <SourceConfigEditor
            sourceType={formType}
            config={formConfig}
            onConfigChange={setFormConfig}
          />

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
        <p className="text-gray-500">No sources configured.</p>
      ) : (
        <div className="space-y-2">
          {sources!.map((src) => (
            <div key={src.id} className="flex items-center justify-between rounded border bg-white p-3">
              <div className="flex items-center gap-3">
                <span className={`rounded px-2 py-0.5 text-xs font-medium ${TYPE_COLORS[src.type]}`}>
                  {TYPE_LABELS[src.type] ?? src.type}
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
                <Link to={src.id} className="rounded px-2 py-1 text-xs text-blue-600 hover:bg-blue-50">View</Link>
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
