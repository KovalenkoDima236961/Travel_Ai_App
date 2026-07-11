import type { DayDistanceSummary } from "@/entities/itinerary/model/distance-utils";
import type { TripAccommodation } from "@/entities/accommodation/model";
import type { BudgetSummary } from "@/entities/budget/model";
import type { TripChecklist, TripChecklistItem } from "@/entities/checklist/model";
import type { Place } from "@/entities/place/model";
import type { PublicTrip } from "@/entities/share/model";
import type { RouteEstimate, TripRoute, TripRouteLeg, TripRouteStop } from "@/entities/route/model";
import type { Itinerary, ItineraryDay, ItineraryItem, Trip } from "@/entities/trip/model";
import type { WeatherForecast } from "@/entities/weather/model";

export type ExportTrip = {
  destination: string;
  startDate?: string | null;
  days: number;
  budgetAmount?: number | null;
  budgetCurrency?: string | null;
  travelers?: number | null;
  interests?: string[];
  pace?: string | null;
  accommodation?: TripAccommodation | null;
  route?: TripRoute | null;
  itinerary?: Itinerary | null;
  weatherSummary?: ExportWeatherDay[] | null;
  distanceSummary?: ExportDistanceDay[] | null;
  budgetSummary?: BudgetSummary | null;
  checklist?: TripChecklist | null;
  source: "private" | "public";
};

export type ExportWeatherDay = {
  dayNumber: number;
  date?: string | null;
  summary?: string | null;
  temperatureMinC?: number | null;
  temperatureMaxC?: number | null;
  precipitationChance?: number | null;
};

export type ExportDistanceDay = {
  dayNumber: number;
  distanceKm?: number | null;
  walkingMinutes?: number | null;
};

export type ExportExtras = {
  weatherSummary?: ExportWeatherDay[] | null;
  distanceSummary?: ExportDistanceDay[] | null;
  budgetSummary?: BudgetSummary | null;
  checklist?: TripChecklist | null;
};

export function toExportTripFromPrivateTrip(
  trip: Trip,
  extras: ExportExtras = {}
): ExportTrip {
  return {
    destination: trip.destination,
    startDate: trip.startDate ?? null,
    days: trip.days,
    budgetAmount: trip.budgetAmount ?? null,
    budgetCurrency: trip.budgetCurrency ?? null,
    travelers: trip.travelers ?? null,
    interests: cloneStringArray(trip.interests),
    pace: trip.pace ?? null,
    accommodation: sanitizeAccommodation(trip.accommodation),
    route: sanitizeRoute(trip.route),
    itinerary: sanitizeItinerary(trip.itinerary),
    weatherSummary: extras.weatherSummary ?? null,
    distanceSummary: extras.distanceSummary ?? null,
    budgetSummary: extras.budgetSummary ?? null,
    checklist: sanitizeChecklist(extras.checklist),
    source: "private"
  };
}

export function toExportTripFromPublicTrip(
  trip: PublicTrip,
  extras: ExportExtras = {}
): ExportTrip {
  return {
    destination: trip.destination,
    startDate: trip.startDate ?? null,
    days: trip.days,
    // The private trip budget is never exposed on the public share. Item-level
    // costs (which carry their own currency) remain in the shared itinerary; the
    // itinerary currency is used only as a display fallback for those costs.
    budgetAmount: null,
    budgetCurrency: trip.itinerary?.currency ?? null,
    travelers: trip.travelers ?? null,
    interests: cloneStringArray(trip.interests),
    pace: trip.pace ?? null,
    accommodation: null,
    route: sanitizeRoute(trip.route),
    itinerary: sanitizeItinerary(trip.itinerary),
    weatherSummary: extras.weatherSummary ?? null,
    distanceSummary: extras.distanceSummary ?? null,
    budgetSummary: null,
    checklist: null,
    source: "public"
  };
}

