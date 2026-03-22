import { useState } from "react";
import type { MonitorType } from "@/types/api";

// ── Mode options per source type ──

const MODE_OPTIONS: Record<string, { value: string; label: string }[]> = {
  reddit: [
    { value: "subreddit", label: "Subreddit (new posts)" },
    { value: "thread", label: "Thread (comments)" },
    { value: "account", label: "Account (inbox & mentions)" },
  ],
  hn: [
    { value: "keywords", label: "Keywords (new stories)" },
    { value: "new", label: "All New Stories" },
    { value: "thread", label: "Thread (comments)" },
    { value: "account", label: "Account (submission replies)" },
  ],
  bluesky: [
    { value: "account", label: "Account (notifications)" },
    { value: "user", label: "User Feed (posts by a user)" },
  ],
};

const DEFAULT_MODE: Record<string, string> = {
  reddit: "subreddit",
  hn: "keywords",
  bluesky: "account",
};

// ── Field definitions per type+mode ──

interface FieldDef {
  key: string;
  label: string;
  type: "text" | "number" | "checkbox";
  placeholder?: string;
  required?: boolean;
  mono?: boolean;
  hint?: string;
}

function getFields(sourceType: MonitorType, mode: string): FieldDef[] {
  if (sourceType === "reddit") {
    const base: FieldDef[] = [];
    if (mode === "subreddit") {
      base.push({ key: "subreddit", label: "Subreddit", type: "text", placeholder: "golang", required: true, hint: "All new posts will be fetched. Use processing rules to filter by keywords, regex, or LLM classification." });
    } else if (mode === "thread") {
      base.push({ key: "thread_id", label: "Thread ID", type: "text", placeholder: "t3_abc123", required: true, mono: true });
      base.push({ key: "subreddit", label: "Subreddit", type: "text", placeholder: "golang" });
    } else if (mode === "account") {
      base.push({ key: "check_inbox", label: "Check inbox (replies & messages)", type: "checkbox" });
      base.push({ key: "check_mentions", label: "Check mentions (u/username)", type: "checkbox" });
    }
    base.push({ key: "poll_interval_secs", label: "Poll interval (seconds)", type: "number", placeholder: "300" });
    return base;
  }
  if (sourceType === "hn") {
    const base: FieldDef[] = [];
    if (mode === "keywords") {
      base.push({ key: "_hint", label: "", type: "text", hint: "Monitors new HN stories. All stories are fetched; use processing rules to filter by keywords, regex, or LLM classification." });
    } else if (mode === "new") {
      base.push({ key: "_hint", label: "", type: "text", hint: "Monitors all new HN stories. Use processing rules to filter by keywords, regex, or LLM classification." });
    } else if (mode === "thread") {
      base.push({ key: "item_id", label: "HN Item ID", type: "number", placeholder: "12345678", required: true, mono: true });
    } else if (mode === "account") {
      base.push({ key: "username", label: "HN Username", type: "text", placeholder: "dang", required: true });
    }
    base.push({ key: "poll_interval_secs", label: "Poll interval (seconds)", type: "number", placeholder: mode === "account" ? "600" : "300" });
    return base;
  }
  if (sourceType === "bluesky") {
    const base: FieldDef[] = [];
    if (mode === "account") {
      base.push({ key: "_hint", label: "", type: "text", hint: "Monitors replies, mentions, and quotes on your Bluesky account. Credentials are taken from your Bluesky platform connection." });
    } else if (mode === "user") {
      base.push({ key: "handle", label: "Bluesky Handle", type: "text", placeholder: "user.bsky.social", required: true, hint: "Monitors all posts by this user. Credentials are taken from your Bluesky platform connection." });
    }
    base.push({ key: "poll_interval_secs", label: "Poll interval (seconds)", type: "number", placeholder: "300" });
    return base;
  }
  if (sourceType === "web") {
    return [
      { key: "urls", label: "URLs (one per line)", type: "text", placeholder: "https://example.com/blog", hint: "Web pages to monitor via changedetection.io. Enter one URL per line." },
    ];
  }
  if (sourceType === "rss") {
    return [
      { key: "feeds", label: "Feed URLs (one per line)", type: "text", placeholder: "https://example.com/feed.xml", hint: "RSS/Atom feeds to monitor. Enter one feed URL per line." },
    ];
  }
  if (sourceType === "github") {
    return [
      { key: "repos", label: "Repositories (one per line)", type: "text", placeholder: "owner/repo", hint: "GitHub repositories to watch. Enter one owner/repo per line." },
    ];
  }
  return [];
}

