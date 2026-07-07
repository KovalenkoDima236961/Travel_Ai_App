import { Button } from "@/shared/ui/button";
import { Select } from "@/shared/ui/select";
import {
  canManageWorkspace,
  formatWorkspaceRole
} from "@/components/workspaces/WorkspaceProvider";
import { inviteRoles } from "../model/workspaceSettingsPageModel";
import type { WorkspaceMember, WorkspaceRole } from "@/entities/workspace/model";

type MemberRowProps = {
  actorRole: WorkspaceRole;
  currentUserId?: string;
  disabled: boolean;
  member: WorkspaceMember;
  onRemove: (member: WorkspaceMember) => void;
  onRoleChange: (role: Exclude<WorkspaceRole, "owner">) => void;
};

export function MemberRow({
  actorRole,
  currentUserId,
  disabled,
  member,
  onRemove,
  onRoleChange
}: MemberRowProps) {
  const canManage = canManageWorkspace(actorRole);
  const isSelf = currentUserId === member.userId;
  const canChangeRole =
    canManage &&
    member.role !== "owner" &&
    !(actorRole === "admin" && member.role === "admin");
  const canRemove =
    (isSelf || canManage) &&
    !(member.role === "owner" && !isSelf) &&
    !(actorRole === "admin" && (member.role === "admin" || member.role === "owner"));
  const roleOptions = inviteRoles.filter((role) => actorRole === "owner" || role !== "admin");

  return (
    <div className="flex flex-col gap-4 p-4 sm:flex-row sm:items-center sm:justify-between">
      <div className="min-w-0">
        <p className="truncate text-sm font-semibold text-slate-950">
          {member.displayName || member.email || member.userId}
        </p>
        <p className="mt-1 text-xs text-slate-500">
          {member.email ? `${member.email} · ` : ""}
          {member.status}
        </p>
      </div>
      <div className="flex flex-wrap items-center gap-2">
        {canChangeRole ? (
          <Select
            className="h-9 w-32"
            disabled={disabled}
            value={member.role}
            onChange={(event) =>
              onRoleChange(event.target.value as Exclude<WorkspaceRole, "owner">)
            }
          >
            {roleOptions.map((role) => (
              <option key={role} value={role}>
                {formatWorkspaceRole(role)}
              </option>
            ))}
          </Select>
        ) : (
          <span className="rounded-full bg-slate-100 px-2.5 py-1 text-xs font-semibold text-slate-700">
            {formatWorkspaceRole(member.role)}
          </span>
        )}
        {canRemove ? (
          <Button disabled={disabled} size="sm" variant="secondary" onClick={() => onRemove(member)}>
            {isSelf ? "Leave" : "Remove"}
          </Button>
        ) : null}
      </div>
    </div>
  );
}
