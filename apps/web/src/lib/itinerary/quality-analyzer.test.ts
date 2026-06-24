import { describe, expect, it } from "vitest";
import { analyzeItineraryQuality } from "@/lib/itinerary/quality-analyzer";
import type { DayDistanceSummary } from "@/lib/itinerary/distance-utils";
import type { OpeningHoursInterval, Place } from "@/types/place";
import type { RouteEstimate } from "@/types/route";
import type { Itinerary, ItineraryItem } from "@/types/trip";
import type { WeatherForecast } from "@/types/weather";

const mondayOpenHours: OpeningHoursInterval[] = [
  { dayOfWeek: 1, open: "09:00", close: "18:00" }
];

function place(name: string, overrides: Partial<Place> = {}): Place {
  return {
    provider: "mock",
    providerPlaceId: `mock-${name.toLowerCase().replace(/\s+/g, "-")}`,
    name,
    address: `${name} address`,
    latitude: 41.8902,
    longitude: 12.4922,
    ...overrides
  };
}

function item(overrides: Partial<ItineraryItem> = {}): ItineraryItem {
  return {
    time: "10:00",
    type: "place",
    name: "Museum",
    place: place("Museum"),
    ...overrides
  };
}

function itinerary(items: ItineraryItem[] = [item()]): Itinerary {
  return {
    days: [
      {
        day: 1,
        title: "Day 1",
        items
      }
    ]
  };
}

function fallbackSummary(distanceKm: number): DayDistanceSummary {
  return {
    dayNumber: 1,
    mappedStops: 2,
    segmentCount: 1,
    straightLineDistanceKm: distanceKm,
    estimatedWalkingMinutes: Math.round(distanceKm * 12),
    exceedsPreference: false,
    maxWalkingKmPerDay: 8,
    segments: []
  };
}

function routeEstimate(distanceKm: number): RouteEstimate {
  return {
    mode: "walking",
    provider: "mock",
    distanceKm,
    durationMinutes: Math.round(distanceKm * 12),
    segments: []
  };
}

const rainyWeather: WeatherForecast = {
  destination: "Rome",
  provider: "mock",
  days: [
    {
      date: "2026-08-10",
      condition: "rain",
      temperatureMinC: 22,
      temperatureMaxC: 28,
      precipitationChance: 70,
      windSpeedKph: 12,
      summary: "Rain likely"
    }
  ]
};

const hotWeather: WeatherForecast = {
  destination: "Rome",
  provider: "mock",
  days: [
    {
      date: "2026-08-10",
      condition: "hot",
      temperatureMinC: 25,
      temperatureMaxC: 34,
      precipitationChance: 10,
      windSpeedKph: 12,
      summary: "Hot"
    }
  ]
};

