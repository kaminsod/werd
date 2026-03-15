import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api, apiMutate } from "@/lib/api";
import type { Member } from "@/types/api";

export function useMembers(projectId: string) {
  return useQuery({
    queryKey: ["projects", projectId, "members"],
    queryFn: () => api<Member[]>(`/projects/${projectId}/members`),
    enabled: !!projectId,
  });
}

export function useAddMember(projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: { user_id: string; role: string }) =>
      apiMutate<Member>(`/projects/${projectId}/members`, "POST", data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["projects", projectId, "members"] }),
  });
}

export function useUpdateMemberRole(projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ userId, role }: { userId: string; role: string }) =>
      apiMutate(`/projects/${projectId}/members/${userId}`, "PUT", { role }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["projects", projectId, "members"] }),
  });
}

export function useRemoveMember(projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (userId: string) =>
      apiMutate(`/projects/${projectId}/members/${userId}`, "DELETE"),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["projects", projectId, "members"] }),
  });
}
