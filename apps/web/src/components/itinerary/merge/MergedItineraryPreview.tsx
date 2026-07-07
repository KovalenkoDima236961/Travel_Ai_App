"use client";

import type { Itinerary } from "@/entities/trip/model";

type MergedItineraryPreviewProps = {
  itinerary: Itinerary;
  affectedDayNumbers: number[];
};

export function MergedItineraryPreview({
  itinerary,
  affectedDayNumbers
}: MergedItineraryPreviewProps) {
  const days = (itinerary.days ?? []).filter((day, index) =>
    affectedDayNumbers.includes(day.day || index + 1)
  );

  if (days.length === 0) {
    return null;
  }

  return (
    <section className="rounded-lg border border-slate-200 bg-slate-50 p-4">
      <div className="flex items-center justify-between gap-3">
        <h3 className="text-sm font-semibold text-slate-950">Merged preview</h3>
        <span className="rounded-full border border-slate-300 bg-white px-2 py-1 text-xs font-medium text-slate-600">
          Preview only
        </span>
      </div>
      <div className="mt-3 space-y-3">
        {days.map((day, index) => {
          const dayNumber = day.day || index + 1;
          return (
            <div className="rounded-lg border border-slate-200 bg-white p-3" key={dayNumber}>
              <p className="text-sm font-medium text-slate-950">
                Day {dayNumber}: {day.title}
              </p>
              <ul className="mt-2 space-y-1 text-sm text-slate-600">
                {(day.items ?? []).map((item, itemIndex) => (
                  <li key={`${dayNumber}-${itemIndex}-${item.time}-${item.name}`}>
                    {item.time} · {item.name}
                  </li>
                ))}
              </ul>
            </div>
          );
        })}
      </div>
    </section>
  );
}
