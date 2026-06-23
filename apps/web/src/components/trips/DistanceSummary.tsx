"use client";

import { useMemo } from "react";
import { Card } from "@/components/ui/Card";
import {
  formatDistanceKm,
  formatWalkingTime,
  getDayDistanceSummaries,
  getTripDistanceTotal,
  type DayDistanceSummary
} from "@/lib/itinerary/distance-utils";
import { cn } from "@/lib/utils";
import type { Itinerary } from "@/types/trip";

type DistanceSummaryProps = {
  itinerary: Itinerary;
  maxWalkingKmPerDay?: number | null;
  className?: string;
};

export function DistanceSummary({
  itinerary,
  maxWalkingKmPerDay,
  className
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
          <DaySummaryRow key={summary.dayNumber} summary={summary} />
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
};

function DaySummaryRow({ summary }: DaySummaryRowProps) {
  const stopsLabel = `${summary.mappedStops} mapped ${
    summary.mappedStops === 1 ? "stop" : "stops"
  }`;

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
        {summary.exceedsPreference ? (
          <span className="inline-flex items-center rounded-full border border-amber-300 bg-amber-100 px-2.5 py-0.5 text-xs font-medium text-amber-900">
            Above your walking preference
          </span>
        ) : null}
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
