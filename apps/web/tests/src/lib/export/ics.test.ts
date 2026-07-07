import { describe, expect, it } from "vitest";
import {
  generateTripIcs,
  getTripIcsEventCount,
  parseItemTime
} from "@/lib/export/ics";
import type { ExportTrip } from "@/lib/export/trip-export-adapter";

describe("generateTripIcs", () => {
  it("generates a calendar and escapes ICS text fields", () => {
    const ics = generateTripIcs(baseExportTrip());

    expect(ics).toContain("BEGIN:VCALENDAR\r\n");
    expect(ics).toContain("VERSION:2.0\r\n");
    expect(ics).toContain("BEGIN:VEVENT\r\n");
    expect(ics).toContain("SUMMARY:Museum\\, East\\; Wing");
    expect(ics).toContain("LOCATION:Via Test\\; Rome");
    expect(ics).toContain("DESCRIPTION:Line 1\\nLine 2");
  });

  it("creates one event per timed item and skips untimed items", () => {
    expect(getTripIcsEventCount(baseExportTrip())).toBe(3);
  });

  it("uses the correct day offset and supports 24-hour and AM/PM times", () => {
    const ics = generateTripIcs(baseExportTrip());

    expect(ics).toContain("DTSTART:20260810T090000");
    expect(ics).toContain("DTEND:20260810T103000");
    expect(ics).toContain("DTSTART:20260811T143000");
    expect(ics).toContain("DTEND:20260811T153000");
  });

  it("includes location, map URL, and estimated cost in descriptions", () => {
    const ics = unfoldIcs(generateTripIcs(baseExportTrip()));

    expect(ics).toContain("Place: Test Museum");
    expect(ics).toContain("Map: https://maps.example.com/test");
    expect(ics).toContain("Estimated cost: €15 ticket");
  });

  it("does not include private fields after adapter sanitization", () => {
    const ics = generateTripIcs(baseExportTrip());

    expect(ics).not.toContain("user-123");
    expect(ics).not.toContain("owner@example.com");
    expect(ics).not.toContain("preferences");
    expect(ics).not.toContain("version");
  });
});

describe("parseItemTime", () => {
  it("parses common 24-hour and AM/PM formats", () => {
    expect(parseItemTime("09:00")).toEqual({ hour: 9, minute: 0 });
    expect(parseItemTime("9:00")).toEqual({ hour: 9, minute: 0 });
    expect(parseItemTime("09:00 AM")).toEqual({ hour: 9, minute: 0 });
    expect(parseItemTime("2:30 PM")).toEqual({ hour: 14, minute: 30 });
    expect(parseItemTime("14:30")).toEqual({ hour: 14, minute: 30 });
  });
});

function baseExportTrip(): ExportTrip {
  return {
    destination: "Rome, Italy",
    startDate: "2026-08-10",
    days: 2,
    budgetCurrency: "EUR",
    itinerary: {
      days: [
        {
          day: 1,
          title: "Arrival",
          items: [
            {
              time: "09:00-10:30",
              type: "activity",
              name: "Museum, East; Wing",
              note: "Line 1\nLine 2",
              estimatedCost: { amount: 15, currency: "EUR", category: "ticket" },
              place: {
                provider: "mock",
                providerPlaceId: "",
                name: "Test Museum",
                address: "Via Test; Rome",
                mapUrl: "https://maps.example.com/test"
              }
            },
            {
              time: "",
              type: "rest",
              name: "Open afternoon"
            }
          ]
        },
        {
          day: 2,
          title: "Center",
          items: [
            {
              time: "2:30 PM",
              type: "place",
              name: "Forum walk"
            },
            {
              time: "14:30",
              type: "transport",
              name: "Taxi back"
            }
          ]
        }
      ]
    },
    source: "private"
  };
}

function unfoldIcs(value: string): string {
  return value.replace(/\r\n /g, "");
}
