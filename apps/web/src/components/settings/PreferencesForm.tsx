"use client";

import { useEffect } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import { Controller, useForm } from "react-hook-form";
import { z } from "zod";
import { cn } from "@/shared/lib/cn";
import { CheckboxGroup } from "@/components/settings/CheckboxGroup";
import {
  FIELD_LABEL_CLASS,
  INPUT_CLASS,
  PrimaryButton,
  SaveNotice,
  SectionHeading,
  SettingsCard
} from "@/components/settings/controls";
import type {
  PatchUserPreferencesRequest,
  TravelPace,
  UserPreferences
} from "@/entities/user/model";

const travelStyleOptions = [
  { value: "budget", label: "Budget" },
  { value: "luxury", label: "Luxury" },
  { value: "food", label: "Food" },
  { value: "hidden_gems", label: "Hidden gems" },
  { value: "culture", label: "Culture" },
  { value: "nature", label: "Nature" },
  { value: "nightlife", label: "Nightlife" },
  { value: "shopping", label: "Shopping" },
  { value: "family", label: "Family" },
  { value: "romantic", label: "Romantic" }
];

const foodPreferenceOptions = [
  { value: "local", label: "Local" },
  { value: "cheap", label: "Cheap" },
  { value: "fine_dining", label: "Fine dining" },
  { value: "street_food", label: "Street food" },
  { value: "cafes", label: "Cafes" },
  { value: "vegetarian_friendly", label: "Vegetarian friendly" }
];

const avoidOptions = [
  { value: "museums", label: "Museums" },
  { value: "nightclubs", label: "Nightclubs" },
  { value: "crowded_places", label: "Crowded places" },
  { value: "expensive_restaurants", label: "Expensive restaurants" },
  { value: "long_walks", label: "Long walks" }
];

const transportOptions = [
  { value: "walking", label: "Walking" },
  { value: "public_transport", label: "Public transport" },
  { value: "taxi", label: "Taxi" },
  { value: "bike", label: "Bike" },
  { value: "rental_car", label: "Rental car" }
];

const accommodationOptions = [
  { value: "budget_hotel", label: "Budget hotel" },
  { value: "apartment", label: "Apartment" },
  { value: "hostel", label: "Hostel" },
  { value: "boutique_hotel", label: "Boutique hotel" },
  { value: "luxury_hotel", label: "Luxury hotel" }
];

const dietaryOptions = [
  { value: "vegetarian", label: "Vegetarian" },
  { value: "vegan", label: "Vegan" },
  { value: "gluten_free", label: "Gluten free" },
  { value: "lactose_free", label: "Lactose free" },
  { value: "halal", label: "Halal" },
  { value: "kosher", label: "Kosher" }
];

const paceOptions: Array<{ value: TravelPace; label: string }> = [
  { value: "relaxed", label: "Relaxed" },
  { value: "balanced", label: "Balanced" },
  { value: "intensive", label: "Intensive" }
];

const SEG_BASE = "h-9 rounded-full px-[18px] text-[13.5px] transition disabled:cursor-not-allowed disabled:opacity-60";
const SEG_ACTIVE = "bg-cocoa-900 font-semibold text-sand-150";
const SEG_IDLE = "font-medium text-cocoa-500 hover:bg-sand-200";

const stringArraySchema = z.array(z.string());

const preferencesFormSchema = z.object({
  travelStyles: stringArraySchema,
  pace: z.enum(["relaxed", "balanced", "intensive"]),
  maxWalkingKmPerDay: z
    .number({ invalid_type_error: "Walking distance must be a number" })
    .min(0, "Walking distance must be at least 0 km")
    .max(50, "Walking distance must be 50 km or fewer")
    .nullable(),
  foodPreferences: stringArraySchema,
  avoid: stringArraySchema,
  preferredTransport: stringArraySchema,
  accommodationStyle: stringArraySchema,
  dietaryRestrictions: stringArraySchema
});

type PreferencesFormValues = z.infer<typeof preferencesFormSchema>;

type PreferencesFormProps = {
  preferences: UserPreferences;
  isSaving?: boolean;
  successMessage?: string | null;
  errorMessage?: string | null;
  onSubmit: (values: PatchUserPreferencesRequest) => void;
};

