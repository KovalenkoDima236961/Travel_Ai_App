import { describe, expect, it } from "vitest";

import { diffItineraries } from "@/lib/itinerary/diff-merge/diff";
import { applyConflictResolutions, mergeItineraries } from "@/lib/itinerary/diff-merge/merge";
import type { Itinerary, ItineraryItem } from "@/types/trip";

const walk = item("09:00", "activity", "Morning walk");
const lunch = item("12:00", "food", "Lunch");
const museum = item("15:00", "place", "Museum");
const dinner = item("19:00", "food", "Dinner");

describe("itinerary diff", () => {
  it("returns no changes for equal itineraries", () => {
    expect(diffItineraries(baseItinerary(), baseItinerary(), "local")).toEqual([]);
  });

  it("detects item modifications, additions, removals, days, and reorders", () => {
    expect(
      diffItineraries(
        baseItinerary(),
        withItem(1, 1, { ...lunch, note: "Vegetarian menu" }),
        "local"
      ).map((change) => change.type)
    ).toEqual(["item_modified"]);

    expect(
      diffItineraries(
        baseItinerary(),
        withDayItems(2, [museum, item("17:00", "activity", "Viewpoint"), dinner]),
        "local"
      ).map((change) => change.type)
    ).toContain("item_added");

    expect(
      diffItineraries(baseItinerary(), withDayItems(2, [museum]), "local").map(
        (change) => change.type
      )
    ).toContain("item_removed");

    expect(
      diffItineraries(
        baseItinerary(),
        {
          ...baseItinerary(),
          days: [...baseItinerary().days, day(4, [item("10:00", "activity", "Park")])]
        },
        "local"
      ).map((change) => change.type)
    ).toContain("day_added");

    expect(
      diffItineraries(
        baseItinerary(),
        { ...baseItinerary(), days: [baseItinerary().days[0]] },
        "local"
      ).map((change) => change.type)
    ).toContain("day_removed");

    expect(
      diffItineraries(baseItinerary(), withDayItems(1, [lunch, walk]), "local").map(
        (change) => change.type
      )
    ).toContain("item_reordered");
  });

  it("ignores volatile metadata but detects budget and place changes", () => {
    const base = baseItinerary();
    const metadataOnly = withItem(1, 0, {
      ...walk,
      placeEnrichment: { status: "matched", reviewStatus: "pending", matchedAt: "2026-01-01" },
      priceEnrichment: { status: "matched", updatedAt: "2026-01-01" }
    });
    expect(diffItineraries(base, metadataOnly, "remote")).toEqual([]);

    const costChange = withItem(1, 0, {
      ...walk,
      estimatedCost: { amount: 10, currency: "EUR", category: "ticket" }
    });
    expect(diffItineraries(base, costChange, "local").map((change) => change.type)).toEqual([
      "item_modified"
    ]);

    const placeChange = withItem(1, 0, {
      ...walk,
      place: {
        provider: "mock",
        providerPlaceId: "place-1",
        name: "Old Town",
        address: "Main Street"
      }
    });
    expect(diffItineraries(base, placeChange, "local").map((change) => change.type)).toEqual([
      "item_modified"
    ]);
  });
});

