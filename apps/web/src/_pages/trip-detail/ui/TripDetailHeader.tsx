import Link from "next/link";
import { GenerateItineraryButton } from "@/features/trip-generation";
import { TripApprovalBadge } from "@/features/trip-approval";
import { TripStatusBadge } from "@/components/trips/TripStatusBadge";
import { formatAccessSource } from "../model/tripDetailPageModel";
import type { GenerationJob } from "@/entities/generation-job/model";
import type { Trip, TripAccess } from "@/entities/trip/model";

type TripDetailHeaderProps = {
  trip: Trip;
  workspaceName?: string | null;
  accessSource?: TripAccess["source"] | null;
  canGenerate: boolean;
  hasActiveGenerationJob: boolean;
  onGenerationJobCreated: (job: GenerationJob) => void;
};

export function TripDetailHeader({
  trip,
  workspaceName,
  accessSource,
  canGenerate,
  hasActiveGenerationJob,
  onGenerationJobCreated
}: TripDetailHeaderProps) {
  return (
    <div className="mb-8 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
      <div>
        <Link className="text-sm font-medium text-primary-700 hover:text-primary-600" href="/trips">
          Back to trips
        </Link>
        <div className="mt-3 flex flex-wrap items-center gap-3">
          <h1 className="text-3xl font-semibold text-slate-950">{trip.destination}</h1>
          <TripStatusBadge status={trip.status} />
          {trip.workspaceId ? (
            <Link
              className="rounded-full bg-primary-50 px-3 py-1 text-xs font-semibold text-primary-700 hover:bg-primary-100"
              href={`/workspaces/${trip.workspaceId}`}
            >
              {workspaceName ? `Workspace: ${workspaceName}` : "Workspace trip"}
            </Link>
          ) : (
            <span className="rounded-full bg-slate-100 px-3 py-1 text-xs font-semibold text-slate-700">
              Personal trip
            </span>
          )}
          {accessSource ? (
            <span className="rounded-full bg-slate-100 px-3 py-1 text-xs font-semibold text-slate-700">
              Access: {formatAccessSource(accessSource)}
            </span>
          ) : null}
          {trip.workspaceId ? <TripApprovalBadge tripId={trip.id} /> : null}
        </div>
      </div>
      {canGenerate ? (
        <GenerateItineraryButton
          disabled={hasActiveGenerationJob}
          itineraryRevision={trip.itineraryRevision}
          onJobCreated={onGenerationJobCreated}
          tripId={trip.id}
        />
      ) : null}
    </div>
  );
}
