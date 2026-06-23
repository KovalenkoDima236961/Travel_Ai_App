import type { Place } from "@/types/place";
import type { Itinerary } from "@/types/trip";

export type MapItineraryMarker = {
  id: string;
  dayNumber: number;
  itemIndex: number;
  time: string;
  itemName: string;
  itemType: string;
  note?: string | null;
  estimatedCost?: number | null;
  place: Place;
  latitude: number;
  longitude: number;
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

export function getItineraryMapMarkers(itinerary: Itinerary): MapItineraryMarker[] {
  return (itinerary.days ?? []).flatMap((day, dayIndex) => {
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
  return Array.from(new Set(markers.map((marker) => marker.dayNumber))).sort(
    (leftDay, rightDay) => leftDay - rightDay
  );
}
