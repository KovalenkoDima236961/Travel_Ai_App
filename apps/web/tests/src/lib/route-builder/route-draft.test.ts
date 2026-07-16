import { describe, expect, it } from "vitest";
import type { TripRoute } from "@/entities/route/model";
import {
  createRouteDraft,
  reorderRouteStops,
  updateRouteStop
} from "@/lib/route-builder/route-draft";

describe("route draft", () => {
  it("detects dirty state and destructive route impacts", () => {
    const route = sampleRoute();
    const reordered = reorderRouteStops(route, 0, 1);
    const state = createRouteDraft(route, reordered, {
      days: [{ day: 1, title: "Vienna", primaryStopId: "vienna", items: [] }]
    });

    expect(state.dirty).toBe(true);
    expect(state.stopOrderChanged).toBe(true);
    expect(state.removedTransportOptionCount).toBe(2);
    expect(state.itineraryImpact).toBe(true);
    expect(state.budgetImpact).toBe(true);
    expect(state.reminderImpact).toBe(true);
  });

  it("preserves unchanged adjacent legs and their selected transport", () => {
    const route = sampleRouteWithFourStops();
    const reordered = reorderRouteStops(route, 0, 1);
    const unchanged = reordered.legs?.find(
      (leg) => leg.fromStopId === "salzburg" && leg.toStopId === "hallstatt"
    );

    expect(unchanged?.id).toBe("leg_4");
    expect(unchanged?.selectedTransportOption?.id).toBe("option_4");
    expect(reordered.legs?.find((leg) => leg.toStopId === "vienna")?.selectedTransportOption).toBeNull();
  });

  it("marks connected selected transport stale when a stop date changes", () => {
    const route = sampleRoute();
    const updated = updateRouteStop(route, "vienna", {
      ...route.stops[0],
      arrivalDate: "2026-09-02"
    });
    const connected = updated.legs?.filter(
      (leg) => leg.fromStopId === "vienna" || leg.toStopId === "vienna"
    );

    expect(connected).toHaveLength(2);
    expect(connected?.every((leg) => leg.providerMetadata?.stale === true)).toBe(true);
    expect(createRouteDraft(route, updated).staleTransportOptionCount).toBe(2);
  });

  it("keeps an untouched route clean", () => {
    expect(createRouteDraft(sampleRoute()).dirty).toBe(false);
  });
});

function sampleRoute(): TripRoute {
  return {
    origin: { name: "Bratislava" },
    stops: [
      { id: "vienna", destination: "Vienna", arrivalDate: "2026-09-01", nights: 2 },
      { id: "salzburg", destination: "Salzburg", arrivalDate: "2026-09-03", nights: 2 }
    ],
    legs: [
      selectedLeg("leg_1", "origin", "vienna", "option_1"),
      selectedLeg("leg_2", "vienna", "salzburg", "option_2")
    ]
  };
}

function sampleRouteWithFourStops(): TripRoute {
  const route = sampleRoute();
  return {
    ...route,
    stops: [
      { id: "brno", destination: "Brno", nights: 1 },
      ...route.stops,
      { id: "hallstatt", destination: "Hallstatt", nights: 1 }
    ],
    legs: [
      selectedLeg("leg_1", "origin", "brno", "option_1"),
      selectedLeg("leg_2", "brno", "vienna", "option_2"),
      selectedLeg("leg_3", "vienna", "salzburg", "option_3"),
      selectedLeg("leg_4", "salzburg", "hallstatt", "option_4")
    ]
  };
}

function selectedLeg(id: string, fromStopId: string, toStopId: string, optionId: string) {
  return {
    id,
    fromStopId,
    toStopId,
    mode: "train" as const,
    selectedTransportOption: {
      id: optionId,
      mode: "train" as const,
      provider: "national_rail",
      durationMinutes: 75,
      estimatedPrice: { amount: 20, currency: "EUR" },
      confidence: "high" as const
    }
  };
}
