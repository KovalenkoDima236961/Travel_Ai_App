"use client";

import { useEffect, type ReactNode } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { useTranslations } from "next-intl";
import { useAppLanguage } from "@/components/i18n/I18nProvider";
import {
  FIELD_LABEL_CLASS,
  GhostButton,
  INPUT_CLASS,
  PrimaryButton,
  SaveNotice,
  SectionHeading,
  SELECT_CLASS,
  SettingsCard
} from "@/components/settings/controls";
import type { UpdateUserProfileRequest, UserProfile } from "@/entities/user/model";

const CURRENCY_OPTIONS = ["EUR", "USD", "GBP", "CZK"];

const profileFormSchema = z.object({
  displayName: z.string().trim().max(100, "Display name must be 100 characters or fewer"),
  homeCity: z.string().trim().max(100, "Home city must be 100 characters or fewer"),
  homeCountry: z.string().trim().max(100, "Home country must be 100 characters or fewer"),
  preferredCurrency: z
    .string()
    .trim()
    .regex(/^[A-Z]{3}$/, "Use a 3-letter uppercase currency code"),
  preferredLanguage: z
    .string()
    .trim()
    .min(2, "Language must be at least 2 characters")
    .max(10, "Language must be 10 characters or fewer")
});

type ProfileFormValues = z.infer<typeof profileFormSchema>;

type ProfileFormProps = {
  profile: UserProfile;
  email?: string | null;
  isSaving?: boolean;
  successMessage?: string | null;
  errorMessage?: string | null;
  onSubmit: (values: UpdateUserProfileRequest) => void;
};

export function ProfileForm({
  profile,
  email,
  isSaving = false,
  successMessage,
  errorMessage,
  onSubmit
}: ProfileFormProps) {
  const translate = useTranslations("settings");
  const { language } = useAppLanguage();
  const form = useForm<ProfileFormValues>({
    resolver: zodResolver(profileFormSchema),
    defaultValues: toFormValues(profile)
  });

  useEffect(() => {
    form.reset(toFormValues(profile));
  }, [form, profile]);

  const {
    formState: { errors },
    handleSubmit,
    register,
    watch
  } = form;

  const initials = initialsFrom(watch("displayName"), email);
  const currentCurrency = watch("preferredCurrency");
  const currencyOptions = CURRENCY_OPTIONS.includes(currentCurrency)
    ? CURRENCY_OPTIONS
    : [currentCurrency, ...CURRENCY_OPTIONS].filter(Boolean);

  function handleValidSubmit(values: ProfileFormValues) {
    onSubmit({
      displayName: cleanOptionalText(values.displayName),
      homeCity: cleanOptionalText(values.homeCity),
      homeCountry: cleanOptionalText(values.homeCountry),
      preferredCurrency: values.preferredCurrency.trim().toUpperCase(),
      preferredLanguage: language
    });
  }

  return (
    <SettingsCard>
      <SectionHeading title={translate("profile")} />

      <div className="mt-6 flex items-center gap-[18px]">
        <span className="flex h-16 w-16 items-center justify-center rounded-full bg-[#3E6B5A] font-newsreader text-2xl font-semibold text-[#EFF5F1]">
          {initials}
        </span>
        <GhostButton type="button" disabled title={translate("photoComingSoon")}>
          {translate("changePhoto")}
        </GhostButton>
      </div>

      <form className="mt-6" onSubmit={handleSubmit(handleValidSubmit)}>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <Field label={translate("displayName")} error={errors.displayName?.message}>
            <input
              id="displayName"
              maxLength={100}
              placeholder="Test Traveler"
              aria-invalid={Boolean(errors.displayName)}
              disabled={isSaving}
              className={INPUT_CLASS}
              {...register("displayName")}
            />
          </Field>

          <Field label={translate("email")}>
            <input
              id="email"
              type="email"
              value={email ?? ""}
              readOnly
              disabled
              className={INPUT_CLASS}
            />
          </Field>

          <Field label={translate("homeCity")} error={errors.homeCity?.message}>
            <input
              id="homeCity"
              maxLength={100}
              placeholder="Bratislava"
              aria-invalid={Boolean(errors.homeCity)}
              disabled={isSaving}
              className={INPUT_CLASS}
              {...register("homeCity")}
            />
          </Field>

          <Field label={translate("homeCountry")} error={errors.homeCountry?.message}>
            <input
              id="homeCountry"
              maxLength={100}
              placeholder="Slovakia"
              aria-invalid={Boolean(errors.homeCountry)}
              disabled={isSaving}
              className={INPUT_CLASS}
              {...register("homeCountry")}
            />
          </Field>

          <Field label={translate("homeCurrency")} error={errors.preferredCurrency?.message}>
            <select
              id="preferredCurrency"
              aria-invalid={Boolean(errors.preferredCurrency)}
              disabled={isSaving}
              className={SELECT_CLASS}
              {...register("preferredCurrency", {
                setValueAs: (value) => String(value).trim().toUpperCase()
              })}
            >
              {currencyOptions.map((currency) => (
                <option key={currency} value={currency}>
                  {currency}
                </option>
              ))}
            </select>
          </Field>

        </div>

        <div className="mt-5">
          <SaveNotice successMessage={successMessage} errorMessage={errorMessage} />
        </div>

        <div className="mt-5 flex justify-end">
          <PrimaryButton disabled={isSaving} type="submit">
            {isSaving ? translate("saving") : translate("saveProfile")}
          </PrimaryButton>
        </div>
      </form>
    </SettingsCard>
  );
}

type FieldProps = {
  label: string;
  error?: string;
  children: ReactNode;
};

function Field({ label, error, children }: FieldProps) {
  return (
    <label className="block">
      <span className={FIELD_LABEL_CLASS}>{label}</span>
      <span className="mt-2 block">{children}</span>
      {error ? <span className="mt-2 block text-[13px] text-clay-deep">{error}</span> : null}
    </label>
  );
}

function toFormValues(profile: UserProfile): ProfileFormValues {
  return {
    displayName: profile.displayName ?? "",
    homeCity: profile.homeCity ?? "",
    homeCountry: profile.homeCountry ?? "",
    preferredCurrency: profile.preferredCurrency || "EUR",
    preferredLanguage: profile.preferredLanguage || "en"
  };
}

function cleanOptionalText(value: string) {
  const trimmed = value.trim();
  return trimmed.length > 0 ? trimmed : null;
}

function initialsFrom(displayName: string | undefined, email: string | null | undefined) {
  const source = (displayName ?? "").trim();
  if (source) {
    const parts = source.split(/\s+/).filter(Boolean);
    const letters = parts
      .slice(0, 2)
      .map((part) => part[0])
      .join("");
    if (letters) {
      return letters.toUpperCase();
    }
  }
  const local = (email ?? "").split("@")[0] ?? "";
  const fallback = local.replace(/[^a-zA-Z]/g, "").slice(0, 2);
  return (fallback || "?").toUpperCase();
}
