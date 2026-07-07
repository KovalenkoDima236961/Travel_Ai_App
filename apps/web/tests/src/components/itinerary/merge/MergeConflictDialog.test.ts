import { createElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";

import { MergeConflictDialog } from "@/components/itinerary/merge/MergeConflictDialog";
import type { ItineraryMergeResult } from "@/entities/itinerary/model/diff-merge/types";

describe("MergeConflictDialog", () => {
  it("renders safe merge copy and action", () => {
    const html = renderToStaticMarkup(
      createElement(MergeConflictDialog, {
        mergeResult: result("safe"),
        latestRevision: 13,
        resolutions: {},
        onApplyMerged: () => {},
        onDiscardLocal: () => {},
        onViewLatest: () => {},
        onCancel: () => {},
        onResolutionChange: () => {}
      })
    );

    expect(html).toContain("This itinerary changed while you were editing");
    expect(html).toContain("You can apply them safely");
    expect(html).toContain("Apply safe merge");
  });

  it("renders partial conflict resolution options", () => {
    const mergeResult = result("partial_conflict");
    const html = renderToStaticMarkup(
      createElement(MergeConflictDialog, {
        mergeResult,
        latestRevision: 13,
        resolutions: { [mergeResult.conflicts[0].conflictKey]: "keep_mine" },
        onApplyMerged: () => {},
        onDiscardLocal: () => {},
        onViewLatest: () => {},
        onCancel: () => {},
        onResolutionChange: () => {}
      })
    );

    expect(html).toContain("Some of your changes overlap");
    expect(html).toContain("Keep latest");
    expect(html).toContain("Keep mine");
    expect(html).toContain("Apply selected resolutions");
  });

  it("renders unsafe conflict copy and secondary actions", () => {
    const html = renderToStaticMarkup(
      createElement(MergeConflictDialog, {
        mergeResult: result("unsafe"),
        latestRevision: 13,
        resolutions: {},
        onApplyMerged: () => {},
        onDiscardLocal: () => {},
        onViewLatest: () => {},
        onCancel: () => {},
        onResolutionChange: () => {}
      })
    );

    expect(html).toContain("Both versions changed the same day or item");
    expect(html).toContain("Discard my changes");
    expect(html).toContain("View latest");
    expect(html).toContain("Cancel");
  });
});

function result(safety: ItineraryMergeResult["safety"]): ItineraryMergeResult {
  const localChange = {
    id: "local:item_modified:1:item-a",
    origin: "local" as const,
    type: "item_modified" as const,
    dayNumber: 1,
    itemKey: "item-a",
    itemIndex: 0,
    before: { time: "09:00", type: "activity", name: "Walk" },
    after: { time: "09:00", type: "activity", name: "Local walk" },
    summary: "Edited Day 1 Walk",
    conflictKey: "day:1:item:item-a"
  };
  const remoteChange = {
    ...localChange,
    id: "remote:item_modified:1:item-a",
    origin: "remote" as const,
    after: { time: "09:00", type: "activity", name: "Remote walk" },
    summary: "Edited Day 1 Walk remotely"
  };
  const conflicts =
    safety === "safe"
      ? []
      : [
          {
            conflictKey: localChange.conflictKey,
            dayNumber: 1,
            itemKey: "item-a",
            localChanges: [localChange],
            remoteChanges: [remoteChange],
            resolution: "keep_latest" as const
          }
        ];

  return {
    safety,
    baseRevision: 12,
    latestRevision: 13,
    localChanges: [localChange],
    remoteChanges: safety === "safe" ? [] : [remoteChange],
    conflicts,
    mergedItinerary: {
      days: [
        {
          day: 1,
          title: "Day 1",
          items: [{ time: "09:00", type: "activity", name: "Local walk" }]
        }
      ]
    },
    summary: {
      localChangeCount: 1,
      remoteChangeCount: safety === "safe" ? 0 : 1,
      conflictCount: conflicts.length,
      safeLocalChangeCount: safety === "safe" ? 1 : 0
    }
  };
}
