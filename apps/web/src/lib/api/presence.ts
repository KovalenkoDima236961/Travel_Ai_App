import { apiFetch } from "@/shared/api/client";
import type { PresenceState } from "@/entities/presence/model";

export function updateTripPresenceState(tripId: string, state: PresenceState) {
  return apiFetch<{ success: boolean }>(`/trips/${tripId}/presence/state`, {
    method: "POST",
    body: JSON.stringify({ state })
  });
}
