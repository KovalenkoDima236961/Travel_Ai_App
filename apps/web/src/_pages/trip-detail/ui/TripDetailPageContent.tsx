"use client";

import Link from "next/link";
import dynamic from "next/dynamic";
import { useEffect, useMemo, useState, type ReactNode } from "react";
import { useParams, useSearchParams } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslations } from "next-intl";
import { EmptyState, ErrorState, PageLoadingState } from "@/components/ui";
import { AiAdaptedTripBanner } from "@/components/trips/AiAdaptedTripBanner";
import { useAuth } from "@/components/auth/AuthProvider";
import { AccommodationPanel } from "@/features/trip-accommodation";
import { useTripApprovalRisk } from "@/features/approval-risk";
import { CalendarSyncPanel } from "@/features/calendar-sync";
import { TripApprovalPanel, useTripApproval } from "@/features/trip-approval";
import { TripPolicyPanel } from "@/components/workspace-policy/TripPolicyPanel";
import { CommandCenterSkeleton, TripCommandCenter } from "@/components/trip-command-center";
import { TripCopilot } from "@/components/copilot";
import { VerificationPanel } from "@/components/verification";
import { BudgetPanel } from "@/features/trip-budget";
import { CollaboratorsPanel, ShareTripPanel } from "@/features/trip-sharing";
import { TripPresenceIndicator } from "@/components/presence/TripPresenceIndicator";
import { BudgetOptimizationRequestDialog } from "@/features/budget-optimization";
import { CreateRepairJobDialog, RepairProposalsPanel } from "@/features/trip-repair";
import { EditLockStatus } from "@/features/trip-edit-lock";
import { SoftEditLockWarningDialog } from "@/features/trip-edit-lock";
import { ExportTripMenu } from "@/features/trip-export";
import { GenerateItineraryButton, GenerationJobStatusCard } from "@/features/trip-generation";
import { MergeConflictDialog } from "@/components/itinerary/merge/MergeConflictDialog";
import { ItemCommentsPanel } from "@/features/trip-comments";
import { TripCommentsSummary } from "@/features/trip-comments";
import { OfflineBanner } from "@/components/offline/OfflineBanner";
import { OfflineTripCompanionPanel } from "@/components/offline/OfflineTripCompanionPanel";
import { TripMuteSettings } from "@/components/notifications/TripMuteSettings";
import { PendingOfflineChangesPanel } from "@/components/offline/PendingOfflineChangesPanel";
import { PresenceEditingWarning } from "@/components/presence/TripPresenceIndicator";
import {
  ItineraryEditor,
  normalizeItineraryForSave,
  prepareItineraryForEdit,
  validateEditableItinerary
} from "@/components/trips/ItineraryEditor";
import { CostSplitRuleEditor } from "@/features/cost-splitting";
import { CostSplittingPanel } from "@/features/cost-splitting";
import { DistanceSummary } from "@/features/route-estimation";
import { OpeningHoursWarnings } from "@/components/trips/OpeningHoursWarnings";
import { OptimizeDayOrderDialog } from "@/features/itinerary-optimization";
import { PlaceEnrichmentReviewPanel } from "@/features/itinerary-optimization";
import { SaveTripAsTemplateDialog } from "@/features/trip-template";
import { TripQualityChecks } from "@/components/trips/TripQualityChecks";
import { ItineraryVersionHistory } from "@/components/trips/ItineraryVersionHistory";
import { GroupPreferencesPanel, PollsPanel } from "@/components/trip-decisions";
import { AvailabilityPanel } from "@/components/trip-availability";
import { Button } from "@/shared/ui/button";
import { useWorkspaces } from "@/components/workspaces/WorkspaceProvider";
import { activityKeys, listTripActivity } from "@/lib/api/activity";
import { getCommandCenterSummary } from "@/lib/api/command-center";
import { approvalRiskKeys } from "@/lib/api/approval-risk";
import { useTripActivityStream } from "@/lib/activity/use-trip-activity-stream";
import { budgetKeys, getTripBudgetSummary } from "@/lib/api/budget";
import { budgetConfidenceKeys } from "@/lib/api/budget-confidence";
import {
  costSplittingKeys,
  updateAccommodationCostSplit,
  updateItemCostSplit
} from "@/lib/api/cost-splitting";
import {
  applyBudgetOptimizationProposal,
  budgetOptimizationKeys,
  createBudgetOptimizationJob,
  discardBudgetOptimizationProposal
} from "@/lib/api/budget-optimization";
import {
  applyTripRepairProposal,
  createTripRepairJob,
  discardTripRepairProposal,
  tripRepairKeys
} from "@/lib/api/trip-repair";
import { commentKeys, listTripCommentCounts } from "@/lib/api/comments";
import { isItineraryConflictError } from "@/shared/api/client";
import {
  cancelGenerationJob,
  createGenerationJob,
  generationJobKeys,
  listGenerationJobs
} from "@/lib/api/generation-jobs";
import { buildCommentCountMap } from "@/entities/comment/model";
import { getWeatherForecast, weatherKeys } from "@/lib/api/weather";
import { workspacePolicyKeys } from "@/lib/api/workspace-policies";
import { tripHealthKeys } from "@/lib/api/trip-health";
import { groupReadinessKeys } from "@/lib/api/group-readiness";
import {
  toExportDistanceSummary,
  toExportTripFromPrivateTrip,
  toExportWeatherSummary
} from "@/lib/export/trip-export-adapter";
import {
  getTrip,
  tripKeys,
  updateTripItinerary
} from "@/lib/api/trips";
import { getMyPreferences, userKeys } from "@/lib/api/user";
import { useRouteEstimates } from "@/features/route-estimation";
import { useBudgetOptimizationProposals } from "@/features/budget-optimization";
import { useTripRepairProposals } from "@/features/trip-repair";
import { useGenerationJob } from "@/features/trip-generation";
import { useNetworkStatus } from "@/hooks/useNetworkStatus";
import { useTripChecklist } from "@/hooks/useTripChecklist";
import { useTripHealth } from "@/hooks/useTripHealth";
import { useTripVerification } from "@/hooks/useTripVerification";
import { useGroupReadiness } from "@/hooks/useGroupReadiness";
import { useBudgetConfidence } from "@/hooks/useBudgetConfidence";
import { useTripExpenses } from "@/hooks/useTripExpenses";
import { useTripExpenseSummary } from "@/hooks/useTripExpenseSummary";
import { useTripSettlements } from "@/hooks/useTripSettlements";
import { useTripReminders } from "@/hooks/useTripReminders";
import { useTripAvailability } from "@/hooks/useTripAvailability";
import { useTripPolls } from "@/hooks/useTripPolls";
import { useTripPolicyEvaluation } from "@/hooks/useTripPolicyEvaluation";
import { useItineraryReactions } from "@/hooks/useItineraryReactions";
import { useCostSplittingSummary } from "@/features/cost-splitting";
import { useOfflineSync } from "@/hooks/useOfflineSync";
import { useTripTravelers } from "@/features/cost-splitting";
import { getDayDistanceSummaries } from "@/entities/itinerary/model/distance-utils";
import { applyConflictResolutions, mergeItineraries } from "@/entities/itinerary/model/diff-merge/merge";
import { cloneItinerary } from "@/entities/itinerary/model/diff-merge/normalize";
import { useTripEditLock } from "@/features/trip-edit-lock";
import { isOfflineLikeError } from "@/lib/offline/network";
import {
  cacheTripSnapshot,
  deleteCachedTrip,
  getCachedTrip,
  updateCachedTripItinerary
} from "@/lib/offline/trip-cache";
import { rollbackOfflineCompanionMutation } from "@/lib/offline/cache-writer";
import { recordPwaEngagement } from "@/lib/pwa/pwa-detection";
import { buildTripCommandCenterData } from "@/lib/trip-command-center/readiness";
import { scrollToTabAnchor, TAB_TO_ANCHOR } from "@/lib/trip-command-center/navigation";
import {
  buildTripCommandCenterDataFromSummary,
  healthFromCommandCenterSummary
} from "@/lib/trip-command-center/summary";
import { queryKeys } from "@/lib/query-keys";
import {
  discardMutation,
  enqueueItineraryUpdate,
  markMutationSynced
} from "@/lib/offline/sync-queue";
import { useTripPresenceState } from "@/lib/presence/use-trip-presence-state";
import { useTripPresenceStream } from "@/lib/presence/use-trip-presence-stream";
import { cn, getErrorMessage } from "@/lib/utils";
import type {
  BudgetOptimizationJobRequest,
  BudgetOptimizationProposal
} from "@/entities/budget-optimization/model";
import type {
  CreateRepairJobInput,
  RepairMode,
  RepairProposal
} from "@/entities/trip-repair/model";
import type { EstimatedCost } from "@/entities/budget/model";
import type { CostSplitRule } from "@/entities/cost-splitting/model";
import type {
  AvailabilityOption,
  AvailabilityResultByItem,
  AvailabilitySearchResponse
} from "@/entities/availability/model";
import type { RouteEstimate } from "@/entities/route/model";
import type { EditLockView } from "@/entities/edit-lock/model";
import type {
  CreateGenerationJobRequest,
  GenerationJob,
  GenerationJobType
} from "@/entities/generation-job/model";
import type {
  ConflictResolution,
} from "@/entities/itinerary/model/diff-merge/types";
import type {
  CachedTripRecord,
  PendingItineraryMutation,
  PendingOfflineMutation,
  SyncResult
} from "@/lib/offline/types";
import { isPendingItineraryMutation } from "@/lib/offline/types";
import type { Itinerary, Trip } from "@/entities/trip/model";
import { BudgetOptimizationProposalsPanel } from "./BudgetOptimizationProposalsPanel";
import { TripDetailHeader } from "./TripDetailHeader";
import { TripDetailSidebar } from "./TripDetailSidebar";
import { TripDetailChromeHeader } from "./TripDetailChromeHeader";
import { ItineraryTimeline } from "./ItineraryTimeline";
import { RightRailActivity } from "./RightRailActivity";
import { PencilSquareIcon, ShareNodesIcon } from "./icons";
import { instrumentSans, newsreader } from "./fonts";
import {
  availabilityCostCategory,
  availabilityResultKey,
  defaultConflictResolutions,
  failureMessageForGenerationJob,
  findActiveGenerationJob,
  getCostSplitTargetDetails,
  isActiveGenerationJob,
  isSignificantPriceChange,
  successMessageForGenerationJob,
  targetFromGenerationJob,
  withPendingOfflineItinerary,
  type CostSplitEditorTarget,
  type MergeRecoveryState,
  type RegeneratingTarget
} from "../model/tripDetailPageModel";

const ExpensesPanel = dynamic(
  () => import("@/components/expenses").then((module) => module.ExpensesPanel),
  { loading: PanelLoading }
);
const GroupReadinessPanel = dynamic(
  () => import("@/components/group-readiness").then((module) => module.GroupReadinessPanel),
  { loading: PanelLoading }
);
const TripHealthPanel = dynamic(
  () => import("@/components/trip-health").then((module) => module.TripHealthPanel),
  { loading: PanelLoading }
);
const TripChecklistPanel = dynamic(
  () => import("@/components/checklists").then((module) => module.TripChecklistPanel),
  { loading: PanelLoading }
);
const TripRemindersPanel = dynamic(
  () => import("@/components/trip-reminders").then((module) => module.TripRemindersPanel),
  { loading: PanelLoading }
);
const RouteBuilderPanel = dynamic(
  () => import("@/components/route-builder").then((module) => module.RouteBuilderPanel),
  { loading: PanelLoading }
);
const RouteAlternativesPanel = dynamic(
  () => import("@/components/route-alternatives").then((module) => module.RouteAlternativesPanel),
  { loading: PanelLoading }
);
const RightRailMap = dynamic(
  () => import("./RightRailMap").then((module) => module.RightRailMap),
  { loading: PanelLoading }
);
const RightRailWeather = dynamic(
  () => import("./RightRailWeather").then((module) => module.RightRailWeather),
  { loading: PanelLoading }
);

