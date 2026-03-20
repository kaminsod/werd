import { useState, useId } from "react";

// -- Field schema --------------------------------------------------------

interface FieldDef {
  key: string;
  label: string;
  type: "text" | "password";
}

const FIELD_SCHEMA: Record<string, Record<string, FieldDef[]>> = {
  bluesky: {
    api: [
      { key: "identifier", label: "Identifier", type: "text" },
      { key: "app_password", label: "App Password", type: "password" },
    ],
    browser: [
      { key: "username", label: "Username", type: "text" },
      { key: "password", label: "Password", type: "password" },
    ],
  },
  reddit: {
    api: [
      { key: "client_id", label: "Client ID", type: "text" },
      { key: "client_secret", label: "Client Secret", type: "password" },
      { key: "username", label: "Username", type: "text" },
      { key: "password", label: "Password", type: "password" },
      { key: "user_agent", label: "User Agent", type: "text" },
      { key: "subreddit", label: "Subreddit", type: "text" },
    ],
    browser: [
      { key: "username", label: "Username", type: "text" },
      { key: "password", label: "Password", type: "password" },
      { key: "subreddit", label: "Subreddit", type: "text" },
    ],
  },
  hn: {
    api: [],
    browser: [
      { key: "username", label: "Username", type: "text" },
      { key: "password", label: "Password", type: "password" },
    ],
  },
};

// -- Helpers -------------------------------------------------------------

function parseValue(value: string): Record<string, string> | null {
  const trimmed = value.trim();
  if (trimmed === "") return {};
  try {
    const parsed = JSON.parse(trimmed);
    if (typeof parsed !== "object" || parsed === null || Array.isArray(parsed)) return null;
    return parsed as Record<string, string>;
  } catch {
    return null;
  }
}

function toJson(obj: Record<string, string>): string {
  // Omit keys with empty values
  const cleaned: Record<string, string> = {};
  for (const [k, v] of Object.entries(obj)) {
    if (v !== "") cleaned[k] = v;
  }
  return Object.keys(cleaned).length === 0 ? "" : JSON.stringify(cleaned, null, 2);
}

// -- Custom field row ----------------------------------------------------

interface CustomFieldRow {
  id: number;
  key: string;
  value: string;
}

// -- Component -----------------------------------------------------------

interface CredentialEditorProps {
  platform: string;
  method: string;
  value: string;
  onChange: (value: string) => void;
}

