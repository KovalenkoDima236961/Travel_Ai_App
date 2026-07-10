"use client";

import type {
  RouteValidationWarning,
  TransportMode,
  TripRoute,
  TripRouteLeg,
  TripRouteStop,
  TripStyle
} from "@/entities/route/model";
import { RouteLegCard } from "./RouteLegCard";
import { RouteStopCard } from "./RouteStopCard";
import { RouteSummaryCard } from "./RouteSummaryCard";
import { RouteValidationWarnings } from "./RouteValidationWarnings";
import { TripStyleSelector } from "./TripStyleSelector";

type TripRouteBuilderProps = {
  value: TripRoute;
  onChange: (route: TripRoute) => void;
  totalDays?: number;
  currency?: string;
};

const INPUT =
  "h-11 w-full rounded-xl border border-sand-400 bg-[#FFFDFA] px-3.5 text-[14px] text-cocoa-900 outline-none transition placeholder:text-cocoa-400 focus:border-clay focus:ring-[3px] focus:ring-clay-tint";
const LABEL = "block text-[13px] font-semibold text-cocoa-700";

export function TripRouteBuilder({ value, onChange, totalDays = 1, currency = "EUR" }: TripRouteBuilderProps) {
  const route = ensureRouteShape(value);
  const warnings = getRouteValidationWarnings(route, totalDays);

  function updateRoute(next: TripRoute) {
    onChange(syncLegs(next));
  }

  function updateStop(index: number, stop: TripRouteStop) {
    updateRoute({ ...route, stops: route.stops.map((item, itemIndex) => (itemIndex === index ? stop : item)) });
  }

  function addStop() {
    updateRoute({
      ...route,
      stops: [
        ...route.stops,
        {
          id: makeStopId(route.stops.length + 1),
          destination: "",
          country: "",
          nights: 1,
          accommodationHint: "unknown"
        }
      ]
    });
  }

  function removeStop(index: number) {
    updateRoute({ ...route, stops: route.stops.filter((_, itemIndex) => itemIndex !== index) });
  }

  function moveStop(index: number, direction: -1 | 1) {
    const next = [...route.stops];
    const target = index + direction;
    if (target < 0 || target >= next.length) {
      return;
    }
    [next[index], next[target]] = [next[target], next[index]];
    updateRoute({ ...route, stops: next });
  }

  function updateLeg(index: number, leg: TripRouteLeg) {
    updateRoute({ ...route, legs: (route.legs ?? []).map((item, itemIndex) => (itemIndex === index ? leg : item)) });
  }

  const styles = route.preferences?.tripStyles ?? [];

  return (
    <div className="flex flex-col gap-5">
      <div className="rounded-[18px] border border-sand-300 bg-sand-50 p-5">
        <label className={LABEL}>
          Origin
          <input
            className={`${INPUT} mt-2`}
            placeholder="Bratislava"
            value={route.origin?.name ?? ""}
            onChange={(event) =>
              updateRoute({
                ...route,
                origin: { ...(route.origin ?? {}), name: event.target.value }
              })
            }
          />
        </label>
      </div>

      <div className="flex flex-col gap-3">
        {route.stops.map((stop, index) => (
          <RouteStopCard
            key={stop.id}
            stop={stop}
            index={index}
            canMoveUp={index > 0}
            canMoveDown={index < route.stops.length - 1}
            onChange={(nextStop) => updateStop(index, nextStop)}
            onMoveUp={() => moveStop(index, -1)}
            onMoveDown={() => moveStop(index, 1)}
            onRemove={() => removeStop(index)}
          />
        ))}
        <button
          type="button"
          onClick={addStop}
          className="h-11 rounded-full border border-dashed border-sand-500 bg-white px-4 text-[14px] font-semibold text-cocoa-600 transition hover:border-clay hover:text-clay-deep"
        >
          Add stop
        </button>
      </div>

      {route.stops.length > 0 ? (
        <div className="flex flex-col gap-3">
          {(route.legs ?? []).map((leg, index) => (
            <RouteLegCard
              key={leg.id}
              leg={leg}
              index={index}
              fromName={
                index === 0
                  ? route.origin?.name || "Origin"
                  : route.stops[index - 1]?.destination || `Stop ${index}`
              }
              toStop={route.stops[index]}
              onChange={(nextLeg) => updateLeg(index, nextLeg)}
            />
          ))}
        </div>
      ) : null}

      <div className="rounded-[18px] border border-sand-300 bg-white p-5">
        <p className={LABEL}>Trip styles</p>
        <div className="mt-3">
          <TripStyleSelector
            value={styles}
            onChange={(nextStyles: TripStyle[]) =>
              updateRoute({
                ...route,
                preferences: {
                  ...(route.preferences ?? {}),
                  tripStyles: nextStyles
                }
              })
            }
          />
        </div>
      </div>

      <RouteValidationWarnings warnings={warnings} />
      <RouteSummaryCard route={route} currency={currency} title="Route summary" />
    </div>
  );
}

