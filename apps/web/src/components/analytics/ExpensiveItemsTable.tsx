import Link from "next/link";
import { Button } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import {
  formatAnalyticsLabel,
  formatAnalyticsMoney,
  formatPercent
} from "@/components/analytics/format";
import type { ExpensiveCostItem } from "@/entities/cost-analytics/model";

type ExpensiveItemsTableProps = {
  items: ExpensiveCostItem[];
  currency: string;
  showTrip?: boolean;
  canEdit?: boolean;
  onOptimizeDay?: (dayNumber: number) => void;
};

export function ExpensiveItemsTable({
  items,
  currency,
  showTrip = false,
  canEdit = false,
  onOptimizeDay
}: ExpensiveItemsTableProps) {
  return (
    <Card className="overflow-hidden p-0">
      <div className="p-5">
        <h2 className="text-lg font-semibold text-slate-950">Expensive items</h2>
      </div>
      {items.length === 0 ? (
        <p className="px-5 pb-5 text-sm text-slate-500">No estimated item costs yet.</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-slate-100 text-sm">
            <thead className="bg-slate-50 text-left text-xs font-semibold uppercase text-slate-500">
              <tr>
                {showTrip ? <th className="px-4 py-3">Trip</th> : null}
                <th className="px-4 py-3">Item</th>
                <th className="px-4 py-3">Category</th>
                <th className="px-4 py-3">Source</th>
                <th className="px-4 py-3 text-right">Estimate</th>
                <th className="px-4 py-3 text-right">Share</th>
                <th className="px-4 py-3 text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100 bg-white">
              {items.map((item, index) => (
                <tr key={`${item.tripId ?? "trip"}-${item.dayNumber ?? 0}-${item.itemIndex ?? index}-${item.name}`}>
                  {showTrip ? (
                    <td className="px-4 py-3 text-slate-700">
                      {item.tripId ? (
                        <Link className="font-medium text-primary-700 hover:text-primary-600" href={`/trips/${item.tripId}`}>
                          {item.tripTitle || item.destination || item.tripId}
                        </Link>
                      ) : (
                        item.tripTitle || item.destination || "Trip"
                      )}
                    </td>
                  ) : null}
                  <td className="px-4 py-3">
                    <div className="font-medium text-slate-900">{item.name}</div>
                    <div className="mt-1 text-xs text-slate-500">
                      {item.dayNumber ? `Day ${item.dayNumber}` : "Trip cost"}
                      {item.itemIndex != null ? ` · Item ${item.itemIndex + 1}` : ""}
                    </div>
                  </td>
                  <td className="px-4 py-3 text-slate-700">{formatAnalyticsLabel(item.category)}</td>
                  <td className="px-4 py-3 text-slate-700">
                    {formatAnalyticsLabel(item.source)}
                    <span className="ml-1 text-xs text-slate-400">
                      {formatAnalyticsLabel(item.confidence)}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-right font-medium text-slate-900">
                    {formatAnalyticsMoney(item.convertedAmount ?? item.amount, currency)}
                  </td>
                  <td className="px-4 py-3 text-right text-slate-700">
                    {formatPercent(item.percentageOfTrip)}
                  </td>
                  <td className="px-4 py-3 text-right">
                    <div className="flex justify-end gap-2">
                      {item.tripId && item.dayNumber ? (
                        <Link
                          className="text-sm font-medium text-primary-700 hover:text-primary-600"
                          href={`/trips/${item.tripId}#day-${item.dayNumber}-item-${item.itemIndex ?? 0}`}
                        >
                          Open
                        </Link>
                      ) : item.dayNumber ? (
                        <Link
                          className="text-sm font-medium text-primary-700 hover:text-primary-600"
                          href={`#day-${item.dayNumber}-item-${item.itemIndex ?? 0}`}
                        >
                          Open
                        </Link>
                      ) : null}
                      {canEdit && item.dayNumber && onOptimizeDay ? (
                        <Button onClick={() => onOptimizeDay(item.dayNumber ?? 1)} size="sm" type="button" variant="ghost">
                          Optimize
                        </Button>
                      ) : null}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </Card>
  );
}
