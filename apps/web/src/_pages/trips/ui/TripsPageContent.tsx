"use client";

import Link from "next/link";
import { useEffect, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { PageContainer } from "@/components/layout/PageContainer";
import { CollaborationInvitationsPanel } from "@/features/trip-sharing";
import { TripCard } from "@/components/trips/TripCard";
import { buttonStyles } from "@/shared/ui/button";
import { listSharedTrips, listTrips, tripKeys } from "@/lib/api/trips";
import { recordPwaEngagement } from "@/lib/pwa/pwa-detection";
import { useWorkspaces } from "@/components/workspaces/WorkspaceProvider";
import { SharedTripCard } from "./SharedTripCard";

export function TripsPageContent() {
  const { currentScope, currentWorkspace, currentWorkspaceId } = useWorkspaces();
  const listParams = useMemo(
    () => ({
      limit: 20,
      offset: 0,
      scope: currentScope,
      workspaceId: currentScope === "workspace" ? currentWorkspaceId : null
    }),
    [currentScope, currentWorkspaceId]
  );
  const scopeLabel =
    currentScope === "workspace"
      ? currentWorkspace?.name ?? "Workspace"
      : currentScope === "personal"
        ? "Personal"
        : "All trips";
  const showSharedTrips = currentScope !== "workspace";

  const tripsQuery = useQuery({
    queryKey: tripKeys.list(listParams),
    queryFn: () => listTrips(listParams),
    enabled: currentScope !== "workspace" || Boolean(currentWorkspaceId)
  });
  const sharedTripsQuery = useQuery({
    queryKey: tripKeys.shared(),
    queryFn: listSharedTrips,
    enabled: showSharedTrips
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
          <h1 className="mt-2 text-3xl font-semibold text-slate-950">{scopeLabel}</h1>
        </div>
        <Link className={buttonStyles()} href="/trips/new">
          Create trip
        </Link>
      </div>

      {showSharedTrips ? <CollaborationInvitationsPanel /> : null}

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
            {currentScope === "workspace"
              ? "Create the first trip for this workspace."
              : "Create your first trip request to start planning."}
          </p>
          <Link className={buttonStyles({ className: "mt-5" })} href="/trips/new">
            Create trip
          </Link>
        </div>
      ) : null}

      {tripsQuery.isSuccess && tripsQuery.data.items.length > 0 ? (
        <section className="mt-8">
          <h2 className="text-xl font-semibold text-slate-950">
            {currentScope === "workspace" ? "Workspace trips" : "My Trips"}
          </h2>
          <div className="mt-4 grid gap-4 md:grid-cols-2 xl:grid-cols-3">
            {tripsQuery.data.items.map((trip) => (
              <TripCard key={trip.id} trip={trip} />
            ))}
          </div>
        </section>
      ) : null}

      {showSharedTrips ? (
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
      ) : null}
    </PageContainer>
  );
}
