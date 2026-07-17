"use client";

import Link from "next/link";
import { useTranslations } from "next-intl";
import { useTripSetupChecklist } from "@/hooks/useTripSetupChecklist";
import type { TripSetupInput, TripSetupItemStatus } from "@/lib/onboarding/trip-setup";

export function TripSetupChecklist(props: TripSetupInput) {
  const t = useTranslations("onboarding.setupChecklist");
  const checklist = useTripSetupChecklist(props);
  if (!checklist.show) {
    return null;
  }

  const progress = Math.round((checklist.completedCount / checklist.items.length) * 100);
  return (
    <section className="rounded-[18px] border border-[#DCE8DD] bg-[#F7FAF6] p-5" aria-labelledby="trip-setup-title">
      <div className="flex items-start justify-between gap-4">
        <div>
          <p className="text-[12px] font-semibold uppercase tracking-[0.08em] text-[#58705E]">{t("eyebrow")}</p>
          <h2 id="trip-setup-title" className="mt-1 font-newsreader text-[22px] font-semibold text-cocoa-900">{t("title")}</h2>
          <p className="mt-1 text-[13.5px] text-cocoa-500">{t("description")}</p>
        </div>
        <button type="button" onClick={checklist.dismiss} aria-label={t("dismissLabel")} className="rounded-full px-2.5 py-1 text-[12.5px] font-semibold text-cocoa-500 transition hover:bg-white">{t("dismiss")}</button>
      </div>
      <div className="mt-4 h-1.5 overflow-hidden rounded-full bg-white" role="progressbar" aria-label={t("progress", { completed: checklist.completedCount, total: checklist.items.length })} aria-valuemin={0} aria-valuemax={100} aria-valuenow={progress}>
        <div className="h-full rounded-full bg-[#3E6B5A]" style={{ width: `${progress}%` }} />
      </div>
      <ul className="mt-4 grid gap-2 sm:grid-cols-2">
        {checklist.items.map((item) => (
          <li key={item.id}>
            <Link href={item.href} className="flex items-center justify-between gap-3 rounded-[12px] border border-[#E3EBE2] bg-white px-3.5 py-3 transition hover:border-[#BFCFBE]">
              <span className="flex min-w-0 items-center gap-2.5">
                <span aria-hidden="true" className={item.status === "complete" ? "text-[#3E6B5A]" : item.status === "needs_attention" ? "text-clay-deep" : "text-[#A08D78]"}>{item.status === "complete" ? "✓" : "○"}</span>
                <span className="truncate text-[13.5px] font-semibold text-cocoa-800">{t(`items.${item.id}`)}</span>
              </span>
              <span className={`shrink-0 rounded-full px-2 py-1 text-[10.5px] font-semibold ${statusClasses[item.status]}`}>{t(`status.${item.status}`)}</span>
            </Link>
          </li>
        ))}
      </ul>
    </section>
  );
}

const statusClasses: Record<TripSetupItemStatus, string> = {
  complete: "bg-[#EAF2ED] text-[#2F5546]",
  recommended: "bg-[#F4EDE4] text-cocoa-600",
  optional: "bg-sand-100 text-cocoa-400",
  needs_attention: "bg-clay-tint text-clay-deep"
};
