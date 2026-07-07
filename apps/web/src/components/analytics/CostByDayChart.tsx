import { Card } from "@/shared/ui/card";
import {
  formatAnalyticsDate,
  formatAnalyticsMoney,
  formatPlainMoney
} from "@/components/analytics/format";
import type { CostByDay } from "@/entities/cost-analytics/model";

type CostByDayChartProps = {
  days: CostByDay[];
  currency: string;
};

export function CostByDayChart({ days, currency }: CostByDayChartProps) {
  const maxAmount = Math.max(...days.map((day) => day.estimatedTotal), 0);

  return (
    <Card>
      <h2 className="text-lg font-semibold text-slate-950">Cost by day</h2>
      {days.length === 0 ? (
        <p className="mt-4 text-sm text-slate-500">No daily cost data yet.</p>
      ) : (
        <div className="mt-5 space-y-4">
          {days.map((day) => {
            const width = maxAmount > 0 ? Math.max(5, (day.estimatedTotal / maxAmount) * 100) : 0;
            const over = (day.overBudgetAmount ?? 0) > 0;
            return (
              <div key={day.dayNumber}>
                <div className="flex flex-col gap-1 text-sm sm:flex-row sm:items-center sm:justify-between">
                  <div>
                    <span className="font-medium text-slate-800">Day {day.dayNumber}</span>
                    <span className="ml-2 text-slate-500">{formatAnalyticsDate(day.date)}</span>
                  </div>
                  <div className={over ? "font-medium text-red-700" : "text-slate-700"}>
                    {formatAnalyticsMoney(day.estimatedTotal, currency)}
                    {day.budgetShare != null ? ` / ${formatPlainMoney(day.budgetShare, currency)}` : ""}
                  </div>
                </div>
                <div className="mt-2 h-3 rounded-full bg-slate-100">
                  <div
                    className={over ? "h-3 rounded-full bg-red-600" : "h-3 rounded-full bg-primary-600"}
                    style={{ width: `${width}%` }}
                  />
                </div>
                <div className="mt-1 flex flex-wrap gap-x-4 gap-y-1 text-xs text-slate-500">
                  {over ? <span>Over by {formatPlainMoney(day.overBudgetAmount, currency)}</span> : null}
                  {day.missingEstimateCount > 0 ? (
                    <span>
                      {day.missingEstimateCount} missing {day.missingEstimateCount === 1 ? "estimate" : "estimates"}
                    </span>
                  ) : null}
                </div>
              </div>
            );
          })}
        </div>
      )}
    </Card>
  );
}
