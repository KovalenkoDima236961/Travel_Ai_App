import { afterEach, describe, expect, it, vi } from "vitest";
import { searchAvailability } from "@/lib/api/availability";
import type { AvailabilitySearchResponse } from "@/types/availability";

const response: AvailabilitySearchResponse = {
  status: "available",
  result: "success",
  provider: "mock",
  providerDisplayName: "Mock Tickets",
  fallbackUsed: false,
  cached: false,
  checkedAt: "2026-07-03T10:00:00Z",
  match: { matched: true, confidence: 0.82 },
  options: [
    {
      id: "mock-colosseum-entry",
      title: "Colosseum entry ticket",
      availability: "available",
      price: { amount: 18, currency: "EUR" },
      priceType: "per_person",
      startTimes: ["09:00", "10:30"],
      bookingUrl: "https://example.com/book/colosseum",
      providerName: "Mock Tickets"
    }
  ],
  warnings: ["Availability and prices can change on the provider website."]
};

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

describe("searchAvailability", () => {
  it("POSTs the request to the availability endpoint", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(response, { ok: true, status: 200 }));
    vi.stubGlobal("fetch", fetchMock);

    const result = await searchAvailability({
      destination: "Rome",
      date: "2026-08-10",
      currency: "EUR",
      item: {
        name: "Colosseum",
        type: "attraction"
      },
      travelers: { adults: 2, children: 0 }
    });

    const [url, init] = fetchMock.mock.calls[0];
    expect(String(url)).toBe("http://localhost:8084/availability/search");
    expect(init?.method).toBe("POST");
    expect(JSON.parse(init?.body as string)).toMatchObject({
      destination: "Rome",
      date: "2026-08-10",
      item: { name: "Colosseum" }
    });
    expect(result.options[0].bookingUrl).toMatch(/^https:\/\//);
  });
});
