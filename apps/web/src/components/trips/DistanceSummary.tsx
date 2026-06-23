"use client";

import { useMemo } from "react";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import {
  formatDistanceKm,
  formatWalkingTime,
  getDayDistanceSummaries,
  type DayDistanceSummary
} from "@/lib/itinerary/distance-utils";
import { useRouteEstimates, type DayRouteEstimateState } from "@/lib/hooks/useRouteEstimates";
import { MIN_OPTIMIZABLE_STOPS } from "@/lib/itinerary/route-optimization-utils";
import { cn } from "@/lib/utils";
import type { Itinerary } from "@/types/trip";

type DistanceSummaryProps = {
  itinerary: Itinerary;
  maxWalkingKmPerDay?: number | null;
  className?: string;
  /**
   * When false, route estimates are not requested and only the Haversine
   * straight-line estimate is shown. Defaults to true.
   */
  routeEstimatesEnabled?: boolean;
  /**
   * Called when the user asks to optimize a day's order. When omitted (e.g. in
   * read-only previews) the optimize controls are not rendered.
   */
  onOptimizeDay?: (dayNumber: number) => void;
};

export function DistanceSummary({
  itinerary,
  maxWalkingKmPerDay,
  className,
  routeEstimatesEnabled = true,
  onOptimizeDay
}: DistanceSummaryProps) {
  const summaries = useMemo(
    () => getDayDistanceSummaries(itinerary, maxWalkingKmPerDay),
    [itinerary, maxWalkingKmPerDay]
  );

  const measuredDays = useMemo(
    () => summaries.filter((summary) => summary.segmentCount > 0),
    [summaries]
  );

  // Hooks must run before any early return. The hook itself produces no queries
  // when there are no days with at least two mapped stops, so it is cheap here.
  const routeEstimates = useRouteEstimates(itinerary, routeEstimatesEnabled);

  const routeProvider = useMemo(() => {
    for (const summary of measuredDays) {
      const estimate = routeEstimates.byDay.get(summary.dayNumber)?.estimate;
      if (estimate) {
        return estimate.provider;
      }
    }
    return null;
  }, [measuredDays, routeEstimates]);

  if (measuredDays.length === 0) {
    return (
      <Card className={className}>
        <h2 className="text-xl font-semibold text-slate-950">Distance estimate</h2>
        <div className="mt-4 rounded-lg border border-dashed border-slate-300 bg-slate-50 p-6 text-sm leading-6 text-slate-600">
          Add at least two mapped places in a day to estimate travel distance. Attach places with
          coordinates to itinerary items to see route and straight-line estimates here.
        </div>
      </Card>
    );
  }

  // Total reflects the route estimate where available and falls back to the
  // straight-line distance for days the route service has not (yet) answered.
  const tripTotalKm = measuredDays.reduce((total, summary) => {
    const estimate = routeEstimates.byDay.get(summary.dayNumber)?.estimate;
    return total + (estimate ? estimate.distanceKm : summary.straightLineDistanceKm);
  }, 0);

  return (
    <Card className={className}>
      <div className="flex flex-col gap-2">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h2 className="text-xl font-semibold text-slate-950">Distance estimate</h2>
          <RouteSourceBadge
            provider={routeProvider}
            isLoading={routeEstimates.isAnyLoading}
          />
        </div>
        <p className="text-sm leading-6 text-slate-600">
          Route estimates come from the External Integrations Service (approximate, not turn-by-turn).
          When it is unavailable, a straight-line Haversine estimate is shown instead.
        </p>
      </div>

      <ul className="mt-5 space-y-3">
        {measuredDays.map((summary) => (
          <DaySummaryRow
            key={summary.dayNumber}
            summary={summary}
            routeState={routeEstimates.byDay.get(summary.dayNumber)}
            onOptimizeDay={onOptimizeDay}
          />
        ))}
      </ul>

      {measuredDays.length > 1 ? (
        <p className="mt-5 border-t border-slate-200 pt-4 text-sm text-slate-600">
          Total mapped distance:{" "}
          <span className="font-semibold text-slate-900">approx. {formatDistanceKm(tripTotalKm)}</span>
        </p>
      ) : null}
    </Card>
  );
}

type RouteSourceBadgeProps = {
  provider: string | null;
  isLoading: boolean;
};

function RouteSourceBadge({ provider, isLoading }: RouteSourceBadgeProps) {
  if (provider) {
    return (
      <span className="inline-flex items-center rounded-full border border-primary-200 bg-primary-50 px-2.5 py-0.5 text-xs font-medium text-primary-700">
        Route estimates by {provider} provider
      </span>
    );
  }

  if (isLoading) {
    return (
      <span className="inline-flex items-center rounded-full border border-slate-200 bg-slate-50 px-2.5 py-0.5 text-xs font-medium text-slate-500">
        Calculating route estimates…
      </span>
    );
  }

  return (
    <span className="inline-flex items-center rounded-full border border-slate-200 bg-slate-50 px-2.5 py-0.5 text-xs font-medium text-slate-500">
      Straight-line fallback estimate
    </span>
  );
}

type DaySummaryRowProps = {
  summary: DayDistanceSummary;
  routeState?: DayRouteEstimateState;
  onOptimizeDay?: (dayNumber: number) => void;
};

