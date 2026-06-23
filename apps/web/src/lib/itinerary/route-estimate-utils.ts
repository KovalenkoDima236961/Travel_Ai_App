import { isValidCoordinate } from "@/lib/itinerary/map-utils";
import type { RouteStop } from "@/types/route";
import type { Itinerary, ItineraryDay } from "@/types/trip";

export type DayRouteStops = {
  dayNumber: number;
  stops: RouteStop[];
};

// A day needs at least two mapped stops before a route estimate is meaningful.
export const MIN_ROUTE_STOPS = 2;

/**
 * Ordered route stops for a single day, built only from items that carry valid
 * place coordinates. Item order is preserved. The stop name prefers the
 * attached place name and falls back to the item name.
 */
export function getRouteStopsForDay(day: ItineraryDay): RouteStop[] {
  return (day.items ?? []).flatMap((item) => {
    const place = item.place;
    if (!place || !isValidCoordinate(place.latitude, place.longitude)) {
      return [];
    }

    const name = place.name?.trim() || item.name?.trim() || "Unnamed stop";

    return [
      {
        name,
        latitude: place.latitude as number,
        longitude: place.longitude as number
      }
    ];
  });
}

/**
 * Route stops grouped by day, including only days that have at least two mapped
 * stops. Day numbers mirror the rest of the app (day.day, falling back to the
 * 1-based index).
 */
export function getRouteStopsByDay(itinerary: Itinerary): DayRouteStops[] {
  return (itinerary.days ?? []).flatMap((day, dayIndex) => {
    const stops = getRouteStopsForDay(day);
    if (stops.length < MIN_ROUTE_STOPS) {
      return [];
    }

    return [
      {
        dayNumber: day.day || dayIndex + 1,
        stops
      }
    ];
  });
}

/**
 * Stable query-cache key for a day's stops, derived from the ordered stop names
 * and coordinates. Two itineraries with identical mapped stops produce the same
 * key so estimates are reused instead of refetched on every render.
 */
export function routeStopsCacheKey(stops: RouteStop[]): string {
  return stops
    .map((stop) => `${stop.name}@${stop.latitude.toFixed(5)},${stop.longitude.toFixed(5)}`)
    .join("|");
}