// ── Helpers ──

function parseConfig(json: string): Record<string, unknown> | null {
  try {
    const parsed = JSON.parse(json.trim() || "{}");
    if (typeof parsed === "object" && parsed !== null && !Array.isArray(parsed)) return parsed;
    return null;
  } catch {
    return null;
  }
}

function configGet(cfg: Record<string, unknown>, key: string): unknown {
  return cfg[key];
}

function configSet(json: string, key: string, value: unknown): string {
  const cfg = parseConfig(json) ?? {};
  if (value === "" || value === undefined) {
    delete cfg[key];
  } else {
    cfg[key] = value;
  }
  return JSON.stringify(cfg, null, 2);
}

function arrayToLines(val: unknown): string {
  if (Array.isArray(val)) return val.join("\n");
  return "";
}

function linesToArray(text: string): string[] {
  return text.split("\n").map(s => s.trim()).filter(Boolean);
}

// ── Array field with local state for natural editing ──

function ArrayField({ label, placeholder, hint, value, onChange }: {
  label: string;
  placeholder?: string;
  hint?: string;
  value: unknown;
  onChange: (arr: string[]) => void;
}) {
  const [localText, setLocalText] = useState<string | null>(null);
  const displayText = localText ?? arrayToLines(value);

  return (
    <div>
      <label className="mb-1 block text-xs font-medium text-gray-600">{label}</label>
      <textarea
        value={displayText}
        onChange={(e) => setLocalText(e.target.value)}
        onBlur={() => {
          if (localText !== null) {
            onChange(linesToArray(localText));
            setLocalText(null);
          }
        }}
        rows={3}
        className="w-full rounded border px-3 py-2 text-sm"
        placeholder={placeholder}
      />
      {hint && <p className="mt-1 text-xs text-gray-500">{hint}</p>}
    </div>
  );
}

// ── Exported helpers for parent pages ──

export { parseConfig };

export function defaultConfigForType(type: MonitorType): string {
  const mode = DEFAULT_MODE[type];
  if (mode) {
    if (type === "reddit" && mode === "subreddit") return JSON.stringify({ mode, subreddit: "", poll_interval_secs: 300 }, null, 2);
    if (type === "hn") return JSON.stringify({ mode, poll_interval_secs: 300 }, null, 2);
    if (type === "bluesky") return JSON.stringify({ mode, poll_interval_secs: 300 }, null, 2);
  }
  return "{}";
}

// ── Main component ──

interface SourceConfigEditorProps {
  sourceType: MonitorType;
  config: string;
  onConfigChange: (config: string) => void;
}

