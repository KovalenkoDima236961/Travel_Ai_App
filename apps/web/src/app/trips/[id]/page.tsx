"use client";

import Link from "next/link";
import { useState } from "react";
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
import { ItineraryVersionHistory } from "@/components/trips/ItineraryVersionHistory";
import { ItineraryView, type RegeneratingTarget } from "@/components/trips/ItineraryView";
import { TripStatusBadge } from "@/components/trips/TripStatusBadge";
import { Button, buttonStyles } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import {
  getTrip,
  regenerateItineraryDay,
  regenerateItineraryItem,
  tripKeys,
  updateTripItinerary
} from "@/lib/api/trips";
import {
  formatBudget,
  formatDate,
  formatInterestLabel,
  formatPaceLabel
} from "@/lib/utils";
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

  const tripQuery = useQuery({
    queryKey: tripKeys.detail(tripId),
    queryFn: () => getTrip(tripId),
    enabled: Boolean(tripId),
    refetchInterval: (query) =>
      query.state.data?.status === "PROCESSING" ? 3000 : false
  });

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
    await tripQuery.refetch();
    setRegenerationError(null);
    setSuccessMessage("Itinerary restored.");
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

        <section className="min-w-0">
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
            isEditing && draftItinerary ? (
              <div className="space-y-4">
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
                  disabled={updateMutation.isPending}
                  errors={editorErrors}
                  itinerary={draftItinerary}
                  onChange={setDraftItinerary}
                />
              </div>
            ) : (
              <div className="space-y-4">
                {canEditItinerary ? (
                  <div className="flex justify-end">
                    <Button onClick={startEditing} type="button" variant="secondary">
                      Edit itinerary
                    </Button>
                  </div>
                ) : null}
                <ItineraryView
                  currency={trip.budgetCurrency}
                  disabled={regenerationMutation.isPending}
                  itinerary={trip.itinerary}
                  onRegenerateDay={regenerateDay}
                  onRegenerateItem={regenerateItem}
                  regeneratingTarget={regeneratingTarget}
                />
                <ItineraryVersionHistory
                  currency={trip.budgetCurrency}
                  onRestored={handleVersionRestored}
                  restoreDisabled={isEditing}
                  tripId={trip.id}
                />
              </div>
            )
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
            <ItineraryView itinerary={trip.itinerary} currency={trip.budgetCurrency} />
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
