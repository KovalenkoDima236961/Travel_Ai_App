"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { PageContainer } from "@/components/layout/PageContainer";
import { TripCard } from "@/components/trips/TripCard";
import { buttonStyles } from "@/components/ui/Button";
import { listTrips, tripKeys } from "@/lib/api/trips";

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

  return (
    <PageContainer>
      <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <p className="text-sm font-semibold uppercase text-primary-700">
            Trips
          </p>
          <h1 className="mt-2 text-3xl font-semibold text-slate-950">Created trips</h1>
        </div>
        <Link className={buttonStyles()} href="/trips/new">
          Create trip
        </Link>
      </div>

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
        <div className="mt-8 grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          {tripsQuery.data.items.map((trip) => (
            <TripCard key={trip.id} trip={trip} />
          ))}
        </div>
      ) : null}
    </PageContainer>
  );
}
