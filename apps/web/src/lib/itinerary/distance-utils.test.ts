import { describe, expect, it } from "vitest";
import {
  estimateWalkingMinutes,
  formatDistanceKm,
  formatWalkingTime,
  getDayDistanceSummaries,
  getTripDistanceTotal,
  haversineDistanceKm
} from "@/lib/itinerary/distance-utils";
import type { Place } from "@/types/place";
import type { Itinerary, ItineraryItem } from "@/types/trip";

const colosseum = { latitude: 41.8902, longitude: 12.4922 };
const romanForum = { latitude: 41.8925, longitude: 12.4853 };
const treviFountain = { latitude: 41.9009, longitude: 12.4833 };

function place(name: string, coordinate?: { latitude: number; longitude: number }): Place {
  return {
    provider: "mock",
    providerPlaceId: `mock-${name.toLowerCase().replace(/\s+/g, "-")}`,
    name,
    address: `${name} address`,
    latitude: coordinate?.latitude,
    longitude: coordinate?.longitude
  };
}

function item(
  name: string,
  time: string,
  attachedPlace?: Place | null
): ItineraryItem {
  return { time, type: "place", name, place: attachedPlace ?? null };
}

describe("haversineDistanceKm", () => {
  it("returns about 0 for the same coordinate", () => {
    expect(haversineDistanceKm(colosseum, colosseum)).toBeCloseTo(0, 6);
  });

  it("returns a reasonable distance between nearby coordinates", () => {
    // Colosseum -> Roman Forum is a short walk, well under a kilometre.
    const distance = haversineDistanceKm(colosseum, romanForum);
    expect(distance).toBeGreaterThan(0.4);
    expect(distance).toBeLessThan(0.8);
  });

  it("approximates Colosseum to Trevi Fountain", () => {
    // Real straight-line distance is roughly 1.4-1.5 km.
    const distance = haversineDistanceKm(colosseum, treviFountain);
    expect(distance).toBeGreaterThan(1.2);
    expect(distance).toBeLessThan(1.7);
  });
});

describe("estimateWalkingMinutes", () => {
  it("estimates 60 minutes for 5 km", () => {
    expect(estimateWalkingMinutes(5)).toBe(60);
  });

  it("estimates 30 minutes for 2.5 km", () => {
    expect(estimateWalkingMinutes(2.5)).toBe(30);
  });

  it("returns 0 for non-positive distances", () => {
    expect(estimateWalkingMinutes(0)).toBe(0);
    expect(estimateWalkingMinutes(-1)).toBe(0);
  });
});

describe("getDayDistanceSummaries", () => {
  const itinerary: Itinerary = {
    days: [
      {
        day: 1,
        title: "Ancient Rome",
        items: [
          item("Colosseum", "09:00", place("Colosseum", colosseum)),
          item("Lunch break", "12:00", null), // no place -> ignored
          item("Roman Forum", "13:00", place("Roman Forum", romanForum)),
          item("Trevi Fountain", "16:00", place("Trevi Fountain", treviFountain))
        ]
      },
      {
        day: 2,
        title: "One stop only",
        items: [
          item("Colosseum again", "09:00", place("Colosseum", colosseum)),
          item("Bad coordinates", "10:00", place("Broken", { latitude: 999, longitude: 12 }))
        ]
      }
    ]
  };

  const summaries = getDayDistanceSummaries(itinerary, 8);

  it("returns one summary per day and preserves day numbers", () => {
    expect(summaries.map((summary) => summary.dayNumber)).toEqual([1, 2]);
  });

  it("ignores items without a place and items with invalid coordinates", () => {
    // Day 1: 3 valid (lunch ignored). Day 2: 1 valid (broken coords ignored).
    expect(summaries[0].mappedStops).toBe(3);
    expect(summaries[1].mappedStops).toBe(1);
  });

  it("builds segments in itinerary order", () => {
    expect(summaries[0].segments.map((segment) => [segment.fromName, segment.toName])).toEqual([
      ["Colosseum", "Roman Forum"],
      ["Roman Forum", "Trevi Fountain"]
    ]);
  });

  it("returns 0 distance for a day with a single mapped stop", () => {
    expect(summaries[1].mappedStops).toBe(1);
    expect(summaries[1].segmentCount).toBe(0);
    expect(summaries[1].straightLineDistanceKm).toBe(0);
    expect(summaries[1].estimatedWalkingMinutes).toBe(0);
  });

  it("rounds the day walking time from the accumulated distance, not per segment", () => {
    const day = summaries[0];
    expect(day.estimatedWalkingMinutes).toBe(estimateWalkingMinutes(day.straightLineDistanceKm));
  });

  it("marks exceedsPreference when the day exceeds the preference", () => {
    const lowPreference = getDayDistanceSummaries(itinerary, 1);
    expect(lowPreference[0].exceedsPreference).toBe(true);
    expect(lowPreference[0].maxWalkingKmPerDay).toBe(1);

    // Day 1 is roughly 2 km, comfortably under 8 km.
    expect(summaries[0].exceedsPreference).toBe(false);
  });

  it("never exceeds the preference when none is provided or it is 0", () => {
    expect(getDayDistanceSummaries(itinerary)[0].exceedsPreference).toBe(false);
    expect(getDayDistanceSummaries(itinerary, null)[0].exceedsPreference).toBe(false);
    expect(getDayDistanceSummaries(itinerary, 0)[0].exceedsPreference).toBe(false);
    expect(getDayDistanceSummaries(itinerary, 0)[0].maxWalkingKmPerDay).toBeNull();
  });
});

describe("getTripDistanceTotal", () => {
  it("sums the raw straight-line distance across days", () => {
    const summaries = getDayDistanceSummaries(
      {
        days: [
          {
            day: 1,
            title: "Day 1",
            items: [
              item("A", "09:00", place("A", colosseum)),
              item("B", "10:00", place("B", treviFountain))
            ]
          }
        ]
      },
      8
    );
    expect(getTripDistanceTotal(summaries)).toBeCloseTo(summaries[0].straightLineDistanceKm, 6);
  });
});

describe("formatDistanceKm", () => {
  it("formats with one decimal place", () => {
    expect(formatDistanceKm(3.84)).toBe("3.8 km");
    expect(formatDistanceKm(0.6)).toBe("0.6 km");
  });
});

describe("formatWalkingTime", () => {
  it("formats sub-hour durations as minutes", () => {
    expect(formatWalkingTime(45)).toBe("45 min");
  });

  it("formats hour-plus durations", () => {
    expect(formatWalkingTime(90)).toBe("1h 30min");
    expect(formatWalkingTime(113)).toBe("1h 53min");
  });

  it("drops the minutes part on a whole hour", () => {
    expect(formatWalkingTime(120)).toBe("2h");
  });
});
