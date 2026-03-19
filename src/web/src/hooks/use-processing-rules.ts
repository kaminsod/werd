import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api, apiMutate } from "@/lib/api";
import type { ProcessingRule } from "@/types/api";

export function useProcessingRule(projectId: string, ruleId: string) {
  return useQuery({
    queryKey: ["projects", projectId, "processing-rules", ruleId],
    queryFn: () => api<ProcessingRule>(`/projects/${projectId}/processing-rules/${ruleId}`),
    enabled: !!projectId && !!ruleId,
  });
}

export function useProcessingRules(projectId: string) {
  return useQuery({
    queryKey: ["projects", projectId, "processing-rules"],
    queryFn: () => api<ProcessingRule[]>(`/projects/${projectId}/processing-rules`),
    enabled: !!projectId,
  });
}

export function useCreateProcessingRule(projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: {
      source_id?: string;
      name: string;
      phase: string;
      rule_type: string;
      config: Record<string, unknown>;
      priority?: number;
      enabled?: boolean;
    }) =>
      apiMutate<ProcessingRule>(`/projects/${projectId}/processing-rules`, "POST", data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["projects", projectId, "processing-rules"] }),
  });
}

export function useUpdateProcessingRule(projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      ruleId,
      ...data
    }: {
      ruleId: string;
      source_id?: string;
      name: string;
      phase: string;
      rule_type: string;
      config: Record<string, unknown>;
      priority?: number;
      enabled?: boolean;
    }) =>
      apiMutate<ProcessingRule>(`/projects/${projectId}/processing-rules/${ruleId}`, "PUT", data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["projects", projectId, "processing-rules"] }),
  });
}

export function useDeleteProcessingRule(projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (ruleId: string) =>
      apiMutate(`/projects/${projectId}/processing-rules/${ruleId}`, "DELETE"),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["projects", projectId, "processing-rules"] }),
  });
}
