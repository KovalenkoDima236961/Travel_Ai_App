import { describe, expect, it } from "vitest";
import {
  getEffectiveReviewStatus,
  getPlaceMatchReviewItems,
  getPlaceMatchReviewSummary,
  removeItemPlaceFromReview,
  replaceItemPlaceFromReview,
  updateItemPlaceReviewStatus
} from "@/entities/itinerary/model/place-enrichment-review-utils";
import type { Place } from "@/entities/place/model";
import type { Itinerary, ItineraryItem } from "@/entities/trip/model";

function place(name = "Colosseum", provider = "mock"): Place {
  return {
    provider,
    providerPlaceId: `${provider}-${name.toLowerCase().replace(/\s+/g, "-")}`,
    name,
    address: `${name} address`,
    latitude: 41.8902,
    longitude: 12.4922,
    category: "landmark"
  };
}

function item(overrides: Partial<ItineraryItem> = {}): ItineraryItem {
  return {
    time: "09:00",
    type: "activity",
    name: "Colosseum",
    note: "Original note",
    estimatedCost: { amount: 12, currency: "EUR", category: "ticket" },
    ...overrides
  };
}

function itinerary(items: ItineraryItem[]): Itinerary {
  return {
    destination: "Rome",
    days: [
      {
        day: 1,
        title: "Arrival",
        items
      }
    ]
  };
}

describe("getEffectiveReviewStatus", () => {
  it("defaults missing matched review status to pending", () => {
    expect(getEffectiveReviewStatus({ status: "matched" })).toBe("pending");
  });

  it("keeps an accepted review status", () => {
    expect(getEffectiveReviewStatus({ status: "matched", reviewStatus: "accepted" })).toBe(
      "accepted"
    );
  });

  it("defaults missing metadata to pending", () => {
    expect(getEffectiveReviewStatus(null)).toBe("pending");
  });
});

describe("getPlaceMatchReviewItems", () => {
  it("includes matched and no-match items but excludes items without enrichment", () => {
    const plan = itinerary([
      item({
        place: place(),
        placeEnrichment: {
          status: "matched",
          confidence: 0.91,
          query: "Colosseum",
          provider: "mock"
        }
      }),
      item({
        time: "12:30",
        type: "food",
        name: "Local trattoria",
        placeEnrichment: {
          status: "no_match",
          confidence: 0.2,
          query: "Local trattoria"
        }
      }),
      item({ time: "15:00", name: "Free walk" })
    ]);

    const items = getPlaceMatchReviewItems(plan);

    expect(items).toHaveLength(2);
    expect(items[0]).toMatchObject({
      id: "day-1-item-0",
      dayIndex: 0,
      dayNumber: 1,
      itemIndex: 0,
      confidence: 0.91,
      provider: "mock",
      query: "Colosseum",
      reviewStatus: "pending"
    });
    expect(items[1]).toMatchObject({
      id: "day-1-item-1",
      status: "no_match",
      itemName: "Local trattoria"
    });
  });

  it("includes an attached place when enrichment metadata exists", () => {
    const plan = itinerary([
      item({
        place: place("Pantheon"),
        placeEnrichment: {
          status: "failed",
          reviewStatus: "pending",
          reason: "search_failed"
        }
      })
    ]);

    expect(getPlaceMatchReviewItems(plan)[0]).toMatchObject({
      status: "failed",
      placeName: "Pantheon",
      placeAddress: "Pantheon address"
    });
  });
});

describe("getPlaceMatchReviewSummary", () => {
  it("counts status and review states", () => {
    const plan = itinerary([
      item({
        place: place(),
        placeEnrichment: { status: "matched", reviewStatus: "accepted" }
      }),
      item({
        place: place("Pantheon"),
        placeEnrichment: { status: "matched", reviewStatus: "changed" }
      }),
      item({
        placeEnrichment: { status: "no_match" }
      }),
      item({
        place: place("Forum"),
        placeEnrichment: { status: "matched", reviewStatus: "removed" }
      })
    ]);

    expect(getPlaceMatchReviewSummary(plan)).toEqual({
      total: 4,
      matched: 3,
      noMatch: 1,
      pending: 1,
      accepted: 1,
      changed: 1,
      removed: 1
    });
  });
});

describe("updateItemPlaceReviewStatus", () => {
  it("sets accepted without mutating the original or removing place metadata", () => {
    const original = itinerary([
      item({
        place: place(),
        placeEnrichment: { status: "matched", confidence: 0.9 }
      })
    ]);
    const snapshot = JSON.parse(JSON.stringify(original));

    const updated = updateItemPlaceReviewStatus(original, 0, 0, "accepted");

    expect(original).toEqual(snapshot);
    expect(updated.days[0].items[0].place?.name).toBe("Colosseum");
    expect(updated.days[0].items[0].placeEnrichment?.reviewStatus).toBe("accepted");
  });
});

describe("replaceItemPlaceFromReview", () => {
  it("attaches a selected place and marks the review as changed", () => {
    const original = itinerary([
      item({
        placeEnrichment: {
          status: "no_match",
          confidence: 0.1,
          query: "Local trattoria"
        }
      })
    ]);
    const selected = place("Trattoria Roma", "foursquare");

    const updated = replaceItemPlaceFromReview(original, 0, 0, selected);

    expect(updated.days[0].items[0].place).toEqual(selected);
    expect(updated.days[0].items[0].placeEnrichment).toMatchObject({
      status: "matched",
      reviewStatus: "changed",
      confidence: 0.1,
      provider: "foursquare",
      reason: "user_changed_match"
    });
  });
});

describe("removeItemPlaceFromReview", () => {
  it("removes the place and preserves useful enrichment metadata", () => {
    const original = itinerary([
      item({
        place: place(),
        placeEnrichment: {
          status: "matched",
          confidence: 0.9,
          query: "Colosseum",
          provider: "mock"
        }
      })
    ]);

    const updated = removeItemPlaceFromReview(original, 0, 0);

    expect(updated.days[0].items[0].place).toBeNull();
    expect(updated.days[0].items[0].placeEnrichment).toMatchObject({
      status: "matched",
      reviewStatus: "removed",
      confidence: 0.9,
      query: "Colosseum",
      provider: "mock",
      reason: "user_removed_match"
    });
  });
});
