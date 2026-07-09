import {
  formatAnalyticsMoney,
  formatPlainMoney
} from "@/components/analytics/format";
import type { CostByDay } from "@/entities/cost-analytics/model";

type CostByDayCardProps = {
  days: CostByDay[];
  currency: string;
};

const MAX_BAR_HEIGHT = 146;

/**
 * Slice-local restyle of the shared CostByDayChart as the mock's vertical bar
 * chart. Heights scale to the largest day (like the original scaled bar width),
 * over-budget days turn red, and the row scrolls horizontally so a long trip's
 * many days degrade gracefully instead of squeezing to slivers.
 */
export function CostByDayCard({ days, currency }: CostByDayCardProps) {
  const maxAmount = Math.max(...days.map((day) => day.estimatedTotal), 0);

  return (
    <div className="rounded-[20px] border border-sand-300 bg-white px-7 py-[26px]">
      <h2 className="font-newsreader text-[20px] font-semibold text-cocoa-900">Cost by day</h2>
      {days.length === 0 ? (
        <p className="mt-6 text-sm text-cocoa-400">No daily cost data yet.</p>
      ) : (
        <div className="mt-6 overflow-x-auto">
          <div className="flex min-w-full items-end justify-around gap-5" style={{ height: 200 }}>
            {days.map((day) => {
              const over = (day.overBudgetAmount ?? 0) > 0;
              const ratio = maxAmount > 0 ? day.estimatedTotal / maxAmount : 0;
              const height = Math.max(6, ratio * MAX_BAR_HEIGHT);
              const color = over ? "#C0392B" : ratio >= 0.8 ? "#C05B3B" : "#E0885E";
              return (
                <div
                  className="flex min-w-[46px] flex-1 flex-col items-center gap-2.5"
                  key={day.dayNumber}
                >
                  <span
                    className={
                      over
                        ? "text-[12.5px] font-semibold text-[#C0392B]"
                        : "text-[12.5px] font-semibold text-cocoa-500"
                    }
                  >
                    {formatAnalyticsMoney(day.estimatedTotal, currency)}
                  </span>
                  <div
                    className="w-full max-w-[52px] rounded-[10px_10px_4px_4px]"
                    style={{ height, background: color }}
                    title={
                      day.budgetShare != null
                        ? `Budget ${formatPlainMoney(day.budgetShare, currency)}`
                        : undefined
                    }
                  />
                  <span className="text-xs text-cocoa-400">Day {day.dayNumber}</span>
                  {day.missingEstimateCount > 0 ? (
                    <span className="text-[11px] font-medium text-[#96682A]">
                      {day.missingEstimateCount} missing
                    </span>
                  ) : null}
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
