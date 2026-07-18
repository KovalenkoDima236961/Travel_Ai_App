"use client";

import Link from "next/link";
import { useEffect, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { cn } from "@/shared/lib/cn";
import { CollaborationInvitationsPanel } from "@/features/trip-sharing";
import { useAuth } from "@/components/auth/AuthProvider";
import {
  FirstRunDashboard,
  HelpfulTripsEmptyState
} from "@/components/onboarding/FirstRunDashboard";
import { useFirstRunStatus } from "@/hooks/useFirstRunStatus";
import { useOnboardingState } from "@/hooks/useOnboardingState";
import { listSharedTrips, listTrips, tripKeys } from "@/lib/api/trips";
import { recordPwaEngagement } from "@/lib/pwa/pwa-detection";
import { useWorkspaces } from "@/components/workspaces/WorkspaceProvider";
import { instrumentSans, newsreader } from "./fonts";
import { ScopeSegmentedControl } from "./ScopeSegmentedControl";
import { ShareNodesIcon } from "./icons";
import { SharedTripCard } from "./SharedTripCard";
import { TripCard } from "./TripCard";
import { TripsHeader } from "./TripsHeader";

export function TripsPageContent() {
  const { user } = useAuth();
  const onboarding = useOnboardingState(user?.id);
  const { currentScope, currentWorkspace, currentWorkspaceId, workspaces } = useWorkspaces();
  const listParams = useMemo(
    () => ({
      limit: 20,
      offset: 0,
      scope: currentScope,
      workspaceId: currentScope === "workspace" ? currentWorkspaceId : null
    }),
    [currentScope, currentWorkspaceId]
  );
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

  const workspaceNames = useMemo(() => {
    const map = new Map<string, string>();
    for (const workspace of workspaces) {
      map.set(workspace.id, workspace.name);
    }
    return map;
  }, [workspaces]);

  const heading =
    currentScope === "workspace" ? currentWorkspace?.name ?? "Workspace trips" : "Your trips";
  const trips = tripsQuery.data?.items ?? [];
  const firstRun = useFirstRunStatus(trips.length, onboarding.state);
  const subtitle = tripsQuery.isSuccess ? describeScope(trips.length, currentScope, currentWorkspace?.name) : null;

  return (
    <div
      className={cn(
        newsreader.variable,
        instrumentSans.variable,
        "min-h-screen bg-sand-50 font-instrument text-cocoa-700 selection:bg-[#F0D9CC]"
      )}
    >
      <TripsHeader />

      {/* Content region is a div, not <main> — the root layout already provides
          the <main> landmark, and nesting a second one is invalid. */}
      <div className="mx-auto max-w-[1280px] px-6 pb-[72px] pt-12 sm:px-10">
        <div className="flex flex-col gap-6 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <h1 className="font-newsreader text-[38px] font-medium leading-[1.05] tracking-[-0.02em] text-cocoa-900 sm:text-[44px]">
              {heading}
            </h1>
            <p className="mt-3 text-[15px] text-cocoa-400">
              {subtitle ?? " "}
            </p>
            <Link href="/library" className="mt-2 inline-flex text-sm font-semibold text-clay hover:underline">
              Looking for past trips? Open Library.
            </Link>
          </div>
          <ScopeSegmentedControl />
        </div>

        {tripsQuery.isPending ? (
          <div className="mt-9 rounded-[20px] border border-sand-300 bg-white/60 p-7 text-[14.5px] text-cocoa-500">
            Loading trips…
          </div>
        ) : null}

        {tripsQuery.isError ? (
          <div className="mt-9 rounded-[20px] border border-clay/30 bg-clay-tint/50 p-7 text-[14.5px] text-clay-deep">
            {tripsQuery.error instanceof Error ? tripsQuery.error.message : "Could not load trips."}
          </div>
        ) : null}

        {tripsQuery.isSuccess && trips.length === 0 && currentScope !== "workspace" && onboarding.hydrated && firstRun.showFirstRunDashboard ? <FirstRunDashboard /> : null}

        {tripsQuery.isSuccess && trips.length === 0 && currentScope !== "workspace" && onboarding.hydrated && firstRun.showHelpfulEmptyState ? <HelpfulTripsEmptyState /> : null}

        {tripsQuery.isSuccess && trips.length === 0 && currentScope === "workspace" ? (
          <div className="mt-9 rounded-[20px] border border-dashed border-sand-400 bg-white/60 px-8 py-14 text-center">
            <h2 className="font-newsreader text-[24px] font-semibold text-cocoa-900">No workspace trips yet</h2>
            <p className="mx-auto mt-2 max-w-md text-[14.5px] text-cocoa-400">Create the first trip for this workspace to start planning together.</p>
            <Link href="/trips/new" className="mt-6 inline-flex h-[42px] items-center rounded-full bg-clay px-6 text-[14px] font-semibold text-sand-100 transition hover:bg-clay-dark">Create trip</Link>
          </div>
        ) : null}

        {tripsQuery.isSuccess && trips.length > 0 ? (
          <div className="mt-9 grid gap-7 sm:grid-cols-2 xl:grid-cols-3">
            {trips.map((trip) => (
              <TripCard
                key={trip.id}
                trip={trip}
                workspaceName={trip.workspaceId ? workspaceNames.get(trip.workspaceId) : null}
              />
            ))}
          </div>
        ) : null}

        {showSharedTrips ? (
          <>
            {/* Renders its own `mt-6` box, or null when there are no pending invites. */}
            <CollaborationInvitationsPanel />

            <section className="mt-14">
              <h2 className="font-newsreader text-[27px] font-semibold tracking-[-0.01em] text-cocoa-900">
                Shared with me
              </h2>

              {sharedTripsQuery.isPending ? (
                <div className="mt-[18px] rounded-[20px] border border-sand-300 bg-white/60 p-7 text-[14.5px] text-cocoa-500">
                  Loading shared trips…
                </div>
              ) : null}

              {sharedTripsQuery.isError ? (
                <div className="mt-[18px] rounded-[20px] border border-clay/30 bg-clay-tint/50 p-7 text-[14.5px] text-clay-deep">
                  {sharedTripsQuery.error instanceof Error
                    ? sharedTripsQuery.error.message
                    : "Could not load shared trips."}
                </div>
              ) : null}

              {sharedTripsQuery.isSuccess && sharedTripsQuery.data.length === 0 ? (
                <div className="mt-[18px] flex items-center gap-4 rounded-[20px] border border-dashed border-sand-400 bg-white/60 px-7 py-[26px]">
                  <span className="flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-sand-150 text-[#A08D78]">
                    <ShareNodesIcon className="h-5 w-5" />
                  </span>
                  <div>
                    <p className="text-[15px] font-semibold text-cocoa-900">No shared trips yet</p>
                    <p className="mt-1 text-[14px] text-cocoa-400">
                      Trips that friends or teammates share with you will appear here.
                    </p>
                  </div>
                </div>
              ) : null}

              {sharedTripsQuery.isSuccess && sharedTripsQuery.data.length > 0 ? (
                <div className="mt-[18px] grid gap-7 sm:grid-cols-2 xl:grid-cols-3">
                  {sharedTripsQuery.data.map((trip) => (
                    <SharedTripCard key={trip.id} trip={trip} />
                  ))}
                </div>
              ) : null}
            </section>
          </>
        ) : null}
      </div>
    </div>
  );
}

function describeScope(count: number, scope: string, workspaceName?: string | null) {
  const noun = count === 1 ? "trip" : "trips";
  if (scope === "personal") {
    return `${count} ${noun} in your personal space`;
  }
  if (scope === "workspace") {
    return `${count} ${noun} in ${workspaceName ?? "this workspace"}`;
  }
  return `${count} ${noun} across personal and workspace plans`;
}
