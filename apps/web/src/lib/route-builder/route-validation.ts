import type { TripRoute, TripRouteLeg, TripRouteStop } from "@/entities/route/model";
import type { Itinerary, ItineraryDay } from "@/entities/trip/model";
import type { TripHealthIssue } from "@/types/trip-health";
import { isLegTransportStale } from "./route-draft";

export type RouteBuilderIssueSeverity = "info" | "warning" | "error";

export type RouteBuilderIssue = {
  id: string;
  category: "route" | "transport" | "itinerary";
  severity: RouteBuilderIssueSeverity;
  title: string;
  description: string;
  stopId?: string;
  legId?: string;
  dayNumber?: number;
  action?: {
    label: string;
    href?: string;
    type?: "find_transport" | "edit_stop" | "open_day" | "open_health";
  };
  source: "draft" | "trip_health";
};

export type StopDayMappingEntry = {
  stop: TripRouteStop;
  days: ItineraryDay[];
  itemCount: number;
  transferDayCount: number;
  warnings: string[];
};

const routeHealthPrefixes = [
  "route_missing",
  "route_incomplete",
  "transport_missing_option",
  "transport_low_confidence",
  "transport_mock_option",
  "transport_duration_high",
  "transport_itinerary_time_conflict",
  "itinerary_route_stop_mismatch",
  "missing_transfer_item",
  "activity_during_transport",
  "activity_before_transport_arrival",
  "too_many_stops_for_duration",
  "route_duration_mismatch",
  "stale_provider_data"
];

const optionRequiredModes = new Set(["train", "bus", "flight", "ferry", "boat"]);

export function getRouteBuilderIssues(input: {
  route: TripRoute;
  itinerary?: Itinerary | null;
  healthIssues?: TripHealthIssue[] | null;
  totalDays?: number;
  tripId?: string;
  multiDestination?: boolean;
}): RouteBuilderIssue[] {
  const local = validateDraftRoute(input.route, input.itinerary, {
    totalDays: input.totalDays,
    tripId: input.tripId,
    multiDestination: input.multiDestination
  });
  const health = filterRouteHealthIssues(input.healthIssues ?? [], input.tripId);
  const seen = new Set(local.map((issue) => issue.id));
  return [...local, ...health.filter((issue) => !seen.has(issue.id))];
}

export function filterRouteHealthIssues(
  healthIssues: TripHealthIssue[],
  tripId?: string
): RouteBuilderIssue[] {
  return healthIssues
    .filter(
      (issue) =>
        issue.status === "open" &&
        (issue.category === "route" || issue.category === "transport" || issue.category === "itinerary") &&
        routeHealthPrefixes.some((prefix) => issue.id.startsWith(prefix))
    )
    .map((issue) => {
      const legId = metadataString(issue.metadata, "routeLegId") ?? idSegment(issue.id, "leg");
      const stopId = metadataString(issue.metadata, "routeStopId") ?? metadataString(issue.metadata, "stopId");
      const dayNumber = metadataNumber(issue.metadata, "dayNumber") ?? dayFromIssueId(issue.id);
      return {
        id: issue.id,
        category: normalizeCategory(issue.category),
        severity: healthSeverity(issue.severity),
        title: issue.title,
        description: issue.description,
        legId,
        stopId,
        dayNumber,
        action: issue.action
          ? { label: issue.action.label, href: normalizeRouteHref(issue.action.href, tripId, legId, stopId) }
          : defaultIssueAction(issue.id, tripId, legId, stopId, dayNumber),
        source: "trip_health" as const
      };
    });
}

