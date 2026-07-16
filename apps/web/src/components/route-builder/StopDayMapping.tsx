import { useTranslations } from "next-intl";
import type { TripRoute } from "@/entities/route/model";
import type { Itinerary } from "@/entities/trip/model";
import { mapItineraryToStops } from "@/lib/route-builder/route-validation";
import { StopItinerarySummary } from "./StopItinerarySummary";

export function StopDayMapping({ route, itinerary }: { route: TripRoute; itinerary?: Itinerary | null }) {
  const t = useTranslations("route");
  const entries = mapItineraryToStops(route, itinerary);
  if (!itinerary?.days.length) {
    return null;
  }
  return (
    <section className="rounded-[18px] border border-sand-300 bg-white p-5">
      <p className="text-[12px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">{t("stopDays")}</p>
      <h3 className="mt-1 font-newsreader text-[21px] font-semibold text-cocoa-900">{t("itineraryConnection")}</h3>
      <div className="mt-4 grid gap-2 sm:grid-cols-2">
        {entries.map((entry) => <StopItinerarySummary entry={entry} key={entry.stop.id} />)}
      </div>
    </section>
  );
}
