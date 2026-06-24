import type { DayDistanceSummary } from "@/lib/itinerary/distance-utils";
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
  itinerary?: Itinerary | null;
  weatherSummary?: ExportWeatherDay[] | null;
  distanceSummary?: ExportDistanceDay[] | null;
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
    itinerary: sanitizeItinerary(trip.itinerary),
    weatherSummary: extras.weatherSummary ?? null,
    distanceSummary: extras.distanceSummary ?? null,
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
    budgetAmount: trip.budgetAmount ?? null,
    budgetCurrency: trip.budgetCurrency ?? null,
    travelers: trip.travelers ?? null,
    interests: cloneStringArray(trip.interests),
    pace: trip.pace ?? null,
    itinerary: sanitizeItinerary(trip.itinerary),
    weatherSummary: extras.weatherSummary ?? null,
    distanceSummary: extras.distanceSummary ?? null,
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
    place: item.place
      ? {
          provider: item.place.provider,
          providerPlaceId: "",
          name: item.place.name,
          address: item.place.address,
          rating: item.place.rating ?? null,
          ratingCount: item.place.ratingCount ?? null,
          mapUrl: item.place.mapUrl ?? null,
          category: item.place.category ?? null,
          website: item.place.website ?? null
        }
      : null
  };
}

function cloneStringArray(values: string[] | null | undefined): string[] {
  return Array.isArray(values) ? values.filter(Boolean).map(String) : [];
}