export function validateDraftRoute(
  route: TripRoute,
  itinerary?: Itinerary | null,
  options: { totalDays?: number; tripId?: string; multiDestination?: boolean } = {}
): RouteBuilderIssue[] {
  const issues: RouteBuilderIssue[] = [];
  const stops = route.stops ?? [];
  const legs = route.legs ?? [];
  const totalDays = Math.max(1, options.totalDays ?? itinerary?.days.length ?? 1);

  if (stops.length === 0 || (options.multiDestination && stops.length < 2)) {
    issues.push(issue("route_missing", "route", "error", "Route needs more stops", options.multiDestination
      ? "Add at least two destinations to this multi-destination trip."
      : "Add at least one destination to build the route."));
  }

  const names = new Map<string, TripRouteStop>();
  for (const [index, stop] of stops.entries()) {
    const name = stopName(stop).trim();
    if (!name) {
      issues.push({
        ...issue(`missing_stop_name:${stop.id || index}`, "route", "error", "Stop name is missing", `Stop ${index + 1} needs a destination.`),
        stopId: stop.id,
        action: { label: "Edit stop", type: "edit_stop" }
      });
      continue;
    }
    const normalized = normalizeName(name);
    const duplicate = names.get(normalized);
    if (duplicate) {
      issues.push({
        ...issue(`duplicate_stop:${duplicate.id}:${stop.id}`, "route", "error", "Duplicate route stop", `${name} appears more than once in this route.`),
        stopId: stop.id,
        action: { label: "Edit stop", type: "edit_stop" }
      });
    } else {
      names.set(normalized, stop);
    }
  }

  if (stops.length > totalDays) {
    issues.push(issue(
      "too_many_stops_for_duration",
      "route",
      "warning",
      "Too many stops for this trip length",
      `${stops.length} stops across ${totalDays} days may make the route feel rushed.`
    ));
  }

  const plannedNights = stops.reduce((sum, stop) => sum + Math.max(0, stop.nights ?? 0), 0);
  if (plannedNights > totalDays) {
    issues.push(issue(
      "route_duration_mismatch",
      "route",
      "warning",
      "Route duration does not match trip length",
      `${plannedNights} planned nights exceed the ${totalDays}-day trip.`
    ));
  }

  const expectedPairs = stops.map((stop, index) => ({
    from: index === 0 ? "origin" : stops[index - 1]?.id,
    to: stop.id
  }));
  for (const [index, expected] of expectedPairs.entries()) {
    const leg = legs[index];
    if (!leg || leg.fromStopId !== expected.from || leg.toStopId !== expected.to) {
      issues.push(issue(
        `route_incomplete:leg_${index + 1}`,
        "route",
        "error",
        "Route leg is incomplete",
        `The transfer into ${stopName(stops[index]) || `stop ${index + 1}`} needs to be rebuilt.`
      ));
      continue;
    }
    validateLeg(leg, issues, options.tripId);
  }

  if (itinerary) {
    issues.push(...validateItineraryConnection(route, itinerary, options.tripId));
  }
  return issues;
}

export function mapItineraryToStops(
  route: TripRoute,
  itinerary?: Itinerary | null
): StopDayMappingEntry[] {
  const days = itinerary?.days ?? [];
  return route.stops.map((stop) => {
    const mappedDays = days.filter((day) => dayMatchesStop(day, stop));
    const warnings: string[] = [];
    for (const day of mappedDays) {
      if (day.primaryStopId && day.primaryStopId !== stop.id) {
        warnings.push(`Day ${day.day} location text matches, but its stop assignment points elsewhere.`);
      }
    }
    return {
      stop,
      days: mappedDays,
      itemCount: mappedDays.reduce((sum, day) => sum + day.items.length, 0),
      transferDayCount: mappedDays.filter((day) => day.transferDay || day.items.some(isTransferItem)).length,
      warnings
    };
  });
}

function validateLeg(leg: TripRouteLeg, issues: RouteBuilderIssue[], tripId?: string) {
  if (!String(leg.mode ?? "").trim()) {
    issues.push({
      ...issue(`missing_leg_mode:${leg.id}`, "route", "error", "Transport mode is missing", "Choose how you will travel for this route leg."),
      legId: leg.id
    });
  }
  if (optionRequiredModes.has(String(leg.mode)) && !leg.selectedTransportOption) {
    issues.push({
      ...issue(`transport_missing_option:${leg.id}`, "transport", "warning", "Missing transport option", "No specific service is selected for this leg."),
      legId: leg.id,
      action: {
        label: "Find transport",
        type: "find_transport",
        href: tripId ? `/trips/${tripId}?tab=route&legId=${encodeURIComponent(leg.id)}` : undefined
      }
    });
  }
  if (leg.selectedTransportOption?.confidence === "low") {
    issues.push({
      ...issue(`transport_low_confidence:${leg.id}`, "transport", "warning", "Low-confidence transport", "Verify this service's schedule and price before booking."),
      legId: leg.id,
      action: { label: "Find transport", type: "find_transport" }
    });
  }
  if (leg.selectedTransportOption?.provider === "mock") {
    issues.push({
      ...issue(`transport_mock_option:${leg.id}`, "transport", "warning", "Estimated transport option", "This option is an estimate rather than a live provider result."),
      legId: leg.id,
      action: { label: "Find transport", type: "find_transport" }
    });
  }
  if (isLegTransportStale(leg)) {
    issues.push({
      ...issue(`stale_provider_data:${leg.id}`, "transport", "warning", "Transport option may be stale", "A stop, date, or route order changed after this option was selected."),
      legId: leg.id,
      action: { label: "Re-search transport", type: "find_transport" }
    });
  }
}

