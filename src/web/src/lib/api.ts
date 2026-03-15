import { useAuthStore } from "@/stores/auth";

const BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "";

class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const token = useAuthStore.getState().token;

  const headers: Record<string, string> = {
    ...(options?.headers as Record<string, string>),
  };

  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const res = await fetch(`${BASE_URL}${path}`, { ...options, headers });

  if (res.status === 401) {
    useAuthStore.getState().clearAuth();
    throw new ApiError(401, "Session expired");
  }

  if (!res.ok) {
    let message = `Request failed (${res.status})`;
    try {
      const body = await res.json();
      if (body.message) message = body.message;
    } catch {
      // ignore parse errors
    }
    throw new ApiError(res.status, message);
  }

  return res.json();
}

export function api<T>(path: string): Promise<T> {
  return request<T>(path);
}

export function apiMutate<T>(
  path: string,
  method: string,
  body?: unknown,
): Promise<T> {
  return request<T>(path, {
    method,
    headers: { "Content-Type": "application/json" },
    body: body ? JSON.stringify(body) : undefined,
  });
}
