"use client";

import Link from "next/link";
import { useTranslations } from "next-intl";
import { useAuth } from "@/components/auth/AuthProvider";
import { useOnboardingState } from "@/hooks/useOnboardingState";
import { TRIP_START_OPTIONS } from "./ChooseTripStart";
import { StartOptionCard } from "./StartOptionCard";

export function FirstRunDashboard() {
  const t = useTranslations("onboarding.firstRun");
  const chooseT = useTranslations("onboarding.chooseStart");
  const { user } = useAuth();
  const onboarding = useOnboardingState(user?.id);

  return (
    <section className="mt-9 rounded-[22px] border border-sand-300 bg-white px-6 py-8 shadow-[0_14px_36px_rgba(34,26,20,0.05)] sm:px-8" aria-labelledby="first-run-title">
      <div className="flex flex-col gap-5 sm:flex-row sm:items-start sm:justify-between">
        <div className="max-w-2xl">
          <p className="text-[12px] font-semibold uppercase tracking-[0.09em] text-[#3E6B5A]">{t("eyebrow")}</p>
          <h2 id="first-run-title" className="mt-2 font-newsreader text-[30px] font-semibold text-cocoa-900">
            {t("title")}
          </h2>
          <p className="mt-2 text-[14.5px] leading-[1.65] text-cocoa-500">{t("description")}</p>
        </div>
        <div className="flex shrink-0 flex-wrap gap-2">
          <Link href="/getting-started?step=preferences" className="inline-flex h-10 items-center rounded-full border border-sand-400 px-4 text-[13px] font-semibold text-cocoa-700 transition hover:border-sand-600">
            {t("preferencesAction")}
          </Link>
          <button type="button" onClick={onboarding.skip} className="h-10 rounded-full px-4 text-[13px] font-semibold text-cocoa-500 transition hover:bg-sand-100">
            {t("skip")}
          </button>
        </div>
      </div>

      <div className="mt-7 grid gap-4 sm:grid-cols-2 xl:grid-cols-5">
        {TRIP_START_OPTIONS.map((option) => (
          <StartOptionCard
            key={option.id}
            href={option.href}
            title={chooseT(`${option.id}.title`)}
            description={chooseT(`${option.id}.shortDescription`)}
            bestFor={chooseT(`${option.id}.bestFor`)}
            estimatedTime={chooseT(`${option.id}.time`)}
            onSelect={() => onboarding.markStepComplete("choose_start", "create_first_trip")}
          />
        ))}
        <StartOptionCard
          href="/demo-trip"
          title={t("demo.title")}
          description={t("demo.description")}
          bestFor={t("demo.bestFor")}
          estimatedTime={t("demo.time")}
        />
      </div>
    </section>
  );
}

export function HelpfulTripsEmptyState() {
  const t = useTranslations("onboarding.firstRun");
  const chooseT = useTranslations("onboarding.chooseStart");
  return (
    <section className="mt-9 rounded-[20px] border border-dashed border-sand-400 bg-white/60 px-7 py-9" aria-labelledby="empty-trip-options-title">
      <h2 id="empty-trip-options-title" className="font-newsreader text-[25px] font-semibold text-cocoa-900">{t("title")}</h2>
      <p className="mt-2 max-w-2xl text-[14.5px] text-cocoa-500">{t("description")}</p>
      <div className="mt-6 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        {TRIP_START_OPTIONS.map((option) => (
          <StartOptionCard
            key={option.id}
            href={option.href}
            title={chooseT(`${option.id}.title`)}
            description={chooseT(`${option.id}.shortDescription`)}
            bestFor={chooseT(`${option.id}.bestFor`)}
            estimatedTime={chooseT(`${option.id}.time`)}
          />
        ))}
      </div>
      <Link href="/demo-trip" className="mt-5 inline-flex text-[13.5px] font-semibold text-clay-deep hover:text-clay">
        {t("demo.title")} →
      </Link>
    </section>
  );
}
