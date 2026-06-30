import type { DayDistanceSummary } from "@/lib/itinerary/distance-utils";
import type { TripAccommodation } from "@/types/accommodation";
import type { BudgetSummary } from "@/types/budget";
import type { Place } from "@/types/place";
import type { PublicTrip } from "@/types/share";
import type { RouteEstimate } from "@/types/route";
import type { Itinerary, ItineraryDay, ItineraryItem, Trip } from "@/types/trip";
import type { WeatherForecast } from "@/types/weather";

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
  itinerary?: Itinerary | null;
  weatherSummary?: ExportWeatherDay[] | null;
  distanceSummary?: ExportDistanceDay[] | null;
  budgetSummary?: BudgetSummary | null;
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
    itinerary: sanitizeItinerary(trip.itinerary),
    weatherSummary: extras.weatherSummary ?? null,
    distanceSummary: extras.distanceSummary ?? null,
    budgetSummary: extras.budgetSummary ?? null,
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
    itinerary: sanitizeItinerary(trip.itinerary),
    weatherSummary: extras.weatherSummary ?? null,
    distanceSummary: extras.distanceSummary ?? null,
    budgetSummary: null,
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
    title: day.title ?? "",
    items: (day.items ?? []).map(sanitizeItem)
  };
}

function sanitizeItem(item: ItineraryItem): ItineraryItem {
  return {
    time: item.time ?? "",
    type: item.type ?? "activity",
    name: item.name ?? "",
    note: item.note ?? null,
    estimatedCost: item.estimatedCost ?? null,
    place: sanitizePlace(item.place)
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
