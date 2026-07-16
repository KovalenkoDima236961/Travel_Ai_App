import { formatMoney } from "@/entities/budget/model";
import type { TripActivityEvent } from "@/entities/activity/model";
import type { Trip } from "@/entities/trip/model";

export function formatRouteSummary(trip: Trip): string {
  const routeStops = trip.route?.stops ?? [];
  if (routeStops.length > 0) {
    return routeStops.map((stop) => stop.destination || stop.city || "Stop").join(" -> ");
  }
  return trip.destination;
}

export function formatTripDates(trip: Trip): string {
  if (!trip.startDate) {
    return `${trip.days} ${trip.days === 1 ? "day" : "days"}`;
  }
  const start = formatShortDate(trip.startDate);
  if (trip.days <= 1) {
    return start;
  }
  const startDate = new Date(`${trip.startDate}T00:00:00`);
  if (Number.isNaN(startDate.getTime())) {
    return `${start} · ${trip.days} days`;
  }
  const endDate = new Date(startDate);
  endDate.setDate(startDate.getDate() + Math.max(trip.days - 1, 0));
  return `${start} - ${formatShortDate(endDate.toISOString())} · ${trip.days} days`;
}

export function formatShortDate(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return new Intl.DateTimeFormat("en", { month: "short", day: "numeric" }).format(date);
}

export function formatPercent(value: number | null | undefined): string {
  if (value == null || !Number.isFinite(value)) {
    return "n/a";
  }
  return `${Math.round(value)}%`;
}

export function formatCurrencyAmount(amount: number | null | undefined, currency: string): string {
  if (amount == null || !Number.isFinite(amount)) {
    return "n/a";
  }
  return formatMoney(amount, currency);
}

export function formatActivityEvent(event: TripActivityEvent): string {
  const fallback = event.eventType.replaceAll("_", " ");
  const title = event.metadata?.title;
  if (typeof title === "string" && title.trim()) {
    return title;
  }
  return fallback.slice(0, 1).toUpperCase() + fallback.slice(1);
}
