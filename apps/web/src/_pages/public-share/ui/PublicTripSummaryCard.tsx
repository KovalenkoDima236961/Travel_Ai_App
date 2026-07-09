import { formatMoney } from "@/entities/budget/model";
import { formatInterestLabel } from "@/lib/utils";
import type { PublicTrip } from "@/entities/share/model";
import { estimateItineraryTotal } from "./publicShareFormat";

type PublicTripSummaryCardProps = {
  trip: PublicTrip;
};

export function PublicTripSummaryCard({ trip }: PublicTripSummaryCardProps) {
  const interests = trip.interests ?? [];
  const travelers = trip.travelers ?? 0;
  const currency = trip.itinerary?.currency ?? "EUR";
  const estimatedTotal = estimateItineraryTotal(trip.itinerary);

  return (
    <div className="rounded-[18px] border border-sand-300 bg-white px-6 py-[22px]">
      <h2 className="text-[13px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">
        Trip summary
      </h2>
      <dl className="mt-4 flex flex-col gap-3 text-[14px]">
        <SummaryRow label="Destination" value={trip.destination} />
        <SummaryRow label="Duration" value={`${trip.days} ${trip.days === 1 ? "day" : "days"}`} />
        <SummaryRow label="Travelers" value={travelers > 0 ? String(travelers) : "Not set"} />
        {estimatedTotal != null ? (
          <SummaryRow label="Estimated" value={formatMoney(estimatedTotal, currency)} />
        ) : null}
      </dl>
      {interests.length > 0 ? (
        <div className="mt-[18px] flex flex-wrap gap-[7px] border-t border-[#F1E8DC] pt-4">
          {interests.map((interest) => (
            <span
              key={interest}
              className="rounded-full bg-[#F4EDE4] px-3 py-[5px] text-[12px] font-medium text-cocoa-500"
            >
              {formatInterestLabel(interest)}
            </span>
          ))}
        </div>
      ) : null}
    </div>
  );
}

type SummaryRowProps = {
  label: string;
  value: string;
};

function SummaryRow({ label, value }: SummaryRowProps) {
  return (
    <div className="flex justify-between gap-3">
      <dt className="text-cocoa-400">{label}</dt>
      <dd className="m-0 text-right font-semibold text-cocoa-900">{value}</dd>
    </div>
  );
}
