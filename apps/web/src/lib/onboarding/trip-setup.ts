import type { Trip } from "@/entities/trip/model";

export type TripSetupItemStatus = "complete" | "recommended" | "optional" | "needs_attention";

export type TripSetupItemId =
  | "destination_dates"
  | "itinerary"
  | "budget"
  | "route_transport"
  | "checklist"
  | "collaborators"
  | "health";

export type TripSetupItem = {
  id: TripSetupItemId;
  status: TripSetupItemStatus;
  href: string;
};

export type TripSetupInput = {
  trip: Trip;
  checklistExists?: boolean;
  collaboratorCount?: number;
  healthLoaded?: boolean;
  healthHasCriticalIssues?: boolean;
};

export function buildTripSetupChecklist({
  trip,
  checklistExists = false,
  collaboratorCount = 0,
  healthLoaded = false,
  healthHasCriticalIssues = false
}: TripSetupInput): TripSetupItem[] {
  const base = `/trips/${trip.id}`;
  const destinationAndDatesComplete = Boolean(trip.destination.trim() && trip.startDate && trip.days > 0);
  const itineraryComplete = Boolean(trip.itinerary?.days?.length);
  const budgetComplete = trip.budgetAmount != null || trip.budget?.amount != null;
  const routeComplete =
    trip.tripType !== "multi_destination" ||
    Boolean(
      trip.route &&
        trip.route.stops.length >= 2 &&
        (trip.route.legs?.length ?? 0) >= trip.route.stops.length - 1 &&
        trip.route.legs?.every((leg) => Boolean(leg.mode))
    );

  return [
    { id: "destination_dates", status: destinationAndDatesComplete ? "complete" : "needs_attention", href: `${base}?tab=dates` },
    { id: "itinerary", status: itineraryComplete ? "complete" : "recommended", href: `${base}?tab=itinerary` },
    { id: "budget", status: budgetComplete ? "complete" : "recommended", href: `${base}?tab=budget` },
    { id: "route_transport", status: routeComplete ? "complete" : "recommended", href: `${base}?tab=route` },
    { id: "checklist", status: checklistExists ? "complete" : "recommended", href: `${base}?tab=checklist` },
    { id: "collaborators", status: collaboratorCount > 0 ? "complete" : "optional", href: `${base}?tab=team` },
    {
      id: "health",
      status: healthHasCriticalIssues ? "needs_attention" : healthLoaded ? "complete" : "recommended",
      href: `${base}?tab=health`
    }
  ];
}

export function completedTripSetupCount(items: TripSetupItem[]) {
  return items.filter((item) => item.status === "complete").length;
}

export function tripSetupDismissalKey(userId: string, tripId: string) {
  return `tripSetupChecklistDismissed:${userId}:${tripId}`;
}
