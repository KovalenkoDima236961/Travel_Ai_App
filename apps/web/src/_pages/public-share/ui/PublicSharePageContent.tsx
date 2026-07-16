"use client";

import { useEffect, useMemo, useState, type ReactNode } from "react";
import { useParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { useTranslations } from "next-intl";
import { ApiError } from "@/shared/api/client";
import { EmptyState, PageLoadingState } from "@/components/ui";
import { RouteSummaryCard } from "@/components/routes/RouteSummaryCard";
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
  toExportWeatherSummary,
  type ExportTrip
} from "@/lib/export/trip-export-adapter";
import { getDayDistanceSummaries } from "@/entities/itinerary/model/distance-utils";
import { cn } from "@/lib/utils";
import {
  clearStoredPublicShareToken,
  publicShareTokenExpiryKey,
  publicShareTokenKey
} from "../model/publicSharePageModel";
import { PublicTripSummaryCard } from "./PublicTripSummaryCard";
import { PublicShareHeader } from "./PublicShareHeader";
import { PublicShareHero } from "./PublicShareHero";
import { PublicShareItinerary } from "./PublicShareItinerary";
import { PublicShareMap } from "./PublicShareMap";
import { PublicShareUnlock } from "./PublicShareUnlock";
import { PublicShareUnavailableState } from "./PublicShareUnavailableState";
import { instrumentSans, newsreader } from "./fonts";

export function PublicSharePageContent() {
  const t = useTranslations("publicShare");
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
  // Weather + distance cards are intentionally absent from the shared mock, but
  // their data is still summarized here so the header Export (PDF/ICS) carries
  // weather and per-day distances just like the private trip export does.
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
    } catch {
      clearStoredPublicShareToken(shareToken);
      setPublicShareAccessToken(null);
      setUnlockError(t("wrongPassword"));
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
      <PublicShareShell>
        <div className="mx-auto max-w-[1080px] px-6 py-10 sm:px-10">
          <PageLoadingState cardCount={3} label={t("loading")} />
        </div>
      </PublicShareShell>
    );
  }

  if (publicShareStatusQuery.isError) {
    return (
      <PublicShareShell>
        <PublicShareUnavailableState
          onRetry={() => void publicShareStatusQuery.refetch()}
          retrying={publicShareStatusQuery.isFetching}
        />
      </PublicShareShell>
    );
  }

  if (status?.passwordRequired && !publicShareAccessToken) {
    return (
      <PublicShareShell>
        <PublicShareUnlock error={unlockError} loading={unlockLoading} onUnlock={handleUnlock} />
      </PublicShareShell>
    );
  }

  if (publicTripQuery.isError || !publicTripQuery.data) {
    if (
      status?.passwordRequired &&
      publicTripQuery.error instanceof ApiError &&
      publicTripQuery.error.status === 401
    ) {
      return (
        <PublicShareShell>
          <PublicShareUnlock error={unlockError} loading={unlockLoading} onUnlock={handleUnlock} />
        </PublicShareShell>
      );
    }
    return (
      <PublicShareShell>
        <PublicShareUnavailableState expired={status?.expired} />
      </PublicShareShell>
    );
  }

  const trip = publicTripQuery.data;
  const itinerary = trip.itinerary ?? null;

  return (
    // Match the original gate (exportTrip && itinerary): a shared DRAFT/PROCESSING
    // trip without an itinerary shows no Export control rather than a pill that
    // opens to two disabled items.
    <PublicShareShell exportTrip={itinerary ? exportTrip : null}>
      <div className="mx-auto max-w-[1080px] px-6 pb-[72px] pt-8 sm:px-10">
        <PublicShareHero trip={trip} />

        <div className="mt-7 grid grid-cols-1 items-start gap-7 lg:grid-cols-[300px_minmax(0,1fr)]">
          <aside className="flex flex-col gap-[18px]">
            <PublicTripSummaryCard trip={trip} />
            {itinerary ? (
              <PublicShareMap itinerary={itinerary} startDate={trip.startDate} />
            ) : null}
          </aside>

          <section className="flex min-w-0 flex-col gap-5">
            {trip.route ? (
              <RouteSummaryCard
                canEditTransport={false}
                currency={trip.itinerary?.currency ?? "EUR"}
                itinerary={itinerary}
                online={false}
                route={trip.route}
                totalDays={trip.days}
                travelers={trip.travelers ?? 1}
              />
            ) : null}
            {itinerary ? (
              <PublicShareItinerary itinerary={itinerary} />
            ) : (
              <EmptyState
                className="rounded-[18px] border-sand-300 bg-white"
                description={t("noItineraryDescription")}
                title={t("noItineraryTitle")}
              />
            )}
            <p className="text-[12.5px] text-[#A08D78]">{t("disclaimer")}</p>
          </section>
        </div>
      </div>
    </PublicShareShell>
  );
}

function PublicShareShell({
  exportTrip,
  children
}: {
  exportTrip?: ExportTrip | null;
  children: ReactNode;
}) {
  return (
    <div
      className={cn(
        newsreader.variable,
        instrumentSans.variable,
        "min-h-screen bg-sand-50 font-instrument text-cocoa-700 selection:bg-[#F0D9CC]"
      )}
    >
      <PublicShareHeader exportTrip={exportTrip} />
      {children}
    </div>
  );
}
