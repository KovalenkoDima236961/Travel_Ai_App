import { apiFetch } from "@/shared/api/client";
import {
  getGoogleCalendarStatus,
  startGoogleCalendarConnect
} from "@/lib/api/calendar";
import type {
  CalendarImportApplyRequest,
  CalendarImportApplyResponse,
  CalendarImportPreviewRequest,
  CalendarImportPreviewResponse
} from "@/types/calendar-free-busy";

export function previewCalendarAvailabilityImport(
  tripId: string,
  input: CalendarImportPreviewRequest
) {
  return apiFetch<CalendarImportPreviewResponse>(
    `/trips/${tripId}/availability/import-calendar/preview`,
    {
      method: "POST",
      body: JSON.stringify(cleanPreviewInput(input))
    }
  );
}

export function applyCalendarAvailabilityImport(
  tripId: string,
  input: CalendarImportApplyRequest
) {
  return apiFetch<CalendarImportApplyResponse>(
    `/trips/${tripId}/availability/import-calendar/apply`,
    {
      method: "POST",
      body: JSON.stringify({
        ...cleanPreviewInput(input),
        mode: input.mode,
        availabilitySettings: {
          availableRanges: input.availabilitySettings.availableRanges ?? [],
          unavailableRanges: input.availabilitySettings.unavailableRanges ?? [],
          preferredRanges: input.availabilitySettings.preferredRanges ?? [],
          minTripDays: input.availabilitySettings.minTripDays ?? undefined,
          maxTripDays: input.availabilitySettings.maxTripDays ?? undefined,
          timezone: input.availabilitySettings.timezone?.trim() ?? "",
          notes: input.availabilitySettings.notes?.trim() ?? ""
        }
      })
    }
  );
}

export { getGoogleCalendarStatus, startGoogleCalendarConnect as connectGoogleCalendar };

function cleanPreviewInput(input: CalendarImportPreviewRequest) {
  return {
    startDate: input.startDate.trim(),
    endDate: input.endDate.trim(),
    timezone: input.timezone.trim(),
    calendarProvider: "google",
    calendarIds: input.calendarIds?.length ? input.calendarIds : ["primary"],
    conversion: {
      fullyBusyThresholdHours: input.conversion.fullyBusyThresholdHours,
      markFullyBusyDaysUnavailable: input.conversion.markFullyBusyDaysUnavailable,
      markPartiallyBusyDaysUnavailable: input.conversion.markPartiallyBusyDaysUnavailable,
      includeWeekendsAsPreferredIfFree:
        input.conversion.includeWeekendsAsPreferredIfFree
    }
  };
}
