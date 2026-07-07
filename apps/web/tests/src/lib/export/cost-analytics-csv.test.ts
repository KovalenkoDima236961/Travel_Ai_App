import { describe, expect, it } from "vitest";
import {
  generateTripCostAnalyticsCsv,
  generateWorkspaceCostAnalyticsCsv
} from "@/lib/export/cost-analytics-csv";
import type {
  TripCostAnalytics,
  WorkspaceCostAnalytics
} from "@/entities/cost-analytics/model";

describe("cost analytics CSV export", () => {
  it("exports trip report sections and escapes values", () => {
    const csv = generateTripCostAnalyticsCsv(baseTripAnalytics());

    expect(csv).toContain("Summary");
    expect(csv).toContain("Cost by day");
    expect(csv).toContain("Expensive items");
    expect(csv).toContain('"Museum, East Wing"');
    expect(csv).toContain("Costs are estimates for planning purposes only.");
  });

  it("exports workspace report sections", () => {
    const csv = generateWorkspaceCostAnalyticsCsv(baseWorkspaceAnalytics());

    expect(csv).toContain("Cost by trip");
    expect(csv).toContain("Cost by month");
    expect(csv).toContain("Tokyo 2026");
  });
});

function baseTripAnalytics(): TripCostAnalytics {
  return {
    tripId: "trip-1",
    workspaceId: null,
    currency: "EUR",
    generatedAt: "2026-07-03T10:00:00Z",
    summary: {
      budgetAmount: 500,
      estimatedTotal: 620,
      remainingAmount: -120,
      overBudgetAmount: 120,
      budgetUtilizationPercent: 124,
      itemEstimatedTotal: 520,
      accommodationTotal: 100,
      missingEstimateCount: 1,
      uncertainEstimateCount: 1,
      convertedItemCount: 4,
      unconvertedItemCount: 0
    },
    byDay: [
      {
        dayNumber: 1,
        date: "2026-08-10",
        estimatedTotal: 220,
        budgetShare: 250,
        overBudgetAmount: 0,
        missingEstimateCount: 1,
        topItems: []
      }
    ],
    byCategory: [{ category: "ticket", amount: 220, percentage: 35.48, itemCount: 2 }],
    bySource: [{ source: "provider", amount: 220, percentage: 35.48, itemCount: 2 }],
    byConfidence: [{ confidence: "high", amount: 220, percentage: 35.48, itemCount: 2 }],
    originalCurrencyTotals: [],
    expensiveItems: [
      {
        dayNumber: 1,
        itemIndex: 0,
        name: "Museum, East Wing",
        type: "ticket",
        category: "ticket",
        amount: 120,
        currency: "EUR",
        convertedAmount: 120,
        source: "provider",
        confidence: "high",
        percentageOfTrip: 19.35
      }
    ],
    insights: [],
    warnings: []
  };
}

function baseWorkspaceAnalytics(): WorkspaceCostAnalytics {
  return {
    workspaceId: "workspace-1",
    currency: "EUR",
    generatedAt: "2026-07-03T10:00:00Z",
    dateRange: { from: null, to: null },
    summary: {
      tripCount: 1,
      estimatedTotal: 620,
      budgetTotal: 500,
      overBudgetTripCount: 1,
      missingEstimateCount: 1,
      uncertainEstimateCount: 1,
      convertedItemCount: 4,
      unconvertedItemCount: 0,
      incompleteBudgetTripCount: 0
    },
    byTrip: [
      {
        tripId: "trip-1",
        title: "Tokyo 2026",
        destination: "Tokyo",
        startDate: "2026-09-10",
        endDate: "2026-09-20",
        budgetAmount: 500,
        estimatedTotal: 620,
        overBudgetAmount: 120,
        missingEstimateCount: 1,
        workspaceId: "workspace-1"
      }
    ],
    byCategory: [{ category: "ticket", amount: 220, percentage: 35.48, itemCount: 2 }],
    bySource: [{ source: "provider", amount: 220, percentage: 35.48, itemCount: 2 }],
    byMonth: [{ month: "2026-09", estimatedTotal: 620, tripCount: 1 }],
    expensiveTrips: [],
    expensiveItems: [],
    insights: [],
    warnings: []
  };
}