function validateItineraryConnection(route: TripRoute, itinerary: Itinerary, tripId?: string) {
  const issues: RouteBuilderIssue[] = [];
  const stopIds = new Set(route.stops.map((stop) => stop.id));
  for (const day of itinerary.days) {
    const stop = route.stops.find((candidate) => dayMatchesStop(day, candidate));
    if (day.primaryStopId && !stopIds.has(day.primaryStopId)) {
      issues.push(dayMismatchIssue(day, `Day ${day.day} points to a stop that is no longer in the route.`, tripId));
    } else if (!day.primaryStopId && !stop) {
      issues.push(dayMismatchIssue(day, `Day ${day.day} has no route stop assignment.`, tripId));
    } else if (day.primaryStopId && stop && stop.id !== day.primaryStopId) {
      issues.push(dayMismatchIssue(
        day,
        `Day ${day.day} is assigned to ${stopName(route.stops.find((item) => item.id === day.primaryStopId)) || "another stop"}, but its location says ${day.locationName}.`,
        tripId
      ));
    }
  }

  for (const leg of route.legs ?? []) {
    const destination = route.stops.find((stop) => stop.id === leg.toStopId);
    const transferFound = itinerary.days.some((day) =>
      day.items.some((item) =>
        isTransferItem(item) &&
        (item.transfer?.legId === leg.id || normalizeName(item.transfer?.to ?? "") === normalizeName(stopName(destination)))
      )
    );
    if (!transferFound && itinerary.days.length > 0) {
      issues.push({
        ...issue(`missing_transfer_item:${leg.id}`, "itinerary", "warning", "Transfer is missing from itinerary", `No itinerary transfer is linked to the route into ${stopName(destination) || leg.toName || "the next stop"}.`),
        legId: leg.id,
        action: { label: "Open Trip Health", type: "open_health", href: tripId ? `/trips/${tripId}?tab=health` : undefined }
      });
    }
    issues.push(...activityOverlapIssues(leg, itinerary, tripId));
  }
  return issues;
}

function activityOverlapIssues(leg: TripRouteLeg, itinerary: Itinerary, tripId?: string): RouteBuilderIssue[] {
  const option = leg.selectedTransportOption;
  if (!option?.departureTime || !option.arrivalTime) {
    return [];
  }
  const departureDate = option.departureDate ?? leg.departureDate;
  const arrivalDate = option.arrivalDate ?? departureDate;
  if (!departureDate || departureDate !== arrivalDate) {
    return [];
  }
  const start = timeToMinutes(option.departureTime);
  const end = timeToMinutes(option.arrivalTime);
  if (start == null || end == null || end <= start) {
    return [];
  }
  const issues: RouteBuilderIssue[] = [];
  for (const day of itinerary.days.filter((candidate) => candidate.date === departureDate)) {
    day.items.forEach((item, itemIndex) => {
      const itemStart = timeToMinutes(item.time);
      if (itemStart == null || itemStart < start || itemStart >= end || isTransferItem(item) || item.type === "rest") {
        return;
      }
      issues.push({
        ...issue(`activity_during_transport:day_${day.day}:item_${itemIndex}:${leg.id}`, "itinerary", "error", "Activity overlaps transport", `${item.name} starts while the selected transport is in progress.`),
        legId: leg.id,
        dayNumber: day.day,
        action: { label: "Open itinerary day", type: "open_day", href: tripId ? `/trips/${tripId}?tab=itinerary&day=${day.day}` : undefined }
      });
    });
  }
  return issues;
}

