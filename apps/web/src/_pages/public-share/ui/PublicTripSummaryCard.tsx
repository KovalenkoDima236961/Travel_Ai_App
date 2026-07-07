import { Card } from "@/shared/ui/card";
import { formatDate, formatInterestLabel, formatPaceLabel } from "@/lib/utils";
import type { PublicTrip } from "@/entities/share/model";

type PublicTripSummaryCardProps = {
  trip: PublicTrip;
};

export function PublicTripSummaryCard({ trip }: PublicTripSummaryCardProps) {
  const interests = trip.interests ?? [];
  const travelers = trip.travelers ?? 0;

  return (
    <Card>
      <h2 className="text-lg font-semibold text-slate-950">Trip summary</h2>
      <dl className="mt-5 space-y-4 text-sm">
        <DetailRow label="Start date" value={trip.startDate ? formatDate(trip.startDate) : "Not set"} />
        <DetailRow label="Duration" value={`${trip.days} ${trip.days === 1 ? "day" : "days"}`} />
        <DetailRow label="Travelers" value={travelers > 0 ? String(travelers) : "Not set"} />
        <DetailRow label="Pace" value={trip.pace ? formatPaceLabel(trip.pace) : "Not set"} />
        {trip.sharedAt ? (
          <DetailRow
            label="Shared"
            value={formatDate(trip.sharedAt, {
              dateStyle: "medium",
              timeStyle: "short"
            })}
          />
        ) : null}
      </dl>
      <div className="mt-6">
        <p className="text-sm font-medium text-slate-700">Interests</p>
        <div className="mt-2 flex flex-wrap gap-2">
          {interests.length > 0 ? (
            interests.map((interest) => (
              <span
                key={interest}
                className="rounded-full border border-slate-200 bg-slate-50 px-3 py-1 text-xs font-medium text-slate-700"
              >
                {formatInterestLabel(interest)}
              </span>
            ))
          ) : (
            <span className="text-sm text-slate-500">No interests listed</span>
          )}
        </div>
      </div>
    </Card>
  );
}

type DetailRowProps = {
  label: string;
  value: string;
};

function DetailRow({ label, value }: DetailRowProps) {
  return (
    <div className="flex items-start justify-between gap-4">
      <dt className="text-slate-500">{label}</dt>
      <dd className="text-right font-medium text-slate-800">{value}</dd>
    </div>
  );
}
