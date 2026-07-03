import Link from "next/link";
import { Card } from "@/components/ui/Card";
import {
  formatAnalyticsDate,
  formatAnalyticsMoney,
  formatPlainMoney
} from "@/components/analytics/format";
import type { TripCostSummary } from "@/types/cost-analytics";

type WorkspaceTripsCostTableProps = {
  trips: TripCostSummary[];
  currency: string;
};

export function WorkspaceTripsCostTable({ trips, currency }: WorkspaceTripsCostTableProps) {
  return (
    <Card className="overflow-hidden p-0">
      <div className="p-5">
        <h2 className="text-lg font-semibold text-slate-950">Cost by trip</h2>
      </div>
      {trips.length === 0 ? (
        <p className="px-5 pb-5 text-sm text-slate-500">No workspace trips match these filters.</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-slate-100 text-sm">
            <thead className="bg-slate-50 text-left text-xs font-semibold uppercase text-slate-500">
              <tr>
                <th className="px-4 py-3">Trip</th>
                <th className="px-4 py-3">Dates</th>
                <th className="px-4 py-3 text-right">Budget</th>
                <th className="px-4 py-3 text-right">Estimated total</th>
                <th className="px-4 py-3 text-right">Over budget</th>
                <th className="px-4 py-3 text-right">Missing</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100 bg-white">
              {trips.map((trip) => {
                const over = (trip.overBudgetAmount ?? 0) > 0;
                return (
                  <tr key={trip.tripId}>
                    <td className="px-4 py-3">
                      <Link className="font-medium text-primary-700 hover:text-primary-600" href={`/trips/${trip.tripId}/analytics`}>
                        {trip.title || trip.destination}
                      </Link>
                      <div className="mt-1 text-xs text-slate-500">{trip.destination}</div>
                    </td>
                    <td className="px-4 py-3 text-slate-700">
                      {formatAnalyticsDate(trip.startDate)}
                      {trip.endDate ? ` - ${formatAnalyticsDate(trip.endDate)}` : ""}
                    </td>
                    <td className="px-4 py-3 text-right text-slate-700">
                      {formatPlainMoney(trip.budgetAmount, currency)}
                    </td>
                    <td className="px-4 py-3 text-right font-medium text-slate-900">
                      {formatAnalyticsMoney(trip.estimatedTotal, currency)}
                    </td>
                    <td className={over ? "px-4 py-3 text-right font-medium text-red-700" : "px-4 py-3 text-right text-slate-500"}>
                      {over ? formatPlainMoney(trip.overBudgetAmount, currency) : "—"}
                    </td>
                    <td className="px-4 py-3 text-right text-slate-700">
                      {trip.missingEstimateCount}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </Card>
  );
}
