import type { TripRoute, TripRouteLeg, TripRouteStop } from "@/entities/route/model";
import type { Itinerary } from "@/entities/trip/model";

export type RouteImpact = {
  affectedLegIds: string[];
  removedTransportOptionCount: number;
  staleTransportOptionCount: number;
  itineraryImpact: boolean;
  budgetImpact: boolean;
  reminderImpact: boolean;
  approvalMayReset: boolean;
  stopOrderChanged: boolean;
  legCountChanged: boolean;
};

export type RouteDraftState = RouteImpact & {
  originalRoute: TripRoute;
  draftRoute: TripRoute;
  dirty: boolean;
};

export function createRouteDraft(
  originalRoute: TripRoute,
  draftRoute: TripRoute = originalRoute,
  itinerary?: Itinerary | null
): RouteDraftState {
  const original = cloneRoute(originalRoute);
  const draft = cloneRoute(draftRoute);
  const dirty = routeFingerprint(original) !== routeFingerprint(draft);
  return {
    originalRoute: original,
    draftRoute: draft,
    dirty,
    ...calculateRouteImpact(original, draft, itinerary)
  };
}

export function updateRouteDraft(
  state: RouteDraftState,
  draftRoute: TripRoute,
  itinerary?: Itinerary | null
): RouteDraftState {
  return createRouteDraft(state.originalRoute, draftRoute, itinerary);
}

export function reorderRouteStops(
  route: TripRoute,
  fromIndex: number,
  toIndex: number
): TripRoute {
  if (
    fromIndex === toIndex ||
    fromIndex < 0 ||
    toIndex < 0 ||
    fromIndex >= route.stops.length ||
    toIndex >= route.stops.length
  ) {
    return cloneRoute(route);
  }
  const stops = [...route.stops];
  const [moved] = stops.splice(fromIndex, 1);
  stops.splice(toIndex, 0, moved);
  return rebuildRouteLegs(route, stops);
}

export function removeRouteStop(route: TripRoute, stopId: string): TripRoute {
  return rebuildRouteLegs(
    route,
    route.stops.filter((stop) => stop.id !== stopId)
  );
}

export function updateRouteStop(
  route: TripRoute,
  stopId: string,
  nextStop: TripRouteStop
): TripRoute {
  const previous = route.stops.find((stop) => stop.id === stopId);
  const stops = route.stops.map((stop) => (stop.id === stopId ? nextStop : stop));
  const next = rebuildRouteLegs(route, stops);
  if (!previous || !transportSensitiveStopChange(previous, nextStop)) {
    return next;
  }
  return {
    ...next,
    legs: (next.legs ?? []).map((leg) =>
      leg.fromStopId === stopId || leg.toStopId === stopId
        ? markSelectedTransportStale(leg, "A connected stop or travel date changed.")
        : leg
    )
  };
}

export function rebuildRouteLegs(route: TripRoute, stops: TripRouteStop[]): TripRoute {
  const existingLegs = route.legs ?? [];
  const existingByPair = new Map(existingLegs.map((leg) => [legPair(leg.fromStopId, leg.toStopId), leg]));
  const fallbackByTarget = new Map(existingLegs.map((leg) => [leg.toStopId, leg]));
  const usedIds = new Set(existingLegs.map((leg) => leg.id));

  const legs = stops.map((stop, index): TripRouteLeg => {
    const fromStopId = index === 0 ? "origin" : stops[index - 1]?.id ?? "origin";
    const fromName =
      index === 0
        ? route.origin?.name || "Origin"
        : stops[index - 1]?.city || stops[index - 1]?.destination || "";
    const toName = stop.city || stop.destination;
    const exact = existingByPair.get(legPair(fromStopId, stop.id));
    if (exact) {
      return {
        ...cloneLeg(exact),
        fromName,
        toName
      };
    }

    const fallback = fallbackByTarget.get(stop.id) ?? existingLegs[index];
    return {
      id: makeLocalLegId(fromStopId, stop.id, usedIds),
      fromStopId,
      toStopId: stop.id,
      fromName,
      toName,
      mode: fallback?.mode ?? route.preferences?.preferredModes?.[0] ?? "train",
      departureDate: stop.arrivalDate ?? fallback?.departureDate ?? null,
      estimatedDurationMinutes: null,
      estimatedDistanceKm: null,
      estimatedCost: null,
      selectedTransportOption: null,
      notes: fallback?.notes ?? null,
      providerMetadata: null,
      warnings: []
    };
  });

  return { ...cloneRoute(route), stops: stops.map(cloneStop), legs };
}

export function markSelectedTransportStale(leg: TripRouteLeg, reason: string): TripRouteLeg {
  if (!leg.selectedTransportOption) {
    return leg;
  }
  const warning = "Transport option may no longer match this leg.";
  return {
    ...cloneLeg(leg),
    providerMetadata: {
      ...(leg.providerMetadata ?? {}),
      stale: true,
      staleReason: reason
    },
    warnings: Array.from(new Set([...(leg.warnings ?? []), warning]))
  };
}

export function isLegTransportStale(leg: TripRouteLeg): boolean {
  return Boolean(
    leg.selectedTransportOption &&
      (leg.providerMetadata?.stale === true ||
        (leg.warnings ?? []).some((warning) => warning.toLowerCase().includes("stale") || warning.toLowerCase().includes("no longer match")))
  );
}

