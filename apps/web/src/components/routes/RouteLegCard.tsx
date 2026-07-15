"use client";

import type { TransportMode, TripRouteLeg, TripRouteStop } from "@/entities/route/model";
import { RouteLegTransportOptions } from "@/components/transport";
import { TransportModeSelector } from "./TransportModeSelector";
import { transportModeLabel } from "./route-options";

type RouteLegCardProps = {
  leg: TripRouteLeg;
  index: number;
  fromName: string;
  toStop: TripRouteStop;
  onChange: (leg: TripRouteLeg) => void;
  tripId?: string;
  currency?: string;
  travelers?: number;
  canEditTransport?: boolean;
  expectedItineraryRevision?: number;
  online?: boolean;
};

export function RouteLegCard({
  leg,
  index,
  fromName,
  toStop,
  onChange,
  tripId,
  currency = "EUR",
  travelers = 1,
  canEditTransport = false,
  expectedItineraryRevision,
  online = true
}: RouteLegCardProps) {
  return (
    <div className="rounded-[16px] border border-sand-300 bg-[#FFFDFA] p-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <p className="text-[13px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">
            Transfer {index + 1}
          </p>
          <p className="mt-1 text-[15px] font-semibold text-cocoa-900">
            {fromName} to {toStop.destination || "next stop"}
          </p>
        </div>
        <span className="rounded-full bg-sand-200 px-3 py-1 text-[12.5px] font-semibold text-cocoa-500">
          {transportModeLabel(leg.mode)}
        </span>
      </div>
      <div className="mt-4">
        <TransportModeSelector
          value={leg.mode as TransportMode}
          onChange={(mode) => onChange({ ...leg, mode })}
        />
      </div>
      <RouteLegTransportOptions
        canEdit={canEditTransport}
        currency={currency}
        expectedItineraryRevision={expectedItineraryRevision}
        leg={leg}
        online={online}
        travelers={travelers}
        tripId={tripId}
      />
    </div>
  );
}
