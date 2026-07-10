"use client";

import { useTranslations } from "next-intl";
import type { TripDiscoverySuggestion } from "@/types/trip-discovery";
import type {
  DiscoverySuggestionVoteSummary,
  DiscoverySuggestionVoteValue
} from "@/types/trip-decisions";
import { DestinationSuggestionCard } from "./DestinationSuggestionCard";

export function DestinationSuggestionsGrid({
  suggestions,
  sessionId,
  voteSummaries,
  groupVotingEnabled = false,
  isVoting = false,
  onSelect,
  onSimilar,
  onReject,
  onVote
}: {
  suggestions: TripDiscoverySuggestion[];
  sessionId?: string;
  voteSummaries?: DiscoverySuggestionVoteSummary[];
  groupVotingEnabled?: boolean;
  isVoting?: boolean;
  onSelect: (suggestion: TripDiscoverySuggestion) => void;
  onSimilar: (suggestion: TripDiscoverySuggestion) => void;
  onReject: (suggestion: TripDiscoverySuggestion) => void;
  onVote?: (suggestion: TripDiscoverySuggestion, vote: DiscoverySuggestionVoteValue) => void;
}) {
  const t = useTranslations("tripDiscovery");
  const votesBySuggestion = new Map(
    (voteSummaries ?? []).map((summary) => [summary.suggestionId, summary])
  );

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
          sessionId={sessionId}
          voteSummary={votesBySuggestion.get(suggestion.id)}
          groupVotingEnabled={groupVotingEnabled}
          isVoting={isVoting}
          onSelect={() => onSelect(suggestion)}
          onSimilar={() => onSimilar(suggestion)}
          onReject={() => onReject(suggestion)}
          onVote={(vote) => onVote?.(suggestion, vote)}
        />
      ))}
    </div>
  );
}
