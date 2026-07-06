"use client";

import { ApprovalStatusBadge } from "@/components/approvals/ApprovalStatusBadge";
import { useTripApproval } from "@/hooks/useTripApproval";

// TripApprovalBadge shows a workspace trip's approval status inline (e.g. next to
// the trip title). It renders nothing for personal trips or while loading, and
// shares the approval query with the panel so it costs no extra request.
export function TripApprovalBadge({ tripId }: { tripId: string }) {
  const { data } = useTripApproval(tripId);
  if (!data || data.status === "not_required" || !data.workspaceId) {
    return null;
  }
  return <ApprovalStatusBadge status={data.status} />;
}
