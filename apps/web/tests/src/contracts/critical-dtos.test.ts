import { describe, expect, it } from "vitest";
import { listTrips } from "@/lib/api/trips";
import { budgetSummaryFixture, notificationsFixture, tripFixture } from "../../../test/fixtures";

describe("critical API contract fixtures", () => {
  it("keeps the frontend Trip list handler aligned with the canonical fixture", async () => {
    const response = await listTrips({ limit: 20, offset: 0 });

    expect(response.items).toEqual([tripFixture]);
    expect(response.items[0].itineraryRevision).toBe(3);
    expect(response.items[0].access?.role).toBe("owner");
  });

  it("carries regression-sensitive budget and notification fields", () => {
    expect(budgetSummaryFixture.byDay.map((day) => day.estimatedTotal)).toEqual([40, 18]);
    expect(budgetSummaryFixture.missingEstimateCount).toBe(0);
    expect(notificationsFixture[0]).toMatchObject({
      category: "trip_updates",
      groupedCount: 1,
      readAt: null,
      tripId: tripFixture.id
    });
  });
});