export default function CredentialEditor({ platform, method, value, onChange }: CredentialEditorProps) {
  const [tab, setTab] = useState<"form" | "json">("form");
  const [nextCustomId, setNextCustomId] = useState(1);
  const [blankCustomFields, setBlankCustomFields] = useState<CustomFieldRow[]>([]);
  const baseId = useId();

  const fields = FIELD_SCHEMA[platform]?.[method] ?? [];
  const schemaKeys = new Set(fields.map((f) => f.key));
  const parsed = parseValue(value);
  const isValid = parsed !== null;

  // Derive custom (non-schema) keys from current value
  const customEntries: [string, string][] = [];
  if (parsed) {
    for (const [k, v] of Object.entries(parsed)) {
      if (!schemaKeys.has(k)) customEntries.push([k, String(v)]);
    }
  }

  // -- Handlers ----------------------------------------------------------

  function setField(key: string, val: string) {
    const obj = parsed ?? {};
    if (val === "") {
      delete obj[key];
    } else {
      obj[key] = val;
    }
    onChange(toJson(obj));
  }

  function deleteField(key: string) {
    if (!parsed) return;
    delete parsed[key];
    onChange(toJson(parsed));
  }

  function renameCustomKey(oldKey: string, newKey: string) {
    if (!parsed || newKey === oldKey) return;
    const val = parsed[oldKey] ?? "";
    delete parsed[oldKey];
    if (newKey !== "") parsed[newKey] = val;
    onChange(toJson(parsed));
  }

  function addBlankRow() {
    setBlankCustomFields((prev) => [...prev, { id: nextCustomId, key: "", value: "" }]);
    setNextCustomId((n) => n + 1);
  }

  function updateBlankRow(id: number, field: "key" | "value", val: string) {
    setBlankCustomFields((prev) =>
      prev.map((r) => (r.id === id ? { ...r, [field]: val } : r)),
    );
  }

  function commitBlankRow(row: CustomFieldRow) {
    if (row.key.trim() === "" || row.value.trim() === "") return;
    const obj = parsed ?? {};
    obj[row.key.trim()] = row.value;
    onChange(toJson(obj));
    setBlankCustomFields((prev) => prev.filter((r) => r.id !== row.id));
  }

  function removeBlankRow(id: number) {
    setBlankCustomFields((prev) => prev.filter((r) => r.id !== id));
  }

  // -- Render ------------------------------------------------------------

  const noFields = fields.length === 0 && platform === "hn" && method === "api";

  return (
    <div>
      {/* Tab pills */}
      <div className="mb-2 flex items-center gap-1">
        <button
          type="button"
          onClick={() => setTab("form")}
          className={`rounded-full px-3 py-1 text-xs ${
            tab === "form"
              ? "bg-blue-100 font-medium text-blue-700"
              : "bg-gray-100 text-gray-600 hover:bg-gray-200"
          }`}
        >
          Form
        </button>
        <button
          type="button"
          onClick={() => setTab("json")}
          className={`rounded-full px-3 py-1 text-xs ${
            tab === "json"
              ? "bg-blue-100 font-medium text-blue-700"
              : "bg-gray-100 text-gray-600 hover:bg-gray-200"
          }`}
        >
          JSON
        </button>
      </div>

      {tab === "json" ? (
        <textarea
          value={value}
          onChange={(e) => onChange(e.target.value)}
          rows={4}
          className="w-full rounded border px-3 py-2 font-mono text-sm"
          placeholder="{}"
        />
      ) : (
        <div className="space-y-2">
          {!isValid && (
            <p className="rounded bg-amber-50 px-3 py-2 text-xs text-amber-700">
              Invalid JSON — switch to JSON tab to fix before editing fields.
            </p>
          )}

          {noFields && isValid && (
            <p className="rounded bg-gray-50 px-3 py-2 text-xs text-gray-500">
              No credentials needed for this configuration.
            </p>
          )}

          {/* Schema fields */}
          {fields.map((f) => (
            <div key={f.key}>
              <label htmlFor={`${baseId}-${f.key}`} className="mb-1 block text-xs font-medium text-gray-600">
                {f.label}
              </label>
              <input
                id={`${baseId}-${f.key}`}
                type={f.type}
                disabled={!isValid}
                value={isValid ? (parsed![f.key] ?? "") : ""}
                onChange={(e) => setField(f.key, e.target.value)}
                className="w-full rounded border px-3 py-2 text-sm disabled:bg-gray-100 disabled:text-gray-400"
                autoComplete="off"
              />
            </div>
          ))}

          {/* Existing custom fields from JSON */}
          {customEntries.map(([key, val]) => (
            <div key={key} className="flex items-end gap-2">
              <div className="flex-1">
                <label className="mb-1 block text-xs text-gray-400">Key</label>
                <input
                  value={key}
                  onChange={(e) => renameCustomKey(key, e.target.value)}
                  disabled={!isValid}
                  className="w-full rounded border px-3 py-2 text-sm disabled:bg-gray-100"
                />
              </div>
              <div className="flex-1">
                <label className="mb-1 block text-xs text-gray-400">Value</label>
                <input
                  value={val}
                  onChange={(e) => setField(key, e.target.value)}
                  disabled={!isValid}
                  className="w-full rounded border px-3 py-2 text-sm disabled:bg-gray-100"
                />
              </div>
              <button
                type="button"
                onClick={() => deleteField(key)}
                disabled={!isValid}
                className="mb-0.5 rounded px-2 py-2 text-xs text-red-500 hover:bg-red-50 disabled:opacity-50"
              >
                Delete
              </button>
            </div>
          ))}

          {/* Blank custom fields (not yet committed to JSON) */}
          {blankCustomFields.map((row) => (
            <div key={row.id} className="flex items-end gap-2">
              <div className="flex-1">
                <label className="mb-1 block text-xs text-gray-400">Key</label>
                <input
                  value={row.key}
                  onChange={(e) => updateBlankRow(row.id, "key", e.target.value)}
                  onBlur={() => commitBlankRow(row)}
                  disabled={!isValid}
                  className="w-full rounded border px-3 py-2 text-sm disabled:bg-gray-100"
                  placeholder="field_name"
                />
              </div>
              <div className="flex-1">
                <label className="mb-1 block text-xs text-gray-400">Value</label>
                <input
                  value={row.value}
                  onChange={(e) => updateBlankRow(row.id, "value", e.target.value)}
                  onBlur={() => commitBlankRow(row)}
                  disabled={!isValid}
                  className="w-full rounded border px-3 py-2 text-sm disabled:bg-gray-100"
                  placeholder="value"
                />
              </div>
              <button
                type="button"
                onClick={() => removeBlankRow(row.id)}
                className="mb-0.5 rounded px-2 py-2 text-xs text-red-500 hover:bg-red-50"
              >
                Delete
              </button>
            </div>
          ))}

          {/* Add custom field button */}
          {isValid && (
            <button
              type="button"
              onClick={addBlankRow}
              className="text-xs text-blue-600 hover:text-blue-800"
            >
              + Add field
            </button>
          )}
        </div>
      )}
    </div>
  );
}
