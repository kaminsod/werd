import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api, apiMutate } from "@/lib/api";
import type { Connection } from "@/types/api";

export function useConnections(projectId: string) {
  return useQuery({
    queryKey: ["projects", projectId, "connections"],
    queryFn: () => api<Connection[]>(`/projects/${projectId}/connections`),
    enabled: !!projectId,
  });
}

export function useCreateConnection(projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: { platform: string; credentials: unknown; enabled?: boolean }) =>
      apiMutate<Connection>(`/projects/${projectId}/connections`, "POST", data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["projects", projectId, "connections"] }),
  });
}

export function useUpdateConnection(projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ connId, ...data }: { connId: string; platform: string; credentials: unknown; enabled?: boolean }) =>
      apiMutate<Connection>(`/projects/${projectId}/connections/${connId}`, "PUT", data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["projects", projectId, "connections"] }),
  });
}

export function useDeleteConnection(projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (connId: string) =>
      apiMutate(`/projects/${projectId}/connections/${connId}`, "DELETE"),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["projects", projectId, "connections"] }),
  });
}
