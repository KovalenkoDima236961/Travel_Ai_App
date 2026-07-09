import { cn } from "@/lib/utils";

export type SummaryCard = {
  label: string;
  value: string;
  detail?: string;
  tone?: "default" | "ok" | "warning" | "danger";
};

type SummaryCardsProps = {
  cards: SummaryCard[];
};

/**
 * Slice-local restyle of the shared CostSummaryCards. The shared component is
 * still rendered by the un-redesigned workspace analytics/budget screens, so the
 * warm palette lives here instead of leaking into it.
 */
export function SummaryCards({ cards }: SummaryCardsProps) {
  return (
    <div className="grid grid-cols-1 gap-5 sm:grid-cols-2 xl:grid-cols-4">
      {cards.map((card) => {
        const warning = card.tone === "warning";
        return (
          <div
            className={cn(
              "rounded-[18px] border px-6 py-[22px]",
              warning ? "border-[#EFD9B8] bg-[#FFFDF7]" : "border-sand-300 bg-white"
            )}
            key={card.label}
          >
            <p
              className={cn(
                "text-[12.5px] font-semibold uppercase tracking-[0.06em]",
                warning ? "text-[#96682A]" : "text-cocoa-400"
              )}
            >
              {card.label}
            </p>
            <p
              className={cn(
                "mt-3 break-words font-newsreader text-[32px] font-semibold leading-none",
                card.tone === "ok" && "text-[#2F7A57]",
                card.tone === "warning" && "text-[#96682A]",
                card.tone === "danger" && "text-[#C0392B]",
                (!card.tone || card.tone === "default") && "text-cocoa-900"
              )}
            >
              {card.value}
            </p>
            {card.detail ? (
              <p className="mt-1.5 text-[13px] text-cocoa-400">{card.detail}</p>
            ) : null}
          </div>
        );
      })}
    </div>
  );
}
