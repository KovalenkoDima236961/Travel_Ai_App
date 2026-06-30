import { describe, expect, it } from "vitest";
import {
  toExportDistanceSummary,
  toExportTripFromPrivateTrip,
  toExportTripFromPublicTrip,
  toExportWeatherSummary
} from "@/lib/export/trip-export-adapter";
import type { DayDistanceSummary } from "@/lib/itinerary/distance-utils";
import type { PublicTrip } from "@/types/share";
import type { Trip } from "@/types/trip";
import type { WeatherForecast } from "@/types/weather";

describe("trip export adapters", () => {
  it("sanitizes private trips and strips private/internal fields", () => {
    const rawTrip = {
      id: "trip-1",
      userId: "user-123",
      ownerEmail: "owner@example.com",
      preferences: { avoid: ["private"] },
      itineraryVersions: [{ id: "version-1" }],
      destination: "Rome",
      startDate: "2026-08-10",
      days: 2,
      budgetAmount: 500,
      budgetCurrency: "EUR",
      travelers: 2,
      interests: ["food"],
      pace: "balanced",
      status: "COMPLETED",
      itineraryRevision: 3,
      createdAt: "2026-01-01T00:00:00Z",
      updatedAt: "2026-01-01T00:00:00Z",
      itinerary: {
        generatedAt: "2026-01-01T00:00:00Z",
        source: "internal",
        days: [
          {
            day: 1,
            title: "Day",
            items: [
              {
                time: "09:00",
                type: "place",
                name: "Colosseum",
                placeEnrichment: {
                  status: "matched",
                  query: "private debug query",
                  provider: "internal-provider"
                },
                place: {
                  provider: "mock",
                  providerPlaceId: "provider-secret-id",
                  name: "Colosseum",
                  address: "Piazza del Colosseo",
                  mapUrl: "https://maps.example.com/colosseum"
                }
              }
            ]
          }
        ]
      }
    } as Trip & Record<string, unknown>;

    const exportTrip = toExportTripFromPrivateTrip(rawTrip);
    const serialized = JSON.stringify(exportTrip);

    expect(exportTrip.source).toBe("private");
    expect(exportTrip.itinerary?.days[0]?.items[0]?.place?.providerPlaceId).toBe("");
    expect(serialized).not.toContain("user-123");
    expect(serialized).not.toContain("owner@example.com");
    expect(serialized).not.toContain("preferences");
    expect(serialized).not.toContain("version-1");
    expect(serialized).not.toContain("private debug query");
    expect(serialized).not.toContain("provider-secret-id");
  });

  it("normalizes public trips from public data only", () => {
    const trip: PublicTrip = {
      destination: "Paris",
      startDate: null,
      days: 3,
      status: "COMPLETED",
      itinerary: { days: [], currency: "EUR" }
    };

    const exported = toExportTripFromPublicTrip(trip);
    expect(exported).toMatchObject({
      destination: "Paris",
      startDate: null,
      source: "public"
    });
    // The private trip budget is never exposed on public exports.
    expect(exported.budgetAmount).toBeNull();
  });
});

describe("export extras", () => {
  it("normalizes weather forecasts", () => {
    const forecast: WeatherForecast = {
      destination: "Rome",
      days: [
        {
          date: "2026-08-10",
          condition: "sunny",
          temperatureMinC: 20,
          temperatureMaxC: 31,
          precipitationChance: 10,
          windSpeedKph: 12,
          summary: "Warm and clear"
        }
      ]
    };

    expect(toExportWeatherSummary(forecast)).toEqual([
      {
        dayNumber: 1,
        date: "2026-08-10",
        summary: "Warm and clear",
        temperatureMinC: 20,
        temperatureMaxC: 31,
        precipitationChance: 10
      }
    ]);
  });

  it("prefers route distance summaries when available", () => {
    const fallback: DayDistanceSummary[] = [
      {
        dayNumber: 1,
        mappedStops: 2,
        segmentCount: 1,
        straightLineDistanceKm: 1,
        estimatedWalkingMinutes: 12,
        exceedsPreference: false,
        segments: []
      }
    ];

    expect(
      toExportDistanceSummary(fallback, {
        1: {
          mode: "walking",
          provider: "mock",
          distanceKm: 1.4,
          durationMinutes: 17,
          segments: []
        }
      })
    ).toEqual([{ dayNumber: 1, distanceKm: 1.4, walkingMinutes: 17 }]);
  });
});
