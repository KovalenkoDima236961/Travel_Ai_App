"use client";

import { useQuery } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { Card } from "@/shared/ui/card";
import { getWeatherForecast, weatherKeys } from "@/lib/api/weather";
import { cn, formatDate } from "@/lib/utils";
import type { WeatherDay } from "@/entities/weather/model";

type WeatherForecastCardProps = {
  destination: string;
  startDate?: string | null;
  days: number;
  offline?: boolean;
  className?: string;
};

export function WeatherForecastCard({
  destination,
  startDate,
  days,
  offline = false,
  className
}: WeatherForecastCardProps) {
  const canFetch = Boolean(destination.trim()) && Boolean(startDate) && days > 0;
  const params = {
    destination,
    startDate: startDate ?? "",
    days
  };

  const forecastQuery = useQuery({
    queryKey: weatherKeys.forecast(params),
    queryFn: () => getWeatherForecast(params),
    enabled: canFetch && !offline,
    staleTime: 10 * 60 * 1000,
    retry: 1
  });

  if (!startDate) {
    return (
      <Card className={className}>
        <h2 className="text-xl font-semibold text-slate-950">Weather context</h2>
        <p className="mt-3 text-sm text-slate-600">
          Add a trip start date to see weather context.
        </p>
      </Card>
    );
  }

  return (
    <Card className={className}>
      <div className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h2 className="text-xl font-semibold text-slate-950">Weather context</h2>
          <p className="mt-1 text-sm text-slate-600">
            Forecast for {destination} from {formatDate(startDate)}.
          </p>
        </div>
        <ProviderBadge provider={forecastQuery.data?.provider} />
      </div>

      {offline ? (
        <div className="mt-4 rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900">
          Weather refresh requires an internet connection.
        </div>
      ) : null}

      {!offline && forecastQuery.isPending ? (
        <div className="mt-4 rounded-lg border border-slate-200 bg-slate-50 p-4 text-sm text-slate-600">
          Loading weather forecast...
        </div>
      ) : null}

      {!offline && forecastQuery.isError ? (
        <div className="mt-4 rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900">
          Weather forecast unavailable.
        </div>
      ) : null}

      {forecastQuery.data ? (
        <div className="mt-5 space-y-3">
          {forecastQuery.data.provider === "mock" ? (
            <p className="rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-xs font-medium text-slate-600">
              Mock forecast for local development
            </p>
          ) : null}

          <ul className="grid gap-3 md:grid-cols-2">
            {forecastQuery.data.days.map((day) => (
              <WeatherDayRow key={day.date} day={day} />
            ))}
          </ul>
        </div>
      ) : null}
    </Card>
  );
}

function WeatherDayRow({ day }: { day: WeatherDay }) {
  const warnings = normalizedWarnings(day);

  return (
    <li className="rounded-lg border border-slate-200 bg-white p-4">
      <div className="flex items-start justify-between gap-3">
        <div>
          <p className="text-sm font-semibold text-slate-950">{formatDate(day.date)}</p>
          <p className="mt-1 text-sm text-slate-700">{day.summary || formatCondition(day.condition)}</p>
        </div>
        <span className={cn("rounded-full px-2.5 py-0.5 text-xs font-medium", conditionClass(day))}>
          {formatCondition(day.condition)}
        </span>
      </div>

      <dl className="mt-4 grid grid-cols-3 gap-3 text-sm">
        <Metric label="Temp" value={<>{formatTemperatureRange(day)}&deg;C</>} />
        <Metric label="Rain" value={`${day.precipitationChance}%`} />
        <Metric label="Wind" value={`${formatNumber(day.windSpeedKph)} kph`} />
      </dl>

      {warnings.length > 0 ? (
        <div className="mt-4 flex flex-wrap gap-2">
          {warnings.map((warning) => (
            <span
              key={warning}
              className="rounded-full border border-amber-200 bg-amber-50 px-2.5 py-1 text-xs font-medium text-amber-900"
            >
              {warning}
            </span>
          ))}
        </div>
      ) : null}
    </li>
  );
}

function ProviderBadge({ provider }: { provider?: string | null }) {
  if (!provider) {
    return (
      <span className="inline-flex w-fit rounded-full border border-slate-200 bg-slate-50 px-2.5 py-0.5 text-xs font-medium text-slate-500">
        Provider pending
      </span>
    );
  }

  return (
    <span className="inline-flex w-fit rounded-full border border-primary-200 bg-primary-50 px-2.5 py-0.5 text-xs font-medium text-primary-700">
      Provider: {provider}
    </span>
  );
}

function Metric({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div>
      <dt className="text-xs text-slate-500">{label}</dt>
      <dd className="mt-0.5 font-medium text-slate-900">{value}</dd>
    </div>
  );
}

function normalizedWarnings(day: WeatherDay) {
  if (day.warnings && day.warnings.length > 0) {
    return day.warnings.map(shortWarningLabel);
  }

  const warnings: string[] = [];
  if (day.temperatureMaxC >= 32) {
    warnings.push("High heat");
  }
  if (day.precipitationChance >= 60) {
    warnings.push("Rain likely");
  }
  if (day.windSpeedKph >= 35) {
    warnings.push("Windy");
  }
  if (day.temperatureMaxC <= 5) {
    warnings.push("Cold");
  }
  return warnings;
}

function shortWarningLabel(warning: string) {
  const lower = warning.toLowerCase();
  if (lower.includes("heat")) {
    return "High heat";
  }
  if (lower.includes("rain")) {
    return "Rain likely";
  }
  if (lower.includes("wind")) {
    return "Windy";
  }
  if (lower.includes("cold")) {
    return "Cold";
  }
  return warning;
}

function formatTemperatureRange(day: WeatherDay) {
  return `${formatNumber(day.temperatureMinC)}-${formatNumber(day.temperatureMaxC)}`;
}

function formatNumber(value: number) {
  return Number.isInteger(value) ? String(value) : value.toFixed(1);
}

function formatCondition(value: string) {
  return value.replace(/_/g, " ").replace(/\b\w/g, (char) => char.toUpperCase());
}

function conditionClass(day: WeatherDay) {
  if (day.temperatureMaxC >= 32 || day.condition === "hot") {
    return "bg-red-50 text-red-700";
  }
  if (day.precipitationChance >= 60 || day.condition.includes("rain")) {
    return "bg-sky-50 text-sky-700";
  }
  if (day.windSpeedKph >= 35 || day.condition === "windy") {
    return "bg-slate-100 text-slate-700";
  }
  if (day.temperatureMaxC <= 5 || day.condition === "cold") {
    return "bg-cyan-50 text-cyan-700";
  }
  return "bg-emerald-50 text-emerald-700";
}
