"use client";

import { useQuery } from "@tanstack/react-query";
import { getTravelDay, travelDayKeys } from "@/lib/api/travel-day";
import { cacheTravelDaySnapshot, getCachedTravelDay } from "@/lib/offline/trip-cache";

export function useTravelDay({
  tripId,
  date,
  userId,
  enabled = true
}: {
  tripId: string;
  date: string;
  userId?: string | null;
  enabled?: boolean;
}) {
  return useQuery({
    queryKey: travelDayKeys.detail(tripId, date),
    enabled: enabled && Boolean(tripId && date),
    staleTime: 20_000,
    queryFn: async () => {
      try {
        const summary = await getTravelDay(tripId, date);
        if (userId) {
          await cacheTravelDaySnapshot({ tripId, userId, date, summary });
        }
        return { summary, offlineCopy: false, cachedAt: null as string | null };
      } catch (error) {
        if (userId) {
          const cached = await getCachedTravelDay(tripId, userId, date);
          if (cached) {
            return {
              summary: { ...cached.summary, offline: { ...cached.summary.offline, lastCachedAt: cached.cachedAt } },
              offlineCopy: true,
              cachedAt: cached.cachedAt
            };
          }
        }
        throw error;
      }
    }
  });
}
