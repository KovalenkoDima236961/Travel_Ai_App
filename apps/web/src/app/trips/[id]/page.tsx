"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { useParams } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { useAuth } from "@/components/auth/AuthProvider";
import { ActivityFeed } from "@/components/activity/ActivityFeed";
import { CalendarSyncPanel } from "@/components/calendar/CalendarSyncPanel";
import { EditLockStatus } from "@/components/edit-locks/EditLockStatus";
import { SoftEditLockWarningDialog } from "@/components/edit-locks/SoftEditLockWarningDialog";
import { ExportTripMenu } from "@/components/export/ExportTripMenu";
import { GenerationJobStatusCard } from "@/components/generation-jobs/GenerationJobStatusCard";
import { ItemCommentsPanel } from "@/components/comments/ItemCommentsPanel";
import { TripCommentsSummary } from "@/components/comments/TripCommentsSummary";
import { PageContainer } from "@/components/layout/PageContainer";
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
import { useGenerationJob } from "@/lib/hooks/useGenerationJob";
import { getDayDistanceSummaries } from "@/lib/itinerary/distance-utils";
import { useTripEditLock } from "@/lib/edit-locks/use-trip-edit-lock";
import { useTripPresenceState } from "@/lib/presence/use-trip-presence-state";
import { useTripPresenceStream } from "@/lib/presence/use-trip-presence-stream";
import {
  formatBudget,
  formatDate,
  formatInterestLabel,
  formatPaceLabel
} from "@/lib/utils";
import type { RouteEstimate } from "@/types/route";
import type { EditLockView } from "@/types/edit-locks";
import type {
  CreateGenerationJobRequest,
  GenerationJob,
  GenerationJobType
} from "@/types/generation-jobs";
import type { Itinerary, Trip } from "@/types/trip";

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
  const [isEditing, setIsEditing] = useState(false);
  const [commentTarget, setCommentTarget] = useState<{
    dayNumber: number;
    itemIndex: number;
    title: string;
    time?: string | null;
  } | null>(null);
  const [draftItinerary, setDraftItinerary] = useState<Itinerary | null>(null);
  const [baseItineraryRevision, setBaseItineraryRevision] = useState<number | null>(null);
  const [editorErrors, setEditorErrors] = useState<string[]>([]);
  const [itineraryConflictMessage, setItineraryConflictMessage] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);
  const [regenerationError, setRegenerationError] = useState<string | null>(null);
  const [activeGenerationJobId, setActiveGenerationJobId] = useState<string | null>(null);
  const [optimizingDayNumber, setOptimizingDayNumber] = useState<number | null>(null);
  const [lockWarning, setLockWarning] = useState<EditLockView | null>(null);

  const tripQuery = useQuery({
    queryKey: tripKeys.detail(tripId),
    queryFn: () => getTrip(tripId),
    enabled: Boolean(tripId),
    refetchInterval: (query) =>
      query.state.data?.status === "PROCESSING" ? 3000 : false
  });

  // Preferences power the walking-distance warning. They are intentionally
  // non-blocking: if the fetch fails we still render the distance estimates and
  // simply omit the preference comparison.
  const preferencesQuery = useQuery({
    queryKey: userKeys.preferences(),
    queryFn: getMyPreferences,
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

  const cancelGenerationJobMutation = useMutation({
    mutationFn: (jobId: string) => cancelGenerationJob(tripId, jobId),
    onSuccess: async (job) => {
      setActiveGenerationJobId(job.id);
      queryClient.setQueryData(generationJobKeys.detail(tripId, job.id), job);
      await queryClient.invalidateQueries({ queryKey: generationJobKeys.list(tripId) });
    }
  });

  const currentItinerary = tripQuery.data?.itinerary ?? null;
  const routeEstimateStates = useRouteEstimates(
    currentItinerary,
    tripQuery.data?.status === "COMPLETED" && Boolean(currentItinerary)
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
      currentItinerary ? getDayDistanceSummaries(currentItinerary, maxWalkingKmPerDay) : [],
    [currentItinerary, maxWalkingKmPerDay]
  );

  const weatherParams = {
    destination: tripQuery.data?.destination ?? "",
    startDate: tripQuery.data?.startDate ?? "",
    days: tripQuery.data?.days ?? 0
  };
  const canFetchWeather =
    Boolean(weatherParams.destination.trim()) &&
    Boolean(weatherParams.startDate) &&
    weatherParams.days > 0;
  const weatherForecastQuery = useQuery({
    queryKey: weatherKeys.forecast(weatherParams),
    queryFn: () => getWeatherForecast(weatherParams),
    enabled: canFetchWeather,
    staleTime: 10 * 60 * 1000,
    retry: 1
  });

  // Comments are a private collaboration feature: anyone who can view this
  // private trip (owner/editor/viewer) may read and add comments. Counts power
  // the per-item badges. The public share page never mounts this page.
  const tripAccess = tripQuery.data?.access;
  const generationJobsQuery = useQuery({
    queryKey: generationJobKeys.list(tripId),
    queryFn: () => listGenerationJobs(tripId),
    enabled: Boolean(tripId) && Boolean(tripAccess),
    refetchInterval: (query) => (findActiveGenerationJob(query.state.data ?? []) ? 3000 : false)
  });
  const latestActiveGenerationJob = findActiveGenerationJob(generationJobsQuery.data ?? []);
  const generationJobPoll = useGenerationJob({
    tripId,
    jobId: activeGenerationJobId ?? latestActiveGenerationJob?.id,
    enabled: Boolean(tripId) && Boolean(activeGenerationJobId ?? latestActiveGenerationJob?.id),
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
  const canComment =
    !tripAccess ||
    tripAccess.role === "owner" ||
    tripAccess.role === "editor" ||
    tripAccess.role === "viewer";
  const commentsEnabled =
    Boolean(tripId) &&
    canComment &&
    tripQuery.data?.status === "COMPLETED" &&
    Boolean(tripQuery.data?.itinerary);
  const commentCountsQuery = useQuery({
    queryKey: commentKeys.counts(tripId),
    queryFn: () => listTripCommentCounts(tripId),
    enabled: commentsEnabled
  });
  const presenceEnabled =
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
  const setPresenceState = useTripPresenceState(tripId, presenceEnabled);
  const editLocksEnabled =
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
    tripQuery.data?.status === "COMPLETED" &&
    Boolean(tripQuery.data?.itinerary);
  const editLock = useTripEditLock({
    tripId,
    enabled: editLocksEnabled,
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

  const commentCounts = commentCountsQuery.data ?? [];
  const commentCountMap = useMemo(
    () => buildCommentCountMap(commentCounts),
    [commentCounts]
  );
  const exportTrip = useMemo(
    () =>
      tripQuery.data
        ? toExportTripFromPrivateTrip(tripQuery.data, {
            weatherSummary: toExportWeatherSummary(weatherForecastQuery.data ?? null),
            distanceSummary: toExportDistanceSummary(
              fallbackDistanceSummaries,
              routeEstimatesByDay
            )
          })
        : null,
    [
      fallbackDistanceSummaries,
      routeEstimatesByDay,
      tripQuery.data,
      weatherForecastQuery.data
    ]
  );

  if (tripQuery.isPending) {
    return (
      <PageContainer>
        <div className="rounded-lg border border-slate-200 bg-white p-6 text-sm text-slate-600">
          Loading trip...
        </div>
      </PageContainer>
    );
  }

  if (tripQuery.isError) {
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

  const trip = tripQuery.data;
  const access = trip.access;
  const canMutateTrip = access?.canEdit ?? true;
  const canManageShare = access?.canManageShare ?? true;
  const canManageCollaborators = access?.canManageCollaborators ?? true;
  const canRestoreVersion = access?.canRestoreVersion ?? canMutateTrip;
  const canGenerate = canMutateTrip && (trip.status === "DRAFT" || trip.status === "FAILED");
  const canEditItinerary = canMutateTrip && trip.status === "COMPLETED" && Boolean(trip.itinerary);
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
    setDraftItinerary(prepareItineraryForEdit(trip.itinerary));
    setBaseItineraryRevision(trip.itineraryRevision);
    setEditorErrors([]);
    setItineraryConflictMessage(null);
    setRegenerationError(null);
    setSuccessMessage(null);
    setIsEditing(true);
    void setPresenceState("editing");
  }

  async function startEditing() {
    if (!canEditItinerary) {
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
    setIsEditing(false);
    setDraftItinerary(null);
    setBaseItineraryRevision(null);
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
      const updated = await updateMutation.mutateAsync({
        itinerary: normalized,
        expectedRevision: baseItineraryRevision ?? trip.itineraryRevision
      });
      queryClient.setQueryData(tripKeys.detail(tripId), updated);
      await queryClient.invalidateQueries({ queryKey: tripKeys.itineraryVersions(tripId) });
      await queryClient.invalidateQueries({ queryKey: ["route-estimate", "walking"] });
      await queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) });
      await tripQuery.refetch();
      await editLock.release();
      setDraftItinerary(null);
      setBaseItineraryRevision(null);
      setIsEditing(false);
      void setPresenceState("viewing");
      setSuccessMessage("Itinerary saved.");
    } catch (error) {
      if (isItineraryConflictError(error)) {
        setItineraryConflictMessage("This itinerary changed while you were editing.");
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
    setDraftItinerary(null);
    setBaseItineraryRevision(null);
    setIsEditing(false);
    void setPresenceState("viewing");
    await tripQuery.refetch();
  }

  async function cancelLocalChangesAfterConflict() {
    await editLock.release();
    setItineraryConflictMessage(null);
    setDraftItinerary(null);
    setBaseItineraryRevision(null);
    setIsEditing(false);
    void setPresenceState("viewing");
    await tripQuery.refetch();
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

  function handleGenerationJobCreated(job: GenerationJob) {
    setActiveGenerationJobId(job.id);
    setSuccessMessage(null);
    setRegenerationError(null);
    queryClient.setQueryData(generationJobKeys.detail(tripId, job.id), job);
  }

  async function handleGenerationJobCompleted(job: GenerationJob) {
    await refreshTripAfterGenerationJob();
    setRegenerationError(null);
    setSuccessMessage(successMessageForGenerationJob(job));
  }

  async function handleGenerationJobFailed(job: GenerationJob) {
    await queryClient.invalidateQueries({ queryKey: generationJobKeys.list(tripId) });
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
      queryClient.invalidateQueries({ queryKey: tripKeys.lists() })
    ]);
    await tripQuery.refetch();
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
    await queryClient.invalidateQueries({ queryKey: ["route-estimate", "walking"] });
    await tripQuery.refetch();
    setRegenerationError(null);
    setSuccessMessage("Itinerary restored.");
  }

  async function handleOptimizationApplied(updatedTrip: Trip) {
    const optimizedDayNumber = optimizingDayNumber;
    queryClient.setQueryData(tripKeys.detail(tripId), updatedTrip);
    await queryClient.invalidateQueries({ queryKey: tripKeys.itineraryVersions(tripId) });
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

          {presenceEnabled ? (
            <TripPresenceIndicator
              currentUserId={currentUserId}
              isConnected={presenceStream.isConnected}
              snapshot={presenceStream.snapshot}
            />
          ) : null}

          {canManageShare ? <ShareTripPanel tripId={trip.id} /> : null}
          {trip.status === "COMPLETED" && trip.itinerary ? (
            <CalendarSyncPanel canSync={canSyncCalendar} trip={trip} />
          ) : null}
          <CollaboratorsPanel
            canManageCollaborators={canManageCollaborators}
            tripId={trip.id}
          />
        </aside>

        <section className="min-w-0">
          <WeatherForecastCard
            className="mb-4"
            days={trip.days}
            destination={trip.destination}
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

          {trip.status === "PROCESSING" ? (
            <div className="rounded-lg border border-amber-200 bg-amber-50 p-6 text-sm text-amber-900">
              The itinerary is being generated. This page will refresh while processing.
            </div>
          ) : null}

          {trip.status === "COMPLETED" && trip.itinerary ? (
            <div className="space-y-4">
              <TripQualityChecks
                fallbackDistanceSummaries={fallbackDistanceSummaries}
                isEditing={isEditing}
                isImproving={createGenerationJobMutation.isPending || hasActiveGenerationJob}
                maxWalkingKmPerDay={maxWalkingKmPerDay}
                onImproveDay={canMutateTrip ? improveDay : undefined}
                onImproveItem={canMutateTrip ? improveItem : undefined}
                routeEstimatesByDay={routeEstimatesByDay}
                trip={trip}
                weatherForecast={weatherForecastQuery.data ?? null}
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
                      This itinerary was updated while you are editing.
                    </div>
                  ) : null}
                  {itineraryConflictMessage ? (
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
                    readOnly={!canMutateTrip}
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
                    onRegenerateDay={canMutateTrip ? regenerateDay : undefined}
                    onRegenerateItem={canMutateTrip ? regenerateItem : undefined}
                    regeneratingTarget={activeRegeneratingTarget}
                    startDate={trip.startDate}
                  />
                  <ItineraryMap itinerary={trip.itinerary} startDate={trip.startDate} />
                  <DistanceSummary
                    itinerary={trip.itinerary}
                    maxWalkingKmPerDay={maxWalkingKmPerDay}
                    onOptimizeDay={canMutateTrip ? setOptimizingDayNumber : undefined}
                  />
                  <ItineraryVersionHistory
                    canRestore={canRestoreVersion}
                    currency={trip.budgetCurrency}
                    itineraryRevision={trip.itineraryRevision}
                    onRestored={handleVersionRestored}
                    restoreDisabled={isEditing || !canRestoreVersion}
                    tripId={trip.id}
                  />
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
    </PageContainer>
  );
}

function findActiveGenerationJob(jobs: GenerationJob[]) {
  return jobs.find(isActiveGenerationJob) ?? null;
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
  return job.errorMessage ?? "Generation failed. The itinerary was not changed.";
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
