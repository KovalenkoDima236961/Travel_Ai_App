import type { TripTemplateJSON } from "@/entities/trip-template/model";

type TemplateItineraryPreviewProps = {
  templateJson: TripTemplateJSON;
};

/**
 * Slice-local itinerary preview for the redesigned Template Detail screen. The
 * shared `features/trip-template` preview stays slate-styled for any future
 * reuse; this one renders the warm day-card layout from the design mock — a
 * card per day with the theme + stop count, then a `time · dot · name` row per
 * item. Per-item cost is intentionally dropped (the trip total lives in the
 * Estimate stat card above), matching the mock's lighter preview.
 */
export function TemplateItineraryPreview({ templateJson }: TemplateItineraryPreviewProps) {
  const days = templateJson.days ?? [];

  if (days.length === 0) {
    return (
      <div className="mt-[18px] rounded-[18px] border border-dashed border-sand-400 bg-white/60 px-6 py-8 text-[14.5px] text-cocoa-500">
        This template has no itinerary preview.
      </div>
    );
  }

  return (
    <div className="mt-[18px] flex flex-col gap-4">
      {days.map((day) => (
        <section
          key={day.dayOffset}
          className="rounded-[18px] border border-sand-300 bg-white px-6 py-5"
        >
          <div className="flex items-baseline justify-between gap-3">
            <h3 className="font-newsreader text-[20px] font-semibold text-cocoa-900">
              Day {day.dayOffset + 1} · {day.title}
            </h3>
            <span className="shrink-0 text-[13px] text-[#A08D78]">
              {day.items.length} {day.items.length === 1 ? "stop" : "stops"}
            </span>
          </div>
          <div className="mt-3.5 flex flex-col gap-2.5">
            {day.items.map((item, index) => (
              <div
                key={item.templateItemId || `${day.dayOffset}-${index}`}
                className="flex items-center gap-3.5"
              >
                <span className="w-11 shrink-0 text-[12.5px] font-bold text-cocoa-400">
                  {item.time || item.startTime || "—"}
                </span>
                <span className="h-[7px] w-[7px] shrink-0 rounded-full bg-clay" />
                <span className="min-w-0 text-[14.5px] font-medium text-cocoa-900">
                  {item.name}
                  {item.place?.name && item.place.name !== item.name ? (
                    <span className="font-normal text-cocoa-400"> · {item.place.name}</span>
                  ) : null}
                </span>
              </div>
            ))}
          </div>
        </section>
      ))}
    </div>
  );
}
