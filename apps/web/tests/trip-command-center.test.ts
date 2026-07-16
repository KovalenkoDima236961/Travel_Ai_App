import { describe, expect, it } from "vitest";

import { selectNextBestAction } from "@/lib/trip-command-center/next-best-action";
import {
  buildApprovalPolicyCard,
  buildBudgetReadinessCard,
  buildChecklistReminderCard,
  buildExpenseSettlementCard,
  buildGroupReadinessCard,
  buildOfflineStatusCard,
  buildRouteReadinessCard
} from "@/lib/trip-command-center/readiness";
import type { BudgetSummary } from "@/entities/budget/model";
import type { ChecklistViewResponse } from "@/entities/checklist/model";
import type { ExpenseSummary, SettlementsResponse } from "@/entities/expense/model";
import type { ReminderViewResponse } from "@/entities/trip-reminder/model";
import type { Trip } from "@/entities/trip/model";
import type { TripCommandCenterInput } from "@/types/trip-command-center";
import type { TripAvailabilityList } from "@/types/trip-availability";
import type { TripHealth, TripHealthIssue } from "@/types/trip-health";
import type { PolicyEvaluation } from "@/types/workspace-policy";

describe("Trip Command Center next best action", () => {
  it("chooses a critical health issue first", () => {
    const input = commandCenterInput({
      health: healthWithIssues([
        healthIssue({
          id: "critical-route",
          severity: "critical",
          category: "route",
          title: "Route is broken",
          action: { type: "open", label: "Open route", href: "/trips/trip_1?tab=route" }
        })
      ])
    });

    expect(selectNextBestAction(input).id).toBe("critical-route");
  });

  it("chooses blocking policy before lower-priority trip setup work", () => {
    const input = commandCenterInput({
      trip: { ...baseTrip(), itinerary: null, status: "DRAFT", workspaceId: "workspace_1" },
      policyEvaluation: policyEvaluation({ blockingCount: 1 })
    });

    expect(selectNextBestAction(input).id).toBe("policy_blocking:maxTripBudget");
  });

  it("chooses missing itinerary for an empty trip", () => {
    const input = commandCenterInput({
      trip: { ...baseTrip(), itinerary: null, status: "DRAFT" }
    });

    expect(selectNextBestAction(input).id).toBe("itinerary_missing");
  });

  it("chooses missing transport when a route leg has no selected option", () => {
    const input = commandCenterInput({
      trip: tripWithRouteLeg()
    });

    expect(selectNextBestAction(input).id).toBe("transport_missing_option:leg_1");
  });

  it("chooses budget exceeded when cost is over budget", () => {
    const input = commandCenterInput({
      budgetSummary: budgetSummary({ estimatedTotal: 1200, tripBudget: 900, overBudgetBy: 300 })
    });

    expect(selectNextBestAction(input).id).toBe("budget_exceeded");
  });

  it("chooses an allowed collaborator action for a viewer", () => {
    const input = commandCenterInput({
      trip: tripWithRouteLeg(),
      availability: availability({ missingCount: 1 }),
      userAccess: {
        canEdit: false,
        canCollaborate: true,
        canView: true,
        currentUserId: "user_1"
      }
    });

    expect(selectNextBestAction(input).id).toBe("availability_missing");
  });

  it("returns ready state when there are no issues", () => {
    const action = selectNextBestAction(
      commandCenterInput({ trip: { ...baseTrip(), days: 1 } })
    );

    expect(action.id).toBe("trip_ready");
    expect(action.title).toBe("Trip looks ready");
  });
});

