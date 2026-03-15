import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api, apiMutate } from "@/lib/api";
import type { Project } from "@/types/api";

export function useProjects() {
  return useQuery({
    queryKey: ["projects"],
    queryFn: () => api<Project[]>("/projects"),
  });
}

export function useProject(id: string) {
  return useQuery({
    queryKey: ["projects", id],
    queryFn: () => api<Project>(`/projects/${id}`),
    enabled: !!id,
  });
}

export function useCreateProject() {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: (data: { name: string; slug: string }) =>
      apiMutate<Project>("/projects", "POST", data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["projects"] }),
  });
}

export function useUpdateProject(id: string) {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: (data: { name: string; slug: string; settings?: Record<string, unknown> }) =>
      apiMutate<Project>(`/projects/${id}`, "PUT", data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["projects"] });
      qc.invalidateQueries({ queryKey: ["projects", id] });
    },
  });
}

export function useDeleteProject(id: string) {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: () => apiMutate(`/projects/${id}`, "DELETE"),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["projects"] }),
  });
}
