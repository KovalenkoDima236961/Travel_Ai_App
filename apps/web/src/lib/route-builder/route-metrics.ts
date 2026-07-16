import { getCostAmount } from "@/entities/budget/model";
import type { TripRoute, TripRouteLeg } from "@/entities/route/model";

export type RouteIntensity = "relaxed" | "balanced" | "intense";

export type RouteMetrics = {
  stopCount: number;
  legCount: number;
  totalTransferMinutes: number;
  estimatedTransportCost: number;
  currency: string;
  selectedTransportCount: number;
  selectedTransportCoverage: number;
  lowConfidenceLegCount: number;
  longestTransferMinutes: number;
  intensity: RouteIntensity;
};

export function getRouteMetrics(
  route: TripRoute,
  totalDays: number,
  fallbackCurrency = "EUR"
): RouteMetrics {
  const legs = route.legs ?? [];
  const selectedTransportCount = legs.filter((leg) => Boolean(leg.selectedTransportOption)).length;
  const totalTransferMinutes = legs.reduce((sum, leg) => sum + legDuration(leg), 0);
  const costs = legs.map(legCost);
  const currency =
    legs.find((leg) => leg.selectedTransportOption?.estimatedPrice?.currency)?.selectedTransportOption?.estimatedPrice?.currency ??
    legs.find((leg) => leg.estimatedCost?.currency)?.estimatedCost?.currency ??
    fallbackCurrency;
  const averageTransfer = legs.length > 0 ? totalTransferMinutes / legs.length : 0;
  const stopsPerDay = route.stops.length / Math.max(1, totalDays);

  return {
    stopCount: route.stops.length,
    legCount: legs.length,
    totalTransferMinutes,
    estimatedTransportCost: costs.reduce((sum, value) => sum + value, 0),
    currency,
    selectedTransportCount,
    selectedTransportCoverage: legs.length > 0 ? selectedTransportCount / legs.length : 0,
    lowConfidenceLegCount: legs.filter((leg) =>
      leg.selectedTransportOption?.confidence === "low" || leg.selectedTransportOption?.provider === "mock"
    ).length,
    longestTransferMinutes: legs.reduce((longest, leg) => Math.max(longest, legDuration(leg)), 0),
    intensity:
      averageTransfer > 240 || stopsPerDay > 0.75
        ? "intense"
        : averageTransfer < 120 && stopsPerDay <= 0.45
          ? "relaxed"
          : "balanced"
  };
}

function legDuration(leg: TripRouteLeg): number {
  return Math.max(0, leg.selectedTransportOption?.durationMinutes ?? leg.estimatedDurationMinutes ?? 0);
}

function legCost(leg: TripRouteLeg): number {
  return Math.max(0, leg.selectedTransportOption?.estimatedPrice?.amount ?? getCostAmount(leg.estimatedCost) ?? 0);
}
