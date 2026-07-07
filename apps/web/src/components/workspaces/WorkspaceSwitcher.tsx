"use client";

import Link from "next/link";
import { Select } from "@/shared/ui/select";
import { buttonStyles } from "@/shared/ui/button";
import { useWorkspaces } from "@/components/workspaces/WorkspaceProvider";

export function WorkspaceSwitcher() {
  const { workspaces, selectionValue, setSelectionValue, isLoading } = useWorkspaces();

  return (
    <div className="flex items-center gap-2">
      <Select
        aria-label="Workspace"
        className="h-9 w-44 text-sm"
        disabled={isLoading}
        value={selectionValue}
        onChange={(event) => setSelectionValue(event.target.value)}
      >
        <option value="all">All trips</option>
        <option value="personal">Personal</option>
        {workspaces.map((workspace) => (
          <option key={workspace.id} value={workspace.id}>
            {workspace.name}
          </option>
        ))}
      </Select>
      <Link className={buttonStyles({ variant: "ghost", size: "sm" })} href="/workspaces">
        Workspaces
      </Link>
    </div>
  );
}
