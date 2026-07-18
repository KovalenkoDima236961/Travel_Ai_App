"use client";

import { Button } from "@/shared/ui/button";
import { useTranslations } from "next-intl";
import type { TravelDayNowNext, TravelDayTimelineItem } from "@/types/travel-day";

type Props = { nowNext: TravelDayNowNext; canUpdate: boolean; onStatus: (item: TravelDayTimelineItem, status: "done" | "skipped" | "delayed") => void; onMap: (item: TravelDayTimelineItem) => void; busy?: boolean };

export function NowNextCard({ nowNext, canUpdate, onStatus, onMap, busy }: Props) {
	const t = useTranslations("travelDay");
  const item = nowNext.currentItem ?? nowNext.nextItem;
  if (!item) return <div className="rounded-[22px] border border-[#DCE8DD] bg-[#F2F7F1] p-5 text-sm font-medium text-[#38543F]">{t("doneToday")}</div>;
  const label = nowNext.currentItem ? t("now") : t("nextUp");
  return <section className="rounded-[24px] bg-cocoa-900 p-5 text-sand-50 shadow-[0_18px_36px_rgba(61,45,37,0.2)]">
    <p className="text-xs font-semibold uppercase tracking-[0.16em] text-[#E9B8A6]">{label}</p>
    <div className="mt-3 flex items-start justify-between gap-3"><div><p className="text-sm text-sand-300">{item.startTime || "Flexible"} · {item.type}</p><h2 className="mt-1 font-newsreader text-3xl font-semibold leading-tight">{item.title}</h2><p className="mt-2 text-sm text-sand-200">{item.locationName || item.place?.address || "Location to confirm"}</p></div><span className="rounded-full bg-white/15 px-2.5 py-1 text-xs font-semibold capitalize">{item.travelStatus.status}</span></div>
    {item.verification?.[0] ? <p className="mt-4 rounded-xl bg-[#7A5727]/40 p-3 text-sm">{item.verification[0].message}</p> : null}
    <div className="mt-5 grid grid-cols-2 gap-2"><Button className="min-h-12" onClick={() => onMap(item)} variant="secondary">{t("openMap")}</Button>{canUpdate ? <Button className="min-h-12" disabled={busy} onClick={() => onStatus(item, "done")}>{t("markDone")}</Button> : null}</div>
    {nowNext.afterNextItems.length ? <div className="mt-4 border-t border-white/15 pt-3"><p className="text-xs font-semibold uppercase tracking-[0.12em] text-sand-300">{t("afterThat")}</p><ul className="mt-2 space-y-1 text-sm text-sand-100">{nowNext.afterNextItems.slice(0, 3).map((next) => <li key={`${next.dayNumber}-${next.itemIndex}`}>{next.startTime ? `${next.startTime} · ` : ""}{next.title}</li>)}</ul></div> : null}
  </section>;
}
