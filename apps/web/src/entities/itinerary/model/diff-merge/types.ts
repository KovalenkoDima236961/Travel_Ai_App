import type { Itinerary } from "@/entities/trip/model";

export type ItineraryChangeType =
  | "day_added"
  | "day_removed"
  | "day_replaced"
  | "item_added"
  | "item_removed"
  | "item_modified"
  | "item_moved"
  | "item_reordered";

export type ChangeOrigin = "local" | "remote";

export type ItineraryChange = {
  id: string;
  origin: ChangeOrigin;
  type: ItineraryChangeType;
  dayNumber: number;
  itemKey?: string | null;
  itemIndex?: number | null;
  before?: unknown;
  after?: unknown;
  summary: string;
  conflictKey: string;
};

export type MergeSafety = "safe" | "partial_conflict" | "unsafe";

export type ConflictResolution = "keep_latest" | "keep_mine";

export type ItineraryMergeConflict = {
  conflictKey: string;
  dayNumber: number;
  itemKey?: string | null;
  localChanges: ItineraryChange[];
  remoteChanges: ItineraryChange[];
  resolution?: ConflictResolution | null;
};

export type ItineraryMergeResult = {
  safety: MergeSafety;
  baseRevision: number;
  latestRevision: number;
  localChanges: ItineraryChange[];
  remoteChanges: ItineraryChange[];
  conflicts: ItineraryMergeConflict[];
  mergedItinerary?: Itinerary | null;
  summary: {
    localChangeCount: number;
    remoteChangeCount: number;
    conflictCount: number;
    safeLocalChangeCount: number;
  };
};

export type MergeOptions = {
  baseRevision: number;
  latestRevision: number;
};

export type ConflictResolutionMap = Record<string, ConflictResolution>;
