import {
  estimateWalkingMinutes,
  haversineDistanceKm,
  type Coordinate
} from "@/lib/itinerary/distance-utils";
import { isValidCoordinate } from "@/lib/itinerary/map-utils";
import type { Itinerary, ItineraryDay, ItineraryItem } from "@/types/trip";

/**
 * Route Optimization v1 (frontend-only).
 *
 * Suggests a better visiting order for the mapped places within a single day
 * using a simple nearest-neighbour heuristic over straight-line (Haversine)
 * distance. This is NOT real routing — there are no external routing APIs and no
 * road/transit network is considered.
 *
 * Key behaviours:
 * - The first mapped place stays fixed so the day keeps its starting point.
 * - Only mapped places (items with valid coordinates) are reordered, and only
 *   among the positions they already occupy.
 * - Optimization reorders places into existing time slots: the optimized place
 *   that lands in a given position inherits that position's original time.
 * - Unmapped items (notes, rest, free time) keep their original positions.
 * - All functions are pure: inputs are never mutated; object fields (including
 *   optional/unknown place metadata) are preserved by spreading.
 */

/** Minimum mapped places required before a day can be optimized. */
export const MIN_OPTIMIZABLE_STOPS = 3;

export type OptimizableStop = {
  originalIndex: number;
  time: string;
  name: string;
  latitude: number;
  longitude: number;
};

export type OptimizedOrderItem = {
  originalIndex: number;
  name: string;
  time: string;
  hasCoordinates: boolean;
};

export type DayOptimizationResult = {
  dayNumber: number;
  canOptimize: boolean;
  reason?: string;
  originalDistanceKm: number;
  optimizedDistanceKm: number;
  savedDistanceKm: number;
  savedWalkingMinutes: number;
  originalOrder: OptimizedOrderItem[];
  optimizedOrder: OptimizedOrderItem[];
  optimizedDay: ItineraryDay;
};

const NOT_ENOUGH_STOPS_REASON =
  "At least three mapped places are needed to optimize this day.";

function getDayNumber(day: ItineraryDay): number {
  return day.day || 1;
}

function hasValidCoordinates(item: ItineraryItem): boolean {
  return Boolean(item.place) && isValidCoordinate(item.place?.latitude, item.place?.longitude);
}

function getItemLabel(item: ItineraryItem): string {
  return item.name?.trim() || item.place?.name?.trim() || "Unnamed stop";
}

function toOrderItem(item: ItineraryItem, originalIndex: number): OptimizedOrderItem {
  return {
    originalIndex,
    name: getItemLabel(item),
    time: item.time ?? "",
    hasCoordinates: hasValidCoordinates(item)
  };
}

function toCoordinate(stop: OptimizableStop): Coordinate {
  return { latitude: stop.latitude, longitude: stop.longitude };
}

/**
 * Extract the mapped stops for a day (items with a valid in-range coordinate),
 * preserving their original positions via `originalIndex`. Items without valid
 * coordinates are ignored. This is the single source of truth for "is this
 * position mapped" used throughout optimization.
 */
export function getMappedStopsForDay(day: ItineraryDay): OptimizableStop[] {
  return (day.items ?? []).flatMap((item, index) => {
    if (!hasValidCoordinates(item)) {
      return [];
    }

    return [
      {
        originalIndex: index,
        time: item.time ?? "",
        name: getItemLabel(item),
        latitude: item.place?.latitude as number,
        longitude: item.place?.longitude as number
      }
    ];
  });
}

/**
 * A day can be optimized only when it has at least three mapped places.
 */
export function canOptimizeDay(day: ItineraryDay): { canOptimize: boolean; reason?: string } {
  const mappedStops = getMappedStopsForDay(day);
  if (mappedStops.length < MIN_OPTIMIZABLE_STOPS) {
    return { canOptimize: false, reason: NOT_ENOUGH_STOPS_REASON };
  }
  return { canOptimize: true };
}

/**
 * Order stops with a simple nearest-neighbour heuristic. The first stop is kept
 * fixed as the starting point; each subsequent stop is the nearest unvisited one
 * by straight-line distance. Returns a new array; the input is not mutated.
 */
export function nearestNeighborOrder(stops: OptimizableStop[]): OptimizableStop[] {
  if (stops.length <= 2) {
    return [...stops];
  }

  const remaining = stops.slice(1);
  const ordered: OptimizableStop[] = [stops[0]];
  let current = stops[0];

  while (remaining.length > 0) {
    let nearestIndex = 0;
    let nearestDistance = Number.POSITIVE_INFINITY;

    for (let index = 0; index < remaining.length; index += 1) {
      const distance = haversineDistanceKm(toCoordinate(current), toCoordinate(remaining[index]));
      if (distance < nearestDistance) {
        nearestDistance = distance;
        nearestIndex = index;
      }
    }

    current = remaining.splice(nearestIndex, 1)[0];
    ordered.push(current);
  }

  return ordered;
}

