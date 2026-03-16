import { useState, type FormEvent } from "react";
import { useParams } from "react-router";
import { useKeywords, useCreateKeyword, useDeleteKeyword } from "@/hooks/use-keywords";
import InfoIcon from "@/components/info-icon";
import { matchType as matchTypeHelp, navKeywords as keywordsHelp } from "@/lib/help-content";
import type { KeywordMatchType } from "@/types/api";

const MATCH_TYPES: KeywordMatchType[] = ["exact", "substring", "regex"];

const MATCH_TYPE_COLORS: Record<KeywordMatchType, string> = {
  exact: "bg-blue-100 text-blue-700",
  substring: "bg-green-100 text-green-700",
  regex: "bg-purple-100 text-purple-700",
};

export default function KeywordsPage() {
  const { id: projectId } = useParams<{ id: string }>();
  const { data: keywords, isLoading, error } = useKeywords(projectId!);
  const createKeyword = useCreateKeyword(projectId!);
  const deleteKeyword = useDeleteKeyword(projectId!);

  const [keyword, setKeyword] = useState("");
  const [matchType, setMatchType] = useState<KeywordMatchType>("substring");

  function handleCreate(e: FormEvent) {
    e.preventDefault();
    createKeyword.mutate(
      { keyword, match_type: matchType },
      { onSuccess: () => setKeyword("") },
    );
  }

  function handleDelete(kwId: string, kwText: string) {
    if (confirm(`Delete keyword "${kwText}"?`)) {
      deleteKeyword.mutate(kwId);
    }
  }

  if (isLoading) return <p className="text-gray-500">Loading keywords...</p>;
  if (error) return <p className="text-red-600">Error: {error.message}</p>;

  return (
    <div>
      <h2 className="mb-2 text-xl font-semibold">
        Keywords
        <InfoIcon tooltip={keywordsHelp.tooltip}>{keywordsHelp.modal}</InfoIcon>
      </h2>
      <p className="mb-4 text-sm text-gray-500">
        Keywords are matched against incoming alerts at ingest time. When a monitor source detects
        new content (e.g. a Reddit post, HN story, or web page change), Werd checks the alert's title
        and body against all keywords defined here. Matched keywords are displayed on each alert for
        quick triage. Use <strong>substring</strong> for general monitoring, <strong>exact</strong> for
        precise terms, or <strong>regex</strong> for complex patterns.
      </p>

      {/* Add keyword form */}
      <form onSubmit={handleCreate} className="mb-6 flex items-end gap-3">
        <div className="flex-1">
          <label className="mb-1 block text-sm font-medium text-gray-700">Keyword</label>
          <input
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            required
            placeholder="Enter keyword or pattern..."
            className="w-full rounded border px-3 py-2 text-sm"
          />
        </div>
        <div>
          <label className="mb-1 block text-sm font-medium text-gray-700">
            Match type
            <InfoIcon tooltip={matchTypeHelp.tooltip}>{matchTypeHelp.modal}</InfoIcon>
          </label>
          <select
            value={matchType}
            onChange={(e) => setMatchType(e.target.value as KeywordMatchType)}
            className="rounded border px-3 py-2 text-sm"
          >
            {MATCH_TYPES.map((t) => (
              <option key={t} value={t}>{t}</option>
            ))}
          </select>
        </div>
        <button
          type="submit"
          disabled={createKeyword.isPending}
          className="rounded bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
        >
          {createKeyword.isPending ? "Adding..." : "Add"}
        </button>
      </form>

      {createKeyword.error && (
        <p className="mb-4 text-sm text-red-600">{createKeyword.error.message}</p>
      )}

      {/* Keyword list */}
      {keywords!.length === 0 ? (
        <p className="text-gray-500">No keywords configured. Add keywords to match against incoming alerts.</p>
      ) : (
        <div className="space-y-2">
          {keywords!.map((kw) => (
            <div key={kw.id} className="flex items-center justify-between rounded border bg-white p-3">
              <div className="flex items-center gap-3">
                <span className={`rounded px-2 py-0.5 text-xs font-medium ${MATCH_TYPE_COLORS[kw.match_type]}`}>
                  {kw.match_type}
                </span>
                <span className="text-sm font-mono">{kw.keyword}</span>
              </div>
              <div className="flex items-center gap-3">
                <span className="text-xs text-gray-400">
                  {new Date(kw.created_at).toLocaleDateString()}
                </span>
                <button
                  onClick={() => handleDelete(kw.id, kw.keyword)}
                  disabled={deleteKeyword.isPending}
                  className="rounded px-2 py-1 text-xs text-red-600 hover:bg-red-50"
                >
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
