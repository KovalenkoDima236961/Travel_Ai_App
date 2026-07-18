"use client";

import dynamic from "next/dynamic";
import { useEffect, useMemo, useState } from "react";
import {
  getAvailableDays,
  getItineraryMapMarkers,
  getMapCenter,
  getRouteMapLines
} from "@/entities/itinerary/model/map-utils";
import type { TripAccommodation } from "@/entities/accommodation/model";
import type { TripRoute } from "@/entities/route/model";
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

type RightRailMapProps = {
  itinerary: Itinerary;
  accommodation?: TripAccommodation | null;
  route?: TripRoute | null;
  startDate?: string | null;
};

/**
 * Warm right-rail map card, forked from the shared ItineraryMap. Reuses the same
 * marker/center helpers and the Leaflet renderer; only the surrounding chrome and
 * day-filter pills are restyled to the Trip Detail mock.
 */
export function RightRailMap({ itinerary, accommodation, route, startDate }: RightRailMapProps) {
  const [selectedDay, setSelectedDay] = useState<number | null>(null);
  const markers = useMemo(
    () => getItineraryMapMarkers(itinerary, accommodation, route),
    [itinerary, accommodation, route]
  );
  const routeLines = useMemo(() => getRouteMapLines(route), [route]);
  const availableDays = useMemo(() => getAvailableDays(markers), [markers]);

  useEffect(() => {
    if (selectedDay != null && !availableDays.includes(selectedDay)) {
      setSelectedDay(null);
    }
  }, [availableDays, selectedDay]);

  const filteredMarkers = useMemo(
    () =>
      selectedDay == null
        ? markers
        : markers.filter(
            (marker) => marker.kind === "accommodation" || marker.dayNumber === selectedDay
          ),
    [markers, selectedDay]
  );
  const center = useMemo(() => getMapCenter(filteredMarkers), [filteredMarkers]);
  const currency = itinerary.currency ?? "EUR";

  if (markers.length === 0) {
    return (
      <div className="rounded-[20px] border border-sand-300 bg-white p-5">
        <h2 className="text-[13px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">
          Map
        </h2>
        <p className="mt-3 text-[13px] leading-[1.5] text-cocoa-400">
          Attach real places to itinerary items to see them on the map.
        </p>
      </div>
    );
  }

  return (
    <div className="overflow-hidden rounded-[20px] border border-sand-300 bg-white shadow-[0_1px_2px_rgba(34,26,20,0.04),0_12px_32px_rgba(34,26,20,0.06)]">
      <div className="h-[300px] bg-sand-200">
        {filteredMarkers.length === 0 ? (
          <div className="flex h-full items-center justify-center px-6 text-center text-[13px] text-cocoa-500">
            No mapped places for this day.
          </div>
        ) : (
          <LeafletItineraryMap
            center={center}
            currency={currency}
            markers={filteredMarkers}
            routeLines={selectedDay == null ? routeLines : []}
            startDate={startDate}
          />
        )}
      </div>
      <div className="flex flex-wrap items-center gap-1.5 px-[18px] py-3.5">
        <FilterPill label="All" selected={selectedDay == null} onClick={() => setSelectedDay(null)} />
        {availableDays.map((dayNumber) => (
          <FilterPill
            key={dayNumber}
            label={`Day ${dayNumber}`}
            selected={selectedDay === dayNumber}
            onClick={() => setSelectedDay(dayNumber)}
          />
        ))}
      </div>
    </div>
  );
}

function FilterPill({
  label,
  selected,
  onClick
}: {
  label: string;
  selected: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      aria-pressed={selected}
      onClick={onClick}
      className={
        selected
          ? "h-[30px] rounded-full bg-cocoa-900 px-3.5 text-[12.5px] font-semibold text-[#F6EDE2]"
          : "h-[30px] rounded-full px-3.5 text-[12.5px] font-medium text-cocoa-500 transition hover:bg-sand-200"
      }
    >
      {label}
    </button>
  );
}
