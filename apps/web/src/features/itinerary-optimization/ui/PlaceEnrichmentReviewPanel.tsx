"use client";

import { useMemo, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { AttachPlaceDialog } from "@/features/place-attachment";
import { Button } from "@/shared/ui/button";
import { isItineraryConflictError } from "@/shared/api/client";
import { tripKeys, updateTripItinerary } from "@/lib/api/trips";
import {
  getPlaceMatchReviewItems,
  getPlaceMatchReviewSummary,
  removeItemPlaceFromReview,
  replaceItemPlaceFromReview,
  updateItemPlaceReviewStatus,
  type PlaceMatchReviewItem
} from "@/entities/itinerary/model/place-enrichment-review-utils";
import type { Place } from "@/entities/place/model";
import type { Itinerary, Trip } from "@/entities/trip/model";

type PlaceEnrichmentReviewPanelProps = {
  trip: Trip;
  readOnly?: boolean;
  onTripUpdated?: (trip: Trip) => void | Promise<void>;
};

export function PlaceEnrichmentReviewPanel({
  trip,
  readOnly = false,
  onTripUpdated
}: PlaceEnrichmentReviewPanelProps) {
  const queryClient = useQueryClient();
  const [error, setError] = useState<string | null>(null);
  const [savingItemId, setSavingItemId] = useState<string | null>(null);
  const [attachTarget, setAttachTarget] = useState<PlaceMatchReviewItem | null>(null);

  const itinerary = trip.itinerary;
  const reviewItems = useMemo(
    () => (itinerary ? getPlaceMatchReviewItems(itinerary) : []),
    [itinerary]
  );
  const summary = useMemo(
    () => (itinerary ? getPlaceMatchReviewSummary(itinerary) : null),
    [itinerary]
  );

  const updateMutation = useMutation({
    mutationFn: (updatedItinerary: Itinerary) =>
      updateTripItinerary(trip.id, updatedItinerary, trip.itineraryRevision)
  });

  if (!itinerary || reviewItems.length === 0 || !summary) {
    return null;
  }

  const isSaving = updateMutation.isPending;

  async function saveReviewChange(updatedItinerary: Itinerary, itemId: string) {
    try {
      setError(null);
      setSavingItemId(itemId);
      const updatedTrip = await updateMutation.mutateAsync(updatedItinerary);
      queryClient.setQueryData(tripKeys.detail(trip.id), updatedTrip);
      await queryClient.invalidateQueries({ queryKey: tripKeys.itineraryVersions(trip.id) });

      if (onTripUpdated) {
        await onTripUpdated(updatedTrip);
      } else {
        await queryClient.invalidateQueries({ queryKey: tripKeys.detail(trip.id) });
      }
    } catch (saveError) {
      if (isItineraryConflictError(saveError)) {
        setError("This itinerary changed. Reload latest version before trying again.");
        await queryClient.invalidateQueries({ queryKey: tripKeys.detail(trip.id) });
      } else {
        setError(saveError instanceof Error ? saveError.message : "Could not save place review.");
      }
    } finally {
      setSavingItemId(null);
    }
  }

  async function acceptMatch(item: PlaceMatchReviewItem) {
    if (!trip.itinerary) {
      return;
    }

    await saveReviewChange(
      updateItemPlaceReviewStatus(trip.itinerary, item.dayIndex, item.itemIndex, "accepted"),
      item.id
    );
  }

  async function removeMatch(item: PlaceMatchReviewItem) {
    if (!trip.itinerary) {
      return;
    }
    if (!window.confirm("Remove attached place from this itinerary item?")) {
      return;
    }

    await saveReviewChange(
      removeItemPlaceFromReview(trip.itinerary, item.dayIndex, item.itemIndex),
      item.id
    );
  }

  async function selectReplacementPlace(place: Place) {
    if (!trip.itinerary || !attachTarget) {
      return;
    }

    await saveReviewChange(
      replaceItemPlaceFromReview(
        trip.itinerary,
        attachTarget.dayIndex,
        attachTarget.itemIndex,
        place
      ),
      attachTarget.id
    );
  }

  return (
    <section className="rounded-lg border border-slate-200 bg-white p-6" id="place-matches">
      <AttachPlaceDialog
        destination={trip.destination}
        initialQuery={attachTarget?.query || attachTarget?.itemName || ""}
        onClose={() => setAttachTarget(null)}
        onSelect={selectReplacementPlace}
        open={attachTarget != null}
      />

      <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h2 className="text-xl font-semibold text-slate-950">Place Matches</h2>
        </div>
        <div className="grid grid-cols-3 gap-2 text-center text-xs sm:grid-cols-6">
          <SummaryPill label="Matched" value={summary.matched} />
          <SummaryPill label="No match" value={summary.noMatch} />
          <SummaryPill label="Pending" value={summary.pending} />
          <SummaryPill label="Accepted" value={summary.accepted} />
          <SummaryPill label="Changed" value={summary.changed} />
          <SummaryPill label="Removed" value={summary.removed} />
        </div>
      </div>

      {error ? (
        <div className="mt-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-800">
          {error}
        </div>
      ) : null}

      <div className="mt-5 divide-y divide-slate-100">
        {reviewItems.map((item) => (
          <ReviewRow
            disabled={isSaving}
            isSaving={savingItemId === item.id}
            item={item}
            key={item.id}
            onAccept={() => acceptMatch(item)}
            onChange={() => setAttachTarget(item)}
            onRemove={() => removeMatch(item)}
            readOnly={readOnly}
          />
        ))}
      </div>
    </section>
  );
}

type SummaryPillProps = {
  label: string;
  value: number;
};

function SummaryPill({ label, value }: SummaryPillProps) {
  return (
    <div className="rounded-md border border-slate-200 bg-slate-50 px-2 py-2">
      <p className="font-semibold text-slate-950">{value}</p>
      <p className="mt-0.5 font-medium text-slate-500">{label}</p>
    </div>
  );
}

type ReviewRowProps = {
  item: PlaceMatchReviewItem;
  disabled: boolean;
  isSaving: boolean;
  onAccept: () => void;
  onChange: () => void;
  onRemove: () => void;
  readOnly: boolean;
};

function ReviewRow({
  item,
  disabled,
  isSaving,
  onAccept,
  onChange,
  onRemove,
  readOnly
}: ReviewRowProps) {
  const hasPlace = Boolean(item.placeName);
  const canAccept = hasPlace && item.status === "matched" && item.reviewStatus !== "accepted";
  const canRemove = hasPlace && item.reviewStatus !== "removed";
  const missingPlaceLabel =
    item.reviewStatus === "removed"
      ? "Attached place removed"
      : item.status === "failed"
        ? "Place match failed"
        : "No confident match found";

  return (
    <div className="grid gap-4 py-4 first:pt-0 last:pb-0 lg:grid-cols-[minmax(0,1fr)_auto]">
      <div className="min-w-0">
        <div className="flex flex-wrap items-center gap-2 text-sm">
          <span className="font-semibold text-slate-900">
            Day {item.dayNumber} · {item.time || "Time TBD"}
          </span>
          <span className="rounded-full bg-primary-50 px-2.5 py-1 text-xs font-medium text-primary-700">
            {formatCategory(item.itemType || "item")}
          </span>
          <StatusBadge status={item.status} />
          <ReviewStatusBadge status={item.reviewStatus} />
        </div>

        <p className="mt-2 font-semibold text-slate-950">AI item: {item.itemName}</p>

        {hasPlace ? (
          <div className="mt-2 space-y-1 text-sm text-slate-600">
            <p>
              <span className="font-medium text-slate-700">Matched place:</span>{" "}
              {item.placeName}
            </p>
            {item.placeAddress ? <p>{item.placeAddress}</p> : null}
            <p className="text-xs font-medium text-slate-500">
              Provider: {formatCategory(item.provider || "unknown")}
              {formatConfidence(item.confidence)}
            </p>
          </div>
        ) : (
          <div className="mt-2 space-y-1 text-sm text-slate-600">
            <p>{missingPlaceLabel}</p>
            {item.query ? <p className="text-xs text-slate-500">Query: {item.query}</p> : null}
          </div>
        )}
      </div>

      {readOnly ? null : (
        <div className="flex flex-wrap items-start gap-2 lg:justify-end">
          {hasPlace ? (
            <>
              <Button
                disabled={disabled || isSaving || !canAccept}
                onClick={onAccept}
                size="sm"
                type="button"
              >
                {isSaving ? "Saving..." : "Accept"}
              </Button>
              <Button
                disabled={disabled || isSaving}
                onClick={onChange}
                size="sm"
                type="button"
                variant="secondary"
              >
                Change
              </Button>
              <Button
                disabled={disabled || isSaving || !canRemove}
                onClick={onRemove}
                size="sm"
                type="button"
                variant="ghost"
              >
                Remove
              </Button>
            </>
          ) : (
            <Button
              disabled={disabled || isSaving}
              onClick={onChange}
              size="sm"
              type="button"
              variant="secondary"
            >
              {isSaving ? "Saving..." : "Search manually"}
            </Button>
          )}
        </div>
      )}
    </div>
  );
}

function StatusBadge({ status }: { status: PlaceMatchReviewItem["status"] }) {
  const label =
    status === "matched"
      ? "Matched"
      : status === "no_match"
        ? "No confident match"
        : status === "failed"
          ? "Failed"
          : "Skipped";
  const className =
    status === "matched"
      ? "bg-emerald-50 text-emerald-700"
      : status === "no_match"
        ? "bg-amber-50 text-amber-800"
        : "bg-slate-100 text-slate-600";

  return (
    <span className={`rounded-full px-2.5 py-1 text-xs font-medium ${className}`}>
      {label}
    </span>
  );
}

function ReviewStatusBadge({ status }: { status: PlaceMatchReviewItem["reviewStatus"] }) {
  const label = status.charAt(0).toUpperCase() + status.slice(1);
  const className =
    status === "accepted"
      ? "bg-emerald-50 text-emerald-700"
      : status === "changed"
        ? "bg-primary-50 text-primary-700"
        : status === "removed"
          ? "bg-red-50 text-red-700"
          : "bg-slate-100 text-slate-600";

  return (
    <span className={`rounded-full px-2.5 py-1 text-xs font-medium ${className}`}>
      {label}
    </span>
  );
}

function formatCategory(value: string) {
  return value
    .split(/[_\s-]+/)
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

function formatConfidence(value: number | null | undefined) {
  if (value == null || Number.isNaN(value)) {
    return "";
  }
  return ` · Confidence: ${Math.round(value * 100)}%`;
}
