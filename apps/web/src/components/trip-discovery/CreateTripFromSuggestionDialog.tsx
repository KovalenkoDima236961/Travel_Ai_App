"use client";

import { useEffect, useState } from "react";
import { useTranslations } from "next-intl";
import type { Workspace } from "@/entities/workspace/model";
import type {
  CreateTripFromSuggestionRequest,
  TripDiscoverySuggestion
} from "@/types/trip-discovery";

export function CreateTripFromSuggestionDialog({
  suggestion,
  sessionWorkspaceId,
  workspaces,
  isPending,
  error,
  onClose,
  onConfirm
}: {
  suggestion: TripDiscoverySuggestion | null;
  sessionWorkspaceId?: string | null;
  workspaces: Workspace[];
  isPending: boolean;
  error?: string | null;
  onClose: () => void;
  onConfirm: (input: CreateTripFromSuggestionRequest) => void;
}) {
  const t = useTranslations("tripDiscovery");
  const [title, setTitle] = useState("");
  const [startDate, setStartDate] = useState("");
  const [duration, setDuration] = useState(4);
  const [budget, setBudget] = useState("");
  const [currency, setCurrency] = useState("EUR");
  const [travelers, setTravelers] = useState(2);
  const [autoGenerate, setAutoGenerate] = useState(true);

  useEffect(() => {
    if (!suggestion) {
      return;
    }
    setTitle(suggestion.tripPreview.title);
    setDuration(suggestion.recommendedDurationDays);
    setBudget(String(suggestion.estimatedBudget.amount));
    setCurrency(suggestion.estimatedBudget.currency);
  }, [suggestion]);

  if (!suggestion) {
    return null;
  }

  const valid = duration >= 1 && duration <= 30 && travelers >= 1 && Number(budget) >= 0;
  const workspaceName = sessionWorkspaceId
    ? workspaces.find((workspace) => workspace.id === sessionWorkspaceId)?.name
    : null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-cocoa-900/55 p-4 backdrop-blur-sm"
      role="presentation"
      onMouseDown={(event) => {
        if (event.target === event.currentTarget && !isPending) onClose();
      }}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="create-discovery-trip-title"
        className="max-h-[92vh] w-full max-w-[620px] overflow-y-auto rounded-[24px] bg-white p-6 shadow-2xl sm:p-8"
      >
        <p className="text-[11px] font-bold uppercase tracking-[0.14em] text-clay">
          {t("confirmDestination")}
        </p>
        <h2
          id="create-discovery-trip-title"
          className="mt-2 font-newsreader text-[29px] font-semibold text-cocoa-900"
        >
          {suggestion.destination}
        </h2>
        <p className="mt-2 text-[13.5px] leading-6 text-cocoa-500">
          {t("confirmDescription")}
        </p>

        <div className="mt-6 grid gap-4 sm:grid-cols-2">
          <Field label={t("title")} wide>
            <input value={title} onChange={(event) => setTitle(event.target.value)} className={inputClass} />
          </Field>
          <Field label={t("startDate")}>
            <input type="date" value={startDate} onChange={(event) => setStartDate(event.target.value)} className={inputClass} />
          </Field>
          <Field label={t("duration")}>
            <input type="number" min={1} max={30} value={duration} onChange={(event) => setDuration(Number(event.target.value))} className={inputClass} />
          </Field>
          <Field label={t("budget")}>
            <div className="flex gap-2">
              <input type="number" min={0} value={budget} onChange={(event) => setBudget(event.target.value)} className={`${inputClass} min-w-0 flex-1`} />
              <input value={currency} maxLength={3} onChange={(event) => setCurrency(event.target.value.toUpperCase())} className={`${inputClass} w-20`} />
            </div>
          </Field>
          <Field label={t("travelers")}>
            <input type="number" min={1} max={50} value={travelers} onChange={(event) => setTravelers(Number(event.target.value))} className={inputClass} />
          </Field>
          <Field label={t("scope")}>
            <div className={`${inputClass} flex items-center bg-sand-50 text-cocoa-600`}>
              {workspaceName ?? t("personalTrip")}
            </div>
          </Field>
        </div>

        <label className="mt-5 flex cursor-pointer items-start gap-3 rounded-[14px] border border-sand-300 bg-sand-50 p-4">
          <input
            type="checkbox"
            checked={autoGenerate}
            onChange={(event) => setAutoGenerate(event.target.checked)}
            className="mt-0.5 h-4 w-4 accent-clay"
          />
          <span>
            <span className="block text-[13.5px] font-semibold text-cocoa-800">
              {t("autoGenerate")}
            </span>
            <span className="mt-0.5 block text-[12.5px] text-cocoa-500">
              {t("autoGenerateHint")}
            </span>
          </span>
        </label>

        <p className="mt-4 text-[12px] leading-5 text-cocoa-400">{t("budgetDisclaimer")}</p>
        {error ? <div role="alert" className="mt-4 rounded-xl bg-red-50 p-3 text-[13px] text-red-800">{error}</div> : null}

        <div className="mt-7 flex justify-end gap-3">
          <button type="button" disabled={isPending} onClick={onClose} className="h-11 rounded-full px-5 text-[13.5px] font-semibold text-cocoa-500 hover:bg-sand-100">
            {t("cancel")}
          </button>
          <button
            type="button"
            disabled={isPending || !valid}
            onClick={() =>
              onConfirm({
                title: title.trim() || suggestion.tripPreview.title,
                startDate: startDate || undefined,
                durationDays: duration,
                budget: { amount: Number(budget), currency },
                travelers,
                workspaceId: sessionWorkspaceId || undefined,
                tripType: suggestion.suggestionType === "route" ? "multi_destination" : "single_destination",
                route: suggestion.suggestionType === "route" ? suggestion.route ?? null : null,
                autoGenerateItinerary: autoGenerate
              })
            }
            className="h-11 rounded-full bg-clay px-6 text-[13.5px] font-semibold text-white hover:bg-clay-dark disabled:opacity-50"
          >
            {isPending ? t("creatingTrip") : t("createTrip")}
          </button>
        </div>
      </div>
    </div>
  );
}

function Field({
  label,
  wide,
  children
}: {
  label: string;
  wide?: boolean;
  children: React.ReactNode;
}) {
  return (
    <label className={wide ? "block sm:col-span-2" : "block"}>
      <span className="mb-1.5 block text-[12.5px] font-semibold text-cocoa-600">{label}</span>
      {children}
    </label>
  );
}

const inputClass =
  "h-11 w-full rounded-xl border border-sand-400 bg-[#FFFDFA] px-3.5 text-[13.5px] text-cocoa-800 outline-none focus:border-clay";
