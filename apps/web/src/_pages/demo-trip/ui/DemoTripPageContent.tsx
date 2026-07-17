"use client";

import Link from "next/link";
import { useTranslations } from "next-intl";
import { cn } from "@/shared/lib/cn";
import { instrumentSans, newsreader } from "@/_pages/trips/ui/fonts";

export function DemoTripPageContent() {
  const t = useTranslations("onboarding.demo");
  const itinerary = [
    { time: "09:00", title: t("sample.breakfastTitle"), detail: t("sample.breakfastDetail") },
    { time: "11:00", title: t("sample.walkTitle"), detail: t("sample.walkDetail") },
    { time: "14:30", title: t("sample.castleTitle"), detail: t("sample.castleDetail") }
  ];
  const checklist = [t("sample.trainTickets"), t("sample.address"), t("sample.jacket")];
  return (
    <div className={cn(newsreader.variable, instrumentSans.variable, "min-h-screen bg-sand-50 font-instrument text-cocoa-700")}>
      <div className="mx-auto max-w-[1180px] px-6 pb-20 pt-10 sm:px-10">
        <div className="rounded-[16px] border border-[#DCE8DD] bg-[#F2F7F1] px-5 py-4" role="status" aria-label={t("readOnlyLabel")}>
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <span className="rounded-full bg-[#3E6B5A] px-2.5 py-1 text-[11px] font-bold uppercase tracking-[0.08em] text-white">{t("badge")}</span>
              <p className="mt-2 text-[14px] font-semibold text-[#38543F]">{t("readOnly")}</p>
              <p className="mt-1 text-[13px] text-[#58705E]">{t("privacy")}</p>
            </div>
            <Link href="/trips/new?mode=destination" className="inline-flex h-10 items-center rounded-full bg-clay px-5 text-[13.5px] font-semibold text-sand-100">{t("createAction")}</Link>
          </div>
        </div>

        <header className="mt-8 rounded-[22px] border border-sand-300 bg-white p-7 sm:p-9">
          <p className="text-[12px] font-semibold uppercase tracking-[0.09em] text-[#3E6B5A]">{t("commandCenter")}</p>
          <h1 className="mt-2 font-newsreader text-[38px] font-semibold tracking-[-0.02em] text-cocoa-900">{t("tripTitle")}</h1>
          <p className="mt-2 text-[14.5px] text-cocoa-500">{t("tripMeta")}</p>
          <div className="mt-6 grid gap-3 sm:grid-cols-3">
            <Metric label={t("health")} value="86 / 100" tone="green" />
            <Metric label={t("budgetConfidence")} value="78 / 100" tone="sand" />
            <Metric label={t("setupProgress")} value="5 / 7" tone="clay" />
          </div>
        </header>

        <div className="mt-5 grid gap-5 lg:grid-cols-[minmax(0,1.35fr)_minmax(300px,0.65fr)]">
          <section className="rounded-[20px] border border-sand-300 bg-white p-6" aria-labelledby="demo-itinerary-title">
            <h2 id="demo-itinerary-title" className="font-newsreader text-[24px] font-semibold text-cocoa-900">{t("itinerary")}</h2>
            <p className="mt-1 text-[13px] text-cocoa-400">{t("itineraryDescription")}</p>
            <div className="mt-5 space-y-3">
              {itinerary.map((item) => (
                <article key={item.time} className="flex gap-4 rounded-[14px] bg-sand-50 p-4">
                  <span className="text-[12.5px] font-bold text-clay-deep">{item.time}</span>
                  <div><h3 className="text-[14px] font-semibold text-cocoa-900">{item.title}</h3><p className="mt-1 text-[12.5px] leading-[1.55] text-cocoa-500">{item.detail}</p></div>
                </article>
              ))}
            </div>
          </section>

          <div className="space-y-5">
            <section className="rounded-[20px] border border-sand-300 bg-white p-6" aria-labelledby="demo-route-title">
              <h2 id="demo-route-title" className="font-newsreader text-[22px] font-semibold text-cocoa-900">{t("route")}</h2>
              <div className="mt-5 space-y-1">
                <RouteStop label="Bratislava" detail={t("sample.twoNights")} />
                <div className="ml-[13px] h-8 border-l border-dashed border-sand-500 pl-5 text-[11px] font-semibold text-cocoa-400">{t("sample.trainOneHour")}</div>
                <RouteStop label="Vienna" detail={t("sample.twoNights")} />
              </div>
            </section>

            <section className="rounded-[20px] border border-sand-300 bg-white p-6" aria-labelledby="demo-checklist-title">
              <h2 id="demo-checklist-title" className="font-newsreader text-[22px] font-semibold text-cocoa-900">{t("checklist")}</h2>
              <ul className="mt-4 space-y-2">
                {checklist.map((item, index) => <li key={item} className="flex items-center gap-2.5 text-[13px] text-cocoa-700"><span aria-hidden="true" className={index < 2 ? "text-[#3E6B5A]" : "text-cocoa-300"}>{index < 2 ? "✓" : "○"}</span>{item}</li>)}
              </ul>
            </section>
          </div>
        </div>

        <section className="mt-5 rounded-[20px] border border-sand-300 bg-white p-6" aria-labelledby="demo-health-title">
          <h2 id="demo-health-title" className="font-newsreader text-[22px] font-semibold text-cocoa-900">{t("healthAndBudget")}</h2>
          <div className="mt-4 grid gap-3 sm:grid-cols-2">
            <DemoNotice title={t("healthNoticeTitle")} description={t("healthNoticeDescription")} />
            <DemoNotice title={t("budgetNoticeTitle")} description={t("budgetNoticeDescription")} />
          </div>
        </section>
      </div>
    </div>
  );
}

function Metric({ label, value, tone }: { label: string; value: string; tone: "green" | "sand" | "clay" }) {
  const classes = tone === "green" ? "bg-[#F2F7F1] text-[#38543F]" : tone === "clay" ? "bg-clay-tint text-clay-deep" : "bg-sand-100 text-cocoa-700";
  return <div className={`rounded-[14px] px-4 py-3 ${classes}`}><p className="text-[11px] font-semibold uppercase tracking-[0.08em] opacity-70">{label}</p><p className="mt-1 text-[18px] font-semibold">{value}</p></div>;
}

function RouteStop({ label, detail }: { label: string; detail: string }) {
  return <div className="flex items-center gap-3"><span className="h-7 w-7 rounded-full border-[7px] border-[#EAF2ED] bg-[#3E6B5A]" /><div><p className="text-[13.5px] font-semibold text-cocoa-900">{label}</p><p className="text-[11.5px] text-cocoa-400">{detail}</p></div></div>;
}

function DemoNotice({ title, description }: { title: string; description: string }) {
  return <article className="rounded-[14px] bg-sand-50 p-4"><h3 className="text-[14px] font-semibold text-cocoa-900">{title}</h3><p className="mt-1 text-[12.5px] leading-[1.55] text-cocoa-500">{description}</p></article>;
}
