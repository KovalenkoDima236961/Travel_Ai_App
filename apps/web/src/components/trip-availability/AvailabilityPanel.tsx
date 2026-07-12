"use client";

import { useMemo, useState } from "react";
import { ApplyDateOptionDialog } from "./ApplyDateOptionDialog";
import { AvailabilitySummaryCard } from "./AvailabilitySummaryCard";
import { CreateDatePollDialog } from "./CreateDatePollDialog";
import { DateOptionsList } from "./DateOptionsList";
import { MissingAvailabilityNotice } from "./MissingAvailabilityNotice";
import { MyAvailabilityForm } from "./MyAvailabilityForm";
import { useTripAvailability } from "@/hooks/useTripAvailability";
import {
  useApplyTripDateOption,
  useCreateDateOptionsPoll,
  useDeleteTripAvailability,
  useGenerateTripDateOptions,
  useRequestTripAvailability,
  useUpsertTripAvailability
} from "@/hooks/useTripAvailabilityMutations";
import { useTripDateOptions } from "@/hooks/useTripDateOptions";
import { getErrorMessage } from "@/lib/utils";
import { Button } from "@/shared/ui/button";
import { Input } from "@/shared/ui/input";
import type { GenerationJob } from "@/entities/generation-job/model";
import type { Trip } from "@/entities/trip/model";
import type { DateOptionsInput, TripDateOption } from "@/types/trip-availability";

type AvailabilityPanelProps = {
  canEdit: boolean;
  currentUserId?: string | null;
  online: boolean;
  trip: Trip;
  onGenerationJobCreated?: (job: GenerationJob) => void;
};

