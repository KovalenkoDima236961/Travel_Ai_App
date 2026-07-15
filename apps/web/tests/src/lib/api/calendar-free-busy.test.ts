import { afterEach, describe, expect, it, vi } from "vitest";
import {
  applyCalendarAvailabilityImport,
  previewCalendarAvailabilityImport
} from "@/lib/api/calendar-free-busy";

function jsonResponse(body: unknown, init: { ok: boolean; status: number }): Response {
  const text = JSON.stringify(body);
  return {
    ok: init.ok,
    status: init.status,
    headers: { get: () => "application/json" },
    json: async () => body,
    text: async () => text
  } as unknown as Response;
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("calendar free/busy import API", () => {
  it("POSTs preview requests to the trip calendar import endpoint", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse(
        {
          preview: {
            source: "google_calendar",
            range: {
              startDate: "2026-09-01",
              endDate: "2026-09-30",
              timezone: "Europe/Bratislava"
            },
            busyBlocksSummary: {
              busyBlockCount: 1,
              busyDays: 1,
              fullyBusyDays: 1,
              partiallyBusyDays: 0
            },
            suggestedUnavailableRanges: [
              { startDate: "2026-09-12", endDate: "2026-09-12", reason: "calendar_fully_busy" }
            ],
            suggestedPreferredRanges: [],
            daySummaries: [],
            warnings: []
          }
        },
        { ok: true, status: 200 }
      )
    );
    vi.stubGlobal("fetch", fetchMock);

    await previewCalendarAvailabilityImport("trip-1", {
      startDate: "2026-09-01",
      endDate: "2026-09-30",
      timezone: "Europe/Bratislava",
      calendarProvider: "google",
      conversion: {
        fullyBusyThresholdHours: 6,
        markFullyBusyDaysUnavailable: true,
        markPartiallyBusyDaysUnavailable: false,
        includeWeekendsAsPreferredIfFree: false
      }
    });

    const [url, init] = fetchMock.mock.calls[0];
    expect(String(url)).toBe("http://localhost:8080/trips/trip-1/availability/import-calendar/preview");
    expect(init?.method).toBe("POST");
    expect(JSON.parse(init?.body as string)).toMatchObject({
      startDate: "2026-09-01",
      endDate: "2026-09-30",
      calendarProvider: "google",
      calendarIds: ["primary"]
    });
  });

  it("POSTs apply requests without event detail fields", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse(
        {
          availability: {
            userId: "user-1",
            displayName: "User",
            availableRanges: [],
            unavailableRanges: [{ startDate: "2026-09-12", endDate: "2026-09-12" }],
            preferredRanges: [],
            submitted: true
          },
          dateOptions: { options: [], summary: { responseCount: 1, totalCollaborators: 1, missingResponseCount: 0 } }
        },
        { ok: true, status: 200 }
      )
    );
    vi.stubGlobal("fetch", fetchMock);

    await applyCalendarAvailabilityImport("trip-1", {
      startDate: "2026-09-01",
      endDate: "2026-09-30",
      timezone: "Europe/Bratislava",
      calendarProvider: "google",
      calendarIds: ["primary"],
      mode: "merge",
      conversion: {
        fullyBusyThresholdHours: 6,
        markFullyBusyDaysUnavailable: true,
        markPartiallyBusyDaysUnavailable: false,
        includeWeekendsAsPreferredIfFree: false
      },
      availabilitySettings: {
        availableRanges: [],
        notes: "Imported from Google Calendar."
      }
    });

    const body = JSON.parse(fetchMock.mock.calls[0][1]?.body as string);
    expect(body.mode).toBe("merge");
    expect(JSON.stringify(body)).not.toMatch(/title|description|attendees|location|eventId/i);
  });
});
