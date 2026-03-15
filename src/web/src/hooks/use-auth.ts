import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api, apiMutate } from "@/lib/api";
import { useAuthStore } from "@/stores/auth";
import type { LoginRequest, LoginResponse, User } from "@/types/api";

export function useLogin() {
  const setAuth = useAuthStore((s) => s.setAuth);

  return useMutation({
    mutationFn: (data: LoginRequest) =>
      apiMutate<LoginResponse>("/auth/login", "POST", data),
    onSuccess: (res) => setAuth(res.token, res.user),
  });
}

export function useMe() {
  const token = useAuthStore((s) => s.token);

  return useQuery({
    queryKey: ["auth", "me"],
    queryFn: () => api<User>("/auth/me"),
    enabled: !!token,
    retry: false,
  });
}

export function useLogout() {
  const clearAuth = useAuthStore((s) => s.clearAuth);
  const queryClient = useQueryClient();

  return () => {
    clearAuth();
    queryClient.clear();
  };
}

export function useChangePassword() {
  return useMutation({
    mutationFn: (data: { current_password: string; new_password: string }) =>
      apiMutate("/auth/me/password", "PUT", data),
  });
}
