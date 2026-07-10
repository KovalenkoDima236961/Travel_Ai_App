"use client";

import type { ReactNode } from "react";
import { formatMoney } from "@/entities/budget/model";
import { transportModeLabel } from "@/components/routes/route-options";
import type { RouteAlternative, RouteAlternativeComparisonSummary } from "@/types/route-alternatives";

type RouteAlternativeComparisonTableProps = {
  alternatives: RouteAlternative[];
  summary?: RouteAlternativeComparisonSummary;
  selectedId?: string;
  onSelect?: (alternative: RouteAlternative) => void;
};

export function RouteAlternativeComparisonTable({
  alternatives,
  summary,
  selectedId,
  onSelect
}: RouteAlternativeComparisonTableProps) {
  if (alternatives.length === 0) {
    return null;
  }

  return (
    <div className="overflow-x-auto rounded-[16px] border border-sand-300 bg-white">
      <table className="min-w-[980px] w-full border-collapse text-left text-[13px]">
        <thead className="bg-sand-50 text-[11px] uppercase tracking-[0.08em] text-cocoa-500">
          <tr>
            <Th>Route</Th>
            <Th>Stops</Th>
            <Th>Transport</Th>
            <Th>Estimated budget</Th>
            <Th>Transfer time</Th>
            <Th>Difficulty</Th>
            <Th>Overall</Th>
            <Th>Budget</Th>
            <Th>Relaxation</Th>
            <Th>Nature</Th>
            <Th>Culture</Th>
            <Th>Policy</Th>
            <Th>Warnings</Th>
          </tr>
        </thead>
        <tbody className="divide-y divide-sand-200">
          {alternatives.map((alternative) => (
            <tr
              key={alternative.id}
              className={selectedId === alternative.id ? "bg-clay-tint/30" : "bg-white"}
            >
              <Td>
                <button
                  type="button"
                  onClick={() => onSelect?.(alternative)}
                  className="text-left font-semibold text-cocoa-900 hover:text-clay"
                >
                  {alternative.title}
                </button>
                <BadgeList alternative={alternative} summary={summary} />
              </Td>
              <Td>{alternative.route.stops.map((stop) => stop.city || stop.destination).join(" → ")}</Td>
              <Td>
                {Array.from(new Set((alternative.route.legs ?? []).map((leg) => leg.mode)))
                  .map(transportModeLabel)
                  .join(", ") || "Flexible"}
              </Td>
              <Td>{formatMoney(alternative.estimatedBudget?.amount, alternative.estimatedBudget?.currency)}</Td>
              <Td>{formatDuration(alternative.estimatedTransferMinutes)}</Td>
              <Td className="capitalize">{alternative.difficulty}</Td>
              <Score value={alternative.scores.overallFit} />
              <Score value={alternative.scores.budgetFit} />
              <Score value={alternative.scores.relaxation} />
              <Score value={alternative.scores.nature} />
              <Score value={alternative.scores.culture} />
              <Score value={alternative.scores.policyCompliance} />
              <Td>{alternative.warnings.length}</Td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function Th({ children }: { children: ReactNode }) {
	return <th className="px-3 py-3 font-bold">{children}</th>;
}

function Td({ children, className = "" }: { children: ReactNode; className?: string }) {
	return <td className={`px-3 py-3 align-top text-cocoa-600 ${className}`}>{children}</td>;
}

function Score({ value }: { value: number }) {
  return (
    <Td>
      <span className="inline-flex min-w-10 justify-center rounded-full bg-sand-100 px-2 py-1 font-semibold text-cocoa-800">
        {Math.round(value)}
      </span>
    </Td>
  );
}

function BadgeList({
  alternative,
  summary
}: {
  alternative: RouteAlternative;
  summary?: RouteAlternativeComparisonSummary;
}) {
  const badges = [
    summary?.bestOverallAlternativeId === alternative.id ? "Best overall" : null,
    summary?.cheapestAlternativeId === alternative.id ? "Cheapest" : null,
    summary?.mostRelaxedAlternativeId === alternative.id ? "Most relaxed" : null,
    summary?.bestNatureAlternativeId === alternative.id ? "Best nature" : null
  ].filter(Boolean);
  if (badges.length === 0) {
    return null;
  }
  return (
    <div className="mt-2 flex flex-wrap gap-1">
      {badges.map((badge) => (
        <span key={badge} className="rounded-full bg-clay-tint px-2 py-0.5 text-[11px] font-semibold text-clay-deep">
          {badge}
        </span>
      ))}
    </div>
  );
}

function formatDuration(minutes: number | null | undefined) {
  if (!minutes || minutes <= 0) {
    return "—";
  }
  const hours = Math.floor(minutes / 60);
  const remainder = minutes % 60;
  return hours === 0 ? `${minutes} min` : remainder === 0 ? `${hours} hr` : `${hours} hr ${remainder} min`;
}
