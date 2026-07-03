"use client";

import { ChangeSummaryList } from "@/components/itinerary/merge/ChangeSummaryList";
import { ConflictResolutionList } from "@/components/itinerary/merge/ConflictResolutionList";
import { MergedItineraryPreview } from "@/components/itinerary/merge/MergedItineraryPreview";
import { Button } from "@/components/ui/Button";
import type {
  ConflictResolution,
  ConflictResolutionMap,
  ItineraryMergeResult
} from "@/lib/itinerary/diff-merge/types";

type MergeConflictDialogProps = {
  mergeResult: ItineraryMergeResult;
  latestRevision: number;
  resolutions: ConflictResolutionMap;
  applying?: boolean;
  error?: string | null;
  onApplyMerged: () => void;
  onDiscardLocal: () => void;
  onViewLatest: () => void;
  onCancel: () => void;
  onResolutionChange: (conflictKey: string, resolution: ConflictResolution) => void;
};

export function MergeConflictDialog({
  mergeResult,
  latestRevision,
  resolutions,
  applying = false,
  error = null,
  onApplyMerged,
  onDiscardLocal,
  onViewLatest,
  onCancel,
  onResolutionChange
}: MergeConflictDialogProps) {
  const affectedDayNumbers = Array.from(
    new Set(
      [...mergeResult.localChanges, ...mergeResult.remoteChanges].map(
        (change) => change.dayNumber
      )
    )
  ).sort((left, right) => left - right);
  const canApply = Boolean(mergeResult.mergedItinerary);
  const applyLabel =
    mergeResult.safety === "safe" ? "Apply safe merge" : "Apply selected resolutions";

  return (
    <div
      aria-modal="true"
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/40 px-4 py-6"
      role="dialog"
    >
      <div className="flex max-h-full w-full max-w-4xl flex-col rounded-lg border border-slate-200 bg-white shadow-xl">
        <div className="border-b border-slate-200 p-6">
          <h2 className="text-lg font-semibold text-slate-950">
            This itinerary changed while you were editing
          </h2>
          <p className="mt-2 text-sm leading-6 text-slate-600">
            {messageForSafety(mergeResult.safety)}
          </p>
          <p className="mt-2 text-xs font-medium text-slate-500">
            Base revision {mergeResult.baseRevision} · Latest revision {latestRevision}
          </p>
        </div>

        <div className="overflow-y-auto p-6">
          {error ? (
            <div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-800">
              {error}
            </div>
          ) : null}

          <div className="grid gap-4 lg:grid-cols-2">
            <ChangeSummaryList changes={mergeResult.localChanges} title="Your changes" />
            <ChangeSummaryList
              changes={mergeResult.remoteChanges}
              emptyLabel="No latest changes detected."
              title="Latest changes from others"
            />
          </div>

          <div className="mt-4 space-y-4">
            <ConflictResolutionList
              conflicts={mergeResult.conflicts}
              onResolutionChange={onResolutionChange}
              resolutions={resolutions}
            />
            {mergeResult.mergedItinerary ? (
              <MergedItineraryPreview
                affectedDayNumbers={affectedDayNumbers}
                itinerary={mergeResult.mergedItinerary}
              />
            ) : null}
          </div>
        </div>

        <div className="border-t border-slate-200 p-4">
          <div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
            <Button disabled={applying} onClick={onCancel} type="button" variant="ghost">
              Cancel
            </Button>
            <Button
              disabled={applying}
              onClick={onViewLatest}
              type="button"
              variant="secondary"
            >
              View latest
            </Button>
            <Button
              disabled={applying}
              onClick={onDiscardLocal}
              type="button"
              variant="secondary"
            >
              Discard my changes
            </Button>
            <Button disabled={applying || !canApply} onClick={onApplyMerged} type="button">
              {applying ? "Applying..." : applyLabel}
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}

function messageForSafety(safety: ItineraryMergeResult["safety"]) {
  if (safety === "safe") {
    return "Your changes do not overlap with the latest changes. You can apply them safely.";
  }
  if (safety === "partial_conflict") {
    return "Some of your changes overlap with newer changes. Review them before applying.";
  }
  return "Both versions changed the same day or item. Choose which changes to keep.";
}
