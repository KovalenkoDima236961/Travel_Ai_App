"use client";

import { formatBudget } from "@/lib/utils";
import type { TripTemplateJSON } from "@/types/trip-template";

type TemplateItineraryPreviewProps = {
  templateJson: TripTemplateJSON;
  currency?: string | null;
};

export function TemplateItineraryPreview({
  templateJson,
  currency
}: TemplateItineraryPreviewProps) {
  const days = templateJson.days ?? [];
  if (days.length === 0) {
    return (
      <div className="rounded-lg border border-slate-200 bg-white p-6 text-sm text-slate-600">
        This template has no itinerary preview.
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {days.map((day) => (
        <section
          className="rounded-lg border border-slate-200 bg-white p-4"
          key={day.dayOffset}
        >
          <div className="flex flex-wrap items-center justify-between gap-3">
            <h3 className="text-base font-semibold text-slate-950">
              Day {day.dayOffset + 1}: {day.title}
            </h3>
            <span className="text-xs font-medium text-slate-500">
              Offset +{day.dayOffset}
            </span>
          </div>
          <div className="mt-4 space-y-3">
            {day.items.map((item, index) => (
              <div
                className="grid gap-3 rounded-md border border-slate-100 bg-slate-50 p-3 text-sm sm:grid-cols-[6rem_minmax(0,1fr)_auto]"
                key={item.templateItemId || `${day.dayOffset}-${index}`}
              >
                <div className="font-medium text-slate-600">
                  {item.time || item.startTime || "Any time"}
                </div>
                <div className="min-w-0">
                  <p className="break-words font-semibold text-slate-900">{item.name}</p>
                  <p className="mt-1 text-xs uppercase tracking-normal text-slate-500">
                    {item.type}
                    {item.place?.name ? ` · ${item.place.name}` : ""}
                  </p>
                  {item.notes ? (
                    <p className="mt-2 break-words text-sm leading-6 text-slate-600">
                      {item.notes}
                    </p>
                  ) : null}
                </div>
                <div className="text-left font-semibold text-slate-800 sm:text-right">
                  {formatBudget(
                    item.estimatedCost?.amount,
                    item.estimatedCost?.currency || currency || "EUR"
                  )}
                </div>
              </div>
            ))}
          </div>
        </section>
      ))}
    </div>
  );
}
