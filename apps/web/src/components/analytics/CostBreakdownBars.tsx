import { Card } from "@/shared/ui/card";
import {
  formatAnalyticsLabel,
  formatAnalyticsMoney,
  formatPercent
} from "@/components/analytics/format";
import type { CostAmountBreakdown } from "@/entities/cost-analytics/model";

type CostBreakdownBarsProps = {
  title: string;
  entries: CostAmountBreakdown[];
  currency: string;
  valueKey: "category" | "source" | "confidence" | "name";
  emptyText?: string;
};

export function CostBreakdownBars({
  title,
  entries,
  currency,
  valueKey,
  emptyText = "No cost data yet."
}: CostBreakdownBarsProps) {
  const maxAmount = Math.max(...entries.map((entry) => entry.amount), 0);

  return (
    <Card>
      <h2 className="text-lg font-semibold text-slate-950">{title}</h2>
      {entries.length === 0 ? (
        <p className="mt-4 text-sm text-slate-500">{emptyText}</p>
      ) : (
        <div className="mt-5 space-y-4">
          {entries.map((entry) => {
            const label = labelForEntry(entry, valueKey);
            const width = maxAmount > 0 ? Math.max(5, (entry.amount / maxAmount) * 100) : 0;
            return (
              <div key={`${title}-${label}`}>
                <div className="flex items-center justify-between gap-3 text-sm">
                  <span className="font-medium text-slate-800">{formatAnalyticsLabel(label)}</span>
                  <span className="text-right text-slate-600">
                    {formatAnalyticsMoney(entry.amount, currency)} · {formatPercent(entry.percentage)}
                  </span>
                </div>
                <div
                  aria-label={`${formatAnalyticsLabel(label)} ${formatAnalyticsMoney(entry.amount, currency)} ${formatPercent(entry.percentage)}`}
                  className="mt-2 h-3 rounded-full bg-slate-100"
                  role="img"
                >
                  <div
                    className="h-3 rounded-full bg-primary-600"
                    style={{ width: `${width}%` }}
                  />
                </div>
                <p className="mt-1 text-xs text-slate-500">
                  {entry.itemCount} {entry.itemCount === 1 ? "item" : "items"}
                </p>
              </div>
            );
          })}
        </div>
      )}
    </Card>
  );
}

function labelForEntry(entry: CostAmountBreakdown, key: CostBreakdownBarsProps["valueKey"]) {
  if (key === "category") {
    return entry.category ?? entry.name ?? "unknown";
  }
  if (key === "source") {
    return entry.source ?? entry.name ?? "unknown";
  }
  if (key === "confidence") {
    return entry.confidence ?? entry.name ?? "unknown";
  }
  return entry.name ?? entry.category ?? entry.source ?? entry.confidence ?? "unknown";
}
