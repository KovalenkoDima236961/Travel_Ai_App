import { formatMoney } from "@/entities/budget/model";
import type { BudgetConfidencePlannedVsActual } from "@/types/budget-confidence";

export function PlannedVsActualAccuracyCard({
  plannedVsActual
}: {
  plannedVsActual: BudgetConfidencePlannedVsActual;
}) {
  const categories = plannedVsActual.categories.filter(
    (category) => category.actual.amount > 0 || category.estimated.amount > 0
  );

  return (
    <div>
      <p className="text-xs font-semibold uppercase tracking-wide text-slate-500">
        Planned vs actual
      </p>
      <div className="mt-2 rounded-md border border-slate-200 bg-slate-50 px-3 py-2">
        <div className="flex flex-wrap items-center justify-between gap-2 text-sm">
          <span className="text-slate-600">Difference</span>
          <span className="font-semibold text-slate-950">
            {formatMoney(
              plannedVsActual.overallDifference.amount,
              plannedVsActual.overallDifference.currency
            )}
            {plannedVsActual.overallDifferencePercent != null
              ? ` (${Math.round(plannedVsActual.overallDifferencePercent)}%)`
              : ""}
          </span>
        </div>
        {categories.length > 0 ? (
          <ul className="mt-2 space-y-1 text-xs">
            {categories.slice(0, 4).map((category) => (
              <li
                className="flex items-center justify-between gap-3"
                key={`${category.category}-${category.status}`}
              >
                <span className="capitalize text-slate-600">
                  {category.category.replaceAll("_", " ")}
                </span>
                <span className="text-right text-slate-900">
                  {formatMoney(category.actual.amount, category.actual.currency)} actual
                  <span className="ml-1 text-slate-400">
                    / {formatMoney(category.estimated.amount, category.estimated.currency)} planned
                  </span>
                </span>
              </li>
            ))}
          </ul>
        ) : (
          <p className="mt-2 text-xs text-slate-500">No actual expenses recorded yet.</p>
        )}
      </div>
    </div>
  );
}
