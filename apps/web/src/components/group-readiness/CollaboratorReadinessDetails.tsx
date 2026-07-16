import { NudgeCollaboratorButton } from "./NudgeCollaboratorButton";
import { ReadinessIssueList } from "./ReadinessIssueList";
import { categoryLabel } from "./readiness-ui";
import type { CollaboratorReadiness } from "@/types/group-readiness";

type CollaboratorReadinessDetailsProps = {
  tripId: string;
  member: CollaboratorReadiness;
  canNudge: boolean;
};

export function CollaboratorReadinessDetails({
  tripId,
  member,
  canNudge
}: CollaboratorReadinessDetailsProps) {
  return (
    <div className="border-t border-sand-200 bg-sand-50 p-4">
      {member.completedItems.length > 0 ? (
        <div className="mb-4">
          <p className="text-[12px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">
            Completed
          </p>
          <div className="mt-2 flex flex-wrap gap-2">
            {member.completedItems.map((item) => (
              <span
                key={`${item.category}:${item.title}`}
                className="rounded-full border border-[#CFE3D3] bg-[#EFF7F1] px-3 py-1 text-[12px] font-medium text-[#2F5C3C]"
              >
                {categoryLabel(item.category)}: {item.title}
              </span>
            ))}
          </div>
        </div>
      ) : null}

      <ReadinessIssueList items={member.items} />

      <div className="mt-4 flex flex-wrap items-center gap-2">
        {member.nextAction ? (
          <a
            className="inline-flex h-9 items-center justify-center rounded-full bg-cocoa-900 px-4 text-[13px] font-semibold text-sand-100 transition hover:bg-cocoa-700"
            href={member.nextAction.href}
          >
            {member.nextAction.label}
          </a>
        ) : null}
        {canNudge ? <NudgeCollaboratorButton member={member} tripId={tripId} /> : null}
      </div>
    </div>
  );
}

