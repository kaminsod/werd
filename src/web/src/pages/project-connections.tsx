import { useState, type FormEvent } from "react";
import { useParams, Link } from "react-router";
import { useConnections, useCreateConnection, useUpdateConnection, useDeleteConnection, useCreateAccount } from "@/hooks/use-connections";
import InfoIcon from "@/components/info-icon";
import { platformCredentials as credsHelp } from "@/lib/help-content";
import type { Connection, ConnectionMethod } from "@/types/api";

const PLATFORMS = ["bluesky", "reddit", "hn"];

const PLATFORM_LABELS: Record<string, string> = {
  bluesky: "Bluesky",
  reddit: "Reddit",
  hn: "Hacker News",
};

const CREDENTIAL_HINTS: Record<string, Record<string, string>> = {
  bluesky: {
    api: '{"identifier": "user.bsky.social", "app_password": "xxxx-xxxx-xxxx-xxxx"}',
    browser: '{"username": "user@example.com", "password": "..."}',
  },
  reddit: {
    api: '{"client_id": "...", "client_secret": "...", "username": "...", "password": "...", "user_agent": "werd/1.0 by u/you", "subreddit": "test"}',
    browser: '{"username": "...", "password": "...", "subreddit": "test"}',
  },
  hn: {
    api: '{}',
    browser: '{"username": "...", "password": "..."}',
  },
};

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
};

// API-publishable platforms (HN needs browser for posting).
const API_PUBLISHABLE = new Set(["bluesky", "reddit"]);

