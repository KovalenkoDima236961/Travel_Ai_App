import { formatMoney, getCostAmount, getCostCurrency } from "@/entities/budget/model";
import type { Itinerary, ItineraryItem } from "@/entities/trip/model";

type PublicShareItineraryProps = {
  itinerary: Itinerary;
};

/**
 * Read-only itinerary timeline for the public share screen, styled to the mock's
 * time-gutter + card layout. This is a deliberately minimal, viewer-facing fork
 * of trip-detail's ItineraryTimeline — no regenerate/split/comment controls and
 * no availability search — since a shared link has no editing affordances.
 */
export function PublicShareItinerary({ itinerary }: PublicShareItineraryProps) {
  if (!itinerary.days || itinerary.days.length === 0) {
    return (
      <div className="rounded-[18px] border border-sand-300 bg-white p-6 text-[14px] text-cocoa-500">
        This shared trip does not have an itinerary yet.
      </div>
    );
  }

  const currency = itinerary.currency ?? "EUR";

  return (
    <div className="flex flex-col gap-5">
      {itinerary.days.map((day, dayIndex) => {
        const dayNumber = day.day || dayIndex + 1;
        return (
          <section key={dayNumber}>
            <h2 className="font-newsreader text-[26px] font-semibold text-cocoa-900">
              Day {dayNumber} <span className="font-normal text-[#A08D78]">·</span>{" "}
              {day.title ? <em className="font-medium not-italic">{day.title}</em> : null}
            </h2>
            <div className="mt-4 flex flex-col gap-3">
              {day.items.map((item, index) => (
                <TimelineItem
                  key={`${dayNumber}-${item.time}-${item.name}-${index}`}
                  item={item}
                  currency={currency}
                />
              ))}
            </div>
          </section>
        );
      })}
    </div>
  );
}

function TimelineItem({ item, currency }: { item: ItineraryItem; currency: string }) {
  const amount = getCostAmount(item.estimatedCost);
  const itemCurrency = getCostCurrency(item.estimatedCost) ?? currency;

  return (
    <div className="grid grid-cols-[48px_minmax(0,1fr)] gap-3.5 sm:grid-cols-[56px_minmax(0,1fr)]">
      <span className="pt-[18px] text-right text-[13px] font-bold text-cocoa-900">{item.time}</span>
      <div className="rounded-[16px] border border-sand-300 bg-white px-5 py-4">
        <div className="flex items-baseline justify-between gap-3">
          <p className="text-[15.5px] font-semibold text-cocoa-900">{item.name}</p>
          {amount != null ? (
            <span className="shrink-0 font-newsreader text-[17px] font-semibold text-cocoa-900">
              {formatMoney(amount, itemCurrency)}
            </span>
          ) : null}
        </div>
        {item.note ? (
          <p className="mt-1.5 text-[13.5px] leading-[1.55] text-cocoa-500">{item.note}</p>
        ) : null}
      </div>
    </div>
  );
}
