"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { useParams } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { AccommodationPanel } from "@/components/accommodation/AccommodationPanel";
import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { useAuth } from "@/components/auth/AuthProvider";
import { ActivityFeed } from "@/components/activity/ActivityFeed";
import { BudgetOptimizationProposalCard } from "@/components/budget-optimization/BudgetOptimizationProposalCard";
import { BudgetOptimizationRequestDialog } from "@/components/budget-optimization/BudgetOptimizationRequestDialog";
import { CalendarSyncPanel } from "@/components/calendar/CalendarSyncPanel";
import { EditLockStatus } from "@/components/edit-locks/EditLockStatus";
import { SoftEditLockWarningDialog } from "@/components/edit-locks/SoftEditLockWarningDialog";
import { ExportTripMenu } from "@/components/export/ExportTripMenu";
import { GenerationJobStatusCard } from "@/components/generation-jobs/GenerationJobStatusCard";
import { MergeConflictDialog } from "@/components/itinerary/merge/MergeConflictDialog";
import { ItemCommentsPanel } from "@/components/comments/ItemCommentsPanel";
import { TripCommentsSummary } from "@/components/comments/TripCommentsSummary";
import { PageContainer } from "@/components/layout/PageContainer";
import { OfflineBanner } from "@/components/offline/OfflineBanner";
import { PendingOfflineChangesPanel } from "@/components/offline/PendingOfflineChangesPanel";
import {
  PresenceEditingWarning,
  TripPresenceIndicator
} from "@/components/presence/TripPresenceIndicator";
import { CollaboratorsPanel } from "@/components/trips/CollaboratorsPanel";
import { GenerateItineraryButton } from "@/components/trips/GenerateItineraryButton";
import {
  ItineraryEditor,
  normalizeItineraryForSave,
  prepareItineraryForEdit,
  validateEditableItinerary
} from "@/components/trips/ItineraryEditor";
import { BudgetPanel } from "@/components/budget/BudgetPanel";
import { DistanceSummary } from "@/components/trips/DistanceSummary";
import { ItineraryMap } from "@/components/trips/ItineraryMap";
import { OpeningHoursWarnings } from "@/components/trips/OpeningHoursWarnings";
import { OptimizeDayOrderDialog } from "@/components/trips/OptimizeDayOrderDialog";
import { PlaceEnrichmentReviewPanel } from "@/components/trips/PlaceEnrichmentReviewPanel";
import { ShareTripPanel } from "@/components/trips/ShareTripPanel";
import { TripQualityChecks } from "@/components/trips/TripQualityChecks";
import { ItineraryVersionHistory } from "@/components/trips/ItineraryVersionHistory";
import { ItineraryView, type RegeneratingTarget } from "@/components/trips/ItineraryView";
import { TripStatusBadge } from "@/components/trips/TripStatusBadge";
import { WeatherForecastCard } from "@/components/trips/WeatherForecastCard";
import { Button, buttonStyles } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { activityKeys } from "@/lib/api/activity";
import { useTripActivityStream } from "@/lib/activity/use-trip-activity-stream";
import { budgetKeys, getTripBudgetSummary } from "@/lib/api/budget";
import {
  applyBudgetOptimizationProposal,
  budgetOptimizationKeys,
  createBudgetOptimizationJob,
  discardBudgetOptimizationProposal
} from "@/lib/api/budget-optimization";
import { commentKeys, listTripCommentCounts } from "@/lib/api/comments";
import { isItineraryConflictError } from "@/lib/api/client";
import {
  cancelGenerationJob,
  createGenerationJob,
  generationJobKeys,
  listGenerationJobs
} from "@/lib/api/generation-jobs";
import { buildCommentCountMap } from "@/lib/comments/comment-counts";
import { getWeatherForecast, weatherKeys } from "@/lib/api/weather";
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
import { useRouteEstimates } from "@/lib/hooks/useRouteEstimates";
import { useBudgetOptimizationProposals } from "@/lib/hooks/useBudgetOptimizationProposals";
import { useGenerationJob } from "@/lib/hooks/useGenerationJob";
import { useNetworkStatus } from "@/hooks/useNetworkStatus";
import { useOfflineSync } from "@/hooks/useOfflineSync";
import { getDayDistanceSummaries } from "@/lib/itinerary/distance-utils";
import { applyConflictResolutions, mergeItineraries } from "@/lib/itinerary/diff-merge/merge";
import { cloneItinerary } from "@/lib/itinerary/diff-merge/normalize";
import { useTripEditLock } from "@/lib/edit-locks/use-trip-edit-lock";
import { isOfflineLikeError } from "@/lib/offline/network";
import {
  cacheTripSnapshot,
  getCachedTrip,
  updateCachedTripItinerary
} from "@/lib/offline/trip-cache";
import { recordPwaEngagement } from "@/lib/pwa/pwa-detection";
import {
  discardMutation,
  enqueueItineraryUpdate,
  markMutationSynced
} from "@/lib/offline/sync-queue";
import { useTripPresenceState } from "@/lib/presence/use-trip-presence-state";
import { useTripPresenceStream } from "@/lib/presence/use-trip-presence-stream";
import {
  formatBudget,
  formatDate,
  getErrorMessage,
  formatInterestLabel,
  formatPaceLabel
} from "@/lib/utils";
import type {
  BudgetOptimizationJobRequest,
  BudgetOptimizationProposal
} from "@/types/budget-optimization";
import type { EstimatedCost } from "@/types/budget";
import type {
  AvailabilityOption,
  AvailabilityResultByItem,
  AvailabilitySearchResponse
} from "@/types/availability";
import type { RouteEstimate } from "@/types/route";
import type { EditLockView } from "@/types/edit-locks";
import type {
  CreateGenerationJobRequest,
  GenerationJob,
  GenerationJobType
} from "@/types/generation-jobs";
import type {
  ConflictResolution,
  ConflictResolutionMap,
  ItineraryMergeResult
} from "@/lib/itinerary/diff-merge/types";
import type {
  CachedTripRecord,
  PendingItineraryMutation,
  SyncResult
} from "@/lib/offline/types";
import type { Itinerary, Trip } from "@/types/trip";

