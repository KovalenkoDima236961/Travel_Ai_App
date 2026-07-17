type QueryFilters = Record<string, unknown> | undefined;

function stableFilters(filters: QueryFilters) {
  if (!filters) {
    return {};
  }
  return Object.fromEntries(
    Object.entries(filters)
      .filter(([, value]) => value !== undefined)
      .sort(([left], [right]) => left.localeCompare(right))
  );
}

function normalizedCurrency(currency?: string | null) {
  return currency?.trim().toUpperCase() || null;
}

/**
 * Canonical React Query key factory. New trip-detail queries and mutations
 * should use this hierarchy so invalidation can stay scoped to one trip.
 */
export const queryKeys = {
  trip: {
    all: ["trips"] as const,
    detail: (tripId: string) => ["trips", "detail", tripId] as const,
    commandCenter: (tripId: string) =>
      [...queryKeys.trip.detail(tripId), "command-center-summary"] as const,
    health: (tripId: string) => [...queryKeys.trip.detail(tripId), "health"] as const,
    budgetSummary: (tripId: string) =>
      [...queryKeys.trip.detail(tripId), "budget-summary"] as const,
    budgetConfidence: (tripId: string, currency?: string | null) =>
      [...queryKeys.trip.detail(tripId), "budget-confidence", normalizedCurrency(currency)] as const,
    groupReadiness: (tripId: string) =>
      [...queryKeys.trip.detail(tripId), "group-readiness"] as const,
    route: (tripId: string) => [...queryKeys.trip.detail(tripId), "route"] as const,
    activity: (tripId: string, filters?: QueryFilters) =>
      [...queryKeys.trip.detail(tripId), "activity", stableFilters(filters)] as const,
    expenses: (tripId: string, filters?: QueryFilters) =>
      [...queryKeys.trip.detail(tripId), "expenses", stableFilters(filters)] as const,
    expenseSummary: (tripId: string, currency?: string | null) =>
      [...queryKeys.trip.detail(tripId), "expenses", "summary", normalizedCurrency(currency)] as const,
    settlements: (tripId: string, currency?: string | null) =>
      [...queryKeys.trip.detail(tripId), "expenses", "settlements", normalizedCurrency(currency)] as const,
    checklist: (tripId: string) => [...queryKeys.trip.detail(tripId), "checklist"] as const,
    reminders: (tripId: string, filters?: QueryFilters) =>
      [...queryKeys.trip.detail(tripId), "reminders", stableFilters(filters)] as const,
    generationJobs: (tripId: string) =>
      [...queryKeys.trip.detail(tripId), "generation-jobs"] as const,
    approval: (tripId: string) => [...queryKeys.trip.detail(tripId), "approval"] as const,
    policy: (tripId: string) => [...queryKeys.trip.detail(tripId), "policy"] as const
  },
  notifications: {
    all: ["notifications"] as const,
    list: (filters?: QueryFilters) =>
      [...queryKeys.notifications.all, "list", stableFilters(filters)] as const,
    unread: () => [...queryKeys.notifications.all, "unread-count"] as const
  },
  ops: {
    aiGenerations: (filters?: QueryFilters) =>
      ["ops", "ai-generations", stableFilters(filters)] as const,
    aiGeneration: (id: string) => ["ops", "ai-generations", "detail", id] as const
  }
} as const;

