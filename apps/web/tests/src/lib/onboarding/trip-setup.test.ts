import { describe, expect, it } from "vitest";
import { buildTripSetupChecklist, completedTripSetupCount } from "@/lib/onboarding/trip-setup";
import type { Trip } from "@/entities/trip/model";

describe("first-trip setup checklist", () => {
  it("computes setup completion from existing trip data", () => {
    const items = buildTripSetupChecklist({
      trip: completeTrip(),
      checklistExists: true,
      collaboratorCount: 1,
      healthLoaded: true
    });
    expect(items.every((item) => item.status === "complete")).toBe(true);
    expect(completedTripSetupCount(items)).toBe(7);
  });

  it("keeps collaboration optional and flags critical health", () => {
    const trip = { ...completeTrip(), itinerary: null, budgetAmount: undefined, startDate: null };
    const items = buildTripSetupChecklist({ trip, healthLoaded: true, healthHasCriticalIssues: true });
    expect(items.find((item) => item.id === "collaborators")?.status).toBe("optional");
    expect(items.find((item) => item.id === "health")?.status).toBe("needs_attention");
    expect(items.find((item) => item.id === "itinerary")?.href).toBe("/trips/trip-1?tab=itinerary");
  });

  it("requires modes for every multi-destination route leg", () => {
    const trip = completeTrip();
    trip.route!.legs = [];
    expect(buildTripSetupChecklist({ trip }).find((item) => item.id === "route_transport")?.status).toBe("recommended");
  });
});

function completeTrip(): Trip {
  return {
    id: "trip-1",
    destination: "Austria route",
    startDate: "2026-08-01",
    days: 4,
    budgetAmount: 800,
    budgetCurrency: "EUR",
    travelers: 2,
    interests: ["culture"],
    pace: "balanced",
    status: "COMPLETED",
    tripType: "multi_destination",
    itinerary: { days: [{ day: 1, title: "Vienna", items: [] }] },
    route: {
      stops: [{ id: "a", destination: "Vienna" }, { id: "b", destination: "Salzburg" }],
      legs: [{ id: "ab", fromStopId: "a", toStopId: "b", mode: "train" }]
    },
    itineraryRevision: 1,
    createdAt: "2026-01-01T00:00:00Z",
    updatedAt: "2026-01-01T00:00:00Z"
  };
}
