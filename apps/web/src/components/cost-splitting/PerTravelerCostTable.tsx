"use client";

import { useState } from "react";
import { TravelerCostDetailDrawer } from "@/components/cost-splitting/TravelerCostDetailDrawer";
import { Card } from "@/components/ui/Card";
import { formatMoney } from "@/lib/budget/format";
import type {
  CostSplittingSummary,
  TravelerCostAllocation
} from "@/types/cost-splitting";

const VISIBLE_CATEGORIES = ["accommodation", "food", "transport", "ticket", "activity", "other"];

type PerTravelerCostTableProps = {
  summary: CostSplittingSummary;
};

export function PerTravelerCostTable({ summary }: PerTravelerCostTableProps) {
  const [selectedTraveler, setSelectedTraveler] = useState<TravelerCostAllocation | null>(null);

  return (
    <Card>
      <h2 className="text-lg font-semibold text-slate-950">Per-traveler summary</h2>
      <div className="mt-4 overflow-x-auto">
        <table className="min-w-full divide-y divide-slate-200 text-sm">
          <thead>
            <tr className="text-left text-xs font-semibold uppercase tracking-wide text-slate-500">
              <th className="py-2 pr-4">Traveler</th>
              <th className="px-4 py-2 text-right">Total</th>
              {VISIBLE_CATEGORIES.map((category) => (
                <th className="px-4 py-2 text-right capitalize" key={category}>
                  {category === "ticket" ? "Tickets" : category}
                </th>
              ))}
              <th className="py-2 pl-4 text-right">Percent</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100">
            {summary.travelers.length > 0 ? (
              summary.travelers.map((traveler) => (
                <tr
                  className="cursor-pointer hover:bg-slate-50"
                  key={traveler.travelerId}
                  onClick={() => setSelectedTraveler(traveler)}
                >
                  <td className="py-3 pr-4">
                    <p className="font-medium text-slate-950">{traveler.name}</p>
                    <p className="text-xs text-slate-500">{traveler.email ?? traveler.role}</p>
                  </td>
                  <td className="px-4 py-3 text-right font-semibold text-slate-900">
                    {formatMoney(traveler.allocatedTotal, summary.currency)}
                  </td>
                  {VISIBLE_CATEGORIES.map((category) => (
                    <td className="px-4 py-3 text-right text-slate-700" key={category}>
                      {formatMoney(categoryAmount(traveler, category), summary.currency)}
                    </td>
                  ))}
                  <td className="py-3 pl-4 text-right text-slate-700">
                    {traveler.percentageOfTotal.toFixed(2)}%
                  </td>
                </tr>
              ))
            ) : (
              <tr>
                <td className="py-4 text-slate-500" colSpan={VISIBLE_CATEGORIES.length + 3}>
                  No traveler allocations yet.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
      <TravelerCostDetailDrawer
        currency={summary.currency}
        onClose={() => setSelectedTraveler(null)}
        traveler={selectedTraveler}
      />
    </Card>
  );
}

function categoryAmount(traveler: TravelerCostAllocation, category: string) {
  return traveler.byCategory.find((entry) => entry.category === category)?.amount ?? 0;
}
