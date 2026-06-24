"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { useParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { ExportTripMenu } from "@/components/export/ExportTripMenu";
import { PublicShareUnlockForm } from "@/components/share/PublicShareUnlockForm";
import { DistanceSummary } from "@/components/trips/DistanceSummary";
import { ItineraryMap } from "@/components/trips/ItineraryMap";
import { ItineraryView } from "@/components/trips/ItineraryView";
import { TripStatusBadge } from "@/components/trips/TripStatusBadge";
import { WeatherForecastCard } from "@/components/trips/WeatherForecastCard";
import { Card } from "@/components/ui/Card";
import { buttonStyles } from "@/components/ui/Button";
import { ApiError } from "@/lib/api/client";
import {
  getPublicShareStatus,
  getPublicTrip,
  tripKeys,
  unlockPublicShare
} from "@/lib/api/trips";
import { getWeatherForecast, weatherKeys } from "@/lib/api/weather";
import {
  toExportDistanceSummary,
  toExportTripFromPublicTrip,
  toExportWeatherSummary
} from "@/lib/export/trip-export-adapter";
import { getDayDistanceSummaries } from "@/lib/itinerary/distance-utils";
import {
  formatBudget,
  formatDate,
  getErrorMessage,
  formatInterestLabel,
  formatPaceLabel
} from "@/lib/utils";