export function toExportWeatherSummary(
  forecast: WeatherForecast | null | undefined
): ExportWeatherDay[] | null {
  if (!forecast?.days?.length) {
    return null;
  }

  return forecast.days.map((day, index) => ({
    dayNumber: index + 1,
    date: day.date ?? null,
    summary: day.summary || day.condition || null,
    temperatureMinC: day.temperatureMinC ?? null,
    temperatureMaxC: day.temperatureMaxC ?? null,
    precipitationChance: day.precipitationChance ?? null
  }));
}

export function toExportDistanceSummary(
  fallbackSummaries: DayDistanceSummary[] | null | undefined,
  routeEstimatesByDay: Record<number, RouteEstimate | null> = {}
): ExportDistanceDay[] | null {
  if (!fallbackSummaries?.length) {
    return null;
  }

  const summaries = fallbackSummaries
    .map((summary) => {
      const routeEstimate = routeEstimatesByDay[summary.dayNumber] ?? null;
      return {
        dayNumber: summary.dayNumber,
        distanceKm: routeEstimate?.distanceKm ?? summary.straightLineDistanceKm,
        walkingMinutes: routeEstimate?.durationMinutes ?? summary.estimatedWalkingMinutes
      };
    })
    .filter((summary) => {
      const distance = summary.distanceKm ?? 0;
      const walkingMinutes = summary.walkingMinutes ?? 0;
      return distance > 0 || walkingMinutes > 0;
    });

  return summaries.length > 0 ? summaries : null;
}

function sanitizeItinerary(itinerary: Itinerary | null | undefined): Itinerary | null {
  if (!itinerary) {
    return null;
  }

  return {
    destination: itinerary.destination,
    summary: itinerary.summary,
    travelers: itinerary.travelers,
    pace: itinerary.pace,
    currency: itinerary.currency,
    totalBudget: itinerary.totalBudget ?? null,
    days: (itinerary.days ?? []).map(sanitizeDay)
  };
}

function sanitizeDay(day: ItineraryDay, index: number): ItineraryDay {
  return {
    day: day.day || index + 1,
    date: day.date ?? null,
    title: day.title ?? "",
    primaryStopId: day.primaryStopId ?? null,
    locationName: day.locationName ?? null,
    transferDay: Boolean(day.transferDay),
    items: (day.items ?? []).map(sanitizeItem)
  };
}

function sanitizeItem(item: ItineraryItem): ItineraryItem {
  return {
    time: item.time ?? "",
    endTime: item.endTime ?? null,
    type: item.type ?? "activity",
    category: item.category ?? null,
    transportMode: item.transportMode ?? null,
    durationMinutes: item.durationMinutes ?? null,
    name: item.name ?? "",
    description: item.description ?? null,
    note: item.note ?? null,
    transfer: item.transfer
      ? {
          legId: item.transfer.legId ?? null,
          from: item.transfer.from,
          to: item.transfer.to,
          mode: item.transfer.mode,
          estimatedDurationMinutes: item.transfer.estimatedDurationMinutes ?? null,
          estimatedDistanceKm: item.transfer.estimatedDistanceKm ?? null,
          estimatedCost: item.transfer.estimatedCost ?? null,
          bookingRequired: Boolean(item.transfer.bookingRequired),
          notes: item.transfer.notes ?? null,
          warnings: item.transfer.warnings ?? []
        }
      : null,
    estimatedCost: item.estimatedCost ?? null,
    place: sanitizePlace(item.place)
  };
}

function sanitizeRoute(route: TripRoute | null | undefined): TripRoute | null {
  if (!route?.stops?.length) {
    return null;
  }

  return {
    origin: route.origin
      ? {
          name: route.origin.name ?? null,
          country: route.origin.country ?? null,
          coordinates: route.origin.coordinates ?? null
        }
      : null,
    returnToOrigin: Boolean(route.returnToOrigin),
    stops: route.stops.map(sanitizeRouteStop),
    legs: (route.legs ?? []).map(sanitizeRouteLeg),
    preferences: route.preferences
      ? {
          preferredModes: route.preferences.preferredModes ?? [],
          avoidModes: route.preferences.avoidModes ?? [],
          carAvailable: Boolean(route.preferences.carAvailable),
          maxTransferHoursPerDay: route.preferences.maxTransferHoursPerDay ?? null,
          tripStyles: route.preferences.tripStyles ?? []
        }
      : undefined
  };
}

