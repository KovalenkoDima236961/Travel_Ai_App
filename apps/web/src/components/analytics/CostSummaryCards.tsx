import { Card } from "@/components/ui/Card";
import { cn } from "@/lib/utils";

export type CostSummaryCard = {
  label: string;
  value: string;
  detail?: string;
  tone?: "default" | "ok" | "warning" | "danger";
};

type CostSummaryCardsProps = {
  cards: CostSummaryCard[];
};

export function CostSummaryCards({ cards }: CostSummaryCardsProps) {
  return (
    <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
      {cards.map((card) => (
        <Card className="p-4" key={card.label}>
          <p className="text-xs font-semibold uppercase text-slate-500">{card.label}</p>
          <p
            className={cn(
              "mt-2 break-words text-2xl font-semibold text-slate-950",
              card.tone === "ok" && "text-emerald-700",
              card.tone === "warning" && "text-amber-700",
              card.tone === "danger" && "text-red-700"
            )}
          >
            {card.value}
          </p>
          {card.detail ? <p className="mt-2 text-sm text-slate-600">{card.detail}</p> : null}
        </Card>
      ))}
    </div>
  );
}