type MergeRecoveryState = {
  latestTrip: Trip;
  mergeResult: ItineraryMergeResult;
  resolutions: ConflictResolutionMap;
  offlineMutation?: PendingItineraryMutation;
};

export default function TripDetailPage() {
  return (
    <ProtectedRoute>
      <TripDetailPageContent />
    </ProtectedRoute>
  );
}

function TripDetailPageContent() {
  const params = useParams<{ id: string }>();
  const tripId = params.id;
  const queryClient = useQueryClient();
  const { user } = useAuth();
  const currentUserId = user?.id;
  const networkStatus = useNetworkStatus();
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
  const [regenerationError, setRegenerationError] = useState<string | null>(null);
  const [activeGenerationJobId, setActiveGenerationJobId] = useState<string | null>(null);
  const [optimizingDayNumber, setOptimizingDayNumber] = useState<number | null>(null);
  const [budgetOptimizationDialogOpen, setBudgetOptimizationDialogOpen] = useState(false);
  const [budgetOptimizationDefaultDayNumber, setBudgetOptimizationDefaultDayNumber] = useState<
    number | null
  >(null);
  const [budgetOptimizationError, setBudgetOptimizationError] = useState<string | null>(null);
  const [availabilityResultsByItem, setAvailabilityResultsByItem] =
    useState<AvailabilityResultByItem>({});
  const [availabilityApplyError, setAvailabilityApplyError] = useState<string | null>(null);
  const [lockWarning, setLockWarning] = useState<EditLockView | null>(null);
  const [cachedTripRecord, setCachedTripRecord] = useState<CachedTripRecord | null>(null);
  const [offlineCacheLoading, setOfflineCacheLoading] = useState(false);
  const [offlineUnavailable, setOfflineUnavailable] = useState(false);

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

  const cancelGenerationJobMutation = useMutation({
    mutationFn: (jobId: string) => cancelGenerationJob(tripId, jobId),
    onSuccess: async (job) => {
      setActiveGenerationJobId(job.id);
      queryClient.setQueryData(generationJobKeys.detail(tripId, job.id), job);
      await queryClient.invalidateQueries({ queryKey: generationJobKeys.list(tripId) });
    }
  });

  const pendingOfflineMutation =
    offlineSync.mutations.find((mutation) => mutation.tripId === tripId) ?? null;
  const sourceTrip = tripQuery.data ?? cachedTripRecord?.trip ?? null;
  const displayedTrip = sourceTrip
    ? withPendingOfflineItinerary(sourceTrip, pendingOfflineMutation)
    : null;
  const isUsingCachedTrip = Boolean(cachedTripRecord) && (!tripQuery.data || !networkStatus.online);
  const hasPendingOfflineChanges = Boolean(pendingOfflineMutation);
  const offlineDataMode = isUsingCachedTrip || !networkStatus.online || hasPendingOfflineChanges;
  const onlineActionsEnabled =
    networkStatus.online && !isUsingCachedTrip && !hasPendingOfflineChanges;
  const cachedBudgetSummary = cachedTripRecord?.budgetSummary ?? null;
  const currentItinerary = displayedTrip?.itinerary ?? null;
  const routeEstimateStates = useRouteEstimates(
    currentItinerary,
    onlineActionsEnabled && displayedTrip?.status === "COMPLETED" && Boolean(currentItinerary),
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
    enabled: canFetchWeather && onlineActionsEnabled,
    staleTime: 10 * 60 * 1000,
    retry: 1
  });

  // Shares the cache key with BudgetPanel so the summary is fetched once and
  // also feeds budget-aware quality checks.
  const budgetSummaryQuery = useQuery({
    queryKey: budgetKeys.summary(tripId),
    queryFn: () => getTripBudgetSummary(tripId),
    enabled: onlineActionsEnabled
  });

  // Comments are a private collaboration feature: anyone who can view this
  // private trip (owner/editor/viewer) may read and add comments. Counts power
  // the per-item badges. The public share page never mounts this page.
  const tripAccess = displayedTrip?.access;
  const budgetOptimizationProposalsQuery = useBudgetOptimizationProposals({
    tripId,
    status: "pending",
    enabled: onlineActionsEnabled && Boolean(tripId) && Boolean(tripAccess)
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
  const commentsEnabled =
    onlineActionsEnabled &&
    Boolean(tripId) &&
    canComment &&
    displayedTrip?.status === "COMPLETED" &&
    Boolean(displayedTrip?.itinerary);
  const commentCountsQuery = useQuery({
    queryKey: commentKeys.counts(tripId),
    queryFn: () => listTripCommentCounts(tripId),
    enabled: commentsEnabled
  });
  const presenceEnabled =
    onlineActionsEnabled &&
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
  const exportTrip = useMemo(
    () =>
      displayedTrip
        ? toExportTripFromPrivateTrip(displayedTrip, {
            weatherSummary: toExportWeatherSummary(weatherForecastQuery.data ?? null),
            distanceSummary: toExportDistanceSummary(
              fallbackDistanceSummaries,
              routeEstimatesByDay
            ),
            budgetSummary: budgetSummaryQuery.data ?? cachedBudgetSummary ?? null
          })
        : null,
    [
      cachedBudgetSummary,
      budgetSummaryQuery.data,
      displayedTrip,
      fallbackDistanceSummaries,
      routeEstimatesByDay,
      weatherForecastQuery.data
    ]
  );

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

  if (!displayedTrip && (tripQuery.isPending || offlineCacheLoading)) {
    return (
      <PageContainer>
        <div className="rounded-lg border border-slate-200 bg-white p-6 text-sm text-slate-600">
          {offlineCacheLoading ? "Loading saved trip..." : "Loading trip..."}
        </div>
      </PageContainer>
    );
  }

  if (!displayedTrip && offlineUnavailable) {
    return (
      <PageContainer>
        <div className="rounded-lg border border-amber-200 bg-amber-50 p-6 text-sm text-amber-900">
          This trip is not available offline yet. Open it once while online.
        </div>
        <Link className={buttonStyles({ variant: "secondary", className: "mt-5" })} href="/trips">
          Back to trips
        </Link>
      </PageContainer>
    );
  }

  if (!displayedTrip && tripQuery.isError) {
    return (
      <PageContainer>
        <div className="rounded-lg border border-red-200 bg-red-50 p-6 text-sm text-red-800">
          {tripQuery.error instanceof Error ? tripQuery.error.message : "Could not load trip."}
        </div>
        <Link className={buttonStyles({ variant: "secondary", className: "mt-5" })} href="/trips">
          Back to trips
        </Link>
      </PageContainer>
    );
  }

  if (!displayedTrip) {
    return (
      <PageContainer>
        <div className="rounded-lg border border-slate-200 bg-white p-6 text-sm text-slate-600">
          Could not load trip.
        </div>
      </PageContainer>
    );
  }

  const trip = displayedTrip;
  const access = trip.access;
  const canEditTripAccess = access?.canEdit ?? true;
  const canMutateTrip = canEditTripAccess && onlineActionsEnabled;
  const canManageShare = (access?.canManageShare ?? true) && onlineActionsEnabled;
  const canManageCollaborators =
    (access?.canManageCollaborators ?? true) && onlineActionsEnabled;
  const canRestoreVersion = (access?.canRestoreVersion ?? canEditTripAccess) && onlineActionsEnabled;
  const canGenerate = canMutateTrip && (trip.status === "DRAFT" || trip.status === "FAILED");
  const canEditItinerary =
    canEditTripAccess && trip.status === "COMPLETED" && Boolean(trip.itinerary);
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

    if (result.status === "synced") {
      queryClient.setQueryData(tripKeys.detail(tripId), result.trip);
      setCachedTripRecord(null);
      setRegenerationError(null);
      setSuccessMessage("Offline changes synced.");
      void Promise.all([
        queryClient.invalidateQueries({ queryKey: tripKeys.itineraryVersions(tripId) }),
        queryClient.invalidateQueries({ queryKey: budgetKeys.summary(tripId) }),
        queryClient.invalidateQueries({ queryKey: ["route-estimate", "walking"] }),
        queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) })
      ]);
      return;
    }

    if (result.status === "conflict") {
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
      queryClient.invalidateQueries({ queryKey: ["route-estimate", "walking"] }),
      queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) })
    ]);
    await tripQuery.refetch();
    await editLock.release();
    clearEditSession();
    void setPresenceState("viewing");
    setSuccessMessage(message);
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
        queryClient.invalidateQueries({ queryKey: budgetOptimizationKeys.all(tripId) }),
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
        queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) })
      ]);
      setSuccessMessage("Budget optimization proposal discarded.");
    } catch (error) {
      setBudgetOptimizationError(getErrorMessage(error, "Could not discard proposal."));
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

    const nextCost: EstimatedCost = {
      amount: option.price.amount,
      currency: option.price.currency,
      category: availabilityCostCategory(currentItem),
      source: "provider",
      confidence: "high",
      note: `Availability provider price checked at ${result.checkedAt}; may change.`
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
    await refreshTripAfterGenerationJob();
    setRegenerationError(null);
    setSuccessMessage(successMessageForGenerationJob(job));
  }

  async function handleGenerationJobFailed(job: GenerationJob) {
    await queryClient.invalidateQueries({ queryKey: generationJobKeys.list(tripId) });
    if (job.jobType === "budget_optimization_day") {
      await queryClient.invalidateQueries({ queryKey: budgetOptimizationKeys.all(tripId) });
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
      queryClient.invalidateQueries({ queryKey: budgetKeys.summary(tripId) }),
      queryClient.invalidateQueries({ queryKey: tripKeys.lists() })
    ]);
    await tripQuery.refetch();
  }

  async function refreshAfterBudgetOptimizationJob() {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: budgetOptimizationKeys.all(tripId) }),
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

  async function handleVersionRestored(updatedTrip: Trip) {
    queryClient.setQueryData(tripKeys.detail(tripId), updatedTrip);
    await queryClient.invalidateQueries({ queryKey: budgetKeys.summary(tripId) });
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

  return (
    <PageContainer>
      <OfflineBanner
        cachedAt={cachedTripRecord?.cachedAt}
        className="mb-6"
        conflictCount={offlineSync.conflicts.length}
        failedCount={offlineSync.failed.length}
        offlineCopy={isUsingCachedTrip}
        online={networkStatus.online}
        pendingCount={offlineSync.pendingCount}
        syncing={offlineSync.syncing}
      />

      <div className="mb-8 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <Link className="text-sm font-medium text-primary-700 hover:text-primary-600" href="/trips">
            Back to trips
          </Link>
          <div className="mt-3 flex flex-wrap items-center gap-3">
            <h1 className="text-3xl font-semibold text-slate-950">{trip.destination}</h1>
            <TripStatusBadge status={trip.status} />
          </div>
        </div>
        {canGenerate ? (
          <GenerateItineraryButton
            disabled={hasActiveGenerationJob}
            itineraryRevision={trip.itineraryRevision}
            onJobCreated={handleGenerationJobCreated}
            tripId={trip.id}
          />
        ) : null}
      </div>

      <div className="grid gap-6 lg:grid-cols-[22rem_minmax(0,1fr)]">
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
          </Card>

          <BudgetPanel
            canEdit={canMutateTrip}
            offline={offlineDataMode}
            offlineSummary={budgetSummaryQuery.data ?? cachedBudgetSummary ?? null}
            onOpenBudgetOptimization={openBudgetOptimization}
            optimizationDisabled={
              isEditing ||
              createBudgetOptimizationMutation.isPending ||
              hasActiveGenerationJob
            }
            trip={trip}
          />
          <AccommodationPanel canEdit={canMutateTrip} trip={trip} />

          {presenceEnabled ? (
            <TripPresenceIndicator
              currentUserId={currentUserId}
              isConnected={presenceStream.isConnected}
              snapshot={presenceStream.snapshot}
            />
          ) : null}

          {canManageShare ? <ShareTripPanel tripId={trip.id} /> : null}
          {onlineActionsEnabled && trip.status === "COMPLETED" && trip.itinerary ? (
            <CalendarSyncPanel canSync={canSyncCalendar} trip={trip} />
          ) : null}
          {onlineActionsEnabled ? (
            <CollaboratorsPanel
              canManageCollaborators={canManageCollaborators}
              tripId={trip.id}
            />
          ) : null}
        </aside>

        <section className="min-w-0">
          {pendingOfflineMutation ? (
            <div className="mb-4">
            <PendingOfflineChangesPanel
              mutation={pendingOfflineMutation}
              online={networkStatus.online}
              onDiscard={discardPendingOfflineChanges}
              onReview={reviewPendingOfflineChanges}
              onSyncNow={offlineSync.syncNow}
              syncing={offlineSync.syncing}
            />
            </div>
          ) : null}

          <WeatherForecastCard
            className="mb-4"
            days={trip.days}
            destination={trip.destination}
            offline={!networkStatus.online || isUsingCachedTrip}
            startDate={trip.startDate}
          />

          {successMessage ? (
            <div className="mb-4 rounded-lg border border-emerald-200 bg-emerald-50 p-4 text-sm text-emerald-800">
              {successMessage}
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
            <div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-800">
              {regenerationError}
            </div>
          ) : null}

          {availabilityApplyError ? (
            <div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-800">
              {availabilityApplyError}
            </div>
          ) : null}

          {trip.status === "PROCESSING" ? (
            <div className="rounded-lg border border-amber-200 bg-amber-50 p-6 text-sm text-amber-900">
              The itinerary is being generated. This page will refresh while processing.
            </div>
          ) : null}

          {trip.status === "COMPLETED" && trip.itinerary ? (
            <div className="space-y-4">
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

              <PresenceEditingWarning
                currentUserId={currentUserId}
                snapshot={presenceStream.snapshot}
              />
              <EditLockStatus lock={editLock.lock} />
              {editLock.error ? (
                <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-800">
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
                  {canEditItinerary ? (
                    <div className="flex flex-col gap-3 rounded-lg border border-slate-200 bg-white p-4 sm:flex-row sm:items-start sm:justify-between">
                      {exportTrip ? <ExportTripMenu exportTrip={exportTrip} /> : <span />}
                      <Button
                        disabled={editLock.loading}
                        onClick={startEditing}
                        type="button"
                        variant="secondary"
                      >
                        {editLock.loading ? "Checking..." : "Edit itinerary"}
                      </Button>
                    </div>
                  ) : null}
                  <PlaceEnrichmentReviewPanel
                    readOnly={!canMutateTrip || offlineDataMode}
                    onTripUpdated={handlePlaceReviewUpdated}
                    trip={trip}
                  />
                  <OpeningHoursWarnings itinerary={trip.itinerary} startDate={trip.startDate} />
                  {canComment ? <TripCommentsSummary counts={commentCounts} /> : null}
                  <ItineraryView
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
                    onRegenerateDay={canMutateTrip ? regenerateDay : undefined}
                    onRegenerateItem={canMutateTrip ? regenerateItem : undefined}
                    regeneratingTarget={activeRegeneratingTarget}
                    startDate={trip.startDate}
                    trip={trip}
                  />
                  <ItineraryMap
                    accommodation={trip.accommodation ?? null}
                    itinerary={trip.itinerary}
                    startDate={trip.startDate}
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
            <div className="rounded-lg border border-slate-200 bg-white p-6 text-sm text-slate-600">
              This trip is completed, but no itinerary was returned.
            </div>
          ) : null}

          {(trip.status === "DRAFT" || trip.status === "FAILED") && !trip.itinerary ? (
            <div className="rounded-lg border border-slate-200 bg-white p-6">
              <h2 className="text-lg font-semibold text-slate-950">No itinerary yet</h2>
              <p className="mt-2 text-sm leading-6 text-slate-600">
                Generate an itinerary when the Trip Service and AI Planning Service are
                running.
              </p>
            </div>
          ) : null}

          {(trip.status === "DRAFT" || trip.status === "FAILED") && trip.itinerary ? (
            <ItineraryView
              currency={trip.budgetCurrency}
              itinerary={trip.itinerary}
              startDate={trip.startDate}
            />
          ) : null}

          <ActivityFeed
            canViewActivity={canComment}
            currentUserId={currentUserId}
            tripId={trip.id}
          />
        </section>
      </div>
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
    </PageContainer>
  );
}

function defaultConflictResolutions(
  mergeResult: ItineraryMergeResult
): ConflictResolutionMap {
  return Object.fromEntries(
    mergeResult.conflicts.map((conflict) => [
      conflict.conflictKey,
      conflict.resolution ?? "keep_latest"
    ])
  );
}

function withPendingOfflineItinerary(
  trip: Trip,
  mutation: PendingItineraryMutation | null
): Trip {
  if (!mutation || mutation.tripId !== trip.id) {
    return trip;
  }

  return {
    ...trip,
    itinerary: mutation.draftItinerary,
    itineraryRevision: mutation.baseRevision
  };
}

function BudgetOptimizationProposalsPanel({
  proposals,
  currentItinerary,
  canMutate,
  error,
  isLoading,
  isApplying,
  isDiscarding,
  onApply,
  onDiscard
}: {
  proposals: BudgetOptimizationProposal[];
  currentItinerary: Itinerary | null;
  canMutate: boolean;
  error: string | null;
  isLoading: boolean;
  isApplying: boolean;
  isDiscarding: boolean;
  onApply: (proposal: BudgetOptimizationProposal) => Promise<void>;
  onDiscard: (proposal: BudgetOptimizationProposal) => Promise<void>;
}) {
  if (!isLoading && proposals.length === 0 && !error) {
    return null;
  }

  return (
    <section className="space-y-3">
      <div>
        <h2 className="text-xl font-semibold text-slate-950">
          Budget Optimization Proposals
        </h2>
        <p className="mt-1 text-sm text-slate-600">
          Review cheaper day plans before applying them to the itinerary.
        </p>
      </div>

      {error ? (
        <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-800">
          {error}
        </div>
      ) : null}

      {isLoading ? (
        <div className="rounded-lg border border-slate-200 bg-white p-4 text-sm text-slate-600">
          Loading budget optimization proposals...
        </div>
      ) : null}

      {proposals.map((proposal) => (
        <BudgetOptimizationProposalCard
          canMutate={canMutate}
          currentDay={findProposalCurrentDay(currentItinerary, proposal)}
          isApplying={isApplying}
          isDiscarding={isDiscarding}
          key={proposal.id}
          onApply={onApply}
          onDiscard={onDiscard}
          proposal={proposal}
        />
      ))}
    </section>
  );
}

function findActiveGenerationJob(jobs: GenerationJob[]) {
  return jobs.find(isActiveGenerationJob) ?? null;
}

function findProposalCurrentDay(
  itinerary: Itinerary | null,
  proposal: BudgetOptimizationProposal
) {
  const dayNumber = proposal.dayNumber ?? proposal.proposal.dayNumber;
  return (
    (itinerary?.days ?? []).find(
      (day, index) => (day.day || index + 1) === dayNumber
    ) ?? null
  );
}

function isActiveGenerationJob(job: GenerationJob) {
  return job.status === "queued" || job.status === "running";
}

function targetFromGenerationJob(job: GenerationJob): RegeneratingTarget | null {
  if (
    (job.jobType === "item_regeneration" || job.jobType === "quality_improvement_item") &&
    job.dayNumber != null &&
    job.itemIndex != null
  ) {
    return { type: "item", dayNumber: job.dayNumber, itemIndex: job.itemIndex };
  }
  if (
    (job.jobType === "day_regeneration" || job.jobType === "quality_improvement_day") &&
    job.dayNumber != null
  ) {
    return { type: "day", dayNumber: job.dayNumber };
  }
  return null;
}

function successMessageForGenerationJob(job: GenerationJob) {
  if (job.jobType === "full_generation") {
    return "Itinerary generated.";
  }
  if (job.jobType === "budget_optimization_day") {
    return "Budget optimization proposal ready.";
  }
  if (
    (job.jobType === "item_regeneration" || job.jobType === "quality_improvement_item") &&
    job.dayNumber != null &&
    job.itemIndex != null
  ) {
    return `Day ${job.dayNumber} item ${job.itemIndex + 1} regenerated.`;
  }
  if (job.dayNumber != null) {
    return `Day ${job.dayNumber} regenerated.`;
  }
  return "Itinerary updated.";
}

function failureMessageForGenerationJob(job: GenerationJob) {
  if (job.errorCode === "itinerary_conflict") {
    return "Generation stopped because the itinerary changed while the job was running. Reload latest version and try again.";
  }
  if (job.errorCode === "no_optimization_found") {
    return "Budget optimization could not find a useful cheaper alternative for that day.";
  }
  if (job.jobType === "budget_optimization_day") {
    return job.errorMessage ?? "Budget optimization failed. The itinerary was not changed.";
  }
  return job.errorMessage ?? "Generation failed. The itinerary was not changed.";
}

function availabilityResultKey(dayNumber: number, itemIndex: number) {
  return `${dayNumber}:${itemIndex}`;
}

function availabilityCostCategory(item: {
  type?: string | null;
  place?: { category?: string | null } | null;
}): EstimatedCost["category"] {
  const text = `${item.type ?? ""} ${item.place?.category ?? ""}`.toLowerCase();
  if (text.includes("tour") || text.includes("activity") || text.includes("experience")) {
    return "activity";
  }
  return "ticket";
}

type DetailRowProps = {
  label: string;
  value: string;
};

function DetailRow({ label, value }: DetailRowProps) {
  return (
    <div className="flex items-start justify-between gap-4">
      <dt className="text-slate-500">{label}</dt>
      <dd className="text-right font-medium text-slate-800">{value}</dd>
    </div>
  );
}
