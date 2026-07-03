import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import { OfflineBanner } from "@/components/offline/OfflineBanner";
import { PendingOfflineChangesPanel } from "@/components/offline/PendingOfflineChangesPanel";
import type { PendingItineraryMutation } from "@/lib/offline/types";

describe("OfflineBanner", () => {
  it("shows offline saved-copy status", () => {
    const html = renderToStaticMarkup(
      <OfflineBanner
        cachedAt="2026-07-03T10:42:00Z"
        offlineCopy
        online={false}
        pendingCount={1}
      />
    );

    expect(html).toContain("You are offline. Viewing saved data.");
    expect(html).toContain("1 pending offline change");
  });

  it("shows sync and conflict states", () => {
    expect(
      renderToStaticMarkup(<OfflineBanner online pendingCount={1} syncing />)
    ).toContain("Syncing offline changes");

    expect(
      renderToStaticMarkup(<OfflineBanner conflictCount={1} online pendingCount={1} />)
    ).toContain("Offline changes need review");
  });
});

describe("PendingOfflineChangesPanel", () => {
  it("disables sync while offline and renders mutation metadata", () => {
    const html = renderToStaticMarkup(
      <PendingOfflineChangesPanel
        mutation={mutation()}
        online={false}
        onDiscard={() => {}}
        onReview={() => {}}
        onSyncNow={() => {}}
        syncing={false}
      />
    );

    expect(html).toContain("Pending offline itinerary changes");
    expect(html).toContain("Base revision");
    expect(html).toContain("Discard offline changes");
    expect(html).toContain("disabled");
    expect(html).toContain("This action requires an internet connection.");
  });
});

function mutation(): PendingItineraryMutation {
  return {
    mutationId: "mutation-1",
    type: "update_itinerary",
    tripId: "trip-1",
    userId: "user-1",
    baseRevision: 7,
    baseItinerary: {
      days: [
        {
          day: 1,
          title: "Day 1",
          items: [{ time: "09:00", type: "activity", name: "Base" }]
        }
      ]
    },
    draftItinerary: {
      days: [
        {
          day: 1,
          title: "Day 1",
          items: [{ time: "09:00", type: "activity", name: "Draft" }]
        }
      ]
    },
    status: "pending",
    createdAt: "2026-07-03T10:00:00Z",
    updatedAt: "2026-07-03T10:30:00Z"
  };
}
