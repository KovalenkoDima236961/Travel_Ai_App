"use client";

import type { ChangeEvent } from "react";
import type { ReactNode } from "react";
import { useTranslations } from "next-intl";
import { Input } from "@/shared/ui/input";
import { Select } from "@/shared/ui/select";
import { Textarea } from "@/shared/ui/textarea";
import type { TransportMode, TripStyle } from "@/entities/route/model";

export type AdvancedTripPreferencesValue = {
  budgetStrictness?: "loose" | "target" | "strict";
  outputLanguage?: "en" | "es" | "uk" | "fr";
  pace?: "relaxed" | "balanced" | "packed";
  maxWalkingKmPerDay?: number | null;
  preferredModes?: TransportMode[];
  avoidModes?: TransportMode[];
  carAvailable?: boolean;
  maxTransferHoursPerDay?: number | null;
  tripStyles?: TripStyle[];
  avoid?: string;
  mustHave?: string;
};

type Props = {
  value: AdvancedTripPreferencesValue;
  onChange: (value: AdvancedTripPreferencesValue) => void;
};

const transportModes: TransportMode[] = [
  "walk",
  "train",
  "bus",
  "public_transport",
  "car",
  "rental_car",
  "flight",
  "ferry",
  "boat",
  "bike",
  "hiking"
];

const tripStyles: TripStyle[] = [
  "city_break",
  "food",
  "culture",
  "nature",
  "hiking",
  "camping",
  "beach",
  "low_budget",
  "luxury"
];

export function AdvancedTripPreferencesForm({ value, onChange }: Props) {
  const t = useTranslations("planningConstraints");
  const transport = useTranslations("transportModes");
  const styles = useTranslations("tripStyles");

  function update(next: Partial<AdvancedTripPreferencesValue>) {
    onChange({ ...value, ...next });
  }

  function toggleList<T extends string>(key: "preferredModes" | "avoidModes" | "tripStyles", item: T) {
    const current = (value[key] ?? []) as T[];
    const next = current.includes(item)
      ? current.filter((entry) => entry !== item)
      : [...current, item];
    onChange({ ...value, [key]: next } as AdvancedTripPreferencesValue);
  }

  return (
    <div className="space-y-5">
      <div className="grid gap-4 md:grid-cols-3">
        <Field label={t("language")}>
          <Select
            value={value.outputLanguage ?? "en"}
            onChange={(event) =>
              update({ outputLanguage: event.target.value as AdvancedTripPreferencesValue["outputLanguage"] })
            }
          >
            <option value="en">English</option>
            <option value="es">Spanish</option>
            <option value="uk">Ukrainian</option>
            <option value="fr">French</option>
          </Select>
        </Field>
        <Field label={t("budgetStrictness")}>
          <Select
            value={value.budgetStrictness ?? "target"}
            onChange={(event) =>
              update({
                budgetStrictness: event.target.value as AdvancedTripPreferencesValue["budgetStrictness"]
              })
            }
          >
            <option value="loose">{t("strictness.loose")}</option>
            <option value="target">{t("strictness.target")}</option>
            <option value="strict">{t("strictness.strict")}</option>
          </Select>
        </Field>
        <Field label={t("walkingLimit")}>
          <Input
            min={0}
            step={0.5}
            type="number"
            value={value.maxWalkingKmPerDay ?? ""}
            onChange={(event: ChangeEvent<HTMLInputElement>) =>
              update({
                maxWalkingKmPerDay:
                  event.target.value === "" ? null : Number(event.target.value)
              })
            }
          />
        </Field>
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <Field label={t("maxTransferHours")}>
          <Input
            min={0}
            step={0.5}
            type="number"
            value={value.maxTransferHoursPerDay ?? ""}
            onChange={(event: ChangeEvent<HTMLInputElement>) =>
              update({
                maxTransferHoursPerDay:
                  event.target.value === "" ? null : Number(event.target.value)
              })
            }
          />
        </Field>
        <label className="flex min-h-11 items-center gap-3 rounded-md border border-slate-200 bg-slate-50 px-3 text-sm text-slate-700">
          <input
            checked={value.carAvailable ?? false}
            className="h-4 w-4 rounded border-slate-300"
            type="checkbox"
            onChange={(event) => update({ carAvailable: event.target.checked })}
          />
          {t("carAvailable")}
        </label>
      </div>

      <OptionGroup
        label={t("preferredTransport")}
        options={transportModes}
        selected={value.preferredModes ?? []}
        onToggle={(mode) => toggleList("preferredModes", mode)}
        formatOption={(mode) => transport(mode)}
      />
      <OptionGroup
        label={t("avoidTransport")}
        options={transportModes}
        selected={value.avoidModes ?? []}
        onToggle={(mode) => toggleList("avoidModes", mode)}
        formatOption={(mode) => transport(mode)}
      />
      <OptionGroup
        label={t("styles")}
        options={tripStyles}
        selected={value.tripStyles ?? []}
        onToggle={(style) => toggleList("tripStyles", style)}
        formatOption={(style) => styles(style)}
      />

      <div className="grid gap-4 md:grid-cols-2">
        <Field label={t("avoidList")}>
          <Textarea
            className="min-h-24"
            value={value.avoid ?? ""}
            onChange={(event) => update({ avoid: event.target.value })}
          />
        </Field>
        <Field label={t("mustHaveList")}>
          <Textarea
            className="min-h-24"
            value={value.mustHave ?? ""}
            onChange={(event) => update({ mustHave: event.target.value })}
          />
        </Field>
      </div>
    </div>
  );
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <label className="block">
      <span className="text-sm font-medium text-slate-800">{label}</span>
      <span className="mt-2 block">{children}</span>
    </label>
  );
}

function OptionGroup<T extends string>({
  label,
  options,
  selected,
  onToggle,
  formatOption
}: {
  label: string;
  options: T[];
  selected: T[];
  onToggle: (value: T) => void;
  formatOption: (value: T) => string;
}) {
  return (
    <fieldset>
      <legend className="text-sm font-medium text-slate-800">{label}</legend>
      <div className="mt-2 flex flex-wrap gap-2">
        {options.map((option) => (
          <label
            key={option}
            className="flex min-h-9 items-center gap-2 rounded-md border border-slate-200 bg-slate-50 px-3 text-sm text-slate-700"
          >
            <input
              checked={selected.includes(option)}
              className="h-4 w-4 rounded border-slate-300"
              type="checkbox"
              onChange={() => onToggle(option)}
            />
            {formatOption(option)}
          </label>
        ))}
      </div>
    </fieldset>
  );
}
