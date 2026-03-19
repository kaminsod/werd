import { useState } from "react";
import { useParams, Link } from "react-router";
import { useAlerts, useUpdateAlertStatus } from "@/hooks/use-alerts";
import { useSources } from "@/hooks/use-sources";
import InfoIcon from "@/components/info-icon";
import AlertTitle from "@/components/alert-title";
import { alertSeverity as severityHelp, alertStatus as statusHelp } from "@/lib/help-content";
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

const SOURCE_TYPES = ["reddit", "hn", "web", "rss", "github", "bluesky"];
const PAGE_SIZE = 20;

export default function AlertsPage() {
  const { id: projectId } = useParams<{ id: string }>();
  const [statusFilter, setStatusFilter] = useState("");
  const [sourceFilter, setSourceFilter] = useState("");
  const [page, setPage] = useState(0);
  const [expandedId, setExpandedId] = useState<string | null>(null);

  const { data, isLoading, error } = useAlerts(projectId!, {
    status: statusFilter || undefined,
    source_type: sourceFilter || undefined,
    limit: PAGE_SIZE,
    offset: page * PAGE_SIZE,
  });

  const { data: sources } = useSources(projectId!);
  const updateStatus = useUpdateAlertStatus(projectId!);

  function handleStatusChange(alertId: string, status: string) {
    updateStatus.mutate({ alertId, status });
  }

  if (isLoading) return <p className="text-gray-500">Loading alerts...</p>;
  if (error) return <p className="text-red-600">Error: {error.message}</p>;

  const { alerts, total } = data!;
  const totalPages = Math.ceil(total / PAGE_SIZE);

  return (
    <div>
      <h2 className="mb-4 text-xl font-semibold">
        Alerts
        <InfoIcon tooltip={severityHelp.tooltip}>{severityHelp.modal}</InfoIcon>
      </h2>

      {/* Filters */}
      <div className="mb-4 flex gap-3">
        <select
          value={statusFilter}
          onChange={(e) => { setStatusFilter(e.target.value); setPage(0); }}
          className="rounded border px-3 py-1.5 text-sm"
        >
          <option value="">All statuses</option>
          {STATUSES.map((s) => (
            <option key={s} value={s}>{s}</option>
          ))}
        </select>

        <select
          value={sourceFilter}
          onChange={(e) => { setSourceFilter(e.target.value); setPage(0); }}
          className="rounded border px-3 py-1.5 text-sm"
        >
          <option value="">All sources</option>
          {SOURCE_TYPES.map((s) => (
            <option key={s} value={s}>{s}</option>
          ))}
        </select>

        <span className="ml-auto text-sm text-gray-500">
          {total} alert{total !== 1 ? "s" : ""}
        </span>
      </div>

      {/* Alert list */}
      {alerts.length === 0 ? (
        <p className="text-gray-500">No alerts found.</p>
      ) : (
        <div className="space-y-2">
          {alerts.map((alert) => (
            <AlertRow
              key={alert.id}
              alert={alert}
              sources={sources}
              expanded={expandedId === alert.id}
              onToggle={() => setExpandedId(expandedId === alert.id ? null : alert.id)}
              onStatusChange={handleStatusChange}
            />
          ))}
        </div>
      )}

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="mt-4 flex items-center justify-between">
          <button
            disabled={page === 0}
            onClick={() => setPage(page - 1)}
            className="rounded border px-3 py-1 text-sm disabled:opacity-40"
          >
            Previous
          </button>
          <span className="text-sm text-gray-500">
            Page {page + 1} of {totalPages}
          </span>
          <button
            disabled={page >= totalPages - 1}
            onClick={() => setPage(page + 1)}
            className="rounded border px-3 py-1 text-sm disabled:opacity-40"
          >
            Next
          </button>
        </div>
      )}
    </div>
  );
}

function sourceDetailLabel(src: Source): string {
  const cfg = src.config;
  const detail = cfg.username ?? cfg.subreddit ?? cfg.item_id ?? cfg.url ?? null;
  return detail ? `${src.type} — ${detail}` : src.type;
}

function resolveSource(alert: Alert, sources?: Source[]): { label: string; linkTo: string } {
  if (alert.monitor_source_id && sources) {
    const src = sources.find((s) => s.id === alert.monitor_source_id);
    if (src) return { label: sourceDetailLabel(src), linkTo: `../sources/${src.id}` };
  }
  // Fallback: find a source matching this alert's source_type
  if (sources) {
    const byType = sources.filter((s) => s.type === alert.source_type);
    if (byType.length === 1) return { label: sourceDetailLabel(byType[0]), linkTo: `../sources/${byType[0].id}` };
  }
  return { label: alert.source_type, linkTo: "../sources" };
}