describe("Trip Command Center readiness cards", () => {
  it("builds route readiness status", () => {
    const card = buildRouteReadinessCard(commandCenterInput({ trip: tripWithRouteLeg() }));

    expect(card.status).toBe("needs_attention");
    expect(card.metrics.find((metric) => metric.label === "Ready legs")?.value).toBe("0/1");
  });

  it("builds budget readiness status", () => {
    const card = buildBudgetReadinessCard(
      commandCenterInput({
        budgetSummary: budgetSummary({ estimatedTotal: 800, tripBudget: 1000, missingEstimateCount: 2 })
      })
    );

    expect(card.status).toBe("almost_ready");
  });

  it("builds group readiness status", () => {
    const card = buildGroupReadinessCard(
      commandCenterInput({
        trip: { ...baseTrip(), workspaceId: "workspace_1" },
        availability: availability({ totalCollaborators: 3, missingCount: 1 })
      })
    );

    expect(card.status).toBe("needs_attention");
  });

  it("builds checklist and reminder status", () => {
    const card = buildChecklistReminderCard(
      commandCenterInput({
        checklist: checklist({ highPriorityUnchecked: 1 }),
        reminders: reminders({ overdue: 1 })
      })
    );

    expect(card.status).toBe("needs_attention");
  });

  it("builds expenses and settlement status", () => {
    const card = buildExpenseSettlementCard(
      commandCenterInput({
        trip: { ...baseTrip(), startDate: "2020-01-01", days: 2 },
        expenseSummary: expenseSummary({ pendingCount: 2 }),
        settlements: settlements()
      })
    );

    expect(card.status).toBe("needs_attention");
  });

  it("builds approval and policy status", () => {
    const card = buildApprovalPolicyCard(
      commandCenterInput({
        trip: { ...baseTrip(), workspaceId: "workspace_1" },
        policyEvaluation: policyEvaluation({ blockingCount: 1 })
      })
    );

    expect(card.status).toBe("blocked");
  });

  it("builds offline status", () => {
    const card = buildOfflineStatusCard(
      commandCenterInput({
        offlineStatus: {
          online: true,
          availableOffline: true,
          pendingCount: 1,
          failedCount: 0,
          conflictCount: 0,
          syncing: false,
          cachedAt: "2026-01-01T00:00:00Z"
        }
      })
    );

    expect(card.status).toBe("almost_ready");
  });
});

function commandCenterInput(
  overrides: Partial<TripCommandCenterInput> = {}
): TripCommandCenterInput {
  return {
    trip: overrides.trip ?? baseTrip(),
    health: overrides.health ?? healthWithIssues([]),
    budgetSummary: overrides.budgetSummary ?? budgetSummary(),
    availability: overrides.availability ?? null,
    checklist: overrides.checklist ?? null,
    reminders: overrides.reminders ?? null,
    expenseSummary: overrides.expenseSummary ?? null,
    settlements: overrides.settlements ?? null,
    approval: overrides.approval ?? null,
    policyEvaluation: overrides.policyEvaluation ?? null,
    approvalRisk: overrides.approvalRisk ?? null,
    activity: overrides.activity ?? [],
    polls: overrides.polls ?? [],
    generationJobs: overrides.generationJobs ?? [],
    offlineStatus:
      overrides.offlineStatus ?? {
        online: true,
        availableOffline: false,
        pendingCount: 0,
        failedCount: 0,
        conflictCount: 0,
        syncing: false,
        cachedAt: null
      },
    userAccess:
      overrides.userAccess ?? {
        canEdit: true,
        canCollaborate: true,
        canView: true,
        currentUserId: "user_1"
      }
  };
}

function baseTrip(): Trip {
  return {
    id: "trip_1",
    destination: "Salzburg",
    startDate: "2026-08-01",
    days: 3,
    budgetCurrency: "EUR",
    travelers: 2,
    interests: ["culture"],
    pace: "balanced",
    status: "COMPLETED",
    itinerary: { days: [] },
    itineraryRevision: 1,
    createdAt: "2026-01-01T00:00:00Z",
    updatedAt: "2026-01-01T00:00:00Z",
    access: {
      role: "owner",
      canEdit: true,
      canDelete: true,
      canManageCollaborators: true,
      canManageShare: true,
      canRestoreVersion: true
    }
  };
}

function tripWithRouteLeg(): Trip {
  return {
    ...baseTrip(),
    tripType: "multi_destination",
    route: {
      stops: [
        { id: "stop_1", destination: "Salzburg" },
        { id: "stop_2", destination: "Hallstatt" }
      ],
      legs: [
        {
          id: "leg_1",
          fromStopId: "stop_1",
          toStopId: "stop_2",
          fromName: "Salzburg",
          toName: "Hallstatt",
          mode: "train"
        }
      ]
    }
  };
}

function healthIssue(overrides: Partial<TripHealthIssue> = {}): TripHealthIssue {
  return {
    id: "issue_1",
    category: "other",
    severity: "warning",
    status: "open",
    title: "Issue",
    description: "Issue description",
    ...overrides
  };
}

