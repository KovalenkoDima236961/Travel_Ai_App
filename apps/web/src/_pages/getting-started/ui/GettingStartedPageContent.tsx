"use client";

import { useTranslations } from "next-intl";
import { OnboardingExperience } from "@/components/onboarding/OnboardingExperience";
import { cn } from "@/shared/lib/cn";
import { instrumentSans, newsreader } from "@/_pages/settings/ui/fonts";

export function GettingStartedPageContent() {
  const t = useTranslations("onboarding");
  return (
    <div className={cn(newsreader.variable, instrumentSans.variable, "min-h-screen bg-sand-50 font-instrument text-cocoa-700")}>
      <div className="mx-auto max-w-[920px] px-6 pb-20 pt-12 sm:px-10">
        <div className="mb-8 max-w-2xl">
          <h1 className="font-newsreader text-[40px] font-medium tracking-[-0.02em] text-cocoa-900">{t("pageTitle")}</h1>
          <p className="mt-3 text-[15px] leading-[1.65] text-cocoa-500">{t("pageDescription")}</p>
        </div>
        <OnboardingExperience />
      </div>
    </div>
  );
}
