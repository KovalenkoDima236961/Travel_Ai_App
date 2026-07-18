"use client";

import Link from "next/link";
import { useTranslations } from "next-intl";
import type { TravelDaySummary } from "@/types/travel-day";

export function TravelDayHeader({ summary }: { summary: TravelDaySummary }) {
	const t = useTranslations("travelDay");
  return (
    <header className="flex items-start justify-between gap-4">
      <div>
        <p className="text-xs font-semibold uppercase tracking-[0.14em] text-clay">{t("travelMode")}</p>
        <h1 className="mt-1 font-newsreader text-3xl font-semibold text-cocoa-900">{summary.trip.title}</h1>
        <p className="mt-1 text-sm text-cocoa-500">{summary.today.title} · {summary.date}</p>
      </div>
      <Link className="rounded-full border border-sand-300 bg-white px-3 py-2 text-sm font-semibold text-cocoa-700" href={`/trips/${summary.tripId}`}>
        {t("openPlanner")}
      </Link>
    </header>
  );
}
