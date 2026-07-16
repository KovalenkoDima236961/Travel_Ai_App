import { describe, expect, it } from "vitest";
import type { TripRoute } from "@/entities/route/model";
import { validateDraftRoute } from "@/lib/route-builder/route-validation";

describe("route validation", () => {
  it("finds duplicate stops and missing transport options", () => {
    const route = sampleRoute();
    route.stops[1] = { ...route.stops[1], destination: " Vienna " };
    route.legs![0] = { ...route.legs![0], selectedTransportOption: null };
    const issues = validateDraftRoute(route, null, { totalDays: 5 });

    expect(issues.some((issue) => issue.id.startsWith("duplicate_stop"))).toBe(true);
    expect(issues.some((issue) => issue.id === "transport_missing_option:leg_1")).toBe(true);
  });

  it("finds a missing leg mode from untrusted route JSON", () => {
    const route = sampleRoute();
    route.legs![0].mode = "" as never;
    expect(validateDraftRoute(route).some((issue) => issue.id === "missing_leg_mode:leg_1")).toBe(true);
  });

  it("finds route-itinerary mismatches", () => {
    const issues = validateDraftRoute(sampleRoute(), {
      days: [
        { day: 1, title: "Vienna", primaryStopId: "removed-stop", locationName: "Vienna", items: [] }
      ]
    });
    expect(issues.some((issue) => issue.id === "itinerary_route_stop_mismatch:1")).toBe(true);
  });

  it("finds activities scheduled during selected transport", () => {
    const route = sampleRoute();
    route.legs![0].selectedTransportOption = {
      id: "train-1",
      mode: "train",
      provider: "national_rail",
      departureDate: "2026-09-01",
      departureTime: "10:00",
      arrivalDate: "2026-09-01",
      arrivalTime: "12:00",
      confidence: "high"
    };
    const issues = validateDraftRoute(route, {
      days: [
        {
          day: 1,
          date: "2026-09-01",
          title: "Vienna",
          primaryStopId: "vienna",
          items: [{ time: "11:00", type: "activity", name: "Museum visit" }]
        }
      ]
    });
    expect(issues.some((issue) => issue.id.startsWith("activity_during_transport"))).toBe(true);
  });
});

function sampleRoute(): TripRoute {
  return {
    origin: { name: "Bratislava" },
    stops: [
      { id: "vienna", destination: "Vienna", nights: 2 },
      { id: "salzburg", destination: "Salzburg", nights: 2 }
    ],
    legs: [
      { id: "leg_1", fromStopId: "origin", toStopId: "vienna", mode: "train" },
      { id: "leg_2", fromStopId: "vienna", toStopId: "salzburg", mode: "train" }
    ]
  };
}
