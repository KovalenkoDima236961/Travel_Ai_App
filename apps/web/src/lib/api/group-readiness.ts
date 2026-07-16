import { apiFetch } from "@/shared/api/client";
import type { GroupReadiness, NudgeRequest, NudgeResponse } from "@/types/group-readiness";

export const groupReadinessKeys = {
  all: ["group-readiness"] as const,
  detail: (tripId: string) => [...groupReadinessKeys.all, tripId] as const
};

export function getGroupReadiness(tripId: string) {
  return apiFetch<GroupReadiness>(`/trips/${tripId}/group-readiness?includeDetails=true`);
}

export function sendGroupReadinessNudge(tripId: string, input: NudgeRequest) {
  return apiFetch<NudgeResponse>(`/trips/${tripId}/group-readiness/nudge`, {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export function nudgeMissingAvailability(tripId: string, input: Omit<NudgeRequest, "categories">) {
  return apiFetch<NudgeResponse>(`/trips/${tripId}/group-readiness/nudge-missing-availability`, {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export function nudgeAssignedTasks(tripId: string, input: Omit<NudgeRequest, "categories">) {
  return apiFetch<NudgeResponse>(`/trips/${tripId}/group-readiness/nudge-assigned-tasks`, {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export function nudgePendingVotes(tripId: string, input: Omit<NudgeRequest, "categories">) {
  return apiFetch<NudgeResponse>(`/trips/${tripId}/group-readiness/nudge-pending-votes`, {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export function nudgePendingSettlements(tripId: string, input: Omit<NudgeRequest, "categories">) {
  return apiFetch<NudgeResponse>(`/trips/${tripId}/group-readiness/nudge-pending-settlements`, {
    method: "POST",
    body: JSON.stringify(input)
  });
}

