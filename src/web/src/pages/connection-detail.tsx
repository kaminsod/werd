import { useState, type FormEvent } from "react";
import { useParams, Link, useNavigate } from "react-router";
import { useConnection, useUpdateConnection, useDeleteConnection } from "@/hooks/use-connections";
import CredentialEditor from "@/components/credential-editor";
import type { ConnectionMethod } from "@/types/api";

const PLATFORM_LABELS: Record<string, string> = {
  bluesky: "Bluesky",
  reddit: "Reddit",
  hn: "Hacker News",
  gmail: "Gmail",
  google_groups: "Google Groups",
};

const PLATFORMS = ["bluesky", "reddit", "hn", "gmail", "google_groups"];

const METHOD_GUIDANCE: Record<string, Record<string, string>> = {
  bluesky: {
    api: "Faster and more reliable. Uses app password.",
    browser: "Automates the Bluesky web interface. Uses account password.",
  },
  reddit: {
    api: "Faster. Requires a Reddit 'script' app (reddit.com/prefs/apps).",
    browser: "No app registration needed. Uses account password.",
  },
  hn: {
    api: "Monitoring only — HN has no posting API.",
    browser: "Required for posting to HN. Uses account password.",
  },
  gmail: {
    api: "Send emails via SMTP. Requires an app password (Google account → 2FA → App passwords).",
  },
  google_groups: {
    api: "Post to a Google Group via email. Requires group membership and an app password.",
  },
};

const API_PUBLISHABLE = new Set(["bluesky", "reddit", "gmail", "google_groups"]);

