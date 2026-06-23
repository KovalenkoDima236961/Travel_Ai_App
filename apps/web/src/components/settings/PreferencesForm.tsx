"use client";

import { useEffect, type ReactNode } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import { Controller, useForm } from "react-hook-form";
import { z } from "zod";
import { CheckboxGroup } from "@/components/settings/CheckboxGroup";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import type {
  PatchUserPreferencesRequest,
  TravelPace,
  UserPreferences
} from "@/types/user";

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
    handleSubmit,
    register
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
    <Card>
      <div>
        <h2 className="text-lg font-semibold text-slate-950">Travel preferences</h2>
        <p className="mt-2 text-sm leading-6 text-slate-600">
          Saved preferences affect future itinerary generation. Trip Service loads them from
          User Service when you generate a plan.
        </p>
      </div>

      <form className="mt-6 space-y-7" onSubmit={handleSubmit(handleValidSubmit)}>
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

        <div className="grid gap-5 md:grid-cols-2">
          <Field label="Pace" error={errors.pace?.message}>
            <Select id="pace" aria-invalid={Boolean(errors.pace)} disabled={isSaving} {...register("pace")}>
              {paceOptions.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </Select>
          </Field>

          <Controller
            control={control}
            name="maxWalkingKmPerDay"
            render={({ field }) => (
              <Field label="Max walking per day" error={errors.maxWalkingKmPerDay?.message}>
                <Input
                  id="maxWalkingKmPerDay"
                  max={50}
                  min={0}
                  placeholder="8"
                  step="0.5"
                  type="number"
                  aria-invalid={Boolean(errors.maxWalkingKmPerDay)}
                  disabled={isSaving}
                  value={field.value ?? ""}
                  onBlur={field.onBlur}
                  onChange={(event) => {
                    const value = event.target.value;
                    field.onChange(value === "" ? null : Number(value));
                  }}
                />
              </Field>
            )}
          />
        </div>

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

        <SaveState successMessage={successMessage} errorMessage={errorMessage} />

        <div className="flex justify-end">
          <Button disabled={isSaving} type="submit">
            {isSaving ? "Saving..." : "Save preferences"}
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