function healthWithIssues(issues: TripHealthIssue[]): TripHealth {
  return {
    tripId: "trip_1",
    score: issues.length > 0 ? 55 : 95,
    level: issues.length > 0 ? "needs_attention" : "ready",
    summary: issues.length > 0 ? "Needs work" : "Ready",
    generatedAt: "2026-01-01T00:00:00Z",
    categories: [],
    issues,
    topFixes: issues.map((issue) => ({
      issueId: issue.id,
      label: issue.title,
      href: issue.action?.href ?? "/trips/trip_1?tab=health"
    })),
    computedFrom: { itineraryRevision: 1 }
  };
}

function budgetSummary(overrides: Partial<BudgetSummary> = {}): BudgetSummary {
  return {
    currency: "EUR",
    tripBudget: 1000,
    estimatedTotal: 700,
    remaining: 300,
    overBudgetBy: null,
    accommodationTotal: null,
    missingEstimateCount: 0,
    estimatedItemCount: 3,
    convertedItemCount: 0,
    unconvertedItemCount: 0,
    unsupportedCurrencyCount: 0,
    originalCurrencyTotals: [],
    conversionWarnings: [],
    exchangeRateInfo: null,
    byDay: [],
    byCategory: [],
    ...overrides
  };
}

function availability({
  totalCollaborators = 2,
  missingCount = 0
}: {
  totalCollaborators?: number;
  missingCount?: number;
} = {}): TripAvailabilityList {
  return {
    tripId: "trip_1",
    responses: [],
    summary: {
      totalCollaborators,
      submittedCount: totalCollaborators - missingCount,
      missingCount,
      missingUsers: []
    }
  };
}

function checklist(overrides: Partial<ChecklistViewResponse["summary"]> = {}): ChecklistViewResponse {
  return {
    checklist: null,
    canGenerate: true,
    summary: {
      totalItems: 4,
      checkedItems: 2,
      uncheckedItems: 2,
      highPriorityUnchecked: 0,
      assignedToMe: 1,
      categories: [],
      ...overrides
    }
  };
}

function reminders(overrides: Partial<ReminderViewResponse["summary"]> = {}): ReminderViewResponse {
  return {
    reminders: [],
    summary: {
      total: 2,
      pending: 1,
      completed: 1,
      overdue: 0,
      dueToday: 0,
      highPriorityPending: 0,
      assignedToMe: 0,
      stale: false,
      ...overrides
    }
  };
}

function expenseSummary({ pendingCount = 0 }: { pendingCount?: number } = {}): ExpenseSummary {
  return {
    tripId: "trip_1",
    currency: "EUR",
    actualTotal: { amount: 400, currency: "EUR" },
    estimatedTotal: { amount: 500, currency: "EUR" },
    plannedVsActual: null,
    originalCurrencyTotals: [],
    byCategory: [],
    byPayer: [],
    balances: [],
    conversionWarnings: [],
    settlementSummary: {
      pendingCount,
      paidCount: 0,
      totalPending: { amount: pendingCount * 50, currency: "EUR" }
    }
  };
}

function settlements(): SettlementsResponse {
  return {
    currency: "EUR",
    suggestions: [],
    paidSettlements: [],
    warnings: []
  };
}

function policyEvaluation({
  blockingCount = 0
}: {
  blockingCount?: number;
} = {}): PolicyEvaluation {
  return {
    tripId: "trip_1",
    workspaceId: "workspace_1",
    policyId: "policy_1",
    status: blockingCount > 0 ? "blocking" : "ok",
    generatedAt: "2026-01-01T00:00:00Z",
    summary: {
      rulesChecked: 1,
      passedCount: blockingCount > 0 ? 0 : 1,
      infoCount: 0,
      warningCount: 0,
      blockingCount
    },
    results:
      blockingCount > 0
        ? [
            {
              ruleKey: "maxTripBudget",
              status: "violation",
              severity: "blocking",
              title: "Trip budget is too high",
              message: "Reduce the trip budget.",
              affectedItems: [],
              suggestedActions: []
            }
          ]
        : [],
    warnings: [],
    notApplicableReason: null
  };
}
