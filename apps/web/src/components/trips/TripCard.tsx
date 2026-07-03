import Link from "next/link";
import { Card } from "@/components/ui/Card";
import { TripStatusBadge } from "@/components/trips/TripStatusBadge";
import type { Trip } from "@/types/trip";
import {
  formatBudget,
  formatDate,
  formatInterestLabel,
  formatPaceLabel
} from "@/lib/utils";

type TripCardProps = {
  trip: Trip;
};

export function TripCard({ trip }: TripCardProps) {
  return (
    <Link className="block h-full" href={`/trips/${trip.id}`}>
      <Card className="flex h-full flex-col gap-5 transition hover:-translate-y-0.5 hover:border-primary-100 hover:shadow-lg">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <h2 className="truncate text-lg font-semibold text-slate-950">{trip.destination}</h2>
            <p className="mt-1 text-sm text-slate-500">Created {formatDate(trip.createdAt)}</p>
          </div>
          <div className="flex shrink-0 flex-col items-end gap-2">
            <TripStatusBadge status={trip.status} />
            {trip.workspaceId ? (
              <span className="rounded-full bg-primary-50 px-2.5 py-1 text-xs font-semibold text-primary-700">
                Workspace
              </span>
            ) : null}
          </div>
        </div>

        <div className="grid grid-cols-2 gap-3 text-sm">
          <TripFact label="Days" value={`${trip.days}`} />
          <TripFact label="Travelers" value={`${trip.travelers}`} />
          <TripFact label="Budget" value={formatBudget(trip.budgetAmount, trip.budgetCurrency)} />
          <TripFact label="Pace" value={formatPaceLabel(trip.pace)} />
        </div>

        <div className="mt-auto flex flex-wrap gap-2">
          {trip.interests.length > 0 ? (
            trip.interests.slice(0, 4).map((interest) => (
              <span
                key={interest}
                className="rounded-full bg-slate-100 px-2.5 py-1 text-xs font-medium text-slate-700"
              >
                {formatInterestLabel(interest)}
              </span>
            ))
          ) : (
            <span className="text-sm text-slate-500">No interests selected</span>
          )}
          {trip.interests.length > 4 ? (
            <span className="rounded-full bg-slate-100 px-2.5 py-1 text-xs font-medium text-slate-700">
              +{trip.interests.length - 4}
            </span>
          ) : null}
        </div>
      </Card>
    </Link>
  );
}

type TripFactProps = {
  label: string;
  value: string;
};

function TripFact({ label, value }: TripFactProps) {
  return (
    <div>
      <p className="text-xs font-medium text-slate-500">{label}</p>
      <p className="mt-1 truncate font-semibold text-slate-800">{value}</p>
    </div>
  );
}
