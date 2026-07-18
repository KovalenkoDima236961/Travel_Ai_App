"use client";

import { useTranslations } from "next-intl";
import type { TravelDaySummary } from "@/types/travel-day";
export function OfflineTravelStatus({ offlineCopy, cachedAt, summary }: { offlineCopy: boolean; cachedAt?: string | null; summary: TravelDaySummary }) { const t = useTranslations("travelDay"); if (!offlineCopy) return null; return <div className="rounded-2xl border border-[#D6DEE8] bg-[#F4F7FA] p-3 text-sm text-[#536171]">{t("offlineCopy")} · {t("lastCached")} {cachedAt ? new Date(cachedAt).toLocaleString() : t("recently")}. {t("staleNotice")}</div>; }