export function AvailabilityPanel({
  canEdit,
  currentUserId,
  online,
  trip,
  onGenerationJobCreated
}: AvailabilityPanelProps) {
  const [error, setError] = useState<string | null>(null);
  const [message, setMessage] = useState<string | null>(null);
  const [applyingOption, setApplyingOption] = useState<TripDateOption | null>(null);
  const [pollOptionIds, setPollOptionIds] = useState<string[]>([]);
  const [dateOptionsInput, setDateOptionsInput] = useState<DateOptionsInput>(() => ({
    minDays: trip.days,
    maxDays: trip.days,
    limit: 8
  }));
  const availabilityQuery = useTripAvailability(trip.id, online);
  const dateOptionsQuery = useTripDateOptions(trip.id, dateOptionsInput, online);
  const upsertMutation = useUpsertTripAvailability(trip.id);
  const deleteMutation = useDeleteTripAvailability(trip.id);
  const requestMutation = useRequestTripAvailability(trip.id);
  const generateMutation = useGenerateTripDateOptions(trip.id);
  const applyMutation = useApplyTripDateOption(trip.id);
  const createPollMutation = useCreateDateOptionsPoll(trip.id);
  const availability = availabilityQuery.data;
  const dateOptions = generateMutation.data ?? dateOptionsQuery.data;
  const currentUserResponse = useMemo(() => {
    if (!currentUserId) {
      return null;
    }
    return availability?.responses.find((response) => response.userId === currentUserId) ?? null;
  }, [availability?.responses, currentUserId]);
  const disabled = !online;

  async function saveAvailability(input: Parameters<typeof upsertMutation.mutateAsync>[0]) {
    try {
      setError(null);
      await upsertMutation.mutateAsync(input);
      setMessage("Availability saved.");
    } catch (err) {
      setError(getErrorMessage(err, "Could not save availability."));
    }
  }

  async function deleteAvailability() {
    try {
      setError(null);
      await deleteMutation.mutateAsync();
      setMessage("Availability removed.");
    } catch (err) {
      setError(getErrorMessage(err, "Could not remove availability."));
    }
  }

  async function requestAvailability() {
    try {
      setError(null);
      await requestMutation.mutateAsync({});
      setMessage("Availability request sent.");
    } catch (err) {
      setError(getErrorMessage(err, "Could not request availability."));
    }
  }

  async function generateOptions() {
    try {
      setError(null);
      await generateMutation.mutateAsync(dateOptionsInput);
      setMessage("Date options refreshed.");
    } catch (err) {
      setError(getErrorMessage(err, "Could not generate date options."));
    }
  }

  async function applyOption(option: TripDateOption, regenerateItinerary: boolean) {
    try {
      setError(null);
      const result = await applyMutation.mutateAsync({
        optionId: option.id,
        input: {
          expectedItineraryRevision: trip.itineraryRevision,
          regenerateItinerary
        }
      });
      if (result.generationJob) {
        onGenerationJobCreated?.(result.generationJob);
      }
      setMessage(regenerateItinerary ? "Dates applied. Regeneration queued." : "Dates applied.");
      setApplyingOption(null);
    } catch (err) {
      setError(getErrorMessage(err, "Could not apply date option."));
    }
  }

  async function createPoll(title: string) {
    try {
      setError(null);
      await createPollMutation.mutateAsync({
        title,
        optionIds: pollOptionIds
      });
      setMessage("Date poll created.");
      setPollOptionIds([]);
    } catch (err) {
      setError(getErrorMessage(err, "Could not create date poll."));
    }
  }

  return (
    <section
      id="dates"
      className="scroll-mt-24 rounded-[18px] border border-sand-300 bg-white p-5"
    >
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h2 className="font-newsreader text-[25px] font-semibold text-cocoa-900">
            Dates
          </h2>
        </div>
        {canEdit ? (
          <Button
            disabled={disabled || requestMutation.isPending}
            onClick={() => void requestAvailability()}
            size="sm"
            type="button"
            variant="secondary"
          >
            {requestMutation.isPending ? "Requesting..." : "Request availability"}
          </Button>
        ) : null}
      </div>

      {!online ? (
        <p className="mt-4 rounded-[12px] bg-sand-100 px-3 py-2 text-[13px] text-cocoa-500">
          You need to be online to update availability.
        </p>
      ) : null}
      {error ? (
        <p className="mt-4 rounded-[12px] border border-[#E5C3B6] bg-[#FBF0EB] px-3 py-2 text-[13px] text-[#B3402E]">
          {error}
        </p>
      ) : null}
      {message ? (
        <p className="mt-4 rounded-[12px] border border-[#DCE8DD] bg-[#F2F7F1] px-3 py-2 text-[13px] text-[#38543F]">
          {message}
        </p>
      ) : null}

      <div className="mt-5 space-y-5">
        <AvailabilitySummaryCard summary={availability?.summary} />
        <MissingAvailabilityNotice summary={availability?.summary} />

        <div className="grid gap-5 lg:grid-cols-[minmax(0,0.95fr)_minmax(0,1.05fr)]">
          <div>
            <h3 className="mb-3 text-[14px] font-semibold text-cocoa-900">
              My availability
            </h3>
            <MyAvailabilityForm
              disabled={disabled}
              isDeleting={deleteMutation.isPending}
              isSaving={upsertMutation.isPending}
              onDelete={deleteAvailability}
              onSave={saveAvailability}
              response={currentUserResponse}
            />
          </div>

          <div className="space-y-4">
            <div className="flex flex-wrap items-end gap-3">
              <label className="w-28 space-y-1 text-[13px] font-medium text-cocoa-600">
                <span>Min days</span>
                <Input
                  disabled={disabled}
                  min={1}
                  onChange={(event) =>
                    setDateOptionsInput((current) => ({
                      ...current,
                      minDays: numberOrUndefined(event.target.value)
                    }))
                  }
                  type="number"
                  value={dateOptionsInput.minDays ?? ""}
                />
              </label>
              <label className="w-28 space-y-1 text-[13px] font-medium text-cocoa-600">
                <span>Max days</span>
                <Input
                  disabled={disabled}
                  min={1}
                  onChange={(event) =>
                    setDateOptionsInput((current) => ({
                      ...current,
                      maxDays: numberOrUndefined(event.target.value)
                    }))
                  }
                  type="number"
                  value={dateOptionsInput.maxDays ?? ""}
                />
              </label>
              <label className="flex items-center gap-2 pb-3 text-[13px] font-medium text-cocoa-600">
                <input
                  checked={Boolean(dateOptionsInput.preferWeekends)}
                  disabled={disabled}
                  onChange={(event) =>
                    setDateOptionsInput((current) => ({
                      ...current,
                      preferWeekends: event.target.checked
                    }))
                  }
                  type="checkbox"
                />
                Prefer weekends
              </label>
              <Button
                disabled={disabled || generateMutation.isPending}
                onClick={() => void generateOptions()}
                size="sm"
                type="button"
              >
                {generateMutation.isPending ? "Refreshing..." : "Refresh options"}
              </Button>
            </div>

            <DateOptionsList
              canEdit={canEdit}
              disabled={disabled}
              isApplying={applyMutation.isPending}
              isCreatingPoll={createPollMutation.isPending}
              onApply={setApplyingOption}
              onCreatePoll={setPollOptionIds}
              result={dateOptions}
            />
          </div>
        </div>
      </div>
      <ApplyDateOptionDialog
        hasItinerary={Boolean(trip.itinerary)}
        isPending={applyMutation.isPending}
        onApply={(regenerateItinerary) => {
          if (applyingOption) {
            void applyOption(applyingOption, regenerateItinerary);
          }
        }}
        onOpenChange={(open) => {
          if (!open) {
            setApplyingOption(null);
          }
        }}
        open={Boolean(applyingOption)}
        option={applyingOption}
      />
      <CreateDatePollDialog
        isPending={createPollMutation.isPending}
        onCreate={(title) => void createPoll(title)}
        onOpenChange={(open) => {
          if (!open) {
            setPollOptionIds([]);
          }
        }}
        open={pollOptionIds.length > 0}
        optionCount={pollOptionIds.length}
      />
    </section>
  );
}

function numberOrUndefined(value: string) {
  const parsed = Number(value);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : undefined;
}