/**
 * Total straight-line distance, in kilometres, walking the stops in the given
 * order. Returns 0 for fewer than two stops.
 */
export function calculateStopOrderDistanceKm(stops: OptimizableStop[]): number {
  let total = 0;
  for (let index = 1; index < stops.length; index += 1) {
    total += haversineDistanceKm(toCoordinate(stops[index - 1]), toCoordinate(stops[index]));
  }
  return total;
}

/**
 * Straight-line distance walking a day's mapped stops in their current itinerary
 * order. Unmapped/invalid-coordinate items are ignored.
 */
export function calculateDayMappedDistanceKm(day: ItineraryDay): number {
  return calculateStopOrderDistanceKm(getMappedStopsForDay(day));
}

/**
 * Build a nearest-neighbour optimized order for a single day and a comparison of
 * the original vs optimized straight-line distance.
 *
 * Mapped places are reordered into the positions they already occupy; each
 * optimized place inherits the time of the position it lands in. Unmapped items
 * stay where they are. The original day/itinerary is never mutated.
 */
export function optimizeDayOrder(day: ItineraryDay): DayOptimizationResult {
  const dayNumber = getDayNumber(day);
  const items = day.items ?? [];
  const originalOrder = items.map((item, index) => toOrderItem(item, index));

  const mappedStops = getMappedStopsForDay(day);
  const originalDistanceKm = calculateStopOrderDistanceKm(mappedStops);

  // Not enough mapped places: echo the original day so callers (e.g. the
  // preview dialog) can still render without crashing.
  if (mappedStops.length < MIN_OPTIMIZABLE_STOPS) {
    return {
      dayNumber,
      canOptimize: false,
      reason: NOT_ENOUGH_STOPS_REASON,
      originalDistanceKm,
      optimizedDistanceKm: originalDistanceKm,
      savedDistanceKm: 0,
      savedWalkingMinutes: 0,
      originalOrder,
      optimizedOrder: originalOrder.map((orderItem) => ({ ...orderItem })),
      optimizedDay: { ...day, items: items.map((item) => ({ ...item })) }
    };
  }

  const optimizedStops = nearestNeighborOrder(mappedStops);
  const optimizedDistanceKm = calculateStopOrderDistanceKm(optimizedStops);

  // Positions that currently hold a mapped place. We refill those positions in
  // order from the optimized sequence, keeping each position's original time.
  const mappedPositions = new Set(mappedStops.map((stop) => stop.originalIndex));
  const optimizedSourceIndices = optimizedStops.map((stop) => stop.originalIndex);

  const optimizedItems: ItineraryItem[] = [];
  const optimizedOrder: OptimizedOrderItem[] = [];
  let pointer = 0;

  items.forEach((item, index) => {
    if (!mappedPositions.has(index)) {
      optimizedItems.push({ ...item });
      optimizedOrder.push(toOrderItem(item, index));
      return;
    }

    const sourceIndex = optimizedSourceIndices[pointer];
    pointer += 1;
    const sourceItem = items[sourceIndex];

    // Spread keeps place metadata and any unknown/future fields; the time stays
    // tied to the position, not the place.
    const placedItem: ItineraryItem = { ...sourceItem, time: item.time };
    optimizedItems.push(placedItem);
    optimizedOrder.push({
      originalIndex: sourceIndex,
      name: getItemLabel(sourceItem),
      time: item.time ?? "",
      hasCoordinates: true
    });
  });

  const savedDistanceKm = Math.max(0, originalDistanceKm - optimizedDistanceKm);

  return {
    dayNumber,
    canOptimize: true,
    originalDistanceKm,
    optimizedDistanceKm,
    savedDistanceKm,
    savedWalkingMinutes: estimateWalkingMinutes(savedDistanceKm),
    originalOrder,
    optimizedOrder,
    optimizedDay: { ...day, items: optimizedItems }
  };
}

/**
 * Replace a single day in the itinerary with an optimized day, matching by day
 * number. Other days are returned untouched (same references), and the original
 * itinerary is not mutated.
 */
export function applyOptimizedDayToItinerary(
  itinerary: Itinerary,
  dayNumber: number,
  optimizedDay: ItineraryDay
): Itinerary {
  return {
    ...itinerary,
    days: (itinerary.days ?? []).map((day, index) => {
      const currentDayNumber = day.day || index + 1;
      if (currentDayNumber !== dayNumber) {
        return day;
      }
      // Keep the original day's identity (number/title); only items are reordered.
      return { ...optimizedDay, day: day.day, title: day.title };
    })
  };
}
