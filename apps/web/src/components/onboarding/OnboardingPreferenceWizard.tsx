"use client";

import { useState } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import { Controller, useForm } from "react-hook-form";
import { useTranslations } from "next-intl";
import { z } from "zod";
import { CheckboxGroup } from "@/components/settings/CheckboxGroup";
import { cn } from "@/shared/lib/cn";
import type { SupportedLanguage } from "@/lib/i18n/languages";
import type { TravelPace, UserPreferences, UserProfile } from "@/entities/user/model";
import { OnboardingProgress } from "./OnboardingProgress";
import { PreferenceStep } from "./PreferenceStep";

const SUPPORTED_CURRENCIES = ["EUR", "USD", "GBP", "CZK"] as const;
const SUPPORTED_LANGUAGES = ["en", "es", "uk", "fr"] as const;
const STEPS = ["basics", "style", "comfort", "preferences"] as const;

type WizardStep = (typeof STEPS)[number];
type Translator = (key: string) => string;

export function createOnboardingPreferenceSchema(t: Translator) {
  return z.object({
    homeCity: z.string().trim().max(100, t("validation.shortText")),
    homeCountry: z.string().trim().max(100, t("validation.shortText")),
    preferredCurrency: z.enum(SUPPORTED_CURRENCIES, {
      errorMap: () => ({ message: t("validation.currency") })
    }),
    preferredLanguage: z.enum(SUPPORTED_LANGUAGES, {
      errorMap: () => ({ message: t("validation.language") })
    }),
    travelStyles: z.array(z.string()),
    budgetComfort: z.enum(["budget", "balanced", "comfort"]),
    pace: z.enum(["relaxed", "balanced", "intensive"]),
    maxWalkingKmPerDay: z
      .number({ invalid_type_error: t("validation.walking") })
      .positive(t("validation.walking"))
      .max(50, t("validation.walking"))
      .nullable(),
    foodPreferences: z.array(z.string()),
    dietaryRestrictions: z.array(z.string()),
    preferredTransport: z.array(z.string()),
    accommodationStyle: z.array(z.string()).max(1)
  });
}

export type OnboardingPreferenceValues = z.infer<
  ReturnType<typeof createOnboardingPreferenceSchema>
>;

type OnboardingPreferenceWizardProps = {
  profile: UserProfile;
  preferences: UserPreferences;
  isSaving?: boolean;
  errorMessage?: string | null;
  onBack: () => void;
  onSkip: () => void;
  onSave: (values: OnboardingPreferenceValues) => Promise<void> | void;
};

const FIELD_LABEL = "block text-[13.5px] font-semibold text-cocoa-700";
const INPUT =
  "mt-2 h-11 w-full rounded-xl border border-sand-400 bg-[#FFFDFA] px-3.5 text-[14px] text-cocoa-900 outline-none transition focus:border-clay focus:ring-[3px] focus:ring-clay-tint";

