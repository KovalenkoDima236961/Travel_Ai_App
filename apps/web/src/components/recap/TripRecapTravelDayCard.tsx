"use client";

import { TripRecapStatusCard } from "./TripRecapStatusCard";

export function TripRecapTravelDayCard({ tripId, mode }: { tripId: string; mode: string }) {
  return mode === "post_trip" ? <TripRecapStatusCard tripId={tripId}/> : null;
}
