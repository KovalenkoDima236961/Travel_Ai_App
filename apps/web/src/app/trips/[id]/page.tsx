"use client";

import Link from "next/link";
import { useMemo, useState } from "react";
import { useParams } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ProtectedRoute } from "@/components/auth/ProtectedRoute";
import { PageContainer } from "@/components/layout/PageContainer";
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
import { getWeatherForecast, weatherKeys } from "@/lib/api/weather";
import {
  getTrip,
  regenerateItineraryDay,
  regenerateItineraryItem,
  tripKeys,
  updateTripItinerary
} from "@/lib/api/trips";
import { getMyPreferences, userKeys } from "@/lib/api/user";
import { useRouteEstimates } from "@/lib/hooks/useRouteEstimates";
import { getDayDistanceSummaries } from "@/lib/itinerary/distance-utils";
import {
  formatBudget,
  formatDate,
  formatInterestLabel,
  formatPaceLabel
} from "@/lib/utils";
import type { RouteEstimate } from "@/types/route";
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
  const [isEditing, setIsEditing] = useState(false);
  const [draftItinerary, setDraftItinerary] = useState<Itinerary | null>(null);
  const [editorErrors, setEditorErrors] = useState<string[]>([]);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);
  const [regenerationError, setRegenerationError] = useState<string | null>(null);
  const [regeneratingTarget, setRegeneratingTarget] = useState<RegeneratingTarget | null>(null);
  const [optimizingDayNumber, setOptimizingDayNumber] = useState<number | null>(null);

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
    mutationFn: (itinerary: Itinerary) => updateTripItinerary(tripId, itinerary)
  });

  const regenerationMutation = useMutation({
    mutationFn: (target: RegeneratingTarget & { instruction?: string }) => {
      if (target.type === "day") {
        return regenerateItineraryDay(tripId, target.dayNumber, target.instruction);
      }
      return regenerateItineraryItem(
        tripId,
        target.dayNumber,
        target.itemIndex,
        target.instruction
      );
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
  const canGenerate = trip.status === "DRAFT" || trip.status === "FAILED";
  const canEditItinerary = trip.status === "COMPLETED" && Boolean(trip.itinerary);
  const optimizingDay =
    optimizingDayNumber != null
      ? (trip.itinerary?.days ?? []).find(
          (day, index) => (day.day || index + 1) === optimizingDayNumber
        ) ?? null
      : null;

  function startEditing() {
    if (!trip.itinerary) {
      return;
    }
    setDraftItinerary(prepareItineraryForEdit(trip.itinerary));
    setEditorErrors([]);
    setRegenerationError(null);
    setSuccessMessage(null);
    setIsEditing(true);
  }

  function cancelEditing() {
    setIsEditing(false);
    setDraftItinerary(null);
    setEditorErrors([]);
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
      setRegenerationError(null);
      const updated = await updateMutation.mutateAsync(normalized);
      queryClient.setQueryData(tripKeys.detail(tripId), updated);
      await queryClient.invalidateQueries({ queryKey: tripKeys.itineraryVersions(tripId) });
      await queryClient.invalidateQueries({ queryKey: ["route-estimate", "walking"] });
      await tripQuery.refetch();
      setDraftItinerary(null);
      setIsEditing(false);
      setSuccessMessage("Itinerary saved.");
    } catch (error) {
      setEditorErrors([
        error instanceof Error ? error.message : "Could not save itinerary."
      ]);
    }
  }

  async function regenerateDay(dayNumber: number, instruction?: string) {
    const target: RegeneratingTarget = { type: "day", dayNumber };
    await regenerateItineraryPart(target, instruction, `Day ${dayNumber} regenerated.`);
  }

  async function regenerateItem(dayNumber: number, itemIndex: number, instruction?: string) {
    const target: RegeneratingTarget = { type: "item", dayNumber, itemIndex };
    await regenerateItineraryPart(
      target,
      instruction,
      `Day ${dayNumber} item ${itemIndex + 1} regenerated.`
    );
  }

  async function regenerateItineraryPart(
    target: RegeneratingTarget,
    instruction: string | undefined,
    message: string
  ) {
    try {
      setRegenerationError(null);
      setSuccessMessage(null);
      setRegeneratingTarget(target);
      const updated = await regenerationMutation.mutateAsync({ ...target, instruction });
      queryClient.setQueryData(tripKeys.detail(tripId), updated);
      await queryClient.invalidateQueries({ queryKey: tripKeys.itineraryVersions(tripId) });
      await queryClient.invalidateQueries({ queryKey: ["route-estimate", "walking"] });
      await tripQuery.refetch();
      setSuccessMessage(message);
    } catch (error) {
      setRegenerationError(
        error instanceof Error ? error.message : "Could not regenerate itinerary."
      );
    } finally {
      setRegeneratingTarget(null);
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
        {canGenerate ? <GenerateItineraryButton tripId={trip.id} /> : null}
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

          <ShareTripPanel tripId={trip.id} />
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
                isImproving={regenerationMutation.isPending}
                maxWalkingKmPerDay={maxWalkingKmPerDay}
                onImproveDay={regenerateDay}
                onImproveItem={regenerateItem}
                routeEstimatesByDay={routeEstimatesByDay}
                trip={trip}
                weatherForecast={weatherForecastQuery.data ?? null}
              />

              {isEditing && draftItinerary ? (
                <>
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
                    <div className="flex justify-end">
                      <Button onClick={startEditing} type="button" variant="secondary">
                        Edit itinerary
                      </Button>
                    </div>
                  ) : null}
                  <PlaceEnrichmentReviewPanel
                    onTripUpdated={handlePlaceReviewUpdated}
                    trip={trip}
                  />
                  <OpeningHoursWarnings itinerary={trip.itinerary} startDate={trip.startDate} />
                  <ItineraryView
                    currency={trip.budgetCurrency}
                    disabled={regenerationMutation.isPending}
                    itinerary={trip.itinerary}
                    onRegenerateDay={regenerateDay}
                    onRegenerateItem={regenerateItem}
                    regeneratingTarget={regeneratingTarget}
                    startDate={trip.startDate}
                  />
                  <ItineraryMap itinerary={trip.itinerary} startDate={trip.startDate} />
                  <DistanceSummary
                    itinerary={trip.itinerary}
                    maxWalkingKmPerDay={maxWalkingKmPerDay}
                    onOptimizeDay={setOptimizingDayNumber}
                  />
                  <ItineraryVersionHistory
                    currency={trip.budgetCurrency}
                    onRestored={handleVersionRestored}
                    restoreDisabled={isEditing}
                    tripId={trip.id}
                  />
                  {trip.itinerary && optimizingDay ? (
                    <OptimizeDayOrderDialog
                      day={optimizingDay}
                      itinerary={trip.itinerary}
                      onApplied={handleOptimizationApplied}
                      onClose={() => setOptimizingDayNumber(null)}
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
        </section>
      </div>
    </PageContainer>
  );
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