export function TripDetailPageContent() {
  const loadingT = useTranslations("loading");
  const errorsT = useTranslations("errors");
  const navigationT = useTranslations("navigation");
  const emptyItineraryT = useTranslations("emptyStates.itinerary");
  const params = useParams<{ id: string }>();
  const searchParams = useSearchParams();
  const tripId = params.id;
  const queryClient = useQueryClient();
  const { user } = useAuth();
  const { workspaces } = useWorkspaces();
  const currentUserId = user?.id;
  const networkStatus = useNetworkStatus();
  const invalidateBudgetConfidence = () =>
    Promise.all([
      queryClient.invalidateQueries({ queryKey: budgetConfidenceKeys.all(tripId) }),
      queryClient.invalidateQueries({ queryKey: queryKeys.trip.commandCenter(tripId) })
    ]);
  const [isEditing, setIsEditing] = useState(false);
  const [commentTarget, setCommentTarget] = useState<{
    dayNumber: number;
    itemIndex: number;
    title: string;
    time?: string | null;
  } | null>(null);
  const [draftItinerary, setDraftItinerary] = useState<Itinerary | null>(null);
  const [baseItinerary, setBaseItinerary] = useState<Itinerary | null>(null);
  const [baseItineraryRevision, setBaseItineraryRevision] = useState<number | null>(null);
  const [editorErrors, setEditorErrors] = useState<string[]>([]);
  const [itineraryConflictMessage, setItineraryConflictMessage] = useState<string | null>(null);
  const [mergeRecovery, setMergeRecovery] = useState<MergeRecoveryState | null>(null);
  const [mergeApplyError, setMergeApplyError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);
  const [deepLinkMessage, setDeepLinkMessage] = useState<string | null>(null);
  const [regenerationError, setRegenerationError] = useState<string | null>(null);
  const [activeGenerationJobId, setActiveGenerationJobId] = useState<string | null>(null);
  const [optimizingDayNumber, setOptimizingDayNumber] = useState<number | null>(null);
  const [budgetOptimizationDialogOpen, setBudgetOptimizationDialogOpen] = useState(false);
  const [budgetOptimizationDefaultDayNumber, setBudgetOptimizationDefaultDayNumber] = useState<
    number | null
  >(null);
  const [budgetOptimizationError, setBudgetOptimizationError] = useState<string | null>(null);
  const [handledBudgetOptimizationDeepLink, setHandledBudgetOptimizationDeepLink] = useState<
    string | null
  >(null);
  const [tripRepairDialogOpen, setTripRepairDialogOpen] = useState(false);
  const [tripRepairDefaultMode, setTripRepairDefaultMode] = useState<RepairMode | null>(null);
  const [tripRepairError, setTripRepairError] = useState<string | null>(null);
  const [availabilityResultsByItem, setAvailabilityResultsByItem] =
    useState<AvailabilityResultByItem>({});
  const [availabilityApplyError, setAvailabilityApplyError] = useState<string | null>(null);
  const [costSplitTarget, setCostSplitTarget] = useState<CostSplitEditorTarget | null>(null);
  const [costSplitError, setCostSplitError] = useState<string | null>(null);
  const [lockWarning, setLockWarning] = useState<EditLockView | null>(null);
  const [cachedTripRecord, setCachedTripRecord] = useState<CachedTripRecord | null>(null);
  const [offlineCacheLoading, setOfflineCacheLoading] = useState(false);
  const [offlineUnavailable, setOfflineUnavailable] = useState(false);
  const [saveTemplateOpen, setSaveTemplateOpen] = useState(false);
  const [routeAlternativesOpen, setRouteAlternativesOpen] = useState(false);
  const [loadedSections, setLoadedSections] = useState<Set<string>>(
    () => new Set(["overview"])
  );
  const sectionEnabled = (...sections: string[]) =>
    sections.some((section) => loadedSections.has(section));

  const offlineSync = useOfflineSync({
    userId: currentUserId,
    enabled: Boolean(currentUserId),
    onSyncResults: handleOfflineSyncResults
  });

  const tripQuery = useQuery({
    queryKey: tripKeys.detail(tripId),
    queryFn: () => getTrip(tripId),
    enabled: Boolean(tripId) && networkStatus.online,
    refetchInterval: (query) =>
      networkStatus.online && query.state.data?.status === "PROCESSING" ? 3000 : false
  });

  // Preferences power the walking-distance warning. They are intentionally
  // non-blocking: if the fetch fails we still render the distance estimates and
  // simply omit the preference comparison.
  const preferencesQuery = useQuery({
    queryKey: userKeys.preferences(),
    queryFn: getMyPreferences,
    enabled: networkStatus.online,
    staleTime: 5 * 60 * 1000
  });
  const maxWalkingKmPerDay = preferencesQuery.data?.maxWalkingKmPerDay ?? null;

  const updateMutation = useMutation({
    mutationFn: ({
      itinerary,
      expectedRevision
    }: {
      itinerary: Itinerary;
      expectedRevision: number;
    }) => updateTripItinerary(tripId, itinerary, expectedRevision)
  });

  const createGenerationJobMutation = useMutation({
    mutationFn: (input: CreateGenerationJobRequest) => createGenerationJob(tripId, input)
  });

  const createBudgetOptimizationMutation = useMutation({
    mutationFn: (input: BudgetOptimizationJobRequest) =>
      createBudgetOptimizationJob(tripId, input)
  });

  const applyBudgetOptimizationMutation = useMutation({
    mutationFn: (proposal: BudgetOptimizationProposal) =>
      applyBudgetOptimizationProposal(tripId, proposal.id, tripQuery.data?.itineraryRevision ?? 0)
  });

  const discardBudgetOptimizationMutation = useMutation({
    mutationFn: (proposal: BudgetOptimizationProposal) =>
      discardBudgetOptimizationProposal(tripId, proposal.id)
  });

  const createTripRepairMutation = useMutation({
    mutationFn: (input: CreateRepairJobInput) => createTripRepairJob(tripId, input)
  });

  const applyTripRepairMutation = useMutation({
    mutationFn: (proposal: RepairProposal) =>
      applyTripRepairProposal(tripId, proposal.id, tripQuery.data?.itineraryRevision ?? 0)
  });

  const discardTripRepairMutation = useMutation({
    mutationFn: (proposal: RepairProposal) => discardTripRepairProposal(tripId, proposal.id)
  });

  const updateItemCostSplitMutation = useMutation({
    mutationFn: ({
      target,
      split
    }: {
      target: Extract<CostSplitEditorTarget, { type: "item" }>;
      split: CostSplitRule;
    }) => {
      if (!displayedTrip) {
        throw new Error("Trip is not loaded.");
      }
      return updateItemCostSplit(
        tripId,
        target.dayNumber,
        target.itemIndex,
        displayedTrip.itineraryRevision,
        split
      );
    },
    onSuccess: async (result) => {
      queryClient.setQueryData(tripKeys.detail(tripId), result.trip);
      setCostSplitTarget(null);
      setCostSplitError(null);
      await invalidateCostSplitDependents(result.trip);
      setSuccessMessage("Cost split updated.");
    },
    onError: (error) => {
      if (isItineraryConflictError(error)) {
        setCostSplitError("This itinerary changed. Reload latest version before updating the split.");
        void tripQuery.refetch();
        return;
      }
      setCostSplitError(getErrorMessage(error, "Could not update the cost split."));
    }
  });

  const updateAccommodationCostSplitMutation = useMutation({
    mutationFn: (split: CostSplitRule) => updateAccommodationCostSplit(tripId, split),
    onSuccess: async (updatedTrip) => {
      queryClient.setQueryData(tripKeys.detail(tripId), updatedTrip);
      setCostSplitTarget(null);
      setCostSplitError(null);
      await invalidateCostSplitDependents(updatedTrip);
      setSuccessMessage("Accommodation split updated.");
    },
    onError: (error) => {
      setCostSplitError(getErrorMessage(error, "Could not update the accommodation split."));
    }
  });

  const cancelGenerationJobMutation = useMutation({
    mutationFn: (jobId: string) => cancelGenerationJob(tripId, jobId),
    onSuccess: async (job) => {
      setActiveGenerationJobId(job.id);
      queryClient.setQueryData(generationJobKeys.detail(tripId, job.id), job);
      await queryClient.invalidateQueries({ queryKey: generationJobKeys.list(tripId) });
    }
  });

  const pendingOfflineMutation =
    offlineSync.mutations.find(
      (mutation): mutation is PendingItineraryMutation =>
        mutation.tripId === tripId && isPendingItineraryMutation(mutation)
    ) ?? null;
  const tripOfflineMutations = offlineSync.mutations.filter(
    (mutation) => mutation.tripId === tripId
  );
  const sourceTrip = tripQuery.data ?? cachedTripRecord?.trip ?? null;
  const displayedTrip = sourceTrip
    ? withPendingOfflineItinerary(sourceTrip, pendingOfflineMutation)
    : null;
  const isUsingCachedTrip = Boolean(cachedTripRecord) && (!tripQuery.data || !networkStatus.online);
  const hasPendingOfflineChanges = tripOfflineMutations.length > 0;
  const hasPendingItineraryDraft = Boolean(pendingOfflineMutation);
  const offlineDataMode = isUsingCachedTrip || !networkStatus.online || hasPendingItineraryDraft;
  const onlineActionsEnabled =
    networkStatus.online && !isUsingCachedTrip && !hasPendingItineraryDraft;
  const cachedBudgetSummary = cachedTripRecord?.budgetSummary ?? null;
  const currentItinerary = displayedTrip?.itinerary ?? null;
  const commandCenterSummaryQuery = useQuery({
    queryKey: queryKeys.trip.commandCenter(tripId),
    queryFn: () => getCommandCenterSummary(tripId),
    enabled: onlineActionsEnabled && Boolean(tripId) && Boolean(displayedTrip),
    staleTime: 30 * 1000,
    retry: 1
  });
  const summaryHealth = useMemo(
    () =>
      commandCenterSummaryQuery.data
        ? healthFromCommandCenterSummary(commandCenterSummaryQuery.data)
        : null,
    [commandCenterSummaryQuery.data]
  );

  useEffect(() => {
    if (!displayedTrip || typeof window === "undefined") {
      return;
    }
    const tab = new URLSearchParams(window.location.search).get("tab");
    const deepLinkedSection = tab ? TAB_TO_ANCHOR[tab] : null;
    if (deepLinkedSection) {
      setLoadedSections((current) => new Set([...current, deepLinkedSection]));
    }

    const elements = Array.from(
      document.querySelectorAll<HTMLElement>("[data-load-section]")
    );
    if (!("IntersectionObserver" in window)) {
      setLoadedSections((current) =>
        new Set([
          ...current,
          ...elements
            .map((element) => element.dataset.loadSection)
            .filter((section): section is string => Boolean(section))
        ])
      );
      return;
    }
    const observer = new IntersectionObserver(
      (entries) => {
        const visible = entries
          .filter((entry) => entry.isIntersecting)
          .map((entry) => (entry.target as HTMLElement).dataset.loadSection)
          .filter((section): section is string => Boolean(section));
        if (visible.length > 0) {
          setLoadedSections((current) => new Set([...current, ...visible]));
        }
      },
      { rootMargin: "800px 0px" }
    );
    elements.forEach((element) => observer.observe(element));
    return () => observer.disconnect();
  }, [displayedTrip?.id, displayedTrip?.status]);
  useEffect(() => {
    if (displayedTrip?.workspaceId) {
      void queryClient.invalidateQueries({
        queryKey: workspacePolicyKeys.evaluation(tripId)
      });
    }
  }, [displayedTrip?.itineraryRevision, displayedTrip?.workspaceId, queryClient, tripId]);
  const routeEstimateStates = useRouteEstimates(
    currentItinerary,
    onlineActionsEnabled &&
      sectionEnabled("itinerary", "route") &&
      displayedTrip?.status === "COMPLETED" &&
      Boolean(currentItinerary),
    displayedTrip?.accommodation ?? null
  );
  const routeEstimatesByDay = useMemo<Record<number, RouteEstimate | null>>(() => {
    const estimates: Record<number, RouteEstimate | null> = {};
    routeEstimateStates.byDay.forEach((state, dayNumber) => {
      estimates[dayNumber] = state.estimate;
    });
    return estimates;
  }, [routeEstimateStates.byDay]);
  const fallbackDistanceSummaries = useMemo(
    () =>
      currentItinerary
        ? getDayDistanceSummaries(
            currentItinerary,
            maxWalkingKmPerDay,
            displayedTrip?.accommodation ?? null
          )
        : [],
    [currentItinerary, displayedTrip?.accommodation, maxWalkingKmPerDay]
  );

  const weatherParams = {
    destination: displayedTrip?.destination ?? "",
    startDate: displayedTrip?.startDate ?? "",
    days: displayedTrip?.days ?? 0
  };
  const canFetchWeather =
    Boolean(weatherParams.destination.trim()) &&
    Boolean(weatherParams.startDate) &&
    weatherParams.days > 0;
  const weatherForecastQuery = useQuery({
    queryKey: weatherKeys.forecast(weatherParams),
    queryFn: () => getWeatherForecast(weatherParams),
    enabled: canFetchWeather && onlineActionsEnabled && sectionEnabled("itinerary", "weather"),
    staleTime: 10 * 60 * 1000,
    retry: 1
  });

  // Shares the cache key with BudgetPanel so the summary is fetched once and
  // also feeds budget-aware quality checks.
  const budgetSummaryQuery = useQuery({
    queryKey: budgetKeys.summary(tripId),
    queryFn: () => getTripBudgetSummary(tripId),
    enabled: onlineActionsEnabled && sectionEnabled("budget", "itinerary")
  });

  // Comments are a private collaboration feature: anyone who can view this
  // private trip (owner/editor/viewer) may read and add comments. Counts power
  // the per-item badges. The public share page never mounts this page.
  const tripAccess = displayedTrip?.access;
  const costSplittingEnabled =
    onlineActionsEnabled &&
    sectionEnabled("budget", "expenses", "cost-split") &&
    Boolean(tripId) &&
    Boolean(tripAccess);
  const tripTravelersQuery = useTripTravelers({
    tripId,
    enabled: costSplittingEnabled
  });
  const costSplittingSummaryQuery = useCostSplittingSummary({
    tripId,
    currency: displayedTrip?.budgetCurrency ?? "EUR",
    enabled: costSplittingEnabled
  });
  const approvalRiskQuery = useTripApprovalRisk(
    tripId,
    onlineActionsEnabled &&
      sectionEnabled("approval", "workspace-policy") &&
      Boolean(displayedTrip?.workspaceId)
  );
  const budgetOptimizationProposalsQuery = useBudgetOptimizationProposals({
    tripId,
    status: "pending",
    enabled:
      onlineActionsEnabled &&
      sectionEnabled("budget", "itinerary") &&
      Boolean(tripId) &&
      Boolean(tripAccess)
  });
  const tripRepairProposalsQuery = useTripRepairProposals({
    tripId,
    status: "pending",
    enabled:
      onlineActionsEnabled &&
      sectionEnabled("approval", "workspace-policy", "itinerary") &&
      Boolean(tripId) &&
      Boolean(tripAccess) &&
      Boolean(displayedTrip?.workspaceId)
  });
  const generationJobsQuery = useQuery({
    queryKey: generationJobKeys.list(tripId),
    queryFn: () => listGenerationJobs(tripId),
    enabled: onlineActionsEnabled && Boolean(tripId) && Boolean(tripAccess),
    refetchInterval: (query) => (findActiveGenerationJob(query.state.data ?? []) ? 3000 : false)
  });
  const latestActiveGenerationJob = findActiveGenerationJob(generationJobsQuery.data ?? []);
  const generationJobPoll = useGenerationJob({
    tripId,
    jobId: activeGenerationJobId ?? latestActiveGenerationJob?.id,
    enabled:
      onlineActionsEnabled &&
      Boolean(tripId) &&
      Boolean(activeGenerationJobId ?? latestActiveGenerationJob?.id),
    onCompleted: (job) => {
      void handleGenerationJobCompleted(job);
    },
    onFailed: (job) => {
      void handleGenerationJobFailed(job);
    },
    onCancelled: (job) => {
      void handleGenerationJobCancelled(job);
    }
  });
  const activeGenerationJob = generationJobPoll.data ?? latestActiveGenerationJob ?? null;
  const hasActiveGenerationJob = Boolean(
    activeGenerationJob && isActiveGenerationJob(activeGenerationJob)
  );
  const activeTripRepairJob =
    (generationJobsQuery.data ?? []).find(
      (job) => job.jobType === "policy_repair" && isActiveGenerationJob(job)
    ) ??
    (activeGenerationJob?.jobType === "policy_repair" &&
    isActiveGenerationJob(activeGenerationJob)
      ? activeGenerationJob
      : null);
  const activeRegeneratingTarget =
    activeGenerationJob && isActiveGenerationJob(activeGenerationJob)
      ? targetFromGenerationJob(activeGenerationJob)
      : null;
  const canUsePrivateCollaboration =
    !tripAccess ||
    tripAccess.role === "owner" ||
    tripAccess.role === "editor" ||
    tripAccess.role === "viewer";
  const canComment = onlineActionsEnabled && canUsePrivateCollaboration;
  const decisionsEnabled = Boolean(tripId) && canUsePrivateCollaboration && onlineActionsEnabled;
  const canCreatePoll = Boolean(tripAccess?.canEdit ?? true) && onlineActionsEnabled;
  const budgetConfidenceQuery = useBudgetConfidence({
    tripId,
    currency: budgetSummaryQuery.data?.currency ?? displayedTrip?.budgetCurrency ?? "EUR",
    enabled:
      onlineActionsEnabled &&
      sectionEnabled("budget") &&
      Boolean(tripId) &&
      Boolean(tripAccess) &&
      canUsePrivateCollaboration
  });
  const tripHealthQuery = useTripHealth(tripId, {
    enabled:
      onlineActionsEnabled &&
      sectionEnabled("health", "route") &&
      Boolean(tripId) &&
      Boolean(tripAccess) &&
      canUsePrivateCollaboration
  });
  const tripVerificationQuery = useTripVerification(tripId, {
    enabled:
      onlineActionsEnabled &&
      Boolean(tripId) &&
      Boolean(tripAccess) &&
      canUsePrivateCollaboration
  });
  const groupReadinessQuery = useGroupReadiness(
    tripId,
    onlineActionsEnabled &&
      sectionEnabled("group-readiness", "dates", "decisions") &&
      Boolean(tripId) &&
      Boolean(tripAccess) &&
      canUsePrivateCollaboration
  );
  const checklistQuery = useTripChecklist(tripId, {
    enabled:
      onlineActionsEnabled &&
      sectionEnabled("checklist") &&
      Boolean(tripId) &&
      canUsePrivateCollaboration &&
      displayedTrip?.status === "COMPLETED"
  });
  const remindersQuery = useTripReminders(
    tripId,
    {},
    {
      enabled:
        onlineActionsEnabled &&
        sectionEnabled("reminders") &&
        Boolean(tripId) &&
        canUsePrivateCollaboration &&
        displayedTrip?.status === "COMPLETED"
    }
  );
  const exportExpensesQuery = useTripExpenses({
    tripId,
    enabled:
      onlineActionsEnabled &&
      sectionEnabled("expenses") &&
      Boolean(tripId) &&
      canUsePrivateCollaboration &&
      displayedTrip?.status === "COMPLETED"
  });
  const expenseSummaryQuery = useTripExpenseSummary({
    tripId,
    currency: displayedTrip?.budgetCurrency ?? "EUR",
    enabled:
      onlineActionsEnabled &&
      sectionEnabled("expenses") &&
      Boolean(tripId) &&
      canUsePrivateCollaboration &&
      displayedTrip?.status === "COMPLETED"
  });
  const settlementsQuery = useTripSettlements({
    tripId,
    currency: displayedTrip?.budgetCurrency ?? "EUR",
    enabled:
      onlineActionsEnabled &&
      sectionEnabled("expenses") &&
      Boolean(tripId) &&
      canUsePrivateCollaboration &&
      displayedTrip?.status === "COMPLETED"
  });
  const availabilitySummaryQuery = useTripAvailability(
    tripId,
    decisionsEnabled && sectionEnabled("dates", "decisions")
  );
  const pollsSummaryQuery = useTripPolls(
    tripId,
    decisionsEnabled && sectionEnabled("decisions")
  );
  const tripApprovalQuery = useTripApproval(
    tripId,
    onlineActionsEnabled &&
      sectionEnabled("approval") &&
      Boolean(tripId) &&
      canUsePrivateCollaboration &&
      Boolean(displayedTrip?.workspaceId)
  );
  const policyEvaluation = useTripPolicyEvaluation(
    tripId,
    onlineActionsEnabled &&
      sectionEnabled("workspace-policy", "approval") &&
      Boolean(tripId) &&
      canUsePrivateCollaboration &&
      Boolean(displayedTrip?.workspaceId)
  );
  const recentActivityQuery = useQuery({
    queryKey: [...activityKeys.all(tripId), "overview", 5] as const,
    queryFn: () => listTripActivity(tripId, { limit: 5 }),
    enabled: canComment && Boolean(tripId) && sectionEnabled("activity"),
    staleTime: 60 * 1000
  });
  useEffect(() => {
    if (!tripId || !onlineActionsEnabled || !canUsePrivateCollaboration) {
      return;
    }
    queryClient.invalidateQueries({ queryKey: groupReadinessKeys.detail(tripId) });
  }, [
    availabilitySummaryQuery.dataUpdatedAt,
    checklistQuery.dataUpdatedAt,
    expenseSummaryQuery.dataUpdatedAt,
    onlineActionsEnabled,
    pollsSummaryQuery.dataUpdatedAt,
    queryClient,
    remindersQuery.dataUpdatedAt,
    settlementsQuery.dataUpdatedAt,
    tripApprovalQuery.dataUpdatedAt,
    tripId,
    canUsePrivateCollaboration
  ]);
  const commentsEnabled =
    onlineActionsEnabled &&
    sectionEnabled("itinerary") &&
    Boolean(tripId) &&
    canComment &&
    displayedTrip?.status === "COMPLETED" &&
    Boolean(displayedTrip?.itinerary);
  const commentCountsQuery = useQuery({
    queryKey: commentKeys.counts(tripId),
    queryFn: () => listTripCommentCounts(tripId),
    enabled: commentsEnabled
  });
  const reactionSummariesQuery = useItineraryReactions(
    tripId,
    decisionsEnabled &&
      sectionEnabled("itinerary", "decisions") &&
      displayedTrip?.status === "COMPLETED" &&
      Boolean(displayedTrip?.itinerary)
  );
  const presenceEnabled =
    onlineActionsEnabled &&
    sectionEnabled("itinerary", "sharing", "activity") &&
    Boolean(tripId) &&
    Boolean(currentUserId) &&
    Boolean(
      tripAccess &&
        (tripAccess.role === "owner" ||
          tripAccess.role === "editor" ||
          tripAccess.role === "viewer")
    );
  const presenceStream = useTripPresenceStream({
    tripId,
    enabled: presenceEnabled
  });
  useTripActivityStream({
    tripId,
    enabled: presenceEnabled,
    onActivityCreated: () => {
      void queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) });
      void queryClient.invalidateQueries({ queryKey: budgetOptimizationKeys.all(tripId) });
      void invalidateBudgetConfidence();
      void queryClient.invalidateQueries({ queryKey: tripHealthKeys.detail(tripId) });
    }
  });
  const setPresenceState = useTripPresenceState(tripId, presenceEnabled);
  const editLocksEnabled =
    onlineActionsEnabled &&
    Boolean(tripId) &&
    Boolean(currentUserId) &&
    Boolean(
      tripAccess &&
        (tripAccess.role === "owner" ||
          tripAccess.role === "editor" ||
          tripAccess.role === "viewer")
    );
  const canEditLoadedItinerary =
    Boolean(tripQuery.data) &&
    (tripAccess?.canEdit ?? true) &&
    displayedTrip?.status === "COMPLETED" &&
    Boolean(displayedTrip?.itinerary);
  const editLock = useTripEditLock({
    tripId,
    enabled: onlineActionsEnabled && editLocksEnabled,
    canEdit: canEditLoadedItinerary,
    onLockConflict: setLockWarning
  });

  useEffect(() => {
    if (!presenceEnabled) {
      return;
    }
    return () => {
      void setPresenceState("viewing");
    };
  }, [presenceEnabled, setPresenceState]);

  useEffect(() => {
    if (!presenceEnabled) {
      return;
    }
    function handleVisibilityChange() {
      if (document.visibilityState === "hidden") {
        void setPresenceState("viewing");
      } else if (isEditing) {
        void setPresenceState("editing");
      }
    }
    document.addEventListener("visibilitychange", handleVisibilityChange);
    return () => {
      document.removeEventListener("visibilitychange", handleVisibilityChange);
    };
  }, [isEditing, presenceEnabled, setPresenceState]);

  useEffect(() => {
    if (!latestActiveGenerationJob) {
      return;
    }
    if (!activeGenerationJob || !isActiveGenerationJob(activeGenerationJob)) {
      setActiveGenerationJobId(latestActiveGenerationJob.id);
    }
  }, [activeGenerationJob, latestActiveGenerationJob]);

  useEffect(() => {
    setAvailabilityResultsByItem({});
  }, [tripId, displayedTrip?.itineraryRevision]);

  const commentCounts = commentCountsQuery.data ?? [];
  const commentCountMap = useMemo(
    () => buildCommentCountMap(commentCounts),
    [commentCounts]
  );
  const reactionSummaryMap = useMemo(() => {
    const entries = reactionSummariesQuery.data ?? [];
    return Object.fromEntries(
      entries.map((summary) => [`${summary.dayNumber}:${summary.itemIndex}`, summary])
    );
  }, [reactionSummariesQuery.data]);
  const exportTrip = useMemo(
    () =>
      displayedTrip
        ? toExportTripFromPrivateTrip(displayedTrip, {
            weatherSummary: toExportWeatherSummary(weatherForecastQuery.data ?? null),
            distanceSummary: toExportDistanceSummary(
              fallbackDistanceSummaries,
              routeEstimatesByDay
            ),
            budgetSummary: budgetSummaryQuery.data ?? cachedBudgetSummary ?? null,
            checklist: checklistQuery.data?.checklist ?? null,
            reminders: remindersQuery.data?.reminders ?? null,
            expenses: exportExpensesQuery.data?.items ?? null
          })
        : null,
    [
      cachedBudgetSummary,
      budgetSummaryQuery.data,
      checklistQuery.data?.checklist,
      displayedTrip,
      exportExpensesQuery.data?.items,
      fallbackDistanceSummaries,
      remindersQuery.data?.reminders,
      routeEstimatesByDay,
      weatherForecastQuery.data
    ]
  );
  const commandCenterOfflineStatus = useMemo(
    () => ({
      online: networkStatus.online,
      availableOffline: Boolean(cachedTripRecord) || isUsingCachedTrip || offlineDataMode,
      pendingCount: tripOfflineMutations.length,
      failedCount: tripOfflineMutations.filter((mutation) => mutation.status === "failed").length,
      conflictCount: tripOfflineMutations.filter((mutation) => mutation.status === "conflict").length,
      syncing: offlineSync.syncing,
      cachedAt: cachedTripRecord?.cachedAt ?? null
    }),
    [
      cachedTripRecord,
      isUsingCachedTrip,
      networkStatus.online,
      offlineDataMode,
      offlineSync.syncing,
      tripOfflineMutations
    ]
  );
  const commandCenterData = useMemo(
    () => {
      if (commandCenterSummaryQuery.data) {
        return buildTripCommandCenterDataFromSummary(
          commandCenterSummaryQuery.data,
          commandCenterOfflineStatus
        );
      }
      return displayedTrip
        ? buildTripCommandCenterData({
            trip: displayedTrip,
            health: tripHealthQuery.data ?? null,
            budgetSummary: budgetSummaryQuery.data ?? cachedBudgetSummary ?? null,
            budgetConfidence: budgetConfidenceQuery.data ?? null,
            availability: availabilitySummaryQuery.data ?? null,
            checklist: checklistQuery.data ?? null,
            reminders: remindersQuery.data ?? null,
            expenseSummary: expenseSummaryQuery.data ?? null,
            settlements: settlementsQuery.data ?? null,
            approval: tripApprovalQuery.data ?? null,
            policyEvaluation: policyEvaluation.query.data ?? null,
            approvalRisk: approvalRiskQuery.data ?? null,
            activity: recentActivityQuery.data?.items ?? null,
            polls: pollsSummaryQuery.data ?? null,
            groupReadiness: groupReadinessQuery.data ?? null,
            generationJobs: generationJobsQuery.data ?? null,
            offlineStatus: commandCenterOfflineStatus,
            userAccess: {
              canEdit: Boolean(displayedTrip.access?.canEdit ?? true) && onlineActionsEnabled,
              canCollaborate: canUsePrivateCollaboration,
              canView: true,
              currentUserId
            }
          })
        : null;
    },
    [
      approvalRiskQuery.data,
      availabilitySummaryQuery.data,
      budgetConfidenceQuery.data,
      budgetSummaryQuery.data,
      cachedBudgetSummary,
      canUsePrivateCollaboration,
      checklistQuery.data,
      commandCenterSummaryQuery.data,
      commandCenterOfflineStatus,
      currentUserId,
      displayedTrip,
      expenseSummaryQuery.data,
      generationJobsQuery.data,
      groupReadinessQuery.data,
      onlineActionsEnabled,
      policyEvaluation.query.data,
      pollsSummaryQuery.data,
      recentActivityQuery.data?.items,
      remindersQuery.data,
      settlementsQuery.data,
      tripApprovalQuery.data,
      tripHealthQuery.data
    ]
  );

  useEffect(() => {
    if (!displayedTrip || typeof window === "undefined") {
      return;
    }
    setDeepLinkMessage(null);
    const tab = searchParams?.get("tab");
    return scrollToTabAnchor(tab);
  }, [displayedTrip?.id, searchParams]);

  useEffect(() => {
    function handleMissingDeepLink() {
      setDeepLinkMessage(errorsT("deepLinkMissing"));
    }
    window.addEventListener("travel-ai:deep-link-missing", handleMissingDeepLink);
    return () => window.removeEventListener("travel-ai:deep-link-missing", handleMissingDeepLink);
  }, [errorsT]);

  useEffect(() => {
    const shouldLoadCachedTrip =
      Boolean(tripId) &&
      Boolean(currentUserId) &&
      (!networkStatus.online ||
        (tripQuery.isError && isOfflineLikeError(tripQuery.error)));

    if (!shouldLoadCachedTrip || !currentUserId) {
      return;
    }

    let cancelled = false;
    setOfflineCacheLoading(true);
    setOfflineUnavailable(false);

    getCachedTrip(tripId, currentUserId)
      .then((record) => {
        if (cancelled) {
          return;
        }
        setCachedTripRecord(record);
        setOfflineUnavailable(!record);
        if (record?.budgetSummary) {
          queryClient.setQueryData(budgetKeys.summary(tripId), record.budgetSummary);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setCachedTripRecord(null);
          setOfflineUnavailable(true);
        }
      })
      .finally(() => {
        if (!cancelled) {
          setOfflineCacheLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [
    currentUserId,
    networkStatus.online,
    queryClient,
    tripId,
    tripQuery.error,
    tripQuery.isError
  ]);

  useEffect(() => {
    if (
      !currentUserId ||
      !networkStatus.online ||
      !tripQuery.data ||
      pendingOfflineMutation
    ) {
      return;
    }

    setCachedTripRecord(null);
    setOfflineUnavailable(false);
    void cacheTripSnapshot({
      userId: currentUserId,
      trip: tripQuery.data,
      budgetSummary: budgetSummaryQuery.data ?? null,
      accommodation: tripQuery.data.accommodation ?? null
    });
  }, [
    budgetSummaryQuery.data,
    currentUserId,
    networkStatus.online,
    pendingOfflineMutation,
    tripQuery.data
  ]);

  useEffect(() => {
    if (displayedTrip) {
      recordPwaEngagement();
    }
  }, [displayedTrip?.id]);

  useEffect(() => {
    if (!displayedTrip || typeof window === "undefined") {
      return;
    }
    const rawDay = new URLSearchParams(window.location.search).get("budgetOptimizeDay");
    if (!rawDay || handledBudgetOptimizationDeepLink === rawDay) {
      return;
    }
    const dayNumber = Number.parseInt(rawDay, 10);
    if (!Number.isInteger(dayNumber) || dayNumber <= 0) {
      setHandledBudgetOptimizationDeepLink(rawDay);
      return;
    }
    const canOpen =
      (displayedTrip.access?.canEdit ?? true) &&
      onlineActionsEnabled &&
      Boolean(displayedTrip.itinerary);
    if (!canOpen) {
      return;
    }
    setBudgetOptimizationDefaultDayNumber(dayNumber);
    setBudgetOptimizationError(null);
    setBudgetOptimizationDialogOpen(true);
    setHandledBudgetOptimizationDeepLink(rawDay);
  }, [displayedTrip, handledBudgetOptimizationDeepLink, onlineActionsEnabled]);

  if (!displayedTrip && (tripQuery.isPending || offlineCacheLoading)) {
    return (
      <DetailShell>
        <PageLoadingState
          cardCount={5}
          label={offlineCacheLoading ? loadingT("savedTrip") : loadingT("trip")}
        />
      </DetailShell>
    );
  }

  if (!displayedTrip && offlineUnavailable) {
    return (
      <DetailShell>
        <div className="rounded-[18px] border border-[#EAD9B8] bg-[#FDF0E3] p-6 text-[14px] text-[#96682A]">
          This trip is not available offline yet. Open it once while online.
        </div>
        <BackToTripsLink />
      </DetailShell>
    );
  }

  if (!displayedTrip && tripQuery.isError) {
    return (
      <DetailShell>
        <ErrorState
          className="rounded-[18px]"
          description={errorsT("tripLoadDescription")}
          developmentDetails={tripQuery.error instanceof Error ? tripQuery.error.message : undefined}
          retryAction={{
            onRetry: () => void tripQuery.refetch(),
            pending: tripQuery.isFetching
          }}
          secondaryAction={{ href: "/trips", label: navigationT("trips") }}
          title={errorsT("tripLoadTitle")}
        />
      </DetailShell>
    );
  }

  if (!displayedTrip) {
    return (
      <DetailShell>
        <ErrorState
          className="rounded-[18px]"
          description={errorsT("tripLoadDescription")}
          secondaryAction={{ href: "/trips", label: navigationT("trips") }}
          title={errorsT("tripLoadTitle")}
        />
      </DetailShell>
    );
  }

  const trip = displayedTrip;
  const access = trip.access;
  const workspaceName =
    trip.workspaceId != null
      ? workspaces.find((workspace) => workspace.id === trip.workspaceId)?.name ?? null
      : null;
  const canEditTripAccess = access?.canEdit ?? true;
  const canMutateTrip = canEditTripAccess && onlineActionsEnabled;
  const canManageShare = (access?.canManageShare ?? true) && onlineActionsEnabled;
  const canManageCollaborators =
    (access?.canManageCollaborators ?? true) && onlineActionsEnabled;
  const canRestoreVersion = (access?.canRestoreVersion ?? canEditTripAccess) && onlineActionsEnabled;
  const canGenerate = canMutateTrip && (trip.status === "DRAFT" || trip.status === "FAILED");
  const canEditItinerary =
    canEditTripAccess && trip.status === "COMPLETED" && Boolean(trip.itinerary);
  const canSaveTemplate = canMutateTrip && trip.status === "COMPLETED" && Boolean(trip.itinerary);
  const canSyncCalendar = canMutateTrip && trip.status === "COMPLETED" && Boolean(trip.itinerary);
  const editingRevisionChanged =
    isEditing &&
    baseItineraryRevision != null &&
    trip.itineraryRevision !== baseItineraryRevision;
  const optimizingDay =
    optimizingDayNumber != null
      ? (trip.itinerary?.days ?? []).find(
          (day, index) => (day.day || index + 1) === optimizingDayNumber
        ) ?? null
      : null;
  const costSplittingSummary = costSplittingSummaryQuery.data ?? null;
  const activeTripTravelers =
    tripTravelersQuery.data?.travelers.filter((traveler) => traveler.status === "active") ?? [];
  const perPersonAverage =
    costSplittingSummary && costSplittingSummary.summary.travelerCount > 0
      ? {
          amount:
            costSplittingSummary.summary.allocatedTotal /
            costSplittingSummary.summary.travelerCount,
          currency: costSplittingSummary.currency
        }
      : null;
  const costSplitTargetDetails = costSplitTarget
    ? getCostSplitTargetDetails(trip, costSplitTarget)
    : null;

  function enterEditMode() {
    if (!trip.itinerary) {
      return;
    }
    const preparedItinerary = prepareItineraryForEdit(trip.itinerary);
    setBaseItinerary(cloneItinerary(preparedItinerary));
    setDraftItinerary(cloneItinerary(preparedItinerary));
    setBaseItineraryRevision(trip.itineraryRevision);
    setEditorErrors([]);
    setItineraryConflictMessage(null);
    setMergeRecovery(null);
    setMergeApplyError(null);
    setRegenerationError(null);
    setSuccessMessage(null);
    setIsEditing(true);
    void setPresenceState("editing");
  }

  function enterEditModeFromOfflineMutation(mutation: PendingItineraryMutation) {
    setBaseItinerary(cloneItinerary(mutation.baseItinerary));
    setDraftItinerary(cloneItinerary(mutation.draftItinerary));
    setBaseItineraryRevision(mutation.baseRevision);
    setEditorErrors([]);
    setItineraryConflictMessage(null);
    setMergeRecovery(null);
    setMergeApplyError(null);
    setRegenerationError(null);
    setSuccessMessage("Offline edit: changes will sync when you are back online.");
    setIsEditing(true);
  }

  async function startEditing() {
    if (!canEditItinerary) {
      return;
    }

    if (pendingOfflineMutation) {
      enterEditModeFromOfflineMutation(pendingOfflineMutation);
      return;
    }

    if (!networkStatus.online || isUsingCachedTrip) {
      enterEditMode();
      setSuccessMessage("Offline edit: changes will sync when you are back online.");
      return;
    }

    try {
      const result = await editLock.acquire();
      if (result.acquired) {
        enterEditMode();
        return;
      }
      if (result.lock) {
        setLockWarning(result.lock);
        return;
      }
      setEditorErrors(["Could not start edit mode."]);
    } catch (error) {
      setEditorErrors([
        error instanceof Error ? error.message : "Could not start edit mode."
      ]);
    }
  }

  async function cancelEditing() {
    await editLock.release();
    clearEditSession();
    setEditorErrors([]);
    setItineraryConflictMessage(null);
    void setPresenceState("viewing");
  }

  async function saveItinerary() {
    if (!draftItinerary) {
      return;
    }

    const normalized = normalizeItineraryForSave(draftItinerary);
    const errors = validateEditableItinerary(normalized);
    if (errors.length > 0) {
      setEditorErrors(errors);
      return;
    }

    try {
      setEditorErrors([]);
      setItineraryConflictMessage(null);
      setRegenerationError(null);

      if (!networkStatus.online || isUsingCachedTrip || pendingOfflineMutation) {
        if (!currentUserId || !baseItinerary || baseItineraryRevision == null) {
          setEditorErrors(["Could not save offline draft. Reload this trip while online."]);
          return;
        }

        await enqueueItineraryUpdate({
          tripId,
          userId: currentUserId,
          baseRevision: baseItineraryRevision,
          baseItinerary,
          draftItinerary: normalized
        });

        let nextCachedRecord = await updateCachedTripItinerary({
          tripId,
          userId: currentUserId,
          itinerary: normalized
        });

        if (!nextCachedRecord) {
          await cacheTripSnapshot({
            userId: currentUserId,
            trip: {
              ...trip,
              itinerary: normalized,
              itineraryRevision: baseItineraryRevision
            },
            budgetSummary: budgetSummaryQuery.data ?? cachedBudgetSummary ?? null,
            accommodation: trip.accommodation ?? null
          });
          nextCachedRecord = await getCachedTrip(tripId, currentUserId);
        }

        if (nextCachedRecord) {
          setCachedTripRecord(nextCachedRecord);
        }
        await offlineSync.refresh();
        clearEditSession();
        setSuccessMessage("Saved offline. Will sync when you are back online.");
        return;
      }

      const updated = await updateMutation.mutateAsync({
        itinerary: normalized,
        expectedRevision: baseItineraryRevision ?? trip.itineraryRevision
      });
      await completeItinerarySave(updated, "Itinerary saved.");
    } catch (error) {
      if (isItineraryConflictError(error)) {
        await prepareMergeRecovery(normalized, error.currentItineraryRevision);
        setEditorErrors([]);
        return;
      }
      setEditorErrors([
        error instanceof Error ? error.message : "Could not save itinerary."
      ]);
    }
  }

  async function reloadLatestAfterConflict() {
    await editLock.release();
    setItineraryConflictMessage(null);
    clearEditSession();
    void setPresenceState("viewing");
    await tripQuery.refetch();
  }

  async function cancelLocalChangesAfterConflict() {
    if (mergeRecovery?.offlineMutation) {
      await discardOfflineMutationWithCache(
        mergeRecovery.offlineMutation,
        mergeRecovery.latestTrip
      );
      return;
    }

    await editLock.release();
    setItineraryConflictMessage(null);
    if (mergeRecovery?.latestTrip) {
      queryClient.setQueryData(tripKeys.detail(tripId), mergeRecovery.latestTrip);
    }
    clearEditSession();
    void setPresenceState("viewing");
    await tripQuery.refetch();
  }

  async function prepareMergeRecovery(
    localDraft: Itinerary,
    latestRevisionHint: number,
    applyErrorMessage?: string
  ) {
    if (!baseItinerary || baseItineraryRevision == null) {
      setItineraryConflictMessage("This itinerary changed while you were editing.");
      setMergeRecovery(null);
      return;
    }

    try {
      const latestTrip = await getTrip(tripId);
      queryClient.setQueryData(tripKeys.detail(tripId), latestTrip);

      if (!latestTrip.itinerary) {
        setItineraryConflictMessage("The latest trip no longer has an itinerary.");
        setMergeRecovery(null);
        return;
      }

      const mergeResult = mergeItineraries(baseItinerary, localDraft, latestTrip.itinerary, {
        baseRevision: baseItineraryRevision,
        latestRevision: latestTrip.itineraryRevision ?? latestRevisionHint
      });
      setDraftItinerary(cloneItinerary(localDraft));
      setMergeRecovery({
        latestTrip,
        mergeResult,
        resolutions: defaultConflictResolutions(mergeResult)
      });
      setItineraryConflictMessage(null);
      setMergeApplyError(applyErrorMessage ?? null);
    } catch (fetchError) {
      setItineraryConflictMessage(
        getErrorMessage(fetchError, "Could not load the latest itinerary for merging.")
      );
      setMergeRecovery(null);
    }
  }

  function handleOfflineSyncResults(results: SyncResult[]) {
    const result = results.find((item) => item.mutation.tripId === tripId);
    if (!result) {
      return;
    }

    if (
      result.status === "synced" &&
      isPendingItineraryMutation(result.mutation) &&
      result.trip
    ) {
      queryClient.setQueryData(tripKeys.detail(tripId), result.trip);
      setCachedTripRecord(null);
      setRegenerationError(null);
      setSuccessMessage("Offline changes synced.");
      void Promise.all([
        queryClient.invalidateQueries({ queryKey: tripKeys.itineraryVersions(tripId) }),
        queryClient.invalidateQueries({ queryKey: budgetKeys.summary(tripId) }),
        invalidateBudgetConfidence(),
        queryClient.invalidateQueries({ queryKey: approvalRiskKeys.trip(tripId) }),
        queryClient.invalidateQueries({ queryKey: tripHealthKeys.detail(tripId) }),
        queryClient.invalidateQueries({ queryKey: ["route-estimate", "walking"] }),
        queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) })
      ]);
      return;
    }

    if (result.status === "synced") {
      setSuccessMessage("Offline companion changes synced.");
      void Promise.all([
        queryClient.invalidateQueries({ queryKey: ["trip-checklists"] }),
        queryClient.invalidateQueries({ queryKey: ["trip-reminders"] }),
        queryClient.invalidateQueries({ queryKey: ["expenses"] }),
        invalidateBudgetConfidence(),
        queryClient.invalidateQueries({ queryKey: tripHealthKeys.detail(tripId) }),
        queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) })
      ]);
      return;
    }

    if (result.status === "conflict" && isPendingItineraryMutation(result.mutation)) {
      void prepareOfflineMergeRecovery(
        result.mutation,
        result.latestTrip ?? null,
        result.currentItineraryRevision ?? null
      );
      return;
    }

    if (result.status === "failed" && !result.retryable) {
      setRegenerationError(
        result.errorMessage ?? "Offline draft could not be saved. Review or discard changes."
      );
    }
  }

  async function prepareOfflineMergeRecovery(
    mutation: PendingItineraryMutation,
    latestTripHint?: Trip | null,
    latestRevisionHint?: number | null
  ) {
    try {
      const latestTrip = latestTripHint ?? (await getTrip(tripId));
      queryClient.setQueryData(tripKeys.detail(tripId), latestTrip);

      if (!latestTrip.itinerary) {
        setItineraryConflictMessage("The latest trip no longer has an itinerary.");
        setMergeRecovery(null);
        return;
      }

      const mergeResult = mergeItineraries(
        mutation.baseItinerary,
        mutation.draftItinerary,
        latestTrip.itinerary,
        {
          baseRevision: mutation.baseRevision,
          latestRevision: latestTrip.itineraryRevision ?? latestRevisionHint ?? mutation.baseRevision
        }
      );
      setBaseItinerary(cloneItinerary(mutation.baseItinerary));
      setDraftItinerary(cloneItinerary(mutation.draftItinerary));
      setBaseItineraryRevision(mutation.baseRevision);
      setMergeRecovery({
        latestTrip,
        mergeResult,
        resolutions: defaultConflictResolutions(mergeResult),
        offlineMutation: mutation
      });
      setItineraryConflictMessage(null);
      setMergeApplyError("This trip changed while you were offline.");
    } catch (fetchError) {
      setItineraryConflictMessage(
        getErrorMessage(fetchError, "Could not load the latest itinerary for merging.")
      );
      setMergeRecovery(null);
    }
  }

  async function applyMergeRecovery() {
    if (!mergeRecovery?.latestTrip.itinerary || !mergeRecovery.mergeResult.mergedItinerary) {
      return;
    }

    const merged =
      mergeRecovery.mergeResult.safety === "safe"
        ? mergeRecovery.mergeResult.mergedItinerary
        : applyConflictResolutions(
            mergeRecovery.latestTrip.itinerary,
            mergeRecovery.mergeResult,
            mergeRecovery.resolutions
          );
    const normalized = normalizeItineraryForSave(merged);
    const errors = validateEditableItinerary(normalized);
    if (errors.length > 0) {
      setMergeApplyError(errors.join(" "));
      return;
    }

    try {
      setMergeApplyError(null);
      setEditorErrors([]);
      const offlineMutation = mergeRecovery.offlineMutation;
      const updated = await updateMutation.mutateAsync({
        itinerary: normalized,
        expectedRevision: mergeRecovery.latestTrip.itineraryRevision
      });
      if (offlineMutation) {
        await markMutationSynced(offlineMutation.mutationId);
        await completeItinerarySave(updated, "Offline changes synced.");
      } else {
        await completeItinerarySave(updated, "Your changes were merged successfully.");
      }
    } catch (error) {
      if (isItineraryConflictError(error)) {
        if (mergeRecovery.offlineMutation && mergeRecovery.latestTrip.itinerary) {
          await prepareOfflineMergeRecovery(
            {
              ...mergeRecovery.offlineMutation,
              baseRevision: mergeRecovery.latestTrip.itineraryRevision,
              baseItinerary: mergeRecovery.latestTrip.itinerary,
              draftItinerary: normalized
            },
            null,
            error.currentItineraryRevision
          );
        } else {
          await prepareMergeRecovery(
            normalized,
            error.currentItineraryRevision,
            "The itinerary changed again while merging. Reload latest and try again."
          );
        }
        return;
      }
      setMergeApplyError(getErrorMessage(error, "Could not apply merged itinerary."));
    }
  }

  function updateConflictResolution(
    conflictKey: string,
    resolution: ConflictResolution
  ) {
    setMergeRecovery((current) =>
      current
        ? {
            ...current,
            resolutions: {
              ...current.resolutions,
              [conflictKey]: resolution
            }
          }
        : current
    );
  }

  async function viewLatestFromMerge() {
    if (!window.confirm("View the latest itinerary and discard your local draft?")) {
      return;
    }
    await cancelLocalChangesAfterConflict();
  }

  function reviewPendingOfflineChanges() {
    if (!pendingOfflineMutation) {
      return;
    }

    if (pendingOfflineMutation.status === "conflict" && networkStatus.online) {
      void prepareOfflineMergeRecovery(pendingOfflineMutation);
      return;
    }

    enterEditModeFromOfflineMutation(pendingOfflineMutation);
  }

  async function discardPendingOfflineChanges() {
    if (!pendingOfflineMutation) {
      return;
    }
    if (!window.confirm("Discard offline itinerary changes?")) {
      return;
    }

    await discardOfflineMutationWithCache(pendingOfflineMutation);
  }

  async function discardTripOfflineMutation(mutation: PendingOfflineMutation) {
    if (isPendingItineraryMutation(mutation)) {
      await discardPendingOfflineChanges();
      return;
    }

    if (!window.confirm("Discard this offline change?")) {
      return;
    }

    await rollbackOfflineCompanionMutation(mutation);
    await discardMutation(mutation.mutationId);
    await offlineSync.refresh();
    setSuccessMessage("Offline change discarded.");
  }

  async function refreshOfflineCopy() {
    if (!currentUserId || !networkStatus.online) {
      setRegenerationError("This action requires internet.");
      return;
    }

    try {
      const latestTrip = tripQuery.data ?? (await getTrip(tripId));
      await cacheTripSnapshot({
        userId: currentUserId,
        trip: latestTrip,
        budgetSummary: budgetSummaryQuery.data ?? null,
        accommodation: latestTrip.accommodation ?? null
      });
      setCachedTripRecord(await getCachedTrip(tripId, currentUserId));
      setOfflineUnavailable(false);
      setRegenerationError(null);
      setSuccessMessage("Offline copy refreshed.");
    } catch (error) {
      setRegenerationError(getErrorMessage(error, "Could not refresh the offline copy."));
    }
  }

  async function removeOfflineCopy() {
    if (!currentUserId) {
      return;
    }
    if (tripOfflineMutations.length > 0) {
      setRegenerationError("Sync or discard pending changes before removing this offline copy.");
      return;
    }
    if (!window.confirm("Remove this trip from offline storage on this device?")) {
      return;
    }

    await deleteCachedTrip(tripId, currentUserId);
    setCachedTripRecord(null);
    setOfflineUnavailable(false);
    setSuccessMessage("Offline copy removed from this device.");
  }

  async function discardOfflineMutationWithCache(
    mutation: PendingItineraryMutation,
    latestTripHint?: Trip | null
  ) {
    await discardMutation(mutation.mutationId);
    let restoredFromServer = false;

    if (latestTripHint) {
      queryClient.setQueryData(tripKeys.detail(tripId), latestTripHint);
      if (currentUserId) {
        await cacheTripSnapshot({
          userId: currentUserId,
          trip: latestTripHint,
          budgetSummary: budgetSummaryQuery.data ?? null,
          accommodation: latestTripHint.accommodation ?? null
        });
      }
      setCachedTripRecord(null);
      restoredFromServer = true;
    } else if (networkStatus.online) {
      try {
        const latestTrip = await getTrip(tripId);
        queryClient.setQueryData(tripKeys.detail(tripId), latestTrip);
        if (currentUserId) {
          await cacheTripSnapshot({
            userId: currentUserId,
            trip: latestTrip,
            budgetSummary: budgetSummaryQuery.data ?? null,
            accommodation: latestTrip.accommodation ?? null
          });
        }
        setCachedTripRecord(null);
        restoredFromServer = true;
      } catch {
        // Fall back to restoring the saved base below.
      }
    }

    if (!restoredFromServer && currentUserId) {
      const restored = await updateCachedTripItinerary({
        tripId,
        userId: currentUserId,
        itinerary: mutation.baseItinerary
      });
      setCachedTripRecord(restored);
    }

    await offlineSync.refresh();
    clearEditSession();
    setMergeRecovery(null);
    setMergeApplyError(null);
    setItineraryConflictMessage(null);
    setSuccessMessage("Offline itinerary changes discarded.");
  }

  async function completeItinerarySave(updated: Trip, message: string) {
    queryClient.setQueryData(tripKeys.detail(tripId), updated);
    if (currentUserId) {
      await cacheTripSnapshot({
        userId: currentUserId,
        trip: updated,
        budgetSummary: budgetSummaryQuery.data ?? null,
        accommodation: updated.accommodation ?? null
      });
      setCachedTripRecord(null);
      await offlineSync.refresh();
    }
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: tripKeys.itineraryVersions(tripId) }),
      queryClient.invalidateQueries({ queryKey: budgetKeys.summary(tripId) }),
      invalidateBudgetConfidence(),
      queryClient.invalidateQueries({ queryKey: approvalRiskKeys.trip(tripId) }),
      queryClient.invalidateQueries({ queryKey: tripHealthKeys.detail(tripId) }),
      queryClient.invalidateQueries({ queryKey: ["route-estimate", "walking"] }),
      queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) })
    ]);
    await tripQuery.refetch();
    await editLock.release();
    clearEditSession();
    void setPresenceState("viewing");
    setSuccessMessage(message);
  }

  async function invalidateCostSplitDependents(updatedTrip: Trip) {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: tripKeys.detail(tripId) }),
      queryClient.invalidateQueries({ queryKey: tripKeys.itineraryVersions(tripId) }),
      queryClient.invalidateQueries({ queryKey: budgetKeys.summary(tripId) }),
      invalidateBudgetConfidence(),
      queryClient.invalidateQueries({ queryKey: approvalRiskKeys.trip(tripId) }),
      queryClient.invalidateQueries({ queryKey: costSplittingKeys.all }),
      queryClient.invalidateQueries({ queryKey: tripHealthKeys.detail(tripId) }),
      queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) }),
      queryClient.invalidateQueries({ queryKey: tripKeys.lists() })
    ]);
    if (currentUserId) {
      await cacheTripSnapshot({
        userId: currentUserId,
        trip: updatedTrip,
        budgetSummary: budgetSummaryQuery.data ?? cachedBudgetSummary ?? null,
        accommodation: updatedTrip.accommodation ?? null
      });
    }
    await tripQuery.refetch();
  }

  function clearEditSession() {
    setIsEditing(false);
    setDraftItinerary(null);
    setBaseItinerary(null);
    setBaseItineraryRevision(null);
    setMergeRecovery(null);
    setMergeApplyError(null);
  }

  async function regenerateDay(dayNumber: number, instruction?: string) {
    await createRegenerationJob("day_regeneration", { type: "day", dayNumber }, instruction);
  }

  async function regenerateItem(dayNumber: number, itemIndex: number, instruction?: string) {
    await createRegenerationJob(
      "item_regeneration",
      { type: "item", dayNumber, itemIndex },
      instruction
    );
  }

  async function improveDay(dayNumber: number, instruction: string) {
    await createRegenerationJob("quality_improvement_day", { type: "day", dayNumber }, instruction);
  }

  async function improveItem(dayNumber: number, itemIndex: number, instruction: string) {
    await createRegenerationJob(
      "quality_improvement_item",
      { type: "item", dayNumber, itemIndex },
      instruction
    );
  }

  async function createRegenerationJob(
    jobType: GenerationJobType,
    target: RegeneratingTarget,
    instruction: string | undefined
  ) {
    if (hasActiveGenerationJob) {
      return;
    }

    try {
      setRegenerationError(null);
      setSuccessMessage(null);
      const job = await createGenerationJobMutation.mutateAsync({
        jobType,
        expectedItineraryRevision: trip.itineraryRevision,
        instruction,
        dayNumber: target.dayNumber,
        ...(target.type === "item" ? { itemIndex: target.itemIndex } : {})
      });
      handleGenerationJobCreated(job);
      await queryClient.invalidateQueries({ queryKey: generationJobKeys.list(tripId) });
    } catch (error) {
      if (isItineraryConflictError(error)) {
        setRegenerationError("This itinerary changed. Reload latest version before trying again.");
        await tripQuery.refetch();
        return;
      }
      setRegenerationError(
        error instanceof Error ? error.message : "Could not regenerate itinerary."
      );
    }
  }

  function openBudgetOptimization(dayNumber?: number | null) {
    if (!canMutateTrip || !trip.itinerary) {
      return;
    }
    setBudgetOptimizationDefaultDayNumber(dayNumber ?? null);
    setBudgetOptimizationError(null);
    setBudgetOptimizationDialogOpen(true);
  }

  function openTripRepair(repairMode: RepairMode = "policy_compliance") {
    if (!canMutateTrip || !trip.itinerary || !trip.workspaceId) {
      return;
    }
    setTripRepairDefaultMode(repairMode);
    setTripRepairError(null);
    setTripRepairDialogOpen(true);
  }

  function saveCostSplit(split: CostSplitRule) {
    if (!costSplitTarget) {
      return;
    }
    setCostSplitError(null);
    setSuccessMessage(null);
    if (costSplitTarget.type === "item") {
      updateItemCostSplitMutation.mutate({ target: costSplitTarget, split });
      return;
    }
    updateAccommodationCostSplitMutation.mutate(split);
  }

  async function createBudgetOptimization(input: BudgetOptimizationJobRequest) {
    if (hasActiveGenerationJob) {
      setBudgetOptimizationError("Wait for the current generation job to finish.");
      return;
    }

    try {
      setBudgetOptimizationError(null);
      setRegenerationError(null);
      setSuccessMessage(null);
      const job = await createBudgetOptimizationMutation.mutateAsync(input);
      handleGenerationJobCreated(job);
      setBudgetOptimizationDialogOpen(false);
      setSuccessMessage("Budget optimization queued.");
      await queryClient.invalidateQueries({ queryKey: generationJobKeys.list(tripId) });
    } catch (error) {
      if (isItineraryConflictError(error)) {
        setBudgetOptimizationError(
          "This itinerary changed. Reload latest version before optimizing the budget."
        );
        await tripQuery.refetch();
        return;
      }
      setBudgetOptimizationError(
        getErrorMessage(error, "Could not start budget optimization.")
      );
    }
  }

  async function applyBudgetOptimization(proposal: BudgetOptimizationProposal) {
    try {
      setBudgetOptimizationError(null);
      setRegenerationError(null);
      setSuccessMessage(null);
      const result = await applyBudgetOptimizationMutation.mutateAsync(proposal);
      queryClient.setQueryData(tripKeys.detail(tripId), result.trip);
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: tripKeys.detail(tripId) }),
        queryClient.invalidateQueries({ queryKey: tripKeys.itineraryVersions(tripId) }),
        queryClient.invalidateQueries({ queryKey: budgetKeys.summary(tripId) }),
        invalidateBudgetConfidence(),
        queryClient.invalidateQueries({ queryKey: approvalRiskKeys.trip(tripId) }),
        queryClient.invalidateQueries({ queryKey: budgetOptimizationKeys.all(tripId) }),
        queryClient.invalidateQueries({ queryKey: tripHealthKeys.detail(tripId) }),
        queryClient.invalidateQueries({ queryKey: ["route-estimate", "walking"] }),
        queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) }),
        queryClient.invalidateQueries({ queryKey: tripKeys.lists() })
      ]);
      await tripQuery.refetch();
      setSuccessMessage(
        `Budget proposal applied to Day ${proposal.dayNumber ?? proposal.proposal.dayNumber}.`
      );
    } catch (error) {
      if (isItineraryConflictError(error)) {
        setBudgetOptimizationError(
          "This proposal is outdated because the itinerary changed. Generate a new optimization."
        );
        await queryClient.invalidateQueries({ queryKey: budgetOptimizationKeys.all(tripId) });
        await tripQuery.refetch();
        return;
      }
      setBudgetOptimizationError(getErrorMessage(error, "Could not apply proposal."));
    }
  }

  async function discardBudgetOptimization(proposal: BudgetOptimizationProposal) {
    if (!window.confirm("Discard this budget optimization proposal?")) {
      return;
    }

    try {
      setBudgetOptimizationError(null);
      await discardBudgetOptimizationMutation.mutateAsync(proposal);
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: budgetOptimizationKeys.all(tripId) }),
        queryClient.invalidateQueries({ queryKey: tripHealthKeys.detail(tripId) }),
        queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) })
      ]);
      setSuccessMessage("Budget optimization proposal discarded.");
    } catch (error) {
      setBudgetOptimizationError(getErrorMessage(error, "Could not discard proposal."));
    }
  }

  async function createTripRepair(input: CreateRepairJobInput) {
    if (hasActiveGenerationJob) {
      setTripRepairError("Wait for the current generation job to finish.");
      return;
    }

    try {
      setTripRepairError(null);
      setRegenerationError(null);
      setSuccessMessage(null);
      const job = await createTripRepairMutation.mutateAsync(input);
      handleGenerationJobCreated(job);
      setTripRepairDialogOpen(false);
      setSuccessMessage("AI repair queued.");
      await queryClient.invalidateQueries({ queryKey: generationJobKeys.list(tripId) });
    } catch (error) {
      if (isItineraryConflictError(error)) {
        setTripRepairError(
          "This itinerary changed. Reload latest version before repairing the trip."
        );
        await tripQuery.refetch();
        return;
      }
      setTripRepairError(getErrorMessage(error, "Could not start AI repair."));
    }
  }

  async function applyTripRepair(proposal: RepairProposal) {
    try {
      setTripRepairError(null);
      setRegenerationError(null);
      setSuccessMessage(null);
      const result = await applyTripRepairMutation.mutateAsync(proposal);
      queryClient.setQueryData(tripKeys.detail(tripId), result.trip);
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: tripKeys.detail(tripId) }),
        queryClient.invalidateQueries({ queryKey: tripKeys.itineraryVersions(tripId) }),
        queryClient.invalidateQueries({ queryKey: budgetKeys.summary(tripId) }),
        invalidateBudgetConfidence(),
        queryClient.invalidateQueries({ queryKey: approvalRiskKeys.trip(tripId) }),
        queryClient.invalidateQueries({ queryKey: workspacePolicyKeys.evaluation(tripId) }),
        queryClient.invalidateQueries({ queryKey: tripRepairKeys.all(tripId) }),
        queryClient.invalidateQueries({ queryKey: budgetOptimizationKeys.all(tripId) }),
        queryClient.invalidateQueries({ queryKey: tripHealthKeys.detail(tripId) }),
        queryClient.invalidateQueries({ queryKey: ["route-estimate", "walking"] }),
        queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) }),
        queryClient.invalidateQueries({ queryKey: tripKeys.lists() })
      ]);
      await tripQuery.refetch();
      setSuccessMessage("AI repair applied.");
    } catch (error) {
      if (isItineraryConflictError(error)) {
        setTripRepairError(
          "This repair proposal is outdated because the itinerary changed. Generate a new repair."
        );
        await queryClient.invalidateQueries({ queryKey: tripRepairKeys.all(tripId) });
        await tripQuery.refetch();
        return;
      }
      setTripRepairError(getErrorMessage(error, "Could not apply repair proposal."));
    }
  }

  async function discardTripRepair(proposal: RepairProposal) {
    if (!window.confirm("Discard this repair proposal?")) {
      return;
    }

    try {
      setTripRepairError(null);
      await discardTripRepairMutation.mutateAsync(proposal);
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: tripRepairKeys.all(tripId) }),
        queryClient.invalidateQueries({ queryKey: tripHealthKeys.detail(tripId) }),
        queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) })
      ]);
      setSuccessMessage("AI repair proposal discarded.");
    } catch (error) {
      setTripRepairError(getErrorMessage(error, "Could not discard repair proposal."));
    }
  }

  function handleAvailabilityResult(
    dayNumber: number,
    itemIndex: number,
    result: AvailabilitySearchResponse
  ) {
    setAvailabilityResultsByItem((current) => ({
      ...current,
      [availabilityResultKey(dayNumber, itemIndex)]: result
    }));
  }

  async function applyAvailabilityPrice(
    dayNumber: number,
    itemIndex: number,
    option: AvailabilityOption,
    result: AvailabilitySearchResponse
  ) {
    if (!trip.itinerary || !option.price) {
      return;
    }

    const dayIndex = trip.itinerary.days.findIndex(
      (day, index) => (day.day || index + 1) === dayNumber
    );
    const currentItem = dayIndex >= 0 ? trip.itinerary.days[dayIndex]?.items[itemIndex] : null;
    if (!currentItem) {
      setAvailabilityApplyError("Could not find that itinerary item.");
      return;
    }

    const providerName = result.providerDisplayName || result.provider || "availability provider";
    const nextCost: EstimatedCost = {
      amount: option.price.amount,
      currency: option.price.currency,
      category: availabilityCostCategory(currentItem),
      source: "provider",
      confidence: result.match?.matched && result.match.confidence >= 0.8 ? "high" : "medium",
      note: `Applied from ${providerName} availability result. Verify the final price before booking.`,
      // Preserve any cost-splitting rule already configured on this item.
      ...(currentItem.estimatedCost?.split ? { split: currentItem.estimatedCost.split } : {})
    };
    const nextItinerary: Itinerary = {
      ...trip.itinerary,
      days: trip.itinerary.days.map((day, index) => {
        if (index !== dayIndex) {
          return day;
        }
        return {
          ...day,
          items: day.items.map((item, innerIndex) =>
            innerIndex === itemIndex
              ? {
                  ...item,
                  estimatedCost: nextCost,
                  priceEnrichment: {
                    ...(item.priceEnrichment ?? { status: "matched" as const }),
                    status: "matched",
                    provider: result.provider,
                    matchConfidence: result.match?.confidence ?? null,
                    priceType: option.priceType,
                    reviewStatus: "changed",
                    updatedAt: result.checkedAt,
                    reason: "availability_provider_price"
                  },
                  availabilityCheck: {
                    provider: result.provider,
                    status: option.availability ?? result.status,
                    checkedAt: result.checkedAt,
                    matchConfidence: option.matchConfidence ?? result.match?.confidence ?? 0,
                    selectedOptionId: option.id,
                    fallbackUsed: result.fallbackUsed,
                    priceChanged: isSignificantPriceChange(
                      currentItem.estimatedCost,
                      option.price
                    )
                  }
                }
              : item
          )
        };
      })
    };

    try {
      setAvailabilityApplyError(null);
      setRegenerationError(null);
      setSuccessMessage(null);
      const updated = await updateMutation.mutateAsync({
        itinerary: normalizeItineraryForSave(nextItinerary),
        expectedRevision: trip.itineraryRevision
      });
      await completeItinerarySave(updated, "Budget price updated from availability.");
    } catch (error) {
      if (isItineraryConflictError(error)) {
        setAvailabilityApplyError("This itinerary changed. Reload latest version before updating the price.");
        await tripQuery.refetch();
        return;
      }
      setAvailabilityApplyError(getErrorMessage(error, "Could not update budget price."));
    }
  }

  function handleGenerationJobCreated(job: GenerationJob) {
    setActiveGenerationJobId(job.id);
    setSuccessMessage(null);
    setRegenerationError(null);
    queryClient.setQueryData(generationJobKeys.detail(tripId, job.id), job);
  }

  async function handleGenerationJobCompleted(job: GenerationJob) {
    if (job.jobType === "budget_optimization_day") {
      await refreshAfterBudgetOptimizationJob();
      setRegenerationError(null);
      setSuccessMessage("Budget optimization proposal ready.");
      return;
    }
    if (job.jobType === "policy_repair") {
      await refreshAfterTripRepairJob();
      setRegenerationError(null);
      setSuccessMessage("AI repair proposal ready.");
      return;
    }
    await refreshTripAfterGenerationJob();
    setRegenerationError(null);
    setSuccessMessage(successMessageForGenerationJob(job));
  }

  async function handleGenerationJobFailed(job: GenerationJob) {
    await queryClient.invalidateQueries({ queryKey: generationJobKeys.list(tripId) });
    if (job.jobType === "budget_optimization_day") {
      await queryClient.invalidateQueries({ queryKey: budgetOptimizationKeys.all(tripId) });
    }
    if (job.jobType === "policy_repair") {
      await queryClient.invalidateQueries({ queryKey: tripRepairKeys.all(tripId) });
    }
    if (job.errorCode === "itinerary_conflict") {
      await tripQuery.refetch();
    }
    setSuccessMessage(null);
    setRegenerationError(failureMessageForGenerationJob(job));
  }

  async function handleGenerationJobCancelled(job: GenerationJob) {
    await queryClient.invalidateQueries({ queryKey: generationJobKeys.list(tripId) });
    setSuccessMessage("Generation cancelled.");
    setRegenerationError(null);
    setActiveGenerationJobId(job.id);
  }

  async function refreshTripAfterGenerationJob() {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: tripKeys.detail(tripId) }),
      queryClient.invalidateQueries({ queryKey: tripKeys.itineraryVersions(tripId) }),
      queryClient.invalidateQueries({ queryKey: ["route-estimate", "walking"] }),
      queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) }),
      queryClient.invalidateQueries({ queryKey: generationJobKeys.list(tripId) }),
      queryClient.invalidateQueries({ queryKey: budgetOptimizationKeys.all(tripId) }),
      queryClient.invalidateQueries({ queryKey: tripRepairKeys.all(tripId) }),
      queryClient.invalidateQueries({ queryKey: approvalRiskKeys.trip(tripId) }),
      queryClient.invalidateQueries({ queryKey: budgetKeys.summary(tripId) }),
      invalidateBudgetConfidence(),
      queryClient.invalidateQueries({ queryKey: tripHealthKeys.detail(tripId) }),
      queryClient.invalidateQueries({ queryKey: tripKeys.lists() })
    ]);
    await tripQuery.refetch();
  }

  async function refreshAfterBudgetOptimizationJob() {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: budgetOptimizationKeys.all(tripId) }),
      queryClient.invalidateQueries({ queryKey: tripHealthKeys.detail(tripId) }),
      queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) }),
      queryClient.invalidateQueries({ queryKey: generationJobKeys.list(tripId) })
    ]);
  }

  async function refreshAfterTripRepairJob() {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: tripRepairKeys.all(tripId) }),
      queryClient.invalidateQueries({ queryKey: approvalRiskKeys.trip(tripId) }),
      queryClient.invalidateQueries({ queryKey: workspacePolicyKeys.evaluation(tripId) }),
      queryClient.invalidateQueries({ queryKey: tripHealthKeys.detail(tripId) }),
      queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) }),
      queryClient.invalidateQueries({ queryKey: generationJobKeys.list(tripId) })
    ]);
  }

  async function cancelActiveGenerationJob() {
    if (!activeGenerationJob || activeGenerationJob.status !== "queued") {
      return;
    }
    try {
      await cancelGenerationJobMutation.mutateAsync(activeGenerationJob.id);
    } catch (error) {
      setRegenerationError(
        error instanceof Error ? error.message : "Could not cancel generation job."
      );
    }
  }

  async function cancelGenerationJobById(job: GenerationJob) {
    if (job.status !== "queued") {
      return;
    }
    try {
      await cancelGenerationJobMutation.mutateAsync(job.id);
    } catch (error) {
      setRegenerationError(getErrorMessage(error, "Could not cancel generation job."));
    }
  }

  async function handleVersionRestored(updatedTrip: Trip) {
    queryClient.setQueryData(tripKeys.detail(tripId), updatedTrip);
    await queryClient.invalidateQueries({ queryKey: budgetKeys.summary(tripId) });
    await invalidateBudgetConfidence();
    await queryClient.invalidateQueries({ queryKey: approvalRiskKeys.trip(tripId) });
    await queryClient.invalidateQueries({ queryKey: tripHealthKeys.detail(tripId) });
    await queryClient.invalidateQueries({ queryKey: ["route-estimate", "walking"] });
    await tripQuery.refetch();
    setRegenerationError(null);
    setSuccessMessage("Itinerary restored.");
  }

  async function handleOptimizationApplied(updatedTrip: Trip) {
    const optimizedDayNumber = optimizingDayNumber;
    queryClient.setQueryData(tripKeys.detail(tripId), updatedTrip);
    await queryClient.invalidateQueries({ queryKey: tripKeys.itineraryVersions(tripId) });
    await queryClient.invalidateQueries({ queryKey: budgetKeys.summary(tripId) });
    await invalidateBudgetConfidence();
    await queryClient.invalidateQueries({ queryKey: approvalRiskKeys.trip(tripId) });
    await queryClient.invalidateQueries({ queryKey: tripHealthKeys.detail(tripId) });
    await queryClient.invalidateQueries({ queryKey: ["route-estimate", "walking"] });
    await tripQuery.refetch();
    setRegenerationError(null);
    setSuccessMessage(
      optimizedDayNumber != null
        ? `Day ${optimizedDayNumber} order optimized.`
        : "Day order optimized."
    );
  }

  async function handlePlaceReviewUpdated(updatedTrip: Trip) {
    queryClient.setQueryData(tripKeys.detail(tripId), updatedTrip);
    await queryClient.invalidateQueries({ queryKey: tripKeys.itineraryVersions(tripId) });
    await queryClient.invalidateQueries({ queryKey: approvalRiskKeys.trip(tripId) });
    await queryClient.invalidateQueries({ queryKey: tripHealthKeys.detail(tripId) });
    await queryClient.invalidateQueries({ queryKey: ["route-estimate", "walking"] });
    await tripQuery.refetch();
    setRegenerationError(null);
    setSuccessMessage("Place match review saved.");
  }

  function openCommentsForItem(dayNumber: number, itemIndex: number) {
    const day = (trip.itinerary?.days ?? []).find(
      (candidate, index) => (candidate.day || index + 1) === dayNumber
    );
    const item = day?.items?.[itemIndex];
    setCommentTarget({
      dayNumber,
      itemIndex,
      title: item?.name ?? `Item ${itemIndex + 1}`,
      time: item?.time ?? null
    });
  }

  function continueAfterEditLockWarning() {
    setLockWarning(null);
    enterEditMode();
  }

  const heroActions = (
    <>
      <Link
        className="inline-flex h-[42px] items-center gap-2 rounded-full border border-sand-400 bg-white px-[18px] text-[14px] font-medium text-cocoa-700 transition hover:border-sand-600 hover:text-cocoa-900"
        href={`/trips/${trip.id}/today`}
      >
        Today
      </Link>
      {canManageShare ? (
        <a
          href="#sharing"
          className="inline-flex h-[42px] items-center gap-2 rounded-full border border-sand-400 bg-white px-[18px] text-[14px] font-medium text-cocoa-700 transition hover:border-sand-600 hover:text-cocoa-900"
        >
          <ShareNodesIcon className="h-4 w-4" />
          Share
        </a>
      ) : null}
      {exportTrip ? <ExportTripMenu exportTrip={exportTrip} /> : null}
      {canGenerate ? (
        <GenerateItineraryButton
          disabled={hasActiveGenerationJob}
          itineraryRevision={trip.itineraryRevision}
          onJobCreated={handleGenerationJobCreated}
          tripId={trip.id}
        />
      ) : canEditItinerary && !isEditing ? (
        <button
          type="button"
          disabled={editLock.loading}
          onClick={startEditing}
          className="inline-flex h-[42px] items-center gap-2 rounded-full bg-clay px-5 text-[14px] font-semibold text-sand-100 shadow-[0_8px_20px_rgba(192,91,59,0.22)] transition hover:bg-clay-dark disabled:cursor-not-allowed disabled:opacity-60"
        >
          <PencilSquareIcon className="h-4 w-4" />
          {editLock.loading ? "Checking…" : "Edit itinerary"}
        </button>
      ) : null}
    </>
  );

  return (
    <div
      className={cn(
        newsreader.variable,
        instrumentSans.variable,
        "min-h-screen bg-sand-50 font-instrument text-cocoa-700 selection:bg-[#F0D9CC]"
      )}
    >
      <TripDetailChromeHeader />
      <div className="mx-auto max-w-[1360px] px-6 pb-20 pt-9 sm:px-10">
        <OfflineBanner
          cachedAt={cachedTripRecord?.cachedAt}
          className="mb-6"
          conflictCount={tripOfflineMutations.filter((mutation) => mutation.status === "conflict").length}
          failedCount={tripOfflineMutations.filter((mutation) => mutation.status === "failed").length}
          offlineCopy={isUsingCachedTrip}
          online={networkStatus.online}
          pendingCount={tripOfflineMutations.length}
          syncing={offlineSync.syncing}
        />

        <AiAdaptedTripBanner className="mb-6" />

        <TripDetailHeader
          accessSource={access?.source}
          actions={heroActions}
          approvalRisk={
            approvalRiskQuery.data
              ? {
                  status: approvalRiskQuery.data.status,
                  score: approvalRiskQuery.data.score,
                  topReasons: approvalRiskQuery.data.topReasons
                }
              : null
          }
          health={tripHealthQuery.data ?? summaryHealth}
          healthLoading={commandCenterSummaryQuery.isLoading && !summaryHealth}
          trip={trip}
          workspaceName={workspaceName}
        />

        <TripMuteSettings tripId={trip.id} className="mt-5" />

        <div className="mt-8 grid grid-cols-1 gap-8 xl:grid-cols-[224px_minmax(0,1fr)_372px]">
          <TripDetailSidebar
            budgetCurrency={trip.budgetCurrency}
            budgetLoading={onlineActionsEnabled && budgetSummaryQuery.isLoading}
            budgetSummary={budgetSummaryQuery.data ?? cachedBudgetSummary ?? null}
            canMutateTrip={canMutateTrip}
            navigationGroups={commandCenterData?.navigationGroups}
            onOpenBudgetOptimization={openBudgetOptimization}
            optimizationDisabled={
              isEditing || createBudgetOptimizationMutation.isPending || hasActiveGenerationJob
            }
            perPersonAverage={perPersonAverage}
            travelers={activeTripTravelers}
            trip={trip}
            tripId={trip.id}
          />

          <div className="flex min-w-0 flex-col gap-4">
            {pendingOfflineMutation ? (
              <PendingOfflineChangesPanel
                mutation={pendingOfflineMutation}
                online={networkStatus.online}
                onDiscard={discardPendingOfflineChanges}
                onReview={reviewPendingOfflineChanges}
                onSyncNow={offlineSync.syncNow}
                syncing={offlineSync.syncing}
              />
            ) : null}

            {currentUserId &&
            (cachedTripRecord || offlineDataMode || tripOfflineMutations.length > 0) ? (
              <OfflineTripCompanionPanel
                cachedAt={cachedTripRecord?.cachedAt}
                mutations={tripOfflineMutations}
                onDiscard={discardTripOfflineMutation}
                onRefreshOfflineCopy={refreshOfflineCopy}
                onRemoveOfflineCopy={removeOfflineCopy}
                onSyncNow={offlineSync.syncNow}
                online={networkStatus.online}
                syncing={offlineSync.syncing}
                tripId={trip.id}
                userId={currentUserId}
              />
            ) : null}

            {successMessage ? (
              <div className="rounded-[14px] border border-[#DCE8DD] bg-[#F2F7F1] p-4 text-[14px] text-[#38543F]">
                {successMessage}
              </div>
            ) : null}

            {deepLinkMessage ? (
              <div className="rounded-[14px] border border-[#EAD9B8] bg-[#FDF7E8] p-4 text-[14px] text-[#7A5727]" role="status">
                {deepLinkMessage}
              </div>
            ) : null}

            {activeGenerationJob ? (
              <GenerationJobStatusCard
                canCancel={
                  activeGenerationJob.status === "queued" &&
                  (activeGenerationJob.requestedByUserId === currentUserId ||
                    access?.role === "owner")
                }
                isCancelling={cancelGenerationJobMutation.isPending}
                job={activeGenerationJob}
                onCancel={cancelActiveGenerationJob}
              />
            ) : null}

            {regenerationError ? (
              <div className="rounded-[14px] border border-[#E5C3B6] bg-[#FBF0EB] p-4 text-[14px] text-[#B3402E]">
                {regenerationError}
              </div>
            ) : null}

            {availabilityApplyError ? (
              <div className="rounded-[14px] border border-[#E5C3B6] bg-[#FBF0EB] p-4 text-[14px] text-[#B3402E]">
                {availabilityApplyError}
              </div>
            ) : null}

            {commandCenterSummaryQuery.isLoading && onlineActionsEnabled ? (
              <CommandCenterSkeleton />
            ) : commandCenterData ? (
              <TripCommandCenter
                approval={tripApprovalQuery.data ?? null}
                data={commandCenterData}
                health={tripHealthQuery.data ?? summaryHealth}
                offlineStatus={commandCenterOfflineStatus}
                onSyncNow={offlineSync.syncNow}
                syncing={offlineSync.syncing}
                trip={trip}
                verification={tripVerificationQuery.data ?? null}
                workspaceName={workspaceName}
                setupChecklist={{
                  checklistExists: Boolean(
                    checklistQuery.data?.checklist ||
                    (commandCenterSummaryQuery.data?.checklist?.totalCount ?? 0) > 0
                  ),
                  collaboratorCount: Math.max(
                    0,
                    (groupReadinessQuery.data?.members.length ??
                      commandCenterSummaryQuery.data?.groupReadiness?.memberCount ??
                      1) - 1
                  ),
                  healthLoaded: Boolean(tripHealthQuery.data ?? summaryHealth),
                  healthHasCriticalIssues: Boolean(
                    (tripHealthQuery.data ?? summaryHealth)?.issues.some(
                      (issue) => issue.status === "open" && issue.severity === "critical"
                    )
                  )
                }}
              />
            ) : null}

            {canUsePrivateCollaboration && onlineActionsEnabled ? (
              <DeferredSection active={sectionEnabled("health")} section="health">
                <TripHealthPanel
                  error={tripHealthQuery.error instanceof Error ? tripHealthQuery.error : null}
                  health={tripHealthQuery.data ?? summaryHealth}
                  loading={tripHealthQuery.isLoading}
                  onRetry={() => void tripHealthQuery.refetch()}
                  retrying={tripHealthQuery.isFetching}
                />
              </DeferredSection>
            ) : null}

            {canUsePrivateCollaboration && onlineActionsEnabled ? (
              <DeferredSection active={sectionEnabled("verification", "health", "route")} section="verification">
                <VerificationPanel readiness={tripVerificationQuery.data ?? null} />
              </DeferredSection>
            ) : null}

            {canUsePrivateCollaboration && onlineActionsEnabled ? (
              <DeferredSection
                active={sectionEnabled("group-readiness")}
                section="group-readiness"
              >
                <GroupReadinessPanel
                  canNudge={Boolean(tripAccess?.canEdit ?? true) && onlineActionsEnabled}
                  error={groupReadinessQuery.error instanceof Error ? groupReadinessQuery.error : null}
                  loading={groupReadinessQuery.isLoading}
                  onRetry={() => void groupReadinessQuery.refetch()}
                  readiness={groupReadinessQuery.data ?? null}
                  retrying={groupReadinessQuery.isFetching}
                  tripId={trip.id}
                />
              </DeferredSection>
            ) : null}

            {canUsePrivateCollaboration ? (
              <>
                <DeferredSection active={sectionEnabled("dates")} section="dates">
                  <AvailabilityPanel
                    canEdit={canMutateTrip}
                    currentUserId={currentUserId}
                    online={onlineActionsEnabled}
                    onGenerationJobCreated={handleGenerationJobCreated}
                    trip={trip}
                  />
                </DeferredSection>
                <DeferredSection active={sectionEnabled("decisions")} section="decisions">
                  <PollsPanel
                    canCreate={canCreatePoll}
                    online={onlineActionsEnabled}
                    tripId={trip.id}
                  />
                  <GroupPreferencesPanel
                    enabled={decisionsEnabled}
                    tripId={trip.id}
                  />
                </DeferredSection>
              </>
            ) : null}

            {trip.status === "PROCESSING" ? (
              <div className="rounded-[14px] border border-[#EAD9B8] bg-[#FDF0E3] p-6 text-[14px] text-[#96682A]">
                The itinerary is being generated. This page will refresh while processing.
              </div>
            ) : null}

            {trip.status === "COMPLETED" && trip.itinerary ? (
              <div className="flex flex-col gap-4" data-load-section="itinerary">
                <section
                  id="route"
                  className="scroll-mt-24 space-y-4"
                  data-load-section="route"
                >
                  <div className="space-y-3">
                    <RouteBuilderPanel
                      canEdit={canEditTripAccess}
                      health={tripHealthQuery.data ?? null}
                      online={onlineActionsEnabled}
                      trip={trip}
                    />
                    {canMutateTrip ? (
                      <div className="flex justify-end">
                        <Button
                          type="button"
                          variant="secondary"
                          onClick={() => setRouteAlternativesOpen((open) => !open)}
                        >
                          {routeAlternativesOpen ? "Hide route options" : "Find better routes"}
                        </Button>
                      </div>
                    ) : null}
                  </div>
                  {routeAlternativesOpen ? (
                    <RouteAlternativesPanel
                      trip={trip}
                      canApply={canMutateTrip}
                      canCreatePoll={canCreatePoll}
                      onRouteApplied={(updatedTrip) => {
                        setSuccessMessage("Route alternative applied.");
                        queryClient.setQueryData(tripKeys.detail(trip.id), updatedTrip);
                        void queryClient.invalidateQueries({
                          queryKey: tripHealthKeys.detail(trip.id)
                        });
                        void queryClient.invalidateQueries({
                          queryKey: queryKeys.trip.commandCenter(trip.id)
                        });
                        setRouteAlternativesOpen(false);
                      }}
                    />
                  ) : null}
                </section>
                <TripQualityChecks
                availabilityResultsByItem={availabilityResultsByItem}
                budgetSummary={budgetSummaryQuery.data ?? cachedBudgetSummary ?? null}
                fallbackDistanceSummaries={fallbackDistanceSummaries}
                isEditing={isEditing}
                isImproving={createGenerationJobMutation.isPending || hasActiveGenerationJob}
                isOptimizingBudget={
                  createBudgetOptimizationMutation.isPending || hasActiveGenerationJob
                }
                maxWalkingKmPerDay={maxWalkingKmPerDay}
                onImproveDay={canMutateTrip ? improveDay : undefined}
                onImproveItem={canMutateTrip ? improveItem : undefined}
                onOptimizeDayForBudget={canMutateTrip ? openBudgetOptimization : undefined}
                routeEstimatesByDay={routeEstimatesByDay}
                trip={trip}
                weatherForecast={weatherForecastQuery.data ?? null}
              />

              <DeferredSection active={sectionEnabled("checklist")} section="checklist">
                <TripChecklistPanel
                  canCheck={canUsePrivateCollaboration}
                  canEdit={canEditTripAccess}
                  currentUserId={currentUserId}
                  enabled={canUsePrivateCollaboration}
                  offline={offlineDataMode}
                  tripId={trip.id}
                  userId={currentUserId}
                />
              </DeferredSection>

              <DeferredSection active={sectionEnabled("reminders")} section="reminders">
                <TripRemindersPanel
                  canEdit={canEditTripAccess}
                  currentUserId={currentUserId}
                  enabled={canUsePrivateCollaboration}
                  offline={offlineDataMode}
                  tripId={trip.id}
                  userId={currentUserId}
                />
              </DeferredSection>

              <BudgetOptimizationProposalsPanel
                canMutate={canMutateTrip}
                currentItinerary={trip.itinerary}
                error={budgetOptimizationError}
                isApplying={applyBudgetOptimizationMutation.isPending}
                isDiscarding={discardBudgetOptimizationMutation.isPending}
                isLoading={budgetOptimizationProposalsQuery.isLoading}
                onApply={applyBudgetOptimization}
                onDiscard={discardBudgetOptimization}
                proposals={budgetOptimizationProposalsQuery.data ?? []}
              />

              {trip.workspaceId ? (
                <RepairProposalsPanel
                  activeJob={activeTripRepairJob}
                  canMutate={canMutateTrip}
                  currentItinerary={trip.itinerary}
                  error={tripRepairError}
                  isApplying={applyTripRepairMutation.isPending}
                  isCancellingJob={cancelGenerationJobMutation.isPending}
                  isDiscarding={discardTripRepairMutation.isPending}
                  isLoading={tripRepairProposalsQuery.isLoading}
                  onApply={applyTripRepair}
                  onCancelJob={cancelGenerationJobById}
                  onCreateRepair={canMutateTrip ? () => openTripRepair() : undefined}
                  onDiscard={discardTripRepair}
                  proposals={tripRepairProposalsQuery.data ?? []}
                  tripId={trip.id}
                />
              ) : null}

              <div id="cost-split" className="scroll-mt-24">
                <CostSplittingPanel
                  canEdit={canMutateTrip}
                  offline={offlineDataMode}
                  onEditAccommodationSplit={
                    trip.accommodation?.estimatedCost?.amount != null
                      ? () => setCostSplitTarget({ type: "accommodation" })
                      : undefined
                  }
                  onEditItemSplit={(dayNumber, itemIndex) =>
                    setCostSplitTarget({ type: "item", dayNumber, itemIndex })
                  }
                  summary={costSplittingSummary}
                  summaryLoading={costSplittingSummaryQuery.isLoading}
                  travelers={tripTravelersQuery.data?.travelers ?? []}
                  travelersLoading={tripTravelersQuery.isLoading}
                  trip={trip}
                />
              </div>

              <PresenceEditingWarning
                currentUserId={currentUserId}
                snapshot={presenceStream.snapshot}
              />
              <EditLockStatus lock={editLock.lock} />
              {editLock.error ? (
                <div className="rounded-[14px] border border-[#E5C3B6] bg-[#FBF0EB] p-4 text-[14px] text-[#B3402E]">
                  {editLock.error}
                </div>
              ) : null}

              {isEditing && draftItinerary ? (
                <>
                  {editingRevisionChanged ? (
                    <div className="rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900">
                      This itinerary has newer changes. Saving may require merge.
                    </div>
                  ) : null}
                  {offlineDataMode ? (
                    <div className="rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900">
                      Offline edit: changes will sync when you are back online.
                    </div>
                  ) : null}
                  {itineraryConflictMessage && !mergeRecovery ? (
                    <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-800">
                      <h2 className="font-semibold">
                        This itinerary changed while you were editing
                      </h2>
                      <p className="mt-1 leading-6">
                        Someone else saved changes before you. To avoid overwriting their
                        work, reload the latest itinerary before editing again.
                      </p>
                      <div className="mt-4 flex flex-col gap-2 sm:flex-row">
                        <Button
                          disabled={tripQuery.isFetching}
                          onClick={reloadLatestAfterConflict}
                          type="button"
                        >
                          Reload latest
                        </Button>
                        <Button
                          disabled={tripQuery.isFetching}
                          onClick={cancelLocalChangesAfterConflict}
                          type="button"
                          variant="secondary"
                        >
                          Cancel my changes
                        </Button>
                      </div>
                    </div>
                  ) : null}
                  <div className="flex flex-col gap-3 rounded-lg border border-slate-200 bg-white p-4 sm:flex-row sm:items-center sm:justify-end">
                    <Button
                      disabled={updateMutation.isPending}
                      onClick={cancelEditing}
                      type="button"
                      variant="secondary"
                    >
                      Cancel
                    </Button>
                    <Button
                      disabled={updateMutation.isPending}
                      onClick={saveItinerary}
                      type="button"
                    >
                      {updateMutation.isPending ? "Saving..." : "Save"}
                    </Button>
                  </div>
                  <ItineraryEditor
                    destination={trip.destination}
                    disabled={updateMutation.isPending}
                    errors={editorErrors}
                    itinerary={draftItinerary}
                    onChange={setDraftItinerary}
                    startDate={trip.startDate}
                  />
                  <div className="rounded-lg border border-slate-200 bg-white p-4 text-sm text-slate-600">
                    Map view and distance estimates are available after saving or leaving edit
                    mode.
                  </div>
                </>
              ) : (
                <>
                  <PlaceEnrichmentReviewPanel
                    readOnly={!canMutateTrip || offlineDataMode}
                    onTripUpdated={handlePlaceReviewUpdated}
                    trip={trip}
                  />
                  <OpeningHoursWarnings itinerary={trip.itinerary} startDate={trip.startDate} />
                  {canComment ? <TripCommentsSummary counts={commentCounts} /> : null}
                  <ItineraryTimeline
                    comments={
                      canComment
                        ? {
                            countByKey: commentCountMap,
                            disabled: createGenerationJobMutation.isPending || hasActiveGenerationJob,
                            onOpenItem: openCommentsForItem
                          }
                        : undefined
                    }
                    currency={trip.budgetCurrency}
                    disabled={createGenerationJobMutation.isPending || hasActiveGenerationJob}
                    itinerary={trip.itinerary}
                    onApplyAvailabilityPrice={
                      canMutateTrip ? applyAvailabilityPrice : undefined
                    }
                    onAvailabilityResult={handleAvailabilityResult}
                    onOpenCostSplit={
                      canMutateTrip
                        ? (dayNumber, itemIndex) =>
                            setCostSplitTarget({ type: "item", dayNumber, itemIndex })
                        : undefined
                    }
                    onRegenerateDay={canMutateTrip ? regenerateDay : undefined}
                    onRegenerateItem={canMutateTrip ? regenerateItem : undefined}
                    reactionSummaries={reactionSummaryMap}
                    regeneratingTarget={activeRegeneratingTarget}
                    startDate={trip.startDate}
                    trip={trip}
                  />
                  <DistanceSummary
                    accommodation={trip.accommodation ?? null}
                    itinerary={trip.itinerary}
                    maxWalkingKmPerDay={maxWalkingKmPerDay}
                    onOptimizeDay={canMutateTrip ? setOptimizingDayNumber : undefined}
                  />
                  {onlineActionsEnabled ? (
                    <ItineraryVersionHistory
                      canRestore={canRestoreVersion}
                      currency={trip.budgetCurrency}
                      itineraryRevision={trip.itineraryRevision}
                      onRestored={handleVersionRestored}
                      restoreDisabled={isEditing || !canRestoreVersion}
                      tripId={trip.id}
                    />
                  ) : null}
                  {trip.itinerary && optimizingDay ? (
                    <OptimizeDayOrderDialog
                      day={optimizingDay}
                      expectedItineraryRevision={trip.itineraryRevision}
                      itinerary={trip.itinerary}
                      onApplied={handleOptimizationApplied}
                      onClose={() => setOptimizingDayNumber(null)}
                      open
                      tripId={trip.id}
                    />
                  ) : null}
                  {commentTarget ? (
                    <ItemCommentsPanel
                      canComment={canComment}
                      currentUserId={currentUserId}
                      dayNumber={commentTarget.dayNumber}
                      itemIndex={commentTarget.itemIndex}
                      itemTime={commentTarget.time}
                      itemTitle={commentTarget.title}
                      onOpenChange={(open) => {
                        if (!open) {
                          setCommentTarget(null);
                        }
                      }}
                      open
                      tripId={trip.id}
                    />
                  ) : null}
                </>
              )}
            </div>
          ) : null}

            {trip.status === "COMPLETED" && !trip.itinerary ? (
              <ErrorState
                className="rounded-[18px]"
                description={errorsT("itineraryGenerationDescription")}
                title={errorsT("itineraryGenerationTitle")}
              />
            ) : null}

            {(trip.status === "DRAFT" || trip.status === "FAILED") && !trip.itinerary ? (
              <div className="flex flex-col gap-4">
                <section
                  id="route"
                  className="scroll-mt-24 space-y-4"
                  data-load-section="route"
                >
                  <div className="space-y-3">
                    <RouteBuilderPanel
                      canEdit={canEditTripAccess}
                      health={tripHealthQuery.data ?? null}
                      online={onlineActionsEnabled}
                      trip={trip}
                    />
                    {canMutateTrip ? (
                      <div className="flex justify-end">
                        <Button
                          type="button"
                          variant="secondary"
                          onClick={() => setRouteAlternativesOpen((open) => !open)}
                        >
                          {routeAlternativesOpen ? "Hide route options" : "Find better routes"}
                        </Button>
                      </div>
                    ) : null}
                  </div>
                  {routeAlternativesOpen ? (
                    <RouteAlternativesPanel
                      trip={trip}
                      canApply={canMutateTrip}
                      canCreatePoll={canCreatePoll}
                      onRouteApplied={(updatedTrip) => {
                        setSuccessMessage("Route alternative applied.");
                        queryClient.setQueryData(tripKeys.detail(trip.id), updatedTrip);
                        void queryClient.invalidateQueries({
                          queryKey: tripHealthKeys.detail(trip.id)
                        });
                        void queryClient.invalidateQueries({
                          queryKey: queryKeys.trip.commandCenter(trip.id)
                        });
                        setRouteAlternativesOpen(false);
                      }}
                    />
                  ) : null}
                </section>
                <EmptyState
                  className="rounded-[18px] border-sand-300 bg-white"
                  description={emptyItineraryT("description")}
                  primaryAction={
                    canGenerate
                      ? {
                          label: emptyItineraryT("action"),
                          onClick: () =>
                            document
                              .querySelector<HTMLButtonElement>("[data-generate-itinerary]")
                              ?.click()
                        }
                      : undefined
                  }
                  title={emptyItineraryT("title")}
                />
              </div>
            ) : null}

            {(trip.status === "DRAFT" || trip.status === "FAILED") && trip.itinerary ? (
              <ItineraryTimeline
                currency={trip.budgetCurrency}
                itinerary={trip.itinerary}
                startDate={trip.startDate}
              />
            ) : null}

            {/* Trip tools: interactive panels relocated from the old sidebar. They
                retain their existing styling and full logic; the warm summary cards
                in the left rail and hero deep-link here. */}
            <section className="mt-2 flex flex-col gap-4 border-t border-sand-300 pt-6">
              <h2 className="font-newsreader text-[22px] font-semibold tracking-[-0.01em] text-cocoa-900">
                Trip tools
              </h2>
              <DeferredSection active={sectionEnabled("budget")} section="budget">
                <div id="budget" className="scroll-mt-24">
                  <BudgetPanel
                    canEdit={canMutateTrip}
                    offline={offlineDataMode}
                    offlineSummary={budgetSummaryQuery.data ?? cachedBudgetSummary ?? null}
                    onOpenBudgetOptimization={openBudgetOptimization}
                    optimizationDisabled={
                      isEditing || createBudgetOptimizationMutation.isPending || hasActiveGenerationJob
                    }
                    perPersonAverage={perPersonAverage}
                    trip={trip}
                  />
                </div>
              </DeferredSection>
              <DeferredSection active={sectionEnabled("expenses")} section="expenses">
                <ExpensesPanel
                  canEdit={canEditTripAccess}
                  currentUserId={currentUserId}
                  offline={offlineDataMode}
                  travelers={tripTravelersQuery.data?.travelers ?? []}
                  trip={trip}
                />
              </DeferredSection>
              <AccommodationPanel
                canEdit={canMutateTrip}
                onOpenCostSplit={
                  canMutateTrip && trip.accommodation?.estimatedCost?.amount != null
                    ? () => setCostSplitTarget({ type: "accommodation" })
                    : undefined
                }
                trip={trip}
              />
              {canSaveTemplate ? (
                <div>
                  <button
                    type="button"
                    onClick={() => setSaveTemplateOpen(true)}
                    className="inline-flex h-[42px] items-center gap-2 rounded-full border border-sand-400 bg-white px-[18px] text-[14px] font-medium text-cocoa-700 transition hover:border-sand-600 hover:text-cocoa-900"
                  >
                    Save as template
                  </button>
                </div>
              ) : null}
              {presenceEnabled ? (
                <TripPresenceIndicator
                  currentUserId={currentUserId}
                  isConnected={presenceStream.isConnected}
                  snapshot={presenceStream.snapshot}
                />
              ) : null}
              {trip.workspaceId ? (
                <>
                  <p className="rounded-[14px] border border-sand-300 bg-sand-50 p-4 text-[13.5px] text-cocoa-500">
                    Workspace policy is used as AI guidance for generation, regeneration, and
                    adaptation. Review the authoritative policy check below.
                  </p>
                  <TripPolicyPanel tripId={trip.id} />
                  <TripApprovalPanel
                    onOpenTripRepair={canMutateTrip ? openTripRepair : undefined}
                    tripId={trip.id}
                  />
                </>
              ) : null}
              {onlineActionsEnabled && trip.status === "COMPLETED" && trip.itinerary ? (
                <CalendarSyncPanel canSync={canSyncCalendar} trip={trip} />
              ) : null}
              <div
                id="sharing"
                className="flex scroll-mt-24 flex-col gap-4"
                data-load-section="sharing"
              >
                {canManageShare ? <ShareTripPanel tripId={trip.id} /> : null}
                {onlineActionsEnabled ? (
                  <>
                    {trip.workspaceId ? (
                      <div className="rounded-[14px] border border-sand-300 bg-sand-50 p-4 text-[13.5px] leading-[1.6] text-cocoa-500">
                        Workspace members may already have access. Trip-specific collaborators can
                        still be invited for exceptions.
                      </div>
                    ) : null}
                    <CollaboratorsPanel
                      canManageCollaborators={canManageCollaborators}
                      tripId={trip.id}
                    />
                  </>
                ) : null}
              </div>
            </section>
          </div>

          <div className="flex flex-col gap-5 xl:sticky xl:top-[84px] xl:self-start">
            {trip.status === "COMPLETED" && trip.itinerary && !isEditing ? (
              <RightRailMap
                accommodation={trip.accommodation ?? null}
                itinerary={trip.itinerary}
                route={trip.route}
                startDate={trip.startDate}
              />
            ) : null}
            <RightRailWeather
              days={trip.days}
              destination={trip.destination}
              offline={!networkStatus.online || isUsingCachedTrip}
              startDate={trip.startDate}
            />
            <DeferredSection active={sectionEnabled("activity")} section="activity">
              <RightRailActivity
                canViewActivity={canComment}
                currentUserId={currentUserId}
                tripId={trip.id}
              />
            </DeferredSection>
          </div>
        </div>
      {canUsePrivateCollaboration && onlineActionsEnabled ? (
        <TripCopilot
          currentPath={`/trips/${trip.id}`}
          currentTab={searchParams?.get("tab") ?? "overview"}
          tripId={trip.id}
        />
      ) : null}
      {lockWarning ? (
        <SoftEditLockWarningDialog
          lock={lockWarning}
          onCancel={() => setLockWarning(null)}
          onContinue={continueAfterEditLockWarning}
        />
      ) : null}
      {mergeRecovery ? (
        <MergeConflictDialog
          applying={updateMutation.isPending}
          description={
            mergeRecovery.offlineMutation
              ? "Review your offline draft against the latest saved itinerary before syncing."
              : undefined
          }
          error={mergeApplyError}
          latestRevision={mergeRecovery.latestTrip.itineraryRevision}
          mergeResult={mergeRecovery.mergeResult}
          onApplyMerged={applyMergeRecovery}
          onCancel={() => {
            setMergeRecovery(null);
            setMergeApplyError(null);
          }}
          onDiscardLocal={cancelLocalChangesAfterConflict}
          onResolutionChange={updateConflictResolution}
          onViewLatest={viewLatestFromMerge}
          resolutions={mergeRecovery.resolutions}
          title={
            mergeRecovery.offlineMutation
              ? "This trip changed while you were offline"
              : undefined
          }
        />
      ) : null}
      <BudgetOptimizationRequestDialog
        budgetSummary={budgetSummaryQuery.data ?? cachedBudgetSummary ?? null}
        defaultDayNumber={budgetOptimizationDefaultDayNumber}
        disabled={createBudgetOptimizationMutation.isPending}
        error={budgetOptimizationError}
        onClose={() => setBudgetOptimizationDialogOpen(false)}
        onSubmit={createBudgetOptimization}
        open={budgetOptimizationDialogOpen}
        trip={trip}
      />
      <CreateRepairJobDialog
        approvalRisk={approvalRiskQuery.data ?? null}
        defaultRepairMode={tripRepairDefaultMode}
        disabled={createTripRepairMutation.isPending}
        error={tripRepairError}
        onClose={() => setTripRepairDialogOpen(false)}
        onSubmit={createTripRepair}
        open={tripRepairDialogOpen}
        trip={trip}
      />
      <SaveTripAsTemplateDialog
        onClose={() => setSaveTemplateOpen(false)}
        onSaved={(template) => {
          setSuccessMessage(`Template saved: ${template.title}`);
        }}
        open={saveTemplateOpen}
        trip={trip}
      />
      {costSplitTargetDetails ? (
        <CostSplitRuleEditor
          costAmount={costSplitTargetDetails.amount}
          costCurrency={costSplitTargetDetails.currency}
          currentSplit={costSplitTargetDetails.currentSplit}
          error={costSplitError}
          isSaving={
            updateItemCostSplitMutation.isPending ||
            updateAccommodationCostSplitMutation.isPending
          }
          onClose={() => {
            setCostSplitTarget(null);
            setCostSplitError(null);
          }}
          onSave={saveCostSplit}
          open={Boolean(costSplitTarget)}
          title={costSplitTargetDetails.title}
          travelers={activeTripTravelers}
        />
      ) : null}
      </div>
    </div>
  );
}

