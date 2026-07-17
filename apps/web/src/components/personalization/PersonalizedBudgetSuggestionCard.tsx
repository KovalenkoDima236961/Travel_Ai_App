"use client";
import type { BudgetSuggestion } from "@/types/personalization";

export function PersonalizedBudgetSuggestionCard({ suggestion, onUse }: { suggestion: BudgetSuggestion; onUse?: () => void }) {
  const { min, max } = suggestion.suggestedRange;
  return <section className="rounded-[18px] border border-sand-300 bg-white p-5"><h3 className="font-newsreader text-xl text-cocoa-900">Suggested budget</h3><p className="mt-2 text-lg font-semibold text-cocoa-800">{min.amount.toLocaleString()}–{max.amount.toLocaleString()} {min.currency}</p><ul className="mt-3 space-y-1 text-sm text-cocoa-600">{suggestion.reasons.map((reason) => <li key={reason}>• {reason}</li>)}</ul>{onUse ? <button type="button" className="mt-4 rounded-full bg-cocoa-900 px-4 py-2 text-[13px] font-semibold text-sand-100" onClick={onUse}>Use suggested budget</button> : null}</section>;
}
