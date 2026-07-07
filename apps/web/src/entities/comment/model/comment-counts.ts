import type { CommentCount } from "@/entities/comment/model";

// makeCommentItemKey builds the stable key used to look up a per-item comment
// count. The dayNumber convention must match the itinerary's stored `day.day`
// (and the value ItineraryView resolves via `day.day || index + 1`) so comment
// badges line up with the right item.
export function makeCommentItemKey(dayNumber: number, itemIndex: number): string {
  return `day-${dayNumber}-item-${itemIndex}`;
}

// buildCommentCountMap turns the grouped count list into a lookup keyed by
// makeCommentItemKey, for O(1) badge rendering.
export function buildCommentCountMap(counts: CommentCount[]): Record<string, number> {
  const map: Record<string, number> = {};
  for (const count of counts) {
    map[makeCommentItemKey(count.dayNumber, count.itemIndex)] = count.count;
  }
  return map;
}
