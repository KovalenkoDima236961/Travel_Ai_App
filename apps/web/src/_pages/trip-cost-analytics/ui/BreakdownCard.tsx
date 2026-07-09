import {
  formatAnalyticsLabel,
  formatAnalyticsMoney,
  formatPercent
} from "@/components/analytics/format";
import type { CostAmountBreakdown } from "@/entities/cost-analytics/model";

type BreakdownCardProps = {
  title: string;
  entries: CostAmountBreakdown[];
  currency: string;
  valueKey: "category" | "source" | "confidence" | "name";
  emptyText?: string;
};

/**
 * Slice-local restyle of the shared CostBreakdownBars — the mock's stacked
 * label + progress bar rows. Reused for the category, source, and confidence
 * breakdowns so the full trip data model survives the redesign.
 */
export function BreakdownCard({
  title,
  entries,
  currency,
  valueKey,
  emptyText = "No cost data yet."
}: BreakdownCardProps) {
  const maxAmount = Math.max(...entries.map((entry) => entry.amount), 0);

  return (
    <div className="rounded-[20px] border border-sand-300 bg-white px-7 py-[26px]">
      <h2 className="font-newsreader text-[20px] font-semibold text-cocoa-900">{title}</h2>
      {entries.length === 0 ? (
        <p className="mt-6 text-sm text-cocoa-400">{emptyText}</p>
      ) : (
        <div className="mt-[22px] flex flex-col gap-4">
          {entries.map((entry) => {
            const label = formatAnalyticsLabel(labelForEntry(entry, valueKey));
            const width = maxAmount > 0 ? Math.max(4, (entry.amount / maxAmount) * 100) : 0;
            return (
              <div key={`${title}-${label}`}>
                <div className="mb-[7px] flex items-center justify-between gap-3">
                  <span className="text-[13.5px] font-medium text-cocoa-700">{label}</span>
                  <span className="text-[13px] text-cocoa-400">
                    {formatAnalyticsMoney(entry.amount, currency)} · {formatPercent(entry.percentage)}
                  </span>
                </div>
                <div
                  aria-label={`${label} ${formatAnalyticsMoney(entry.amount, currency)} ${formatPercent(entry.percentage)}`}
                  className="h-[9px] overflow-hidden rounded-full bg-sand-200"
                  role="img"
                >
                  <div className="h-full rounded-full bg-clay" style={{ width: `${width}%` }} />
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

function labelForEntry(entry: CostAmountBreakdown, key: BreakdownCardProps["valueKey"]) {
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