describe("itinerary merge", () => {
  it("safely merges local and remote changes on different days", () => {
    const result = mergeItineraries(
      baseItinerary(),
      withItem(3, 0, { ...item("10:00", "activity", "Beach"), note: "Bring towels" }),
      withItem(1, 1, { ...lunch, note: "Remote lunch" }),
      { baseRevision: 12, latestRevision: 13 }
    );

    expect(result.safety).toBe("safe");
    expect(dayItems(result.mergedItinerary, 1)[1].note).toBe("Remote lunch");
    expect(dayItems(result.mergedItinerary, 3)[0].note).toBe("Bring towels");
  });

  it("safely merges different items in the same day when neither side reordered", () => {
    const result = mergeItineraries(
      baseItinerary(),
      withItem(1, 0, { ...walk, note: "Local walk" }),
      withItem(1, 1, { ...lunch, note: "Remote lunch" }),
      { baseRevision: 1, latestRevision: 2 }
    );

    expect(result.safety).toBe("safe");
    expect(dayItems(result.mergedItinerary, 1)[0].note).toBe("Local walk");
    expect(dayItems(result.mergedItinerary, 1)[1].note).toBe("Remote lunch");
  });

  it("keeps latest remote changes while applying unrelated local additions", () => {
    const result = mergeItineraries(
      baseItinerary(),
      withDayItems(2, [museum, item("17:00", "activity", "Viewpoint"), dinner]),
      withItem(1, 0, { ...walk, note: "Remote walk" }),
      { baseRevision: 1, latestRevision: 2 }
    );

    expect(result.safety).toBe("safe");
    expect(dayItems(result.mergedItinerary, 1)[0].note).toBe("Remote walk");
    expect(dayItems(result.mergedItinerary, 2).map((entry) => entry.name)).toEqual([
      "Museum",
      "Viewpoint",
      "Dinner"
    ]);
  });

  it("marks overlapping item edits and removals as conflicts", () => {
    const sameItem = mergeItineraries(
      baseItinerary(),
      withItem(1, 0, { ...walk, note: "Local" }),
      withItem(1, 0, { ...walk, note: "Remote" }),
      { baseRevision: 1, latestRevision: 2 }
    );
    expect(sameItem.safety).toBe("partial_conflict");
    expect(sameItem.conflicts).toHaveLength(1);

    const removedVsModified = mergeItineraries(
      baseItinerary(),
      withDayItems(1, [lunch]),
      withItem(1, 0, { ...walk, note: "Remote" }),
      { baseRevision: 1, latestRevision: 2 }
    );
    expect(removedVsModified.safety).toBe("partial_conflict");

    const modifiedVsRemoved = mergeItineraries(
      baseItinerary(),
      withItem(1, 0, { ...walk, note: "Local" }),
      withDayItems(1, [lunch]),
      { baseRevision: 1, latestRevision: 2 }
    );
    expect(modifiedVsRemoved.safety).toBe("partial_conflict");
  });

  it("marks same-day reorder and broad day changes as unsafe", () => {
    const reorder = mergeItineraries(
      baseItinerary(),
      withDayItems(1, [lunch, walk]),
      withDayItems(1, [lunch, walk]),
      { baseRevision: 1, latestRevision: 2 }
    );
    expect(reorder.safety).toBe("unsafe");

    const remoteReorder = mergeItineraries(
      baseItinerary(),
      withItem(1, 0, { ...walk, note: "Local" }),
      withDayItems(1, [lunch, walk]),
      { baseRevision: 1, latestRevision: 2 }
    );
    expect(remoteReorder.safety).toBe("unsafe");

    const dayReplacement = mergeItineraries(
      baseItinerary(),
      withItem(2, 0, { ...museum, note: "Local" }),
      { ...baseItinerary(), days: [baseItinerary().days[0], day(2, [item("08:00", "activity", "Remote day")]), baseItinerary().days[2]] },
      { baseRevision: 1, latestRevision: 2 }
    );
    expect(dayReplacement.safety).toBe("unsafe");
  });

  it("applies selected conflict resolutions", () => {
    const result = mergeItineraries(
      baseItinerary(),
      withItem(1, 0, { ...walk, note: "Local" }),
      withItem(1, 0, { ...walk, note: "Remote" }),
      { baseRevision: 1, latestRevision: 2 }
    );
    const conflictKey = result.conflicts[0].conflictKey;

    const keepLatest = applyConflictResolutions(withItem(1, 0, { ...walk, note: "Remote" }), result, {
      [conflictKey]: "keep_latest"
    });
    expect(dayItems(keepLatest, 1)[0].note).toBe("Remote");

    const keepMine = applyConflictResolutions(withItem(1, 0, { ...walk, note: "Remote" }), result, {
      [conflictKey]: "keep_mine"
    });
    expect(dayItems(keepMine, 1)[0].note).toBe("Local");
  });
});

function baseItinerary(): Itinerary {
  return {
    destination: "Rome",
    currency: "EUR",
    days: [day(1, [walk, lunch]), day(2, [museum, dinner]), day(3, [item("10:00", "activity", "Beach")])]
  };
}

function day(dayNumber: number, items: ItineraryItem[]) {
  return {
    day: dayNumber,
    title: `Day ${dayNumber}`,
    items
  };
}

function item(time: string, type: string, name: string): ItineraryItem {
  return {
    time,
    type,
    name,
    note: "",
    estimatedCost: null,
    place: null,
    placeEnrichment: null
  };
}

function withDayItems(dayNumber: number, items: ItineraryItem[]): Itinerary {
  return {
    ...baseItinerary(),
    days: baseItinerary().days.map((entry) =>
      entry.day === dayNumber ? { ...entry, items } : entry
    )
  };
}

function withItem(dayNumber: number, itemIndex: number, nextItem: ItineraryItem): Itinerary {
  return {
    ...baseItinerary(),
    days: baseItinerary().days.map((entry) =>
      entry.day === dayNumber
        ? {
            ...entry,
            items: entry.items.map((itemEntry, index) => (index === itemIndex ? nextItem : itemEntry))
          }
        : entry
    )
  };
}

function dayItems(itinerary: Itinerary | null | undefined, dayNumber: number) {
  return itinerary?.days.find((entry) => entry.day === dayNumber)?.items ?? [];
}
