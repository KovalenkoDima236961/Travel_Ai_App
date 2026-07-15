import { formatMoney, getCostAmount } from "@/entities/budget/model";
import type { TripRoute, TripRouteLeg } from "@/entities/route/model";
import { RouteLegTransportOptions } from "@/components/transport";
import { transportModeLabel, tripStyleLabel } from "./route-options";

type RouteSummaryCardProps = {
  route: TripRoute | null | undefined;
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

  const totalMinutes = (route.legs ?? []).reduce(
    (sum, leg) => sum + (leg.estimatedDurationMinutes ?? 0),
    0
  );
  const totalCost = (route.legs ?? []).reduce((sum, leg) => {
    const amount = getRouteLegCostAmount(leg);
    return sum + (amount ?? 0);
  }, 0);
  const styles = route.preferences?.tripStyles ?? [];

  return (
    <section className="rounded-[20px] border border-sand-300 bg-white p-5 shadow-[0_1px_2px_rgba(34,26,20,0.04)]">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <p className="text-[13px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">
            {title}
          </p>
          <h2 className="mt-1 font-newsreader text-[24px] font-semibold text-cocoa-900">
            {route.stops.map((stop) => stop.city || stop.destination).join(" to ")}
          </h2>
        </div>
        <div className="text-right text-[13px] font-medium text-cocoa-500">
          {totalMinutes > 0 ? <p>{formatDuration(totalMinutes)} transfers</p> : null}
          {totalCost > 0 ? <p>{formatMoney(totalCost, currency)} estimated transport</p> : null}
        </div>
      </div>

      <ol className="mt-4 flex flex-col gap-3">
        {route.stops.map((stop, index) => (
          <li key={stop.id} className="flex gap-3">
            <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-cocoa-900 text-[12px] font-semibold text-sand-100">
              {index + 1}
            </span>
            <div className="min-w-0">
              <p className="text-[14.5px] font-semibold text-cocoa-900">
                {stop.city || stop.destination}
                {stop.country ? <span className="font-medium text-cocoa-400">, {stop.country}</span> : null}
              </p>
              <p className="text-[13px] text-cocoa-500">
                {stop.nights != null ? `${stop.nights} night${stop.nights === 1 ? "" : "s"}` : "Flexible stay"}
                {stop.arrivalDate ? ` · arrives ${stop.arrivalDate}` : ""}
              </p>
              {index < (route.legs ?? []).length ? (
                <div className="mt-1">
                  <p className="text-[12.5px] font-medium text-clay-deep">
                    {transportModeLabel(route.legs?.[index]?.mode)} transfer
                    {route.legs?.[index]?.estimatedDurationMinutes
                      ? ` | ${formatDuration(route.legs[index].estimatedDurationMinutes ?? 0)}`
                      : ""}
                  </p>
                  {route.legs?.[index] ? (
                    <RouteLegTransportOptions
                      canEdit={canEditTransport}
                      currency={currency}
                      expectedItineraryRevision={expectedItineraryRevision}
                      leg={route.legs[index]}
                      online={online}
                      travelers={travelers}
                      tripId={tripId}
                    />
                  ) : null}
                </div>
              ) : null}
            </div>
          </li>
        ))}
      </ol>

      {styles.length > 0 ? (
        <div className="mt-4 flex flex-wrap gap-2">
          {styles.map((style) => (
            <span
              key={style}
              className="rounded-full bg-sand-200 px-3 py-1 text-[12.5px] font-semibold text-cocoa-500"
            >
              {tripStyleLabel(style)}
            </span>
          ))}
        </div>
      ) : null}
    </section>
  );
}

function formatDuration(minutes: number) {
  if (minutes < 60) {
    return `${minutes} min`;
  }
  const hours = Math.floor(minutes / 60);
  const remainder = minutes % 60;
  return remainder === 0 ? `${hours} hr` : `${hours} hr ${remainder} min`;
}

function getRouteLegCostAmount(leg: TripRouteLeg) {
  if (leg.selectedTransportOption?.estimatedPrice) {
    return leg.selectedTransportOption.estimatedPrice.amount;
  }
  return getCostAmount(leg.estimatedCost);
}