function DaySummaryRow({ summary, routeState, onOptimizeDay }: DaySummaryRowProps) {
  const stopsLabel = `${summary.mappedStops} mapped ${
    summary.mappedStops === 1 ? "stop" : "stops"
  }`;
  // The summary already counts mapped stops with valid coordinates using the
  // same validation as the optimizer, so this matches canOptimizeDay(day).
  const canOptimize = Boolean(onOptimizeDay) && summary.mappedStops >= MIN_OPTIMIZABLE_STOPS;

  const routeEstimate = routeState?.estimate ?? null;
  const routeIsLoading = routeState?.isLoading ?? false;
  const routeIsError = routeState?.isError ?? false;
  const usingRoute = routeEstimate !== null;

  // Preference comparison uses the route distance when available, otherwise the
  // straight-line fallback. The warning label states which estimate was used.
  const effectiveDistanceKm = usingRoute
    ? routeEstimate.distanceKm
    : summary.straightLineDistanceKm;
  const hasPreference =
    typeof summary.maxWalkingKmPerDay === "number" && summary.maxWalkingKmPerDay > 0;
  const exceedsPreference = hasPreference && effectiveDistanceKm > summary.maxWalkingKmPerDay!;
  const estimateLabel = usingRoute ? "route estimate" : "straight-line estimate";

  const segments = usingRoute
    ? routeEstimate.segments.map((segment) => ({
        fromName: segment.fromName,
        toName: segment.toName,
        distanceKm: segment.distanceKm,
        durationMinutes: segment.durationMinutes
      }))
    : summary.segments.map((segment) => ({
        fromName: segment.fromName,
        toName: segment.toName,
        distanceKm: segment.distanceKm,
        durationMinutes: segment.estimatedWalkingMinutes
      }));

  return (
    <li
      className={cn(
        "rounded-lg border p-4",
        exceedsPreference ? "border-amber-300 bg-amber-50" : "border-slate-200 bg-white"
      )}
    >
      <div className="flex flex-wrap items-center justify-between gap-2">
        <p className="text-sm font-semibold text-slate-950">Day {summary.dayNumber}</p>
        <div className="flex flex-wrap items-center gap-2">
          {exceedsPreference ? (
            <span className="inline-flex items-center rounded-full border border-amber-300 bg-amber-100 px-2.5 py-0.5 text-xs font-medium text-amber-900">
              Above your walking preference
            </span>
          ) : null}
          {canOptimize ? (
            <Button
              onClick={() => onOptimizeDay?.(summary.dayNumber)}
              size="sm"
              type="button"
              variant="secondary"
            >
              Optimize order
            </Button>
          ) : null}
        </div>
      </div>

      <p className="mt-1 text-sm text-slate-500">{stopsLabel}</p>

      {usingRoute ? (
        <div className="mt-1 space-y-0.5">
          <p className="text-sm text-slate-700">
            Route estimate:{" "}
            <span className="font-medium text-slate-900">
              {formatDistanceKm(routeEstimate.distanceKm)}
            </span>{" "}
            · ~{formatWalkingTime(routeEstimate.durationMinutes)} walking
          </p>
          <p className="text-xs text-slate-500">
            Straight-line fallback: {formatDistanceKm(summary.straightLineDistanceKm)}
          </p>
        </div>
      ) : routeIsLoading ? (
        <div className="mt-1 space-y-0.5">
          <p className="text-sm text-slate-500">Calculating route estimate…</p>
          <p className="text-sm text-slate-700">
            Straight-line estimate:{" "}
            <span className="font-medium text-slate-900">
              {formatDistanceKm(summary.straightLineDistanceKm)}
            </span>{" "}
            · ~{formatWalkingTime(summary.estimatedWalkingMinutes)} walking
          </p>
        </div>
      ) : (
        <div className="mt-1 space-y-0.5">
          {routeIsError ? (
            <p className="text-xs text-amber-700">
              Route service unavailable. Showing straight-line estimate.
            </p>
          ) : null}
          <p className="text-sm text-slate-700">
            Straight-line estimate:{" "}
            <span className="font-medium text-slate-900">
              {formatDistanceKm(summary.straightLineDistanceKm)}
            </span>{" "}
            · ~{formatWalkingTime(summary.estimatedWalkingMinutes)} walking
          </p>
        </div>
      )}

      {hasPreference ? (
        <p
          className={cn(
            "mt-1 text-xs",
            exceedsPreference ? "text-amber-900" : "text-slate-500"
          )}
        >
          {exceedsPreference
            ? `Above your walking preference of ${summary.maxWalkingKmPerDay} km/day (${estimateLabel})`
            : `Your preference: max ${summary.maxWalkingKmPerDay} km/day (${estimateLabel})`}
        </p>
      ) : null}

      <details className="mt-3 text-sm">
        <summary className="cursor-pointer select-none text-xs font-medium text-primary-700 hover:text-primary-600">
          {segments.length === 1 ? "1 segment" : `${segments.length} segments`}
          {usingRoute ? " (route)" : " (straight-line)"}
        </summary>
        <ul className="mt-2 space-y-1.5 border-l border-slate-200 pl-3">
          {segments.map((segment, index) => (
            <li key={index} className="text-xs leading-5 text-slate-600">
              <span className="font-medium text-slate-800">
                {segment.fromName} → {segment.toName}
              </span>
              : {formatDistanceKm(segment.distanceKm)} · ~{formatWalkingTime(segment.durationMinutes)}
            </li>
          ))}
        </ul>
      </details>
    </li>
  );
}
