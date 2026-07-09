"use client";

import { useQuery } from "@tanstack/react-query";
import { getWeatherForecast, weatherKeys } from "@/lib/api/weather";
import type { WeatherDay } from "@/entities/weather/model";
import { CloudIcon, SunIcon } from "./icons";

type RightRailWeatherProps = {
  destination: string;
  startDate?: string | null;
  days: number;
  offline?: boolean;
};

/**
 * Warm right-rail weather card, forked from the shared WeatherForecastCard. Uses
 * the same forecast query and only renders when a forecast is available (matching
 * the mock's compact 4-up grid); it stays quiet on error/offline to avoid slate
 * fallbacks leaking into the warm layout.
 */
export function RightRailWeather({
  destination,
  startDate,
  days,
  offline = false
}: RightRailWeatherProps) {
  const canFetch = Boolean(destination.trim()) && Boolean(startDate) && days > 0;
  const params = { destination, startDate: startDate ?? "", days };

  const forecastQuery = useQuery({
    queryKey: weatherKeys.forecast(params),
    queryFn: () => getWeatherForecast(params),
    enabled: canFetch && !offline,
    staleTime: 10 * 60 * 1000,
    retry: 1
  });

  const forecast = forecastQuery.data;
  if (!forecast || forecast.days.length === 0) {
    return null;
  }

  const warning = forecast.days.find((day) => normalizedWarning(day));

  return (
    <div className="rounded-[20px] border border-sand-300 bg-white px-[22px] py-5">
      <div className="flex items-baseline justify-between gap-2">
        <h2 className="text-[13px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">
          Weather
        </h2>
        <span className="text-[12px] text-[#B09E8A]">{forecast.provider ?? "forecast"}</span>
      </div>
      <div className="mt-4 grid grid-cols-4 gap-2">
        {forecast.days.map((day) => (
          <WeatherCell key={day.date} day={day} />
        ))}
      </div>
      {warning ? (
        <p className="mt-3.5 flex items-center gap-2 text-[12.5px] text-[#96682A]">
          <CloudIcon className="h-[13px] w-[13px] shrink-0" />
          {normalizedWarning(warning)}
        </p>
      ) : null}
    </div>
  );
}

function WeatherCell({ day }: { day: WeatherDay }) {
  const rainy = day.precipitationChance >= 60 || day.condition.toLowerCase().includes("rain");
  return (
    <div className="rounded-[14px] bg-sand-50 px-2 py-3 text-center">
      <p className="text-[11.5px] font-semibold text-cocoa-400">{formatShortDay(day.date)}</p>
      {rainy ? (
        <CloudIcon className="mx-auto mt-2 block h-5 w-5 text-[#7C93A6]" />
      ) : (
        <SunIcon className="mx-auto mt-2 block h-5 w-5 text-[#D9A441]" />
      )}
      <p className="mt-2 text-[12.5px] font-semibold text-cocoa-900">
        {Math.round(day.temperatureMaxC)}°
      </p>
      <p className="mt-0.5 text-[11px] text-[#A08D78]">{Math.round(day.temperatureMinC)}°</p>
    </div>
  );
}

function formatShortDay(dateStr: string): string {
  const date = new Date(dateStr);
  if (Number.isNaN(date.getTime())) {
    return dateStr;
  }
  const weekday = new Intl.DateTimeFormat("en", { weekday: "short" }).format(date);
  return `${weekday} ${date.getDate()}`;
}

function normalizedWarning(day: WeatherDay): string | null {
  if (day.precipitationChance >= 60 || day.condition.toLowerCase().includes("rain")) {
    return `Rain likely ${formatShortDay(day.date)} — plan indoor backups.`;
  }
  if (day.temperatureMaxC >= 32) {
    return `High heat ${formatShortDay(day.date)} — pace outdoor time.`;
  }
  return null;
}
