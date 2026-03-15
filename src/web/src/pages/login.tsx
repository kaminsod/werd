import { useState, type FormEvent } from "react";
import { Navigate, useNavigate } from "react-router";
import { useLogin } from "@/hooks/use-auth";
import { useAuthStore } from "@/stores/auth";

export default function LoginPage() {
  const token = useAuthStore((s) => s.token);
  const navigate = useNavigate();
  const login = useLogin();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");

  if (token) return <Navigate to="/projects" replace />;

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    login.mutate(
      { email, password },
      { onSuccess: () => navigate("/projects") },
    );
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50">
      <form
        onSubmit={handleSubmit}
        className="w-full max-w-sm space-y-4 rounded-lg border bg-white p-8 shadow-sm"
      >
        <h1 className="text-center text-2xl font-bold">Werd</h1>
        <p className="text-center text-sm text-gray-500">Sign in to your account</p>

        {login.error && (
          <p className="rounded bg-red-50 p-2 text-sm text-red-600">
            {login.error.message}
          </p>
        )}

        <div>
          <label htmlFor="email" className="mb-1 block text-sm font-medium">
            Email
          </label>
          <input
            id="email"
            type="email"
            required
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            className="w-full rounded border px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>

        <div>
          <label htmlFor="password" className="mb-1 block text-sm font-medium">
            Password
          </label>
          <input
            id="password"
            type="password"
            required
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className="w-full rounded border px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>

        <button
          type="submit"
          disabled={login.isPending}
          className="w-full rounded bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
        >
          {login.isPending ? "Signing in..." : "Sign in"}
        </button>
      </form>
    </div>
  );
}
