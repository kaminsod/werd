import { Navigate, Outlet } from "react-router";
import { useAuthStore } from "@/stores/auth";
import { useMe } from "@/hooks/use-auth";

export default function ProtectedRoute() {
  const token = useAuthStore((s) => s.token);

  // Validate token on mount (auto-clears on 401).
  useMe();

  if (!token) {
    return <Navigate to="/login" replace />;
  }

  return <Outlet />;
}