export function PreferencesForm({
  preferences,
  isSaving = false,
  successMessage,
  errorMessage,
  onSubmit
}: PreferencesFormProps) {
  const form = useForm<PreferencesFormValues>({
    resolver: zodResolver(preferencesFormSchema),
    defaultValues: toFormValues(preferences)
  });

  useEffect(() => {
    form.reset(toFormValues(preferences));
  }, [form, preferences]);

  const {
    control,
    formState: { errors },
    handleSubmit
  } = form;

  function handleValidSubmit(values: PreferencesFormValues) {
    onSubmit({
      travelStyles: values.travelStyles,
      pace: values.pace,
      maxWalkingKmPerDay: values.maxWalkingKmPerDay,
      foodPreferences: values.foodPreferences,
      avoid: values.avoid,
      preferredTransport: values.preferredTransport,
      accommodationStyle: values.accommodationStyle,
      dietaryRestrictions: values.dietaryRestrictions
    });
  }

  return (
    <SettingsCard>
      <SectionHeading
        title="Travel preferences"
        subtitle="Applied as defaults when the AI drafts a new itinerary."
      />

      <form className="mt-6 flex flex-col gap-[22px]" onSubmit={handleSubmit(handleValidSubmit)}>
        <Controller
          control={control}
          name="pace"
          render={({ field }) => (
            <div>
              <p className={FIELD_LABEL_CLASS}>Default pace</p>
              <div
                role="radiogroup"
                aria-label="Default pace"
                className="mt-2 inline-flex gap-1 rounded-full border border-sand-300 bg-sand-50 p-1"
              >
                {paceOptions.map((option) => {
                  const active = field.value === option.value;
                  return (
                    <button
                      key={option.value}
                      type="button"
                      role="radio"
                      aria-checked={active}
                      disabled={isSaving}
                      onClick={() => field.onChange(option.value)}
                      className={cn(SEG_BASE, active ? SEG_ACTIVE : SEG_IDLE)}
                    >
                      {option.label}
                    </button>
                  );
                })}
              </div>
              {errors.pace ? (
                <p className="mt-2 text-[13px] text-clay-deep">{errors.pace.message}</p>
              ) : null}
            </div>
          )}
        />

        <Controller
          control={control}
          name="maxWalkingKmPerDay"
          render={({ field }) => (
            <div>
              <label htmlFor="maxWalkingKmPerDay" className={FIELD_LABEL_CLASS}>
                Max walking per day
              </label>
              <p className="mt-2 text-[14px] text-cocoa-500">
                Used to flag long routes when the AI plans your days.
              </p>
              <div className="mt-2.5 flex items-center gap-2.5">
                <input
                  id="maxWalkingKmPerDay"
                  max={50}
                  min={0}
                  placeholder="8"
                  step="0.5"
                  type="number"
                  aria-invalid={Boolean(errors.maxWalkingKmPerDay)}
                  disabled={isSaving}
                  className={cn(INPUT_CLASS, "w-28")}
                  value={field.value ?? ""}
                  onBlur={field.onBlur}
                  onChange={(event) => {
                    const value = event.target.value;
                    field.onChange(value === "" ? null : Number(value));
                  }}
                />
                <span className="text-[14px] text-cocoa-500">km/day</span>
              </div>
              {errors.maxWalkingKmPerDay ? (
                <p className="mt-2 text-[13px] text-clay-deep">
                  {errors.maxWalkingKmPerDay.message}
                </p>
              ) : null}
            </div>
          )}
        />

        <Controller
          control={control}
          name="travelStyles"
          render={({ field }) => (
            <CheckboxGroup
              disabled={isSaving}
              error={errors.travelStyles?.message}
              label="Travel styles"
              options={travelStyleOptions}
              value={field.value}
              onChange={field.onChange}
            />
          )}
        />

        <Controller
          control={control}
          name="foodPreferences"
          render={({ field }) => (
            <CheckboxGroup
              disabled={isSaving}
              error={errors.foodPreferences?.message}
              label="Food preferences"
              options={foodPreferenceOptions}
              value={field.value}
              onChange={field.onChange}
            />
          )}
        />

        <Controller
          control={control}
          name="avoid"
          render={({ field }) => (
            <CheckboxGroup
              disabled={isSaving}
              error={errors.avoid?.message}
              label="Avoid"
              options={avoidOptions}
              value={field.value}
              onChange={field.onChange}
            />
          )}
        />

        <Controller
          control={control}
          name="preferredTransport"
          render={({ field }) => (
            <CheckboxGroup
              disabled={isSaving}
              error={errors.preferredTransport?.message}
              label="Preferred transport"
              options={transportOptions}
              value={field.value}
              onChange={field.onChange}
            />
          )}
        />

        <Controller
          control={control}
          name="accommodationStyle"
          render={({ field }) => (
            <CheckboxGroup
              disabled={isSaving}
              error={errors.accommodationStyle?.message}
              label="Accommodation style"
              options={accommodationOptions}
              value={field.value}
              onChange={field.onChange}
            />
          )}
        />

        <Controller
          control={control}
          name="dietaryRestrictions"
          render={({ field }) => (
            <CheckboxGroup
              disabled={isSaving}
              error={errors.dietaryRestrictions?.message}
              label="Dietary restrictions"
              options={dietaryOptions}
              value={field.value}
              onChange={field.onChange}
            />
          )}
        />

        <SaveNotice successMessage={successMessage} errorMessage={errorMessage} />

        <div className="flex justify-end">
          <PrimaryButton disabled={isSaving} type="submit">
            {isSaving ? "Saving…" : "Save preferences"}
          </PrimaryButton>
        </div>
      </form>
    </SettingsCard>
  );
}

function toFormValues(preferences: UserPreferences): PreferencesFormValues {
  return {
    travelStyles: preferences.travelStyles ?? [],
    pace: preferences.pace || "balanced",
    maxWalkingKmPerDay: preferences.maxWalkingKmPerDay,
    foodPreferences: preferences.foodPreferences ?? [],
    avoid: preferences.avoid ?? [],
    preferredTransport: preferences.preferredTransport ?? [],
    accommodationStyle: preferences.accommodationStyle ?? [],
    dietaryRestrictions: preferences.dietaryRestrictions ?? []
  };
}
