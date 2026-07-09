import Link from "next/link";
import { formatWorkspaceRole } from "@/components/workspaces/WorkspaceProvider";
import { formatDate } from "@/lib/utils";
import type { Workspace } from "@/entities/workspace/model";
import { UsersIcon } from "./icons";

// The mock cycles the avatar tile through three warm brand colors. Real
// workspaces carry no color, so pick one deterministically from the id (stable
// across reorders, unlike a list index). Green already appears in AccountMenu.
const AVATAR_BACKGROUNDS = ["#C05B3B", "#3E6B5A", "#96682A"];

function avatarBackground(id: string) {
  let hash = 0;
  for (let index = 0; index < id.length; index += 1) {
    hash = (hash * 31 + id.charCodeAt(index)) | 0;
  }
  return AVATAR_BACKGROUNDS[Math.abs(hash) % AVATAR_BACKGROUNDS.length];
}

function initials(name: string) {
  const words = name.trim().split(/\s+/).filter(Boolean);
  if (words.length === 0) {
    return "?";
  }
  if (words.length === 1) {
    return words[0].slice(0, 2).toUpperCase();
  }
  return (words[0][0] + words[1][0]).toUpperCase();
}

export function WorkspaceCard({ workspace }: { workspace: Workspace }) {
  return (
    <Link
      href={`/workspaces/${workspace.id}`}
      className="flex flex-col rounded-[18px] border border-sand-300 bg-white p-6 shadow-[0_1px_2px_rgba(34,26,20,0.04)] transition duration-200 hover:-translate-y-[3px] hover:shadow-[0_18px_40px_rgba(34,26,20,0.1)]"
    >
      <div className="flex items-center gap-3.5">
        <span
          className="flex h-12 w-12 shrink-0 items-center justify-center rounded-[14px] font-newsreader text-[20px] font-semibold text-sand-100"
          style={{ background: avatarBackground(workspace.id) }}
        >
          {initials(workspace.name)}
        </span>
        <div className="min-w-0">
          <h2 className="truncate font-newsreader text-[21px] font-semibold text-cocoa-900">
            {workspace.name}
          </h2>
          <p className="mt-[3px] text-[12.5px] font-semibold text-clay-deep">
            {formatWorkspaceRole(workspace.currentUserRole)}
          </p>
        </div>
      </div>

      <p className="mt-4 flex-1 text-[13.5px] leading-[1.55] text-cocoa-500">
        {workspace.description || "No description yet."}
      </p>

      <div className="mt-4 flex items-center justify-between gap-3 border-t border-sand-200 pt-3.5">
        <span className="inline-flex items-center gap-1.5 text-[13px] text-cocoa-400">
          <UsersIcon className="h-[15px] w-[15px] text-sand-600" />
          {workspace.memberCount} {workspace.memberCount === 1 ? "member" : "members"}
        </span>
        <span className="text-[13px] text-[#A08D78]">
          Created {formatDate(workspace.createdAt)}
        </span>
      </div>
    </Link>
  );
}
