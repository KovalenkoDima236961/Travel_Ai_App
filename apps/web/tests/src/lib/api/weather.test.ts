import { afterEach, describe, expect, it, vi } from "vitest";
import { getWeatherForecast } from "@/lib/api/weather";
import type { WeatherForecast } from "@/entities/weather/model";

const forecast: WeatherForecast = {
  destination: "Rome",
  provider: "mock",
  days: [
    {
      date: "2026-08-10",
      condition: "hot",
      temperatureMinC: 24,
      temperatureMaxC: 35,
      precipitationChance: 5,
      windSpeedKph: 10,
      summary: "Hot and sunny",
      warnings: ["High heat: avoid long outdoor walks at midday"]
    }
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

describe("getWeatherForecast", () => {
  it("GETs the encoded weather forecast endpoint", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(forecast, { ok: true, status: 200 }));
    vi.stubGlobal("fetch", fetchMock);

    const result = await getWeatherForecast({
      destination: "Rome",
      startDate: "2026-08-10",
      days: 3
    });

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, init] = fetchMock.mock.calls[0];
    expect(String(url)).toBe(
      "http://localhost:8084/weather/forecast?destination=Rome&startDate=2026-08-10&days=3"
    );
    expect(init?.headers).toMatchObject({ Accept: "application/json" });
    expect(result).toEqual(forecast);
  });

  it("parses JSON error messages from non-2xx responses", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(jsonResponse({ error: "days must be at most 30" }, { ok: false, status: 400 }));
    vi.stubGlobal("fetch", fetchMock);

    await expect(
      getWeatherForecast({ destination: "Rome", startDate: "2026-08-10", days: 31 })
    ).rejects.toThrow("days must be at most 30");
  });
});