export default function ConnectionsPage() {
  const { id: projectId } = useParams<{ id: string }>();
  const { data: connections, isLoading, error } = useConnections(projectId!);
  const createConn = useCreateConnection(projectId!);
  const updateConn = useUpdateConnection(projectId!);
  const deleteConn = useDeleteConnection(projectId!);

  const createAccount = useCreateAccount(projectId!);

  const [showForm, setShowForm] = useState(false);
  const [editId, setEditId] = useState<string | null>(null);
  const [formPlatform, setFormPlatform] = useState("bluesky");
  const [formMethod, setFormMethod] = useState<ConnectionMethod>("api");
  const [formCreds, setFormCreds] = useState("");
  const [formEnabled, setFormEnabled] = useState(true);

  const [showCreateAccount, setShowCreateAccount] = useState(false);
  const [caPlatform, setCaPlatform] = useState("bluesky");
  const [caEmail, setCaEmail] = useState("");
  const [caUsername, setCaUsername] = useState("");
  const [caPassword, setCaPassword] = useState("");

  function resetForm() {
    setFormPlatform("bluesky");
    setFormMethod("api");
    setFormCreds("");
    setFormEnabled(true);
    setShowForm(false);
    setEditId(null);
  }

  function startEdit(conn: Connection) {
    setEditId(conn.id);
    setFormPlatform(conn.platform);
    setFormMethod(conn.method);
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
        { connId: editId, platform: formPlatform, method: formMethod, credentials, enabled: formEnabled },
        { onSuccess: resetForm },
      );
    } else {
      createConn.mutate(
        { platform: formPlatform, method: formMethod, credentials, enabled: formEnabled },
        { onSuccess: resetForm },
      );
    }
  }

  function handleDelete(connId: string) {
    if (confirm("Delete this platform connection?")) {
      deleteConn.mutate(connId);
    }
  }

  function handleCreateAccount(e: FormEvent) {
    e.preventDefault();
    createAccount.mutate(
      { platform: caPlatform, email: caEmail || undefined, username: caUsername, password: caPassword },
      {
        onSuccess: () => {
          setShowCreateAccount(false);
          setCaEmail("");
          setCaUsername("");
          setCaPassword("");
        },
      },
    );
  }

  if (isLoading) return <p className="text-gray-500">Loading connections...</p>;
  if (error) return <p className="text-red-600">Error: {error.message}</p>;

  return (
    <div>
      <div className="mb-4 flex items-center justify-between">
        <h2 className="text-xl font-semibold">Platform Connections</h2>
        <div className="flex gap-2">
          <button
            onClick={() => { setShowCreateAccount(!showCreateAccount); setShowForm(false); resetForm(); }}
            className="rounded bg-green-600 px-4 py-2 text-sm font-medium text-white hover:bg-green-700"
          >
            {showCreateAccount ? "Cancel" : "Create Account"}
          </button>
          <button
            onClick={() => { resetForm(); setShowForm(!showForm); setShowCreateAccount(false); }}
            className="rounded bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
          >
            {showForm && !editId ? "Cancel" : "Add Connection"}
          </button>
        </div>
      </div>

      {showCreateAccount && (
        <form onSubmit={handleCreateAccount} className="mb-6 space-y-3 rounded border bg-green-50 p-4">
          <h3 className="text-sm font-medium">Create Account via Browser Automation</h3>
          <p className="text-xs text-gray-500">Creates a new account on the selected platform and automatically adds it as a browser connection.</p>

          {createAccount.error && (
            <p className="text-sm text-red-600">{createAccount.error.message}</p>
          )}

          <div>
            <label className="mb-1 block text-xs font-medium text-gray-600">Platform</label>
            <select value={caPlatform} onChange={(e) => setCaPlatform(e.target.value)} className="rounded border px-3 py-2 text-sm">
              {PLATFORMS.map((p) => (
                <option key={p} value={p}>{PLATFORM_LABELS[p] ?? p}</option>
              ))}
            </select>
          </div>

          {caPlatform !== "hn" && (
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-600">Email {caPlatform === "hn" ? "(not used)" : ""}</label>
              <input value={caEmail} onChange={(e) => setCaEmail(e.target.value)} type="email" placeholder="user@example.com" className="w-full rounded border px-3 py-2 text-sm" />
            </div>
          )}

          <div>
            <label className="mb-1 block text-xs font-medium text-gray-600">Username</label>
            <input value={caUsername} onChange={(e) => setCaUsername(e.target.value)} required placeholder="myusername" className="w-full rounded border px-3 py-2 text-sm" />
          </div>

          <div>
            <label className="mb-1 block text-xs font-medium text-gray-600">Password</label>
            <input value={caPassword} onChange={(e) => setCaPassword(e.target.value)} required type="password" placeholder="..." className="w-full rounded border px-3 py-2 text-sm" />
          </div>

          <button
            type="submit"
            disabled={createAccount.isPending}
            className="rounded bg-green-600 px-4 py-2 text-sm font-medium text-white hover:bg-green-700 disabled:opacity-50"
          >
            {createAccount.isPending ? "Creating..." : "Create Account & Connect"}
          </button>
        </form>
      )}

      {showForm && (
        <form onSubmit={handleSubmit} className="mb-6 space-y-3 rounded border bg-gray-50 p-4">
          <h3 className="text-sm font-medium">{editId ? "Edit Connection" : "New Connection"}</h3>

          {(createConn.error || updateConn.error) && (
            <p className="text-sm text-red-600">
              {(createConn.error || updateConn.error)?.message}
            </p>
          )}

          <div className="flex items-end gap-3">
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
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-600">Status</label>
              <label className="flex h-[38px] items-center gap-2">
                <input type="checkbox" checked={formEnabled} onChange={(e) => setFormEnabled(e.target.checked)} />
                <span className="text-sm">Enabled</span>
              </label>
            </div>
          </div>

          {METHOD_GUIDANCE[formPlatform]?.[formMethod] && (
            <p className="rounded bg-blue-50 px-3 py-2 text-xs text-blue-700">
              {METHOD_GUIDANCE[formPlatform][formMethod]}
            </p>
          )}

          {formMethod === "api" && !API_PUBLISHABLE.has(formPlatform) && (
            <p className="rounded bg-amber-50 px-3 py-2 text-xs text-amber-700">
              {PLATFORM_LABELS[formPlatform] ?? formPlatform} API is monitoring-only. Use browser method for posting.
            </p>
          )}

          <div>
            <label className="mb-1 block text-xs font-medium text-gray-600">
              Credentials (JSON)
              <InfoIcon tooltip={credsHelp.tooltip}>{credsHelp.modal}</InfoIcon>
              {editId && <span className="font-normal text-gray-400"> — re-enter to update</span>}
            </label>
            <textarea
              value={formCreds}
              onChange={(e) => setFormCreds(e.target.value)}
              required={formMethod === "browser" || API_PUBLISHABLE.has(formPlatform)}
              rows={3}
              className="w-full rounded border px-3 py-2 font-mono text-sm"
              placeholder={CREDENTIAL_HINTS[formPlatform]?.[formMethod] ?? "{}"}
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
                <span className={`rounded px-2 py-0.5 text-xs ${conn.method === "browser" ? "bg-purple-100 text-purple-700" : "bg-sky-100 text-sky-700"}`}>
                  {conn.method}
                </span>
                {conn.target && (
                  <span className="rounded bg-gray-50 px-2 py-0.5 text-xs text-gray-600">{conn.target}</span>
                )}
                <span className={`rounded px-2 py-0.5 text-xs ${conn.enabled ? "bg-green-50 text-green-700" : "bg-gray-100 text-gray-400"}`}>
                  {conn.enabled ? "enabled" : "disabled"}
                </span>
                {conn.method === "api" && !API_PUBLISHABLE.has(conn.platform) && (
                  <span className="rounded bg-amber-50 px-2 py-0.5 text-xs text-amber-600">monitoring only</span>
                )}
                <span className="text-xs text-gray-400">
                  Connected {new Date(conn.created_at).toLocaleDateString()}
                </span>
              </div>
              <div className="flex items-center gap-2">
                <Link to={conn.id} className="rounded px-2 py-1 text-xs text-blue-600 hover:bg-blue-50">View</Link>
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
