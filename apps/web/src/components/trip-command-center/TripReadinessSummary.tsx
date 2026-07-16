import { readinessStatusClasses, readinessStatusLabel } from "./status-ui";
import type { ReadinessCard } from "@/types/trip-command-center";

export function TripReadinessSummary({ cards }: { cards: ReadinessCard[] }) {
  const visible = cards.filter((card) => card.status !== "unavailable");
  const ready = visible.filter((card) => card.status === "ready").length;
  const needsAttention = visible.filter(
    (card) => card.status === "needs_attention" || card.status === "blocked"
  ).length;

  return (
    <section className="rounded-[18px] border border-sand-300 bg-white p-5">
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <p className="text-[13px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">
            Trip readiness
          </p>
          <h2 className="mt-1 font-newsreader text-[25px] font-semibold text-cocoa-900">
            {ready} of {visible.length} areas ready
          </h2>
          <p className="mt-1 text-[14px] text-cocoa-500">
            {needsAttention > 0
              ? `${needsAttention} area(s) need attention before this trip is fully ready.`
              : "No urgent readiness blockers are open."}
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          {cards.map((card) => (
            <span
              key={card.id}
              className={`rounded-full border px-3 py-1.5 text-[12px] font-semibold ${readinessStatusClasses(
                card.status
              )}`}
            >
              {card.title}: {readinessStatusLabel[card.status]}
            </span>
          ))}
        </div>
      </div>
    </section>
  );
}
