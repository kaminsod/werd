import { useState, type FormEvent } from "react";
import { useParams, Link } from "react-router";
import { useMembers, useAddMember, useUpdateMemberRole, useRemoveMember } from "@/hooks/use-members";
import { useAuthStore } from "@/stores/auth";
import InfoIcon from "@/components/info-icon";
import { navMembers as membersHelp, memberUserId as userIdHelp } from "@/lib/help-content";
import type { Member, ProjectRole } from "@/types/api";

const ROLES: ProjectRole[] = ["owner", "admin", "member", "viewer"];
const ASSIGNABLE_ROLES: ProjectRole[] = ["admin", "member", "viewer"];

const ROLE_COLORS: Record<ProjectRole, string> = {
  owner: "bg-purple-100 text-purple-700",
  admin: "bg-blue-100 text-blue-700",
  member: "bg-green-100 text-green-700",
  viewer: "bg-gray-100 text-gray-500",
};

export default function MembersPage() {
  const { id: projectId } = useParams<{ id: string }>();
  const { data: members, isLoading, error } = useMembers(projectId!);
  const addMember = useAddMember(projectId!);
  const updateRole = useUpdateMemberRole(projectId!);
  const removeMember = useRemoveMember(projectId!);
  const currentUser = useAuthStore((s) => s.user);

  const [showForm, setShowForm] = useState(false);
  const [userId, setUserId] = useState("");
  const [role, setRole] = useState<ProjectRole>("member");

  const currentMember = members?.find((m) => m.user_id === currentUser?.id);
  const canManage = currentMember?.role === "owner" || currentMember?.role === "admin";

  function handleAdd(e: FormEvent) {
    e.preventDefault();
    addMember.mutate(
      { user_id: userId, role },
      {
        onSuccess: () => {
          setUserId("");
          setRole("member");
          setShowForm(false);
        },
      },
    );
  }

  function handleRoleChange(member: Member, newRole: string) {
    if (confirm(`Change ${member.name}'s role to ${newRole}?`)) {
      updateRole.mutate({ userId: member.user_id, role: newRole });
    }
  }

  function handleRemove(member: Member) {
    const isSelf = member.user_id === currentUser?.id;
    const msg = isSelf ? "Leave this project?" : `Remove ${member.name} from the project?`;
    if (confirm(msg)) {
      removeMember.mutate(member.user_id);
    }
  }

  if (isLoading) return <p className="text-gray-500">Loading members...</p>;
  if (error) return <p className="text-red-600">Error: {error.message}</p>;

  return (
    <div>
      <div className="mb-4 flex items-center justify-between">
        <h2 className="text-xl font-semibold">
          Members
          <InfoIcon tooltip={membersHelp.tooltip}>{membersHelp.modal}</InfoIcon>
        </h2>
        {canManage && (
          <button
            onClick={() => setShowForm(!showForm)}
            className="rounded bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
          >
            {showForm ? "Cancel" : "Add Member"}
          </button>
        )}
      </div>

      {showForm && (
        <form onSubmit={handleAdd} className="mb-6 space-y-3 rounded border bg-gray-50 p-4">
          <h3 className="text-sm font-medium">Add Member</h3>

          {addMember.error && (
            <p className="text-sm text-red-600">{addMember.error.message}</p>
          )}

          <div className="flex gap-3">
            <div className="flex-1">
              <label className="mb-1 block text-xs font-medium text-gray-600">
                User ID
                <InfoIcon tooltip={userIdHelp.tooltip}>{userIdHelp.modal}</InfoIcon>
              </label>
              <input
                value={userId}
                onChange={(e) => setUserId(e.target.value)}
                required
                placeholder="UUID of the user to add"
                className="w-full rounded border px-3 py-2 text-sm font-mono"
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-600">Role</label>
              <select
                value={role}
                onChange={(e) => setRole(e.target.value as ProjectRole)}
                className="rounded border px-3 py-2 text-sm"
              >
                {ASSIGNABLE_ROLES.map((r) => (
                  <option key={r} value={r}>{r}</option>
                ))}
              </select>
            </div>
          </div>

          <button
            type="submit"
            disabled={addMember.isPending}
            className="rounded bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
          >
            {addMember.isPending ? "Adding..." : "Add"}
          </button>
        </form>
      )}

      {members!.length === 0 ? (
        <p className="text-gray-500">No members.</p>
      ) : (
        <div className="space-y-2">
          {members!.map((member) => (
            <div key={member.user_id} className="flex items-center justify-between rounded border bg-white p-3">
              <div className="flex items-center gap-3">
                <span className={`rounded px-2 py-0.5 text-xs font-medium ${ROLE_COLORS[member.role]}`}>
                  {member.role}
                </span>
                <div>
                  <Link to={member.user_id} className="text-sm font-medium text-blue-600 hover:underline">{member.name}</Link>
                  <span className="ml-2 text-sm text-gray-500">{member.email}</span>
                </div>
                {member.user_id === currentUser?.id && (
                  <span className="text-xs text-gray-400">(you)</span>
                )}
              </div>
              <div className="flex items-center gap-2">
                {canManage && member.role !== "owner" && (
                  <select
                    value={member.role}
                    onChange={(e) => handleRoleChange(member, e.target.value)}
                    className="rounded border px-2 py-1 text-xs"
                  >
                    {ROLES.filter((r) => currentMember?.role === "owner" || r !== "owner").map((r) => (
                      <option key={r} value={r}>{r}</option>
                    ))}
                  </select>
                )}
                {member.role !== "owner" && (member.user_id === currentUser?.id || canManage) && (
                  <button
                    onClick={() => handleRemove(member)}
                    className="rounded px-2 py-1 text-xs text-red-600 hover:bg-red-50"
                  >
                    {member.user_id === currentUser?.id ? "Leave" : "Remove"}
                  </button>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
