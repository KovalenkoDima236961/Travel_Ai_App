"use client";

import { useTranslations } from "next-intl";
import { Button } from "@/shared/ui/button";
import type { TripRecapStatusResponse } from "@/types/recap";

export function RecapStatusEmptyState({ status, onGenerate, pending }: { status: TripRecapStatusResponse; onGenerate: () => void; pending: boolean }) {
  const t = useTranslations("recap");
  const canGenerate = status.eligible && status.canGenerate;
  return <section className="rounded-3xl border border-sand-300 bg-white p-7 shadow-sm"><p className="text-sm font-semibold text-clay">{t("eyebrow")}</p><h1 className="mt-2 font-newsreader text-4xl text-cocoa-900">{t("emptyTitle")}</h1><p className="mt-3 max-w-2xl text-cocoa-600">{canGenerate ? t("emptyDescription") : status.reason || t("notReady")}</p>{canGenerate ? <Button className="mt-5" disabled={pending} onClick={onGenerate}>{pending ? t("generating") : t("generate")}</Button> : null}</section>;
}
