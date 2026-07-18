"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { updateTravelItemStatus } from "@/lib/api/travel-day";
import { activityKeys } from "@/lib/api/activity";
import { tripKeys } from "@/lib/api/trips";
import { queryKeys } from "@/lib/query-keys";
import { getCachedTrip, putCachedTravelDay, updateCachedTripItinerary } from "@/lib/offline/trip-cache";
import { enqueueItineraryUpdate } from "@/lib/offline/sync-queue";
import type { Itinerary, TravelStatus } from "@/entities/trip/model";
import type { TravelDaySummary } from "@/types/travel-day";

type Input = {
  dayNumber: number;
  itemIndex: number;
  status: TravelStatus;
  note?: string;
};

export function useUpdateTravelItemStatus({
  tripId,
  date,
  userId,
  offline = false
}: {
  tripId: string;
  date: string;
  userId?: string | null;
  offline?: boolean;
}) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: Input) => {
      if (offline) {
        if (!userId) throw new Error("Open this trip online once before changing travel status offline.");
        return updateOfflineTravelStatus({ ...input, tripId, date, userId, queryClient });
      }
      return updateTravelItemStatus(tripId, input.dayNumber, input.itemIndex, {
        status: input.status,
        note: input.note,
        expectedItineraryRevision: currentRevision(queryClient, tripId, date)
      });
    },
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queryKeys.trip.travelDay(tripId, date) }),
        queryClient.invalidateQueries({ queryKey: tripKeys.detail(tripId) }),
        queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) }),
        queryClient.invalidateQueries({ queryKey: queryKeys.trip.commandCenter(tripId) })
      ]);
    }
  });
}

function currentRevision(queryClient: ReturnType<typeof useQueryClient>, tripId: string, date: string) {
  const data = queryClient.getQueryData<{ summary?: TravelDaySummary }>(
    queryKeys.trip.travelDay(tripId, date)
  );
  return data?.summary?.itineraryRevision ?? 0;
}

async function updateOfflineTravelStatus(input: Input & {
  tripId: string;
  date: string;
  userId: string;
  queryClient: ReturnType<typeof useQueryClient>;
}) {
  const record = await getCachedTrip(input.tripId, input.userId);
  if (!record?.trip.itinerary) throw new Error("No offline itinerary is available for this trip.");
  const itinerary = structuredClone(record.trip.itinerary) as Itinerary;
  const day = itinerary.days.find((candidate, index) => (candidate.day || index + 1) === input.dayNumber);
  const item = day?.items[input.itemIndex];
  if (!item) throw new Error("This itinerary item is no longer available offline.");
  item.travelStatus = { status: input.status, updatedAt: new Date().toISOString(), note: input.note };
  await updateCachedTripItinerary({ tripId: input.tripId, userId: input.userId, itinerary });
  await enqueueItineraryUpdate({
    tripId: input.tripId,
    userId: input.userId,
    baseRevision: record.itineraryRevision,
    baseItinerary: record.trip.itinerary,
    draftItinerary: itinerary
  });
  const cached = input.queryClient.getQueryData<{ summary: TravelDaySummary }>(
    queryKeys.trip.travelDay(input.tripId, input.date)
  );
  if (cached?.summary) {
    const summary = structuredClone(cached.summary) as TravelDaySummary;
    const timelineItem = summary.timeline.find(
      (candidate) => candidate.dayNumber === input.dayNumber && candidate.itemIndex === input.itemIndex
    );
    if (timelineItem) timelineItem.travelStatus = item.travelStatus;
    await putCachedTravelDay({ tripId: input.tripId, userId: input.userId, date: input.date, summary });
    input.queryClient.setQueryData(queryKeys.trip.travelDay(input.tripId, input.date), {
      summary,
      offlineCopy: true,
      cachedAt: new Date().toISOString()
    });
  }
  return { status: input.status, itineraryRevision: record.itineraryRevision };
}
