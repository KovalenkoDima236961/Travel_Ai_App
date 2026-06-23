import { describe, expect, it } from "vitest";
import {
  getRouteStopsByDay,
  getRouteStopsForDay,
  routeStopsCacheKey
} from "@/lib/itinerary/route-estimate-utils";
import type { Place } from "@/types/place";
import type { Itinerary, ItineraryDay, ItineraryItem } from "@/types/trip";

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

function item(name: string, attachedPlace?: Place | null): ItineraryItem {
  return { time: "09:00", type: "place", name, place: attachedPlace ?? null };
}

const A = { latitude: 41.8902, longitude: 12.4922 };
const B = { latitude: 41.9009, longitude: 12.4833 };
const C = { latitude: 41.8986, longitude: 12.4768 };

describe("getRouteStopsForDay", () => {
  it("ignores items without coordinates", () => {
    const day: ItineraryDay = {
      day: 1,
      title: "Day 1",
      items: [
        item("Breakfast", null),
        item("Colosseum", place("Colosseum", A)),
        item("Free time", place("No coords place")),
        item("Trevi Fountain", place("Trevi Fountain", B))
      ]
    };

    const stops = getRouteStopsForDay(day);
    expect(stops).toHaveLength(2);
    expect(stops.map((stop) => stop.name)).toEqual(["Colosseum", "Trevi Fountain"]);
  });

  it("preserves itinerary item order", () => {
    const day: ItineraryDay = {
      day: 1,
      title: "Day 1",
      items: [
        item("First", place("First", A)),
        item("Second", place("Second", B)),
        item("Third", place("Third", C))
      ]
    };

    expect(getRouteStopsForDay(day).map((stop) => stop.name)).toEqual([
      "First",
      "Second",
      "Third"
    ]);
  });

  it("prefers the place name over the item name", () => {
    const day: ItineraryDay = {
      day: 1,
      title: "Day 1",
      items: [
        item("Morning visit", place("Colosseum", A)),
        item("Afternoon visit", place("Trevi Fountain", B))
      ]
    };

    expect(getRouteStopsForDay(day).map((stop) => stop.name)).toEqual([
      "Colosseum",
      "Trevi Fountain"
    ]);
  });

  it("falls back to the item name when the place has no name", () => {
    const namelessPlace = { ...place("placeholder", A), name: "" };
    const day: ItineraryDay = {
      day: 1,
      title: "Day 1",
      items: [item("Item label", namelessPlace), item("Second", place("Second", B))]
    };

    expect(getRouteStopsForDay(day)[0].name).toBe("Item label");
  });

  it("ignores items with out-of-range coordinates", () => {
    const day: ItineraryDay = {
      day: 1,
      title: "Day 1",
      items: [
        item("Bad", place("Bad", { latitude: 200, longitude: 12 })),
        item("Good A", place("Good A", A)),
        item("Good B", place("Good B", B))
      ]
    };

    expect(getRouteStopsForDay(day).map((stop) => stop.name)).toEqual(["Good A", "Good B"]);
  });
});

describe("getRouteStopsByDay", () => {
  it("only includes days with at least two mapped stops", () => {
    const itinerary: Itinerary = {
      days: [
        { day: 1, title: "One stop", items: [item("Solo", place("Solo", A))] },
        {
          day: 2,
          title: "Two stops",
          items: [item("A", place("A", A)), item("B", place("B", B))]
        },
        { day: 3, title: "No stops", items: [item("Walk", null)] }
      ]
    };

    const result = getRouteStopsByDay(itinerary);
    expect(result).toHaveLength(1);
    expect(result[0].dayNumber).toBe(2);
    expect(result[0].stops).toHaveLength(2);
  });

  it("falls back to the 1-based index when day numbers are missing", () => {
    const itinerary: Itinerary = {
      days: [
        {
          day: 0,
          title: "Untitled",
          items: [item("A", place("A", A)), item("B", place("B", B))]
        }
      ]
    };

    expect(getRouteStopsByDay(itinerary)[0].dayNumber).toBe(1);
  });
});

describe("routeStopsCacheKey", () => {
  it("is stable for identical stops and differs when coordinates change", () => {
    const stopsA = getRouteStopsForDay({
      day: 1,
      title: "Day 1",
      items: [item("A", place("A", A)), item("B", place("B", B))]
    });
    const stopsACopy = getRouteStopsForDay({
      day: 1,
      title: "Day 1",
      items: [item("A", place("A", A)), item("B", place("B", B))]
    });
    const stopsDifferent = getRouteStopsForDay({
      day: 1,
      title: "Day 1",
      items: [item("A", place("A", A)), item("C", place("C", C))]
    });

    expect(routeStopsCacheKey(stopsA)).toBe(routeStopsCacheKey(stopsACopy));
    expect(routeStopsCacheKey(stopsA)).not.toBe(routeStopsCacheKey(stopsDifferent));
  });
});
