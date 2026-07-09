"use client";

import dynamic from "next/dynamic";
import { useMemo } from "react";
import { getItineraryMapMarkers, getMapCenter } from "@/entities/itinerary/model/map-utils";
import type { Itinerary } from "@/entities/trip/model";

const LeafletItineraryMap = dynamic(
  () =>
    import("@/components/trips/ItineraryLeafletMap").then(
      (module) => module.ItineraryLeafletMap
    ),
  {
    loading: () => (
      <div className="flex h-full items-center justify-center bg-sand-200 text-[13px] text-cocoa-500">
        Loading map…
      </div>
    ),
    ssr: false
  }
);

type PublicShareMapProps = {
  itinerary: Itinerary;
  startDate?: string | null;
};

/**
 * Slice-local map card for the shared itinerary. Reuses the same marker/center
 * helpers and Leaflet renderer as the trip-detail right rail, slimmed to the
 * mock's compact sidebar tile (no day-filter pills). Renders nothing when no
 * itinerary place has coordinates to map.
 */
export function PublicShareMap({ itinerary, startDate }: PublicShareMapProps) {
  const markers = useMemo(() => getItineraryMapMarkers(itinerary), [itinerary]);
  const center = useMemo(() => getMapCenter(markers), [markers]);

  if (markers.length === 0) {
    return null;
  }

  return (
    <div className="overflow-hidden rounded-[18px] border border-sand-300 bg-white">
      <div className="h-[200px] bg-sand-200">
        <LeafletItineraryMap
          center={center}
          currency={itinerary.currency ?? "EUR"}
          markers={markers}
          startDate={startDate}
        />
      </div>
    </div>
  );
}