function dayMismatchIssue(day: ItineraryDay, description: string, tripId?: string): RouteBuilderIssue {
  return {
    ...issue(`itinerary_route_stop_mismatch:${day.day}`, "itinerary", "warning", "Activity does not match route stop", description),
    dayNumber: day.day,
    action: { label: "Open itinerary day", type: "open_day", href: tripId ? `/trips/${tripId}?tab=itinerary&day=${day.day}` : undefined }
  };
}

function dayMatchesStop(day: ItineraryDay, stop: TripRouteStop): boolean {
  if (day.primaryStopId === stop.id) {
    return true;
  }
  const dayLocation = normalizeName(day.locationName ?? "");
  if (!dayLocation) {
    return false;
  }
  return [stop.destination, stop.city].some((name) => normalizeName(name ?? "") === dayLocation);
}

function isTransferItem(item: ItineraryDay["items"][number]): boolean {
  return item.type === "transfer" || item.type === "transport" || Boolean(item.transfer);
}

function issue(
  id: string,
  category: RouteBuilderIssue["category"],
  severity: RouteBuilderIssueSeverity,
  title: string,
  description: string
): RouteBuilderIssue {
  return { id, category, severity, title, description, source: "draft" };
}

function defaultIssueAction(
  issueId: string,
  tripId?: string,
  legId?: string,
  stopId?: string,
  dayNumber?: number
): RouteBuilderIssue["action"] {
  if (issueId.startsWith("transport_") && legId) {
    return { label: "Find transport", type: "find_transport", href: tripId ? `/trips/${tripId}?tab=route&legId=${encodeURIComponent(legId)}` : undefined };
  }
  if (dayNumber) {
    return { label: "Open itinerary day", type: "open_day", href: tripId ? `/trips/${tripId}?tab=itinerary&day=${dayNumber}` : undefined };
  }
  if (stopId) {
    return { label: "Edit stop", type: "edit_stop", href: tripId ? `/trips/${tripId}?tab=route&stopId=${encodeURIComponent(stopId)}` : undefined };
  }
  return { label: "Open Trip Health", type: "open_health", href: tripId ? `/trips/${tripId}?tab=health` : undefined };
}

function normalizeRouteHref(href: string, tripId?: string, legId?: string, stopId?: string): string {
  if (!tripId || !href.includes("tab=route")) {
    return href;
  }
  if (legId && !href.includes("legId=")) {
    return `${href}&legId=${encodeURIComponent(legId)}`;
  }
  if (stopId && !href.includes("stopId=")) {
    return `${href}&stopId=${encodeURIComponent(stopId)}`;
  }
  return href;
}

function normalizeCategory(category: TripHealthIssue["category"]): RouteBuilderIssue["category"] {
  return category === "transport" ? "transport" : category === "itinerary" ? "itinerary" : "route";
}

function healthSeverity(severity: TripHealthIssue["severity"]): RouteBuilderIssueSeverity {
  return severity === "critical" || severity === "high" ? "error" : severity === "warning" ? "warning" : "info";
}

function metadataString(metadata: Record<string, unknown> | undefined, key: string): string | undefined {
  const value = metadata?.[key];
  return typeof value === "string" && value.trim() ? value : undefined;
}

function metadataNumber(metadata: Record<string, unknown> | undefined, key: string): number | undefined {
  const value = metadata?.[key];
  return typeof value === "number" && Number.isFinite(value) ? value : undefined;
}

function idSegment(id: string, prefix: string): string | undefined {
  const value = id.split(":").find((segment) => segment.startsWith(prefix));
  return value || undefined;
}

function dayFromIssueId(id: string): number | undefined {
  const match = id.match(/(?:day_?|:)(\d+)/);
  return match ? Number(match[1]) : undefined;
}

function stopName(stop?: TripRouteStop): string {
  return stop?.city || stop?.destination || "";
}

function normalizeName(value: string): string {
  return value.trim().toLocaleLowerCase().replace(/\s+/g, " ");
}

function timeToMinutes(value: string): number | null {
  const match = value.match(/^(\d{1,2}):(\d{2})/);
  if (!match) {
    return null;
  }
  const hours = Number(match[1]);
  const minutes = Number(match[2]);
  return hours >= 0 && hours < 24 && minutes >= 0 && minutes < 60 ? hours * 60 + minutes : null;
}
