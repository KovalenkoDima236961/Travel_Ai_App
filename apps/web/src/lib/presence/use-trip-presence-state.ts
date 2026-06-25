"use client";

import { useCallback } from "react";

import { updateTripPresenceState } from "@/lib/api/presence";
import type { PresenceState } from "@/types/presence";

export function useTripPresenceState(tripId: string, enabled: boolean) {
  return useCallback(
    async (state: PresenceState) => {
      if (!enabled || !tripId) {
        return false;
      }
      try {
        await updateTripPresenceState(tripId, state);
        return true;
      } catch {
        return false;
      }
    },
    [enabled, tripId]
  );
}
