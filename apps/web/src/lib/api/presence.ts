import { apiFetch } from "@/lib/api/client";
import type { PresenceState } from "@/types/presence";

export function updateTripPresenceState(tripId: string, state: PresenceState) {
  return apiFetch<{ success: boolean }>(`/trips/${tripId}/presence/state`, {
    method: "POST",
    body: JSON.stringify({ state })
  });
}
