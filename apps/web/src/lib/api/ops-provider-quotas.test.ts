import { afterEach, describe, expect, it, vi } from "vitest";
import {
  getProviderQuotaDetail,
  getProviderQuotas,
  resetProviderQuotaDev
} from "@/lib/api/ops";
import type { ProviderQuotasResponse } from "@/types/ops";

const quotas: ProviderQuotasResponse = {
  date: "2026-07-03",
  enabled: true,
  resetAllowed: true,
  providers: [
    {
      provider: "ors",
      category: "routes",
      enabled: true,
      rateLimitPerMinute: 30,
      dailyQuota: 1500,
      usedToday: 312,
      remainingToday: 1188,
      blockedToday: 4,
      fallbackToday: 2,
      status: "quota_exceeded",
      operations: [
        { operation: "route_estimate", usedToday: 312, blockedToday: 4, fallbackToday: 2 }
      ]
    }
  ]
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

describe("provider quota ops client", () => {
  it("GETs provider quotas from External Integrations Service", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(quotas, { ok: true, status: 200 }));
    vi.stubGlobal("fetch", fetchMock);

    const result = await getProviderQuotas();

    const [url, init] = fetchMock.mock.calls[0];
    expect(String(url)).toBe("http://localhost:8084/ops/providers/quotas");
    expect(init?.method ?? "GET").toBe("GET");
    expect(result.providers[0].status).toBe("quota_exceeded");
  });

  it("passes the date query when provided", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(quotas, { ok: true, status: 200 }));
    vi.stubGlobal("fetch", fetchMock);

    await getProviderQuotas("2026-07-01");

    const [url] = fetchMock.mock.calls[0];
    expect(String(url)).toBe("http://localhost:8084/ops/providers/quotas?date=2026-07-01");
  });

  it("GETs a single provider detail", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse(
        { date: "2026-07-03", enabled: true, resetAllowed: true, provider: quotas.providers[0], history: [] },
        { ok: true, status: 200 }
      )
    );
    vi.stubGlobal("fetch", fetchMock);

    await getProviderQuotaDetail("ors");

    const [url] = fetchMock.mock.calls[0];
    expect(String(url)).toBe("http://localhost:8084/ops/providers/quotas/ors");
  });

  it("POSTs a dev reset request", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({ reset: true, provider: "ors", date: "2026-07-03" }, { ok: true, status: 200 })
    );
    vi.stubGlobal("fetch", fetchMock);

    const result = await resetProviderQuotaDev("ors");

    const [url, init] = fetchMock.mock.calls[0];
    expect(String(url)).toBe("http://localhost:8084/ops/providers/quotas/ors/reset-dev");
    expect(init?.method).toBe("POST");
    expect(result.reset).toBe(true);
  });

  it("surfaces controlled provider-limit errors from a non-2xx response", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(jsonResponse({ error: "forbidden" }, { ok: false, status: 403 }));
    vi.stubGlobal("fetch", fetchMock);

    await expect(getProviderQuotas()).rejects.toBeTruthy();
  });
});
