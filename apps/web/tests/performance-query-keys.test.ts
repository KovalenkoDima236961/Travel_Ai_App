import { describe, expect, it } from "vitest";

import { queryKeys } from "@/lib/query-keys";
import { buildTripCommandCenterDataFromSummary } from "@/lib/trip-command-center/summary";
import type { CommandCenterSummary } from "@/lib/api/command-center";

describe("performance query contracts", () => {
  it("normalizes filter order into one stable trip-scoped key", () => {
    expect(queryKeys.trip.expenses("trip-1", { offset: 20, category: "food" })).toEqual(
      queryKeys.trip.expenses("trip-1", { category: "food", offset: 20 })
    );
    expect(queryKeys.trip.expenses("trip-1", { category: "food" }).slice(0, 3)).toEqual([
      "trips",
      "detail",
      "trip-1"
    ]);
  });

  it("builds the command center from the compact summary without detailed module data", () => {
    const data = buildTripCommandCenterDataFromSummary(summary(), {
      online: true,
      availableOffline: false,
      pendingCount: 0,
      failedCount: 0,
      conflictCount: 0,
      syncing: false
    });

    expect(data.cards.map((card) => card.id)).toContain("health");
    expect(data.cards.map((card) => card.id)).toContain("budget");
    expect(data.nextBestAction.id).toBe("route-fix");
    expect(JSON.stringify(data)).not.toMatch(/rawText|storageKey|contentRedacted/);
  });
});

function summary(): CommandCenterSummary {
  return {
    tripId: "trip-1",
    trip: {
      destination: "Rome",
      startDate: "2026-08-10",
      days: 3,
      tripType: "single_destination",
      itineraryRevision: 2,
      updatedAt: "2026-07-17T10:00:00Z",
      travelers: 2,
      budgetCurrency: "EUR",
      accessRole: "owner",
      canEdit: true
    },
    health: {
      score: 70,
      level: "needs_attention",
      summary: "Transport needs attention.",
      criticalIssueCount: 0,
      highIssueCount: 1,
      warningIssueCount: 0,
      topFixes: [{
        id: "route-fix",
        title: "Select transport",
        description: "One route leg has no transport.",
        recommendation: "Choose an option.",
        severity: "high",
        category: "transport",
        label: "Open route",
        href: "#route"
      }]
    },
    budget: {
      confidenceScore: 82,
      confidenceLevel: "high",
      riskLevel: "low",
      summary: "Budget estimates are reliable.",
      coverage: 90,
      currency: "EUR",
      estimatedTotal: { amount: 420, currency: "EUR" },
      actualTotal: { amount: 85, currency: "EUR" },
      budgetExceeded: false,
      missingEstimateCount: 1
    },
    route: {
      stopCount: 2,
      legCount: 1,
      selectedTransportCoverage: 0,
      missingTransportCount: 1
    },
    sectionErrors: [{ section: "activity", code: "activity_summary_timeout", message: "Activity summary is temporarily unavailable." }],
    computedAt: "2026-07-17T10:00:00Z"
  };
}

