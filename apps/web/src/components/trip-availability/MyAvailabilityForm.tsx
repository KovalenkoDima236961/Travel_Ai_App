"use client";

import { FormEvent, useEffect, useState } from "react";
import { DateRangeListEditor } from "./DateRangeListEditor";
import { Button } from "@/shared/ui/button";
import { Input } from "@/shared/ui/input";
import { Textarea } from "@/shared/ui/textarea";
import type {
  AvailabilityDateRange,
  TripAvailabilityResponseInfo,
  UpsertTripAvailabilityInput
} from "@/types/trip-availability";

type MyAvailabilityFormProps = {
  disabled?: boolean;
  isDeleting?: boolean;
  isSaving?: boolean;
  response?: TripAvailabilityResponseInfo | null;
  onDelete: () => Promise<void> | void;
  onSave: (input: UpsertTripAvailabilityInput) => Promise<void> | void;
};

export function MyAvailabilityForm({
  disabled = false,
  isDeleting = false,
  isSaving = false,
  response,
  onDelete,
  onSave
}: MyAvailabilityFormProps) {
  const [availableRanges, setAvailableRanges] = useState<AvailabilityDateRange[]>([
    { startDate: "", endDate: "" }
  ]);
  const [unavailableRanges, setUnavailableRanges] = useState<AvailabilityDateRange[]>([]);
  const [preferredRanges, setPreferredRanges] = useState<AvailabilityDateRange[]>([]);
  const [minTripDays, setMinTripDays] = useState("");
  const [maxTripDays, setMaxTripDays] = useState("");
  const [timezone, setTimezone] = useState(defaultTimezone);
  const [notes, setNotes] = useState("");
  const [localError, setLocalError] = useState<string | null>(null);

  useEffect(() => {
    if (!response || !response.submitted) {
      return;
    }
    setAvailableRanges(response.availableRanges.length > 0 ? response.availableRanges : []);
    setUnavailableRanges(response.unavailableRanges ?? []);
    setPreferredRanges(response.preferredRanges ?? []);
    setMinTripDays(response.minTripDays != null ? String(response.minTripDays) : "");
    setMaxTripDays(response.maxTripDays != null ? String(response.maxTripDays) : "");
    setTimezone(response.timezone || defaultTimezone());
    setNotes(response.notes ?? "");
  }, [response]);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const available = cleanRanges(availableRanges);
    if (available.length === 0) {
      setLocalError("Add at least one available date range.");
      return;
    }
    setLocalError(null);
    await onSave({
      availableRanges: available,
      unavailableRanges: cleanRanges(unavailableRanges),
      preferredRanges: cleanRanges(preferredRanges),
      minTripDays: numberOrNull(minTripDays),
      maxTripDays: numberOrNull(maxTripDays),
      timezone,
      notes
    });
  }

  return (
    <form className="space-y-4" onSubmit={submit}>
      <DateRangeListEditor
        disabled={disabled || isSaving}
        emptyLabel="No available dates yet."
        onChange={setAvailableRanges}
        ranges={availableRanges}
        title="Available"
      />
      <DateRangeListEditor
        disabled={disabled || isSaving}
        emptyLabel="Optional conflicts you already know about."
        onChange={setUnavailableRanges}
        ranges={unavailableRanges}
        title="Unavailable"
      />
      <DateRangeListEditor
        disabled={disabled || isSaving}
        emptyLabel="Optional preferred windows."
        onChange={setPreferredRanges}
        ranges={preferredRanges}
        title="Preferred"
      />

      <div className="grid gap-3 sm:grid-cols-3">
        <label className="space-y-1 text-[13px] font-medium text-cocoa-600">
          <span>Minimum days</span>
          <Input
            disabled={disabled || isSaving}
            min={1}
            onChange={(event) => setMinTripDays(event.target.value)}
            type="number"
            value={minTripDays}
          />
        </label>
        <label className="space-y-1 text-[13px] font-medium text-cocoa-600">
          <span>Maximum days</span>
          <Input
            disabled={disabled || isSaving}
            min={1}
            onChange={(event) => setMaxTripDays(event.target.value)}
            type="number"
            value={maxTripDays}
          />
        </label>
        <label className="space-y-1 text-[13px] font-medium text-cocoa-600">
          <span>Timezone</span>
          <Input
            disabled={disabled || isSaving}
            onChange={(event) => setTimezone(event.target.value)}
            value={timezone}
          />
        </label>
      </div>

      <label className="block space-y-1 text-[13px] font-medium text-cocoa-600">
        <span>Notes</span>
        <Textarea
          disabled={disabled || isSaving}
          maxLength={500}
          onChange={(event) => setNotes(event.target.value)}
          rows={3}
          value={notes}
        />
      </label>

      {localError ? <p className="text-[13px] text-red-700">{localError}</p> : null}

      <div className="flex flex-wrap items-center gap-2">
        <Button disabled={disabled || isSaving} type="submit">
          {isSaving ? "Saving..." : "Save availability"}
        </Button>
        {response?.submitted ? (
          <Button
            disabled={disabled || isDeleting}
            onClick={() => void onDelete()}
            type="button"
            variant="ghost"
          >
            {isDeleting ? "Removing..." : "Remove my response"}
          </Button>
        ) : null}
      </div>
    </form>
  );
}

function cleanRanges(ranges: AvailabilityDateRange[]) {
  return ranges.filter((range) => range.startDate && range.endDate);
}

function numberOrNull(value: string) {
  const parsed = Number(value);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : null;
}

function defaultTimezone() {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC";
  } catch {
    return "UTC";
  }
}
