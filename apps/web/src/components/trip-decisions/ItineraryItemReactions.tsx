"use client";

import { useSetItineraryReaction } from "@/hooks/useSetItineraryReaction";
import type { ItineraryItemReactionSummary, ItineraryReaction } from "@/types/trip-decisions";

const REACTIONS: Array<{ value: ItineraryReaction; label: string }> = [
  { value: "must_have", label: "Must" },
  { value: "want_to_do", label: "Want" },
  { value: "neutral", label: "Neutral" },
  { value: "skip", label: "Skip" }
];

type ItineraryItemReactionsProps = {
  tripId: string;
  dayNumber: number;
  itemIndex: number;
  itemId?: string;
  summary?: ItineraryItemReactionSummary;
  disabled?: boolean;
};

export function ItineraryItemReactions({
  tripId,
  dayNumber,
  itemIndex,
  itemId,
  summary,
  disabled = false
}: ItineraryItemReactionsProps) {
  const mutation = useSetItineraryReaction(tripId);

  return (
    <div className="flex flex-wrap gap-1.5">
      {REACTIONS.map((reaction) => {
        const selected = summary?.currentUserReaction === reaction.value;
        const count = summary?.counts?.[reaction.value] ?? 0;
        return (
          <button
            key={reaction.value}
            className={`h-7 rounded-full border px-2 text-[11px] font-semibold transition ${
              selected
                ? "border-clay bg-[#FBF0EB] text-clay-deep"
                : "border-sand-300 bg-white text-cocoa-400 hover:border-sand-500 hover:text-cocoa-800"
            } disabled:cursor-not-allowed disabled:opacity-50`}
            disabled={disabled || mutation.isPending}
            onClick={() =>
              mutation.mutate({
                dayNumber,
                itemIndex,
                itemId,
                reaction: reaction.value
              })
            }
            title={`${reaction.label}: ${count}`}
            type="button"
          >
            {reaction.label}
            {count > 0 ? <span className="ml-1 text-[10px]">{count}</span> : null}
          </button>
        );
      })}
    </div>
  );
}
