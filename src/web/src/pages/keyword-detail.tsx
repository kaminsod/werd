import { useParams, Link, useNavigate } from "react-router";
import { useKeywords, useDeleteKeyword } from "@/hooks/use-keywords";
import type { KeywordMatchType } from "@/types/api";

const MATCH_TYPE_COLORS: Record<KeywordMatchType, string> = {
  exact: "bg-blue-100 text-blue-700",
  substring: "bg-green-100 text-green-700",
  regex: "bg-purple-100 text-purple-700",
};

export default function KeywordDetailPage() {
  const { id: projectId, kwId } = useParams<{ id: string; kwId: string }>();
  const navigate = useNavigate();
  const { data: keywords, isLoading, error } = useKeywords(projectId!);
  const deleteKeyword = useDeleteKeyword(projectId!);

  const keyword = keywords?.find((kw) => kw.id === kwId);

  function handleDelete() {
    if (keyword && confirm(`Delete keyword "${keyword.keyword}"?`)) {
      deleteKeyword.mutate(kwId!, { onSuccess: () => navigate("../keywords") });
    }
  }

  if (isLoading) return <p className="text-gray-500">Loading keyword...</p>;
  if (error) return <p className="text-red-600">Error: {error.message}</p>;
  if (!keyword) return (
    <div>
      <Link to="../keywords" className="mb-4 inline-block text-sm text-blue-600 hover:underline">&larr; Back to Keywords</Link>
      <p className="text-gray-500">Keyword not found.</p>
    </div>
  );

  return (
    <div>
      <Link to="../keywords" className="mb-4 inline-block text-sm text-blue-600 hover:underline">&larr; Back to Keywords</Link>

      <div className="mb-4 flex items-center gap-3">
        <h2 className="text-xl font-semibold font-mono">{keyword.keyword}</h2>
        <span className={`rounded px-2 py-0.5 text-xs font-medium ${MATCH_TYPE_COLORS[keyword.match_type]}`}>{keyword.match_type}</span>
      </div>

      <div className="space-y-4 rounded border bg-white p-4">
        <div className="grid grid-cols-2 gap-4 text-sm">
          <div>
            <span className="font-medium text-gray-500">Keyword</span>
            <p className="mt-1 font-mono">{keyword.keyword}</p>
          </div>
          <div>
            <span className="font-medium text-gray-500">Match Type</span>
            <p className="mt-1">{keyword.match_type}</p>
          </div>
        </div>

        <div className="text-sm">
          <span className="font-medium text-gray-500">Created</span>
          <p className="mt-1">{new Date(keyword.created_at).toLocaleString()}</p>
        </div>

        <div className="flex gap-2 border-t pt-4">
          <button onClick={handleDelete} className="rounded px-2 py-1 text-xs text-red-600 hover:bg-red-50">Delete</button>
        </div>
      </div>
    </div>
  );
}