export default function ConnectionDetailPage() {
  const { id: projectId, connId } = useParams<{ id: string; connId: string }>();
  const navigate = useNavigate();
  const { data: conn, isLoading, error } = useConnection(projectId!, connId!);
  const updateConn = useUpdateConnection(projectId!);
  const deleteConn = useDeleteConnection(projectId!);

  const [editing, setEditing] = useState(false);
  const [formPlatform, setFormPlatform] = useState("bluesky");
  const [formMethod, setFormMethod] = useState<ConnectionMethod>("api");
  const [formCreds, setFormCreds] = useState("");
  const [formEnabled, setFormEnabled] = useState(true);

  function startEdit() {
    if (!conn) return;
    setFormPlatform(conn.platform);
    setFormMethod(conn.method);
    setFormCreds("");
    setFormEnabled(conn.enabled);
    setEditing(true);
  }

  function handleSave(e: FormEvent) {
    e.preventDefault();
    let credentials: unknown;
    try {
      credentials = formCreds.trim() === "" ? {} : JSON.parse(formCreds);
    } catch {
      alert("Invalid JSON in credentials field");
      return;
    }
    updateConn.mutate(
      { connId: connId!, platform: formPlatform, method: formMethod, credentials, enabled: formEnabled },
      { onSuccess: () => setEditing(false) },
    );
  }

  function handleDelete() {
    if (confirm("Delete this platform connection?")) {
      deleteConn.mutate(connId!, { onSuccess: () => navigate("../connections") });
    }
  }

  if (isLoading) return <p className="text-gray-500">Loading connection...</p>;
  if (error) return <p className="text-red-600">Error: {error.message}</p>;
  if (!conn) return <p className="text-gray-500">Connection not found.</p>;

  return (
    <div>
      <Link to="../connections" className="mb-4 inline-block text-sm text-blue-600 hover:underline">&larr; Back to Connections</Link>

      <div className="mb-4 flex items-center gap-3">
        <h2 className="text-xl font-semibold">{PLATFORM_LABELS[conn.platform] ?? conn.platform} Connection</h2>
        <span className={`rounded px-2 py-0.5 text-xs ${conn.method === "browser" ? "bg-purple-100 text-purple-700" : "bg-sky-100 text-sky-700"}`}>
          {conn.method}
        </span>
        {conn.target && (
          <span className="rounded bg-gray-50 px-2 py-0.5 text-xs text-gray-600">{conn.target}</span>
        )}
        <span className={`rounded px-2 py-0.5 text-xs ${conn.enabled ? "bg-green-50 text-green-700" : "bg-gray-100 text-gray-400"}`}>
          {conn.enabled ? "enabled" : "disabled"}
        </span>
      </div>

      {editing ? (
        <form onSubmit={handleSave} className="space-y-3 rounded border bg-gray-50 p-4">
          <h3 className="text-sm font-medium">Edit Connection</h3>
          {updateConn.error && <p className="text-sm text-red-600">{updateConn.error.message}</p>}

          <div className="flex items-end gap-3">
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-600">Platform</label>
              <select value={formPlatform} onChange={(e) => setFormPlatform(e.target.value)} className="rounded border px-3 py-2 text-sm">
                {PLATFORMS.map((p) => <option key={p} value={p}>{PLATFORM_LABELS[p] ?? p}</option>)}
              </select>
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-600">Method</label>
              <div className="flex h-[38px] items-center gap-3">
                <label className="flex items-center gap-1.5">
                  <input type="radio" name="method" value="api" checked={formMethod === "api"} onChange={() => setFormMethod("api")} />
                  <span className="text-sm">API</span>
                </label>
                <label className="flex items-center gap-1.5">
                  <input type="radio" name="method" value="browser" checked={formMethod === "browser"} onChange={() => setFormMethod("browser")} />
                  <span className="text-sm">Browser</span>
                </label>
              </div>
            </div>
            <label className="flex items-center gap-2">
              <input type="checkbox" checked={formEnabled} onChange={(e) => setFormEnabled(e.target.checked)} />
              <span className="text-sm">Enabled</span>
            </label>
          </div>

          {METHOD_GUIDANCE[formPlatform]?.[formMethod] && (
            <p className="rounded bg-blue-50 px-3 py-2 text-xs text-blue-700">{METHOD_GUIDANCE[formPlatform][formMethod]}</p>
          )}

          <div>
            <label className="mb-1 block text-xs font-medium text-gray-600">Credentials — re-enter to update</label>
            <CredentialEditor
              platform={formPlatform}
              method={formMethod}
              value={formCreds}
              onChange={setFormCreds}
            />
          </div>

          <div className="flex gap-2">
            <button type="submit" disabled={updateConn.isPending} className="rounded bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50">Update</button>
            <button type="button" onClick={() => setEditing(false)} className="text-sm text-gray-500 hover:text-gray-700">Cancel</button>
          </div>
        </form>
      ) : (
        <div className="space-y-4 rounded border bg-white p-4">
          <div className="grid grid-cols-2 gap-4 text-sm">
            <div>
              <span className="font-medium text-gray-500">Platform</span>
              <p className="mt-1">{PLATFORM_LABELS[conn.platform] ?? conn.platform}</p>
            </div>
            <div>
              <span className="font-medium text-gray-500">Method</span>
              <p className="mt-1">{conn.method}</p>
            </div>
          </div>

          {METHOD_GUIDANCE[conn.platform]?.[conn.method] && (
            <p className="rounded bg-blue-50 px-3 py-2 text-xs text-blue-700">{METHOD_GUIDANCE[conn.platform][conn.method]}</p>
          )}

          {conn.method === "api" && !API_PUBLISHABLE.has(conn.platform) && (
            <p className="rounded bg-amber-50 px-3 py-2 text-xs text-amber-700">
              {PLATFORM_LABELS[conn.platform] ?? conn.platform} API is monitoring-only. Use browser method for posting.
            </p>
          )}

          <div className="grid grid-cols-2 gap-4 text-sm">
            <div>
              <span className="font-medium text-gray-500">Connected</span>
              <p className="mt-1">{new Date(conn.created_at).toLocaleString()}</p>
            </div>
            <div>
              <span className="font-medium text-gray-500">Updated</span>
              <p className="mt-1">{new Date(conn.updated_at).toLocaleString()}</p>
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
