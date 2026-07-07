"use client";

import type { ItineraryChange } from "@/entities/itinerary/model/diff-merge/types";

type ChangeSummaryListProps = {
  title: string;
  changes: ItineraryChange[];
  emptyLabel?: string;
};

export function ChangeSummaryList({
  title,
  changes,
  emptyLabel = "No changes detected."
}: ChangeSummaryListProps) {
  return (
    <section className="rounded-lg border border-slate-200 bg-white p-4">
      <h3 className="text-sm font-semibold text-slate-950">{title}</h3>
      {changes.length > 0 ? (
        <ul className="mt-3 space-y-2 text-sm text-slate-700">
          {changes.map((change) => (
            <li className="flex gap-2" key={change.id}>
              <span className="mt-2 h-1.5 w-1.5 shrink-0 rounded-full bg-primary-600" />
              <span>{change.summary}</span>
            </li>
          ))}
        </ul>
      ) : (
        <p className="mt-3 text-sm text-slate-500">{emptyLabel}</p>
      )}
    </section>
  );
}
