"use client";

import { useEffect } from "react";
import L from "leaflet";
import { MapContainer, Marker, Popup, TileLayer, useMap } from "react-leaflet";
import { formatInterestLabel } from "@/lib/utils";
import { costBadgeLabel } from "@/entities/budget/model";
import type { MapItineraryMarker } from "@/entities/itinerary/model/map-utils";
import {
  formatOpeningHoursForDay,
  getDayOfWeekMondayBased,
  getOpeningStatus,
  getTripItemDate
} from "@/entities/itinerary/model/opening-hours-utils";

type ItineraryLeafletMapProps = {
  markers: MapItineraryMarker[];
  center: [number, number];
  currency: string;
  startDate?: string | null;
};

export function ItineraryLeafletMap({
  markers,
  center,
  currency,
  startDate
}: ItineraryLeafletMapProps) {
  return (
    <MapContainer
      center={center}
      className="h-full w-full"
      scrollWheelZoom={false}
      zoom={markers.length === 1 ? 14 : 13}
    >
      <MapBounds markers={markers} />
      <TileLayer
        attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
        url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
      />
      {markers.map((marker) => (
        <Marker
          icon={createMarkerIcon(
            marker.kind === "accommodation" ? "A" : `${marker.dayNumber}.${marker.itemIndex + 1}`
          )}
          key={marker.id}
          position={[marker.latitude, marker.longitude]}
        >
          <Popup>
            <div className="min-w-52 text-sm text-slate-700">
              <p className="text-xs font-semibold uppercase text-slate-500">
                {marker.kind === "accommodation"
                  ? "Accommodation"
                  : `Day ${marker.dayNumber} - ${marker.time}`}
              </p>
              <h3 className="mt-1 text-base font-semibold text-slate-950">
                {marker.itemName}
              </h3>
              {marker.place.name !== marker.itemName ? (
                <p className="mt-1 font-medium text-slate-800">{marker.place.name}</p>
              ) : null}
              <p className="mt-2 leading-5 text-slate-600">{marker.place.address}</p>
              <div className="mt-3 space-y-1 text-xs font-medium text-slate-500">
                <p>Provider: {formatInterestLabel(marker.place.provider || "unknown")}</p>
                <p>{formatInterestLabel(marker.itemType)}</p>
                {marker.place.category ? (
                  <p>{formatInterestLabel(marker.place.category)}</p>
                ) : null}
                {marker.place.rating != null ? (
                  <p>
                    Rating {marker.place.rating}
                    {marker.place.ratingCount != null
                      ? ` (${marker.place.ratingCount.toLocaleString()})`
                      : ""}
                  </p>
                ) : null}
                {costBadgeLabel(marker.estimatedCost, currency) ? (
                  <p>Estimated cost {costBadgeLabel(marker.estimatedCost, currency)}</p>
                ) : null}
              </div>
              <MapOpeningHoursStatus marker={marker} startDate={startDate} />
              {marker.note ? <p className="mt-3 leading-5 text-slate-600">{marker.note}</p> : null}
              {marker.place.mapUrl ? (
                <a
                  className="mt-3 inline-flex font-semibold text-primary-700 hover:text-primary-600"
                  href={marker.place.mapUrl}
                  rel="noreferrer"
                  target="_blank"
                >
                  Open map
                </a>
              ) : null}
            </div>
          </Popup>
        </Marker>
      ))}
    </MapContainer>
  );
}

function MapOpeningHoursStatus({
  marker,
  startDate
}: {
  marker: MapItineraryMarker;
  startDate?: string | null;
}) {
  if (marker.kind === "accommodation") {
    return null;
  }
  if (!marker.place.openingHours || marker.place.openingHours.length === 0) {
    return null;
  }

  const status = getOpeningStatus({
    startDate,
    dayNumber: marker.dayNumber,
    itemTime: marker.time,
    openingHours: marker.place.openingHours
  });
  const itemDate = startDate ? getTripItemDate(startDate, marker.dayNumber) : null;
  const dayOfWeek = itemDate ? getDayOfWeekMondayBased(itemDate) : null;
  const hours =
    dayOfWeek == null
      ? "Closed or unknown"
      : formatOpeningHoursForDay(marker.place.openingHours, dayOfWeek);

  return (
    <div className="mt-3 space-y-1 text-xs font-medium text-slate-500">
      <p>{status.label}</p>
      <p>Hours: {hours}</p>
    </div>
  );
}

function MapBounds({ markers }: { markers: MapItineraryMarker[] }) {
  const map = useMap();

  useEffect(() => {
    if (markers.length === 0) {
      return;
    }

    if (markers.length === 1) {
      map.setView([markers[0].latitude, markers[0].longitude], 14);
      return;
    }

    const bounds = L.latLngBounds(
      markers.map((marker) => [marker.latitude, marker.longitude])
    );
    map.fitBounds(bounds, { maxZoom: 15, padding: [36, 36] });
  }, [map, markers]);

  return null;
}

function createMarkerIcon(label: string) {
  return L.divIcon({
    className: "",
    html: `<div class="itinerary-map-marker">${label}</div>`,
    iconAnchor: [18, 18],
    iconSize: [36, 36],
    popupAnchor: [0, -18]
  });
}
