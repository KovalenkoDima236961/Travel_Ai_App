"use client";

import Link from "next/link";
import { useEffect } from "react";
import { useQuery } from "@tanstack/react-query";
import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { PageContainer } from "@/components/layout/PageContainer";
import { CollaborationInvitationsPanel } from "@/components/trips/CollaborationInvitationsPanel";
import { TripCard } from "@/components/trips/TripCard";
import { TripStatusBadge } from "@/components/trips/TripStatusBadge";
import { Card } from "@/components/ui/Card";
import { buttonStyles } from "@/components/ui/Button";
import { listSharedTrips, listTrips, tripKeys } from "@/lib/api/trips";
import { recordPwaEngagement } from "@/lib/pwa/pwa-detection";
import { formatDate } from "@/lib/utils";
import type { SharedTripSummary } from "@/types/collaboration";

export default function TripsPage() {
  return (
    <ProtectedRoute>
      <TripsPageContent />
    </ProtectedRoute>
  );
}

function TripsPageContent() {
  const tripsQuery = useQuery({
    queryKey: tripKeys.list({ limit: 20, offset: 0 }),
    queryFn: () => listTrips({ limit: 20, offset: 0 })
  });
  const sharedTripsQuery = useQuery({
    queryKey: tripKeys.shared(),
    queryFn: listSharedTrips
  });

  useEffect(() => {
    if (tripsQuery.isSuccess || sharedTripsQuery.isSuccess) {
      recordPwaEngagement();
    }
  }, [sharedTripsQuery.isSuccess, tripsQuery.isSuccess]);

  return (
    <PageContainer>
      <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <p className="text-sm font-semibold uppercase text-primary-700">
            Trips
          </p>
          <h1 className="mt-2 text-3xl font-semibold text-slate-950">Trips</h1>
        </div>
        <Link className={buttonStyles()} href="/trips/new">
          Create trip
        </Link>
      </div>

      <CollaborationInvitationsPanel />

      {tripsQuery.isPending ? (
        <div className="mt-8 rounded-lg border border-slate-200 bg-white p-6 text-sm text-slate-600">
          Loading trips...
        </div>
      ) : null}

      {tripsQuery.isError ? (
        <div className="mt-8 rounded-lg border border-red-200 bg-red-50 p-6 text-sm text-red-800">
          {tripsQuery.error instanceof Error
            ? tripsQuery.error.message
            : "Could not load trips."}
        </div>
      ) : null}

      {tripsQuery.isSuccess && tripsQuery.data.items.length === 0 ? (
        <div className="mt-8 rounded-lg border border-slate-200 bg-white p-8 text-center">
          <h2 className="text-lg font-semibold text-slate-950">No trips yet</h2>
          <p className="mt-2 text-sm text-slate-600">
            Create your first trip request to start planning.
          </p>
          <Link className={buttonStyles({ className: "mt-5" })} href="/trips/new">
            Create trip
          </Link>
        </div>
      ) : null}

      {tripsQuery.isSuccess && tripsQuery.data.items.length > 0 ? (
        <section className="mt-8">
          <h2 className="text-xl font-semibold text-slate-950">My Trips</h2>
          <div className="mt-4 grid gap-4 md:grid-cols-2 xl:grid-cols-3">
            {tripsQuery.data.items.map((trip) => (
              <TripCard key={trip.id} trip={trip} />
            ))}
          </div>
        </section>
      ) : null}

      <section className="mt-10">
        <h2 className="text-xl font-semibold text-slate-950">Shared with me</h2>

        {sharedTripsQuery.isPending ? (
          <div className="mt-4 rounded-lg border border-slate-200 bg-white p-6 text-sm text-slate-600">
            Loading shared trips...
          </div>
        ) : null}

        {sharedTripsQuery.isError ? (
          <div className="mt-4 rounded-lg border border-red-200 bg-red-50 p-6 text-sm text-red-800">
            {sharedTripsQuery.error instanceof Error
              ? sharedTripsQuery.error.message
              : "Could not load shared trips."}
          </div>
        ) : null}

        {sharedTripsQuery.isSuccess && sharedTripsQuery.data.length === 0 ? (
          <div className="mt-4 rounded-lg border border-slate-200 bg-white p-6 text-sm text-slate-600">
            No accepted shared trips yet.
          </div>
        ) : null}

        {sharedTripsQuery.isSuccess && sharedTripsQuery.data.length > 0 ? (
          <div className="mt-4 grid gap-4 md:grid-cols-2 xl:grid-cols-3">
            {sharedTripsQuery.data.map((trip) => (
              <SharedTripCard key={trip.id} trip={trip} />
            ))}
          </div>
        ) : null}
      </section>
    </PageContainer>
  );
}

function SharedTripCard({ trip }: { trip: SharedTripSummary }) {
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
