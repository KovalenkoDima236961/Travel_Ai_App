"use client";

import { useEffect, type ReactNode } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import type { UpdateUserProfileRequest, UserProfile } from "@/types/user";

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
  isSaving?: boolean;
  successMessage?: string | null;
  errorMessage?: string | null;
  onSubmit: (values: UpdateUserProfileRequest) => void;
};

export function ProfileForm({
  profile,
  isSaving = false,
  successMessage,
  errorMessage,
  onSubmit
}: ProfileFormProps) {
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
    register
  } = form;

  function handleValidSubmit(values: ProfileFormValues) {
    onSubmit({
      displayName: cleanOptionalText(values.displayName),
      homeCity: cleanOptionalText(values.homeCity),
      homeCountry: cleanOptionalText(values.homeCountry),
      preferredCurrency: values.preferredCurrency.trim().toUpperCase(),
      preferredLanguage: values.preferredLanguage.trim()
    });
  }

  return (
    <Card>
      <div>
        <h2 className="text-lg font-semibold text-slate-950">Profile</h2>
        <p className="mt-2 text-sm leading-6 text-slate-600">
          Save basic details the planner can use when shaping future trips.
        </p>
      </div>

      <form className="mt-6 space-y-6" onSubmit={handleSubmit(handleValidSubmit)}>
        <div className="grid gap-5 md:grid-cols-2">
          <Field label="Display name" error={errors.displayName?.message}>
            <Input
              id="displayName"
              maxLength={100}
              placeholder="Test Traveler"
              aria-invalid={Boolean(errors.displayName)}
              disabled={isSaving}
              {...register("displayName")}
            />
          </Field>

          <Field label="Home city" error={errors.homeCity?.message}>
            <Input
              id="homeCity"
              maxLength={100}
              placeholder="Bratislava"
              aria-invalid={Boolean(errors.homeCity)}
              disabled={isSaving}
              {...register("homeCity")}
            />
          </Field>

          <Field label="Home country" error={errors.homeCountry?.message}>
            <Input
              id="homeCountry"
              maxLength={100}
              placeholder="Slovakia"
              aria-invalid={Boolean(errors.homeCountry)}
              disabled={isSaving}
              {...register("homeCountry")}
            />
          </Field>

          <Field label="Preferred currency" error={errors.preferredCurrency?.message}>
            <Input
              id="preferredCurrency"
              maxLength={3}
              placeholder="EUR"
              aria-invalid={Boolean(errors.preferredCurrency)}
              disabled={isSaving}
              {...register("preferredCurrency", {
                setValueAs: (value) => String(value).trim().toUpperCase()
              })}
            />
          </Field>

          <Field label="Preferred language" error={errors.preferredLanguage?.message}>
            <Input
              id="preferredLanguage"
              maxLength={10}
              placeholder="en"
              aria-invalid={Boolean(errors.preferredLanguage)}
              disabled={isSaving}
              {...register("preferredLanguage", {
                setValueAs: (value) => String(value).trim()
              })}
            />
          </Field>
        </div>

        <SaveState successMessage={successMessage} errorMessage={errorMessage} />

        <div className="flex justify-end">
          <Button disabled={isSaving} type="submit">
            {isSaving ? "Saving..." : "Save profile"}
          </Button>
        </div>
      </form>
    </Card>
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
      <span className="text-sm font-medium text-slate-800">{label}</span>
      <span className="mt-2 block">{children}</span>
      {error ? <span className="mt-2 block text-sm text-red-700">{error}</span> : null}
    </label>
  );
}

function SaveState({
  successMessage,
  errorMessage
}: {
  successMessage?: string | null;
  errorMessage?: string | null;
}) {
  if (errorMessage) {
    return (
      <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800" role="alert">
        {errorMessage}
      </div>
    );
  }

  if (successMessage) {
    return (
      <div className="rounded-md border border-emerald-200 bg-emerald-50 p-3 text-sm text-emerald-800" role="status">
        {successMessage}
      </div>
    );
  }

  return null;
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
