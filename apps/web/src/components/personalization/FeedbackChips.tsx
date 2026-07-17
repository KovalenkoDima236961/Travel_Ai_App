"use client";

import { useState } from "react";
import { useSubmitPersonalizationFeedback } from "@/hooks/usePersonalization";
import type { FeedbackType, PersonalizationFeedbackInput } from "@/types/personalization";

type Chip = { type: FeedbackType; label: string };
const defaultChips: Chip[] = [
  { type: "too_expensive", label: "Too expensive" },
  { type: "not_my_vibe", label: "Not my vibe" },
  { type: "more_nature", label: "More nature" },
  { type: "prefer_trains", label: "Prefer trains" }
];

export function FeedbackChips({ input, chips = defaultChips }: { input: Omit<PersonalizationFeedbackInput, "feedbackType">; chips?: Chip[] }) {
  const mutation = useSubmitPersonalizationFeedback();
  const [selected, setSelected] = useState<FeedbackType | null>(null);
  return (
    <div className="flex flex-wrap gap-1.5" aria-label="Suggestion feedback">
      {chips.map((chip) => (
        <button
          key={chip.type}
          type="button"
          disabled={mutation.isPending}
          aria-pressed={selected === chip.type}
          onClick={() => mutation.mutate({ ...input, feedbackType: chip.type }, { onSuccess: () => setSelected(chip.type) })}
          className={`rounded-full border px-2.5 py-1 text-[11px] font-semibold transition disabled:opacity-50 ${selected === chip.type ? "border-clay bg-[#FBF0EB] text-clay-deep" : "border-sand-300 bg-white text-cocoa-500 hover:border-sand-500"}`}
        >
          {selected === chip.type ? "Saved" : chip.label}
        </button>
      ))}
    </div>
  );
}
