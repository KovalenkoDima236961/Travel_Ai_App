"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import { useTranslations } from "next-intl";
import {
  TripCreateModeSelector,
  type TripCreateMode
} from "@/components/trip-discovery/TripCreateModeSelector";
import { TripDiscoveryExperience } from "@/components/trip-discovery/TripDiscoveryExperience";
import { cn } from "@/shared/lib/cn";
import { CreateTripForm } from "./CreateTripForm";
import { CreateTripHeader } from "./CreateTripHeader";
import { ArrowLeftIcon, ArrowRightIcon } from "./icons";
import { instrumentSans, newsreader } from "./fonts";

const STEPS = [
  {
    title: "Trip is created as a draft",
    body: "You can adjust any detail before generating."
  },
  {
    title: "AI drafts the itinerary",
    body: "Usually under a minute — with places, routes, weather, and costs."
  },
  {
    title: "Review and refine",
    body: "Edit days, swap places, share, and split costs."
  }
];

export function NewTripPageContent() {
  const translate = useTranslations("trips");
  const discovery = useTranslations("tripDiscovery");
  const searchParams = useSearchParams();
  const requestedMode = searchParams.get("mode");
  const [mode, setMode] = useState<TripCreateMode>(requestedMode === "discovery" ? "discover" : "known");

  useEffect(() => {
    setMode(requestedMode === "discovery" ? "discover" : "known");
  }, [requestedMode]);
  return (
    <div
      className={cn(
        newsreader.variable,
        instrumentSans.variable,
        "min-h-screen bg-sand-50 font-instrument text-cocoa-700 selection:bg-[#F0D9CC]"
      )}
    >
      <CreateTripHeader />

      {/* Content region is a div, not <main> — the root layout already provides
          the single <main> landmark. */}
      <div className="mx-auto max-w-[1120px] px-4 pb-20 pt-8 sm:px-10 sm:pt-12">
        <Link
          href="/trips"
          className="inline-flex items-center gap-2 text-[14px] font-medium text-clay-deep transition hover:text-clay"
        >
          <ArrowLeftIcon className="h-[15px] w-[15px]" />
          {translate("backToTrips")}
        </Link>

        <h1 className="mt-[18px] font-newsreader text-[38px] font-medium leading-[1.05] tracking-[-0.02em] text-cocoa-900 sm:text-[44px]">
          {translate("whereNext")}
        </h1>
        <p className="mt-3.5 max-w-[560px] text-[16px] leading-[1.6] text-cocoa-500">
          {mode === "known" ? translate("createDescription") : discovery("pageDescription")}
        </p>

        <div className="mt-8 max-w-[760px]">
          <TripCreateModeSelector value={mode} onChange={setMode} />
        </div>

        {mode === "known" ? (
          <div className="mt-8 grid grid-cols-1 items-start gap-8 lg:grid-cols-[minmax(0,1fr)_320px]">
            <CreateTripForm initialTripMode={requestedMode === "route" ? "multi_destination" : "single_destination"} />

            <aside className="flex flex-col gap-5">
            <div className="rounded-[20px] border border-sand-300 bg-white px-[26px] py-6">
              <h2 className="font-newsreader text-[19px] font-semibold text-cocoa-900">
                What happens next
              </h2>
              <div className="mt-[18px] flex flex-col">
                {STEPS.map((step, index) => (
                  <div key={step.title} className="flex gap-3.5">
                    <div className="flex flex-col items-center">
                      <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-clay-tint text-[12.5px] font-bold text-clay-dark">
                        {index + 1}
                      </span>
                      {index < STEPS.length - 1 ? (
                        <span className="my-1 w-px flex-1 bg-sand-300" />
                      ) : null}
                    </div>
                    <div className={index < STEPS.length - 1 ? "pb-[18px]" : ""}>
                      <p className="text-[14px] font-semibold text-cocoa-900">{step.title}</p>
                      <p className="mt-1 text-[13px] leading-[1.55] text-cocoa-400">{step.body}</p>
                    </div>
                  </div>
                ))}
              </div>
            </div>

            <Link
              href="/templates"
              className="block rounded-[20px] border border-sand-300 bg-[#F4EDE4] px-[26px] py-[22px] transition hover:border-sand-400"
            >
              <div className="flex items-center justify-between gap-3">
                <div>
                  <p className="font-newsreader text-[18px] font-semibold text-cocoa-900">
                    Start from a template
                  </p>
                  <p className="mt-1.5 text-[13.5px] leading-[1.55] text-cocoa-500">
                    Reuse a past trip&apos;s structure — or adapt it to a new city with AI.
                  </p>
                </div>
                <ArrowRightIcon className="h-[18px] w-[18px] shrink-0 text-clay-dark" />
              </div>
            </Link>
            </aside>
          </div>
        ) : (
          <div className="mt-8">
            <TripDiscoveryExperience />
          </div>
        )}
      </div>
    </div>
  );
}
