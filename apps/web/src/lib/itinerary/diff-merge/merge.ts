import { diffItineraries } from "@/lib/itinerary/diff-merge/diff";
import {
  cloneItinerary,
  getItemId,
  itemIdentitySignature
} from "@/lib/itinerary/diff-merge/normalize";
import type {
  ConflictResolutionMap,
  ItineraryChange,
  ItineraryMergeConflict,
  ItineraryMergeResult,
  MergeOptions,
  MergeSafety
} from "@/lib/itinerary/diff-merge/types";
import type { Itinerary, ItineraryDay, ItineraryItem } from "@/types/trip";

export function mergeItineraries(
  baseItinerary: Itinerary,
  draftItinerary: Itinerary,
  latestItinerary: Itinerary,
  options: MergeOptions
): ItineraryMergeResult {
  const localChanges = diffItineraries(baseItinerary, draftItinerary, "local");
  const remoteChanges = diffItineraries(baseItinerary, latestItinerary, "remote");
  const conflictMap = new Map<string, ItineraryMergeConflict>();
  const safeLocalChanges: ItineraryChange[] = [];

  localChanges.forEach((localChange) => {
    const overlappingRemoteChanges = remoteChanges.filter((remoteChange) =>
      changesConflict(localChange, remoteChange)
    );

    if (overlappingRemoteChanges.length === 0) {
      safeLocalChanges.push(localChange);
      return;
    }

    overlappingRemoteChanges.forEach((remoteChange) => {
      const conflictKey = conflictKeyForPair(localChange, remoteChange);
      const existing = conflictMap.get(conflictKey) ?? {
        conflictKey,
        dayNumber: localChange.dayNumber,
        itemKey: localChange.itemKey ?? remoteChange.itemKey ?? null,
        localChanges: [],
        remoteChanges: [],
        resolution: "keep_latest" as const
      };
      addUniqueChange(existing.localChanges, localChange);
      addUniqueChange(existing.remoteChanges, remoteChange);
      conflictMap.set(conflictKey, existing);
    });
  });

  const conflicts = Array.from(conflictMap.values());
  const mergedItinerary = applyChanges(latestItinerary, safeLocalChanges);
  const safety = determineSafety(conflicts);

  return {
    safety,
    baseRevision: options.baseRevision,
    latestRevision: options.latestRevision,
    localChanges,
    remoteChanges,
    conflicts,
    mergedItinerary,
    summary: {
      localChangeCount: localChanges.length,
      remoteChangeCount: remoteChanges.length,
      conflictCount: conflicts.length,
      safeLocalChangeCount: safeLocalChanges.length
    }
  };
}

export function applyConflictResolutions(
  latestItinerary: Itinerary,
  mergeResult: ItineraryMergeResult,
  resolutions: ConflictResolutionMap
): Itinerary {
  const resolvedItinerary = cloneItinerary(mergeResult.mergedItinerary ?? latestItinerary);
  const keepMineChanges = mergeResult.conflicts.flatMap((conflict) => {
    const resolution = resolutions[conflict.conflictKey] ?? conflict.resolution ?? "keep_latest";
    return resolution === "keep_mine" ? conflict.localChanges : [];
  });

  return applyChanges(resolvedItinerary, keepMineChanges);
}

function applyChanges(itinerary: Itinerary, changes: ItineraryChange[]): Itinerary {
  let next = cloneItinerary(itinerary);
  changes.forEach((change) => {
    next = applyChange(next, change);
  });
  return next;
}

function applyChange(itinerary: Itinerary, change: ItineraryChange): Itinerary {
  switch (change.type) {
    case "day_added":
    case "day_replaced":
      return upsertDay(itinerary, change.dayNumber, change.after);
    case "day_removed":
      return normalizeDayNumbers({
        ...itinerary,
        days: (itinerary.days ?? []).filter(
          (day, index) => (day.day || index + 1) !== change.dayNumber
        )
      });
    case "item_added":
      return insertItem(itinerary, change);
    case "item_removed":
      return removeItem(itinerary, change);
    case "item_modified":
      return replaceItem(itinerary, change);
    case "item_moved":
      return insertItem(removeItem(itinerary, change), change);
    case "item_reordered":
      return replaceDayItems(itinerary, change.dayNumber, change.after);
    default:
      return itinerary;
  }
}

function changesConflict(localChange: ItineraryChange, remoteChange: ItineraryChange): boolean {
  if (localChange.dayNumber !== remoteChange.dayNumber) {
    return false;
  }
  if (isBroadDayChange(localChange) || isBroadDayChange(remoteChange)) {
    return true;
  }
  if (isOrderChange(localChange) || isOrderChange(remoteChange)) {
    return true;
  }
  return localChange.conflictKey === remoteChange.conflictKey;
}

function conflictKeyForPair(localChange: ItineraryChange, remoteChange: ItineraryChange) {
  if (
    localChange.conflictKey === remoteChange.conflictKey &&
    localChange.conflictKey.includes(":item:")
  ) {
    return localChange.conflictKey;
  }
  return `day:${localChange.dayNumber}`;
}

function determineSafety(conflicts: ItineraryMergeConflict[]): MergeSafety {
  if (conflicts.length === 0) {
    return "safe";
  }
  if (
    conflicts.some((conflict) =>
      [...conflict.localChanges, ...conflict.remoteChanges].some(
        (change) => isBroadDayChange(change) || isOrderChange(change)
      )
    )
  ) {
    return "unsafe";
  }
  return "partial_conflict";
}

function isBroadDayChange(change: ItineraryChange) {
  return change.type === "day_added" || change.type === "day_removed" || change.type === "day_replaced";
}

