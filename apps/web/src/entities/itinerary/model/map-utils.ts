import type { TripAccommodation } from "@/entities/accommodation/model";
import type { EstimatedCost } from "@/entities/budget/model";
import type { Place } from "@/entities/place/model";
import type { TripRoute } from "@/entities/route/model";
import type { Itinerary } from "@/entities/trip/model";

export type MapItineraryMarker = {
  id: string;
  kind: "itinerary" | "accommodation" | "route_stop";
  dayNumber: number;
  itemIndex: number;
  time: string;
  itemName: string;
  itemType: string;
  note?: string | null;
  estimatedCost?: EstimatedCost | null;
  place: Place;
  latitude: number;
  longitude: number;
  routeOrder?: number;
};

export type MapRouteLine = {
  id: string;
  from: [number, number];
  to: [number, number];
  label: string;
};

const fallbackMapCenter: [number, number] = [48.1486, 17.1077];

export function isValidCoordinate(latitude: unknown, longitude: unknown) {
  return (
    typeof latitude === "number" &&
    Number.isFinite(latitude) &&
    latitude >= -90 &&
    latitude <= 90 &&
    typeof longitude === "number" &&
    Number.isFinite(longitude) &&
    longitude >= -180 &&
    longitude <= 180
  );
}

export function getItineraryMapMarkers(
  itinerary: Itinerary,
  accommodation?: TripAccommodation | null,
  route?: TripRoute | null
): MapItineraryMarker[] {
  const markers = (itinerary.days ?? []).flatMap((day, dayIndex) => {
    const dayNumber = day.day || dayIndex + 1;

    return (day.items ?? []).flatMap((item, itemIndex) => {
      const place = item.place;
      const latitude = place?.latitude;
      const longitude = place?.longitude;

      if (!place || !isValidCoordinate(latitude, longitude)) {
        return [];
      }

      return [
        {
          id: `day-${dayNumber}-item-${itemIndex}-${place.providerPlaceId}`,
          kind: "itinerary" as const,
          dayNumber,
          itemIndex,
          time: item.time,
          itemName: item.name,
          itemType: item.type,
          note: item.note,
          estimatedCost: item.estimatedCost,
          place,
          latitude: latitude as number,
          longitude: longitude as number
        }
      ];
    });
  });

  const accommodationMarker = getAccommodationMarker(accommodation);
  const routeMarkers = getRouteMapMarkers(route);
  return [
    ...(accommodationMarker ? [accommodationMarker] : []),
    ...routeMarkers,
    ...markers
  ];
}

export function getRouteMapMarkers(route?: TripRoute | null): MapItineraryMarker[] {
  if (!route?.stops?.length) {
    return [];
  }
  return route.stops.flatMap((stop, index) => {
    const latitude = stop.coordinates?.lat;
    const longitude = stop.coordinates?.lng;
    if (!isValidCoordinate(latitude, longitude)) {
      return [];
    }
    const place: Place = {
      provider: "route",
      providerPlaceId: stop.id,
      name: stop.city || stop.destination,
      address: [stop.city || stop.destination, stop.country].filter(Boolean).join(", "),
      latitude,
      longitude,
      category: "route_stop"
    };
    return [{
      id: `route-stop-${stop.id}`,
      kind: "route_stop" as const,
      dayNumber: 0,
      itemIndex: index,
      time: "Stop",
      itemName: stop.city || stop.destination,
      itemType: "route_stop",
      note: stop.notes,
      estimatedCost: null,
      place,
      latitude: latitude as number,
      longitude: longitude as number,
      routeOrder: index + 1
    }];
  });
}

export function getRouteMapLines(route?: TripRoute | null): MapRouteLine[] {
  if (!route?.stops?.length) {
    return [];
  }
  const byId = new Map(route.stops.map((stop) => [stop.id, stop]));
  return (route.legs ?? []).flatMap((leg) => {
    const to = byId.get(leg.toStopId);
    const from =
      leg.fromStopId === "origin"
        ? route.origin
        : byId.get(leg.fromStopId);
    const fromCoordinates =
      leg.fromStopId === "origin" ? route.origin?.coordinates : from?.coordinates;
    const toCoordinates = to?.coordinates;
    if (
      !isValidCoordinate(fromCoordinates?.lat, fromCoordinates?.lng) ||
      !isValidCoordinate(toCoordinates?.lat, toCoordinates?.lng)
    ) {
      return [];
    }
    return [{
      id: `route-line-${leg.id}`,
      from: [fromCoordinates?.lat as number, fromCoordinates?.lng as number],
      to: [toCoordinates?.lat as number, toCoordinates?.lng as number],
      label: `${leg.fromName || "Origin"} to ${leg.toName || to?.destination || "stop"}`
    }];
  });
}

function getAccommodationMarker(
  accommodation?: TripAccommodation | null
): MapItineraryMarker | null {
  const place = accommodation?.place;
  const latitude = place?.latitude;
  const longitude = place?.longitude;

  if (!accommodation || !place || !isValidCoordinate(latitude, longitude)) {
    return null;
  }

  return {
    id: `accommodation-${place.provider}-${place.providerPlaceId}`,
    kind: "accommodation",
    dayNumber: 0,
    itemIndex: -1,
    time: "Stay",
    itemName: accommodation.name,
    itemType: accommodation.type,
    note: accommodation.notes,
    estimatedCost: accommodation.estimatedCost ?? null,
    place,
    latitude: latitude as number,
    longitude: longitude as number
  };
}

export function getMapCenter(markers: MapItineraryMarker[]): [number, number] {
  if (markers.length === 0) {
    return fallbackMapCenter;
  }

  const totals = markers.reduce(
    (currentTotals, marker) => ({
      latitude: currentTotals.latitude + marker.latitude,
      longitude: currentTotals.longitude + marker.longitude
    }),
    { latitude: 0, longitude: 0 }
  );

  return [totals.latitude / markers.length, totals.longitude / markers.length];
}

export function getAvailableDays(markers: MapItineraryMarker[]) {
  return Array.from(
    new Set(
      markers
        .filter((marker) => marker.kind === "itinerary")
        .map((marker) => marker.dayNumber)
    )
  ).sort((leftDay, rightDay) => leftDay - rightDay);
}
