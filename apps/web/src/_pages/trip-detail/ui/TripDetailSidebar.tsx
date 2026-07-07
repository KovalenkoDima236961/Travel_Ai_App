import Link from "next/link";
import { AccommodationPanel } from "@/features/trip-accommodation";
import { CalendarSyncPanel } from "@/features/calendar-sync";
import { TripApprovalPanel } from "@/features/trip-approval";
import { BudgetPanel } from "@/features/trip-budget";
import { CollaboratorsPanel, ShareTripPanel } from "@/features/trip-sharing";
import { TripPresenceIndicator } from "@/components/presence/TripPresenceIndicator";
import { buttonStyles } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import {
  formatBudget,
  formatDate,
  formatInterestLabel,
  formatPaceLabel
} from "@/lib/utils";
import { DetailRow } from "./DetailRow";
import type { BudgetSummary } from "@/entities/budget/model";
import type { TripPresenceSnapshot } from "@/entities/presence/model";
import type { Trip } from "@/entities/trip/model";

type TripDetailSidebarProps = {
  trip: Trip;
  canMutateTrip: boolean;
  offlineDataMode: boolean;
  budgetSummary: BudgetSummary | null;
  perPersonAverage?: { amount: number; currency: string } | null;
  optimizationDisabled: boolean;
  onOpenBudgetOptimization: (dayNumber: number) => void;
  onOpenAccommodationCostSplit: () => void;
  presenceEnabled: boolean;
  currentUserId?: string | null;
  presenceSnapshot: TripPresenceSnapshot | null;
  presenceConnected: boolean;
  onlineActionsEnabled: boolean;
  canManageShare: boolean;
  canSyncCalendar: boolean;
  canManageCollaborators: boolean;
};

export function TripDetailSidebar({
  trip,
  canMutateTrip,
  offlineDataMode,
  budgetSummary,
  perPersonAverage,
  optimizationDisabled,
  onOpenBudgetOptimization,
  onOpenAccommodationCostSplit,
  presenceEnabled,
  currentUserId,
  presenceSnapshot,
  presenceConnected,
  onlineActionsEnabled,
  canManageShare,
  canSyncCalendar,
  canManageCollaborators
}: TripDetailSidebarProps) {
  return (
    <aside className="space-y-6">
      <Card>
        <h2 className="text-lg font-semibold text-slate-950">Trip details</h2>
        <dl className="mt-5 space-y-4 text-sm">
          <DetailRow label="Start date" value={trip.startDate ? formatDate(trip.startDate) : "Not set"} />
          <DetailRow label="Duration" value={`${trip.days} ${trip.days === 1 ? "day" : "days"}`} />
          <DetailRow label="Travelers" value={`${trip.travelers}`} />
          <DetailRow label="Budget" value={formatBudget(trip.budgetAmount, trip.budgetCurrency)} />
          <DetailRow label="Pace" value={formatPaceLabel(trip.pace)} />
          <DetailRow
            label="Created"
            value={formatDate(trip.createdAt, {
              dateStyle: "medium",
              timeStyle: "short"
            })}
          />
        </dl>
        <div className="mt-6">
          <p className="text-sm font-medium text-slate-700">Interests</p>
          <div className="mt-2 flex flex-wrap gap-2">
            {trip.interests.length > 0 ? (
              trip.interests.map((interest) => (
                <span
                  key={interest}
                  className="rounded-full border border-slate-200 bg-slate-50 px-3 py-1 text-xs font-medium text-slate-700"
                >
                  {formatInterestLabel(interest)}
                </span>
              ))
            ) : (
              <span className="text-sm text-slate-500">No interests selected</span>
            )}
          </div>
        </div>
        <Link
          className={buttonStyles({ variant: "secondary", className: "mt-6 w-full" })}
          href={`/trips/${trip.id}/analytics`}
        >
          View cost analytics
        </Link>
      </Card>

      <BudgetPanel
        canEdit={canMutateTrip}
        offline={offlineDataMode}
        offlineSummary={budgetSummary}
        onOpenBudgetOptimization={onOpenBudgetOptimization}
        optimizationDisabled={optimizationDisabled}
        perPersonAverage={perPersonAverage}
        trip={trip}
      />
      <AccommodationPanel
        canEdit={canMutateTrip}
        onOpenCostSplit={
          canMutateTrip && trip.accommodation?.estimatedCost?.amount != null
            ? onOpenAccommodationCostSplit
            : undefined
        }
        trip={trip}
      />

      {presenceEnabled ? (
        <TripPresenceIndicator
          currentUserId={currentUserId}
          isConnected={presenceConnected}
          snapshot={presenceSnapshot}
        />
      ) : null}

      {trip.workspaceId ? <TripApprovalPanel tripId={trip.id} /> : null}
      {canManageShare ? <ShareTripPanel tripId={trip.id} /> : null}
      {onlineActionsEnabled && trip.status === "COMPLETED" && trip.itinerary ? (
        <CalendarSyncPanel canSync={canSyncCalendar} trip={trip} />
      ) : null}
      {onlineActionsEnabled ? (
        <>
          {trip.workspaceId ? (
            <div className="rounded-lg border border-slate-200 bg-slate-50 p-4 text-sm leading-6 text-slate-600">
              Workspace members may already have access. Trip-specific collaborators can still be
              invited for exceptions.
            </div>
          ) : null}
          <CollaboratorsPanel
            canManageCollaborators={canManageCollaborators}
            tripId={trip.id}
          />
        </>
      ) : null}
    </aside>
  );
}