/**
 * Warm chrome shell for the Trip Detail loading/error states so they keep the
 * redesigned header (AppHeader is suppressed for this route) instead of rendering
 * bare content.
 */
function DetailShell({ children }: { children: ReactNode }) {
  return (
    <div
      className={cn(
        newsreader.variable,
        instrumentSans.variable,
        "min-h-screen bg-sand-50 font-instrument text-cocoa-700 selection:bg-[#F0D9CC]"
      )}
    >
      <TripDetailChromeHeader />
      <div className="mx-auto max-w-[1360px] px-6 pb-20 pt-9 sm:px-10">{children}</div>
    </div>
  );
}

function BackToTripsLink() {
  return (
    <Link
      href="/trips"
      className="mt-5 inline-flex h-[42px] items-center rounded-full border border-sand-400 bg-white px-5 text-[14px] font-medium text-cocoa-700 transition hover:border-sand-600 hover:text-cocoa-900"
    >
      Back to trips
    </Link>
  );
}

function DeferredSection({
  active,
  children,
  section
}: {
  active: boolean;
  children: ReactNode;
  section: string;
}) {
  return (
    <div data-load-section={section}>
      {active ? (
        children
      ) : (
        <div
          id={section}
          aria-label={`Loading ${section.replaceAll("-", " ")}`}
          className="min-h-36 animate-pulse scroll-mt-24 rounded-[18px] border border-sand-300 bg-white p-5"
        >
          <div className="h-4 w-32 rounded-full bg-sand-200" />
          <div className="mt-4 h-5 w-2/3 rounded-full bg-sand-100" />
          <div className="mt-3 h-4 w-full rounded-full bg-sand-100" />
        </div>
      )}
    </div>
  );
}

function PanelLoading() {
  return (
    <div className="min-h-32 animate-pulse rounded-[18px] border border-sand-300 bg-white p-5">
      <div className="h-4 w-32 rounded-full bg-sand-200" />
      <div className="mt-4 h-5 w-2/3 rounded-full bg-sand-100" />
      <div className="mt-3 h-4 w-full rounded-full bg-sand-100" />
    </div>
  );
}