export default function SourceConfigEditor({ sourceType, config, onConfigChange }: SourceConfigEditorProps) {
  const [tab, setTab] = useState<"fields" | "json">("fields");

  const parsed = parseConfig(config);
  const isValidJson = parsed !== null;
  const currentMode = (isValidJson && typeof parsed.mode === "string") ? parsed.mode : (DEFAULT_MODE[sourceType] ?? "");
  const hasModes = sourceType in MODE_OPTIONS;
  const fields = getFields(sourceType, currentMode);

  function handleModeChange(newMode: string) {
    const cfg = parsed ?? {};
    const poll = cfg.poll_interval_secs;
    const next: Record<string, unknown> = { mode: newMode };
    if (poll !== undefined) next.poll_interval_secs = poll;
    if (sourceType === "reddit" && newMode === "account") {
      next.check_inbox = true;
      next.check_mentions = true;
    }
    onConfigChange(JSON.stringify(next, null, 2));
  }

  function setFieldValue(key: string, value: unknown) {
    onConfigChange(configSet(config, key, value));
  }

  return (
    <div>
      <div className="mb-2 flex items-center gap-1">
        <button
          type="button"
          onClick={() => setTab("fields")}
          className={`rounded-full px-3 py-1 text-xs ${
            tab === "fields"
              ? "bg-blue-100 font-medium text-blue-700"
              : "bg-gray-100 text-gray-600 hover:bg-gray-200"
          }`}
        >
          Fields
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
          value={config}
          onChange={(e) => onConfigChange(e.target.value)}
          rows={6}
          className="w-full rounded border px-3 py-2 font-mono text-sm"
          placeholder="{}"
        />
      ) : (
        <div className="space-y-3">
          {!isValidJson && (
            <p className="rounded bg-amber-50 px-3 py-2 text-xs text-amber-700">
              Invalid JSON — switch to JSON tab to fix before editing fields.
            </p>
          )}

          {hasModes && isValidJson && (
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-600">Mode</label>
              <select
                value={currentMode}
                onChange={(e) => handleModeChange(e.target.value)}
                className="rounded border px-3 py-2 text-sm"
              >
                {MODE_OPTIONS[sourceType]?.map((opt) => (
                  <option key={opt.value} value={opt.value}>{opt.label}</option>
                ))}
              </select>
            </div>
          )}

          {isValidJson && fields.map((field) => {
            if (field.key === "_hint") {
              return field.hint ? (
                <p key={field.key + currentMode} className="text-xs text-gray-500">{field.hint}</p>
              ) : null;
            }

            if (field.key === "urls" || field.key === "feeds" || field.key === "repos") {
              return (
                <ArrayField
                  key={field.key}
                  label={field.label}
                  placeholder={field.placeholder}
                  hint={field.hint}
                  value={configGet(parsed!, field.key)}
                  onChange={(arr) => setFieldValue(field.key, arr.length > 0 ? arr : undefined)}
                />
              );
            }

            if (field.type === "checkbox") {
              const checked = configGet(parsed!, field.key) !== false;
              return (
                <label key={field.key} className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    checked={checked}
                    onChange={(e) => setFieldValue(field.key, e.target.checked)}
                  />
                  <span className="text-sm">{field.label}</span>
                </label>
              );
            }

            if (field.type === "number") {
              const val = configGet(parsed!, field.key);
              return (
                <div key={field.key}>
                  <label className="mb-1 block text-xs font-medium text-gray-600">{field.label}</label>
                  <input
                    type="number"
                    value={val !== undefined && val !== null ? String(val) : ""}
                    onChange={(e) => {
                      const num = parseInt(e.target.value);
                      setFieldValue(field.key, isNaN(num) ? undefined : num);
                    }}
                    min={field.key === "poll_interval_secs" ? 60 : undefined}
                    placeholder={field.placeholder}
                    required={field.required}
                    className={`rounded border px-3 py-2 text-sm ${field.key === "poll_interval_secs" ? "w-32" : "w-full"} ${field.mono ? "font-mono" : ""}`}
                  />
                  {field.hint && <p className="mt-1 text-xs text-gray-500">{field.hint}</p>}
                </div>
              );
            }

            const val = configGet(parsed!, field.key);
            return (
              <div key={field.key}>
                <label className="mb-1 block text-xs font-medium text-gray-600">{field.label}</label>
                <input
                  type="text"
                  value={typeof val === "string" ? val : ""}
                  onChange={(e) => setFieldValue(field.key, e.target.value || undefined)}
                  placeholder={field.placeholder}
                  required={field.required}
                  className={`w-full rounded border px-3 py-2 text-sm ${field.mono ? "font-mono" : ""}`}
                />
                {field.hint && <p className="mt-1 text-xs text-gray-500">{field.hint}</p>}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
