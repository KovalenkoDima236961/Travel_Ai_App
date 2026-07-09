"use client";

import { formatApproxMoney } from "@/entities/budget/model";
import type { RepairDiff } from "@/entities/trip-repair/model";
import type { Itinerary, ItineraryDay } from "@/entities/trip/model";

type ItineraryRepairDiffProps = {
  currentItinerary: Itinerary;
  repairedItinerary: Itinerary;
  diff: RepairDiff;
};

export function ItineraryRepairDiff({
  currentItinerary,
  repairedItinerary,
  diff
}: ItineraryRepairDiffProps) {
  const changedDayNumbers = getChangedDayNumbers(diff);
  const currentDays = filterDays(currentItinerary.days, changedDayNumbers);
  const repairedDays = filterDays(repairedItinerary.days, changedDayNumbers);

  return (
    <div className="space-y-4">
      <DiffSummary diff={diff} />
      <div className="grid gap-4 lg:grid-cols-2">
        <ItineraryColumn days={currentDays} emptyMessage="Current itinerary unavailable." title="Current" />
        <ItineraryColumn days={repairedDays} emptyMessage="Repaired itinerary unavailable." title="Repaired" />
      </div>
    </div>
  );
}

function DiffSummary({ diff }: { diff: RepairDiff }) {
  const groups = [
    { label: "Added", values: diff.itemsAdded },
    { label: "Removed", values: diff.itemsRemoved },
    { label: "Modified", values: diff.itemsModified },
    { label: "Moved", values: diff.itemsMoved }
  ].filter((group) => group.values.length > 0);

  if (groups.length === 0 && !diff.warnings?.length) {
    return null;
  }

  return (
    <div className="rounded-md border border-slate-200 bg-slate-50 p-3">
      <div className="grid gap-3 sm:grid-cols-2">
        {groups.map((group) => (
          <div key={group.label}>
            <p className="text-xs font-semibold uppercase tracking-wide text-slate-500">
              {group.label}
            </p>
            <ul className="mt-2 space-y-1 text-sm text-slate-700">
              {group.values.slice(0, 4).map((change, index) => (
                <li key={`${group.label}-${index}`}>{changeLabel(change)}</li>
              ))}
            </ul>
          </div>
        ))}
      </div>
      {diff.warnings?.length ? (
        <p className="mt-3 text-xs text-slate-500">{diff.warnings[0]}</p>
      ) : null}
    </div>
  );
}

function ItineraryColumn({
  days,
  title,
  emptyMessage
}: {
  days: ItineraryDay[];
  title: string;
  emptyMessage: string;
}) {
  return (
    <section className="rounded-md border border-slate-200 bg-slate-50 p-4">
      <h3 className="text-sm font-semibold text-slate-950">{title}</h3>
      {days.length > 0 ? (
        <div className="mt-3 space-y-4">
          {days.map((day) => (
            <div key={day.day}>
              <p className="text-sm font-medium text-slate-900">
                Day {day.day}: {day.title}
              </p>
              <ul className="mt-2 space-y-2">
                {day.items.map((item, index) => (
                  <li
                    className="rounded-md border border-slate-200 bg-white p-3"
                    key={`${day.day}-${item.time}-${item.name}-${index}`}
                  >
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
                            item.estimatedCost.currency ?? "EUR"
                          )}
                        </span>
                      ) : null}
                    </div>
                    {item.note ? (
                      <p className="mt-2 text-xs leading-5 text-slate-500">{item.note}</p>
                    ) : null}
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>
      ) : (
        <p className="mt-2 text-sm text-slate-500">{emptyMessage}</p>
      )}
    </section>
  );
}

function getChangedDayNumbers(diff: RepairDiff) {
  const dayNumbers = new Set<number>();
  [
    ...diff.daysChanged,
    ...diff.itemsAdded,
    ...diff.itemsRemoved,
    ...diff.itemsModified,
    ...diff.itemsMoved
  ].forEach((change) => {
    if (change.dayNumber != null) {
      dayNumbers.add(change.dayNumber);
    }
  });
  return dayNumbers;
}

function filterDays(days: ItineraryDay[], changedDayNumbers: Set<number>) {
  if (changedDayNumbers.size === 0) {
    return days.slice(0, 2);
  }
  return days.filter((day) => changedDayNumbers.has(day.day));
}

function changeLabel(change: { dayNumber?: number | null; itemIndex?: number | null; type: string; reason?: string | null; after?: Record<string, unknown> | null; before?: Record<string, unknown> | null }) {
  const name = stringFromRecord(change.after, "name") ?? stringFromRecord(change.before, "name");
  const day = change.dayNumber != null ? `Day ${change.dayNumber}` : "Trip";
  const item = change.itemIndex != null ? `, item ${change.itemIndex + 1}` : "";
  return `${day}${item}: ${name ?? change.type.replaceAll("_", " ")}`;
}

function stringFromRecord(record: Record<string, unknown> | null | undefined, key: string) {
  const value = record?.[key];
  return typeof value === "string" && value.trim() ? value.trim() : null;
}
