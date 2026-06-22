import type { Itinerary } from "@/types/trip";
import { formatDate, formatInterestLabel, formatMoney, formatPaceLabel } from "@/lib/utils";

type ItineraryViewProps = {
  itinerary: Itinerary;
  currency?: string;
};

export function ItineraryView({ itinerary, currency = "EUR" }: ItineraryViewProps) {
  if (!itinerary.days || itinerary.days.length === 0) {
    return (
      <div className="rounded-lg border border-slate-200 bg-white p-6 text-sm text-slate-600">
        No itinerary days were returned.
      </div>
    );
  }

  const displayCurrency = itinerary.currency || currency;

  return (
    <div className="space-y-5">
      <div className="rounded-lg border border-slate-200 bg-white p-6">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <h2 className="text-xl font-semibold text-slate-950">Generated itinerary</h2>
            {itinerary.summary ? (
              <p className="mt-2 text-sm leading-6 text-slate-600">{itinerary.summary}</p>
            ) : null}
          </div>
          {itinerary.totalBudget != null ? (
            <div className="rounded-md border border-slate-200 bg-slate-50 px-3 py-2 text-sm">
              <p className="text-xs font-medium text-slate-500">Budget</p>
              <p className="font-semibold text-slate-900">
                {formatMoney(itinerary.totalBudget, displayCurrency)}
              </p>
            </div>
          ) : null}
        </div>

        <div className="mt-5 flex flex-wrap gap-x-5 gap-y-2 text-sm text-slate-600">
          {itinerary.destination ? <span>{itinerary.destination}</span> : null}
          {itinerary.travelers ? (
            <span>
              {itinerary.travelers} {itinerary.travelers === 1 ? "traveler" : "travelers"}
            </span>
          ) : null}
          {itinerary.pace ? <span>{formatPaceLabel(itinerary.pace)} pace</span> : null}
          {itinerary.generatedAt ? <span>Generated {formatDate(itinerary.generatedAt)}</span> : null}
        </div>
      </div>

      {itinerary.days.map((day) => (
        <section key={day.day} className="rounded-lg border border-slate-200 bg-white p-6">
          <h3 className="text-lg font-semibold text-slate-950">
            Day {day.day} — {day.title}
          </h3>
          <ol className="mt-5 divide-y divide-slate-100">
            {day.items.map((item, index) => (
              <li
                key={`${day.day}-${item.time}-${item.name}-${index}`}
                className="grid gap-3 py-4 first:pt-0 last:pb-0 sm:grid-cols-[6.5rem_minmax(0,1fr)_auto]"
              >
                <div className="text-sm font-semibold text-slate-900">{item.time}</div>
                <div className="min-w-0">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="rounded-full bg-primary-50 px-2.5 py-1 text-xs font-medium text-primary-700">
                      {formatInterestLabel(item.type)}
                    </span>
                    <p className="font-semibold text-slate-950">{item.name}</p>
                  </div>
                  {item.note ? (
                    <p className="mt-2 text-sm leading-6 text-slate-600">{item.note}</p>
                  ) : null}
                </div>
                {item.estimatedCost != null ? (
                  <div className="text-sm font-semibold text-slate-900">
                    {formatMoney(item.estimatedCost, displayCurrency)}
                  </div>
                ) : null}
              </li>
            ))}
          </ol>
        </section>
      ))}
    </div>
  );
}
