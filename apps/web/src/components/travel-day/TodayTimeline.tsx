"use client";

import { useTranslations } from "next-intl";
import { TimelineItemCard } from "./TimelineItemCard";
import type { TravelDayTimelineItem } from "@/types/travel-day";

export function TodayTimeline({ items, canUpdate, onStatus, onMap, busy }: { items: TravelDayTimelineItem[]; canUpdate: boolean; onStatus: (item: TravelDayTimelineItem, status: "done" | "skipped" | "delayed") => void; onMap: (item: TravelDayTimelineItem) => void; busy?: boolean }) {
  const t = useTranslations("travelDay");
  return <section><div className="mb-3 flex items-center justify-between"><h2 className="font-newsreader text-2xl font-semibold text-cocoa-900">{t("timeline")}</h2><span className="text-sm text-cocoa-500">{t("items", { count: items.length })}</span></div>{items.length ? <ol className="space-y-3">{items.map((item) => <TimelineItemCard key={`${item.dayNumber}-${item.itemIndex}`} {...{ item, canUpdate, onStatus, onMap, busy }} />)}</ol> : <p className="rounded-2xl border border-dashed border-sand-400 p-5 text-sm text-cocoa-500">{t("noPlan")}</p>}</section>;
}
