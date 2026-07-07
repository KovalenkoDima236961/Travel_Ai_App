"use client";

import { useMemo, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { TravelerFormDialog } from "./TravelerFormDialog";
import { Button } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import {
  costSplittingKeys,
  createTripTraveler,
  removeTripTraveler,
  updateTripTraveler
} from "@/lib/api/cost-splitting";
import { formatMoney } from "@/entities/budget/model";
import { getErrorMessage } from "@/lib/utils";
import type {
  CostSplittingSummary,
  CreateTripTravelerInput,
  TripTraveler
} from "@/entities/cost-splitting/model";

type TravelersPanelProps = {
  tripId: string;
  travelers: TripTraveler[];
  summary?: CostSplittingSummary | null;
  currency: string;
  canEdit: boolean;
  isLoading?: boolean;
};

export function TravelersPanel({
  tripId,
  travelers,
  summary,
  currency,
  canEdit,
  isLoading = false
}: TravelersPanelProps) {
  const queryClient = useQueryClient();
  const [dialogTraveler, setDialogTraveler] = useState<TripTraveler | null | undefined>(undefined);
  const [error, setError] = useState<string | null>(null);
  const allocationByTraveler = useMemo(
    () => new Map((summary?.travelers ?? []).map((traveler) => [traveler.travelerId, traveler])),
    [summary]
  );

  async function invalidate() {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: costSplittingKeys.travelers(tripId) }),
      queryClient.invalidateQueries({ queryKey: costSplittingKeys.all })
    ]);
  }

  const createMutation = useMutation({
    mutationFn: (input: CreateTripTravelerInput) => createTripTraveler(tripId, input),
    onSuccess: async () => {
      setError(null);
      setDialogTraveler(undefined);
      await invalidate();
    },
    onError: (err) => setError(getErrorMessage(err, "Could not add traveler."))
  });

  const updateMutation = useMutation({
    mutationFn: ({ travelerId, input }: { travelerId: string; input: CreateTripTravelerInput }) =>
      updateTripTraveler(tripId, travelerId, input),
    onSuccess: async () => {
      setError(null);
      setDialogTraveler(undefined);
      await invalidate();
    },
    onError: (err) => setError(getErrorMessage(err, "Could not update traveler."))
  });

  const removeMutation = useMutation({
    mutationFn: (travelerId: string) => removeTripTraveler(tripId, travelerId),
    onSuccess: invalidate,
    onError: (err) => setError(getErrorMessage(err, "Could not remove traveler."))
  });

  function submitTraveler(input: CreateTripTravelerInput) {
    if (dialogTraveler) {
      updateMutation.mutate({ travelerId: dialogTraveler.id, input });
      return;
    }
    createMutation.mutate(input);
  }

  function removeTraveler(traveler: TripTraveler) {
    if (!window.confirm(`Remove ${traveler.name} from cost planning?`)) {
      return;
    }
    removeMutation.mutate(traveler.id);
  }

  return (
    <Card>
      <div className="flex items-start justify-between gap-3">
        <h2 className="text-lg font-semibold text-slate-950">Travelers</h2>
        {canEdit ? (
          <Button onClick={() => setDialogTraveler(null)} size="sm" type="button" variant="secondary">
            Add traveler
          </Button>
        ) : null}
      </div>

      {error ? <p className="mt-3 text-sm text-red-700">{error}</p> : null}

      {isLoading ? (
        <p className="mt-4 text-sm text-slate-500">Loading travelers...</p>
      ) : travelers.length === 0 ? (
        <p className="mt-4 text-sm leading-6 text-slate-600">
          No travelers yet. Add travelers to split estimated trip costs.
        </p>
      ) : (
        <ul className="mt-4 divide-y divide-slate-100">
          {travelers.map((traveler) => {
            const allocation = allocationByTraveler.get(traveler.id);
            return (
              <li className="flex items-start justify-between gap-3 py-3" key={traveler.id}>
                <div className="min-w-0">
                  <p className="font-medium text-slate-950">{traveler.name}</p>
                  <p className="mt-1 text-xs text-slate-500">
                    {traveler.email ?? "No email"} · {traveler.role}
                    {traveler.linkedUserId ? " · linked user" : ""}
                  </p>
                  {allocation ? (
                    <p className="mt-1 text-sm font-medium text-slate-800">
                      {formatMoney(allocation.allocatedTotal, currency)}
                    </p>
                  ) : null}
                </div>
                {canEdit ? (
                  <div className="flex shrink-0 gap-2">
                    <Button onClick={() => setDialogTraveler(traveler)} size="sm" type="button" variant="ghost">
                      Edit
                    </Button>
                    <Button
                      disabled={removeMutation.isPending}
                      onClick={() => removeTraveler(traveler)}
                      size="sm"
                      type="button"
                      variant="ghost"
                    >
                      Remove
                    </Button>
                  </div>
                ) : null}
              </li>
            );
          })}
        </ul>
      )}

      <TravelerFormDialog
        error={error}
        isSaving={createMutation.isPending || updateMutation.isPending}
        onClose={() => {
          setDialogTraveler(undefined);
          setError(null);
        }}
        onSubmit={submitTraveler}
        open={dialogTraveler !== undefined}
        traveler={dialogTraveler || null}
      />
    </Card>
  );
}
