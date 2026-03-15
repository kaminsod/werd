import { useState, type FormEvent } from "react";
import { useParams } from "react-router";
import { useConnections, useCreateConnection, useUpdateConnection, useDeleteConnection } from "@/hooks/use-connections";
import InfoIcon from "@/components/info-icon";
import { platformCredentials as credsHelp } from "@/lib/help-content";
import type { Connection } from "@/types/api";

const PLATFORMS = ["bluesky"];

const PLATFORM_LABELS: Record<string, string> = {
  bluesky: "Bluesky",
};

const CREDENTIAL_HINTS: Record<string, string> = {
  bluesky: '{"identifier": "user.bsky.social", "app_password": "xxxx-xxxx-xxxx-xxxx"}',
};

export default function ConnectionsPage() {
  const { id: projectId } = useParams<{ id: string }>();
  const { data: connections, isLoading, error } = useConnections(projectId!);
  const createConn = useCreateConnection(projectId!);
  const updateConn = useUpdateConnection(projectId!);
  const deleteConn = useDeleteConnection(projectId!);

  const [showForm, setShowForm] = useState(false);
  const [editId, setEditId] = useState<string | null>(null);
  const [formPlatform, setFormPlatform] = useState("bluesky");
  const [formCreds, setFormCreds] = useState("");
  const [formEnabled, setFormEnabled] = useState(true);

  function resetForm() {
    setFormPlatform("bluesky");
    setFormCreds("");
    setFormEnabled(true);
    setShowForm(false);
    setEditId(null);
  }

  function startEdit(conn: Connection) {
    setEditId(conn.id);
    setFormPlatform(conn.platform);
    setFormCreds("");
    setFormEnabled(conn.enabled);
    setShowForm(true);
  }

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    let credentials: unknown;
    try {
      credentials = JSON.parse(formCreds);
    } catch {
      alert("Invalid JSON in credentials field");
      return;
    }

    if (editId) {
      updateConn.mutate(
        { connId: editId, platform: formPlatform, credentials, enabled: formEnabled },
        { onSuccess: resetForm },
      );
    } else {
      createConn.mutate(
        { platform: formPlatform, credentials, enabled: formEnabled },
        { onSuccess: resetForm },
      );
    }
  }

  function handleDelete(connId: string) {
    if (confirm("Delete this platform connection?")) {
      deleteConn.mutate(connId);
    }
  }

  if (isLoading) return <p className="text-gray-500">Loading connections...</p>;
  if (error) return <p className="text-red-600">Error: {error.message}</p>;

  return (
    <div>
      <div className="mb-4 flex items-center justify-between">
        <h2 className="text-xl font-semibold">Platform Connections</h2>
        <button
          onClick={() => { resetForm(); setShowForm(!showForm); }}
          className="rounded bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
        >
          {showForm && !editId ? "Cancel" : "Add Connection"}
        </button>
      </div>

      {showForm && (
        <form onSubmit={handleSubmit} className="mb-6 space-y-3 rounded border bg-gray-50 p-4">
          <h3 className="text-sm font-medium">{editId ? "Edit Connection" : "New Connection"}</h3>

          {(createConn.error || updateConn.error) && (
            <p className="text-sm text-red-600">
              {(createConn.error || updateConn.error)?.message}
            </p>
          )}

          <div className="flex gap-3">
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-600">Platform</label>
              <select
                value={formPlatform}
                onChange={(e) => setFormPlatform(e.target.value)}
                className="rounded border px-3 py-2 text-sm"
              >
                {PLATFORMS.map((p) => (
                  <option key={p} value={p}>{PLATFORM_LABELS[p] ?? p}</option>
                ))}
              </select>
            </div>
            <label className="flex items-center gap-2 self-end">
              <input type="checkbox" checked={formEnabled} onChange={(e) => setFormEnabled(e.target.checked)} />
              <span className="text-sm">Enabled</span>
            </label>
          </div>

          <div>
            <label className="mb-1 block text-xs font-medium text-gray-600">
              Credentials (JSON)
              <InfoIcon tooltip={credsHelp.tooltip}>{credsHelp.modal}</InfoIcon>
              {editId && <span className="font-normal text-gray-400"> — re-enter to update</span>}
            </label>
            <textarea
              value={formCreds}
              onChange={(e) => setFormCreds(e.target.value)}
              required
              rows={3}
              className="w-full rounded border px-3 py-2 font-mono text-sm"
              placeholder={CREDENTIAL_HINTS[formPlatform] ?? "{}"}
            />
          </div>

          <button
            type="submit"
            disabled={createConn.isPending || updateConn.isPending}
            className="rounded bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
          >
            {createConn.isPending || updateConn.isPending ? "Saving..." : editId ? "Update" : "Connect"}
          </button>
          {editId && (
            <button type="button" onClick={resetForm} className="ml-2 text-sm text-gray-500 hover:text-gray-700">
              Cancel edit
            </button>
          )}
        </form>
      )}

      {connections!.length === 0 ? (
        <p className="text-gray-500">No platform connections. Add a connection to start publishing.</p>
      ) : (
        <div className="space-y-2">
          {connections!.map((conn) => (
            <div key={conn.id} className="flex items-center justify-between rounded border bg-white p-3">
              <div className="flex items-center gap-3">
                <span className="rounded bg-indigo-100 px-2 py-0.5 text-xs font-medium text-indigo-700">
                  {PLATFORM_LABELS[conn.platform] ?? conn.platform}
                </span>
                <span className={`rounded px-2 py-0.5 text-xs ${conn.enabled ? "bg-green-50 text-green-700" : "bg-gray-100 text-gray-400"}`}>
                  {conn.enabled ? "enabled" : "disabled"}
                </span>
                <span className="text-xs text-gray-400">
                  Connected {new Date(conn.created_at).toLocaleDateString()}
                </span>
              </div>
              <div className="flex items-center gap-2">
                <button onClick={() => startEdit(conn)} className="rounded px-2 py-1 text-xs text-blue-600 hover:bg-blue-50">
                  Edit
                </button>
                <button onClick={() => handleDelete(conn.id)} className="rounded px-2 py-1 text-xs text-red-600 hover:bg-red-50">
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
