"use client";

import dynamic from "next/dynamic";
import { useEffect, useMemo, useState } from "react";
import { Card } from "@/components/ui/Card";
import {
  getAvailableDays,
  getItineraryMapMarkers,
  getMapCenter
} from "@/lib/itinerary/map-utils";
import { cn } from "@/lib/utils";
import type { Itinerary } from "@/types/trip";

const LeafletItineraryMap = dynamic(
  () => import("./ItineraryLeafletMap").then((module) => module.ItineraryLeafletMap),
  {
    loading: () => (
      <div className="flex h-full items-center justify-center bg-slate-50 text-sm text-slate-600">
        Loading map...
      </div>
    ),
    ssr: false
  }
);

type ItineraryMapProps = {
  itinerary: Itinerary;
  startDate?: string | null;
  className?: string;
};

export function ItineraryMap({ itinerary, startDate, className }: ItineraryMapProps) {
  const [selectedDay, setSelectedDay] = useState<number | null>(null);
  const markers = useMemo(() => getItineraryMapMarkers(itinerary), [itinerary]);
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
        : markers.filter((marker) => marker.dayNumber === selectedDay),
    [markers, selectedDay]
  );
  const center = useMemo(() => getMapCenter(filteredMarkers), [filteredMarkers]);
  const currency = itinerary.currency ?? "EUR";

  if (markers.length === 0) {
    return (
      <Card className={className}>
        <h2 className="text-xl font-semibold text-slate-950">Map view</h2>
        <div className="mt-4 rounded-lg border border-dashed border-slate-300 bg-slate-50 p-6">
          <p className="font-semibold text-slate-900">No map locations yet.</p>
          <p className="mt-2 text-sm leading-6 text-slate-600">
            Attach real places to itinerary items in edit mode.
          </p>
        </div>
      </Card>
    );
  }

  return (
    <Card className={className}>
      <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div>
          <h2 className="text-xl font-semibold text-slate-950">Map view</h2>
          <p className="mt-2 text-sm leading-6 text-slate-600">
            Read-only locations from attached itinerary places.
          </p>
        </div>
        <div className="flex flex-wrap gap-2" aria-label="Filter map markers by day">
          <FilterButton
            label="All days"
            onClick={() => setSelectedDay(null)}
            selected={selectedDay == null}
          />
          {availableDays.map((dayNumber) => (
            <FilterButton
              key={dayNumber}
              label={`Day ${dayNumber}`}
              onClick={() => setSelectedDay(dayNumber)}
              selected={selectedDay === dayNumber}
            />
          ))}
        </div>
      </div>

      {filteredMarkers.length === 0 ? (
        <div className="mt-4 rounded-lg border border-dashed border-slate-300 bg-slate-50 p-6 text-sm text-slate-600">
          No mapped places for this day.
        </div>
      ) : (
        <div className="mt-5 h-[420px] overflow-hidden rounded-lg border border-slate-200 bg-slate-100">
          <LeafletItineraryMap
            center={center}
            currency={currency}
            markers={filteredMarkers}
            startDate={startDate}
          />
        </div>
      )}
    </Card>
  );
}

type FilterButtonProps = {
  label: string;
  selected: boolean;
  onClick: () => void;
};

function FilterButton({ label, selected, onClick }: FilterButtonProps) {
  return (
    <button
      aria-pressed={selected}
      className={cn(
        "rounded-md border px-3 py-2 text-sm font-medium transition focus:outline-none focus:ring-2 focus:ring-primary-600 focus:ring-offset-2",
        selected
          ? "border-primary-600 bg-primary-600 text-white"
          : "border-slate-300 bg-white text-slate-700 hover:bg-slate-50"
      )}
      onClick={onClick}
      type="button"
    >
      {label}
    </button>
  );
}
