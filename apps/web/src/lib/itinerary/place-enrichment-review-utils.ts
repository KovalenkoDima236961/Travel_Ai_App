import type { Place } from "@/types/place";
import type {
  Itinerary,
  ItineraryItem,
  PlaceEnrichmentMeta,
  PlaceEnrichmentReviewStatus
} from "@/types/trip";

export type PlaceMatchReviewItem = {
  id: string;
  dayIndex: number;
  dayNumber: number;
  itemIndex: number;
  time: string;
  itemName: string;
  itemType: string;
  placeName?: string | null;
  placeAddress?: string | null;
  confidence?: number | null;
  provider?: string | null;
  status: "matched" | "no_match" | "skipped" | "failed";
  reviewStatus: PlaceEnrichmentReviewStatus;
  query?: string | null;
};

export type PlaceMatchReviewSummary = {
  total: number;
  matched: number;
  noMatch: number;
  pending: number;
  accepted: number;
  changed: number;
  removed: number;
};

const reviewStatuses: PlaceEnrichmentReviewStatus[] = [
  "pending",
  "accepted",
  "changed",
  "removed"
];

export function getEffectiveReviewStatus(
  meta?: PlaceEnrichmentMeta | null
): PlaceEnrichmentReviewStatus {
  return meta?.reviewStatus && reviewStatuses.includes(meta.reviewStatus)
    ? meta.reviewStatus
    : "pending";
}

export function getPlaceMatchReviewItems(itinerary: Itinerary): PlaceMatchReviewItem[] {
  return (itinerary.days ?? []).flatMap((day, dayIndex) => {
    const dayNumber = day.day || dayIndex + 1;

    return (day.items ?? []).flatMap((item, itemIndex) => {
      const meta = item.placeEnrichment;
      if (!shouldIncludeReviewItem(item)) {
        return [];
      }

      return [
        {
          id: `day-${dayNumber}-item-${itemIndex}`,
          dayIndex,
          dayNumber,
          itemIndex,
          time: item.time,
          itemName: item.name,
          itemType: item.type,
          placeName: item.place?.name ?? null,
          placeAddress: item.place?.address ?? null,
          confidence: meta?.confidence ?? null,
          provider: item.place?.provider ?? meta?.provider ?? null,
          status: meta?.status ?? "matched",
          reviewStatus: getEffectiveReviewStatus(meta),
          query: meta?.query ?? null
        }
      ];
    });
  });
}

export function getPlaceMatchReviewSummary(itinerary: Itinerary): PlaceMatchReviewSummary {
  const items = getPlaceMatchReviewItems(itinerary);

  return items.reduce<PlaceMatchReviewSummary>(
    (summary, item) => ({
      total: summary.total + 1,
      matched: summary.matched + (item.status === "matched" ? 1 : 0),
      noMatch: summary.noMatch + (item.status === "no_match" ? 1 : 0),
      pending: summary.pending + (item.reviewStatus === "pending" ? 1 : 0),
      accepted: summary.accepted + (item.reviewStatus === "accepted" ? 1 : 0),
      changed: summary.changed + (item.reviewStatus === "changed" ? 1 : 0),
      removed: summary.removed + (item.reviewStatus === "removed" ? 1 : 0)
    }),
    {
      total: 0,
      matched: 0,
      noMatch: 0,
      pending: 0,
      accepted: 0,
      changed: 0,
      removed: 0
    }
  );
}

export function updateItemPlaceReviewStatus(
  itinerary: Itinerary,
  dayIndex: number,
  itemIndex: number,
  reviewStatus: PlaceEnrichmentReviewStatus
): Itinerary {
  return updateItineraryItem(itinerary, dayIndex, itemIndex, (item) => {
    const meta = ensurePlaceEnrichmentMeta(item, reviewStatus);
    return {
      ...item,
      placeEnrichment: {
        ...meta,
        reviewStatus
      }
    };
  });
}

export function replaceItemPlaceFromReview(
  itinerary: Itinerary,
  dayIndex: number,
  itemIndex: number,
  place: Place
): Itinerary {
  return updateItineraryItem(itinerary, dayIndex, itemIndex, (item) => {
    const meta = item.placeEnrichment ?? {
      status: "matched" as const,
      confidence: null,
      query: item.name,
      provider: place.provider
    };

    return {
      ...item,
      place,
      placeEnrichment: {
        ...meta,
        status: "matched",
        reviewStatus: "changed",
        confidence: meta.confidence ?? null,
        provider: place.provider,
        reason: "user_changed_match"
      }
    };
  });
}

export function removeItemPlaceFromReview(
  itinerary: Itinerary,
  dayIndex: number,
  itemIndex: number
): Itinerary {
  return updateItineraryItem(itinerary, dayIndex, itemIndex, (item) => {
    const meta = item.placeEnrichment ?? {
      status: "matched" as const,
      confidence: null,
      query: item.name,
      provider: item.place?.provider ?? null
    };

    return {
      ...item,
      place: null,
      placeEnrichment: {
        ...meta,
        reviewStatus: "removed",
        reason: "user_removed_match"
      }
    };
  });
}

function shouldIncludeReviewItem(item: ItineraryItem) {
  const status = item.placeEnrichment?.status;
  return status === "matched" || status === "no_match" || Boolean(item.place && item.placeEnrichment);
}

function updateItineraryItem(
  itinerary: Itinerary,
  dayIndex: number,
  itemIndex: number,
  updateItem: (item: ItineraryItem) => ItineraryItem
): Itinerary {
  const day = itinerary.days?.[dayIndex];
  const item = day?.items?.[itemIndex];

  if (!day || !item) {
    return itinerary;
  }

  return {
    ...itinerary,
    days: itinerary.days.map((currentDay, currentDayIndex) => {
      if (currentDayIndex !== dayIndex) {
        return currentDay;
      }

      return {
        ...currentDay,
        items: currentDay.items.map((currentItem, currentItemIndex) =>
          currentItemIndex === itemIndex ? updateItem(currentItem) : currentItem
        )
      };
    })
  };
}

function ensurePlaceEnrichmentMeta(
  item: ItineraryItem,
  reviewStatus: PlaceEnrichmentReviewStatus
): PlaceEnrichmentMeta {
  if (item.placeEnrichment) {
    return item.placeEnrichment;
  }

  return {
    status: item.place ? "matched" : "no_match",
    reviewStatus,
    provider: item.place?.provider ?? null,
    query: item.name,
    reason: reviewStatus === "accepted" ? "user_accepted_match" : undefined
  };
}
