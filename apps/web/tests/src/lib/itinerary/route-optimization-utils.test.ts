import { describe, expect, it } from "vitest";
import {
  applyOptimizedDayToItinerary,
  calculateDayMappedDistanceKm,
  canOptimizeDay,
  getMappedStopsForDay,
  nearestNeighborOrder,
  optimizeDayOrder,
  type OptimizableStop
} from "@/entities/itinerary/model/route-optimization-utils";
import type { Place } from "@/entities/place/model";
import type { Itinerary, ItineraryDay, ItineraryItem } from "@/entities/trip/model";

// Points spread along a single line of longitude so distances are predictable:
// A < B < C < D, each ~0.01 deg apart.
const A = { latitude: 0, longitude: 0 };
const B = { latitude: 0, longitude: 0.01 };
const C = { latitude: 0, longitude: 0.02 };
const D = { latitude: 0, longitude: 0.03 };

function place(name: string, coordinate?: { latitude: number; longitude: number }): Place {
  return {
    provider: "mock",
    providerPlaceId: `mock-${name.toLowerCase().replace(/\s+/g, "-")}`,
    name,
    address: `${name} address`,
    latitude: coordinate?.latitude,
    longitude: coordinate?.longitude,
    category: "attraction"
  };
}

function item(name: string, time: string, attachedPlace?: Place | null): ItineraryItem {
  return { time, type: "place", name, place: attachedPlace ?? null };
}

function stop(name: string, coordinate: { latitude: number; longitude: number }, index: number): OptimizableStop {
  return { originalIndex: index, time: "09:00", name, latitude: coordinate.latitude, longitude: coordinate.longitude };
}

// A day whose mapped places are visited in a zig-zag order (A -> D -> B -> C),
// with unmapped items interleaved. Nearest-neighbour from A should reorder the
// mapped places into A -> B -> C -> D.
function zigZagDay(): ItineraryDay {
  return {
    day: 2,
    title: "Old Town",
    items: [
      item("Breakfast", "08:00", null),
      item("Castle (A)", "09:00", place("Castle", A)),
      item("Tower (D)", "11:00", place("Tower", D)),
      item("Free time", "13:00", null),
      item("Cafe (B)", "14:00", place("Cafe", B)),
      item("Park (C)", "16:00", place("Park", C))
    ]
  };
}

describe("canOptimizeDay", () => {
  it("is false for a day with no mapped places", () => {
    const day: ItineraryDay = {
      day: 1,
      title: "No places",
      items: [item("Breakfast", "08:00", null), item("Walk", "10:00", null)]
    };
    expect(canOptimizeDay(day).canOptimize).toBe(false);
    expect(canOptimizeDay(day).reason).toMatch(/three mapped places/i);
  });

  it("is false for a day with only two mapped places", () => {
    const day: ItineraryDay = {
      day: 1,
      title: "Two places",
      items: [item("A", "09:00", place("A", A)), item("B", "10:00", place("B", B))]
    };
    expect(canOptimizeDay(day).canOptimize).toBe(false);
  });

  it("is true for a day with three mapped places", () => {
    const day: ItineraryDay = {
      day: 1,
      title: "Three places",
      items: [
        item("A", "09:00", place("A", A)),
        item("B", "10:00", place("B", B)),
        item("C", "11:00", place("C", C))
      ]
    };
    expect(canOptimizeDay(day).canOptimize).toBe(true);
    expect(canOptimizeDay(day).reason).toBeUndefined();
  });

  it("ignores items with invalid coordinates when counting mapped places", () => {
    const day: ItineraryDay = {
      day: 1,
      title: "Mixed",
      items: [
        item("A", "09:00", place("A", A)),
        item("B", "10:00", place("B", B)),
        item("Broken", "11:00", place("Broken", { latitude: 999, longitude: 12 }))
      ]
    };
    expect(canOptimizeDay(day).canOptimize).toBe(false);
  });
});

describe("getMappedStopsForDay", () => {
  it("returns only mapped items and preserves their original positions", () => {
    const stops = getMappedStopsForDay(zigZagDay());
    expect(stops.map((s) => s.originalIndex)).toEqual([1, 2, 4, 5]);
    expect(stops.map((s) => s.name)).toEqual(["Castle (A)", "Tower (D)", "Cafe (B)", "Park (C)"]);
  });
});

describe("nearestNeighborOrder", () => {
  it("keeps the first mapped stop as the starting point", () => {
    const ordered = nearestNeighborOrder([stop("A", A, 0), stop("D", D, 1), stop("B", B, 2), stop("C", C, 3)]);
    expect(ordered[0].name).toBe("A");
  });

  it("orders the remaining stops by nearest distance", () => {
    const ordered = nearestNeighborOrder([stop("A", A, 0), stop("D", D, 1), stop("B", B, 2), stop("C", C, 3)]);
    expect(ordered.map((s) => s.name)).toEqual(["A", "B", "C", "D"]);
  });

  it("does not mutate the input array", () => {
    const stops = [stop("A", A, 0), stop("D", D, 1), stop("B", B, 2)];
    const snapshot = stops.map((s) => s.name);
    nearestNeighborOrder(stops);
    expect(stops.map((s) => s.name)).toEqual(snapshot);
  });
});