function sanitizeRouteStop(stop: TripRouteStop): TripRouteStop {
  return {
    id: stop.id,
    destination: stop.destination,
    city: stop.city ?? null,
    country: stop.country ?? null,
    arrivalDate: stop.arrivalDate ?? null,
    departureDate: stop.departureDate ?? null,
    nights: stop.nights ?? null,
    coordinates: stop.coordinates ?? null,
    accommodationHint: stop.accommodationHint ?? null,
    notes: null
  };
}

function sanitizeRouteLeg(leg: TripRouteLeg): TripRouteLeg {
  return {
    id: leg.id,
    fromStopId: leg.fromStopId,
    toStopId: leg.toStopId,
    fromName: leg.fromName ?? null,
    toName: leg.toName ?? null,
    mode: leg.mode,
    departureDate: leg.departureDate ?? null,
    estimatedDurationMinutes: leg.estimatedDurationMinutes ?? null,
    estimatedDistanceKm: leg.estimatedDistanceKm ?? null,
    estimatedCost: leg.estimatedCost ?? null,
    notes: null,
    providerMetadata: null
  };
}

function sanitizeAccommodation(
  accommodation: TripAccommodation | null | undefined
): TripAccommodation | null {
  if (!accommodation) {
    return null;
  }

  return {
    name: accommodation.name,
    type: accommodation.type,
    address: accommodation.address ?? null,
    place: sanitizePlace(accommodation.place),
    checkInDate: accommodation.checkInDate ?? null,
    checkOutDate: accommodation.checkOutDate ?? null,
    estimatedCost: accommodation.estimatedCost ?? null,
    notes: accommodation.notes ?? null
  };
}

function sanitizeChecklist(checklist: TripChecklist | null | undefined): TripChecklist | null {
  if (!checklist) {
    return null;
  }
  return {
    ...checklist,
    summary: checklist.summary ?? null,
    items: (checklist.items ?? []).map(sanitizeChecklistItem).sort((a, b) => {
      if (a.sortOrder !== b.sortOrder) {
        return a.sortOrder - b.sortOrder;
      }
      return a.title.localeCompare(b.title);
    }),
    metadata: checklist.metadata ?? {}
  };
}

function sanitizeChecklistItem(item: TripChecklistItem): TripChecklistItem {
  return {
    ...item,
    description: item.description ?? null,
    quantity: item.quantity ?? null,
    assignedToUserId: item.assignedToUserId ?? null,
    assignedToDisplayName: item.assignedToDisplayName ?? null,
    dueDate: item.dueDate ?? null,
    checkedAt: item.checkedAt ?? null,
    checkedByUserId: item.checkedByUserId ?? null,
    reason: item.reason ?? null,
    relatedDayNumber: item.relatedDayNumber ?? null,
    relatedItemIndex: item.relatedItemIndex ?? null,
    relatedItemId: item.relatedItemId ?? null,
    metadata: item.metadata ?? {}
  };
}

function sanitizePlace(place: Place | null | undefined): Place | null {
  if (!place) {
    return null;
  }

  return {
    provider: place.provider,
    providerPlaceId: "",
    name: place.name,
    address: place.address,
    latitude: place.latitude ?? null,
    longitude: place.longitude ?? null,
    rating: place.rating ?? null,
    ratingCount: place.ratingCount ?? null,
    mapUrl: place.mapUrl ?? null,
    category: place.category ?? null,
    website: place.website ?? null,
    openingHours: place.openingHours ?? null
  };
}

function cloneStringArray(values: string[] | null | undefined): string[] {
  return Array.isArray(values) ? values.filter(Boolean).map(String) : [];
}
