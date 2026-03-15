import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api, apiMutate } from "@/lib/api";
import type { Alert, AlertListResponse } from "@/types/api";

interface AlertFilters {
  status?: string;
  source_type?: string;
  limit?: number;
  offset?: number;
}

export function useAlerts(projectId: string, filters: AlertFilters = {}) {
  const { status, source_type, limit = 50, offset = 0 } = filters;

  const params = new URLSearchParams();
  if (status) params.set("status", status);
  if (source_type) params.set("source_type", source_type);
  params.set("limit", String(limit));
  params.set("offset", String(offset));

  return useQuery({
    queryKey: ["projects", projectId, "alerts", { status, source_type, limit, offset }],
    queryFn: () => api<AlertListResponse>(`/projects/${projectId}/alerts?${params}`),
    enabled: !!projectId,
  });
}

export function useAlert(projectId: string, alertId: string) {
  return useQuery({
    queryKey: ["projects", projectId, "alerts", alertId],
    queryFn: () => api<Alert>(`/projects/${projectId}/alerts/${alertId}`),
    enabled: !!projectId && !!alertId,
  });
}

export function useUpdateAlertStatus(projectId: string) {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: ({ alertId, status }: { alertId: string; status: string }) =>
      apiMutate<Alert>(`/projects/${projectId}/alerts/${alertId}`, "PUT", { status }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["projects", projectId, "alerts"] });
    },
  });
}
