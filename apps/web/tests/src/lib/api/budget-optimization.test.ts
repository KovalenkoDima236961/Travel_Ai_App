import { afterEach, describe, expect, it, vi } from "vitest";
import {
  applyBudgetOptimizationProposal,
  createBudgetOptimizationJob,
  discardBudgetOptimizationProposal,
  listBudgetOptimizationProposals
} from "@/lib/api/budget-optimization";
import type { BudgetOptimizationProposal } from "@/entities/budget-optimization/model";
import type { GenerationJob } from "@/entities/generation-job/model";

const job: GenerationJob = {
  id: "job-1",
  tripId: "trip-1",
  requestedByUserId: "user-1",
  jobType: "budget_optimization_day",
  status: "queued",
  expectedItineraryRevision: 4,
  dayNumber: 2,
  createdAt: "2026-06-25T00:00:00Z",
  updatedAt: "2026-06-25T00:00:00Z"
};

const proposal: BudgetOptimizationProposal = {
  id: "proposal-1",
  tripId: "trip-1",
  jobId: "job-1",
  createdByUserId: "user-1",
  scope: "day",
  dayNumber: 2,
  status: "pending",
  expectedItineraryRevision: 4,
  baseItineraryRevision: 4,
  currency: "EUR",
  targetReductionAmount: 50,
  estimatedSavingsAmount: 40,
  proposal: {
    summary: "Cheaper day proposal.",
    scope: "day",
    dayNumber: 2,
    currency: "EUR",
    baseDayEstimatedTotal: 140,
    proposedDayEstimatedTotal: 100,
    estimatedSavingsAmount: 40,
    confidence: "medium",
    changes: [
      {
        type: "replace_item",
        oldItemIndex: 1,
        oldItemName: "Paid tour",
        newItemName: "Self-guided visit",
        estimatedSavingsAmount: 40,
        currency: "EUR"
      }
    ],
    proposedDay: {
      day: 2,
      title: "Budget day",
      items: [
        {
          time: "09:00",
          type: "activity",
          name: "Self-guided visit",
          estimatedCost: { amount: 20, currency: "EUR", source: "ai" }
        }
      ]
    }
  },
  createdAt: "2026-06-25T00:00:00Z",
  updatedAt: "2026-06-25T00:00:00Z"
};

function jsonResponse(body: unknown, init: { ok: boolean; status: number }): Response {
  return {
    ok: init.ok,
    status: init.status,
    text: async () => JSON.stringify(body),
    json: async () => body
  } as unknown as Response;
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("budget optimization API", () => {
  it("creates a budget optimization job with revision and constraints", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ job }, { ok: true, status: 202 }));
    vi.stubGlobal("fetch", fetchMock);

    const result = await createBudgetOptimizationJob("trip-1", {
      scope: "day",
      dayNumber: 2,
      targetReductionAmount: 50,
      currency: " eur ",
      expectedItineraryRevision: 4,
      constraints: {
        preserveMustSeeItems: true,
        keepMealCount: true,
        avoidReplacingManualCosts: true,
        maxWalkingIncreaseKm: 2
      },
      instruction: " keep museums "
    });

    expect(result).toEqual(job);
    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining("/trips/trip-1/budget-optimization-jobs"),
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({
          scope: "day",
          dayNumber: 2,
          expectedItineraryRevision: 4,
          targetReductionAmount: 50,
          currency: "EUR",
          constraints: {
            preserveMustSeeItems: true,
            keepMealCount: true,
            avoidReplacingManualCosts: true,
            maxWalkingIncreaseKm: 2
          },
          instruction: "keep museums"
        })
      })
    );
  });

  it("lists, applies, and discards proposals", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        jsonResponse({ proposals: [proposal], limit: 20 }, { ok: true, status: 200 })
      )
      .mockResolvedValueOnce(
        jsonResponse({ trip: { id: "trip-1" }, proposal }, { ok: true, status: 200 })
      )
      .mockResolvedValueOnce(jsonResponse({ proposal }, { ok: true, status: 200 }));
    vi.stubGlobal("fetch", fetchMock);

    await expect(listBudgetOptimizationProposals("trip-1", "pending")).resolves.toEqual([
      proposal
    ]);
    await expect(
      applyBudgetOptimizationProposal("trip-1", "proposal-1", 4)
    ).resolves.toEqual({ trip: { id: "trip-1" }, proposal });
    await expect(discardBudgetOptimizationProposal("trip-1", "proposal-1")).resolves.toEqual(
      proposal
    );

    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      expect.stringContaining("/trips/trip-1/budget-optimization-proposals/proposal-1/apply"),
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ expectedItineraryRevision: 4 })
      })
    );
  });
});
