"use client";

import { useState } from "react";
import { CollaboratorReadinessDetails } from "./CollaboratorReadinessDetails";
import { ReadinessScoreBadge } from "./ReadinessScoreBadge";
import { categoryLabel } from "./readiness-ui";
import type { CollaboratorReadiness } from "@/types/group-readiness";

type CollaboratorReadinessRowProps = {
  tripId: string;
  member: CollaboratorReadiness;
  canNudge: boolean;
};

export function CollaboratorReadinessRow({
  tripId,
  member,
  canNudge
}: CollaboratorReadinessRowProps) {
  const [expanded, setExpanded] = useState(member.isCurrentUser && member.items.length > 0);
  const issueCategories = Array.from(new Set(member.items.map((item) => item.category)));
  const initials = initialsFor(member.displayName);

  return (
    <div className="overflow-hidden rounded-[16px] border border-sand-300 bg-white">
      <button
        type="button"
        aria-expanded={expanded}
        className="flex w-full items-center gap-4 p-4 text-left transition hover:bg-sand-50 focus:outline-none focus:ring-2 focus:ring-clay"
        onClick={() => setExpanded((value) => !value)}
      >
        <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-cocoa-900 text-[13px] font-semibold text-sand-100">
          {initials}
        </span>
        <span className="min-w-0 flex-1">
          <span className="flex flex-wrap items-center gap-2">
            <span className="truncate text-[15px] font-semibold text-cocoa-900">
              {member.displayName}
            </span>
            {member.isCurrentUser ? (
              <span className="rounded-full bg-sand-100 px-2 py-0.5 text-[11px] font-semibold text-cocoa-500">
                You
              </span>
            ) : null}
          </span>
          <span className="mt-1 block text-[12px] capitalize text-cocoa-400">
            {member.role.replaceAll("_", " ")}
          </span>
          {issueCategories.length > 0 ? (
            <span className="mt-2 flex flex-wrap gap-1.5">
              {issueCategories.slice(0, 5).map((category) => (
                <span
                  key={category}
                  className="rounded-full border border-sand-300 bg-sand-50 px-2 py-0.5 text-[11px] font-medium text-cocoa-500"
                >
                  {categoryLabel(category)}
                </span>
              ))}
            </span>
          ) : null}
        </span>
        <ReadinessScoreBadge level={member.level} score={member.score} />
      </button>
      {expanded ? (
        <CollaboratorReadinessDetails canNudge={canNudge} member={member} tripId={tripId} />
      ) : null}
    </div>
  );
}

function initialsFor(name: string) {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) {
    return "?";
  }
  if (parts.length === 1) {
    return parts[0].slice(0, 2).toUpperCase();
  }
  return `${parts[0][0]}${parts[1][0]}`.toUpperCase();
}

