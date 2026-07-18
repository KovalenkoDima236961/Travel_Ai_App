import Link from "next/link";
import type { ReactNode } from "react";
import { RiskBadge } from "@/features/approval-risk";
import { TripApprovalBadge } from "@/features/trip-approval";
import { TripHealthBadge } from "@/components/trip-health";
import { formatPaceLabel } from "@/lib/utils";
import { formatAccessSource } from "../model/tripDetailPageModel";
import { formatTripDateRange } from "./tripDetailFormat";
import { StatusPill } from "./StatusPill";
import { ArrowLeftIcon, BoltIcon, CalendarIcon, UsersIcon } from "./icons";
import type { ApprovalRiskQueueSummary } from "@/entities/approval-risk/model";
import type { Trip, TripAccess } from "@/entities/trip/model";
import type { TripHealth } from "@/types/trip-health";

type TripDetailHeaderProps = {
  trip: Trip;
  workspaceName?: string | null;
  accessSource?: TripAccess["source"] | null;
  approvalRisk?: ApprovalRiskQueueSummary | null;
  health?: TripHealth | null;
  healthLoading?: boolean;
  /** Action cluster (Share / Export / Edit or Generate) composed by the page. */
  actions?: ReactNode;
};

export function TripDetailHeader({
  trip,
  workspaceName,
  accessSource,
  approvalRisk,
  health,
  healthLoading,
  actions
}: TripDetailHeaderProps) {
  const dateRange = formatTripDateRange(trip.startDate, trip.days);

  return (
    <div className="flex flex-wrap items-end justify-between gap-6">
      <div className="min-w-0">
        <Link
          href="/trips"
          className="inline-flex items-center gap-2 text-[14px] font-medium text-clay-deep transition hover:text-clay"
        >
          <ArrowLeftIcon className="h-[15px] w-[15px]" />
          Trips
        </Link>
        <div className="mt-3 flex flex-wrap items-center gap-4">
          <h1 className="font-newsreader text-[38px] font-medium leading-[1] tracking-[-0.02em] text-cocoa-900 sm:text-[46px]">
            {trip.destination}
          </h1>
          <StatusPill status={trip.status} />
          {trip.workspaceId ? (
            <Link
              href={`/workspaces/${trip.workspaceId}`}
              className="inline-flex items-center rounded-full border border-sand-300 bg-white px-3.5 py-1.5 text-[13px] font-medium text-cocoa-500 transition hover:border-sand-400 hover:text-cocoa-900"
            >
              {workspaceName ? `Workspace: ${workspaceName}` : "Workspace trip"}
            </Link>
          ) : (
            <span className="inline-flex items-center rounded-full border border-sand-300 bg-white px-3.5 py-1.5 text-[13px] font-medium text-cocoa-500">
              Personal trip
            </span>
          )}
          {accessSource ? (
            <span className="inline-flex items-center rounded-full border border-sand-300 bg-white px-3.5 py-1.5 text-[13px] font-medium text-cocoa-500">
              Access: {formatAccessSource(accessSource)}
            </span>
          ) : null}
          <TripHealthBadge health={health} loading={healthLoading} />
          {trip.workspaceId ? <TripApprovalBadge tripId={trip.id} /> : null}
          {trip.workspaceId ? (
            <RiskBadge
              status={approvalRisk?.status ?? "unknown"}
              score={approvalRisk?.score ?? null}
            />
          ) : null}
        </div>
        <p className="mt-3.5 flex flex-wrap items-center gap-x-[18px] gap-y-2 text-[14.5px] text-cocoa-500">
          <span className="inline-flex items-center gap-2">
            <CalendarIcon className="h-4 w-4 text-[#B09E8A]" />
            {dateRange}
          </span>
          <span className="inline-flex items-center gap-2">
            <UsersIcon className="h-4 w-4 text-[#B09E8A]" />
            {trip.travelers} {trip.travelers === 1 ? "traveler" : "travelers"}
          </span>
          <span className="inline-flex items-center gap-2">
            <BoltIcon className="h-4 w-4 text-[#B09E8A]" />
            {formatPaceLabel(trip.pace)} pace
          </span>
        </p>
      </div>
      <div className="flex flex-wrap items-center gap-2.5"><Link className="inline-flex items-center rounded-md border border-sand-300 bg-white px-3 py-2 text-sm font-medium text-cocoa-700 transition hover:border-sand-400 hover:text-cocoa-900" href={`/trips/${trip.id}/recap`}>Trip recap</Link>{actions}</div>
    </div>
  );
}
