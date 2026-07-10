"use client";

import { useTranslations } from "next-intl";
import type { Workspace } from "@/entities/workspace/model";
import { TripDiscoveryQuickChips } from "./TripDiscoveryQuickChips";
import { SurpriseMeButton } from "./SurpriseMeButton";

export type DiscoveryDraft = {
  prompt: string;
  chips: string[];
  workspaceId: string;
  durationDays: number;
  budgetAmount: string;
  budgetCurrency: string;
  travelers: number;
  origin: string;
};

export function TripDiscoveryPromptBox({
  value,
  workspaces,
  isPending,
  isSurprisePending,
  onChange,
  onSubmit,
  onSurprise
}: {
  value: DiscoveryDraft;
  workspaces: Workspace[];
  isPending: boolean;
  isSurprisePending: boolean;
  onChange: (next: DiscoveryDraft) => void;
  onSubmit: () => void;
  onSurprise: () => void;
}) {
  const t = useTranslations("tripDiscovery");
  const toggleChip = (chip: string) =>
    onChange({
      ...value,
      chips: value.chips.includes(chip)
        ? value.chips.filter((item) => item !== chip)
        : [...value.chips, chip]
    });

  return (
    <section className="rounded-[22px] border border-sand-300 bg-white px-6 py-7 shadow-[0_10px_32px_rgba(34,26,20,0.06)] sm:px-8">
      <label htmlFor="trip-discovery-prompt" className="text-[14px] font-semibold text-cocoa-800">
        {t("promptLabel")}
      </label>
      <textarea
        id="trip-discovery-prompt"
        maxLength={1000}
        rows={4}
        value={value.prompt}
        onChange={(event) => onChange({ ...value, prompt: event.target.value })}
        placeholder={t("promptPlaceholder")}
        className="mt-3 w-full resize-none rounded-[16px] border border-sand-400 bg-[#FFFDFA] px-4 py-4 text-[15px] leading-6 text-cocoa-900 outline-none transition placeholder:text-cocoa-400 focus:border-clay focus:ring-[3px] focus:ring-clay-tint"
      />

      <div className="mt-5">
        <p className="mb-2.5 text-[12.5px] font-semibold uppercase tracking-[0.08em] text-cocoa-400">
          {t("quickPreferences")}
        </p>
        <TripDiscoveryQuickChips selected={value.chips} onToggle={toggleChip} />
      </div>

      <details className="mt-6 rounded-[14px] border border-sand-300 bg-sand-50 px-4 py-3">
        <summary className="cursor-pointer text-[13.5px] font-semibold text-cocoa-700">
          {t("tripDetails")}
        </summary>
        <div className="mt-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          <Field label={t("duration")}>
            <input
              type="number"
              min={1}
              max={30}
              value={value.durationDays}
              onChange={(event) =>
                onChange({ ...value, durationDays: Number(event.target.value) })
              }
              className={inputClass}
            />
          </Field>
          <Field label={t("budget")}>
            <div className="flex gap-2">
              <input
                type="number"
                min={0}
                value={value.budgetAmount}
                onChange={(event) => onChange({ ...value, budgetAmount: event.target.value })}
                placeholder="700"
                className={`${inputClass} min-w-0 flex-1`}
              />
              <select
                value={value.budgetCurrency}
                onChange={(event) => onChange({ ...value, budgetCurrency: event.target.value })}
                className={`${inputClass} w-[84px]`}
              >
                <option>EUR</option>
                <option>USD</option>
                <option>GBP</option>
                <option>CZK</option>
              </select>
            </div>
          </Field>
          <Field label={t("travelers")}>
            <input
              type="number"
              min={1}
              max={50}
              value={value.travelers}
              onChange={(event) => onChange({ ...value, travelers: Number(event.target.value) })}
              className={inputClass}
            />
          </Field>
          <Field label={t("origin")}>
            <input
              value={value.origin}
              onChange={(event) => onChange({ ...value, origin: event.target.value })}
              placeholder="Bratislava, Slovakia"
              className={inputClass}
            />
          </Field>
          <Field label={t("scope")}>
            <select
              value={value.workspaceId}
              onChange={(event) => onChange({ ...value, workspaceId: event.target.value })}
              className={inputClass}
            >
              <option value="">{t("personalTrip")}</option>
              {workspaces.map((workspace) => (
                <option key={workspace.id} value={workspace.id}>
                  {workspace.name}
                </option>
              ))}
            </select>
          </Field>
        </div>
      </details>

      <div className="mt-6 flex flex-col gap-3 sm:flex-row">
        <button
          type="button"
          disabled={isPending || (!value.prompt.trim() && value.chips.length === 0)}
          onClick={onSubmit}
          className="inline-flex h-12 items-center justify-center rounded-full bg-clay px-7 text-[14px] font-semibold text-white shadow-[0_8px_20px_rgba(192,91,59,0.22)] transition hover:bg-clay-dark disabled:cursor-not-allowed disabled:opacity-50"
        >
          {isPending ? t("findingPlaces") : t("getSuggestions")}
        </button>
        <SurpriseMeButton isPending={isSurprisePending} onClick={onSurprise} />
      </div>
    </section>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="block">
      <span className="mb-1.5 block text-[12px] font-semibold text-cocoa-500">{label}</span>
      {children}
    </label>
  );
}

const inputClass =
  "h-10 w-full rounded-lg border border-sand-400 bg-white px-3 text-[13.5px] text-cocoa-800 outline-none focus:border-clay";
