"use client";

import type { RouteAlternativeScores as RouteAlternativeScoresValue } from "@/types/route-alternatives";

const SCORE_LABELS: Array<{ key: keyof RouteAlternativeScoresValue; label: string }> = [
  { key: "overallFit", label: "Overall" },
  { key: "budgetFit", label: "Budget" },
  { key: "relaxation", label: "Relaxation" },
  { key: "nature", label: "Nature" },
  { key: "culture", label: "Culture" },
  { key: "transportSimplicity", label: "Transport" },
  { key: "policyCompliance", label: "Policy" }
];

type RouteAlternativeScoresProps = {
  scores: RouteAlternativeScoresValue;
  compact?: boolean;
};

export function RouteAlternativeScores({ scores, compact = false }: RouteAlternativeScoresProps) {
  const items = compact ? SCORE_LABELS.slice(0, 4) : SCORE_LABELS;
  return (
    <div className={compact ? "space-y-2" : "grid gap-3 sm:grid-cols-2"}>
      {items.map((item) => {
        const value = clampScore(scores[item.key]);
        return (
          <div key={item.key}>
            <div className="flex items-center justify-between gap-3 text-[12px] font-semibold text-cocoa-600">
              <span>{item.label}</span>
              <span>{value}</span>
            </div>
            <div className="mt-1 h-2 overflow-hidden rounded-full bg-sand-200">
              <div
                className="h-full rounded-full bg-clay"
                style={{ width: `${value}%` }}
              />
            </div>
          </div>
        );
      })}
    </div>
  );
}

function clampScore(value: number | null | undefined) {
  if (typeof value !== "number" || Number.isNaN(value)) {
    return 0;
  }
  return Math.max(0, Math.min(100, Math.round(value)));
}
