"use client";

import { useMemo } from "react";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import {
  formatDistanceKm,
  formatWalkingTime,
  getDayDistanceSummaries,
  getTripDistanceTotal,
  type DayDistanceSummary
} from "@/lib/itinerary/distance-utils";
import { MIN_OPTIMIZABLE_STOPS } from "@/lib/itinerary/route-optimization-utils";
import { cn } from "@/lib/utils";
import type { Itinerary } from "@/types/trip";

type DistanceSummaryProps = {
  itinerary: Itinerary;
  maxWalkingKmPerDay?: number | null;
  className?: string;
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

  if (measuredDays.length === 0) {
    return (
      <Card className={className}>
        <h2 className="text-xl font-semibold text-slate-950">Distance estimate</h2>
        <div className="mt-4 rounded-lg border border-dashed border-slate-300 bg-slate-50 p-6 text-sm leading-6 text-slate-600">
          Distance estimates will appear after you attach places with coordinates to at least
          two items in a day.
        </div>
      </Card>
    );
  }

  const tripTotalKm = getTripDistanceTotal(measuredDays);

  return (
    <Card className={className}>
      <div className="flex flex-col gap-1">
        <h2 className="text-xl font-semibold text-slate-950">Distance estimate</h2>
        <p className="text-sm leading-6 text-slate-600">
          Approximate straight-line distance between mapped places. Real walking distance may be
          higher.
        </p>
      </div>

      <ul className="mt-5 space-y-3">
        {measuredDays.map((summary) => (
          <DaySummaryRow
            key={summary.dayNumber}
            summary={summary}
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

type DaySummaryRowProps = {
  summary: DayDistanceSummary;
  onOptimizeDay?: (dayNumber: number) => void;
};

function DaySummaryRow({ summary, onOptimizeDay }: DaySummaryRowProps) {
  const stopsLabel = `${summary.mappedStops} mapped ${
    summary.mappedStops === 1 ? "stop" : "stops"
  }`;
  // The summary already counts mapped stops with valid coordinates using the
  // same validation as the optimizer, so this matches canOptimizeDay(day).
  const canOptimize = Boolean(onOptimizeDay) && summary.mappedStops >= MIN_OPTIMIZABLE_STOPS;

  return (
    <li
      className={cn(
        "rounded-lg border p-4",
        summary.exceedsPreference
          ? "border-amber-300 bg-amber-50"
          : "border-slate-200 bg-white"
      )}
    >
      <div className="flex flex-wrap items-center justify-between gap-2">
        <p className="text-sm font-semibold text-slate-950">Day {summary.dayNumber}</p>
        <div className="flex flex-wrap items-center gap-2">
          {summary.exceedsPreference ? (
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

      <p className="mt-1 text-sm text-slate-700">
        {stopsLabel} · approx. {formatDistanceKm(summary.straightLineDistanceKm)} · ~
        {formatWalkingTime(summary.estimatedWalkingMinutes)} walking
      </p>

      {summary.maxWalkingKmPerDay ? (
        <p
          className={cn(
            "mt-1 text-xs",
            summary.exceedsPreference ? "text-amber-900" : "text-slate-500"
          )}
        >
          Your preference: max {summary.maxWalkingKmPerDay} km/day
        </p>
      ) : null}

      <details className="mt-3 text-sm">
        <summary className="cursor-pointer select-none text-xs font-medium text-primary-700 hover:text-primary-600">
          {summary.segmentCount === 1 ? "1 segment" : `${summary.segmentCount} segments`}
        </summary>
        <ul className="mt-2 space-y-1.5 border-l border-slate-200 pl-3">
          {summary.segments.map((segment, index) => (
            <li key={index} className="text-xs leading-5 text-slate-600">
              <span className="font-medium text-slate-800">
                {segment.fromName} → {segment.toName}
              </span>
              : {formatDistanceKm(segment.distanceKm)} · ~
              {formatWalkingTime(segment.estimatedWalkingMinutes)}
            </li>
          ))}
        </ul>
      </details>
    </li>
  );
}
