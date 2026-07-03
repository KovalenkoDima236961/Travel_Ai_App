import type { ItineraryChange, ItineraryMergeConflict } from "@/lib/itinerary/diff-merge/types";

export function describeItineraryChange(change: Omit<ItineraryChange, "summary">): string {
  const itemName = getItemName(change.after) || getItemName(change.before);

  switch (change.type) {
    case "day_added":
      return `Added Day ${change.dayNumber}`;
    case "day_removed":
      return `Removed Day ${change.dayNumber}${describeDayTitle(change.before)}`;
    case "day_replaced":
      return `Updated Day ${change.dayNumber}${describeDayTitle(change.after)}`;
    case "item_added":
      return `Added ${itemName || "an item"} to Day ${change.dayNumber}`;
    case "item_removed":
      return `Removed ${itemName || "an item"} from Day ${change.dayNumber}`;
    case "item_modified":
      return `Edited Day ${change.dayNumber}${itemName ? ` ${itemName}` : ""}`;
    case "item_moved":
      return `Moved ${itemName || "an item"} on Day ${change.dayNumber}`;
    case "item_reordered":
      return `Reordered Day ${change.dayNumber}`;
    default:
      return `Changed Day ${change.dayNumber}`;
  }
}

export function describeConflict(conflict: ItineraryMergeConflict): string {
  const local = conflict.localChanges[0];
  const remote = conflict.remoteChanges[0];

  if (local?.type === "item_reordered" || remote?.type === "item_reordered") {
    return `Day ${conflict.dayNumber} order changed in both versions.`;
  }
  if (conflict.itemKey) {
    return `Day ${conflict.dayNumber} item changed in both versions.`;
  }
  return `Day ${conflict.dayNumber} changed in both versions.`;
}

export function getItemName(value: unknown): string | null {
  if (!value || typeof value !== "object") {
    return null;
  }
  const name = (value as { name?: unknown }).name;
  return typeof name === "string" && name.trim() ? name.trim() : null;
}

function describeDayTitle(value: unknown): string {
  if (!value || typeof value !== "object") {
    return "";
  }
  const title = (value as { title?: unknown }).title;
  return typeof title === "string" && title.trim() ? ` (${title.trim()})` : "";
}
