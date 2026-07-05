"use client";

import { Button } from "@/components/ui/Button";
import { formatMoney } from "@/lib/budget/format";
import type { TravelerCostAllocation } from "@/types/cost-splitting";

type TravelerCostDetailDrawerProps = {
  traveler: TravelerCostAllocation | null;
  currency: string;
  onClose: () => void;
};

export function TravelerCostDetailDrawer({
  traveler,
  currency,
  onClose
}: TravelerCostDetailDrawerProps) {
  if (!traveler) {
    return null;
  }

  return (
    <div className="fixed inset-0 z-40 flex justify-end bg-slate-950/30">
      <aside className="h-full w-full max-w-xl overflow-y-auto border-l border-slate-200 bg-white p-5 shadow-xl">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h2 className="text-lg font-semibold text-slate-950">{traveler.name}</h2>
            <p className="mt-1 text-sm text-slate-600">
              {formatMoney(traveler.allocatedTotal, currency)} estimated allocation
            </p>
          </div>
          <Button onClick={onClose} size="sm" type="button" variant="ghost">
            Close
          </Button>
        </div>

        <section className="mt-6">
          <h3 className="text-sm font-semibold uppercase tracking-wide text-slate-500">
            By category
          </h3>
          <ul className="mt-2 space-y-1 text-sm">
            {traveler.byCategory.map((entry) => (
              <li className="flex justify-between gap-3" key={entry.category}>
                <span className="capitalize text-slate-600">{entry.category}</span>
                <span className="font-medium text-slate-900">
                  {formatMoney(entry.amount, currency)}
                </span>
              </li>
            ))}
          </ul>
        </section>

        {traveler.byDay.length > 0 ? (
          <section className="mt-6">
            <h3 className="text-sm font-semibold uppercase tracking-wide text-slate-500">
              By day
            </h3>
            <ul className="mt-2 space-y-1 text-sm">
              {traveler.byDay.map((entry) => (
                <li className="flex justify-between gap-3" key={entry.dayNumber}>
                  <span className="text-slate-600">Day {entry.dayNumber}</span>
                  <span className="font-medium text-slate-900">
                    {formatMoney(entry.amount, currency)}
                  </span>
                </li>
              ))}
            </ul>
          </section>
        ) : null}

        <section className="mt-6">
          <h3 className="text-sm font-semibold uppercase tracking-wide text-slate-500">
            Allocated items
          </h3>
          <div className="mt-3 divide-y divide-slate-100 rounded-md border border-slate-200">
            {traveler.items.length > 0 ? (
              traveler.items.map((item, index) => (
                <div className="p-3 text-sm" key={`${item.type}-${item.dayNumber}-${item.itemIndex}-${index}`}>
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <p className="font-medium text-slate-950">{item.name}</p>
                      <p className="mt-1 text-xs text-slate-500">
                        {item.dayNumber ? `Day ${item.dayNumber}` : "Accommodation"} · {item.category} · {item.splitType}
                      </p>
                    </div>
                    <span className="font-semibold text-slate-900">
                      {formatMoney(item.allocatedAmount, currency)}
                    </span>
                  </div>
                  <p className="mt-1 text-xs text-slate-500">
                    Original: {formatMoney(item.originalCostAmount, item.originalCostCurrency)}
                  </p>
                </div>
              ))
            ) : (
              <p className="p-3 text-sm text-slate-500">No allocated items.</p>
            )}
          </div>
        </section>
      </aside>
    </div>
  );
}