export function calculateRouteImpact(
  originalRoute: TripRoute,
  draftRoute: TripRoute,
  itinerary?: Itinerary | null
): RouteImpact {
  const originalLegs = originalRoute.legs ?? [];
  const draftLegs = draftRoute.legs ?? [];
  const draftByPair = new Map(draftLegs.map((leg) => [legPair(leg.fromStopId, leg.toStopId), leg]));
  const affectedLegIds = originalLegs
    .filter((leg) => {
      const next = draftByPair.get(legPair(leg.fromStopId, leg.toStopId));
      return !next || routeLegFingerprint(leg) !== routeLegFingerprint(next);
    })
    .map((leg) => leg.id);
  const removedTransportOptionCount = originalLegs.filter((leg) => {
    if (!leg.selectedTransportOption) {
      return false;
    }
    const next = draftByPair.get(legPair(leg.fromStopId, leg.toStopId));
    return next?.selectedTransportOption?.id !== leg.selectedTransportOption.id;
  }).length;
  const staleTransportOptionCount = draftLegs.filter(isLegTransportStale).length;
  const stopOrderChanged =
    originalRoute.stops.map((stop) => stop.id).join("|") !==
    draftRoute.stops.map((stop) => stop.id).join("|");
  const legCountChanged = originalLegs.length !== draftLegs.length;
  const dirty = routeFingerprint(originalRoute) !== routeFingerprint(draftRoute);
  const transportOrTimingChanged =
    dirty &&
    (stopOrderChanged ||
      legCountChanged ||
      affectedLegIds.length > 0 ||
      stopTimingFingerprint(originalRoute) !== stopTimingFingerprint(draftRoute));

  return {
    affectedLegIds,
    removedTransportOptionCount,
    staleTransportOptionCount,
    itineraryImpact: Boolean(itinerary?.days.length) && dirty,
    budgetImpact: transportOrTimingChanged,
    reminderImpact: transportOrTimingChanged,
    approvalMayReset: Boolean(itinerary?.days.length) && dirty,
    stopOrderChanged,
    legCountChanged
  };
}

export function cloneRoute(route: TripRoute): TripRoute {
  return {
    ...route,
    origin: route.origin
      ? {
          ...route.origin,
          coordinates: route.origin.coordinates ? { ...route.origin.coordinates } : route.origin.coordinates
        }
      : route.origin,
    stops: (route.stops ?? []).map(cloneStop),
    legs: (route.legs ?? []).map(cloneLeg),
    preferences: route.preferences
      ? {
          ...route.preferences,
          preferredModes: [...(route.preferences.preferredModes ?? [])],
          avoidModes: [...(route.preferences.avoidModes ?? [])],
          tripStyles: [...(route.preferences.tripStyles ?? [])]
        }
      : route.preferences
  };
}

function cloneStop(stop: TripRouteStop): TripRouteStop {
  return {
    ...stop,
    coordinates: stop.coordinates ? { ...stop.coordinates } : stop.coordinates
  };
}

function cloneLeg(leg: TripRouteLeg): TripRouteLeg {
  return {
    ...leg,
    estimatedCost: leg.estimatedCost ? { ...leg.estimatedCost } : leg.estimatedCost,
    selectedTransportOption: leg.selectedTransportOption
      ? {
          ...leg.selectedTransportOption,
          estimatedPrice: leg.selectedTransportOption.estimatedPrice
            ? { ...leg.selectedTransportOption.estimatedPrice }
            : leg.selectedTransportOption.estimatedPrice,
          warnings: [...(leg.selectedTransportOption.warnings ?? [])]
        }
      : leg.selectedTransportOption,
    providerMetadata: leg.providerMetadata ? { ...leg.providerMetadata } : leg.providerMetadata,
    warnings: [...(leg.warnings ?? [])]
  };
}

function routeFingerprint(route: TripRoute): string {
  return JSON.stringify(route);
}

function routeLegFingerprint(leg: TripRouteLeg): string {
  return JSON.stringify(leg);
}

function stopTimingFingerprint(route: TripRoute): string {
  return route.stops
    .map((stop) => `${stop.id}:${stop.destination}:${stop.city ?? ""}:${stop.arrivalDate ?? ""}:${stop.departureDate ?? ""}:${stop.nights ?? ""}`)
    .join("|");
}

function transportSensitiveStopChange(left: TripRouteStop, right: TripRouteStop): boolean {
  return (
    left.destination !== right.destination ||
    left.city !== right.city ||
    left.country !== right.country ||
    left.arrivalDate !== right.arrivalDate ||
    left.departureDate !== right.departureDate
  );
}

function legPair(fromStopId: string, toStopId: string): string {
  return `${fromStopId}->${toStopId}`;
}

function makeLocalLegId(fromStopId: string, toStopId: string, usedIds: Set<string>): string {
  const stem = `local_leg_${safeId(fromStopId)}_${safeId(toStopId)}`;
  let candidate = stem;
  let suffix = 2;
  while (usedIds.has(candidate)) {
    candidate = `${stem}_${suffix}`;
    suffix += 1;
  }
  usedIds.add(candidate);
  return candidate;
}

function safeId(value: string): string {
  return value.replace(/[^a-zA-Z0-9_-]/g, "_");
}
