import { formatApproxMoney } from "@/entities/budget/model";
import type { BudgetOptimizationProposal } from "@/entities/budget-optimization/model";
import type { ItineraryDay } from "@/entities/trip/model";

type BudgetOptimizationPreviewProps = {
  currentDay?: ItineraryDay | null;
  proposal: BudgetOptimizationProposal;
};

export function BudgetOptimizationPreview({
  currentDay,
  proposal
}: BudgetOptimizationPreviewProps) {
  const proposedDay = proposal.proposal.proposedDay;
  const currency = proposal.currency;

  return (
    <div className="grid gap-4 lg:grid-cols-2">
      <DayPreview
        currency={currency}
        day={currentDay ?? null}
        emptyMessage="Current day is unavailable."
        title="Current Day"
      />
      <DayPreview
        currency={currency}
        day={proposedDay}
        emptyMessage="Proposed day is unavailable."
        title="Proposed Day"
      />
    </div>
  );
}

function DayPreview({
  title,
  day,
  emptyMessage,
  currency
}: {
  title: string;
  day: ItineraryDay | null;
  emptyMessage: string;
  currency: string;
}) {
  return (
    <section className="rounded-md border border-slate-200 bg-slate-50 p-4">
      <h3 className="text-sm font-semibold text-slate-950">{title}</h3>
      {day ? (
        <>
          <p className="mt-1 text-sm text-slate-600">{day.title}</p>
          <ul className="mt-3 space-y-3">
            {day.items.map((item, index) => (
              <li className="rounded-md border border-slate-200 bg-white p-3" key={`${item.time}-${item.name}-${index}`}>
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <p className="text-sm font-medium text-slate-950">{item.name}</p>
                    <p className="mt-1 text-xs text-slate-500">
                      {item.time} · {item.type}
                    </p>
                  </div>
                  {item.estimatedCost?.amount != null ? (
                    <span className="shrink-0 text-xs font-medium text-slate-700">
                      {formatApproxMoney(
                        item.estimatedCost.amount,
                        item.estimatedCost.currency ?? currency
                      )}
                    </span>
                  ) : null}
                </div>
                {item.note ? <p className="mt-2 text-xs leading-5 text-slate-500">{item.note}</p> : null}
              </li>
            ))}
          </ul>
        </>
      ) : (
        <p className="mt-2 text-sm text-slate-500">{emptyMessage}</p>
      )}
    </section>
  );
}