export function OnboardingPreferenceWizard({
  profile,
  preferences,
  isSaving = false,
  errorMessage,
  onBack,
  onSkip,
  onSave
}: OnboardingPreferenceWizardProps) {
  const t = useTranslations("onboarding.preferences");
  const [stepIndex, setStepIndex] = useState(0);
  const schema = createOnboardingPreferenceSchema(t);
  const form = useForm<OnboardingPreferenceValues>({
    resolver: zodResolver(schema),
    defaultValues: toDefaultValues(profile, preferences)
  });
  const step = STEPS[stepIndex];
  const isLast = stepIndex === STEPS.length - 1;
  const options = optionGroups(t);

  async function continueWizard() {
    const valid = await form.trigger(fieldsForStep(step));
    if (!valid) {
      return;
    }
    if (isLast) {
      await form.handleSubmit(onSave)();
      return;
    }
    setStepIndex((current) => current + 1);
  }

  function goBack() {
    if (stepIndex === 0) {
      onBack();
      return;
    }
    setStepIndex((current) => current - 1);
  }

  return (
    <section className="rounded-[22px] border border-sand-300 bg-white p-6 shadow-[0_14px_36px_rgba(34,26,20,0.06)] sm:p-8">
      <OnboardingProgress
        current={stepIndex + 1}
        total={STEPS.length}
        label={t("progress", { current: stepIndex + 1, total: STEPS.length })}
      />

      <form className="mt-7" onSubmit={(event) => event.preventDefault()}>
        {step === "basics" ? (
          <PreferenceStep title={t("basics.title")} description={t("basics.description")}>
            <div className="grid gap-4 sm:grid-cols-2">
              <Field label={t("fields.homeCity")} error={form.formState.errors.homeCity?.message}>
                <input className={INPUT} autoComplete="address-level2" {...form.register("homeCity")} />
              </Field>
              <Field label={t("fields.homeCountry")} error={form.formState.errors.homeCountry?.message}>
                <input className={INPUT} autoComplete="country-name" {...form.register("homeCountry")} />
              </Field>
              <Field label={t("fields.currency")} error={form.formState.errors.preferredCurrency?.message}>
                <select className={INPUT} {...form.register("preferredCurrency")}>
                  {SUPPORTED_CURRENCIES.map((currency) => <option key={currency}>{currency}</option>)}
                </select>
              </Field>
              <Field label={t("fields.language")} error={form.formState.errors.preferredLanguage?.message}>
                <select className={INPUT} {...form.register("preferredLanguage")}>
                  {SUPPORTED_LANGUAGES.map((language) => (
                    <option key={language} value={language}>{t(`languages.${language}`)}</option>
                  ))}
                </select>
              </Field>
            </div>
          </PreferenceStep>
        ) : null}

        {step === "style" ? (
          <PreferenceStep title={t("style.title")} description={t("style.description")}>
            <Controller
              control={form.control}
              name="travelStyles"
              render={({ field }) => (
                <CheckboxGroup label={t("fields.travelStyles")} options={options.travelStyles} value={field.value} onChange={field.onChange} />
              )}
            />
          </PreferenceStep>
        ) : null}

        {step === "comfort" ? (
          <PreferenceStep title={t("comfort.title")} description={t("comfort.description")}>
            <SegmentedField
              label={t("fields.pace")}
              value={form.watch("pace")}
              options={options.pace}
              onChange={(value) => form.setValue("pace", value as TravelPace, { shouldDirty: true })}
            />
            <Field label={t("fields.walking")} error={form.formState.errors.maxWalkingKmPerDay?.message}>
              <div className="flex items-center gap-2">
                <input
                  className={cn(INPUT, "w-32")}
                  type="number"
                  min={0.5}
                  max={50}
                  step={0.5}
                  value={form.watch("maxWalkingKmPerDay") ?? ""}
                  onChange={(event) => form.setValue("maxWalkingKmPerDay", event.target.value ? Number(event.target.value) : null)}
                />
                <span className="pt-2 text-[13px] text-cocoa-500">{t("fields.kmPerDay")}</span>
              </div>
            </Field>
            <SegmentedField
              label={t("fields.budgetComfort")}
              value={form.watch("budgetComfort")}
              options={options.budgetComfort}
              onChange={(value) => form.setValue("budgetComfort", value as OnboardingPreferenceValues["budgetComfort"], { shouldDirty: true })}
            />
          </PreferenceStep>
        ) : null}

        {step === "preferences" ? (
          <PreferenceStep title={t("details.title")} description={t("details.description")}>
            <Controller control={form.control} name="foodPreferences" render={({ field }) => (
              <CheckboxGroup label={t("fields.food")} options={options.food} value={field.value} onChange={field.onChange} />
            )} />
            <Controller control={form.control} name="dietaryRestrictions" render={({ field }) => (
              <CheckboxGroup label={t("fields.dietary")} options={options.dietary} value={field.value} onChange={field.onChange} />
            )} />
            <Controller control={form.control} name="preferredTransport" render={({ field }) => (
              <CheckboxGroup label={t("fields.transport")} options={options.transport} value={field.value} onChange={field.onChange} />
            )} />
            <SegmentedField
              label={t("fields.accommodation")}
              value={form.watch("accommodationStyle")[0] ?? ""}
              options={options.accommodation}
              onChange={(value) => form.setValue("accommodationStyle", value ? [value] : [], { shouldDirty: true })}
            />
          </PreferenceStep>
        ) : null}

        {errorMessage ? <p className="mt-5 text-[13px] text-clay-deep" role="alert">{errorMessage}</p> : null}

        <div className="mt-8 flex flex-wrap items-center justify-between gap-3 border-t border-sand-200 pt-5">
          <button type="button" onClick={goBack} disabled={isSaving} className="h-10 rounded-full border border-sand-400 px-5 text-[13.5px] font-semibold text-cocoa-700 transition hover:border-sand-600">
            {t("back")}
          </button>
          <div className="flex items-center gap-2">
            <button type="button" onClick={onSkip} disabled={isSaving} className="h-10 rounded-full px-4 text-[13.5px] font-semibold text-cocoa-500 transition hover:bg-sand-100">
              {t("skip")}
            </button>
            <button type="button" onClick={() => void continueWizard()} disabled={isSaving} className="h-10 rounded-full bg-clay px-6 text-[13.5px] font-semibold text-sand-100 transition hover:bg-clay-dark disabled:opacity-60">
              {isSaving ? t("saving") : isLast ? t("finish") : t("continue")}
            </button>
          </div>
        </div>
      </form>
    </section>
  );
}

