import { Card } from "@/shared/ui/card";
import { formatMoney } from "@/entities/budget/model";
import type { CostSplittingSummary } from "@/entities/cost-splitting/model";

type CostSplitSummaryCardsProps = {
  summary: CostSplittingSummary;
};

export function CostSplitSummaryCards({ summary }: CostSplitSummaryCardsProps) {
  const cards = [
    {
      label: "Estimated total",
      value: formatMoney(summary.summary.estimatedTotal, summary.currency)
    },
    {
      label: "Allocated total",
      value: formatMoney(summary.summary.allocatedTotal, summary.currency),
      tone: "ok"
    },
    {
      label: "Unassigned",
      value: formatMoney(summary.summary.unassignedTotal, summary.currency),
      tone: summary.summary.unassignedTotal > 0 ? "warning" : "ok"
    },
    {
      label: "Travelers",
      value: String(summary.summary.travelerCount)
    },
    {
      label: "Missing estimates",
      value: String(summary.summary.missingEstimateCount),
      tone: summary.summary.missingEstimateCount > 0 ? "warning" : "ok"
    },
    {
      label: "Invalid splits",
      value: String(summary.summary.invalidSplitCount),
      tone: summary.summary.invalidSplitCount > 0 ? "warning" : "ok"
    }
  ];

  return (
    <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
      {cards.map((card) => (
        <Card className="p-4 shadow-none" key={card.label}>
          <p className="text-xs font-semibold uppercase tracking-wide text-slate-500">
            {card.label}
          </p>
          <p className={valueClass(card.tone)}>{card.value}</p>
        </Card>
      ))}
    </div>
  );
}

function valueClass(tone?: string) {
  if (tone === "warning") {
    return "mt-2 text-xl font-semibold text-amber-700";
  }
  if (tone === "ok") {
    return "mt-2 text-xl font-semibold text-emerald-700";
  }
  return "mt-2 text-xl font-semibold text-slate-950";
}
