import { apiFetch } from "@/shared/api/client";
import type {
  TripCostAnalytics,
  WorkspaceCostAnalytics,
  WorkspaceCostAnalyticsParams
} from "@/entities/cost-analytics/model";

export const costAnalyticsKeys = {
  all: ["cost-analytics"] as const,
  trip: (tripId: string, currency?: string | null) =>
    [...costAnalyticsKeys.all, "trip", tripId, currency ?? null] as const,
  workspace: (workspaceId: string, params: WorkspaceCostAnalyticsParams = {}) =>
    [...costAnalyticsKeys.all, "workspace", workspaceId, normalizeWorkspaceParams(params)] as const
};

export function getTripCostAnalytics(tripId: string, currency?: string | null) {
  const params = new URLSearchParams();
  if (currency) {
    params.set("currency", currency.trim().toUpperCase());
  }
  const query = params.toString();
  return apiFetch<TripCostAnalytics>(
    `/trips/${tripId}/analytics/costs${query ? `?${query}` : ""}`
  );
}

export function getWorkspaceCostAnalytics(
  workspaceId: string,
  params: WorkspaceCostAnalyticsParams = {}
) {
  const searchParams = new URLSearchParams();
  const normalized = normalizeWorkspaceParams(params);

  if (normalized.currency) {
    searchParams.set("currency", normalized.currency);
  }
  if (normalized.from) {
    searchParams.set("from", normalized.from);
  }
  if (normalized.to) {
    searchParams.set("to", normalized.to);
  }
  if (normalized.includeArchived) {
    searchParams.set("includeArchived", "true");
  }

  const query = searchParams.toString();
  return apiFetch<WorkspaceCostAnalytics>(
    `/workspaces/${workspaceId}/analytics/costs${query ? `?${query}` : ""}`
  );
}

function normalizeWorkspaceParams(params: WorkspaceCostAnalyticsParams) {
  return {
    currency: params.currency?.trim().toUpperCase() || undefined,
    from: params.from?.trim() || undefined,
    to: params.to?.trim() || undefined,
    includeArchived: params.includeArchived === true
  };
}
