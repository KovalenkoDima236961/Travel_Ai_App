import Link from "next/link";
import { TripStatusBadge } from "@/components/trips/TripStatusBadge";
import { Card } from "@/shared/ui/card";
import { formatDate } from "@/lib/utils";
import type { SharedTripSummary } from "@/entities/collaboration/model";

export function SharedTripCard({ trip }: { trip: SharedTripSummary }) {
  return (
    <Link className="block h-full" href={`/trips/${trip.id}`}>
      <Card className="flex h-full flex-col gap-5 transition hover:-translate-y-0.5 hover:border-primary-100 hover:shadow-lg">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <h3 className="truncate text-lg font-semibold text-slate-950">{trip.destination}</h3>
            <p className="mt-1 text-sm text-slate-500">
              {trip.updatedAt ? `Updated ${formatDate(trip.updatedAt)}` : "Shared trip"}
            </p>
          </div>
          <TripStatusBadge status={trip.status} />
        </div>
        <div className="grid grid-cols-2 gap-3 text-sm">
          <TripFact label="Days" value={`${trip.days}`} />
          <TripFact label="Start" value={trip.startDate ? formatDate(trip.startDate) : "Not set"} />
          <TripFact label="Role" value={trip.role === "editor" ? "Editor" : "Viewer"} />
          <TripFact label="Access" value="Private" />
        </div>
      </Card>
    </Link>
  );
}

function TripFact({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-xs font-medium text-slate-500">{label}</p>
      <p className="mt-1 truncate font-semibold text-slate-800">{value}</p>
    </div>
  );
}
