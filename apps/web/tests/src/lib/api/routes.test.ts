import { afterEach, describe, expect, it, vi } from "vitest";
import { estimateRoute } from "@/lib/api/routes";
import type { RouteEstimate, RouteEstimateRequest } from "@/entities/route/model";

const request: RouteEstimateRequest = {
  mode: "walking",
  stops: [
    { name: "Colosseum", latitude: 41.8902, longitude: 12.4922 },
    { name: "Trevi Fountain", latitude: 41.9009, longitude: 12.4833 }
  ]
};

const estimate: RouteEstimate = {
  mode: "walking",
  provider: "mock",
  distanceKm: 2.1,
  durationMinutes: 28,
  segments: [
    { fromName: "Colosseum", toName: "Trevi Fountain", distanceKm: 2.1, durationMinutes: 28 }
  ]
};

function jsonResponse(body: unknown, init: { ok: boolean; status: number }): Response {
  return {
    ok: init.ok,
    status: init.status,
    json: async () => body
  } as unknown as Response;
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("estimateRoute", () => {
  it("POSTs the request to the /routes/estimate endpoint", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(estimate, { ok: true, status: 200 }));
    vi.stubGlobal("fetch", fetchMock);

    const result = await estimateRoute(request);

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, init] = fetchMock.mock.calls[0];
    expect(String(url)).toBe("http://localhost:8084/routes/estimate");
    expect(init?.method).toBe("POST");
    expect(init?.headers).toMatchObject({ "Content-Type": "application/json" });
    expect(JSON.parse(init?.body as string)).toEqual(request);
    expect(result).toEqual(estimate);
  });

  it("parses the JSON error message from a non-2xx response", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(jsonResponse({ error: "unsupported mode" }, { ok: false, status: 400 }));
    vi.stubGlobal("fetch", fetchMock);

    await expect(estimateRoute(request)).rejects.toThrow("unsupported mode");
  });

  it("throws a readable error when the service is unreachable", async () => {
    const fetchMock = vi.fn().mockRejectedValue(new TypeError("network down"));
    vi.stubGlobal("fetch", fetchMock);

    await expect(estimateRoute(request)).rejects.toThrow(/Could not reach the route service/);
  });
});
