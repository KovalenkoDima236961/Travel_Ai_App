"use client";

import { useMemo, useState } from "react";
import { NudgeDialog } from "./NudgeDialog";
import type { CollaboratorReadiness, ReadinessCategory } from "@/types/group-readiness";

type NudgeCollaboratorButtonProps = {
  tripId: string;
  member: CollaboratorReadiness;
  disabled?: boolean;
};

export function NudgeCollaboratorButton({
  tripId,
  member,
  disabled = false
}: NudgeCollaboratorButtonProps) {
  const [open, setOpen] = useState(false);
  const categories = useMemo(
    () =>
      Array.from(
        new Set(
          member.items
            .map((item) => item.category)
            .filter((category): category is ReadinessCategory =>
              ["availability", "polls", "checklist", "reminders", "settlements"].includes(category)
            )
        )
      ),
    [member.items]
  );

  if (member.isCurrentUser || categories.length === 0) {
    return null;
  }

  return (
    <>
      <button
        type="button"
        className="inline-flex h-9 items-center justify-center rounded-full border border-sand-400 bg-white px-4 text-[13px] font-semibold text-cocoa-700 transition hover:border-sand-600 hover:text-cocoa-900 disabled:cursor-not-allowed disabled:opacity-60"
        disabled={disabled}
        onClick={() => setOpen(true)}
      >
        Remind
      </button>
      <NudgeDialog
        initialCategories={categories}
        onClose={() => setOpen(false)}
        open={open}
        targetLabel={member.displayName}
        targetUserIds={[member.userId]}
        tripId={tripId}
      />
    </>
  );
}

