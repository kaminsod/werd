import { useParams, Link, useNavigate } from "react-router";
import { useMembers, useUpdateMemberRole, useRemoveMember } from "@/hooks/use-members";
import { useAuthStore } from "@/stores/auth";
import type { ProjectRole } from "@/types/api";

const ROLES: ProjectRole[] = ["owner", "admin", "member", "viewer"];

const ROLE_COLORS: Record<ProjectRole, string> = {
  owner: "bg-purple-100 text-purple-700",
  admin: "bg-blue-100 text-blue-700",
  member: "bg-green-100 text-green-700",
  viewer: "bg-gray-100 text-gray-500",
};

export default function MemberDetailPage() {
  const { id: projectId, userId } = useParams<{ id: string; userId: string }>();
  const navigate = useNavigate();
  const { data: members, isLoading, error } = useMembers(projectId!);
  const updateRole = useUpdateMemberRole(projectId!);
  const removeMember = useRemoveMember(projectId!);
  const currentUser = useAuthStore((s) => s.user);

  const member = members?.find((m) => m.user_id === userId);
  const currentMember = members?.find((m) => m.user_id === currentUser?.id);
  const canManage = currentMember?.role === "owner" || currentMember?.role === "admin";
  const isSelf = userId === currentUser?.id;

  function handleRoleChange(newRole: string) {
    if (member && confirm(`Change ${member.name}'s role to ${newRole}?`)) {
      updateRole.mutate({ userId: userId!, role: newRole });
    }
  }

  function handleRemove() {
    const msg = isSelf ? "Leave this project?" : `Remove ${member?.name} from the project?`;
    if (confirm(msg)) {
      removeMember.mutate(userId!, { onSuccess: () => navigate("../members") });
    }
  }

  if (isLoading) return <p className="text-gray-500">Loading member...</p>;
  if (error) return <p className="text-red-600">Error: {error.message}</p>;
  if (!member) return (
    <div>
      <Link to="../members" className="mb-4 inline-block text-sm text-blue-600 hover:underline">&larr; Back to Members</Link>
      <p className="text-gray-500">Member not found.</p>
    </div>
  );

  return (
    <div>
      <Link to="../members" className="mb-4 inline-block text-sm text-blue-600 hover:underline">&larr; Back to Members</Link>

      <div className="mb-4 flex items-center gap-3">
        <h2 className="text-xl font-semibold">{member.name}</h2>
        <span className={`rounded px-2 py-0.5 text-xs font-medium ${ROLE_COLORS[member.role]}`}>{member.role}</span>
        {isSelf && <span className="text-xs text-gray-400">(you)</span>}
      </div>

      <div className="space-y-4 rounded border bg-white p-4">
        <div className="grid grid-cols-2 gap-4 text-sm">
          <div>
            <span className="font-medium text-gray-500">Name</span>
            <p className="mt-1">{member.name}</p>
          </div>
          <div>
            <span className="font-medium text-gray-500">Email</span>
            <p className="mt-1">{member.email}</p>
          </div>
        </div>

        <div className="grid grid-cols-2 gap-4 text-sm">
          <div>
            <span className="font-medium text-gray-500">Role</span>
            <div className="mt-1 flex items-center gap-2">
              {canManage && member.role !== "owner" ? (
                <select
                  value={member.role}
                  onChange={(e) => handleRoleChange(e.target.value)}
                  className="rounded border px-2 py-1 text-sm"
                >
                  {ROLES.filter((r) => currentMember?.role === "owner" || r !== "owner").map((r) => (
                    <option key={r} value={r}>{r}</option>
                  ))}
                </select>
              ) : (
                <p>{member.role}</p>
              )}
            </div>
          </div>
          <div>
            <span className="font-medium text-gray-500">Joined</span>
            <p className="mt-1">{new Date(member.created_at).toLocaleString()}</p>
          </div>
        </div>

        {member.role !== "owner" && (isSelf || canManage) && (
          <div className="flex gap-2 border-t pt-4">
            <button onClick={handleRemove} className="rounded px-2 py-1 text-xs text-red-600 hover:bg-red-50">
              {isSelf ? "Leave Project" : "Remove Member"}
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
