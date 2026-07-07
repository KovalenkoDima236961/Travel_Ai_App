import { isValidCoordinate } from "@/entities/itinerary/model/map-utils";
import type { TripAccommodation } from "@/entities/accommodation/model";
import type { RouteStop } from "@/entities/route/model";
import type { Itinerary, ItineraryDay } from "@/entities/trip/model";

export type DayRouteStops = {
  dayNumber: number;
  stops: RouteStop[];
  usesAccommodationAnchor: boolean;
};

// A day needs at least two mapped stops before a route estimate is meaningful.
export const MIN_ROUTE_STOPS = 2;

/**
 * Ordered route stops for a single day, built only from items that carry valid
 * place coordinates. Item order is preserved. The stop name prefers the
 * attached place name and falls back to the item name.
 */
export function getAccommodationRouteStop(
  accommodation?: TripAccommodation | null
): RouteStop | null {
  const place = accommodation?.place;
  if (!place || !isValidCoordinate(place.latitude, place.longitude)) {
    return null;
  }

  return {
    name: accommodation.name?.trim() || place.name?.trim() || "Accommodation",
    latitude: place.latitude as number,
    longitude: place.longitude as number
  };
}

export function getRouteStopsForDay(
  day: ItineraryDay,
  accommodation?: TripAccommodation | null
): RouteStop[] {
  const itemStops = (day.items ?? []).flatMap((item) => {
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

  if (itemStops.length === 0) {
    return [];
  }

  const accommodationStop = getAccommodationRouteStop(accommodation);
  if (!accommodationStop) {
    return itemStops;
  }

  return [accommodationStop, ...itemStops, accommodationStop];
}

/**
 * Route stops grouped by day, including only days that have at least two mapped
 * stops. Day numbers mirror the rest of the app (day.day, falling back to the
 * 1-based index).
 */
export function getRouteStopsByDay(
  itinerary: Itinerary,
  accommodation?: TripAccommodation | null
): DayRouteStops[] {
  return (itinerary.days ?? []).flatMap((day, dayIndex) => {
    const stops = getRouteStopsForDay(day, accommodation);
    if (stops.length < MIN_ROUTE_STOPS) {
      return [];
    }

    return [
      {
        dayNumber: day.day || dayIndex + 1,
        stops,
        usesAccommodationAnchor: Boolean(getAccommodationRouteStop(accommodation))
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
