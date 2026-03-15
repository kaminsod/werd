import { useState, type FormEvent } from "react";
import { Link } from "react-router";
import { useProjects, useCreateProject } from "@/hooks/use-projects";
import { useAuthStore } from "@/stores/auth";
import { useLogout } from "@/hooks/use-auth";

export default function ProjectsPage() {
  const { data: projects, isLoading, error } = useProjects();
  const createProject = useCreateProject();
  const user = useAuthStore((s) => s.user);
  const logout = useLogout();

  const [showForm, setShowForm] = useState(false);
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");

  function handleCreate(e: FormEvent) {
    e.preventDefault();
    createProject.mutate(
      { name, slug },
      {
        onSuccess: () => {
          setName("");
          setSlug("");
          setShowForm(false);
        },
      },
    );
  }

  return (
    <div className="mx-auto max-w-3xl p-6">
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold">Projects</h1>
        <div className="flex items-center gap-3">
          <span className="text-sm text-gray-500">{user?.email}</span>
          <button
            onClick={logout}
            className="rounded border px-3 py-1 text-sm text-gray-600 hover:bg-gray-50"
          >
            Logout
          </button>
        </div>
      </div>

      <button
        onClick={() => setShowForm(!showForm)}
        className="mb-4 rounded bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
      >
        {showForm ? "Cancel" : "New Project"}
      </button>

      {showForm && (
        <form onSubmit={handleCreate} className="mb-6 space-y-3 rounded border bg-gray-50 p-4">
          {createProject.error && (
            <p className="text-sm text-red-600">{createProject.error.message}</p>
          )}
          <div className="flex gap-3">
            <input
              placeholder="Project name"
              required
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="flex-1 rounded border px-3 py-2 text-sm"
            />
            <input
              placeholder="slug"
              required
              value={slug}
              onChange={(e) => setSlug(e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, ""))}
              className="w-48 rounded border px-3 py-2 text-sm font-mono"
            />
          </div>
          <button
            type="submit"
            disabled={createProject.isPending}
            className="rounded bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
          >
            {createProject.isPending ? "Creating..." : "Create"}
          </button>
        </form>
      )}

      {isLoading && <p className="text-gray-500">Loading projects...</p>}
      {error && <p className="text-red-600">Error: {error.message}</p>}

      {projects && projects.length === 0 && (
        <p className="text-gray-500">No projects yet. Create one to get started.</p>
      )}

      {projects && projects.length > 0 && (
        <ul className="space-y-2">
          {projects.map((p) => (
            <li key={p.id}>
              <Link
                to={`/projects/${p.id}`}
                className="block rounded border bg-white p-4 hover:border-blue-300 hover:bg-blue-50"
              >
                <div className="flex items-center justify-between">
                  <div>
                    <div className="font-medium">{p.name}</div>
                    <div className="text-sm text-gray-500">{p.slug}</div>
                  </div>
                  <div className="text-xs text-gray-400">
                    {new Date(p.created_at).toLocaleDateString()}
                  </div>
                </div>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
