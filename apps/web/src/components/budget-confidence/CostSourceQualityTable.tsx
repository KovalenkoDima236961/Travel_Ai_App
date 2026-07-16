import { formatMoney } from "@/entities/budget/model";
import type { BudgetConfidenceSourceQuality } from "@/types/budget-confidence";

export function CostSourceQualityTable({
  sources
}: {
  sources: BudgetConfidenceSourceQuality[];
}) {
  if (sources.length === 0) {
    return null;
  }

  return (
    <div>
      <p className="text-xs font-semibold uppercase tracking-wide text-slate-500">Source quality</p>
      <div className="mt-2 overflow-hidden rounded-md border border-slate-200">
        <table className="w-full text-left text-xs">
          <thead className="bg-slate-50 text-slate-500">
            <tr>
              <th className="px-3 py-2 font-semibold">Source</th>
              <th className="px-3 py-2 text-right font-semibold">Items</th>
              <th className="px-3 py-2 text-right font-semibold">Quality</th>
              <th className="px-3 py-2 text-right font-semibold">Total</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100 text-slate-700">
            {sources.slice(0, 6).map((source) => (
              <tr key={source.source}>
                <td className="px-3 py-2">
                  <span className="font-medium text-slate-900">{formatLabel(source.source)}</span>
                  {source.reason ? (
                    <span className="block text-[11px] text-slate-500">{source.reason}</span>
                  ) : null}
                </td>
                <td className="px-3 py-2 text-right">{source.itemCount}</td>
                <td className="px-3 py-2 text-right">{source.qualityScore}%</td>
                <td className="px-3 py-2 text-right">
                  {formatMoney(source.totalAmount.amount, source.totalAmount.currency)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function formatLabel(value: string) {
  return value.replaceAll("_", " ");
}
