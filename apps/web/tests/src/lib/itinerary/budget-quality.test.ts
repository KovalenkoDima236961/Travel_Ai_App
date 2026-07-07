import { describe, expect, it } from "vitest";
import { getBudgetIssues } from "@/entities/itinerary/model/quality-analyzer";
import type { BudgetSummary } from "@/entities/budget/model";
import type { Itinerary } from "@/entities/trip/model";

function itinerary(): Itinerary {
  return {
    days: [
      {
        day: 1,
        title: "Day 1",
        items: [
          {
            time: "09:00",
            type: "ticket",
            name: "Museum",
            estimatedCost: { amount: 20, currency: "EUR", category: "ticket" }
          },
          { time: "12:00", type: "transport", name: "Metro" },
          { time: "14:00", type: "food", name: "Lunch" },
          { time: "16:00", type: "activity", name: "Tour" }
        ]
      }
    ]
  };
}

function summary(overrides: Partial<BudgetSummary> = {}): BudgetSummary {
  return {
    currency: "EUR",
    tripBudget: 100,
    estimatedTotal: 120,
    remaining: -20,
    overBudgetBy: 20,
    missingEstimateCount: 3,
    estimatedItemCount: 1,
    byDay: [
      {
        dayNumber: 1,
        estimatedTotal: 140,
        missingEstimateCount: 3,
        dailyBudgetShare: 100,
        overDailyBudgetBy: 40
      }
    ],
    byCategory: [],
    ...overrides
  };
}

describe("getBudgetIssues", () => {
  it("detects trip_budget_exceeded", () => {
    const issues = getBudgetIssues({
      itinerary: itinerary(),
      budgetSummary: summary({ estimatedTotal: 130, overBudgetBy: 30 })
    });
    const issue = issues.find((entry) => entry.type === "trip_budget_exceeded");
    expect(issue).toBeTruthy();
    // 130 > 100 * 1.2 -> critical
    expect(issue?.severity).toBe("critical");
    expect(issue?.scope).toBe("trip");
    expect(issue?.message).toContain("about 130 EUR");
  });

  it("uses warning severity when modestly over budget", () => {
    const issues = getBudgetIssues({
      itinerary: itinerary(),
      budgetSummary: summary({ estimatedTotal: 110, overBudgetBy: 10 })
    });
    const issue = issues.find((entry) => entry.type === "trip_budget_exceeded");
    expect(issue?.severity).toBe("warning");
  });

  it("detects day_budget_high when a day exceeds its share", () => {
    const issues = getBudgetIssues({
      itinerary: itinerary(),
      budgetSummary: summary()
    });
    const issue = issues.find((entry) => entry.type === "day_budget_high");
    expect(issue).toBeTruthy();
    expect(issue?.dayNumber).toBe(1);
  });

  it("groups missing cost estimates per day when many are missing", () => {
    const issues = getBudgetIssues({
      itinerary: itinerary(),
      budgetSummary: summary()
    });
    const grouped = issues.filter((entry) => entry.type === "missing_cost_estimate");
    // 3 paid items without a cost -> one grouped day issue.
    expect(grouped).toHaveLength(1);
    expect(grouped[0].scope).toBe("day");
  });

  it("emits per-item missing estimates when only a few are missing", () => {
    const onlyOneMissing: Itinerary = {
      days: [
        {
          day: 1,
          title: "Day 1",
          items: [
            {
              time: "09:00",
              type: "ticket",
              name: "Museum",
              estimatedCost: { amount: 20, currency: "EUR" }
            },
            { time: "12:00", type: "food", name: "Lunch" }
          ]
        }
      ]
    };
    const issues = getBudgetIssues({
      itinerary: onlyOneMissing,
      budgetSummary: summary({ missingEstimateCount: 1 })
    });
    const missing = issues.filter((entry) => entry.type === "missing_cost_estimate");
    expect(missing).toHaveLength(1);
    expect(missing[0].scope).toBe("item");
    expect(missing[0].itemIndex).toBe(1);
  });

  it("detects expensive_item above the budget share", () => {
    const expensive: Itinerary = {
      days: [
        {
          day: 1,
          title: "Day 1",
          items: [
            {
              time: "09:00",
              type: "activity",
              name: "Private boat tour",
              estimatedCost: { amount: 60, currency: "EUR", category: "activity" }
            }
          ]
        }
      ]
    };
    // Budget 100 -> 30% share is 30; a 60 item is over that -> warning.
    const issues = getBudgetIssues({
      itinerary: expensive,
      budgetSummary: summary({ estimatedTotal: 60, overBudgetBy: 0, missingEstimateCount: 0 })
    });
    const issue = issues.find((entry) => entry.type === "expensive_item");
    expect(issue).toBeTruthy();
    expect(issue?.severity).toBe("warning");
    expect(issue?.itemIndex).toBe(0);
  });

  it("detects conversion_unavailable when costs could not be converted", () => {
    const issues = getBudgetIssues({
      itinerary: itinerary(),
      budgetSummary: summary({
        estimatedTotal: 80,
        overBudgetBy: 0,
        unconvertedItemCount: 1,
        conversionWarnings: [{ currency: "XXX", amount: 99, reason: "unsupported_currency" }]
      })
    });

    const issue = issues.find((entry) => entry.type === "conversion_unavailable");
    expect(issue).toBeTruthy();
    expect(issue?.severity).toBe("info");
    expect(issue?.message).toContain("could not be converted");
  });

  it("returns no issues without a budget summary", () => {
    expect(getBudgetIssues({ itinerary: itinerary() })).toEqual([]);
  });
});