describe("calculateDayMappedDistanceKm", () => {
  it("calculates distance walking mapped stops in their current item order", () => {
    const day = zigZagDay();
    // A -> D -> B -> C in degrees of longitude: 0.03 + 0.02 + 0.01 = 0.06.
    const optimal = calculateDayMappedDistanceKm({
      ...day,
      items: [
        item("A", "09:00", place("A", A)),
        item("B", "10:00", place("B", B)),
        item("C", "11:00", place("C", C)),
        item("D", "12:00", place("D", D))
      ]
    });
    const zigZag = calculateDayMappedDistanceKm(day);
    expect(zigZag).toBeGreaterThan(optimal);
  });

  it("ignores unmapped and invalid-coordinate items", () => {
    const withNoise: ItineraryDay = {
      day: 1,
      title: "Noise",
      items: [
        item("A", "09:00", place("A", A)),
        item("Note", "10:00", null),
        item("Broken", "11:00", place("Broken", { latitude: 999, longitude: 12 })),
        item("B", "12:00", place("B", B))
      ]
    };
    const clean: ItineraryDay = {
      day: 1,
      title: "Clean",
      items: [item("A", "09:00", place("A", A)), item("B", "12:00", place("B", B))]
    };
    expect(calculateDayMappedDistanceKm(withNoise)).toBeCloseTo(calculateDayMappedDistanceKm(clean), 9);
  });
});

describe("optimizeDayOrder", () => {
  it("does not mutate the original day", () => {
    const day = zigZagDay();
    const snapshot = JSON.parse(JSON.stringify(day));
    optimizeDayOrder(day);
    expect(day).toEqual(snapshot);
  });

  it("keeps unmapped items in their original positions", () => {
    const result = optimizeDayOrder(zigZagDay());
    const names = result.optimizedDay.items.map((i) => i.name);
    expect(names[0]).toBe("Breakfast");
    expect(names[3]).toBe("Free time");
  });

  it("reorders mapped places while preserving the original time slots", () => {
    const result = optimizeDayOrder(zigZagDay());
    const times = result.optimizedDay.items.map((i) => i.time);
    // Times stay tied to positions, not to the places that move into them.
    expect(times).toEqual(["08:00", "09:00", "11:00", "13:00", "14:00", "16:00"]);
    // Mapped positions [1, 2, 4, 5] now hold A, B, C, D in nearest-neighbour order.
    expect([1, 2, 4, 5].map((index) => result.optimizedDay.items[index].name)).toEqual([
      "Castle (A)",
      "Cafe (B)",
      "Park (C)",
      "Tower (D)"
    ]);
  });

  it("preserves place metadata on moved items", () => {
    const result = optimizeDayOrder(zigZagDay());
    // Position 2 now holds the Cafe (B); its place metadata must come along.
    const movedItem = result.optimizedDay.items[2];
    expect(movedItem.place?.providerPlaceId).toBe("mock-cafe");
    expect(movedItem.place?.longitude).toBe(B.longitude);
    expect(movedItem.place?.category).toBe("attraction");
  });

  it("returns a distance comparison where the optimized order is shorter", () => {
    const result = optimizeDayOrder(zigZagDay());
    expect(result.canOptimize).toBe(true);
    expect(result.optimizedDistanceKm).toBeLessThan(result.originalDistanceKm);
    expect(result.savedDistanceKm).toBeGreaterThan(0);
    expect(result.savedDistanceKm).toBeCloseTo(
      result.originalDistanceKm - result.optimizedDistanceKm,
      9
    );
  });

  it("returns a non-crashing, canOptimize=false result for too few mapped places", () => {
    const day: ItineraryDay = {
      day: 1,
      title: "Two places",
      items: [item("A", "09:00", place("A", A)), item("B", "10:00", place("B", B))]
    };
    const result = optimizeDayOrder(day);
    expect(result.canOptimize).toBe(false);
    expect(result.reason).toMatch(/three mapped places/i);
    expect(result.savedDistanceKm).toBe(0);
    expect(result.optimizedOrder.map((o) => o.name)).toEqual(result.originalOrder.map((o) => o.name));
  });
});

describe("applyOptimizedDayToItinerary", () => {
  function twoDayItinerary(): Itinerary {
    return {
      destination: "Test City",
      days: [
        {
          day: 1,
          title: "Day one",
          items: [item("Keep", "09:00", place("Keep", A))]
        },
        zigZagDay()
      ]
    };
  }

  it("replaces only the selected day and preserves other days by reference", () => {
    const itinerary = twoDayItinerary();
    const result = optimizeDayOrder(itinerary.days[1]);
    const updated = applyOptimizedDayToItinerary(itinerary, 2, result.optimizedDay);

    // Day 1 is untouched (same reference).
    expect(updated.days[0]).toBe(itinerary.days[0]);
    // Day 2 is the optimized day.
    expect(updated.days[1].items.map((i) => i.name)).toEqual(
      result.optimizedDay.items.map((i) => i.name)
    );
    // Day identity (number/title) is preserved.
    expect(updated.days[1].day).toBe(2);
    expect(updated.days[1].title).toBe("Old Town");
  });

  it("preserves top-level itinerary fields", () => {
    const itinerary = twoDayItinerary();
    const result = optimizeDayOrder(itinerary.days[1]);
    const updated = applyOptimizedDayToItinerary(itinerary, 2, result.optimizedDay);
    expect(updated.destination).toBe("Test City");
  });

  it("does not mutate the original itinerary", () => {
    const itinerary = twoDayItinerary();
    const snapshot = JSON.parse(JSON.stringify(itinerary));
    const result = optimizeDayOrder(itinerary.days[1]);
    applyOptimizedDayToItinerary(itinerary, 2, result.optimizedDay);
    expect(itinerary).toEqual(snapshot);
  });
});
