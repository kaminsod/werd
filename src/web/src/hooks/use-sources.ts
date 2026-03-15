import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api, apiMutate } from "@/lib/api";
import type { Source } from "@/types/api";

export function useSources(projectId: string) {
  return useQuery({
    queryKey: ["projects", projectId, "sources"],
    queryFn: () => api<Source[]>(`/projects/${projectId}/sources`),
    enabled: !!projectId,
  });
}

export function useCreateSource(projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: { type: string; config: Record<string, unknown>; enabled?: boolean }) =>
      apiMutate<Source>(`/projects/${projectId}/sources`, "POST", data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["projects", projectId, "sources"] }),
  });
}

export function useUpdateSource(projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ sourceId, ...data }: { sourceId: string; type: string; config: Record<string, unknown>; enabled?: boolean }) =>
      apiMutate<Source>(`/projects/${projectId}/sources/${sourceId}`, "PUT", data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["projects", projectId, "sources"] }),
  });
}

export function useDeleteSource(projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (sourceId: string) =>
      apiMutate(`/projects/${projectId}/sources/${sourceId}`, "DELETE"),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["projects", projectId, "sources"] }),
  });
}
