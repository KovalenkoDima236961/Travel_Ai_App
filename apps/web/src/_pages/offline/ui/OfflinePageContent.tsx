"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useAuth } from "@/components/auth/AuthProvider";
import { listCachedTrips } from "@/lib/offline/trip-cache";
import type { CachedTripRecord } from "@/lib/offline/types";
import { cn } from "@/shared/lib/cn";
import { formatDate } from "@/lib/utils";
import { instrumentSans, newsreader } from "./fonts";
import {
  ArrowPathIcon,
  ChevronRightIcon,
  GlobeIcon,
  MapPinIcon,
  XCircleIcon
} from "./icons";

// The PWA offline fallback (precached in public/sw.js and served for any failed
// navigation while offline). Restyled to the warm editorial redesign. The
// "Available offline" list reads the real IndexedDB cache via listCachedTrips —
// but auth is validated over the network, so while genuinely offline the user is
// usually null and the honest empty state is the common render. There is no link
// to /offline-trips here on purpose: only /offline is precached, so a navigation
// there while offline just bounces back to this page.
export function OfflinePageContent() {
  const { user, isLoading: authLoading } = useAuth();
  const [trips, setTrips] = useState<CachedTripRecord[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (authLoading) {
      return;
    }

    let active = true;
    const userId = user?.id;

    if (!userId) {
      setTrips([]);
      setLoading(false);
      return;
    }

    setLoading(true);
    listCachedTrips(userId)
      .then((records) => {
        if (active) {
          setTrips(records);
        }
      })
      .catch(() => {
        if (active) {
          setTrips([]);
        }
      })
      .finally(() => {
        if (active) {
          setLoading(false);
        }
      });

    return () => {
      active = false;
    };
  }, [authLoading, user?.id]);

  return (
    <div
      className={cn(
        newsreader.variable,
        instrumentSans.variable,
        "flex min-h-screen items-center justify-center bg-sand-50 p-6 font-instrument text-cocoa-700 selection:bg-[#F0D9CC] sm:p-12"
      )}
    >
      <div className="w-full max-w-[760px]">
        <div className="flex items-center gap-2.5 text-cocoa-900">
          <span className="flex h-8 w-8 items-center justify-center rounded-full bg-clay text-sand-100">
            <GlobeIcon className="h-[18px] w-[18px]" />
          </span>
          <span className="font-newsreader text-[19px] font-semibold tracking-[-0.01em]">
            Travel AI Planner
          </span>
        </div>

        <div className="mt-10 flex items-center gap-3">
          <span className="flex h-12 w-12 items-center justify-center rounded-full bg-[#FDF0E3] text-[#96682A]">
            <XCircleIcon className="h-6 w-6" />
          </span>
          <span className="inline-flex items-center gap-[7px] rounded-full bg-[#FAEFDA] px-3.5 py-1.5 text-[12.5px] font-semibold text-[#96682A]">
            <span className="h-[7px] w-[7px] rounded-full bg-[#96682A]" />
            Offline
          </span>
        </div>

        <h1 className="mt-5 font-newsreader text-[48px] font-medium leading-[1.05] tracking-[-0.02em] text-cocoa-900">
          You&apos;re offline — but your trips aren&apos;t.
        </h1>
        <p className="mt-4 max-w-[520px] text-base leading-[1.6] text-cocoa-500">
          We can&apos;t reach the network right now. Any trip you&apos;ve opened before is saved on
          this device and ready to view.
        </p>

        <div className="mt-8 overflow-hidden rounded-[20px] border border-sand-300 bg-white">
          <p className="px-[22px] pb-3 pt-4 text-[12.5px] font-semibold uppercase tracking-[0.06em] text-[#A08D78]">
            Available offline
          </p>

          {loading ? (
            <p className="border-t border-sand-200 px-[22px] py-4 text-sm text-cocoa-400">
              Checking this device…
            </p>
          ) : trips.length > 0 ? (
            trips.map((record) => (
              <Link
                key={record.tripId}
                href={`/trips/${record.tripId}`}
                className="flex items-center justify-between gap-4 border-t border-sand-200 px-[22px] py-4 transition-colors hover:bg-sand-100"
              >
                <div className="flex items-center gap-3.5">
                  <span className="flex h-10 w-10 items-center justify-center rounded-[12px] bg-clay-tint text-clay-dark">
                    <MapPinIcon className="h-[19px] w-[19px]" />
                  </span>
                  <div>
                    <p className="text-[15px] font-semibold text-cocoa-900">
                      {record.trip.destination}
                    </p>
                    <p className="mt-0.5 text-[13px] text-cocoa-400">
                      {record.trip.days} {record.trip.days === 1 ? "day" : "days"} · saved{" "}
                      {formatDate(record.cachedAt, { month: "short", day: "numeric" })}
                    </p>
                  </div>
                </div>
                <ChevronRightIcon className="h-4 w-4 text-[#B09E8A]" />
              </Link>
            ))
          ) : (
            <p className="border-t border-sand-200 px-[22px] py-4 text-sm leading-6 text-cocoa-400">
              No trips are saved on this device yet. Trips you open while online are stored here
              automatically so you can view them offline.
            </p>
          )}
        </div>

        <button
          type="button"
          onClick={() => window.location.reload()}
          className="mt-7 inline-flex h-12 items-center gap-2.5 rounded-full bg-clay px-6 text-[15px] font-semibold text-sand-100 transition-colors hover:bg-clay-dark"
        >
          <ArrowPathIcon className="h-[17px] w-[17px]" />
          Try to reconnect
        </button>
      </div>
    </div>
  );
}
