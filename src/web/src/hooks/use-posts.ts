import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api, apiMutate } from "@/lib/api";
import type { Post, PostListResponse, PublishResponse } from "@/types/api";

interface PostFilters {
  status?: string;
  limit?: number;
  offset?: number;
}

export function usePosts(projectId: string, filters: PostFilters = {}) {
  const { status, limit = 50, offset = 0 } = filters;
  const params = new URLSearchParams();
  if (status) params.set("status", status);
  params.set("limit", String(limit));
  params.set("offset", String(offset));

  return useQuery({
    queryKey: ["projects", projectId, "posts", { status, limit, offset }],
    queryFn: () => api<PostListResponse>(`/projects/${projectId}/posts?${params}`),
    enabled: !!projectId,
  });
}

export function usePost(projectId: string, postId: string) {
  return useQuery({
    queryKey: ["projects", projectId, "posts", postId],
    queryFn: () => api<Post>(`/projects/${projectId}/posts/${postId}`),
    enabled: !!projectId && !!postId,
  });
}

export function useCreatePost(projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: { title?: string; content: string; url?: string; post_type?: string; platforms: string[]; reply_to_url?: string }) =>
      apiMutate<Post>(`/projects/${projectId}/posts`, "POST", data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["projects", projectId, "posts"] }),
  });
}

export function useUpdatePost(projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ postId, ...data }: { postId: string; title?: string; content: string; url?: string; post_type?: string; platforms: string[]; reply_to_url?: string }) =>
      apiMutate<Post>(`/projects/${projectId}/posts/${postId}`, "PUT", data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["projects", projectId, "posts"] }),
  });
}

export function useDeletePost(projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (postId: string) =>
      apiMutate(`/projects/${projectId}/posts/${postId}`, "DELETE"),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["projects", projectId, "posts"] }),
  });
}

export function usePublishPost(projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (postId: string) =>
      apiMutate<PublishResponse>(`/projects/${projectId}/posts/${postId}/publish`, "POST"),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["projects", projectId, "posts"] }),
  });
}

export function useSetPostMonitor(projectId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ postId, enable }: { postId: string; enable: boolean }) =>
      apiMutate(`/projects/${projectId}/posts/${postId}/monitor`, "PUT", { enable }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["projects", projectId, "posts"] }),
  });
}
