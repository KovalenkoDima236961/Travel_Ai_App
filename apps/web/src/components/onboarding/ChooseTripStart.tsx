"use client";

import { useTranslations } from "next-intl";
import { StartOptionCard } from "./StartOptionCard";

type ChooseTripStartProps = {
  onSelect?: () => void;
};

export const TRIP_START_OPTIONS = [
  { id: "known", href: "/trips/new?mode=destination" },
  { id: "discovery", href: "/trips/new?mode=discovery" },
  { id: "template", href: "/templates?firstRun=true" },
  { id: "route", href: "/trips/new?mode=route" }
] as const;

export function ChooseTripStart({ onSelect }: ChooseTripStartProps) {
  const t = useTranslations("onboarding.chooseStart");
  return (
    <section aria-labelledby="choose-trip-start-title">
      <h2 id="choose-trip-start-title" className="font-newsreader text-[29px] font-semibold text-cocoa-900">
        {t("title")}
      </h2>
      <p className="mt-2 max-w-2xl text-[14.5px] leading-[1.6] text-cocoa-500">
        {t("description")}
      </p>
      <div className="mt-6 grid gap-4 sm:grid-cols-2">
        {TRIP_START_OPTIONS.map((option) => (
          <StartOptionCard
            key={option.id}
            href={option.href}
            title={t(`${option.id}.title`)}
            description={t(`${option.id}.description`)}
            bestFor={t(`${option.id}.bestFor`)}
            estimatedTime={t(`${option.id}.time`)}
            onSelect={onSelect}
          />
        ))}
      </div>
    </section>
  );
}