export function createDefaultTripRoute(): TripRoute {
  return syncLegs({
    origin: { name: "" },
    returnToOrigin: false,
    stops: [
      { id: "stop_1", destination: "", country: "", nights: 2, accommodationHint: "unknown" },
      { id: "stop_2", destination: "", country: "", nights: 2, accommodationHint: "unknown" }
    ],
    legs: [],
    preferences: {
      preferredModes: ["train"],
      avoidModes: [],
      carAvailable: false,
      maxTransferHoursPerDay: 4,
      tripStyles: ["train_trip"]
    }
  });
}

export function getRouteValidationWarnings(route: TripRoute, totalDays = 1): RouteValidationWarning[] {
  const warnings: RouteValidationWarning[] = [];
  const stopCount = route.stops.length;
  if (stopCount === 0) {
    warnings.push({ code: "missing_stops", message: "Add at least one stop.", severity: "error" });
  }
  if (stopCount > 20) {
    warnings.push({ code: "too_many_stops", message: "Routes can include up to 20 stops.", severity: "error" });
  }
  if (stopCount > Math.max(1, totalDays)) {
    warnings.push({
      code: "rushed_route",
      message: `${stopCount} stops in ${totalDays} day${totalDays === 1 ? "" : "s"} may feel rushed.`,
      severity: "warning"
    });
  }
  for (const [index, stop] of route.stops.entries()) {
    if (!stop.destination.trim()) {
      warnings.push({ code: `missing_stop_${index}`, message: `Stop ${index + 1} needs a destination.`, severity: "error" });
    }
  }
  const avoidModes = new Set(route.preferences?.avoidModes ?? []);
  for (const leg of route.legs ?? []) {
    if (avoidModes.has(leg.mode as TransportMode)) {
      warnings.push({
        code: `avoid_${leg.id}`,
        message: `${leg.mode.replace("_", " ")} is selected but appears in avoided transport modes.`,
        severity: "warning"
      });
    }
    const maxHours = route.preferences?.maxTransferHoursPerDay;
    if (maxHours && leg.estimatedDurationMinutes && leg.estimatedDurationMinutes > maxHours * 60) {
      warnings.push({
        code: `long_${leg.id}`,
        message: "This transfer is longer than your max transfer time.",
        severity: "warning"
      });
    }
  }
  const styles = route.preferences?.tripStyles ?? [];
  if (styles.includes("camping") && !route.stops.some((stop) => stop.accommodationHint === "campsite")) {
    warnings.push({
      code: "camping_without_campsite",
      message: "Camping selected, but no campsite accommodation stop is configured.",
      severity: "warning"
    });
  }
  if (styles.includes("hiking")) {
    warnings.push({
      code: "hiking_approximate",
      message: "Hiking routes are approximate and should be checked with local maps.",
      severity: "info"
    });
  }
  return warnings;
}

function ensureRouteShape(route: TripRoute): TripRoute {
  return syncLegs({
    ...route,
    stops: route.stops ?? [],
    legs: route.legs ?? [],
    preferences: route.preferences ?? {}
  });
}

function syncLegs(route: TripRoute): TripRoute {
  const existingByTarget = new Map((route.legs ?? []).map((leg) => [leg.toStopId, leg]));
  const legs = route.stops.map((stop, index) => {
    const previous = existingByTarget.get(stop.id);
    const fromStopId = index === 0 ? "origin" : route.stops[index - 1]?.id ?? "origin";
    const fromName = index === 0 ? route.origin?.name || "Origin" : route.stops[index - 1]?.destination || "";
    return {
      id: previous?.id ?? `leg_${index + 1}`,
      fromStopId,
      toStopId: stop.id,
      fromName,
      toName: stop.destination,
      mode: (previous?.mode ?? route.preferences?.preferredModes?.[0] ?? "train") as TransportMode,
      estimatedDurationMinutes: previous?.estimatedDurationMinutes ?? null,
      estimatedDistanceKm: previous?.estimatedDistanceKm ?? null,
      estimatedCost: previous?.estimatedCost ?? null,
      notes: previous?.notes ?? null
    };
  });
  return { ...route, legs };
}

function makeStopId(index: number) {
  return `stop_${index}_${Date.now().toString(36)}`;
}
