"use client";

import { Button } from "@/shared/ui/button";
import { useTranslations } from "next-intl";
import type { TravelDayTimelineItem } from "@/types/travel-day";

export function TimelineItemCard({ item, canUpdate, onStatus, onMap, busy }: { item: TravelDayTimelineItem; canUpdate: boolean; onStatus: (item: TravelDayTimelineItem, status: "done" | "skipped" | "delayed") => void; onMap: (item: TravelDayTimelineItem) => void; busy?: boolean }) {
  const t = useTranslations("travelDay");
  return <li className="rounded-2xl border border-sand-300 bg-white p-4"><div className="flex gap-3"><p className="w-12 shrink-0 pt-0.5 text-sm font-semibold text-clay">{item.startTime || t("anyTime")}</p><div className="min-w-0 flex-1"><div className="flex items-start justify-between gap-2"><div><h3 className="font-semibold text-cocoa-900">{item.title}</h3><p className="mt-1 text-sm text-cocoa-500">{item.locationName || item.type}</p></div><span className="rounded-full bg-sand-100 px-2 py-1 text-xs font-semibold capitalize text-cocoa-600">{item.travelStatus.status}</span></div>{item.selectedTransport ? <p className="mt-2 text-sm text-cocoa-600">{item.selectedTransport.mode} · {item.selectedTransport.operatorName || item.selectedTransport.serviceName || t("selectedTransport")}</p> : null}{item.verification?.[0] ? <p className="mt-2 text-sm text-[#9B4E29]">{item.verification[0].message}</p> : null}<div className="mt-3 flex flex-wrap gap-2"><Button onClick={() => onMap(item)} size="sm" variant="ghost">{t("openMap")}</Button>{canUpdate && item.travelStatus.status !== "done" ? <Button disabled={busy} onClick={() => onStatus(item, "done")} size="sm">{t("markDone")}</Button> : null}{canUpdate ? <Button disabled={busy} onClick={() => onStatus(item, "skipped")} size="sm" variant="ghost">{t("markSkipped")}</Button> : null}{canUpdate ? <Button disabled={busy} onClick={() => onStatus(item, "delayed")} size="sm" variant="ghost">{t("markDelayed")}</Button> : null}</div></div></div></li>;
}
