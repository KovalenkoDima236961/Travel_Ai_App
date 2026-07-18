"use client";

import Link from "next/link";
import { useTranslations } from "next-intl";
import { useTripRecapStatus } from "@/hooks/useTripRecap";

export function TripRecapStatusCard({ tripId }: { tripId: string }) {
  const t = useTranslations("recap");
  const { data: status } = useTripRecapStatus(tripId);
  if (!status?.eligible) return null;
  return <article className="rounded-3xl border border-clay/20 bg-[#FFF8F3] p-5"><p className="text-sm font-semibold text-clay">{t("eyebrow")}</p><h2 className="mt-1 font-newsreader text-2xl text-cocoa-900">{status.hasRecap ? t("cardReady") : t("cardStart")}</h2><p className="mt-2 text-sm text-cocoa-600">{status.hasRecap ? t("cardReadyDescription") : t("cardStartDescription")}</p><Link className="mt-4 inline-flex text-sm font-semibold text-clay underline underline-offset-4" href={`/trips/${tripId}/recap`}>{status.hasRecap ? t("open") : t("generate")}</Link></article>;
}
