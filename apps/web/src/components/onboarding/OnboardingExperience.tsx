"use client";

import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useRouter, useSearchParams } from "next/navigation";
import { useTranslations } from "next-intl";
import { useAuth } from "@/components/auth/AuthProvider";
import { useAppLanguage } from "@/components/i18n/I18nProvider";
import { ChooseTripStart } from "@/components/onboarding/ChooseTripStart";
import {
  OnboardingPreferenceWizard,
  wizardValuesToTravelStyles,
  type OnboardingPreferenceValues
} from "@/components/onboarding/OnboardingPreferenceWizard";
import { useOnboardingState } from "@/hooks/useOnboardingState";
import {
  getMyPreferences,
  getMyProfile,
  patchMyPreferences,
  updateMyProfile,
  userKeys
} from "@/lib/api/user";
import { getErrorMessage } from "@/lib/utils";
import type { OnboardingStep } from "@/lib/onboarding/state";

export function OnboardingExperience() {
  const t = useTranslations("onboarding");
  const { user } = useAuth();
  const { setLanguage } = useAppLanguage();
  const router = useRouter();
  const searchParams = useSearchParams();
  const queryClient = useQueryClient();
  const onboarding = useOnboardingState(user?.id);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const handledQueryRef = useRef(false);
  const profileQuery = useQuery({ queryKey: userKeys.profile(), queryFn: getMyProfile });
  const preferencesQuery = useQuery({ queryKey: userKeys.preferences(), queryFn: getMyPreferences });

  useEffect(() => {
    if (!onboarding.hydrated || handledQueryRef.current) {
      return;
    }
    handledQueryRef.current = true;
    if (searchParams.get("restart") === "true") {
      onboarding.restart();
      router.replace("/getting-started");
      return;
    }
    const requestedStep = searchParams.get("step");
    if (requestedStep && isOnboardingStep(requestedStep)) {
      onboarding.goToStep(requestedStep);
    }
  }, [onboarding, router, searchParams]);

  async function savePreferences(values: OnboardingPreferenceValues) {
    if (!profileQuery.data || !preferencesQuery.data) {
      return;
    }
    setSaving(true);
    setSaveError(null);
    try {
      const [profile, preferences] = await Promise.all([
        updateMyProfile({
          displayName: profileQuery.data.displayName,
          homeCity: values.homeCity || null,
          homeCountry: values.homeCountry || null,
          preferredCurrency: values.preferredCurrency,
          preferredLanguage: values.preferredLanguage
        }),
        patchMyPreferences({
          travelStyles: wizardValuesToTravelStyles(values),
          pace: values.pace,
          maxWalkingKmPerDay: values.maxWalkingKmPerDay,
          foodPreferences: values.foodPreferences,
          dietaryRestrictions: values.dietaryRestrictions,
          preferredTransport: values.preferredTransport,
          accommodationStyle: values.accommodationStyle,
          avoid: preferencesQuery.data.avoid
        })
      ]);
      queryClient.setQueryData(userKeys.profile(), profile);
      queryClient.setQueryData(userKeys.preferences(), preferences);
      setLanguage(values.preferredLanguage);
      onboarding.markStepComplete("preferences", "choose_start");
    } catch (error) {
      setSaveError(getErrorMessage(error, t("preferences.saveFailed")));
    } finally {
      setSaving(false);
    }
  }

  if (!onboarding.hydrated || profileQuery.isPending || preferencesQuery.isPending) {
    return <div className="rounded-[20px] border border-sand-300 bg-white p-8 text-[14px] text-cocoa-500">{t("loading")}</div>;
  }

  if (profileQuery.isError || preferencesQuery.isError) {
    return (
      <section className="rounded-[20px] border border-clay/30 bg-white p-8">
        <h2 className="font-newsreader text-[24px] font-semibold text-cocoa-900">{t("loadError.title")}</h2>
        <p className="mt-2 text-[14px] text-cocoa-500">{t("loadError.description")}</p>
        <button type="button" onClick={() => { void profileQuery.refetch(); void preferencesQuery.refetch(); }} className="mt-5 h-10 rounded-full bg-clay px-5 text-[13px] font-semibold text-white">{t("loadError.retry")}</button>
      </section>
    );
  }

  const step = onboarding.state.onboardingStep;
  if (step === "preferences" && profileQuery.data && preferencesQuery.data) {
    return (
      <OnboardingPreferenceWizard
        profile={profileQuery.data}
        preferences={preferencesQuery.data}
        isSaving={saving}
        errorMessage={saveError}
        onBack={() => onboarding.goToStep("welcome")}
        onSkip={() => onboarding.goToStep("choose_start")}
        onSave={savePreferences}
      />
    );
  }

  if (step === "choose_start" || step === "create_first_trip") {
    return (
      <div className="rounded-[22px] border border-sand-300 bg-white p-6 sm:p-8">
        <ChooseTripStart onSelect={() => onboarding.markStepComplete("choose_start", "create_first_trip")} />
        <button type="button" onClick={onboarding.skip} className="mt-6 text-[13.5px] font-semibold text-cocoa-500 hover:text-cocoa-900">{t("skipForNow")}</button>
      </div>
    );
  }

  if (step === "first_trip_setup") {
    return (
      <section className="rounded-[22px] border border-sand-300 bg-white p-8">
        <p className="text-[12px] font-semibold uppercase tracking-[0.09em] text-[#3E6B5A]">{t("setup.eyebrow")}</p>
        <h2 className="mt-2 font-newsreader text-[30px] font-semibold text-cocoa-900">{t("setup.title")}</h2>
        <p className="mt-3 max-w-2xl text-[14.5px] leading-[1.65] text-cocoa-500">{t("setup.description")}</p>
        <div className="mt-6 flex flex-wrap gap-3">
          <Link href="/trips" className="inline-flex h-11 items-center rounded-full bg-clay px-6 text-[14px] font-semibold text-sand-100">{t("setup.openTrip")}</Link>
          <button type="button" onClick={onboarding.complete} className="h-11 rounded-full border border-sand-400 px-6 text-[14px] font-semibold text-cocoa-700">{t("setup.finish")}</button>
        </div>
      </section>
    );
  }

  if (step === "completed" || step === "skipped") {
    return (
      <div className="space-y-4">
        <section className="rounded-[18px] border border-sand-300 bg-white px-6 py-5">
          <div className="flex flex-wrap items-center justify-between gap-4">
            <div>
              <h2 className="font-newsreader text-[22px] font-semibold text-cocoa-900">{step === "completed" ? t("status.completed") : t("status.skipped")}</h2>
              <p className="mt-1 text-[13.5px] text-cocoa-500">{t("status.description")}</p>
            </div>
            <button type="button" onClick={onboarding.restart} className="h-10 rounded-full border border-sand-400 px-5 text-[13px] font-semibold text-cocoa-700">{t("restart")}</button>
          </div>
        </section>
        <div className="rounded-[22px] border border-sand-300 bg-white p-6 sm:p-8"><ChooseTripStart /></div>
      </div>
    );
  }

  return (
    <section className="rounded-[22px] border border-sand-300 bg-white p-8 shadow-[0_14px_36px_rgba(34,26,20,0.06)]">
      <p className="text-[12px] font-semibold uppercase tracking-[0.09em] text-[#3E6B5A]">{t("welcome.eyebrow")}</p>
      <h2 className="mt-2 font-newsreader text-[32px] font-semibold text-cocoa-900">{t("welcome.title")}</h2>
      <p className="mt-3 max-w-2xl text-[15px] leading-[1.7] text-cocoa-500">{t("welcome.description")}</p>
      <div className="mt-7 flex flex-wrap gap-3">
        <button type="button" onClick={() => onboarding.markStepComplete("welcome", "preferences")} className="h-11 rounded-full bg-clay px-6 text-[14px] font-semibold text-sand-100">{t("welcome.preferencesAction")}</button>
        <button type="button" onClick={() => onboarding.markStepComplete("welcome", "choose_start")} className="h-11 rounded-full border border-sand-400 px-6 text-[14px] font-semibold text-cocoa-700">{t("welcome.chooseAction")}</button>
        <button type="button" onClick={onboarding.skip} className="h-11 rounded-full px-5 text-[14px] font-semibold text-cocoa-500">{t("skipForNow")}</button>
      </div>
    </section>
  );
}

function isOnboardingStep(value: string): value is OnboardingStep {
  return ["welcome", "preferences", "choose_start", "create_first_trip", "first_trip_setup", "completed", "skipped"].includes(value);
}