describe("analyzeItineraryQuality", () => {
  it("detects walking distance above preference using route estimates first", () => {
    const summary = analyzeItineraryQuality({
      itinerary: itinerary(),
      fallbackDistanceSummaries: [fallbackSummary(3)],
      routeEstimatesByDay: { 1: routeEstimate(9.4) },
      maxWalkingKmPerDay: 8
    });

    const issue = summary.byDay[1]?.find((currentIssue) => currentIssue.type === "walking_distance_high");
    expect(issue?.severity).toBe("warning");
    expect(issue?.message).toContain("9.4 km");
    expect(issue?.metadata?.estimateSource).toBe("route");
  });

  it("marks walking distance critical when it is far above preference", () => {
    const summary = analyzeItineraryQuality({
      itinerary: itinerary(),
      fallbackDistanceSummaries: [fallbackSummary(13)],
      maxWalkingKmPerDay: 8
    });

    expect(summary.critical).toBe(1);
    expect(summary.byDay[1]?.[0]?.type).toBe("walking_distance_high");
  });

  it("detects closed place items from opening hours", () => {
    const summary = analyzeItineraryQuality({
      itinerary: itinerary([
        item({
          time: "20:00",
          name: "Late Museum",
          place: place("Late Museum", { openingHours: mondayOpenHours })
        })
      ]),
      tripStartDate: "2026-08-10"
    });

    const issue = summary.itemIssues.find((currentIssue) => currentIssue.type === "place_may_be_closed");
    expect(issue?.id).toBe("day-1-item-0-place-closed");
    expect(issue?.message).toContain("Late Museum may be closed");
  });

  it("detects rainy outdoor-heavy days", () => {
    const summary = analyzeItineraryQuality({
      itinerary: itinerary([
        item({ name: "Garden walk", type: "walking" }),
        item({ name: "City square", type: "landmark" }),
        item({ name: "Dinner", type: "food" })
      ]),
      tripStartDate: "2026-08-10",
      weatherForecast: rainyWeather
    });

    expect(summary.byDay[1]?.some((issue) => issue.type === "weather_rain_outdoor")).toBe(true);
  });

  it("detects hot outdoor-heavy days", () => {
    const summary = analyzeItineraryQuality({
      itinerary: itinerary([
        item({ name: "Park", type: "park" }),
        item({ name: "Viewpoint", type: "viewpoint" })
      ]),
      tripStartDate: "2026-08-10",
      weatherForecast: hotWeather
    });

    expect(summary.byDay[1]?.some((issue) => issue.type === "weather_heat_outdoor")).toBe(true);
  });

  it("detects pending place reviews", () => {
    const summary = analyzeItineraryQuality({
      itinerary: itinerary([
        item({
          name: "Auto place",
          placeEnrichment: { status: "matched", confidence: 0.9 }
        })
      ])
    });

    expect(summary.itemIssues.some((issue) => issue.type === "place_match_pending_review")).toBe(true);
  });

  it("detects low-confidence place matches", () => {
    const summary = analyzeItineraryQuality({
      itinerary: itinerary([
        item({
          name: "Maybe place",
          placeEnrichment: {
            status: "matched",
            confidence: 0.6,
            reviewStatus: "accepted"
          }
        })
      ])
    });

    const issue = summary.itemIssues.find(
      (currentIssue) => currentIssue.type === "place_match_low_confidence"
    );
    expect(issue?.severity).toBe("warning");
  });

  it("detects no confident place matches", () => {
    const summary = analyzeItineraryQuality({
      itinerary: itinerary([
        item({
          name: "Ambiguous item",
          place: null,
          placeEnrichment: { status: "no_match", query: "Ambiguous item" }
        })
      ])
    });

    expect(summary.itemIssues.some((issue) => issue.type === "place_no_confident_match")).toBe(true);
  });

  it("detects missing coordinates only for enriched map-ready items", () => {
    const summary = analyzeItineraryQuality({
      itinerary: itinerary([
        item({
          name: "Mapped candidate",
          type: "museum",
          place: place("No coordinate museum", { latitude: null, longitude: null }),
          placeEnrichment: {
            status: "matched",
            confidence: 0.95,
            reviewStatus: "accepted"
          }
        }),
        item({
          name: "Plain old item",
          type: "museum",
          place: null
        })
      ])
    });

    expect(summary.itemIssues.filter((issue) => issue.type === "missing_place_coordinates")).toHaveLength(1);
  });

  it("handles missing weather and route estimates", () => {
    const summary = analyzeItineraryQuality({
      itinerary: itinerary(),
      weatherForecast: null,
      routeEstimatesByDay: {}
    });

    expect(summary.total).toBe(0);
  });

  it("returns no issues for a clean itinerary", () => {
    const summary = analyzeItineraryQuality({
      itinerary: itinerary([
        item({
          time: "10:00",
          type: "museum",
          name: "Open Museum",
          place: place("Open Museum", { openingHours: mondayOpenHours }),
          placeEnrichment: {
            status: "matched",
            confidence: 0.95,
            reviewStatus: "accepted"
          }
        })
      ]),
      tripStartDate: "2026-08-10",
      weatherForecast: {
        destination: "Rome",
        days: [
          {
            date: "2026-08-10",
            condition: "clear",
            temperatureMinC: 18,
            temperatureMaxC: 26,
            precipitationChance: 10,
            windSpeedKph: 8,
            summary: "Clear"
          }
        ]
      },
      fallbackDistanceSummaries: [fallbackSummary(2)],
      maxWalkingKmPerDay: 8
    });

    expect(summary.total).toBe(0);
  });
});
