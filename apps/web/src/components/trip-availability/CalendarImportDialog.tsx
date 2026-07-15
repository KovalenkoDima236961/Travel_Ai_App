"use client";

import { useMemo, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { CalendarConnectionPrompt } from "./CalendarConnectionPrompt";
import { CalendarImportApplyOptions } from "./CalendarImportApplyOptions";
import { CalendarImportPreview } from "./CalendarImportPreview";
import { CalendarPrivacyNotice } from "./CalendarPrivacyNotice";
import { useApplyCalendarAvailabilityImport } from "@/hooks/useApplyCalendarAvailabilityImport";
import { useCalendarAvailabilityImportPreview } from "@/hooks/useCalendarAvailabilityImportPreview";
import {
  connectGoogleCalendar,
  getGoogleCalendarStatus
} from "@/lib/api/calendar-free-busy";
import { calendarKeys } from "@/lib/api/calendar";
import { getErrorMessage } from "@/lib/utils";
import { Button } from "@/shared/ui/button";
import { Input } from "@/shared/ui/input";
import type { Trip } from "@/entities/trip/model";
import type {
  CalendarImportConversionSettings,
  CalendarImportMode
} from "@/types/calendar-free-busy";
import type { TripAvailabilityResponseInfo } from "@/types/trip-availability";

type CalendarImportDialogProps = {
  currentResponse?: TripAvailabilityResponseInfo | null;
  onApplied?: () => void;
  onOpenChange: (open: boolean) => void;
  open: boolean;
  trip: Trip;
};

const defaultConversion: CalendarImportConversionSettings = {
  fullyBusyThresholdHours: 6,
  markFullyBusyDaysUnavailable: true,
  markPartiallyBusyDaysUnavailable: false,
  includeWeekendsAsPreferredIfFree: false
};

export function CalendarImportDialog({
  currentResponse,
  onApplied,
  onOpenChange,
  open,
  trip
}: CalendarImportDialogProps) {
  const defaults = useMemo(() => defaultRange(trip), [trip]);
  const [startDate, setStartDate] = useState(defaults.startDate);
  const [endDate, setEndDate] = useState(defaults.endDate);
  const [timezone, setTimezone] = useState(
    currentResponse?.timezone || Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC"
  );
  const [conversion, setConversion] = useState(defaultConversion);
  const [mode, setMode] = useState<CalendarImportMode>("merge");
  const [error, setError] = useState<string | null>(null);
  const statusQuery = useQuery({
    queryKey: calendarKeys.googleConnection(),
    queryFn: getGoogleCalendarStatus,
    enabled: open
  });
  const connectMutation = useMutation({ mutationFn: connectGoogleCalendar });
  const previewMutation = useCalendarAvailabilityImportPreview(trip.id);
  const applyMutation = useApplyCalendarAvailabilityImport(trip.id);
  const preview = previewMutation.data?.preview;
  const connected = Boolean(statusQuery.data?.connected);

  async function connect() {
    try {
      setError(null);
      const returnUrl = `${window.location.origin}${window.location.pathname}?calendar_import=1`;
      const { authUrl } = await connectMutation.mutateAsync(returnUrl);
      window.location.assign(authUrl);
    } catch (err) {
      setError(getErrorMessage(err, "Could not start Google Calendar connection."));
    }
  }

  async function previewImport() {
    try {
      setError(null);
      await previewMutation.mutateAsync({
        startDate,
        endDate,
        timezone,
        calendarProvider: "google",
        calendarIds: ["primary"],
        conversion
      });
    } catch (err) {
      setError(getErrorMessage(err, "Could not preview calendar import."));
    }
  }

  async function applyImport() {
    try {
      setError(null);
      await applyMutation.mutateAsync({
        startDate,
        endDate,
        timezone,
        calendarProvider: "google",
        calendarIds: ["primary"],
        conversion,
        mode,
        availabilitySettings: {
          availableRanges: [],
          minTripDays: currentResponse?.minTripDays ?? trip.days,
          maxTripDays: currentResponse?.maxTripDays ?? trip.days,
          timezone,
          notes: "Imported from Google Calendar."
        }
      });
      onApplied?.();
      onOpenChange(false);
    } catch (err) {
      setError(getErrorMessage(err, "Could not apply calendar import."));
    }
  }

  if (!open) {
    return null;
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 px-4 py-6">
      <div className="max-h-[90vh] w-full max-w-2xl overflow-auto rounded-lg bg-white p-5 shadow-xl">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h2 className="text-lg font-semibold text-slate-950">
              Import from Google Calendar
            </h2>
            <p className="mt-1 text-sm text-slate-600">Verify before applying.</p>
          </div>
          <Button onClick={() => onOpenChange(false)} size="sm" type="button" variant="ghost">
            Close
          </Button>
        </div>

        <div className="mt-4 space-y-4">
          <CalendarPrivacyNotice />
          {error ? (
            <div className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-800">
              {error}
            </div>
          ) : null}

          {!connected ? (
            <CalendarConnectionPrompt isPending={connectMutation.isPending} onConnect={connect} />
          ) : (
            <>
              <div className="grid gap-3 sm:grid-cols-3">
                <label className="text-sm font-medium text-slate-700">
                  Start date
                  <Input
                    className="mt-1"
                    onChange={(event) => setStartDate(event.target.value)}
                    type="date"
                    value={startDate}
                  />
                </label>
                <label className="text-sm font-medium text-slate-700">
                  End date
                  <Input
                    className="mt-1"
                    onChange={(event) => setEndDate(event.target.value)}
                    type="date"
                    value={endDate}
                  />
                </label>
                <label className="text-sm font-medium text-slate-700">
                  Timezone
                  <Input
                    className="mt-1"
                    onChange={(event) => setTimezone(event.target.value)}
                    value={timezone}
                  />
                </label>
              </div>

              <CalendarImportApplyOptions
                conversion={conversion}
                mode={mode}
                onConversionChange={setConversion}
                onModeChange={setMode}
              />

              <div className="flex flex-wrap gap-2">
                <Button
                  disabled={previewMutation.isPending || applyMutation.isPending}
                  onClick={() => void previewImport()}
                  type="button"
                >
                  {previewMutation.isPending ? "Previewing..." : "Preview import"}
                </Button>
                <Button
                  disabled={!preview || applyMutation.isPending}
                  onClick={() => void applyImport()}
                  type="button"
                  variant="secondary"
                >
                  {applyMutation.isPending ? "Applying..." : "Apply to my availability"}
                </Button>
              </div>

              {preview ? <CalendarImportPreview preview={preview} /> : null}
            </>
          )}
        </div>
      </div>
    </div>
  );
}

function defaultRange(trip: Trip) {
  const start = trip.startDate?.slice(0, 10) || new Date().toISOString().slice(0, 10);
  const startDate = new Date(`${start}T00:00:00Z`);
  const endDate = new Date(startDate);
  endDate.setUTCDate(startDate.getUTCDate() + 29);
  return {
    startDate: start,
    endDate: endDate.toISOString().slice(0, 10)
  };
}
