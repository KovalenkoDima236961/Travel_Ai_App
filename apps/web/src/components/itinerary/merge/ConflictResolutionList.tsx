"use client";

import { ChangeSummaryList } from "@/components/itinerary/merge/ChangeSummaryList";
import { describeConflict } from "@/entities/itinerary/model/diff-merge/describe";
import type {
  ConflictResolution,
  ConflictResolutionMap,
  ItineraryMergeConflict
} from "@/entities/itinerary/model/diff-merge/types";

type ConflictResolutionListProps = {
  conflicts: ItineraryMergeConflict[];
  resolutions: ConflictResolutionMap;
  onResolutionChange: (conflictKey: string, resolution: ConflictResolution) => void;
};

export function ConflictResolutionList({
  conflicts,
  resolutions,
  onResolutionChange
}: ConflictResolutionListProps) {
  if (conflicts.length === 0) {
    return null;
  }

  return (
    <section className="space-y-3">
      <h3 className="text-sm font-semibold text-slate-950">Conflicts</h3>
      {conflicts.map((conflict) => {
        const selected = resolutions[conflict.conflictKey] ?? conflict.resolution ?? "keep_latest";
        return (
          <div
            className="rounded-lg border border-amber-200 bg-amber-50 p-4"
            key={conflict.conflictKey}
          >
            <p className="text-sm font-medium text-amber-950">
              {describeConflict(conflict)}
            </p>
            <div className="mt-3 grid gap-3 md:grid-cols-2">
              <ChangeSummaryList
                changes={conflict.localChanges}
                emptyLabel="No local changes."
                title="Your version"
              />
              <ChangeSummaryList
                changes={conflict.remoteChanges}
                emptyLabel="No latest changes."
                title="Latest version"
              />
            </div>
            <div className="mt-4 flex flex-col gap-2 text-sm text-slate-800 sm:flex-row">
              <label className="inline-flex items-center gap-2">
                <input
                  checked={selected === "keep_latest"}
                  name={`resolution-${conflict.conflictKey}`}
                  onChange={() => onResolutionChange(conflict.conflictKey, "keep_latest")}
                  type="radio"
                  value="keep_latest"
                />
                Keep latest
              </label>
              <label className="inline-flex items-center gap-2">
                <input
                  checked={selected === "keep_mine"}
                  name={`resolution-${conflict.conflictKey}`}
                  onChange={() => onResolutionChange(conflict.conflictKey, "keep_mine")}
                  type="radio"
                  value="keep_mine"
                />
                Keep mine
              </label>
            </div>
          </div>
        );
      })}
    </section>
  );
}
