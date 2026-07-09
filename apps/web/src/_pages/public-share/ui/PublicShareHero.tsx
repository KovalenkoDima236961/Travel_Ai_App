import { formatPaceLabel } from "@/lib/utils";
import type { PublicTrip } from "@/entities/share/model";
import { formatTripDateRange } from "./publicShareFormat";

type PublicShareHeroProps = {
  trip: PublicTrip;
};

export function PublicShareHero({ trip }: PublicShareHeroProps) {
  const dateRange = formatTripDateRange(trip.startDate, trip.days);
  const paceLabel = trip.pace ? `${formatPaceLabel(trip.pace)} pace` : null;
  const subtitle = [dateRange, paceLabel].filter(Boolean).join(" · ");

  return (
    <div className="relative h-[300px] overflow-hidden rounded-[24px] bg-cocoa-900">
      {/* Destination photo slot — on-brand gradient placeholder; swap for a real
          destination photo. Mirrors the landing/auth image-slot convention. */}
      <div
        className="absolute inset-0 opacity-[0.72]"
        style={{
          backgroundImage:
            "radial-gradient(120% 90% at 74% 10%, #F7CDA1 0%, transparent 46%), linear-gradient(165deg, #D98A5A 0%, #B5613C 44%, #3A2418 100%)"
        }}
      />
      <div className="pointer-events-none absolute inset-0 bg-gradient-to-b from-cocoa-900/10 to-cocoa-900/[0.72]" />
      <div className="pointer-events-none absolute inset-x-0 bottom-0 px-9 pb-8">
        <span className="inline-flex items-center gap-1.5 rounded-full bg-white/90 px-3 py-[5px] text-[11.5px] font-bold uppercase tracking-[0.06em] text-clay-deep">
          Shared itinerary · read-only
        </span>
        <h1 className="mt-3.5 font-newsreader text-[52px] font-medium leading-none tracking-[-0.02em] text-[#F6EDE2]">
          {trip.destination}
        </h1>
        {subtitle ? (
          <p className="mt-3 text-[15px] text-[#F6EDE2]/85">{subtitle}</p>
        ) : null}
      </div>
    </div>
  );
}
