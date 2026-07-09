import Link from "next/link";
import {
  formatAnalyticsLabel,
  formatAnalyticsMoney,
  formatPercent
} from "@/components/analytics/format";
import type { ExpensiveCostItem } from "@/entities/cost-analytics/model";

type ExpensiveItemsCardProps = {
  items: ExpensiveCostItem[];
  currency: string;
  canEdit?: boolean;
  onOptimizeDay?: (dayNumber: number) => void;
};

const COLS = "grid-cols-[minmax(0,2.2fr)_1fr_1fr_1fr_auto]";

/**
 * Slice-local restyle of the shared ExpensiveItemsTable as the mock's warm grid
 * table. Keeps the source/confidence, share, and per-item Open/Optimize actions
 * the trip page exposes today rather than the mock's four-column subset.
 */
export function ExpensiveItemsCard({
  items,
  currency,
  canEdit = false,
  onOptimizeDay
}: ExpensiveItemsCardProps) {
  return (
    <div className="rounded-[20px] border border-sand-300 bg-white px-7 py-[26px]">
      <h2 className="font-newsreader text-[20px] font-semibold text-cocoa-900">
        Most expensive items
      </h2>
      {items.length === 0 ? (
        <p className="mt-[18px] text-sm text-cocoa-400">No estimated item costs yet.</p>
      ) : (
        <div className="mt-[18px] overflow-hidden rounded-[14px] border border-sand-200">
          <div
            className={`grid ${COLS} gap-3 bg-sand-50 px-[18px] py-3 text-xs font-semibold uppercase tracking-[0.04em] text-cocoa-400`}
          >
            <span>Item</span>
            <span>Category</span>
            <span>Source</span>
            <span className="text-right">Estimate</span>
            <span className="text-right">Actions</span>
          </div>
          {items.map((item, index) => (
            <div
              className={`grid ${COLS} items-center gap-3 border-t border-sand-200 px-[18px] py-3.5`}
              key={`${item.tripId ?? "trip"}-${item.dayNumber ?? 0}-${item.itemIndex ?? index}-${item.name}`}
            >
              <div className="min-w-0">
                <p className="truncate text-sm font-medium text-cocoa-900">{item.name}</p>
                <p className="mt-0.5 text-xs text-cocoa-400">
                  {item.dayNumber ? `Day ${item.dayNumber}` : "Trip cost"}
                  {item.itemIndex != null ? ` · Item ${item.itemIndex + 1}` : ""}
                </p>
              </div>
              <span className="text-[13.5px] text-cocoa-500">
                {formatAnalyticsLabel(item.category)}
              </span>
              <span className="text-[13.5px] text-cocoa-500">
                {formatAnalyticsLabel(item.source)}
                <span className="ml-1 text-xs text-cocoa-400">
                  {formatAnalyticsLabel(item.confidence)}
                </span>
              </span>
              <div className="text-right">
                <p className="text-sm font-semibold text-cocoa-900">
                  {formatAnalyticsMoney(item.convertedAmount ?? item.amount, currency)}
                </p>
                <p className="mt-0.5 text-xs text-cocoa-400">
                  {formatPercent(item.percentageOfTrip)}
                </p>
              </div>
              <div className="flex items-center justify-end gap-3">
                {item.tripId && item.dayNumber ? (
                  <Link
                    className="text-[13px] font-semibold text-clay-deep hover:text-clay"
                    href={`/trips/${item.tripId}#day-${item.dayNumber}-item-${item.itemIndex ?? 0}`}
                  >
                    Open
                  </Link>
                ) : item.dayNumber ? (
                  <Link
                    className="text-[13px] font-semibold text-clay-deep hover:text-clay"
                    href={`#day-${item.dayNumber}-item-${item.itemIndex ?? 0}`}
                  >
                    Open
                  </Link>
                ) : null}
                {canEdit && item.dayNumber && onOptimizeDay ? (
                  <button
                    className="text-[13px] font-semibold text-cocoa-500 hover:text-cocoa-900"
                    onClick={() => onOptimizeDay(item.dayNumber ?? 1)}
                    type="button"
                  >
                    Optimize
                  </button>
                ) : null}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
