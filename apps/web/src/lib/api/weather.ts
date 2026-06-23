import { getExternalIntegrationsServiceUrl } from "@/lib/config";
import type { WeatherForecast } from "@/types/weather";

type ApiErrorPayload = {
  error?: string;
};

export const weatherKeys = {
  all: ["weather"] as const,
  forecast: (params: WeatherForecastParams) => [...weatherKeys.all, "forecast", params] as const
};

type WeatherForecastParams = {
  destination: string;
  startDate: string;
  days: number;
};

export async function getWeatherForecast(
  params: WeatherForecastParams
): Promise<WeatherForecast> {
  const endpoint = new URL("/weather/forecast", getExternalIntegrationsServiceUrl());
  endpoint.searchParams.set("destination", params.destination);
  endpoint.searchParams.set("startDate", params.startDate);
  endpoint.searchParams.set("days", String(params.days));

  let response: Response;
  try {
    response = await fetch(endpoint.toString(), {
      headers: {
        Accept: "application/json"
      }
    });
  } catch {
    throw new Error(
      "Could not reach the weather service. Confirm the local stack is running and CORS allows this origin."
    );
  }

  if (!response.ok) {
    const payload = await readJson<ApiErrorPayload>(response);
    const message =
      typeof payload?.error === "string" && payload.error.trim().length > 0
        ? payload.error
        : `Weather service request failed with status ${response.status}`;
    throw new Error(message);
  }

  return (await response.json()) as WeatherForecast;
}

async function readJson<T>(response: Response): Promise<T | null> {
  try {
    return (await response.json()) as T;
  } catch {
    return null;
  }
}
