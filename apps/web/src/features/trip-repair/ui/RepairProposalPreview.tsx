"use client";

import { ItineraryRepairDiff } from "./ItineraryRepairDiff";
import { useTripRepairProposal } from "../model/useTripRepairProposal";
import type { RepairProposal } from "@/entities/trip-repair/model";
import type { Itinerary } from "@/entities/trip/model";

type RepairProposalPreviewProps = {
  tripId: string;
  proposal: RepairProposal;
  currentItinerary: Itinerary;
};

export function RepairProposalPreview({
  tripId,
  proposal,
  currentItinerary
}: RepairProposalPreviewProps) {
  const proposalQuery = useTripRepairProposal({
    tripId,
    proposalId: proposal.id
  });

  if (proposalQuery.isLoading) {
    return (
      <div className="rounded-md border border-slate-200 bg-slate-50 p-4 text-sm text-slate-500">
        Loading repair preview...
      </div>
    );
  }

  if (proposalQuery.isError || !proposalQuery.data) {
    return (
      <div className="rounded-md border border-red-200 bg-red-50 p-4 text-sm text-red-800">
        Could not load repair preview.
      </div>
    );
  }

  const detail = proposalQuery.data;
  const content = detail.proposal;

  return (
    <div className="space-y-4">
      {detail.issues.length > 0 ? (
        <div className="rounded-md border border-slate-200 bg-slate-50 p-3">
          <p className="text-sm font-semibold text-slate-950">Issues addressed</p>
          <ul className="mt-2 space-y-1 text-sm text-slate-600">
            {detail.issues.slice(0, 6).map((issue, index) => (
              <li key={`${issue.type}-${index}`}>
                {issue.affected?.dayNumber ? `Day ${issue.affected.dayNumber}: ` : ""}
                {issue.message}
              </li>
            ))}
          </ul>
        </div>
      ) : null}
      <ItineraryRepairDiff
        currentItinerary={currentItinerary}
        diff={content.diff}
        repairedItinerary={content.repairedItinerary}
      />
    </div>
  );
}
