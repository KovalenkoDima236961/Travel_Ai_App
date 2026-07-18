import { ApprovalPolicyCard } from "./ApprovalPolicyCard";
import { BudgetReadinessCard } from "./BudgetReadinessCard";
import { ChecklistReminderCard } from "./ChecklistReminderCard";
import { ExpenseSettlementCard } from "./ExpenseSettlementCard";
import { GroupReadinessCard } from "./GroupReadinessCard";
import { NextBestActionCard } from "./NextBestActionCard";
import { OfflineStatusCard } from "./OfflineStatusCard";
import { QuickNavigationGrid } from "./QuickNavigationGrid";
import { ReadinessCard } from "./ReadinessCard";
import { RecentActivityCard } from "./RecentActivityCard";
import { RouteReadinessCard } from "./RouteReadinessCard";
import { TopFixesPanel } from "./TopFixesPanel";
import { TripOverviewHeader } from "./TripOverviewHeader";
import { TripReadinessSummary } from "./TripReadinessSummary";
import { TripSetupChecklist } from "@/components/onboarding/TripSetupChecklist";
import { RealWorldReadinessCard } from "@/components/verification";
import { TripRecapStatusCard } from "@/components/recap";
import type { TripApprovalState } from "@/entities/approval/model";
import type { Trip } from "@/entities/trip/model";
import type {
  OfflineCommandCenterStatus,
  ReadinessCard as ReadinessCardModel,
  TripCommandCenterData
} from "@/types/trip-command-center";
import type { TripHealth } from "@/types/trip-health";
import type { RealWorldReadiness } from "@/types/verification";

type TripCommandCenterProps = {
  trip: Trip;
  data: TripCommandCenterData;
  health?: TripHealth | null;
  verification?: RealWorldReadiness | null;
  approval?: TripApprovalState | null;
  offlineStatus: OfflineCommandCenterStatus;
  workspaceName?: string | null;
  onSyncNow?: () => void;
  syncing?: boolean;
  setupChecklist?: {
    checklistExists?: boolean;
    collaboratorCount?: number;
    healthLoaded?: boolean;
    healthHasCriticalIssues?: boolean;
  };
};

export function TripCommandCenter({
  trip,
  data,
  health,
  verification,
  approval,
  offlineStatus,
  workspaceName,
  onSyncNow,
  syncing = false,
  setupChecklist
}: TripCommandCenterProps) {
  const cardsById = Object.fromEntries(data.cards.map((card) => [card.id, card])) as Partial<
    Record<ReadinessCardModel["id"], ReadinessCardModel>
  >;

  return (
    <section id="overview" className="scroll-mt-24 space-y-4">
      <TripOverviewHeader
        approval={approval}
        health={health}
        offlineStatus={offlineStatus}
        trip={trip}
        workspaceName={workspaceName}
      />
      <TripSetupChecklist trip={trip} {...setupChecklist} />
      <TripRecapStatusCard tripId={trip.id} />
      <NextBestActionCard action={data.nextBestAction} />
      <TripReadinessSummary cards={data.cards} />
      <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_340px]">
        <TopFixesPanel fixes={data.topFixes} />
        <QuickNavigationGrid groups={data.navigationGroups} />
      </div>
      <div className="grid gap-4 md:grid-cols-2">
        {cardsById.health ? <ReadinessCard card={cardsById.health} /> : null}
        {verification ? <RealWorldReadinessCard readiness={verification} sectionId="real-world-readiness" /> : null}
        {cardsById.route_transport ? <RouteReadinessCard card={cardsById.route_transport} /> : null}
        {cardsById.budget ? <BudgetReadinessCard card={cardsById.budget} /> : null}
        {cardsById.group ? <GroupReadinessCard card={cardsById.group} /> : null}
        {cardsById.checklist_reminders ? (
          <ChecklistReminderCard card={cardsById.checklist_reminders} />
        ) : null}
        {cardsById.expenses_settlements ? (
          <ExpenseSettlementCard card={cardsById.expenses_settlements} />
        ) : null}
        {cardsById.approval_policy ? (
          <ApprovalPolicyCard card={cardsById.approval_policy} />
        ) : null}
        {cardsById.offline ? (
          <OfflineStatusCard card={cardsById.offline} onSyncNow={onSyncNow} syncing={syncing} />
        ) : null}
      </div>
      {cardsById.activity ? (
        <RecentActivityCard card={cardsById.activity} activity={data.recentActivity} />
      ) : null}
    </section>
  );
}
