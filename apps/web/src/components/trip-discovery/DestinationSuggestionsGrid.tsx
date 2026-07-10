"use client";

import { useTranslations } from "next-intl";
import type { TripDiscoverySuggestion } from "@/types/trip-discovery";
import { DestinationSuggestionCard } from "./DestinationSuggestionCard";

export function DestinationSuggestionsGrid({
  suggestions,
  onSelect,
  onSimilar,
  onReject
}: {
  suggestions: TripDiscoverySuggestion[];
  onSelect: (suggestion: TripDiscoverySuggestion) => void;
  onSimilar: (suggestion: TripDiscoverySuggestion) => void;
  onReject: (suggestion: TripDiscoverySuggestion) => void;
}) {
  const t = useTranslations("tripDiscovery");
  if (suggestions.length === 0) {
    return (
      <div className="rounded-[20px] border border-dashed border-sand-400 bg-white p-8 text-center">
        <h3 className="font-newsreader text-[22px] text-cocoa-900">{t("emptyTitle")}</h3>
        <p className="mt-2 text-[14px] text-cocoa-500">{t("emptyBody")}</p>
      </div>
    );
  }
  return (
    <div className="grid gap-5 lg:grid-cols-2">
      {suggestions.map((suggestion) => (
        <DestinationSuggestionCard
          key={suggestion.id}
          suggestion={suggestion}
          onSelect={() => onSelect(suggestion)}
          onSimilar={() => onSimilar(suggestion)}
          onReject={() => onReject(suggestion)}
        />
      ))}
    </div>
  );
}
