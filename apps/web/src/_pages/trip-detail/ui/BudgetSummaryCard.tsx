import { formatMoney } from "@/entities/budget/model";
import type { BudgetSummary } from "@/entities/budget/model";
import { SparklesIcon } from "./icons";

type BudgetSummaryCardProps = {
  summary: BudgetSummary | null;
  currency: string;
  isLoading: boolean;
  canEdit: boolean;
  optimizationDisabled: boolean;
  perPersonAverage?: { amount: number; currency: string } | null;
  onOpenBudgetOptimization?: (dayNumber: number) => void;
};

/**
 * Warm left-rail budget summary, forked from the shared BudgetPanel's read view.
 * Editing still lives in the full BudgetPanel anchored at #budget below; this card
 * mirrors the mock (spend vs budget bar + AI-optimize CTA) and reuses the same
 * suggested-day heuristic so "Optimize with AI" targets the right day.
 */
export function BudgetSummaryCard({
  summary,
  currency,
  isLoading,
  canEdit,
  optimizationDisabled,
  perPersonAverage,
  onOpenBudgetOptimization
}: BudgetSummaryCardProps) {
  return (
    <div className="rounded-[18px] border border-sand-300 bg-white p-5">
      <div className="flex items-baseline justify-between gap-2">
        <h2 className="text-[13px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">
          Budget
        </h2>
        <a
          href="#budget"
          className="text-[12.5px] font-semibold text-clay-deep transition hover:text-clay"
        >
          Edit
        </a>
      </div>

      {isLoading ? (
        <p className="mt-3.5 text-[14px] text-cocoa-400">Loading budget…</p>
      ) : !summary ? (
        <p className="mt-3.5 text-[14px] text-cocoa-400">Budget summary is unavailable.</p>
      ) : (
        <BudgetSummaryBody
          summary={summary}
          currency={currency}
          canEdit={canEdit}
          optimizationDisabled={optimizationDisabled}
          perPersonAverage={perPersonAverage}
          onOpenBudgetOptimization={onOpenBudgetOptimization}
        />
      )}
    </div>
  );
}

function BudgetSummaryBody({
  summary,
  currency,
  canEdit,
  optimizationDisabled,
  perPersonAverage,
  onOpenBudgetOptimization
}: {
  summary: BudgetSummary;
  currency: string;
  canEdit: boolean;
  optimizationDisabled: boolean;
  perPersonAverage?: { amount: number; currency: string } | null;
  onOpenBudgetOptimization?: (dayNumber: number) => void;
}) {
  const hasBudget = summary.tripBudget != null;
  const overBudget = (summary.overBudgetBy ?? 0) > 0;
  const spent = summary.estimatedTotal;
  const budget = summary.tripBudget ?? 0;
  const ratio = hasBudget && budget > 0 ? Math.min(1, spent / budget) : overBudget ? 1 : 0;
  const suggestedDay = getSuggestedOptimizationDay(summary);
  const canOptimize = canEdit && suggestedDay != null && Boolean(onOpenBudgetOptimization);
  const perPerson = perPersonAverage
    ? ` · ~${formatMoney(perPersonAverage.amount, perPersonAverage.currency)} per person`
    : "";

  return (
    <>
      <p className="mt-3.5 font-newsreader text-[26px] font-semibold text-cocoa-900">
        {formatMoney(spent, currency)}
        {hasBudget ? (
          <span className="text-[15px] font-normal text-cocoa-400">
            {" "}
            of {formatMoney(summary.tripBudget, currency)}
          </span>
        ) : null}
      </p>

      {hasBudget ? (
        <div className="mt-3 h-[7px] overflow-hidden rounded-full bg-sand-200">
          <div
            className="h-full rounded-full"
            style={{
              width: `${Math.round(ratio * 100)}%`,
              background: overBudget ? "#B3402E" : "#C05B3B"
            }}
          />
        </div>
      ) : null}

      {hasBudget && overBudget ? (
        <>
          <p className="mt-2.5 text-[13px] font-semibold text-[#B3402E]">
            {formatMoney(summary.overBudgetBy, currency)} over budget
          </p>
          {canOptimize ? (
            <button
              type="button"
              disabled={optimizationDisabled}
              onClick={() => onOpenBudgetOptimization?.(suggestedDay)}
              className="mt-3 inline-flex h-9 items-center gap-1.5 rounded-full border border-[#E5C3B6] bg-[#FBF0EB] px-3.5 text-[13px] font-semibold text-clay-deep transition hover:bg-[#F7E4DB] disabled:cursor-not-allowed disabled:opacity-60"
            >
              <SparklesIcon className="h-3.5 w-3.5" />
              Optimize with AI
            </button>
          ) : null}
        </>
      ) : hasBudget ? (
        <p className="mt-2.5 text-[13px] text-cocoa-400">
          {formatMoney(summary.remaining, currency)} remaining{perPerson}
        </p>
      ) : (
        <p className="mt-2.5 text-[13px] text-cocoa-400">No budget set{perPerson}</p>
      )}

      {summary.missingEstimateCount > 0 ? (
        <p className="mt-2.5 text-[12.5px] text-cocoa-400">
          {summary.missingEstimateCount} item
          {summary.missingEstimateCount === 1 ? "" : "s"} still need a cost estimate.
        </p>
      ) : null}
    </>
  );
}

/** Mirrors BudgetPanel.getSuggestedOptimizationDay so the CTA targets the same day. */
function getSuggestedOptimizationDay(summary: BudgetSummary): number | null {
  if (summary.byDay.length === 0) {
    return null;
  }
  const overBudgetDay = [...summary.byDay]
    .filter((day) => (day.overDailyBudgetBy ?? 0) > 0)
    .sort((left, right) => (right.overDailyBudgetBy ?? 0) - (left.overDailyBudgetBy ?? 0))[0];
  if (overBudgetDay) {
    return overBudgetDay.dayNumber;
  }
  if ((summary.overBudgetBy ?? 0) <= 0) {
    return null;
  }
  return [...summary.byDay].sort(
    (left, right) => right.estimatedTotal - left.estimatedTotal
  )[0].dayNumber;
}