function Field({ label, error, children }: { label: string; error?: string; children: React.ReactNode }) {
  return (
    <label className="block">
      <span className={FIELD_LABEL}>{label}</span>
      {children}
      {error ? <span className="mt-1.5 block text-[12.5px] text-clay-deep">{error}</span> : null}
    </label>
  );
}

function SegmentedField({ label, value, options, onChange }: { label: string; value: string; options: Array<{ value: string; label: string }>; onChange: (value: string) => void }) {
  return (
    <div>
      <p className={FIELD_LABEL}>{label}</p>
      <div className="mt-2 flex flex-wrap gap-2" role="radiogroup" aria-label={label}>
        {options.map((option) => (
          <button
            key={option.value}
            type="button"
            role="radio"
            aria-checked={value === option.value}
            onClick={() => onChange(option.value)}
            className={value === option.value ? "h-9 rounded-full bg-cocoa-900 px-4 text-[13px] font-semibold text-sand-100" : "h-9 rounded-full border border-sand-300 px-4 text-[13px] font-medium text-cocoa-600 transition hover:bg-sand-100"}
          >
            {option.label}
          </button>
        ))}
      </div>
    </div>
  );
}

function fieldsForStep(step: WizardStep): Array<keyof OnboardingPreferenceValues> {
  if (step === "basics") return ["homeCity", "homeCountry", "preferredCurrency", "preferredLanguage"];
  if (step === "style") return ["travelStyles"];
  if (step === "comfort") return ["pace", "maxWalkingKmPerDay", "budgetComfort"];
  return ["foodPreferences", "dietaryRestrictions", "preferredTransport", "accommodationStyle"];
}

function toDefaultValues(profile: UserProfile, preferences: UserPreferences): OnboardingPreferenceValues {
  return {
    homeCity: profile.homeCity ?? "",
    homeCountry: profile.homeCountry ?? "",
    preferredCurrency: SUPPORTED_CURRENCIES.includes(profile.preferredCurrency as (typeof SUPPORTED_CURRENCIES)[number]) ? profile.preferredCurrency as OnboardingPreferenceValues["preferredCurrency"] : "EUR",
    preferredLanguage: SUPPORTED_LANGUAGES.includes(profile.preferredLanguage as SupportedLanguage) ? profile.preferredLanguage : "en",
    travelStyles: preferences.travelStyles ?? [],
    budgetComfort: preferences.travelStyles?.includes("budget") ? "budget" : preferences.travelStyles?.includes("luxury") ? "comfort" : "balanced",
    pace: preferences.pace ?? "balanced",
    maxWalkingKmPerDay: preferences.maxWalkingKmPerDay ?? 8,
    foodPreferences: preferences.foodPreferences ?? [],
    dietaryRestrictions: preferences.dietaryRestrictions ?? [],
    preferredTransport: preferences.preferredTransport ?? [],
    accommodationStyle: preferences.accommodationStyle?.slice(0, 1) ?? []
  };
}

function optionGroups(t: Translator) {
  const options = (group: string, values: string[]) => values.map((value) => ({ value, label: t(`options.${group}.${value}`) }));
  return {
    travelStyles: options("styles", ["city_break", "nature", "food", "culture", "adventure", "budget", "comfort", "hidden_gems"]),
    pace: options("pace", ["relaxed", "balanced", "intensive"]),
    budgetComfort: options("budget", ["budget", "balanced", "comfort"]),
    food: options("food", ["local", "street_food", "cafes", "fine_dining"]),
    dietary: options("dietary", ["vegetarian", "vegan", "gluten_free", "lactose_free", "halal", "kosher"]),
    transport: options("transport", ["walking", "public_transport", "train", "bus", "bike", "rental_car"]),
    accommodation: options("accommodation", ["budget_hotel", "apartment", "hostel", "boutique_hotel", "luxury_hotel"])
  };
}

export function wizardValuesToTravelStyles(values: OnboardingPreferenceValues) {
  const withoutBudgetSignals = values.travelStyles.filter((style) => style !== "budget" && style !== "luxury");
  if (values.budgetComfort === "budget") return Array.from(new Set([...withoutBudgetSignals, "budget"]));
  if (values.budgetComfort === "comfort") return Array.from(new Set([...withoutBudgetSignals, "luxury"]));
  return withoutBudgetSignals;
}