function AlertRow({
  alert,
  sources,
  expanded,
  onToggle,
  onStatusChange,
}: {
  alert: Alert;
  sources?: Source[];
  expanded: boolean;
  onToggle: () => void;
  onStatusChange: (alertId: string, status: string) => void;
}) {
  const resolved = resolveSource(alert, sources);

  return (
    <div className="rounded border bg-white">
      <div
        onClick={onToggle}
        className="flex w-full cursor-pointer items-center gap-3 p-3 text-left hover:bg-gray-50"
      >
        <span className={`shrink-0 rounded px-2 py-0.5 text-xs font-medium ${SEVERITY_COLORS[alert.severity]}`}>
          {alert.severity}
        </span>
        <span className={`shrink-0 rounded px-2 py-0.5 text-xs font-medium ${STATUS_COLORS[alert.status]}`}>
          {alert.status}
        </span>
        <span className="shrink-0 rounded bg-gray-50 px-2 py-0.5 text-xs text-gray-500">
          {alert.source_type}
        </span>
        <span className="min-w-0 flex-1 truncate text-sm font-medium">
          <AlertTitle title={alert.title} />
        </span>
        {alert.tags && alert.tags.length > 0 && (
          <span className="flex shrink-0 gap-1">
            {alert.tags.map((tag) => (
              <span key={tag} className="rounded bg-blue-50 px-1.5 py-0.5 text-xs text-blue-700">
                {tag}
              </span>
            ))}
          </span>
        )}
        {alert.matched_keywords.length > 0 && (
          <span className="shrink-0 text-xs text-gray-400">
            {alert.matched_keywords.length} keyword{alert.matched_keywords.length !== 1 ? "s" : ""}
          </span>
        )}
        <span className="shrink-0 text-xs text-gray-400">
          {new Date(alert.created_at).toLocaleDateString()}
        </span>
      </div>

      {expanded && (
        <div className="border-t px-4 py-3">
          {alert.content && (
            <p className="mb-3 whitespace-pre-wrap text-sm text-gray-700">{alert.content}</p>
          )}

          {alert.url && (
            <p className="mb-3">
              <a href={alert.url} target="_blank" rel="noopener noreferrer" className="text-sm text-blue-600 hover:underline">
                {alert.url}
              </a>
            </p>
          )}

          <div className="mb-3">
            <Link to={resolved.linkTo} className="text-xs font-medium text-blue-600 hover:underline">Source: </Link>
            <span className="text-xs text-gray-600">{resolved.label}</span>
          </div>

          {alert.matched_keywords.length > 0 && (
            <div className="mb-3">
              <Link to="../keywords" className="text-xs font-medium text-blue-600 hover:underline">Matched keywords: </Link>
              {alert.matched_keywords.map((kw) => (
                <span key={kw} className="mr-1 rounded bg-yellow-50 px-1.5 py-0.5 text-xs text-yellow-700">
                  {kw}
                </span>
              ))}
            </div>
          )}

          {alert.tags && alert.tags.length > 0 && (
            <div className="mb-3">
              <span className="text-xs font-medium text-gray-500">Tags: </span>
              {alert.tags.map((tag) => (
                <span key={tag} className="mr-1 rounded bg-blue-50 px-1.5 py-0.5 text-xs text-blue-700">
                  {tag}
                </span>
              ))}
            </div>
          )}

          {alert.classification_reason && (
            <div className="mb-3">
              <Link to="../processing" className="text-xs font-medium text-blue-600 hover:underline">Classification: </Link>
              <span className="text-xs text-gray-600">{alert.classification_reason}</span>
            </div>
          )}

          <div className="flex items-center gap-2">
            <Link to={alert.id} className="rounded border border-blue-200 px-2 py-1 text-xs text-blue-600 hover:bg-blue-50">
              View details
            </Link>
            <span className="text-xs text-gray-500">
            Set status:
            <InfoIcon tooltip={statusHelp.tooltip}>{statusHelp.modal}</InfoIcon>
          </span>
            {STATUSES.filter((s) => s !== alert.status).map((s) => (
              <button
                key={s}
                onClick={() => onStatusChange(alert.id, s)}
                className="rounded border px-2 py-1 text-xs hover:bg-gray-50"
              >
                {s}
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
