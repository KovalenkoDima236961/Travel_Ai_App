import { afterEach, describe, expect, it, vi } from "vitest";
import {
  createTripFromSuggestion,
  getTripDiscoverySuggestions,
  refineTripDiscovery,
  surpriseMe
} from "@/lib/api/trip-discovery";

function jsonResponse(body: unknown): Response {
  return {
    ok: true,
    status: 200,
    text: async () => JSON.stringify(body),
    json: async () => body
  } as unknown as Response;
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("trip discovery API", () => {
  it("calls prompt, surprise, and refine endpoints", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ id: "session-1" }));
    vi.stubGlobal("fetch", fetchMock);

    await getTripDiscoverySuggestions({
      prompt: "warm food weekend",
      scope: "personal",
      outputLanguage: "en"
    });
    await surpriseMe({
      scope: "personal",
      outputLanguage: "en",
      noveltyLevel: "balanced"
    });
    await refineTripDiscovery("session-1", {
      instruction: "Cheaper",
      outputLanguage: "en"
    });

    expect(String(fetchMock.mock.calls[0][0])).toContain("/trip-discovery/suggestions");
    expect(String(fetchMock.mock.calls[1][0])).toContain("/trip-discovery/surprise-me");
    expect(String(fetchMock.mock.calls[2][0])).toContain(
      "/trip-discovery/session-1/refine"
    );
  });

  it("creates a trip only through the explicit confirmation endpoint", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({ trip: { id: "trip-1" }, generationJob: null })
    );
    vi.stubGlobal("fetch", fetchMock);

    await createTripFromSuggestion("session-1", "valencia-spain", {
      durationDays: 4,
      travelers: 2,
      autoGenerateItinerary: false
    });

    expect(String(fetchMock.mock.calls[0][0])).toContain(
      "/trip-discovery/session-1/suggestions/valencia-spain/create-trip"
    );
    const body = JSON.parse((fetchMock.mock.calls[0][1] as RequestInit).body as string);
    expect(body.autoGenerateItinerary).toBe(false);
  });
});
