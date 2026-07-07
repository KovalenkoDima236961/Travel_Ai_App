import { afterEach, describe, expect, it, vi } from "vitest";
import {
  getTripCostAnalytics,
  getWorkspaceCostAnalytics
} from "@/lib/api/cost-analytics";

function jsonResponse(body: unknown, init: { ok: boolean; status: number }): Response {
  return {
    ok: init.ok,
    status: init.status,
    text: async () => JSON.stringify(body),
    json: async () => body
  } as unknown as Response;
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("cost analytics API", () => {
  it("fetches trip analytics with an optional currency", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({ tripId: "trip-1", currency: "EUR" }, { ok: true, status: 200 })
    );
    vi.stubGlobal("fetch", fetchMock);

    await getTripCostAnalytics("trip-1", " eur ");

    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining("/trips/trip-1/analytics/costs?currency=EUR"),
      expect.any(Object)
    );
  });

  it("fetches workspace analytics with filters", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({ workspaceId: "workspace-1", currency: "USD" }, { ok: true, status: 200 })
    );
    vi.stubGlobal("fetch", fetchMock);

    await getWorkspaceCostAnalytics("workspace-1", {
      currency: "usd",
      from: "2026-01-01",
      to: "2026-12-31",
      includeArchived: true
    });

    const url = String(fetchMock.mock.calls[0][0]);
    expect(url).toContain("/workspaces/workspace-1/analytics/costs?");
    expect(url).toContain("currency=USD");
    expect(url).toContain("from=2026-01-01");
    expect(url).toContain("to=2026-12-31");
    expect(url).toContain("includeArchived=true");
  });
});
