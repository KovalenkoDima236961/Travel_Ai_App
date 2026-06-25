"use client";

import { useMemo, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Button } from "@/components/ui/Button";
import { isItineraryConflictError } from "@/lib/api/client";
import { tripKeys, updateTripItinerary } from "@/lib/api/trips";
import {
  applyOptimizedDayToItinerary,
  optimizeDayOrder,
  type OptimizedOrderItem
} from "@/lib/itinerary/route-optimization-utils";
import {
  estimateWalkingMinutes,
  formatDistanceKm,
  formatWalkingTime
} from "@/lib/itinerary/distance-utils";
import type { Itinerary, ItineraryDay, Trip } from "@/types/trip";

type OptimizeDayOrderDialogProps = {
  open: boolean;
  onClose: () => void;
  tripId: string;
  itinerary: Itinerary;
  expectedItineraryRevision: number;
  day: ItineraryDay;
  onApplied?: (updatedTrip: Trip) => void;
};

// Below this threshold the suggestion barely changes the route, so we keep the
// apply button enabled (per spec) but tell the user it will not help much.
const MIN_MEANINGFUL_SAVING_KM = 0.1;

export function OptimizeDayOrderDialog({
  open,
  onClose,
  tripId,
  itinerary,
  expectedItineraryRevision,
  day,
  onApplied
}: OptimizeDayOrderDialogProps) {
  const queryClient = useQueryClient();
  const [error, setError] = useState<string | null>(null);

  const result = useMemo(() => optimizeDayOrder(day), [day]);

  const applyMutation = useMutation({
    mutationFn: (updatedItinerary: Itinerary) =>
      updateTripItinerary(tripId, updatedItinerary, expectedItineraryRevision)
  });

  if (!open) {
    return null;
  }

  const isSaving = applyMutation.isPending;
  const hasMeaningfulSaving = result.savedDistanceKm >= MIN_MEANINGFUL_SAVING_KM;

  async function handleApply() {
    if (!result.canOptimize) {
      return;
    }

    try {
      setError(null);
      const updatedItinerary = applyOptimizedDayToItinerary(
        itinerary,
        result.dayNumber,
        result.optimizedDay
      );
      const updatedTrip = await applyMutation.mutateAsync(updatedItinerary);
      onApplied?.(updatedTrip);
      onClose();
    } catch (applyError) {
      if (isItineraryConflictError(applyError)) {
        setError("This itinerary changed. Reload latest version before trying again.");
        await queryClient.invalidateQueries({ queryKey: tripKeys.detail(tripId) });
      } else {
        setError(
          applyError instanceof Error ? applyError.message : "Could not apply optimized order."
        );
      }
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-slate-950/40 px-4 py-10">
      <div
        aria-modal="true"
        className="w-full max-w-3xl rounded-lg bg-white p-5 shadow-xl"
        role="dialog"
      >
        <div className="flex items-start justify-between gap-4">
          <div>
            <h2 className="text-lg font-semibold text-slate-950">
              Optimize Day {result.dayNumber} order
            </h2>
            <p className="mt-1 text-sm leading-6 text-slate-600">
              This uses approximate straight-line distance between mapped places. Real walking
              routes may differ.
            </p>
          </div>
          <Button disabled={isSaving} onClick={onClose} type="button" variant="ghost">
            Close
          </Button>
        </div>

        {!result.canOptimize ? (
          <div className="mt-5 rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900">
            {result.reason}
          </div>
        ) : (
          <>
            <div className="mt-5 grid gap-4 sm:grid-cols-2">
              <OrderColumn title="Current order" items={result.originalOrder} />
              <OrderColumn title="Suggested order" items={result.optimizedOrder} highlight />
            </div>

            <div className="mt-5 rounded-lg border border-slate-200 bg-slate-50 p-4">
              <dl className="grid gap-x-6 gap-y-2 text-sm sm:grid-cols-3">
                <ComparisonRow
                  label="Original"
                  value={`${formatDistanceKm(result.originalDistanceKm)} · ~${formatWalkingTime(
                    estimateWalkingMinutes(result.originalDistanceKm)
                  )}`}
                />
                <ComparisonRow
                  label="Optimized"
                  value={`${formatDistanceKm(result.optimizedDistanceKm)} · ~${formatWalkingTime(
                    estimateWalkingMinutes(result.optimizedDistanceKm)
                  )}`}
                />
                <ComparisonRow
                  label="Estimated saving"
                  value={`${formatDistanceKm(result.savedDistanceKm)} · ~${formatWalkingTime(
                    result.savedWalkingMinutes
                  )} walking`}
                  emphasize={hasMeaningfulSaving}
                />
              </dl>
              {!hasMeaningfulSaving ? (
                <p className="mt-3 text-sm text-slate-600">
                  This suggestion does not significantly reduce distance. You can still apply it,
                  but the order may stay nearly the same.
                </p>
              ) : null}
              <p className="mt-3 text-xs text-slate-500">
                Optimization keeps the first mapped place fixed and reorders places into the
                existing time slots. Distances are approximate, not real walking routes.
              </p>
            </div>

            {error ? (
              <div className="mt-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-800">
                {error}
              </div>
            ) : null}

            <div className="mt-6 flex flex-col gap-3 sm:flex-row sm:justify-end">
              <Button disabled={isSaving} onClick={onClose} type="button" variant="secondary">
                Cancel
              </Button>
              <Button disabled={isSaving} onClick={handleApply} type="button">
                {isSaving ? "Applying..." : "Apply optimized order"}
              </Button>
            </div>
          </>
        )}
      </div>
    </div>
  );
}

type OrderColumnProps = {
  title: string;
  items: OptimizedOrderItem[];
  highlight?: boolean;
};

function OrderColumn({ title, items, highlight = false }: OrderColumnProps) {
  return (
    <div
      className={
        highlight
          ? "rounded-lg border border-primary-200 bg-primary-50/40 p-4"
          : "rounded-lg border border-slate-200 bg-white p-4"
      }
    >
      <p className="text-sm font-semibold text-slate-950">{title}</p>
      <ol className="mt-3 space-y-2">
        {items.map((item, index) => (
          <li
            key={`${item.originalIndex}-${index}`}
            className="flex items-baseline gap-2 text-sm text-slate-700"
          >
            <span className="w-5 shrink-0 text-right font-medium text-slate-400">{index + 1}.</span>
            {item.time ? (
              <span className="shrink-0 font-medium text-slate-900">{item.time}</span>
            ) : null}
            <span className="min-w-0 flex-1">
              {item.name}
              {!item.hasCoordinates ? (
                <span className="ml-2 text-xs font-medium text-slate-400">No coordinates</span>
              ) : null}
            </span>
          </li>
        ))}
      </ol>
    </div>
  );
}

type ComparisonRowProps = {
  label: string;
  value: string;
  emphasize?: boolean;
};

function ComparisonRow({ label, value, emphasize = false }: ComparisonRowProps) {
  return (
    <div className="flex flex-col">
      <dt className="text-xs font-medium uppercase tracking-wide text-slate-500">{label}</dt>
      <dd
        className={
          emphasize ? "mt-0.5 font-semibold text-emerald-700" : "mt-0.5 font-semibold text-slate-900"
        }
      >
        {value}
      </dd>
    </div>
  );
}
