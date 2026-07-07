import Link from "next/link";
import { Card } from "@/shared/ui/card";
import {
  formatAnalyticsDate,
  formatAnalyticsMoney,
  formatPercent,
  formatPlainMoney
} from "@/components/analytics/format";
import type { WorkspaceBudgetByTrip } from "@/entities/workspace-budget/model";

type BudgetUtilizationProps = {
  amount: number;
  currency: string;
  summary: {
    estimatedTotal: number;
    utilizationPercent: number;
    overBudgetAmount: number;
  };
};

export function BudgetUtilization({ amount, currency, summary }: BudgetUtilizationProps) {
  const progress = Math.min(Math.max(summary.utilizationPercent, 0), 100);
  const over = summary.overBudgetAmount > 0;
  return (
    <Card>
      <div className="flex items-center justify-between gap-3">
        <div>
          <h2 className="text-lg font-semibold text-slate-950">Utilization</h2>
          <p className="mt-1 text-sm text-slate-600">
            {formatAnalyticsMoney(summary.estimatedTotal, currency)} of {formatPlainMoney(amount, currency)}
          </p>
        </div>
        <span className={over ? "text-lg font-semibold text-red-700" : "text-lg font-semibold text-primary-700"}>
          {formatPercent(summary.utilizationPercent)}
        </span>
      </div>
      <div className="mt-5 h-4 rounded-full bg-slate-100">
        <div
          className={over ? "h-4 rounded-full bg-red-600" : "h-4 rounded-full bg-primary-600"}
          style={{ width: `${progress}%` }}
        />
      </div>
    </Card>
  );
}

export function BudgetTripsTable({
  trips,
  currency
}: {
  trips: WorkspaceBudgetByTrip[];
  currency: string;
}) {
  return (
    <Card className="overflow-hidden p-0">
      <div className="p-5">
        <h2 className="text-lg font-semibold text-slate-950">Cost by trip</h2>
      </div>
      {trips.length === 0 ? (
        <p className="px-5 pb-5 text-sm text-slate-500">No workspace trips match this budget period.</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-slate-100 text-sm">
            <thead className="bg-slate-50 text-left text-xs font-semibold uppercase text-slate-500">
              <tr>
                <th className="px-4 py-3">Trip</th>
                <th className="px-4 py-3">Start date</th>
                <th className="px-4 py-3 text-right">Estimated total</th>
                <th className="px-4 py-3 text-right">Budget share</th>
                <th className="px-4 py-3 text-right">Missing</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100 bg-white">
              {trips.map((trip) => (
                <tr key={trip.tripId}>
                  <td className="px-4 py-3">
                    <Link
                      className="font-medium text-primary-700 hover:text-primary-600"
                      href={`/trips/${trip.tripId}/analytics`}
                    >
                      {trip.title || trip.destination}
                    </Link>
                    <div className="mt-1 text-xs text-slate-500">{trip.destination}</div>
                  </td>
                  <td className="px-4 py-3 text-slate-700">{formatAnalyticsDate(trip.startDate)}</td>
                  <td className="px-4 py-3 text-right font-medium text-slate-900">
                    {formatAnalyticsMoney(trip.estimatedTotal, currency)}
                  </td>
                  <td className="px-4 py-3 text-right text-slate-700">
                    {formatPercent(trip.percentageOfBudget)}
                  </td>
                  <td className="px-4 py-3 text-right text-slate-700">
                    {trip.missingEstimateCount}
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
