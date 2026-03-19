import { useParams, Link } from "react-router";
import { useAlert, useUpdateAlertStatus } from "@/hooks/use-alerts";
import { useSources } from "@/hooks/use-sources";
import AlertTitle from "@/components/alert-title";
import type { Alert, AlertSeverity, AlertStatus, Source } from "@/types/api";

const SEVERITY_COLORS: Record<AlertSeverity, string> = {
  critical: "bg-red-100 text-red-800",
  high: "bg-orange-100 text-orange-800",
  medium: "bg-yellow-100 text-yellow-800",
  low: "bg-gray-100 text-gray-700",
};

const STATUS_COLORS: Record<AlertStatus, string> = {
  new: "bg-blue-100 text-blue-800",
  seen: "bg-gray-100 text-gray-700",
  triaged: "bg-purple-100 text-purple-800",
  dismissed: "bg-gray-100 text-gray-400",
  responded: "bg-green-100 text-green-800",
};

const STATUSES: AlertStatus[] = ["new", "seen", "triaged", "dismissed", "responded"];

function sourceDetailLabel(src: Source): string {
  const cfg = src.config;
  const detail = cfg.username ?? cfg.subreddit ?? cfg.item_id ?? cfg.url ?? null;
  return detail ? `${src.type} — ${detail}` : src.type;
}

function resolveSource(alert: Alert, sources?: Source[]): { source: Source | null; linkTo: string } {
  if (alert.monitor_source_id && sources) {
    const src = sources.find((s) => s.id === alert.monitor_source_id);
    if (src) return { source: src, linkTo: `../sources/${src.id}` };
  }
  // Fallback: find a source matching this alert's source_type
  if (sources) {
    const byType = sources.filter((s) => s.type === alert.source_type);
    if (byType.length === 1) return { source: byType[0], linkTo: `../sources/${byType[0].id}` };
  }
  return { source: null, linkTo: "../sources" };
}

export default function AlertDetailPage() {
  const { id: projectId, alertId } = useParams<{ id: string; alertId: string }>();
  const { data: alert, isLoading, error } = useAlert(projectId!, alertId!);
  const { data: sources } = useSources(projectId!);
  const updateStatus = useUpdateAlertStatus(projectId!);

  if (isLoading) return <p className="text-gray-500">Loading alert...</p>;
  if (error) return <p className="text-red-600">Error: {error.message}</p>;
  if (!alert) return <p className="text-gray-500">Alert not found.</p>;

  const { source, linkTo } = resolveSource(alert, sources);

  return (
    <div>
      <Link to="../alerts" className="mb-4 inline-block text-sm text-blue-600 hover:underline">&larr; Back to Alerts</Link>

      <div className="mb-4 flex items-center gap-3">
        <h2 className="text-xl font-semibold"><AlertTitle title={alert.title} /></h2>
        <span className={`rounded px-2 py-0.5 text-xs font-medium ${SEVERITY_COLORS[alert.severity]}`}>{alert.severity}</span>
        <span className={`rounded px-2 py-0.5 text-xs font-medium ${STATUS_COLORS[alert.status]}`}>{alert.status}</span>
      </div>

      <div className="space-y-4 rounded border bg-white p-4">
        <div className="grid grid-cols-2 gap-4 text-sm">
          <div>
            <span className="font-medium text-gray-500">Source Type</span>
            <p className="mt-1">{alert.source_type}</p>
          </div>
          <div>
            <span className="font-medium text-gray-500">Source</span>
            <p className="mt-1">
              <Link to={linkTo} className="text-blue-600 hover:underline">
                {source ? sourceDetailLabel(source) : alert.source_type}
              </Link>
            </p>
          </div>
        </div>

        {alert.content && (
          <div>
            <span className="text-sm font-medium text-gray-500">Content</span>
            <p className="mt-1 whitespace-pre-wrap text-sm text-gray-700">{alert.content}</p>
          </div>
        )}

        {alert.url && (
          <div>
            <span className="text-sm font-medium text-gray-500">URL</span>
            <p className="mt-1">
              <a href={alert.url} target="_blank" rel="noopener noreferrer" className="text-sm text-blue-600 hover:underline">
                {alert.url}
              </a>
            </p>
          </div>
        )}

        {alert.matched_keywords.length > 0 && (
          <div>
            <Link to="../keywords" className="text-sm font-medium text-blue-600 hover:underline">Matched Keywords</Link>
            <div className="mt-1 flex flex-wrap gap-1">
              {alert.matched_keywords.map((kw) => (
                <span key={kw} className="rounded bg-yellow-50 px-2 py-0.5 text-xs text-yellow-700">{kw}</span>
              ))}
            </div>
          </div>
        )}

        {alert.tags && alert.tags.length > 0 && (
          <div>
            <span className="text-sm font-medium text-gray-500">Tags</span>
            <div className="mt-1 flex flex-wrap gap-1">
              {alert.tags.map((tag) => (
                <span key={tag} className="rounded bg-blue-50 px-2 py-0.5 text-xs text-blue-700">{tag}</span>
              ))}
            </div>
          </div>
        )}

        {alert.classification_reason && (
          <div>
            <Link to="../processing" className="text-sm font-medium text-blue-600 hover:underline">Classification</Link>
            <p className="mt-1 text-sm text-gray-700">{alert.classification_reason}</p>
          </div>
        )}

        <div className="grid grid-cols-2 gap-4 text-sm">
          <div>
            <span className="font-medium text-gray-500">Created</span>
            <p className="mt-1">{new Date(alert.created_at).toLocaleString()}</p>
          </div>
          <div>
            <span className="font-medium text-gray-500">Updated</span>
            <p className="mt-1">{new Date(alert.updated_at).toLocaleString()}</p>
          </div>
        </div>

        <div>
          <span className="text-sm font-medium text-gray-500">Set Status</span>
          <div className="mt-2 flex gap-2">
            {STATUSES.filter((s) => s !== alert.status).map((s) => (
              <button
                key={s}
                onClick={() => updateStatus.mutate({ alertId: alert.id, status: s })}
                disabled={updateStatus.isPending}
                className="rounded border px-3 py-1 text-xs hover:bg-gray-50 disabled:opacity-50"
              >
                {s}
              </button>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
