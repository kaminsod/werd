import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api, apiMutate } from "@/lib/api";
import type { Keyword } from "@/types/api";

export function useKeywords(projectId: string) {
  return useQuery({
    queryKey: ["projects", projectId, "keywords"],
    queryFn: () => api<Keyword[]>(`/projects/${projectId}/keywords`),
    enabled: !!projectId,
  });
}

export function useCreateKeyword(projectId: string) {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: (data: { keyword: string; match_type?: string }) =>
      apiMutate<Keyword>(`/projects/${projectId}/keywords`, "POST", data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["projects", projectId, "keywords"] });
    },
  });
}

export function useDeleteKeyword(projectId: string) {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: (kwId: string) =>
      apiMutate(`/projects/${projectId}/keywords/${kwId}`, "DELETE"),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["projects", projectId, "keywords"] });
    },
  });
}