function isOrderChange(change: ItineraryChange) {
  return change.type === "item_reordered" || change.type === "item_moved";
}

function addUniqueChange(changes: ItineraryChange[], nextChange: ItineraryChange) {
  if (!changes.some((change) => change.id === nextChange.id)) {
    changes.push(nextChange);
  }
}

function upsertDay(itinerary: Itinerary, dayNumber: number, dayValue: unknown): Itinerary {
  if (!isItineraryDay(dayValue)) {
    return itinerary;
  }
  const day = cloneDay(dayValue);
  const days = [...(itinerary.days ?? [])];
  const existingIndex = findDayIndex(days, dayNumber);

  if (existingIndex >= 0) {
    days[existingIndex] = { ...day, day: dayNumber };
  } else {
    days.splice(Math.max(0, Math.min(dayNumber - 1, days.length)), 0, {
      ...day,
      day: dayNumber
    });
  }

  return normalizeDayNumbers({ ...itinerary, days });
}

function insertItem(itinerary: Itinerary, change: ItineraryChange): Itinerary {
  if (!isItineraryItem(change.after)) {
    return itinerary;
  }
  const days = [...(itinerary.days ?? [])];
  const dayIndex = findDayIndex(days, change.dayNumber);
  if (dayIndex < 0) {
    return itinerary;
  }
  const day = days[dayIndex];
  const items = [...(day.items ?? [])];
  const insertIndex = Math.max(0, Math.min(change.itemIndex ?? items.length, items.length));
  items.splice(insertIndex, 0, cloneItem(change.after));
  days[dayIndex] = { ...day, items };
  return { ...itinerary, days };
}

function removeItem(itinerary: Itinerary, change: ItineraryChange): Itinerary {
  const days = [...(itinerary.days ?? [])];
  const dayIndex = findDayIndex(days, change.dayNumber);
  if (dayIndex < 0) {
    return itinerary;
  }
  const day = days[dayIndex];
  const itemIndex = findItemIndex(day, change);
  if (itemIndex < 0) {
    return itinerary;
  }
  days[dayIndex] = {
    ...day,
    items: day.items.filter((_, index) => index !== itemIndex)
  };
  return { ...itinerary, days };
}

function replaceItem(itinerary: Itinerary, change: ItineraryChange): Itinerary {
  if (!isItineraryItem(change.after)) {
    return itinerary;
  }
  const replacement = change.after;
  const days = [...(itinerary.days ?? [])];
  const dayIndex = findDayIndex(days, change.dayNumber);
  if (dayIndex < 0) {
    return itinerary;
  }
  const day = days[dayIndex];
  const itemIndex = findItemIndex(day, change);
  if (itemIndex < 0) {
    return itinerary;
  }
  days[dayIndex] = {
    ...day,
    items: day.items.map((item, index) => (index === itemIndex ? cloneItem(replacement) : item))
  };
  return { ...itinerary, days };
}

function replaceDayItems(itinerary: Itinerary, dayNumber: number, itemValue: unknown): Itinerary {
  if (!Array.isArray(itemValue) || !itemValue.every(isItineraryItem)) {
    return itinerary;
  }
  const days = [...(itinerary.days ?? [])];
  const dayIndex = findDayIndex(days, dayNumber);
  if (dayIndex < 0) {
    return itinerary;
  }
  days[dayIndex] = {
    ...days[dayIndex],
    items: itemValue.map(cloneItem)
  };
  return { ...itinerary, days };
}

function findDayIndex(days: ItineraryDay[], dayNumber: number) {
  return days.findIndex((day, index) => (day.day || index + 1) === dayNumber);
}

function findItemIndex(day: ItineraryDay, change: ItineraryChange) {
  const items = day.items ?? [];
  const id = itemIdFromChange(change);
  if (id) {
    const index = items.findIndex((item) => getItemId(item) === id);
    if (index >= 0) {
      return index;
    }
  }

  const signature = itemSignatureFromChange(change);
  if (signature) {
    const index = items.findIndex((item) => itemIdentitySignature(item) === signature);
    if (index >= 0) {
      return index;
    }
  }

  if (change.itemIndex != null && change.itemIndex >= 0 && change.itemIndex < items.length) {
    return change.itemIndex;
  }
  return -1;
}

function itemIdFromChange(change: ItineraryChange) {
  if (change.itemKey?.startsWith("id:")) {
    return change.itemKey.slice(3);
  }
  if (isItineraryItem(change.before)) {
    return getItemId(change.before);
  }
  if (isItineraryItem(change.after)) {
    return getItemId(change.after);
  }
  return null;
}

function itemSignatureFromChange(change: ItineraryChange) {
  if (isItineraryItem(change.before)) {
    return itemIdentitySignature(change.before);
  }
  if (isItineraryItem(change.after)) {
    return itemIdentitySignature(change.after);
  }
  return null;
}

function normalizeDayNumbers(itinerary: Itinerary): Itinerary {
  return {
    ...itinerary,
    days: (itinerary.days ?? []).map((day, index) => ({ ...day, day: index + 1 }))
  };
}

function isItineraryDay(value: unknown): value is ItineraryDay {
  return Boolean(value) && typeof value === "object" && Array.isArray((value as ItineraryDay).items);
}

function isItineraryItem(value: unknown): value is ItineraryItem {
  return Boolean(value) && typeof value === "object" && typeof (value as ItineraryItem).name === "string";
}

function cloneDay(day: ItineraryDay): ItineraryDay {
  return JSON.parse(JSON.stringify(day)) as ItineraryDay;
}

function cloneItem(item: ItineraryItem): ItineraryItem {
  return JSON.parse(JSON.stringify(item)) as ItineraryItem;
}
