import { useState, useEffect, type FormEvent } from "react";
import { useParams, useNavigate } from "react-router";
import { useProject, useUpdateProject, useDeleteProject } from "@/hooks/use-projects";

export default function SettingsPage() {
  const { id: projectId } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { data: project, isLoading, error } = useProject(projectId!);
  const updateProject = useUpdateProject(projectId!);
  const deleteProject = useDeleteProject(projectId!);

  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");

  useEffect(() => {
    if (project) {
      setName(project.name);
      setSlug(project.slug);
    }
  }, [project]);

  function handleUpdate(e: FormEvent) {
    e.preventDefault();
    updateProject.mutate({ name, slug, settings: project?.settings });
  }

  function handleDelete() {
    if (confirm(`Delete project "${project?.name}"? This cannot be undone.`)) {
      deleteProject.mutate(undefined, {
        onSuccess: () => navigate("/projects"),
      });
    }
  }

  if (isLoading) return <p className="text-gray-500">Loading settings...</p>;
  if (error) return <p className="text-red-600">Error: {error.message}</p>;

  return (
    <div className="max-w-lg">
      <h2 className="mb-4 text-xl font-semibold">Project Settings</h2>

      <form onSubmit={handleUpdate} className="space-y-4">
        {updateProject.error && (
          <p className="text-sm text-red-600">{updateProject.error.message}</p>
        )}
        {updateProject.isSuccess && (
          <p className="text-sm text-green-600">Settings saved.</p>
        )}

        <div>
          <label className="mb-1 block text-sm font-medium">Project name</label>
          <input
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
            className="w-full rounded border px-3 py-2 text-sm"
          />
        </div>

        <div>
          <label className="mb-1 block text-sm font-medium">Slug</label>
          <input
            value={slug}
            onChange={(e) => setSlug(e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, ""))}
            required
            className="w-full rounded border px-3 py-2 text-sm font-mono"
          />
          <p className="mt-1 text-xs text-gray-400">Lowercase letters, numbers, and hyphens only.</p>
        </div>

        <button
          type="submit"
          disabled={updateProject.isPending}
          className="rounded bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
        >
          {updateProject.isPending ? "Saving..." : "Save Changes"}
        </button>
      </form>

      {/* Danger zone */}
      <div className="mt-10 rounded border border-red-200 bg-red-50 p-4">
        <h3 className="mb-2 text-sm font-semibold text-red-800">Danger Zone</h3>
        <p className="mb-3 text-sm text-red-700">
          Deleting this project removes all its data permanently: alerts, keywords, rules, connections, and posts.
        </p>
        <button
          onClick={handleDelete}
          disabled={deleteProject.isPending}
          className="rounded bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50"
        >
          {deleteProject.isPending ? "Deleting..." : "Delete Project"}
        </button>
        {deleteProject.error && (
          <p className="mt-2 text-sm text-red-600">{deleteProject.error.message}</p>
        )}
      </div>
    </div>
  );
}
