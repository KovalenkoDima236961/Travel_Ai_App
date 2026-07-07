import Link from "next/link";
import { Card } from "@/shared/ui/card";
import { formatWorkspaceRole } from "@/components/workspaces/WorkspaceProvider";
import { formatDate } from "@/lib/utils";
import type { Workspace, WorkspaceRole } from "@/entities/workspace/model";

export function WorkspaceCard({ workspace }: { workspace: Workspace }) {
  return (
    <Link className="block h-full" href={`/workspaces/${workspace.id}`}>
      <Card className="flex h-full flex-col gap-5 transition hover:-translate-y-0.5 hover:border-primary-100 hover:shadow-lg">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <h2 className="truncate text-lg font-semibold text-slate-950">{workspace.name}</h2>
            <p className="mt-1 text-sm text-slate-500">
              Created {formatDate(workspace.createdAt)}
            </p>
          </div>
          <RoleBadge role={workspace.currentUserRole} />
        </div>
        {workspace.description ? (
          <p className="line-clamp-3 text-sm leading-6 text-slate-600">
            {workspace.description}
          </p>
        ) : (
          <p className="text-sm text-slate-500">No description</p>
        )}
        <div className="mt-auto text-sm font-medium text-slate-700">
          {workspace.memberCount} {workspace.memberCount === 1 ? "member" : "members"}
        </div>
      </Card>
    </Link>
  );
}

function RoleBadge({ role }: { role: WorkspaceRole }) {
  return (
    <span className="shrink-0 rounded-full bg-slate-100 px-2.5 py-1 text-xs font-semibold text-slate-700">
      {formatWorkspaceRole(role)}
    </span>
  );
}
