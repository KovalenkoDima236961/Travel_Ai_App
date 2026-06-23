import { isValidCoordinate } from "@/lib/itinerary/map-utils";
import type { Itinerary, ItineraryItem } from "@/types/trip";

export type Coordinate = {
  latitude: number;
  longitude: number;
};

export type DistanceSegment = {
  fromName: string;
  toName: string;
  fromTime?: string;
  toTime?: string;
  distanceKm: number;
  estimatedWalkingMinutes: number;
};

export type DayDistanceSummary = {
  dayNumber: number;
  mappedStops: number;
  segmentCount: number;
  straightLineDistanceKm: number;
  estimatedWalkingMinutes: number;
  exceedsPreference: boolean;
  maxWalkingKmPerDay?: number | null;
  segments: DistanceSegment[];
};

const EARTH_RADIUS_KM = 6371;
const WALKING_SPEED_KM_PER_HOUR = 5;

function toRadians(degrees: number): number {
  return (degrees * Math.PI) / 180;
}

/**
 * Great-circle distance between two coordinates in kilometres.
 *
 * This is a straight-line ("as the crow flies") estimate, not a real walking
 * route distance. Rounding is intentionally left to the formatting helpers.
 */
export function haversineDistanceKm(a: Coordinate, b: Coordinate): number {
  const latDelta = toRadians(b.latitude - a.latitude);
  const lonDelta = toRadians(b.longitude - a.longitude);

  const sinLat = Math.sin(latDelta / 2);
  const sinLon = Math.sin(lonDelta / 2);

  const h =
    sinLat * sinLat +
    Math.cos(toRadians(a.latitude)) * Math.cos(toRadians(b.latitude)) * sinLon * sinLon;

  const centralAngle = 2 * Math.atan2(Math.sqrt(h), Math.sqrt(1 - h));

  return EARTH_RADIUS_KM * centralAngle;
}

/**
 * Rough walking time for a distance, assuming a flat 5 km/h pace. Rounded to
 * the nearest whole minute.
 */
export function estimateWalkingMinutes(distanceKm: number): number {
  if (!Number.isFinite(distanceKm) || distanceKm <= 0) {
    return 0;
  }

  return Math.round((distanceKm / WALKING_SPEED_KM_PER_HOUR) * 60);
}

function getStopCoordinate(item: ItineraryItem): Coordinate | null {
  const place = item.place;
  if (!place || !isValidCoordinate(place.latitude, place.longitude)) {
    return null;
  }

  return {
    latitude: place.latitude as number,
    longitude: place.longitude as number
  };
}

function getStopLabel(item: ItineraryItem): string {
  const itemName = item.name?.trim();
  if (itemName) {
    return itemName;
  }

  return item.place?.name?.trim() || "Unnamed stop";
}

/**
 * Per-day distance summaries built from itinerary items that have valid place
 * coordinates. Returns one entry per day (preserving day numbers), with zeros
 * for days that have fewer than two mapped stops.
 */
export function getDayDistanceSummaries(
  itinerary: Itinerary,
  maxWalkingKmPerDay?: number | null
): DayDistanceSummary[] {
  const hasPreference = typeof maxWalkingKmPerDay === "number" && maxWalkingKmPerDay > 0;

  return (itinerary.days ?? []).map((day, dayIndex) => {
    const dayNumber = day.day || dayIndex + 1;

    const mappedStops = (day.items ?? []).flatMap((item) => {
      const coordinate = getStopCoordinate(item);
      if (!coordinate) {
        return [];
      }
      return [{ item, coordinate }];
    });

    const segments: DistanceSegment[] = [];
    let straightLineDistanceKm = 0;

    for (let index = 1; index < mappedStops.length; index += 1) {
      const previous = mappedStops[index - 1];
      const current = mappedStops[index];
      const distanceKm = haversineDistanceKm(previous.coordinate, current.coordinate);

      straightLineDistanceKm += distanceKm;
      segments.push({
        fromName: getStopLabel(previous.item),
        toName: getStopLabel(current.item),
        fromTime: previous.item.time || undefined,
        toTime: current.item.time || undefined,
        distanceKm,
        estimatedWalkingMinutes: estimateWalkingMinutes(distanceKm)
      });
    }

    return {
      dayNumber,
      mappedStops: mappedStops.length,
      segmentCount: segments.length,
      straightLineDistanceKm,
      // Round once from the day total, never from summed per-segment minutes.
      estimatedWalkingMinutes: estimateWalkingMinutes(straightLineDistanceKm),
      exceedsPreference: hasPreference
        ? straightLineDistanceKm > (maxWalkingKmPerDay as number)
        : false,
      maxWalkingKmPerDay: hasPreference ? maxWalkingKmPerDay : null,
      segments
    };
  });
}

/**
 * Total mapped straight-line distance across all days, in kilometres. Raw,
 * unrounded so callers can format it consistently.
 */
export function getTripDistanceTotal(summaries: DayDistanceSummary[]): number {
  return summaries.reduce((total, summary) => total + summary.straightLineDistanceKm, 0);
}

export function formatDistanceKm(value: number): string {
  const safeValue = Number.isFinite(value) ? Math.max(value, 0) : 0;
  return `${safeValue.toFixed(1)} km`;
}

export function formatWalkingTime(minutes: number): string {
  const safeMinutes = Number.isFinite(minutes) ? Math.max(Math.round(minutes), 0) : 0;
  const hours = Math.floor(safeMinutes / 60);
  const remainingMinutes = safeMinutes % 60;

  if (hours === 0) {
    return `${remainingMinutes} min`;
  }

  if (remainingMinutes === 0) {
    return `${hours}h`;
  }

  return `${hours}h ${remainingMinutes}min`;
}
