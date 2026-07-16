import { RouteMetricsCard, RouteTimeline } from "@/components/route-builder";
import type { TripRoute } from "@/entities/route/model";
import type { Itinerary } from "@/entities/trip/model";

type RouteSummaryCardProps = {
  route: TripRoute | null | undefined;
  itinerary?: Itinerary | null;
  totalDays?: number;
  currency?: string;
  title?: string;
  tripId?: string;
  travelers?: number;
  canEditTransport?: boolean;
  expectedItineraryRevision?: number;
  online?: boolean;
};

export function RouteSummaryCard({
  route,
  itinerary,
  totalDays,
  currency = "EUR",
  title = "Route overview",
  tripId,
  travelers = 1,
  canEditTransport = false,
  expectedItineraryRevision,
  online = true
}: RouteSummaryCardProps) {
  if (!route || route.stops.length === 0) {
    return null;
  }
  const routeDays = totalDays ?? Math.max(1, itinerary?.days.length ?? route.stops.reduce((sum, stop) => sum + (stop.nights ?? 0), 0));
  return (
    <div className="w-full space-y-3">
      <section className="rounded-[20px] border border-sand-300 bg-white p-5 shadow-[0_1px_2px_rgba(34,26,20,0.04)]">
        <div className="mb-5">
          <p className="text-[13px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">{title}</p>
          <h2 className="mt-1 font-newsreader text-[24px] font-semibold text-cocoa-900">
            {route.stops.map((stop) => stop.city || stop.destination).join(" → ")}
          </h2>
        </div>
        <RouteTimeline
          canEditTransport={canEditTransport}
          currency={currency}
          expectedItineraryRevision={expectedItineraryRevision}
          itinerary={itinerary}
          online={online}
          route={route}
          travelers={travelers}
          tripId={tripId}
        />
        <p className="mt-4 text-[12px] text-cocoa-400">
          Not a booking confirmation. Verify schedules and prices before travel.
        </p>
      </section>
      <RouteMetricsCard currency={currency} route={route} totalDays={routeDays} />
    </div>
  );
}
