import { ApprovalStatusBadge } from "@/features/trip-approval/ui/ApprovalStatusBadge";
import Link from "next/link";
import { TripHealthBadge } from "@/components/trip-health";
import { formatRouteSummary, formatTripDates } from "@/lib/trip-command-center/format";
import type { TripApprovalState } from "@/entities/approval/model";
import type { Trip } from "@/entities/trip/model";
import type { OfflineCommandCenterStatus } from "@/types/trip-command-center";
import type { TripHealth } from "@/types/trip-health";

type TripOverviewHeaderProps = {
  trip: Trip;
  health?: TripHealth | null;
  approval?: TripApprovalState | null;
  offlineStatus: OfflineCommandCenterStatus;
  workspaceName?: string | null;
};

export function TripOverviewHeader({
  trip,
  health,
  approval,
  offlineStatus,
  workspaceName
}: TripOverviewHeaderProps) {
  return (
    <section className="rounded-[22px] border border-sand-300 bg-[#FFFDF8] p-6">
      <div className="flex flex-col gap-5 lg:flex-row lg:items-start lg:justify-between">
        <div className="min-w-0">
          <p className="text-[13px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">
            Trip Command Center
          </p>
          <h1 className="mt-2 font-newsreader text-[34px] font-semibold tracking-[-0.01em] text-cocoa-900">
            {trip.destination}
          </h1>
          <p className="mt-2 max-w-[780px] text-[15px] leading-[1.6] text-cocoa-500">
            {formatRouteSummary(trip)} · {formatTripDates(trip)}
          </p>
          <div className="mt-4 flex flex-wrap gap-2">
            <span className="inline-flex rounded-full border border-sand-300 bg-white px-3 py-1.5 text-[13px] font-semibold text-cocoa-600">
              {trip.tripType === "multi_destination" ? "Multi-destination" : "Single destination"}
            </span>
            {workspaceName ? (
              <span className="inline-flex rounded-full border border-sand-300 bg-white px-3 py-1.5 text-[13px] font-semibold text-cocoa-600">
                {workspaceName}
              </span>
            ) : null}
            <span className="inline-flex rounded-full border border-sand-300 bg-white px-3 py-1.5 text-[13px] font-semibold text-cocoa-600">
              {trip.travelers} {trip.travelers === 1 ? "traveler" : "travelers"}
            </span>
          </div>
        </div>
        <div className="flex flex-wrap gap-2 lg:justify-end">
          <Link className="inline-flex rounded-full bg-clay px-3.5 py-1.5 text-[13px] font-semibold text-white" href={`/trips/${trip.id}/today`}>
            Travel mode
          </Link>
          <TripHealthBadge health={health} />
          {approval && approval.workspaceId && approval.status !== "not_required" ? (
            <ApprovalStatusBadge status={approval.status} />
          ) : null}
          <span className="inline-flex rounded-full border border-[#D6DEE8] bg-[#F4F7FA] px-3.5 py-1.5 text-[13px] font-semibold text-[#536171]">
            {offlineStatus.availableOffline
              ? offlineStatus.pendingCount > 0
                ? `${offlineStatus.pendingCount} offline pending`
                : "Offline ready"
              : "Offline not enabled"}
          </span>
        </div>
      </div>
    </section>
  );
}
