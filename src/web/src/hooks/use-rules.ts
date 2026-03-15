import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api, apiMutate } from "@/lib/api";
import type { Rule } from "@/types/api";

export function useRules(projectId: string) {
  return useQuery({
    queryKey: ["projects", projectId, "rules"],
    queryFn: () => api<Rule[]>(`/projects/${projectId}/rules`),
    enabled: !!projectId,
  });
}

export function useCreateRule(projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: {
      source_type?: string;
      min_severity?: string;
      destination: string;
      config: Record<string, unknown>;
      enabled?: boolean;
    }) => apiMutate<Rule>(`/projects/${projectId}/rules`, "POST", data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["projects", projectId, "rules"] }),
  });
}

export function useUpdateRule(projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ ruleId, ...data }: {
      ruleId: string;
      source_type: string;
      min_severity: string;
      destination: string;
      config: Record<string, unknown>;
      enabled?: boolean;
    }) => apiMutate<Rule>(`/projects/${projectId}/rules/${ruleId}`, "PUT", data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["projects", projectId, "rules"] }),
  });
}

export function useDeleteRule(projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (ruleId: string) =>
      apiMutate(`/projects/${projectId}/rules/${ruleId}`, "DELETE"),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["projects", projectId, "rules"] }),
  });
}
