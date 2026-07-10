"use client";

import { useTranslations } from "next-intl";
import { transportModeLabel } from "@/components/routes/route-options";
import type { TripDiscoverySuggestion } from "@/types/trip-discovery";

export function DestinationSuggestionCard({
  suggestion,
  onSelect,
  onSimilar,
  onReject
}: {
  suggestion: TripDiscoverySuggestion;
  onSelect: () => void;
  onSimilar: () => void;
  onReject: () => void;
}) {
  const t = useTranslations("tripDiscovery");
  return (
    <article className="flex h-full flex-col overflow-hidden rounded-[22px] border border-sand-300 bg-white shadow-[0_10px_28px_rgba(34,26,20,0.07)]">
      <div className="relative bg-gradient-to-br from-[#E9D6C8] via-[#F3E8DC] to-[#D8DFCF] px-6 py-6">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h3 className="font-newsreader text-[27px] font-semibold text-cocoa-900">
              {suggestion.city}
            </h3>
            <p className="mt-1 text-[13px] font-medium text-cocoa-500">{suggestion.country}</p>
          </div>
          <div
            className="rounded-full border border-white/70 bg-white/80 px-3 py-1.5 text-[12px] font-bold text-cocoa-800 shadow-sm"
            title={t("matchTooltip")}
          >
            {suggestion.matchScore}% {t("match")}
          </div>
        </div>
        <div className="mt-5 flex flex-wrap gap-1.5">
          {suggestion.tags.slice(0, 4).map((tag) => (
            <span key={tag} className="rounded-full bg-white/70 px-2.5 py-1 text-[11px] font-semibold text-cocoa-600">
              {tag.replaceAll("_", " ")}
            </span>
          ))}
        </div>
      </div>

      <div className="flex flex-1 flex-col px-6 py-6">
        <div className="grid grid-cols-2 gap-3 rounded-[14px] bg-sand-50 p-3">
          <Metric
            label={t("estimatedBudget")}
            value={`${suggestion.estimatedBudget.amount.toLocaleString()} ${suggestion.estimatedBudget.currency}`}
          />
          <Metric
            label={t("recommendedDuration")}
            value={t("daysCount", { count: suggestion.recommendedDurationDays })}
          />
        </div>

        <div className="mt-5">
          <h4 className="text-[12px] font-bold uppercase tracking-[0.08em] text-cocoa-400">
            {t("whyItFits")}
          </h4>
          <p className="mt-2 text-[14px] leading-6 text-cocoa-700">{suggestion.whyItFits}</p>
        </div>

        <div className="mt-4">
          <h4 className="text-[12px] font-bold uppercase tracking-[0.08em] text-cocoa-400">
            {t("tripPreview")}
          </h4>
          <p className="mt-2 text-[13.5px] leading-6 text-cocoa-600">
            {suggestion.tripPreview.summary}
          </p>
          <ul className="mt-2 space-y-1.5 text-[13px] text-cocoa-500">
            {suggestion.tripPreview.sampleDay.map((item) => (
              <li key={item} className="flex gap-2">
                <span className="text-clay" aria-hidden="true">•</span>
                {item}
              </li>
            ))}
          </ul>
        </div>

        {suggestion.suggestionType === "route" && suggestion.route ? (
          <div className="mt-4 rounded-[14px] border border-sand-300 bg-[#FFFDFA] p-3">
            <h4 className="text-[12px] font-bold uppercase tracking-[0.08em] text-cocoa-400">
              Route
            </h4>
            <p className="mt-2 text-[13.5px] font-semibold text-cocoa-800">
              {suggestion.route.stops.map((stop) => stop.city || stop.destination).join(" to ")}
            </p>
            {suggestion.route.legs?.[0] ? (
              <p className="mt-1 text-[12.5px] text-cocoa-500">
                {transportModeLabel(suggestion.route.legs[0].mode)}
                {suggestion.route.legs.length > 1 ? ` · ${suggestion.route.legs.length} transfers` : ""}
              </p>
            ) : null}
          </div>
        ) : null}

        {suggestion.possibleDownsides.length > 0 ? (
          <details className="mt-4 text-[13px] text-cocoa-500">
            <summary className="cursor-pointer font-semibold text-cocoa-600">
              {t("possibleDownsides")}
            </summary>
            <ul className="mt-2 list-disc space-y-1 pl-5">
              {suggestion.possibleDownsides.map((item) => <li key={item}>{item}</li>)}
            </ul>
          </details>
        ) : null}

        <div className="mt-auto flex flex-wrap gap-2 pt-6">
          <button
            type="button"
            onClick={onSelect}
            className="flex-1 rounded-full bg-cocoa-900 px-4 py-2.5 text-[13px] font-semibold text-sand-100 hover:bg-cocoa-700"
          >
            {suggestion.suggestionType === "route" ? "Use this route" : t("useDestination")}
          </button>
          <button type="button" onClick={onSimilar} className={secondaryButton}>
            {t("showSimilar")}
          </button>
          <button type="button" onClick={onReject} className={secondaryButton}>
            {t("notThisVibe")}
          </button>
        </div>
      </div>
    </article>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-[10.5px] font-bold uppercase tracking-[0.06em] text-cocoa-400">{label}</p>
      <p className="mt-1 text-[14px] font-semibold text-cocoa-800">{value}</p>
    </div>
  );
}

const secondaryButton =
  "rounded-full border border-sand-400 px-3.5 py-2.5 text-[12.5px] font-semibold text-cocoa-600 transition hover:border-clay/50 hover:text-clay-deep";
