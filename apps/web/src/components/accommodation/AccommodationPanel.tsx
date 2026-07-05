"use client";

import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { AccommodationForm } from "@/components/accommodation/AccommodationForm";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import {
  accommodationKeys,
  deleteTripAccommodation,
  updateTripAccommodation
} from "@/lib/api/accommodation";
import { activityKeys } from "@/lib/api/activity";
import { budgetKeys } from "@/lib/api/budget";
import { tripKeys } from "@/lib/api/trips";
import { formatMoney } from "@/lib/budget/format";
import { getErrorMessage } from "@/lib/utils";
import type { TripAccommodation } from "@/types/accommodation";
import type { Trip } from "@/types/trip";

type AccommodationPanelProps = {
  trip: Trip;
  canEdit: boolean;
  onOpenCostSplit?: () => void;
};

export function AccommodationPanel({ trip, canEdit, onOpenCostSplit }: AccommodationPanelProps) {
  const queryClient = useQueryClient();
  const [isEditing, setIsEditing] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const accommodation = trip.accommodation ?? null;
  const defaultCurrency = trip.budget?.currency ?? trip.budgetCurrency ?? "EUR";

  const updateMutation = useMutation({
    mutationFn: (nextAccommodation: TripAccommodation) =>
      updateTripAccommodation(trip.id, nextAccommodation),
    onSuccess: async (nextAccommodation) => {
      setError(null);
      setMessage(accommodation ? "Accommodation updated." : "Accommodation added.");
      setIsEditing(false);
      setTripAccommodationCache(queryClient, trip.id, nextAccommodation);
      await invalidateAccommodationDependents(queryClient, trip.id);
    },
    onError: (err) => {
      setMessage(null);
      setError(getErrorMessage(err, "Could not save accommodation."));
    }
  });

  const deleteMutation = useMutation({
    mutationFn: () => deleteTripAccommodation(trip.id),
    onSuccess: async () => {
      setError(null);
      setMessage("Accommodation cleared.");
      setIsEditing(false);
      setTripAccommodationCache(queryClient, trip.id, null);
      await invalidateAccommodationDependents(queryClient, trip.id);
    },
    onError: (err) => {
      setMessage(null);
      setError(getErrorMessage(err, "Could not clear accommodation."));
    }
  });

  const isSaving = updateMutation.isPending || deleteMutation.isPending;

  function handleClearAccommodation() {
    if (!window.confirm("Remove accommodation from this trip?")) {
      return;
    }
    deleteMutation.mutate();
  }

  return (
    <Card>
      <div className="flex items-start justify-between gap-3">
        <h2 className="text-lg font-semibold text-slate-950">Accommodation</h2>
        {canEdit && !isEditing ? (
          <Button onClick={() => setIsEditing(true)} size="sm" type="button" variant="secondary">
            {accommodation ? "Edit" : "Add stay"}
          </Button>
        ) : null}
      </div>

      {message ? <p className="mt-2 text-sm text-emerald-700">{message}</p> : null}
      {error ? <p className="mt-2 text-sm text-red-700">{error}</p> : null}

      {isEditing ? (
        <div className="mt-4">
          <AccommodationForm
            defaultCurrency={defaultCurrency}
            destination={trip.destination}
            initial={accommodation}
            isSaving={isSaving}
            onCancel={() => {
              setIsEditing(false);
              setError(null);
            }}
            onClear={accommodation ? handleClearAccommodation : undefined}
            onSave={(nextAccommodation) => updateMutation.mutate(nextAccommodation)}
          />
        </div>
      ) : (
        <AccommodationSummary
          accommodation={accommodation}
          defaultCurrency={defaultCurrency}
          onOpenCostSplit={onOpenCostSplit}
        />
      )}
    </Card>
  );
}

function AccommodationSummary({
  accommodation,
  defaultCurrency,
  onOpenCostSplit
}: {
  accommodation: TripAccommodation | null;
  defaultCurrency: string;
  onOpenCostSplit?: () => void;
}) {
  if (!accommodation) {
    return (
      <p className="mt-4 text-sm leading-6 text-slate-600">No accommodation added yet.</p>
    );
  }

  const cost = accommodation.estimatedCost;
  const currency = cost?.currency ?? defaultCurrency;

  return (
    <div className="mt-4 space-y-3 text-sm">
      <div>
        <p className="font-semibold text-slate-950">{accommodation.name}</p>
        <p className="mt-1 capitalize text-slate-600">{accommodation.type}</p>
      </div>

      {accommodation.address ? (
        <p className="leading-6 text-slate-700">{accommodation.address}</p>
      ) : null}

      <dl className="space-y-2">
        {accommodation.checkInDate || accommodation.checkOutDate ? (
          <SummaryRow
            label="Dates"
            value={[accommodation.checkInDate, accommodation.checkOutDate]
              .filter(Boolean)
              .join(" to ")}
          />
        ) : null}
        {cost?.amount != null ? (
          <SummaryRow label="Estimated cost" value={formatMoney(cost.amount, currency)} />
        ) : null}
        {accommodation.place ? (
          <SummaryRow label="Place" value={`${accommodation.place.provider} attached`} />
        ) : null}
      </dl>

      {accommodation.notes ? (
        <p className="rounded-md border border-slate-200 bg-slate-50 p-3 text-slate-700">
          {accommodation.notes}
        </p>
      ) : null}

      {onOpenCostSplit ? (
        <Button
          disabled={cost?.amount == null}
          onClick={onOpenCostSplit}
          size="sm"
          type="button"
          variant="secondary"
        >
          {cost?.amount == null ? "Add cost first" : "Split accommodation cost"}
        </Button>
      ) : null}
    </div>
  );
}

function SummaryRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-3">
      <dt className="text-slate-600">{label}</dt>
      <dd className="text-right font-medium text-slate-900">{value}</dd>
    </div>
  );
}

function setTripAccommodationCache(
  queryClient: ReturnType<typeof useQueryClient>,
  tripId: string,
  accommodation: TripAccommodation | null
) {
  queryClient.setQueryData<Trip>(tripKeys.detail(tripId), (current) =>
    current ? { ...current, accommodation } : current
  );
}

async function invalidateAccommodationDependents(
  queryClient: ReturnType<typeof useQueryClient>,
  tripId: string
) {
  await Promise.all([
    queryClient.invalidateQueries({ queryKey: accommodationKeys.detail(tripId) }),
    queryClient.invalidateQueries({ queryKey: budgetKeys.summary(tripId) }),
    queryClient.invalidateQueries({ queryKey: tripKeys.detail(tripId) }),
    queryClient.invalidateQueries({ queryKey: activityKeys.all(tripId) }),
    queryClient.invalidateQueries({ queryKey: ["route-estimate"] })
  ]);
}
