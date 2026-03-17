import { useState } from "react";
import { useParams } from "react-router";
import { useAlerts, useUpdateAlertStatus } from "@/hooks/use-alerts";
import InfoIcon from "@/components/info-icon";
import { alertSeverity as severityHelp, alertStatus as statusHelp } from "@/lib/help-content";
import type { Alert, AlertSeverity, AlertStatus } from "@/types/api";

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

function AlertRow({
  alert,
  expanded,
  onToggle,
  onStatusChange,
}: {
  alert: Alert;
  expanded: boolean;
  onToggle: () => void;
  onStatusChange: (alertId: string, status: string) => void;
}) {
  return (
    <div className="rounded border bg-white">
      <button
        onClick={onToggle}
        className="flex w-full items-center gap-3 p-3 text-left hover:bg-gray-50"
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
          {alert.title || "(no title)"}
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
      </button>

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

          {alert.matched_keywords.length > 0 && (
            <div className="mb-3">
              <span className="text-xs font-medium text-gray-500">Matched keywords: </span>
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
              <span className="text-xs font-medium text-gray-500">Classification: </span>
              <span className="text-xs text-gray-600">{alert.classification_reason}</span>
            </div>
          )}

          <div className="flex items-center gap-2">
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
