"use client";

import { useState } from "react";
import { TravelerCostDetailDrawer } from "./TravelerCostDetailDrawer";
import { ResponsiveDataView } from "@/components/ui";
import { Card } from "@/shared/ui/card";
import { formatMoney } from "@/entities/budget/model";
import type {
  CostSplittingSummary,
  TravelerCostAllocation
} from "@/entities/cost-splitting/model";

const VISIBLE_CATEGORIES = ["accommodation", "food", "transport", "ticket", "activity", "other"];

type PerTravelerCostTableProps = {
  summary: CostSplittingSummary;
};

export function PerTravelerCostTable({ summary }: PerTravelerCostTableProps) {
  const [selectedTraveler, setSelectedTraveler] = useState<TravelerCostAllocation | null>(null);

  return (
    <Card>
      <h2 className="text-lg font-semibold text-slate-950">Per-traveler summary</h2>
      <ResponsiveDataView
        className="mt-4"
        empty={<p className="text-sm text-slate-500">No traveler allocations yet.</p>}
        getKey={(traveler) => traveler.travelerId}
        items={summary.travelers}
        desktop={
          <div className="overflow-x-auto">
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
                {summary.travelers.map((traveler) => (
                  <tr
                    aria-label={`View allocation details for ${traveler.name}`}
                    className="cursor-pointer outline-none hover:bg-slate-50 focus-visible:bg-slate-50"
                    key={traveler.travelerId}
                    onClick={() => setSelectedTraveler(traveler)}
                    onKeyDown={(event) => {
                      if (event.key === "Enter" || event.key === " ") {
                        event.preventDefault();
                        setSelectedTraveler(traveler);
                      }
                    }}
                    tabIndex={0}
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
                ))}
              </tbody>
            </table>
          </div>
        }
        renderMobileCard={(traveler) => (
          <button
            aria-label={`View allocation details for ${traveler.name}`}
            className="block min-h-11 w-full rounded-lg border border-slate-200 bg-slate-50 p-4 text-left transition hover:border-slate-300 hover:bg-white"
            onClick={() => setSelectedTraveler(traveler)}
            type="button"
          >
            <div className="flex items-start justify-between gap-3">
              <div className="min-w-0">
                <p className="truncate font-semibold text-slate-950">{traveler.name}</p>
                <p className="mt-1 truncate text-xs text-slate-500">{traveler.email ?? traveler.role}</p>
              </div>
              <span className="shrink-0 text-right text-sm font-semibold text-slate-950">
                {formatMoney(traveler.allocatedTotal, summary.currency)}
              </span>
            </div>
            <dl className="mt-3 grid grid-cols-2 gap-x-3 gap-y-2 text-xs">
              {VISIBLE_CATEGORIES.slice(0, 4).map((category) => (
                <div className="flex justify-between gap-2" key={category}>
                  <dt className="capitalize text-slate-500">
                    {category === "ticket" ? "Tickets" : category}
                  </dt>
                  <dd className="font-medium text-slate-700">
                    {formatMoney(categoryAmount(traveler, category), summary.currency)}
                  </dd>
                </div>
              ))}
            </dl>
            <p className="mt-3 text-xs font-medium text-primary-700">
              {traveler.percentageOfTotal.toFixed(2)}% of trip total · View details
            </p>
          </button>
        )}
      />
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