export default function PublicSharePage() {
  const params = useParams<{ shareToken: string }>();
  const shareToken = params.shareToken;
  const [publicShareAccessToken, setPublicShareAccessToken] = useState<string | null>(null);
  const [storedTokenChecked, setStoredTokenChecked] = useState(false);
  const [unlockLoading, setUnlockLoading] = useState(false);
  const [unlockError, setUnlockError] = useState<string | null>(null);

  const publicShareStatusQuery = useQuery({
    queryKey: tripKeys.publicShareStatus(shareToken),
    queryFn: () => getPublicShareStatus(shareToken),
    enabled: Boolean(shareToken),
    retry: false
  });
  const status = publicShareStatusQuery.data ?? null;
  const shouldFetchTrip =
    Boolean(shareToken) &&
    Boolean(status?.available) &&
    (!status?.passwordRequired || Boolean(publicShareAccessToken));
  const publicTripQuery = useQuery({
    queryKey: [...tripKeys.publicShare(shareToken), publicShareAccessToken ?? "anonymous"],
    queryFn: () => getPublicTrip(shareToken, publicShareAccessToken),
    enabled: shouldFetchTrip,
    retry: false
  });

  useEffect(() => {
    setStoredTokenChecked(false);
    setPublicShareAccessToken(null);
    setUnlockError(null);

    if (!shareToken || typeof window === "undefined") {
      setStoredTokenChecked(true);
      return;
    }

    const token = sessionStorage.getItem(publicShareTokenKey(shareToken));
    const expiresAt = sessionStorage.getItem(publicShareTokenExpiryKey(shareToken));
    if (token && expiresAt && new Date(expiresAt).getTime() <= Date.now()) {
      clearStoredPublicShareToken(shareToken);
      setStoredTokenChecked(true);
      return;
    }
    setPublicShareAccessToken(token);
    setStoredTokenChecked(true);
  }, [shareToken]);

  useEffect(() => {
    if (
      publicTripQuery.isError &&
      publicTripQuery.error instanceof ApiError &&
      publicTripQuery.error.status === 401 &&
      status?.passwordRequired
    ) {
      clearStoredPublicShareToken(shareToken);
      setPublicShareAccessToken(null);
      setUnlockError(null);
    }
  }, [publicTripQuery.error, publicTripQuery.isError, shareToken, status?.passwordRequired]);

  const sharedTrip = publicTripQuery.data ?? null;
  const weatherParams = {
    destination: sharedTrip?.destination ?? "",
    startDate: sharedTrip?.startDate ?? "",
    days: sharedTrip?.days ?? 0
  };
  const publicWeatherForecastQuery = useQuery({
    queryKey: weatherKeys.forecast(weatherParams),
    queryFn: () => getWeatherForecast(weatherParams),
    enabled:
      Boolean(weatherParams.destination.trim()) &&
      Boolean(weatherParams.startDate) &&
      weatherParams.days > 0,
    staleTime: 10 * 60 * 1000,
    retry: 1
  });
  const publicItinerary = sharedTrip?.itinerary ?? null;
  const publicDistanceSummaries = useMemo(
    () => (publicItinerary ? getDayDistanceSummaries(publicItinerary) : []),
    [publicItinerary]
  );
  const exportTrip = useMemo(
    () =>
      sharedTrip
        ? toExportTripFromPublicTrip(sharedTrip, {
            weatherSummary: toExportWeatherSummary(publicWeatherForecastQuery.data ?? null),
            distanceSummary: toExportDistanceSummary(publicDistanceSummaries)
          })
        : null,
    [publicDistanceSummaries, publicWeatherForecastQuery.data, sharedTrip]
  );

  async function handleUnlock(password: string) {
    setUnlockLoading(true);
    setUnlockError(null);
    try {
      const unlocked = await unlockPublicShare(shareToken, password);
      sessionStorage.setItem(publicShareTokenKey(shareToken), unlocked.accessToken);
      sessionStorage.setItem(publicShareTokenExpiryKey(shareToken), unlocked.expiresAt);
      setPublicShareAccessToken(unlocked.accessToken);
    } catch (error) {
      clearStoredPublicShareToken(shareToken);
      setPublicShareAccessToken(null);
      setUnlockError(getErrorMessage(error, "Invalid password."));
    } finally {
      setUnlockLoading(false);
    }
  }

  if (
    publicShareStatusQuery.isPending ||
    (status?.passwordRequired && !storedTokenChecked) ||
    (shouldFetchTrip && publicTripQuery.isPending)
  ) {
    return (
      <div className="mx-auto w-full max-w-6xl px-4 py-8 sm:px-6 lg:px-8">
        <div className="rounded-lg border border-slate-200 bg-white p-6 text-sm text-slate-600">
          Loading shared itinerary...
        </div>
      </div>
    );
  }

  if (publicShareStatusQuery.isError) {
    return (
      <div className="mx-auto w-full max-w-3xl px-4 py-12 sm:px-6 lg:px-8">
        <div className="rounded-lg border border-amber-200 bg-amber-50 p-6">
          <h1 className="text-xl font-semibold text-amber-950">Shared trip unavailable</h1>
          <p className="mt-2 text-sm leading-6 text-amber-900">
            This shared trip is unavailable, expired, or disabled.
          </p>
          <Link className={buttonStyles({ variant: "secondary", className: "mt-5" })} href="/">
            Go to home
          </Link>
        </div>
      </div>
    );
  }

  if (status?.passwordRequired && !publicShareAccessToken) {
    return (
      <PublicShareUnlockForm
        error={unlockError}
        loading={unlockLoading}
        onUnlock={handleUnlock}
      />
    );
  }

  if (publicTripQuery.isError || !publicTripQuery.data) {
    if (
      status?.passwordRequired &&
      publicTripQuery.error instanceof ApiError &&
      publicTripQuery.error.status === 401
    ) {
      return (
        <PublicShareUnlockForm
          error={unlockError}
          loading={unlockLoading}
          onUnlock={handleUnlock}
        />
      );
    }
    return (
      <div className="mx-auto w-full max-w-3xl px-4 py-12 sm:px-6 lg:px-8">
        <div className="rounded-lg border border-amber-200 bg-amber-50 p-6">
          <h1 className="text-xl font-semibold text-amber-950">Shared trip unavailable</h1>
          <p className="mt-2 text-sm leading-6 text-amber-900">
            This shared trip is unavailable, expired, or disabled.
          </p>
          <Link className={buttonStyles({ variant: "secondary", className: "mt-5" })} href="/">
            Go to home
          </Link>
        </div>
      </div>
    );
  }

  const trip = publicTripQuery.data;
  const itinerary = trip.itinerary ?? null;
  const interests = trip.interests ?? [];
  const currency = trip.budgetCurrency ?? "EUR";
  const travelers = trip.travelers ?? 0;

  return (
    <div className="mx-auto w-full max-w-6xl px-4 py-8 sm:px-6 lg:px-8">
      <div className="mb-8 flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <p className="text-sm font-medium text-primary-700">Shared itinerary</p>
          <div className="mt-2 flex flex-wrap items-center gap-3">
            <h1 className="text-3xl font-semibold text-slate-950">{trip.destination}</h1>
            <TripStatusBadge status={trip.status} />
          </div>
        </div>
        <div className="flex flex-col items-start gap-3 sm:items-end">
          <Link className={buttonStyles({ variant: "secondary", size: "sm" })} href="/">
            Travel AI Planner
          </Link>
          {exportTrip && itinerary ? <ExportTripMenu exportTrip={exportTrip} /> : null}
        </div>
      </div>

      <div className="grid gap-6 lg:grid-cols-[22rem_minmax(0,1fr)]">
        <Card>
          <h2 className="text-lg font-semibold text-slate-950">Trip summary</h2>
          <dl className="mt-5 space-y-4 text-sm">
            <DetailRow label="Start date" value={trip.startDate ? formatDate(trip.startDate) : "Not set"} />
            <DetailRow label="Duration" value={`${trip.days} ${trip.days === 1 ? "day" : "days"}`} />
            <DetailRow label="Travelers" value={travelers > 0 ? String(travelers) : "Not set"} />
            <DetailRow label="Budget" value={formatBudget(trip.budgetAmount, currency)} />
            <DetailRow label="Pace" value={trip.pace ? formatPaceLabel(trip.pace) : "Not set"} />
            {trip.sharedAt ? (
              <DetailRow
                label="Shared"
                value={formatDate(trip.sharedAt, {
                  dateStyle: "medium",
                  timeStyle: "short"
                })}
              />
            ) : null}
          </dl>
          <div className="mt-6">
            <p className="text-sm font-medium text-slate-700">Interests</p>
            <div className="mt-2 flex flex-wrap gap-2">
              {interests.length > 0 ? (
                interests.map((interest) => (
                  <span
                    key={interest}
                    className="rounded-full border border-slate-200 bg-slate-50 px-3 py-1 text-xs font-medium text-slate-700"
                  >
                    {formatInterestLabel(interest)}
                  </span>
                ))
              ) : (
                <span className="text-sm text-slate-500">No interests listed</span>
              )}
            </div>
          </div>
        </Card>

        <section className="min-w-0">
          <WeatherForecastCard
            className="mb-4"
            days={trip.days}
            destination={trip.destination}
            startDate={trip.startDate}
          />

          {itinerary ? (
            <div className="space-y-4">
              <ItineraryView
                currency={currency}
                itinerary={itinerary}
                startDate={trip.startDate}
              />
              <ItineraryMap itinerary={itinerary} startDate={trip.startDate} />
              <DistanceSummary itinerary={itinerary} />
            </div>
          ) : (
            <div className="rounded-lg border border-slate-200 bg-white p-6 text-sm text-slate-600">
              This shared trip does not have an itinerary yet.
            </div>
          )}
        </section>
      </div>
    </div>
  );
}

function publicShareTokenKey(shareToken: string) {
  return `public-share-access-token:${shareToken}`;
}

function publicShareTokenExpiryKey(shareToken: string) {
  return `public-share-access-token-exp:${shareToken}`;
}

function clearStoredPublicShareToken(shareToken: string) {
  if (typeof window === "undefined") {
    return;
  }
  sessionStorage.removeItem(publicShareTokenKey(shareToken));
  sessionStorage.removeItem(publicShareTokenExpiryKey(shareToken));
}

type DetailRowProps = {
  label: string;
  value: string;
};

function DetailRow({ label, value }: DetailRowProps) {
  return (
    <div className="flex items-start justify-between gap-4">
      <dt className="text-slate-500">{label}</dt>
      <dd className="text-right font-medium text-slate-800">{value}</dd>
    </div>
  );
}
